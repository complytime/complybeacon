package applier

import "github.com/ossf/gemara/layer4"

// Status defines the compliance status value
type Status int

const (
	// Unknown is the default status when a status is not explicit defined.
	Unknown Status = iota
	// Compliant defines then status when a  resource in compliant
	Compliant
	// NotCompliant define the status when a resource is not compliant.
	NotCompliant
	// NotApplicable define the status when a resource does not fall into any applicability category.
	NotApplicable
	// Exempt defines the status when a resource has an active compliance exception.
	Exempt
)

var toString = map[Status]string{
	Compliant:     "Compliant",
	NotCompliant:  "Non-Compliant",
	NotApplicable: "Not Applicable",
	Exempt:        "Exempt",
	Unknown:       "Unknown",
}

func (s Status) String() string {
	return toString[s]
}

func parseResult(resultStr string) layer4.Result {
	switch resultStr {
	case "Not Run":
		return layer4.NotRun
	case "Not Applicable":
		return layer4.NotApplicable
	case "Passed":
		return layer4.Passed
	case "Failed":
		return layer4.Failed
	default:
		return layer4.Unknown
	}
}

func mapResult(resultStr string) Status {
	result := parseResult(resultStr)
	switch result {
	case layer4.Passed:
		return Compliant
	case layer4.Failed:
		return NotCompliant
	case layer4.NotApplicable, layer4.NotRun:
		return NotApplicable
	default:
		return Unknown
	}
}
