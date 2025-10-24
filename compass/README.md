# `compass` API

## Overview

The `compass` API provides compliance context enrichment for OpenTelemetry log records. 

## Usage

The `compass` API can be deployed as an independent API, enabling it to be called in any OpenTelemetry-based log enrichment pipeline.

> **Note:** The `compass` API commonly receives an enrichment request from the `truthbeam` processor. The `compass` API will perform policy look-ups, and return compliance-context attributes that can be injected back into the log records using the `truthbeam` processor.

## Example Enrichment Process

1. **Log Record:** `{policy.id: "github_branch_protection", policy.decision: "fail"}`
2. **Enrichment Request:** `{evidence: {policyId: "github_branch_protection", decision: "fail"}}`
3. **Compass API Response:** `{compliance: {catalog: "NIST-800-53", control: "AC-2"}, status: {title: "Fail"}}`
4. **Enriched Log:** `{policy.id: "github_branch_protection", compliance.status: "Fail", compliance.control: "AC-2"}`

> Review guidelines for writing tests in the [DEVELOPMENT.md](https://github.com/complytime/complybeacon/blob/main/docs/DEVELOPMENT.md).