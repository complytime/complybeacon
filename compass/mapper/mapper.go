package mapper

import (
	"github.com/ossf/gemara/layer2"
	"github.com/ossf/gemara/layer4"

	"github.com/complytime/complybeacon/compass/api"
)

// Mapper defines a set of methods a plugin must implement for
// mapper Policy into a `gemara` AssessmentPlan.
type Mapper interface {
	PluginName() ID
	Map(policy api.Policy, scope Scope) api.Compliance
	AddEvaluationPlan(catalogId string, plans ...layer4.AssessmentPlan)
}

// ID represents the identity for a transformer.
type ID string

// NewID returns a new ID for a given id string.
func NewID(id string) ID {
	return ID(id)
}

// Set defines Transformers by ID
type Set map[ID]Mapper

// Scope defined in scope Layer2 Catalogs by the
// catalog ID
type Scope map[string]layer2.Catalog
