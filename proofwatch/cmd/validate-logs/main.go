package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	ocsf "github.com/Santiago-Labs/go-ocsf/ocsf/v1_5_0"
	"github.com/ossf/gemara/layer4"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/otel/attribute"

	"github.com/complytime/complybeacon/proofwatch"
)

// generateGemaraLog creates a sample Gemara log record with compliance attributes
func generateGemaraLog() plog.Logs {
	evidence := proofwatch.GemaraEvidence{
		Metadata: layer4.Metadata{
			Id: "test-assessment-001",
			Author: layer4.Author{
				Name: "test-policy-engine",
			},
		},
		AssessmentLog: layer4.AssessmentLog{
			Requirement: layer4.Mapping{
				EntryId:     "OSPS-QA-07.01",
				ReferenceId: "OSPS-B",
			},
			Procedure: layer4.Mapping{
				EntryId: "deny-root-user",
			},
			Result:  layer4.Passed,
			Message: "Policy evaluation completed successfully",
			End:     layer4.Datetime(time.Now().Format(time.RFC3339)),
		},
	}

	attrs := evidence.Attributes()
	jsonData, _ := evidence.ToJSON()
	return createLogFromAttributes(attrs, jsonData)
}

// generateOCSFLog creates a sample OCSF log record with compliance attributes
func generateOCSFLog() plog.Logs {
	status := "success"
	policyName := "test-policy"
	policyUID := "github_branch_protection"
	productName := "test-engine"

	evidence := proofwatch.OCSFEvidence{
		ScanActivity: ocsf.ScanActivity{
			Time: time.Now().UnixMilli(),
			Metadata: ocsf.Metadata{
				Product: ocsf.Product{
					Name: &productName,
				},
			},
			Status:  &status,
			Message: stringPtr("Policy check completed"),
		},
		Policy: ocsf.Policy{
			Name: &policyName,
			Uid:  &policyUID,
		},
	}

	attrs := evidence.Attributes()
	jsonData, _ := evidence.ToJSON()
	return createLogFromAttributes(attrs, jsonData)
}

// createLogFromAttributes creates OTLP log records from attributes
func createLogFromAttributes(attrs []attribute.KeyValue, body []byte) plog.Logs {
	logs := plog.NewLogs()
	rl := logs.ResourceLogs().AppendEmpty()
	resource := rl.Resource()
	resource.Attributes().PutStr("service.name", "test-service")

	sl := rl.ScopeLogs().AppendEmpty()
	scope := sl.Scope()
	scope.SetName("github.com/complytime/complybeacon/proofwatch")

	lr := sl.LogRecords().AppendEmpty()
	lr.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	lr.SetObservedTimestamp(pcommon.NewTimestampFromTime(time.Now()))
	lr.SetSeverityNumber(plog.SeverityNumberInfo)
	lr.SetSeverityText("INFO")
	lr.Body().SetStr(string(body))

	// Add attributes to log record
	for _, attr := range attrs {
		switch attr.Value.Type() {
		case attribute.STRING:
			lr.Attributes().PutStr(string(attr.Key), attr.Value.AsString())
		case attribute.INT64:
			lr.Attributes().PutInt(string(attr.Key), attr.Value.AsInt64())
		case attribute.BOOL:
			lr.Attributes().PutBool(string(attr.Key), attr.Value.AsBool())
		}
	}

	return logs
}

// simulateTruthBeamEnrichment simulates TruthBeam enrichment by adding compliance attributes
// This mimics what TruthBeam does after calling the Compass API
// It adds comprehensive test data covering various enum values
func simulateTruthBeamEnrichment(logs plog.Logs) plog.Logs {
	rl := logs.ResourceLogs()
	logRecordIndex := 0
	for i := 0; i < rl.Len(); i++ {
		rs := rl.At(i)
		ilss := rs.ScopeLogs()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			logRecords := ils.LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				lr := logRecords.At(k)

				// Simulate TruthBeam enrichment - add compliance attributes
				// These would normally come from the Compass API via ApplyAttributes

				// Use different enum values based on log record index to test various scenarios
				switch logRecordIndex % 3 {
				case 0:
					// First log: COMPLIANT scenario
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_STATUS, "Compliant")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_RISK_LEVEL, "Low")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_REMEDIATION_ACTION, "Allow")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_REMEDIATION_STATUS, "Success")
				case 1:
					// Second log: NON_COMPLIANT scenario
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_STATUS, "Non-Compliant")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_RISK_LEVEL, "High")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_REMEDIATION_ACTION, "Block")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_REMEDIATION_STATUS, "Success")
				default:
					// Third log: NOT_APPLICABLE scenario
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_STATUS, "Not Applicable")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_RISK_LEVEL, "Informational")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_REMEDIATION_ACTION, "Waive")
					lr.Attributes().PutStr(proofwatch.COMPLIANCE_REMEDIATION_STATUS, "Skipped")
				}

				// Common attributes for all logs
				lr.Attributes().PutStr(proofwatch.COMPLIANCE_CONTROL_ID, "OSPS-QA-07.01")
				lr.Attributes().PutStr(proofwatch.COMPLIANCE_CONTROL_CATALOG_ID, "OSPS-B")
				lr.Attributes().PutStr(proofwatch.COMPLIANCE_CONTROL_CATEGORY, "Access Control")
				lr.Attributes().PutStr(proofwatch.COMPLIANCE_ENRICHMENT_STATUS, "Success")
				lr.Attributes().PutStr(proofwatch.COMPLIANCE_ASSESSMENT_ID, fmt.Sprintf("test-assessment-%03d", logRecordIndex))

				// Add arrays for frameworks and requirements
				frameworks := lr.Attributes().PutEmptySlice(proofwatch.COMPLIANCE_FRAMEWORKS)
				frameworks.AppendEmpty().SetStr("NIST-800-53")
				if logRecordIndex%2 == 0 {
					frameworks.AppendEmpty().SetStr("ISO-27001")
				}

				requirements := lr.Attributes().PutEmptySlice(proofwatch.COMPLIANCE_REQUIREMENTS)
				requirements.AppendEmpty().SetStr("AC-1")
				if logRecordIndex%2 == 0 {
					requirements.AppendEmpty().SetStr("AC-2")
				}

				logRecordIndex++
			}
		}
	}
	return logs
}

// convertToWeaverFormat converts plog.Logs to weaver attribute format
func convertToWeaverFormat(logs plog.Logs) ([]byte, error) {
	// Extract attributes from log records and convert to weaver format
	// Weaver expects an array of attribute objects with "name" and "value" fields
	var attributes []map[string]interface{}

	rl := logs.ResourceLogs()
	for i := 0; i < rl.Len(); i++ {
		rs := rl.At(i)
		ilss := rs.ScopeLogs()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			logRecords := ils.LogRecords()
			for k := 0; k < logRecords.Len(); k++ {
				lr := logRecords.At(k)

				// Extract all attributes from the log record
				lr.Attributes().Range(func(k string, v pcommon.Value) bool {
					attrValue := valueToString(v)
					attributes = append(attributes, map[string]interface{}{
						"attribute": map[string]interface{}{
							"name":  k,
							"value": attrValue,
						},
					})
					return true
				})
			}
		}
	}

	return json.Marshal(attributes)
}

func valueToString(v pcommon.Value) interface{} {
	switch v.Type() {
	case pcommon.ValueTypeStr:
		return v.Str()
	case pcommon.ValueTypeInt:
		return v.Int()
	case pcommon.ValueTypeDouble:
		return v.Double()
	case pcommon.ValueTypeBool:
		return v.Bool()
	case pcommon.ValueTypeSlice:
		var arr []interface{}
		sl := v.Slice()
		for i := 0; i < sl.Len(); i++ {
			elem := sl.At(i)
			if elem.Type() == pcommon.ValueTypeStr {
				arr = append(arr, elem.Str())
			}
		}
		return arr
	default:
		return v.AsString()
	}
}

func stringPtr(s string) *string {
	return &s
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <gemara|ocsf|both> [output-file]\n", os.Args[0])
		os.Exit(1)
	}

	format := os.Args[1]
	outputFile := "stdout"
	if len(os.Args) > 2 {
		outputFile = os.Args[2]
	}

	ctx := context.Background()
	_ = ctx // Suppress unused variable warning
	var allLogs []plog.Logs

	if format == "gemara" || format == "both" {
		gemaraLogs := generateGemaraLog()
		enrichedGemara := simulateTruthBeamEnrichment(gemaraLogs)
		allLogs = append(allLogs, enrichedGemara)
	}

	if format == "ocsf" || format == "both" {
		ocsfLogs := generateOCSFLog()
		enrichedOCSF := simulateTruthBeamEnrichment(ocsfLogs)
		allLogs = append(allLogs, enrichedOCSF)
	}

	// Combine all logs into a single OTLP export
	var combinedLogs plog.Logs
	if len(allLogs) > 0 {
		combinedLogs = allLogs[0]
		for i := 1; i < len(allLogs); i++ {
			allLogs[i].ResourceLogs().MoveAndAppendTo(combinedLogs.ResourceLogs())
		}
	}

	jsonBytes, err := convertToWeaverFormat(combinedLogs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error converting logs: %v\n", err)
		os.Exit(1)
	}

	if outputFile == "stdout" {
		fmt.Print(string(jsonBytes))
	} else {
		if err := os.WriteFile(outputFile, jsonBytes, 0600); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Generated test logs written to %s\n", outputFile)
	}
}
