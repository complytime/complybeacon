# OpenTelemetry Attributes

This defines a set of attributes used for raw evidence metadata and risk context.

## Raw Evidence

| Attribute Name    | Type     | Description                                                                                             |
|:------------------|:---------|:--------------------------------------------------------------------------------------------------------|
| `evidence.id`     | `string` | A unique identifier for the evidence. This value is used to enrich the log record with compliance data. |
| `policy.id`       | `string` | The identifier for the policy that was applied.                                                         |
| `policy.decision` | `string` | The outcome of the policy evaluation (e.g., "deny", "allow").                                           |
| `policy.source`   | `string` | The source of the policy (e.g., a file path or URL).                                                    |


## Risk Context

| Attribute Name            | Type       | Description                                                       |
|:--------------------------|:-----------|:------------------------------------------------------------------|
| `compliance.result`       | `string`   | The overall compliance result from the enrichment API.            |
| `compliance.baselines`    | `string[]` | An array of identifiers for the impacted compliance baselines.    |
| `compliance.requirements` | `string[]` | An array of identifiers for the impacted compliance requirements. |


https://schema.ocsf.io/1.5.0/objects/compliance
https://schema.ocsf.io/1.5.0/objects/attack