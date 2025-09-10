package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
)

// ApplyAttributes enriches attributes in the log record with compliance impact data.
func ApplyAttributes(ctx context.Context, client *Client, serverURL string, _ pcommon.Resource, logRecord plog.LogRecord) error {
	attrs := logRecord.Attributes()

	evidenceIDVal, ok := attrs.Get("evidence.id")
	if !ok {
		return fmt.Errorf("missing attribute 'evidence.id'")
	}

	policyIDVal, ok := attrs.Get("policy.id")
	if !ok {
		return fmt.Errorf("missing attribute 'policy.id'")
	}

	policyDecisionVal, ok := attrs.Get("policy.decision")
	if !ok {
		return fmt.Errorf("missing attribute 'policy.decision'")
	}

	policySourceVal, ok := attrs.Get("policy.source")
	if !ok {
		return fmt.Errorf("missing attribute 'policy.source'")
	}

	// Default base event data
	categoryIDVal, ok := attrs.Get("category.id")
	if !ok {
		categoryIDVal = pcommon.NewValueInt(0)
	}

	classIDVal, ok := attrs.Get("class.id")
	if !ok {
		classIDVal = pcommon.NewValueInt(0)
	}

	categoryId := int(categoryIDVal.Int())
	classId := int(classIDVal.Int())

	logData, err := json.Marshal(attrs.AsRaw())
	if err != nil {
		return fmt.Errorf("error marshalling raw log data: %w", err)
	}

	enrichReq := EnrichmentRequest{
		Evidence: RawEvidence{
			Id:         evidenceIDVal.Str(),
			Timestamp:  logRecord.Timestamp().AsTime(),
			CategoryId: &categoryId,
			ClassId:    &classId,
			Source:     policySourceVal.Str(),
			PolicyId:   policyIDVal.Str(),
			Decision:   policyDecisionVal.Str(),
			RawData:    logData,
		},
	}

	enrichRes, err := callEnrichAPI(ctx, client, serverURL, enrichReq)
	if err != nil {
		return err
	}

	attrs.PutStr("compliance.status", string(enrichRes.Status.Title))
	baselines := attrs.PutEmptySlice("compliance.baselines")
	requirements := attrs.PutEmptySlice("compliance.requirements")
	standards := attrs.PutEmptySlice("compliance.standards")
	controls := attrs.PutEmptySlice("compliance.controls")

	for _, impacted := range enrichRes.Compliance {
		newBench := baselines.AppendEmpty()
		newBench.SetStr(impacted.Benchmark)
		newControl := controls.AppendEmpty()
		newControl.SetStr(impacted.Control)
		for _, req := range impacted.Requirements {
			newReq := requirements.AppendEmpty()
			newReq.SetStr(req)
		}
		for _, std := range impacted.Standards {
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
