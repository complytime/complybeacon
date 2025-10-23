package basic

import (
	"testing"
	"time"

	"github.com/ossf/gemara/layer2"
	"github.com/ossf/gemara/layer4"
	"github.com/stretchr/testify/assert"

	"github.com/complytime/complybeacon/compass/api"
	"github.com/complytime/complybeacon/compass/mapper"
)

func TestNewBasicMapper(t *testing.T) {
	basicMapper := NewBasicMapper()

	assert.NotNil(t, basicMapper)
	assert.Equal(t, ID, basicMapper.PluginName())
	assert.NotNil(t, basicMapper.plans)
	assert.Empty(t, basicMapper.plans)
}

func TestBasicMapper_MapWithPlans(t *testing.T) {
	tests := []struct {
		name           string
		status         api.EvidencePolicyEvaluationStatus
		expectedStatus api.ComplianceStatus
	}{
		{
			name:           "compliance status is passed",
			status:         api.EvidencePolicyEvaluationStatusPassed,
			expectedStatus: api.COMPLIANT,
		},
		{
			name:           "compliance status is failed",
			status:         api.EvidencePolicyEvaluationStatusFailed,
			expectedStatus: api.NONCOMPLIANT,
		},
		{
			name:           "compliance status is not run",
			status:         api.EvidencePolicyEvaluationStatusNotRun,
			expectedStatus: api.NOTAPPLICABLE,
		},
		{
			name:           "compliance status is not applicable",
			status:         api.EvidencePolicyEvaluationStatusNotApplicable,
			expectedStatus: api.NOTAPPLICABLE,
		},
		{
			name:           "unmapped compliance status defaults to unknown",
			status:         api.EvidencePolicyEvaluationStatusUnknown,
			expectedStatus: api.UNKNOWN,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			basicMapper := NewBasicMapper()

			// Add a test plan
			plans := []layer4.AssessmentPlan{
				{
					Control: layer4.Mapping{EntryId: "AC-1", ReferenceId: "test-catalog"},
					Assessments: []layer4.Assessment{
						{
							Requirement: layer4.Mapping{EntryId: "AC-1-REQ", ReferenceId: "test-catalog"},
							Procedures: []layer4.AssessmentProcedure{
								{
									Id:            "AC-1",
									Documentation: "Test procedure",
								},
							},
						},
					},
				},
			}
			basicMapper.AddEvaluationPlan("test-catalog", plans...)

			// Create a test catalog
			catalog := layer2.Catalog{
				Metadata: layer2.Metadata{Id: "test-catalog"},
				ControlFamilies: []layer2.ControlFamily{
					{
						Title: "Access Control",
						Controls: []layer2.Control{
							{
								Id: "AC-1",
								GuidelineMappings: []layer2.Mapping{
									{
										ReferenceId: "NIST-800-53",
										Entries: []layer2.MappingEntry{
											{ReferenceId: "AC-1"},
										},
									},
								},
							},
						},
					},
				},
			}

			evidence := api.Evidence{
				PolicyEngineName:       "test-policy-engine",
				PolicyRuleId:           "AC-1",
				PolicyEvaluationStatus: tt.status,
				Timestamp:              time.Now(),
			}
			scope := mapper.Scope{
				"test-catalog": catalog,
			}

			compliance := basicMapper.Map(evidence, scope)

			assert.NotNil(t, compliance)
			assert.Equal(t, tt.expectedStatus, compliance.Status)
			assert.Equal(t, api.ComplianceEnrichmentStatusSuccess, compliance.EnrichmentStatus)
			assert.Equal(t, "AC-1-REQ", compliance.Control.Id)
			assert.Equal(t, "Access Control", compliance.Control.Category)
			assert.Equal(t, "test-catalog", compliance.Control.CatalogId)
		})
	}
}

func TestBasicMapper_MapUnmapped(t *testing.T) {
	basicMapper := NewBasicMapper()
	evidence := api.Evidence{
		PolicyEngineName:       "test-policy-engine",
		PolicyRuleId:           "AC-1",
		PolicyEvaluationStatus: api.EvidencePolicyEvaluationStatusFailed,
		Timestamp:              time.Now(),
	}
	scope := make(mapper.Scope)

	compliance := basicMapper.Map(evidence, scope)

	// For basic mapper without plans, we expect an empty compliance object
	// with only enrichment status set to "unmapped"
	assert.NotNil(t, compliance)
	assert.Equal(t, api.ComplianceEnrichmentStatusUnmapped, compliance.EnrichmentStatus)
	// Status should be empty when no plans are configured
	assert.Equal(t, api.ComplianceStatus(""), compliance.Status)
}

func TestBasicMapper_AddEvaluationPlan(t *testing.T) {
	t.Run("adds evaluation plan", func(t *testing.T) {
		basicMapper := NewBasicMapper()
		plans := []layer4.AssessmentPlan{
			{Control: layer4.Mapping{ReferenceId: "AC-1"}},
		}

		basicMapper.AddEvaluationPlan("test-catalog", plans...)

		assert.Len(t, basicMapper.plans, 1)
		assert.Contains(t, basicMapper.plans, "test-catalog")
		assert.Len(t, basicMapper.plans["test-catalog"], 1)
		assert.Equal(t, "AC-1", basicMapper.plans["test-catalog"][0].Control.ReferenceId)
	})

	t.Run("appends to existing evaluation plans", func(t *testing.T) {
		basicMapper := NewBasicMapper()
		initialPlans := []layer4.AssessmentPlan{
			{Control: layer4.Mapping{ReferenceId: "AC-1"}},
		}
		additionalPlans := []layer4.AssessmentPlan{
			{Control: layer4.Mapping{ReferenceId: "AC-2"}},
		}

		basicMapper.AddEvaluationPlan("test-catalog", initialPlans...)
		basicMapper.AddEvaluationPlan("test-catalog", additionalPlans...)

		assert.Len(t, basicMapper.plans, 1)
		assert.Len(t, basicMapper.plans["test-catalog"], 2)
		assert.Equal(t, "AC-1", basicMapper.plans["test-catalog"][0].Control.ReferenceId)
		assert.Equal(t, "AC-2", basicMapper.plans["test-catalog"][1].Control.ReferenceId)
	})
}
