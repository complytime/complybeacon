package proofwatch

import ocsf "github.com/Santiago-Labs/go-ocsf/ocsf/v1_5_0"

// OCSF-based evidence structured, with some security control profile fields. Attributes for `compliance` findings
// by the `compass` service based on `gemara` based during pipeline enrichment.

type Evidence struct {
	ocsf.ScanActivity `json:",inline"`
	// From the security-control profile
	Policy   ocsf.Policy `json:"policy" parquet:"policy"`
	Action   *string     `json:"action,omitempty" parquet:"action,optional"`
	ActionID *int32      `json:"action_id,omitempty" parquet:"action_id,optional"`
}
