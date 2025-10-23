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
		assert.Equal(t, "test-policy-123", req.Evidence.PolicyRuleId)
		assert.Equal(t, "test-source", req.Evidence.PolicyEngineName)
		assert.Equal(t, EvidencePolicyEvaluationStatus("compliant"), req.Evidence.PolicyEvaluationStatus)

		// Return a mock response
		response := EnrichmentResponse{
			Compliance: Compliance{
				Control: ComplianceControl{
					CatalogId:              "NIST-800-53",
					Category:               "Access Control",
					Id:                     "AC-1",
					RemediationDescription: stringPtr("Implement proper access controls"),
				},
				Frameworks: ComplianceFrameworks{
					Requirements: []string{"req-1", "req-2"},
					Frameworks:   []string{"NIST-800-53", "ISO-27001"},
				},
				Status:           "Pass",
				EnrichmentStatus: ComplianceEnrichmentStatusSuccess,
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
		COMPLIANCE_STATUS:                  "Pass",
		COMPLIANCE_CONTROL_ID:              "AC-1",
		COMPLIANCE_CONTROL_CATALOG_ID:      "NIST-800-53",
		COMPLIANCE_CONTROL_CATEGORY:        "Access Control",
		COMPLIANCE_REMEDIATION_DESCRIPTION: "Implement proper access controls",
	})

	// Check requirements and standards arrays
	requirements := attrs.AsRaw()[COMPLIANCE_REQUIREMENTS].([]interface{})
	assert.Len(t, requirements, 2)
	assert.Contains(t, requirements, "req-1")
	assert.Contains(t, requirements, "req-2")

	standards := attrs.AsRaw()[COMPLIANCE_FRAMEWORKS].([]interface{})
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
			name: "missing policy.rule.id",
			configRecord: func(logRecord plog.LogRecord) {
				attrs := logRecord.Attributes()
				attrs.Remove(POLICY_RULE_ID)
				attrs.PutStr(POLICY_ENGINE_NAME, "test-source")
				attrs.PutStr(POLICY_EVALUATION_RESULT, "compliant")
			},
			expectedAttribute: POLICY_RULE_ID,
		},
		{
			name: "missing policy.engine.name",
			configRecord: func(logRecord plog.LogRecord) {
				attrs := logRecord.Attributes()
				attrs.PutStr(POLICY_RULE_ID, "test-policy-123")
				attrs.Remove(POLICY_ENGINE_NAME)
				attrs.PutStr(POLICY_EVALUATION_RESULT, "compliant")
			},
			expectedAttribute: POLICY_ENGINE_NAME,
		},
		{
			name: "missing policy.evaluation.result",
			configRecord: func(logRecord plog.LogRecord) {
				attrs := logRecord.Attributes()
				attrs.PutStr(POLICY_RULE_ID, "test-policy-123")
				attrs.PutStr(POLICY_ENGINE_NAME, "test-source")
				attrs.Remove(POLICY_EVALUATION_RESULT)
			},
			expectedAttribute: POLICY_EVALUATION_RESULT,
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
				// Return a response with empty compliance but with enrichment status
				response := EnrichmentResponse{
					Compliance: Compliance{
						EnrichmentStatus: ComplianceEnrichmentStatusUnknown,
						Status:           "",
						Control:          ComplianceControl{},
						Frameworks:       ComplianceFrameworks{},
					},
				}
				_ = json.NewEncoder(w).Encode(response)
			},
			expectErr: false,
			assertFunc: func(t *testing.T, attrs map[string]interface{}, err error) {
				assert.NoError(t, err)
				// Only enrichment status should be present, not the other compliance attributes
				assert.Equal(t, string(ComplianceEnrichmentStatusUnknown), attrs[COMPLIANCE_ENRICHMENT_STATUS])
				// Compliance attributes should not be present since enrichment was not successful
				_, hasStatus := attrs[COMPLIANCE_STATUS]
				assert.False(t, hasStatus, "Compliance status should not be present when enrichment is not successful")
			},
		},
		{
			name: "omits remediation when nil",
			handler: func(w http.ResponseWriter, r *http.Request) {
				_ = json.NewEncoder(w).Encode(EnrichmentResponse{
					Compliance: Compliance{
						Control: ComplianceControl{
							CatalogId:              "NIST-800-53",
							Category:               "Access Control",
							Id:                     "AC-1",
							RemediationDescription: nil,
						},
						Frameworks: ComplianceFrameworks{
							Requirements: []string{"req-1"},
							Frameworks:   []string{"NIST-800-53"},
						},
						Status:           "Pass",
						EnrichmentStatus: ComplianceEnrichmentStatusSuccess,
					},
				})
			},
			expectErr: false,
			assertFunc: func(t *testing.T, attrs map[string]interface{}, err error) {
				assert.NoError(t, err)
				_, exists := attrs[COMPLIANCE_REMEDIATION_DESCRIPTION]
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

// createTestLogRecord is a helper function for easy test setup
func createTestLogRecord() (plog.LogRecord, pcommon.Resource) {
	logRecord := plog.NewLogRecord()
	attrs := logRecord.Attributes()
	attrs.PutStr(POLICY_RULE_ID, "test-policy-123")
	attrs.PutStr(POLICY_ENGINE_NAME, "test-source")
	attrs.PutStr(POLICY_EVALUATION_RESULT, "compliant")

	resource := pcommon.NewResource()
	return logRecord, resource
}
