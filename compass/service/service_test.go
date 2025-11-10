package service

import (
	"encoding/json"
	"testing"

	"github.com/complytime/complybeacon/compass/mapper/plugins/basic"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/ossf/gemara/layer2"
	"github.com/ossf/gemara/layer4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complybeacon/compass/api"
	"github.com/complytime/complybeacon/compass/mapper"
)

func TestNewService(t *testing.T) {
	mappers := make(mapper.Set)
	scope := make(mapper.Scope)

	service := NewService(mappers, scope)

	assert.NotNil(t, service)
	assert.Equal(t, mappers, service.set)
	assert.Equal(t, scope, service.scope)
	assert.Equal(t, "1.0.0", service.version)
}

func TestEnrich(t *testing.T) {
	t.Run("Enrichment with mapping", func(t *testing.T) {
		// Load the OpenAPI spec for validation
		swagger, err := api.GetSwagger()
		require.NoError(t, err)

		// Set up a mapper with plans and catalog for successful mapping
		mapperPlugin := basic.NewBasicMapper()
		plans := []layer4.AssessmentPlan{
			{
				Control: layer4.Mapping{EntryId: "AC-1", ReferenceId: "test-catalog"},
				Assessments: []layer4.Assessment{
					{
						Requirement: layer4.Mapping{EntryId: "AC-1-REQ", ReferenceId: "test-catalog"},
						Procedures: []layer4.AssessmentProcedure{
							{
								Id:            "AC-1",
								Documentation: "Test procedure documentation",
							},
						},
					},
				},
			},
		}
		mapperPlugin.AddEvaluationPlan("test-catalog", plans...)

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

		evidence := api.Policy{
			PolicyEngineName: "test-policy-engine",
			PolicyRuleId:     "AC-1",
		}
		scope := mapper.Scope{
			"test-catalog": catalog,
		}

		compliance := mapperPlugin.Map(evidence, scope)
		response := api.EnrichmentResponse{
			Compliance: compliance,
		}

		assert.Equal(t, api.Success, response.Compliance.EnrichmentStatus)
		assert.Equal(t, "AC-1-REQ", response.Compliance.Control.Id)
		assert.Equal(t, "test-catalog", response.Compliance.Control.CatalogId)

		err = validateEnrichmentResponse(t, response, swagger)
		assert.NoError(t, err)
	})

	t.Run("Enrichment Unmapped", func(t *testing.T) {
		swagger, err := api.GetSwagger()
		require.NoError(t, err)

		// Set up a mapper without plans or with empty scope to trigger unmapped response
		mapperPlugin := basic.NewBasicMapper()
		evidence := api.Policy{
			PolicyEngineName: "test-policy-engine",
			PolicyRuleId:     "AC-1",
		}
		scope := make(mapper.Scope)

		compliance := mapperPlugin.Map(evidence, scope)
		response := api.EnrichmentResponse{
			Compliance: compliance,
		}

		assert.Equal(t, api.Unmapped, response.Compliance.EnrichmentStatus)
		assert.Equal(t, "UNMAPPED", response.Compliance.Control.Id)
		assert.Equal(t, "UNMAPPED", response.Compliance.Control.CatalogId)
		assert.Equal(t, "UNCATEGORIZED", response.Compliance.Control.Category)

		err = validateEnrichmentResponse(t, response, swagger)
		assert.NoError(t, err, "Enrichment response with unmapped status should validate against OpenAPI schema")
	})
}

// validateEnrichmentResponse validates an EnrichmentResponse against the OpenAPI schema
func validateEnrichmentResponse(t *testing.T, response api.EnrichmentResponse, swagger *openapi3.T) error {
	t.Helper()
	pathItem := swagger.Paths.Find("/v1/enrich")
	require.NotNil(t, pathItem)
	operation := pathItem.Post
	require.NotNil(t, operation)

	responsesMap := operation.Responses.Map()
	responseRef, ok := responsesMap["200"]
	require.True(t, ok)
	require.NotNil(t, responseRef)

	responseValue := responseRef.Value
	require.NotNil(t, responseValue)

	content := responseValue.Content
	require.NotNil(t, content)

	mediaType := content.Get("application/json")
	require.NotNil(t, mediaType)

	schemaRef := mediaType.Schema
	require.NotNil(t, schemaRef)
	schema := schemaRef.Value
	require.NotNil(t, schema)

	responseJSON, err := json.Marshal(response)
	require.NoError(t, err)

	var responseBody interface{}
	require.NoError(t, json.Unmarshal(responseJSON, &responseBody))
	assert.NoError(t, schema.VisitJSON(responseBody))

	return nil
}
