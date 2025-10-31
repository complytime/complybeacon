package truthbeam

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/confighttp"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.uber.org/zap/zaptest"

	"github.com/complytime/complybeacon/truthbeam/internal/client"
)

// The processor tests validate the core processor functionality including log processing,
// enrichment request formation, and response handling.

func TestNewTruthBeamProcessor(t *testing.T) {
	cfg := &Config{
		ClientConfig: confighttp.NewDefaultClientConfig(),
	}
	cfg.ClientConfig.Endpoint = "http://localhost:8081"

	settings := processortest.NewNopSettings(component.MustNewType("test"))
	settings.Logger = zaptest.NewLogger(t)

	processor, err := newTruthBeamProcessor(cfg, settings)
	require.NoError(t, err, "Error creating truth beam processor")
	require.NotNil(t, processor, "Processor should not be nil")
	assert.Equal(t, cfg, processor.config)
	assert.Nil(t, processor.client)
	assert.NotNil(t, processor.logger)
}

func TestNewTruthBeamProcessorWithInvalidConfig(t *testing.T) {
	processor, err := newTruthBeamProcessor(nil, processortest.NewNopSettings(component.MustNewType("test")))
	assert.Error(t, err, "Expected error with nil config")
	assert.Nil(t, processor, "Processor should be nil with invalid config")

	wrongConfig := struct{}{}
	processor, err = newTruthBeamProcessor(wrongConfig, processortest.NewNopSettings(component.MustNewType("test")))
	assert.Error(t, err, "Expected error with wrong config type")
	assert.Contains(t, err.Error(), "invalid configuration provided")
	assert.Nil(t, processor, "Processor should be nil with wrong config type")
}

func TestProcessLogs(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/enrich", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req client.EnrichmentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		assert.Equal(t, "test-policy-123", req.Evidence.PolicyRuleId)
		assert.Equal(t, "test-source", req.Evidence.PolicyEngineName)
		assert.Equal(t, client.EvidencePolicyEvaluationStatus("compliant"), req.Evidence.PolicyEvaluationStatus)

		response := client.EnrichmentResponse{
			Compliance: client.Compliance{
				Control: client.ComplianceControl{
					CatalogId:              "NIST-800-53",
					Category:               "Access Control",
					Id:                     "AC-1",
					RemediationDescription: stringPtr("Implement proper access controls"),
				},
				Frameworks: client.ComplianceFrameworks{
					Requirements: []string{"req-1", "req-2"},
					Frameworks:   []string{"NIST-800-53", "ISO-27001"},
				},
				Status:           "Pass",
				EnrichmentStatus: client.ComplianceEnrichmentStatusSuccess,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer mockServer.Close()

	processor := createTestProcessor(t, mockServer.URL)
	logs := createTestLogs()
	setRequiredAttributes(logs)

	ctx := context.Background()
	result, err := processor.processLogs(ctx, logs)
	require.NoError(t, err)
	require.NotNil(t, result)

	processedLogRecord := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs := processedLogRecord.Attributes()

	// Verify compliance attributes were added
	assert.Equal(t, "Pass", attrs.AsRaw()[client.COMPLIANCE_STATUS])
	assert.Equal(t, "AC-1", attrs.AsRaw()[client.COMPLIANCE_CONTROL_ID])
	assert.Equal(t, "NIST-800-53", attrs.AsRaw()[client.COMPLIANCE_CONTROL_CATALOG_ID])
	assert.Equal(t, "Access Control", attrs.AsRaw()[client.COMPLIANCE_CONTROL_CATEGORY])
	assert.Equal(t, "Implement proper access controls", attrs.AsRaw()[client.COMPLIANCE_REMEDIATION_DESCRIPTION])

	requirements := attrs.AsRaw()[client.COMPLIANCE_REQUIREMENTS].([]interface{})
	assert.Len(t, requirements, 2)
	assert.Contains(t, requirements, "req-1")
	assert.Contains(t, requirements, "req-2")

	standards := attrs.AsRaw()[client.COMPLIANCE_FRAMEWORKS].([]interface{})
	assert.Len(t, standards, 2)
	assert.Contains(t, standards, "NIST-800-53")
	assert.Contains(t, standards, "ISO-27001")
}

func TestProcessLogsWithMissingAttributes(t *testing.T) {
	processor := createTestProcessor(t, "http://localhost:8081")
	logs := createTestLogs()
	logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Missing policy.rule.id attribute
	logRecord.Attributes().PutStr(client.POLICY_ENGINE_NAME, "test-source")
	logRecord.Attributes().PutStr(client.POLICY_EVALUATION_RESULT, "compliant")

	ctx := context.Background()
	result, err := processor.processLogs(ctx, logs)
	require.NoError(t, err, "Processor should not fail even with missing attributes")
	require.NotNil(t, result)
}

func TestProcessLogsWithHTTPError(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		errorResponse := client.Error{
			Code:    500,
			Message: "Internal server error",
		}
		_ = json.NewEncoder(w).Encode(errorResponse)
	}))
	defer mockServer.Close()

	processor := createTestProcessor(t, mockServer.URL)
	logs := createTestLogs()
	setRequiredAttributes(logs)

	ctx := context.Background()
	result, err := processor.processLogs(ctx, logs)
	require.NoError(t, err, "Processor should not fail even with HTTP errors")
	require.NotNil(t, result)
}

func TestProcessLogsWithMixedValidAndInvalidRecords(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/v1/enrich", r.URL.Path)

		var req client.EnrichmentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		require.NoError(t, err)

		// Only process valid records (with policy.rule.id)
		if req.Evidence.PolicyRuleId == "test-policy-123" || req.Evidence.PolicyRuleId == "test-policy-456" {
			response := client.EnrichmentResponse{
				Compliance: client.Compliance{
					Control: client.ComplianceControl{
						CatalogId:              "NIST-800-53",
						Category:               "Access Control",
						Id:                     "AC-1",
						RemediationDescription: stringPtr("Implement proper access controls"),
					},
					Frameworks: client.ComplianceFrameworks{
						Requirements: []string{"req-1", "req-2"},
						Frameworks:   []string{"NIST-800-53"},
					},
					Status:           "Pass",
					EnrichmentStatus: client.ComplianceEnrichmentStatusSuccess,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer mockServer.Close()

	processor := createTestProcessor(t, mockServer.URL)

	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()

	validRecord1 := scopeLogs.LogRecords().AppendEmpty()
	validRecord1.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	validRecord1.Attributes().PutStr(client.POLICY_RULE_ID, "test-policy-123")
	validRecord1.Attributes().PutStr(client.POLICY_ENGINE_NAME, "test-source")
	validRecord1.Attributes().PutStr(client.POLICY_EVALUATION_RESULT, "compliant")

	// Invalid record (missing policy.rule.id)
	invalidRecord := scopeLogs.LogRecords().AppendEmpty()
	invalidRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	invalidRecord.Attributes().PutStr(client.POLICY_ENGINE_NAME, "test-source")
	invalidRecord.Attributes().PutStr(client.POLICY_EVALUATION_RESULT, "compliant")

	validRecord2 := scopeLogs.LogRecords().AppendEmpty()
	validRecord2.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	validRecord2.Attributes().PutStr(client.POLICY_RULE_ID, "test-policy-456")
	validRecord2.Attributes().PutStr(client.POLICY_ENGINE_NAME, "test-source")
	validRecord2.Attributes().PutStr(client.POLICY_EVALUATION_RESULT, "compliant")

	ctx := context.Background()
	result, err := processor.processLogs(ctx, logs)
	require.NoError(t, err)
	require.NotNil(t, result)

	// Verify we have 3 records
	require.Equal(t, 3, result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().Len())

	// Check valid record - should be enriched
	validRecord1Result := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	attrs1 := validRecord1Result.Attributes()
	assert.Equal(t, "Pass", attrs1.AsRaw()[client.COMPLIANCE_STATUS])
	assert.Equal(t, "AC-1", attrs1.AsRaw()[client.COMPLIANCE_CONTROL_ID])
	assert.Equal(t, "NIST-800-53", attrs1.AsRaw()[client.COMPLIANCE_CONTROL_CATALOG_ID])

	// Check invalid record - should remain unchanged (no compliance attributes)
	invalidRecordResult := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(1)
	attrs2 := invalidRecordResult.Attributes()
	assert.Nil(t, attrs2.AsRaw()[client.COMPLIANCE_STATUS])
	assert.Nil(t, attrs2.AsRaw()[client.COMPLIANCE_CONTROL_ID])
	assert.Nil(t, attrs2.AsRaw()[client.COMPLIANCE_CONTROL_CATALOG_ID])
	// Original attributes should still be there
	assert.Equal(t, "test-source", attrs2.AsRaw()[client.POLICY_ENGINE_NAME])

	// Check valid record 2 - should be enriched
	validRecord2Result := result.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(2)
	attrs3 := validRecord2Result.Attributes()
	assert.Equal(t, "Pass", attrs3.AsRaw()[client.COMPLIANCE_STATUS])
	assert.Equal(t, "AC-1", attrs3.AsRaw()[client.COMPLIANCE_CONTROL_ID])
	assert.Equal(t, "NIST-800-53", attrs3.AsRaw()[client.COMPLIANCE_CONTROL_CATALOG_ID])
}

// Helper functions
func createTestProcessor(t *testing.T, endpoint string) *truthBeamProcessor {
	cfg := &Config{
		ClientConfig: confighttp.NewDefaultClientConfig(),
	}
	cfg.ClientConfig.Endpoint = endpoint

	settings := processortest.NewNopSettings(component.MustNewType("test"))
	settings.Logger = zaptest.NewLogger(t)

	processor, err := newTruthBeamProcessor(cfg, settings)
	require.NoError(t, err)
	err = processor.start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)
	return processor
}

func createTestLogs() plog.Logs {
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()
	logRecord.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	return logs
}

func setRequiredAttributes(logs plog.Logs) {
	logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	logRecord.Attributes().PutStr(client.POLICY_RULE_ID, "test-policy-123")
	logRecord.Attributes().PutStr(client.POLICY_ENGINE_NAME, "test-source")
	logRecord.Attributes().PutStr(client.POLICY_EVALUATION_RESULT, "compliant")
}

func stringPtr(s string) *string {
	return &s
}
