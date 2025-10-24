package proofwatch

import (
	"testing"
	"time"

	ocsf "github.com/Santiago-Labs/go-ocsf/ocsf/v1_5_0"
	"github.com/stretchr/testify/assert"
)

func TestOCSFEvidenceAttributes(t *testing.T) {
	evidence := createTestEvidence()
	attrs := evidence.Attributes()

	// Check that required attributes are present
	attrMap := make(map[string]interface{})
	for _, attr := range attrs {
		attrMap[string(attr.Key)] = attr.Value.AsInterface()
	}

	// Verify policy attributes
	assert.Equal(t, "test-policy", attrMap[POLICY_ID])
	assert.Equal(t, "test-policy", attrMap[POLICY_NAME])
	assert.Equal(t, "test-product", attrMap[POLICY_SOURCE])

	// Verify evaluation status mapping
	assert.Equal(t, "pass", attrMap[POLICY_EVALUATION_STATUS])
}

func TestMapEvaluationStatus(t *testing.T) {
	tests := []struct {
		name     string
		status   *string
		expected string
	}{
		{
			name:     "success status",
			status:   stringPtr("success"),
			expected: "pass",
		},
		{
			name:     "failure status",
			status:   stringPtr("failure"),
			expected: "fail",
		},
		{
			name:     "unknown status",
			status:   stringPtr("unknown"),
			expected: "unknown",
		},
		{
			name:     "nil status",
			status:   nil,
			expected: "error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapEvaluationStatus(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapEnforcementAction(t *testing.T) {
	tests := []struct {
		name          string
		actionID      *int32
		dispositionID *int32
		expected      string
	}{
		{
			name:     "denied action",
			actionID: int32Ptr(2),
			expected: "block",
		},
		{
			name:     "modified action",
			actionID: int32Ptr(4),
			expected: "mutate",
		},
		{
			name:     "observed action",
			actionID: int32Ptr(3),
			expected: "audit",
		},
		{
			name:     "no action",
			actionID: int32Ptr(16),
			expected: "audit",
		},
		{
			name:     "logged action",
			actionID: int32Ptr(17),
			expected: "audit",
		},
		{
			name:     "nil action",
			actionID: nil,
			expected: "audit",
		},
		{
			name:     "unknown action",
			actionID: int32Ptr(99),
			expected: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapEnforcementAction(tt.actionID, tt.dispositionID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapEnforcementStatus(t *testing.T) {
	tests := []struct {
		name          string
		actionID      *int32
		dispositionID *int32
		expected      string
	}{
		{
			name:     "nil action",
			actionID: nil,
			expected: "success",
		},
		{
			name:          "successful block",
			actionID:      int32Ptr(2),
			dispositionID: int32Ptr(2),
			expected:      "success",
		},
		{
			name:          "successful correction",
			actionID:      int32Ptr(4),
			dispositionID: int32Ptr(11),
			expected:      "success",
		},
		{
			name:          "failed enforcement",
			actionID:      int32Ptr(2),
			dispositionID: int32Ptr(1),
			expected:      "fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapEnforcementStatus(tt.actionID, tt.dispositionID)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateEvidenceFields(t *testing.T) {
	tests := []struct {
		name     string
		evidence OCSFEvidence
		wantErr  bool
	}{
		{
			name: "valid evidence",
			evidence: func() OCSFEvidence {
				policyName := "test-policy"
				productName := "test-product"
				status := "success"
				return OCSFEvidence{
					ScanActivity: ocsf.ScanActivity{
						Time: time.Now().UnixMilli(),
						Metadata: ocsf.Metadata{
							Product: ocsf.Product{
								Name: &productName,
							},
						},
						Status: &status,
					},
					Policy: ocsf.Policy{
						Uid:  &policyName,
						Name: &policyName,
					},
				}
			}(),
			wantErr: false,
		},
		{
			name: "missing policy id",
			evidence: func() OCSFEvidence {
				productName := "test-product"
				status := "success"
				return OCSFEvidence{
					Policy: ocsf.Policy{},
					ScanActivity: ocsf.ScanActivity{
						Time: time.Now().UnixMilli(),
						Metadata: ocsf.Metadata{
							Product: ocsf.Product{
								Name: &productName,
							},
						},
						Status: &status,
					},
				}
			}(),
			wantErr: true,
		},
		{
			name: "missing product name",
			evidence: func() OCSFEvidence {
				policyName := "test-policy"
				status := "success"
				return OCSFEvidence{
					ScanActivity: ocsf.ScanActivity{
						Time:     time.Now().UnixMilli(),
						Metadata: ocsf.Metadata{},
						Status:   &status,
					},
					Policy: ocsf.Policy{
						Uid:  &policyName,
						Name: &policyName,
					},
				}
			}(),
			wantErr: true,
		},
		{
			name: "missing status",
			evidence: func() OCSFEvidence {
				policyName := "test-policy"
				productName := "test-product"
				return OCSFEvidence{
					ScanActivity: ocsf.ScanActivity{
						Time: time.Now().UnixMilli(),
						Metadata: ocsf.Metadata{
							Product: ocsf.Product{
								Name: &productName,
							},
						},
					},
					Policy: ocsf.Policy{
						Uid:  &policyName,
						Name: &policyName,
					},
				}
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEvidenceFields(tt.evidence)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestStringVal(t *testing.T) {
	tests := []struct {
		name         string
		s            *string
		defaultValue string
		expected     string
	}{
		{
			name:         "valid string",
			s:            stringPtr("test"),
			defaultValue: "default",
			expected:     "test",
		},
		{
			name:         "nil string",
			s:            nil,
			defaultValue: "default",
			expected:     "default",
		},
		{
			name:         "empty string",
			s:            stringPtr(""),
			defaultValue: "default",
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := stringVal(tt.s, tt.defaultValue)
			if result != tt.expected {
				t.Errorf("stringVal() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// Helper functions
func stringPtr(s string) *string {
	return &s
}

func int32Ptr(i int32) *int32 {
	return &i
}

// Helper function to create test evidence
func createTestEvidence() OCSFEvidence {
	policyName := "test-policy"
	productName := "test-product"
	status := "success"

	return OCSFEvidence{
		ScanActivity: ocsf.ScanActivity{
			Time: time.Now().UnixMilli(),
			Metadata: ocsf.Metadata{
				Product: ocsf.Product{
					Name: &productName,
				},
			},
			Status: &status,
		},
		Policy: ocsf.Policy{
			Uid:  &policyName,
			Name: &policyName,
		},
	}
}
