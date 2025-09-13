# ComplyBeacon Design Documentation

## Key Features

- **OpenTelemetry Native**: Built on the OpenTelemetry standard for seamless integration with existing observability pipelines.
- **Automated Enrichment**: Enriches raw evidence with risk scores, threat mappings, and regulatory requirements via the Compass service.
- **Composability**: Components are designed as a toolkit; they are not required to be used together, and users can compose their own pipelines.
- **Compliance-as-Code**: Leverages the `gemara` model for a robust, auditable, and automated approach to risk assessment.

## Architecture Overview

### Design Principles

* **Modularity:** The system is composed of small, focused, and interchangeable services.

* **Standardization:** The architecture is built on OpenTelemetry to ensure broad compatibility and interoperability.

* **Operational Experience:** The toolkit is built for easy deployment, configuration, and maintenance using familiar cloud-native practices and protocols.

### Data Flow

### Data Flow

The ComplyBeacon architecture is centered around a unified enrichment pipeline that processes and enriches compliance evidence. The primary data flow begins with a source that generates OpenTelemetry-compliant logs.

1.  **Log Ingestion**: A source generates OCSF-compliant evidence and sends it as a structured log record to the `Beacon` collector, typically using `ProofWatch` to handle the emission. This can also be done by an OpenTelemetry collector agent.
2.  **Enrichment Request**: The log record is received by the `Beacon` collector and forwarded to the `truthbeam` processor. `truthbeam` extracts key attributes from the record and sends an enrichment request to the `Compass` API.
3.  **Enrichment Lookup**: The `Compass` service performs a lookup based on the provided attributes and returns a response containing compliance-related context (e.g., impacted baselines, requirements, and a compliance result).
4.  **Attribute Injection**: `truthbeam` adds these new attributes from `Compass` to the original log record.
5.  **Export**: The now-enriched log record is exported from the `Beacon` collector to a final destination (e.g., a SIEM, logging backend, or data lake) for analysis and correlation.

```
┌───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                                                   │
│                                                                                                                   │
│                                                    ┌─────────────────────────┐                                    │
│                                                    │                         │                                    │
│                                                    │ Beacon Collector Distro │                                    │
│   ┌────────────────────┐   ┌───────────────────┐   │                         │                                    │
│   │                    │   │                   │   ├─────────────────────────┤                                    │
│   │                    ├───┤    ProofWatch     ├───┼────┐                    │                                    │
│   │                    │   │                   │   │    │                    │                                    │
│   │    Policy Log      │   └───────────────────┘   │   ┌┴─────────────────┐  │                                    │
│   │    Source App      │                           │   │                  │  │                                    │
│   │                    │                           │   │      OTLP        │  │                                    │
│   │                    │                           │   │      Reciever    │  │                                    │
│   │                    │  ┌────────────────────────┼───┤                  │  │                                    │
│   └────────────────────┘  │                        │   └────────┬─────────┘  │               ┌─────────────┐      │
│                           │                        │            │            │               │             │      │
│                           │                        │   ┌────────┴─────────┐  │               │             │      │
│                           │                        │   │                  │  │               │ Compass API │      │
│                           │                        │   │    TruthBeam     │──┼──────────────►│             │      │
│   ┌───────────────────────┴───┐                    │   │    Processor     │  │               │             │      │
│   │                           │                    │   │                  │  │               └─────────────┘      │
│   │                           │                    │   └────────┬─────────┘  │                                    │
│   │      OpenTelemetry        │                    │            │            │                                    │
│   │      Collector Agent      │                    │   ┌────────┴─────────┐  │                                    │
│   │                           │                    │   │    Exporter      │  │                                    │
│   │                           │                    │   │   (e.g. Loki     │  │                                    │
│   │                           │                    │   │   Splunk)        │  │                                    │
│   │                           │                    │   └──────────────────┘  │                                    │
│   │                           │                    └─────────────────────────┘                                    │
│   └───────────────────────────┘                                                                                   │
│                                                                                                                   │
└───────────────────────────────────────────────────────────────────────────────────────────────────────────────────┘
```

### Deployment Patterns

ComplyBeacon is designed to be a flexible toolkit. Its components can be used in different combinations to fit a variety of operational needs.

* **Full Pipeline**: The most common use case where `ProofWatch` emits events to the `Beacon` collector, which in turn uses `TruthBeam` and `Compass` to enrich and export logs to a final destination.
* **Integrating `TruthBeam`**: `TruthBeam` can be included in an existing OpenTelemetry Collector distribution, allowing you to add enrichment capabilities to your current observability pipeline.
* **Standalone `Compass`**: The `Compass` service can be deployed as an independent API, enabling it to be called by any application or a different enrichment processor within an existing OpenTelemetry or custom logging pipeline.

## Component Analysis

### 1. ProofWatch

**Purpose**: A helper library that acts as a "log bridge" for security events. Its purpose is to take pre-normalized, OCSF-compliant evidence and emit it to an OpenTelemetry Collector as a standardized log stream.

**Key Responsibilities**:
* Converts OCSF-formatted data into a standardized OpenTelemetry log record.
* Emits this log record to the OpenTelemetry Collector using the OTLP (OpenTelemetry Protocol).

`proofwatch` attributes defined [here](./ATTRIBUTES.md)

_Example code snippet_
```go
watcher, err := proofwatch.NewProofWatch("myexample", meter)
if err != nil {
    return fmt.Errorf("error setting up watcher: %w", err)
}

logger.Debug("found evidence", "path", evidence.Href)
var e proofwatch.Evidence
evidenceData, err := os.ReadFile(evidence.Href)
if err != nil {
    return err
}
if err = json.Unmarshal(evidenceData, &e); err != nil {
    return err
}
if err = watcher.Log(ctx, e); err != nil {
    return err
}
```

### 2. Beacon Collector Distro

**Purpose**: A minimal OpenTelemetry Collector distribution that acts as the runtime environment for the `complybeacon` evidence pipeline, specifically by hosting the `truthbeam` processor.

**Key Responsibilities**:
* Receiving log records from sources like `proofwatch`
* Running the `truthbeam` log processor on each log record.
* Exporting the processed, enriched logs to a configured backend.

### 3. TruthBeam

**Purpose**: To enrich log records with compliance-related context by querying the `compass` service. This is the core logic that transforms a simple policy check into an actionable compliance event.

**Key Responsibilities**:
* Maintains a local, in-memory cache of previously enriched data to reduce API calls and improve performance.
* Queries the Compass API for enrichment data based on attributes in the log record.
* Skips enrichment on API failures, tagging the log record with an enrichment_status: skipped attribute to enable graceful degradation.
* Adds the returned enrichment data as new attributes to the log record.

### 4. Compass

**Purpose**: A centralized lookup service that provides compliance context. It's the source of truth for mapping policies to standards and risk attributes.

**Key Responsibilities**:
* Receiving an EnrichmentRequest from `truthbeam`.
* Performing a lookup based on the policy details.
* Returning an EnrichmentResponse with compliance attributes.
