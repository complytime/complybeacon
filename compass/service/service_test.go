package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/complytime/complybeacon/compass/api"
	"github.com/complytime/complybeacon/compass/mapper"
	"github.com/complytime/complybeacon/compass/mapper/plugins/basic"
)

func TestNewService(t *testing.T) {
	mappers := make(mapper.Set)
	scope := make(mapper.Scope)

	service := NewService(mappers, scope)

	assert.NotNil(t, service)
	assert.Equal(t, mappers, service.set)
	assert.Equal(t, scope, service.scope)
}

func TestEnrich(t *testing.T) {
	t.Run("enriches evidence with compliance data", func(t *testing.T) {
		evidence := api.Evidence{
			PolicyEngineName:       "test-policy-engine",
			PolicyRuleId:           "AC-1",
			PolicyEvaluationStatus: api.EvidencePolicyEvaluationStatusPassed,
			Timestamp:              time.Now(),
		}
		scope := make(mapper.Scope)
		mapperPlugin := basic.NewBasicMapper()

		// Enrich the evidence with the basic mapper
		response := enrich(evidence, mapperPlugin, scope)

		assert.NotEmpty(t, response)
		assert.NotEmpty(t, response.Compliance)
		// Compliance may be empty - expected behavior for basic mapper
	})
}
