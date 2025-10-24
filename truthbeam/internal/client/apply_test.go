package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// The apply tests validate attribute application logic for enrichment of log records
// with compliance data from the compass API.

// TestApplyAttributes tests the ApplyAttributes functionality with a valid response.
func TestApplyAttributes(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/enrich", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		// Parse the request body
		var req EnrichmentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Verify request content
		assert.Equal(t, "test-policy-123", req.Evidence.PolicyId)
		assert.Equal(t, "test-source", req.Evidence.Source)
		assert.Equal(t, "compliant", req.Evidence.Decision)
		assert.Equal(t, "audit", req.Evidence.Action)

		// Return a mock response
		response := EnrichmentResponse{
			Compliance: Compliance{
				Catalog:      "NIST-800-53",
				Category:     "Access Control",
				Control:      "AC-1",
				Remediation:  stringPtr("Implement proper access controls"),
				Requirements: []string{"req-1", "req-2"},
				Standards:    []string{"NIST-800-53", "ISO-27001"},
			},
			Status: Status{
				Id:    statusIdPtr(1),
				Title: "Pass",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	// Create client
	client, err := NewClient(mockServer.URL)
	require.NoError(t, err)

	// Create test log record
	logRecord, resource := createTestLogRecord()
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	attrs := logRecord.Attributes()

	// Apply attributes for log enrichment
	ctx := context.Background()
	err = ApplyAttributes(ctx, client, mockServer.URL, resource, logRecord)
	require.NoError(t, err)

	// Verify that compliance attributes were added
	assertAttributesEqual(t, attrs.AsRaw(), map[string]interface{}{
		COMPLIANCE_STATUS:                          "Pass",
		COMPLIANCE_CONTROL_ID:                      "AC-1",
		COMPLIANCE_CONTROL_CATALOG_ID:              "NIST-800-53",
		COMPLIANCE_CATEGORY:                        "Access Control",
		COMPLIANCE_CONTROL_REMEDIATION_DESCRIPTION: "Implement proper access controls",
	})

	// Check requirements and standards arrays
	requirements := attrs.AsRaw()[COMPLIANCE_REQUIREMENTS].([]interface{})
	assert.Len(t, requirements, 2)
	assert.Contains(t, requirements, "req-1")
	assert.Contains(t, requirements, "req-2")

	standards := attrs.AsRaw()[COMPLIANCE_STANDARDS].([]interface{})
	assert.Len(t, standards, 2)
	assert.Contains(t, standards, "NIST-800-53")
	assert.Contains(t, standards, "ISO-27001")
}

// Table-driven coverage for missing required attributes
func TestApplyAttributesMissingRequiredAttributes(t *testing.T) {
	client, err := NewClient("http://localhost:8081")
	require.NoError(t, err)

	tests := []struct {
		name              string
		configRecord      func(plog.LogRecord)
		expectedAttribute string
	}{
		{
			name: "missing policy.id",
			configRecord: func(logRecord plog.LogRecord) {
				attrs := logRecord.Attributes()
				attrs.Remove(POLICY_ID)
				attrs.PutStr(POLICY_SOURCE, "test-source")
				attrs.PutStr(POLICY_EVALUATION_STATUS, "compliant")
				attrs.PutStr(POLICY_ENFORCEMENT_ACTION, "audit")
			},
			expectedAttribute: POLICY_ID,
		},
		{
			name: "missing policy.source",
			configRecord: func(logRecord plog.LogRecord) {
				attrs := logRecord.Attributes()
				attrs.PutStr(POLICY_ID, "test-policy-123")
				attrs.Remove(POLICY_SOURCE)
				attrs.PutStr(POLICY_EVALUATION_STATUS, "compliant")
				attrs.PutStr(POLICY_ENFORCEMENT_ACTION, "audit")
			},
			expectedAttribute: POLICY_SOURCE,
		},
		{
			name: "missing policy.evaluation.status",
			configRecord: func(logRecord plog.LogRecord) {
				attrs := logRecord.Attributes()
				attrs.PutStr(POLICY_ID, "test-policy-123")
				attrs.PutStr(POLICY_SOURCE, "test-source")
				attrs.Remove(POLICY_EVALUATION_STATUS)
				attrs.PutStr(POLICY_ENFORCEMENT_ACTION, "audit")
			},
			expectedAttribute: POLICY_EVALUATION_STATUS,
		},
		{
			name: "missing policy.enforcement.action",
			configRecord: func(logRecord plog.LogRecord) {
				attrs := logRecord.Attributes()
				attrs.PutStr(POLICY_ID, "test-policy-123")
				attrs.PutStr(POLICY_SOURCE, "test-source")
				attrs.PutStr(POLICY_EVALUATION_STATUS, "compliant")
				attrs.Remove(POLICY_ENFORCEMENT_ACTION)
			},
			expectedAttribute: POLICY_ENFORCEMENT_ACTION,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logRecord := plog.NewLogRecord()
			resource := pcommon.NewResource()
			tt.configRecord(logRecord)

			ctx := context.Background()
			err := ApplyAttributes(ctx, client, "http://localhost:8081", resource, logRecord)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "missing required attribute")
			assert.Contains(t, err.Error(), tt.expectedAttribute)
		})
	}
}

func TestApplyAttributes_ServerResponses(t *testing.T) {
	tests := []struct {
		name       string
		handler    http.HandlerFunc // if nil, use endpoint directly (e.g., network error)
		endpoint   string           // optional override for endpoint
		expectErr  bool
		assertFunc func(t *testing.T, attrs map[string]interface{}, err error)
	}{
		{
			name: "http 500 error",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(Error{Code: 500, Message: "Internal server error"})
			},
			expectErr: true,
			assertFunc: func(t *testing.T, _ map[string]interface{}, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "API call failed with status 500")
			},
		},
		{
			name:      "network error",
			handler:   nil,
			endpoint:  "http://invalid-host:9999",
			expectErr: true,
			assertFunc: func(t *testing.T, _ map[string]interface{}, err error) {
				assert.Error(t, err)
			},
		},
		{
			name: "empty response",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte("{}"))
			},
			expectErr: false,
			assertFunc: func(t *testing.T, attrs map[string]interface{}, err error) {
				assert.NoError(t, err)
				// Expect blank values when the API returns an empty JSON object
				assertAttributesEqual(t, attrs, map[string]interface{}{
					COMPLIANCE_STATUS:             "",
					COMPLIANCE_CONTROL_ID:         "",
					COMPLIANCE_CONTROL_CATALOG_ID: "",
					COMPLIANCE_CATEGORY:           "",
				})
				// Ensure requirements and standards are present as empty arrays
				requirements := attrs[COMPLIANCE_REQUIREMENTS].([]interface{})
				assert.Len(t, requirements, 0)
				standards := attrs[COMPLIANCE_STANDARDS].([]interface{})
				assert.Len(t, standards, 0)
			},
		},
		{
			name: "omits remediation when nil",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(EnrichmentResponse{
					Compliance: Compliance{
						Catalog:      "NIST-800-53",
						Category:     "Access Control",
						Control:      "AC-1",
						Remediation:  nil,
						Requirements: []string{"req-1"},
						Standards:    []string{"NIST-800-53"},
					},
					Status: Status{Id: statusIdPtr(1), Title: "Pass"},
				})
			},
			expectErr: false,
			assertFunc: func(t *testing.T, attrs map[string]interface{}, err error) {
				assert.NoError(t, err)
				_, exists := attrs[COMPLIANCE_CONTROL_REMEDIATION_DESCRIPTION]
				assert.False(t, exists, "Remediation attribute should not exist when nil")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var endpoint string

			// If a handler is provided, use it to create a mock server
			// Otherwise, use the endpoint directly to test network errors
			if tt.handler != nil {
				mockServer := httptest.NewServer(tt.handler)
				endpoint = mockServer.URL
				defer mockServer.Close()
			} else {
				endpoint = tt.endpoint
			}

			client, err := NewClient(endpoint)
			require.NoError(t, err)

			logRecord, resource := createTestLogRecord()
			ctx := context.Background()
			err = ApplyAttributes(ctx, client, endpoint, resource, logRecord)

			tt.assertFunc(t, logRecord.Attributes().AsRaw(), err)
		})
	}
}

// assertAttributesEqual compares expected key/value pairs against the attributes map.
func assertAttributesEqual(t *testing.T, attrs map[string]interface{}, expected map[string]interface{}) {
	t.Helper()
	assert.Subset(t, attrs, expected)
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func statusIdPtr(id int) *StatusId {
	statusId := StatusId(id)
	return &statusId
}

// createTestLogRecord is a helper function for easy test setup
func createTestLogRecord() (plog.LogRecord, pcommon.Resource) {
	logRecord := plog.NewLogRecord()
	attrs := logRecord.Attributes()
	attrs.PutStr(POLICY_ID, "test-policy-123")
	attrs.PutStr(POLICY_SOURCE, "test-source")
	attrs.PutStr(POLICY_EVALUATION_STATUS, "compliant")
	attrs.PutStr(POLICY_ENFORCEMENT_ACTION, "audit")

	resource := pcommon.NewResource()
	return logRecord, resource
}
