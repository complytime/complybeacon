package basic

import (
	"github.com/ossf/gemara/layer2"
	"github.com/ossf/gemara/layer4"

	"github.com/complytime/complybeacon/compass/api"
	"github.com/complytime/complybeacon/compass/mapper"
)

// A basic mapper provide context in a shallow manner by parsing the known attributes.

var (
	_  mapper.Mapper = (*Mapper)(nil)
	ID               = mapper.NewID("basic")
)

type Mapper struct {
	plans map[string][]layer4.AssessmentPlan
}

func (m *Mapper) AddEvaluationPlan(catalogId string, plans []layer4.AssessmentPlan) {
	existingPlans, ok := m.plans[catalogId]
	if !ok {
		m.plans[catalogId] = plans
	} else {
		existingPlans = append(existingPlans, plans...)
		m.plans[catalogId] = existingPlans
	}
}

func NewBasicMapper() *Mapper {
	return &Mapper{
		plans: make(map[string][]layer4.AssessmentPlan),
	}
}

func (m *Mapper) PluginName() mapper.ID {
	return ID
}

func (m *Mapper) Map(evidence api.RawEvidence, scope mapper.Scope) (api.Compliance, api.Status) {
	var (
		status   api.StatusTitle
		statusId api.StatusId
	)

	switch evidence.Decision {
	case "pass", "Pass", "success":
		status = api.Pass
		statusId = api.N1
	case "fail", "Fail", "failure":
		status = api.Fail
		statusId = api.N2
	case "Other", "Warning", "Unknown":
		status = api.Warning
		statusId = api.N3
	}

	for catalogId, plans := range m.plans {
		catalog, ok := scope[catalogId]
		if !ok {
			continue
		}

		proceduresById := make(map[string]struct {
			ControlID     string
			RequirementID string
			Documentation string
		})

		for _, plan := range plans {
			for _, requirement := range plan.Assessments {
				for _, procedure := range requirement.Procedures {
					proceduresById[procedure.Id] = struct {
						ControlID     string
						RequirementID string
						Documentation string
					}{
						ControlID:     plan.ControlId,
						RequirementID: requirement.RequirementId,
						Documentation: procedure.Documentation,
					}
				}
			}
		}

		controlData := make(map[string]struct {
			Mappings []layer2.Mapping
			Category string
		})

		for _, family := range catalog.ControlFamilies {
			for _, control := range family.Controls {
				controlData[control.Id] = struct {
					Mappings []layer2.Mapping
					Category string
				}{
					Mappings: control.GuidelineMappings,
					Category: family.Title,
				}
			}
		}

		if procedureInfo, ok := proceduresById[evidence.PolicyId]; ok {

			if ctrlData, ok := controlData[procedureInfo.ControlID]; ok {

				var requirements, standards []string
				for _, mapping := range ctrlData.Mappings {
					standards = append(standards, mapping.ReferenceId)
					for _, entry := range mapping.Entries {
						requirements = append(requirements, entry.ReferenceId)
					}
				}

				compliance := api.Compliance{
					Benchmark:    catalogId,
					Control:      procedureInfo.RequirementID,
					Requirements: requirements,
					Standards:    standards,
					Category:     ctrlData.Category,
					Remediation:  &procedureInfo.Documentation,
				}
				return compliance, api.Status{Title: status, Id: &statusId}
			}
		}
	}

	return api.Compliance{}, api.Status{Title: status, Id: &statusId}
}
