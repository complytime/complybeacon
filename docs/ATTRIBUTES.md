# OpenTelemetry Attributes

This defines a set of attributes used for raw evidence metadata and risk context.

## Raw Evidence

| Attribute Name    | Type      | Description                                                                                             |
|:------------------|:----------|:--------------------------------------------------------------------------------------------------------|
| `evidence.id`     | `string`  | A unique identifier for the evidence. This value is used to enrich the log record with compliance data. |
| `policy.id`       | `string`  | The identifier for the policy that was applied.                                                         |
| `policy.decision` | `string`  | The outcome of the policy evaluation (e.g., "deny", "allow").                                           |
| `policy.source`   | `string`  | The source of the policy.                                                                               |
| `category.id`     | `integer` | A unique identifier for the event's category, providing a high-level grouping.                          |
| `class.id`        | `integer` | A unique identifier for the event's class, providing a more granular event type.                        |


## Compliance Context

From https://schema.ocsf.io/1.5.0/objects/compliance

| Attribute Name            | Type       | Description                                                                                   |
|:--------------------------|:-----------|:----------------------------------------------------------------------------------------------|
| `compliance.status`       | `string`   | The normalized status identifier of the compliance check                                      |
| `compliance.control`      | `string`   | A Control is a prescriptive, actionable set of specifications that strengthens device posture |
| `compliance.benchmarks`   | `string`   | A security catalog identifier                                                                 |
| `compliance.requirements` | `string[]` | The specific compliance requirements being evaluated                                          |
| `compliance.standards`    | `string[]` | The regulatory or industry standards being evaluated for compliance.                          |
| `compliance.category`     | string     | The category a control framework pertains                                                     |
| `remediation.desc`        | string     | The description of the remediation strategy.                                                  |

