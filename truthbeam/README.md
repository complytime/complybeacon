# `truthbeam` processor

## Overview

The `truthbeam` custom OpenTelemetry Processor is a component in the OpenTelemetry Pipeline that ingests and validates normalized logs for required attributes. It then formulates an enrichment request to query the `compass` API. Once enriched with compliance-context attributes from `compass`, `truthbeam` adds these new attributes back to the original log record.

## Usage

The `truthbeam` processor can be integrated into any OpenTelemetry Collector distribution.

> **Note:** The `truthbeam` processor **gracefully** handles API failures to ensure log records won't be discarded.

### Example Code Snippet **Log -> Enrichment Request -> Enrichment Response -> Enriched Log**

**Log Record:** The log record from the `sameple_logs.json` is an example of a log record that would be ingested by the `truthbeam` processor. 

```json
 "attributes": [
                {
                  "key": "policy.id",
                  "value": {
                    "stringValue": "github_branch_protection"
                  }
                },
                {
                  "key": "policy.evaluation.status",
                  "value": {
                    "stringValue": "fail"
                  }
                },
```

**Enrichment Request:** The enrichment request is formed by the `truthbeam` processor based on the log record from the policy-engine source.

```json
{
  "evidence": {
    "timestamp": "2025-01-05T12:30:00Z",
    "source": "conforma",
    "policyId": "github_branch_protection",
    "decision": "fail",
    "action": "audit",
    "categoryId": 6,
    "classId": 6007
  }
}
```

**Enrichment Response:** The enrichment response is the response from the `compass` API.

```json
{
  "compliance": {
    "catalog": "NIST-800-53",
    "control": "AC-2"
  },
  "status": {
    "title": "Fail"
  }
}
```

**Enriched Log:** The `truthbeam` processor adds the enrichment response as attributes to the log record.

## Development

> Review guidelines for writing tests in the [DEVELOPMENT.md](https://github.com/complytime/complybeacon/blob/main/docs/DEVELOPMENT.md).