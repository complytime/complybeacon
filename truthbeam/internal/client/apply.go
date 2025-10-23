package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// ApplyAttributes enriches attributes in the log record with compliance impact data.
func ApplyAttributes(ctx context.Context, client *Client, serverURL string, _ pcommon.Resource, logRecord plog.LogRecord) error {
	attrs := logRecord.Attributes()

	// Retrieve lookup attributes
	var missingAttrs []string

	policyRuleIDVal, ok := attrs.Get(POLICY_RULE_ID)
	if !ok {
		missingAttrs = append(missingAttrs, POLICY_RULE_ID)
	}

	policySourceVal, ok := attrs.Get(POLICY_ENGINE_NAME)
	if !ok {
		missingAttrs = append(missingAttrs, POLICY_ENGINE_NAME)
	}

	policyEvalStatusVal, ok := attrs.Get(POLICY_EVALUATION_RESULT)
	if !ok {
		missingAttrs = append(missingAttrs, POLICY_EVALUATION_RESULT)
	}

	if len(missingAttrs) > 0 {
		return fmt.Errorf("missing required attributes: %s", strings.Join(missingAttrs, ", "))
	}

	enrichReq := EnrichmentRequest{
		Evidence: Evidence{
			Timestamp:              logRecord.Timestamp().AsTime(),
			PolicyEngineName:       policySourceVal.Str(),
			PolicyRuleId:           policyRuleIDVal.Str(),
			PolicyEvaluationStatus: EvidencePolicyEvaluationStatus(policyEvalStatusVal.Str()),
		},
	}

	enrichRes, err := callEnrichAPI(ctx, client, serverURL, enrichReq)
	if err != nil {
		return err
	}

	// Add enrichment status
	attrs.PutStr(COMPLIANCE_ENRICHMENT_STATUS, string(enrichRes.Compliance.EnrichmentStatus))

	// Only add compliance attributes if enrichment was successful
	if enrichRes.Compliance.EnrichmentStatus == ComplianceEnrichmentStatusSuccess {
		attrs.PutStr(COMPLIANCE_STATUS, string(enrichRes.Compliance.Status))
		attrs.PutStr(COMPLIANCE_CONTROL_ID, enrichRes.Compliance.Control.Id)
		attrs.PutStr(COMPLIANCE_CONTROL_CATALOG_ID, enrichRes.Compliance.Control.CatalogId)
		attrs.PutStr(COMPLIANCE_CONTROL_CATEGORY, enrichRes.Compliance.Control.Category)
		requirements := attrs.PutEmptySlice(COMPLIANCE_REQUIREMENTS)
		standards := attrs.PutEmptySlice(COMPLIANCE_FRAMEWORKS)

		if enrichRes.Compliance.Control.RemediationDescription != nil {
			attrs.PutStr(COMPLIANCE_REMEDIATION_DESCRIPTION, *enrichRes.Compliance.Control.RemediationDescription)
		}

		for _, req := range enrichRes.Compliance.Frameworks.Requirements {
			newReq := requirements.AppendEmpty()
			newReq.SetStr(req)
		}
		for _, std := range enrichRes.Compliance.Frameworks.Frameworks {
			newStd := standards.AppendEmpty()
			newStd.SetStr(std)
		}
	}

	return nil
}

// callEnrichAPI is a helper function to perform the actual HTTP request.
func callEnrichAPI(ctx context.Context, client *Client, serverURL string, req EnrichmentRequest) (*EnrichmentResponse, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	// Create the HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", serverURL+"/v1/enrich", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Perform the request
	resp, err := client.Client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Handle non-200 status codes
	if resp.StatusCode != http.StatusOK {
		var errRes Error
		err := json.NewDecoder(resp.Body).Decode(&errRes)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("API call failed with status %d: %v", resp.StatusCode, errRes.Message)
	}

	// Decode the successful response
	var enrichRes EnrichmentResponse
	if err := json.NewDecoder(resp.Body).Decode(&enrichRes); err != nil {
		return nil, err
	}

	return &enrichRes, nil
}
