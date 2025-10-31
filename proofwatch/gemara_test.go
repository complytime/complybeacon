package proofwatch

import (
	"testing"
	"time"

	"github.com/ossf/gemara/layer4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
)

func attrsToMap(t *testing.T, attrs []attribute.KeyValue) map[string]any {
	t.Helper()
	m := make(map[string]any, len(attrs))
	for _, a := range attrs {
		m[string(a.Key)] = a.Value.AsInterface()
	}
	return m
}

func TestGemaraEvidenceAttributes(t *testing.T) {
	evidence := createTestGemaraEvidence()
	attrs := evidence.Attributes()
	require.NotEmpty(t, attrs)

	attrMap := attrsToMap(t, attrs)

	// Core compliance attributes
	assert.Equal(t, "test-author", attrMap[POLICY_ENGINE_NAME])
	assert.Equal(t, "test-control-id", attrMap[COMPLIANCE_CONTROL_ID])
	assert.Equal(t, "test-catalog-id", attrMap[COMPLIANCE_CONTROL_CATALOG_ID])
	assert.Equal(t, "Passed", attrMap[POLICY_EVALUATION_RESULT])
	assert.Equal(t, "test-procedure-id", attrMap[POLICY_RULE_ID])
	assert.Equal(t, "test-audit-id", attrMap[COMPLIANCE_ASSESSMENT_ID])

	// Optional attributes
	assert.Equal(t, "Test assessment message", attrMap[POLICY_EVALUATION_MESSAGE])
	assert.Equal(t, "Test recommendation", attrMap[COMPLIANCE_REMEDIATION_DESCRIPTION])
}

func TestGemaraEvidenceTimestamp(t *testing.T) {
	tests := []struct {
		name      string
		endTime   string
		expectErr bool
	}{
		{
			name:      "valid RFC3339 timestamp",
			endTime:   "2023-12-01T10:30:00Z",
			expectErr: false,
		},
		{
			name:      "valid RFC3339 with timezone",
			endTime:   "2023-12-01T10:30:00-05:00",
			expectErr: false,
		},
		{
			name:      "invalid timestamp format",
			endTime:   "invalid-timestamp",
			expectErr: true,
		},
		{
			name:      "empty timestamp",
			endTime:   "",
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := GemaraEvidence{
				AssessmentLog: layer4.AssessmentLog{
					End: layer4.Datetime(tt.endTime),
				},
			}

			ts := evidence.Timestamp()
			if tt.expectErr {
				assert.WithinDuration(t, time.Now(), ts, time.Second)
			} else {
				expected, err := time.Parse(time.RFC3339, tt.endTime)
				require.NoError(t, err)
				assert.Equal(t, expected, ts)
			}
		})
	}
}

func TestGemaraEvidenceAttributesEmptyFields(t *testing.T) {
	// Empty optional fields
	evidence := GemaraEvidence{
		Metadata: layer4.Metadata{
			Author: layer4.Author{
				Name: "test-author",
			},
		},
		AssessmentLog: layer4.AssessmentLog{
			Requirement: layer4.Mapping{
				EntryId:     "test-control-id",
				ReferenceId: "test-catalog-id",
			},
			Procedure: layer4.Mapping{
				EntryId: "test-procedure-id",
			},
			Result: layer4.Passed,
			// Message and Recommendation are empty
		},
	}

	attrs := evidence.Attributes()
	require.NotEmpty(t, attrs)
	attrMap := attrsToMap(t, attrs)

	// Required present
	assert.Equal(t, "test-author", attrMap[POLICY_ENGINE_NAME])
	assert.Equal(t, "test-control-id", attrMap[COMPLIANCE_CONTROL_ID])

	// Optional omitted
	assert.NotContains(t, attrMap, POLICY_EVALUATION_MESSAGE)
	assert.NotContains(t, attrMap, COMPLIANCE_REMEDIATION_DESCRIPTION)
}

func TestGemaraEvidenceAttributesDifferentResults(t *testing.T) {
	tests := []struct {
		name     string
		result   layer4.Result
		expected string
	}{
		{
			name:     "passed result",
			result:   layer4.Passed,
			expected: "Passed",
		},
		{
			name:     "failed result",
			result:   layer4.Failed,
			expected: "Failed",
		},
		{
			name:     "not applicable result",
			result:   layer4.NotApplicable,
			expected: "Not Applicable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evidence := GemaraEvidence{
				Metadata: layer4.Metadata{
					Author: layer4.Author{
						Name: "test-author",
					},
				},
				AssessmentLog: layer4.AssessmentLog{
					Requirement: layer4.Mapping{
						EntryId:     "test-control-id",
						ReferenceId: "test-catalog-id",
					},
					Procedure: layer4.Mapping{
						EntryId: "test-procedure-id",
					},
					Result: tt.result,
				},
			}

			attrs := evidence.Attributes()
			require.NotEmpty(t, attrs)
			attrMap := attrsToMap(t, attrs)

			assert.Equal(t, tt.expected, attrMap[POLICY_EVALUATION_RESULT])
		})
	}
}

// This remains the canonical helper for Gemara evidence tests.
func createTestGemaraEvidence() GemaraEvidence {
	return GemaraEvidence{
		Metadata: layer4.Metadata{
			Id:      "test-audit-id",
			Version: "1.0.0",
			Author: layer4.Author{
				Name:    "test-author",
				Uri:     "https://example.com",
				Version: "1.0.0",
			},
		},
		AssessmentLog: layer4.AssessmentLog{
			Requirement: layer4.Mapping{
				EntryId:     "test-control-id",
				ReferenceId: "test-catalog-id",
				Strength:    8,
				Remarks:     "Test control mapping",
			},
			Procedure: layer4.Mapping{
				EntryId:     "test-procedure-id",
				ReferenceId: "test-procedure-ref",
				Remarks:     "Test procedure",
			},
			Description:    "Test assessment description",
			Message:        "Test assessment message",
			Result:         layer4.Passed,
			Applicability:  []string{"test-scope-1", "test-scope-2"},
			StepsExecuted:  5,
			Recommendation: "Test recommendation",
			End:            "2023-12-01T10:30:00Z",
		},
	}
}
