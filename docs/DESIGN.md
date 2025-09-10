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

The ComplyBeacon architecture is designed to handle two primary data ingestion scenarios, each feeding into a unified enrichment pipeline.

#### The Collector Pipeline
For log sources, ProofWatch can be used to send OCSF-compliant logs directly to the collector. ProofWatch validates the raw evidence against a standardized OCSF schema, and 
converts it into a structured LogRecord. This ensures the required attributes are present while retaining the original data within the log body.

Once a LogRecord is ingested into the `collector` via a configured receiver, it proceeds through the following pipeline:

1. The LogRecord is received and forwarded to the `truthbeam` processor.
2. The `truthbeam` processor extracts key attributes (e.g., `policy.id`) from the log record.
3. It then sends an enrichment request containing this data to the `compass` API.
4. The `compass` service performs a lookup based on the provided attributes and returns a response with compliance-related context (e.g., impacted baselines, requirements, and a compliance result).
5. `truthbeam` adds these new attributes to the original LogRecord.

The now-enriched log record is exported from the `collector` to a final destination (e.g., a SIEM, logging backend, or data lake) for analysis and correlation.
```
┌─────────────────────────────────────────────────────────────────────────────────────────────────────────────┐
│                                                                                                             │
│                                                    ┌─────────────────────────┐                              │
│                                                    │                         │                              │
│                                                    │ Beacon Collector Distro │                              │
│   ┌────────────────────┐   ┌───────────────────┐   │                         │                              │
│   │                    │   │                   │   ├─────────────────────────┤                              │
│   │                    ├───┤    ProofWatch     ├───┼────┐                    │                              │
│   │                    │   │                   │   │    │                    │                              │
│   │   Normalization    │   └───────────────────┘   │   ┌┴─────────────────┐  │                              │
│   │   At the Edge      │                           │   │       OTLP       │  │                              │
│   │                    │                           │   │       Filelog    │  │                              │
│   │                    │                           │   │      Reciever    │  │                              │
│   │                    │  ┌────────────────────────┼───┤                  │  │                              │
│   └────────────────────┘  │                        │   └────────┬─────────┘  │                              │
│                           │                        │            │            │               ┌─────────────┐│
│                           │                        │   ┌────────┴─────────┐  │               │             ││
│                           │                        │   │                  │  │               │             ││
│                           │                        │   │    Tranform      │──┼──────────────►│ Compass API ││
│   ┌───────────────────────┴───┐                    │   │    TruthBeam     │  │               │             ││
│   │                           │                    │   │    Processor     │  │               │             ││
│   │                           │                    │   └────────┬─────────┘  │               └─────────────┘│
│   │      In-collector         │                    │            │            │                              │
│   │     Normlalization        │                    │            │            │                              │
│   │                           │                    │   ┌────────┴─────────┐  │                              │
│   │                           │                    │   │    Exporter      │  │                              │
│   │                           │                    │   │   (e.g. Loki     │  │                              │
│   │                           │                    │   │   Splunk)        │  │                              │
│   │                           │                    │   └──────────────────┘  │                              │
│   │                           │                    └─────────────────────────┘                              │
│   └───────────────────────────┘                                                                             │
│                                                                                                             │
└─────────────────────────────────────────────────────────────────────────────────────────────────────────────┘          
```

### Deployment Patterns

ComplyBeacon is designed to be a flexible toolkit. Its components can be used in different combinations to fit a variety of operational needs.

* Full Pipeline: This is the most common use case for all sources. Policy Engine (w/ ProofWatch) -> Collector -> Compass -> Final Destination

* Include TruthBeam in an existing Collector Distro: If you already have a Collector distribution, simply add truthbeam to your distribution manifest.

* Using Compass as a Standalone Service: The compass service can be deployed as an independent API, allowing it to be called by any application or a different enrichment processor within an existing OpenTelemetry or custom logging pipeline.

## Component Analysis

### 1. ProofWatch

**Purpose**: A helper library that acts as a "log bridge" for security events. Its purpose is to take policy decision data from an application and send it to an OpenTelemetry Collector as standardized logs.

**Key Responsibilities**:

* It validates this data against the Open Cybersecurity Schema Framework (OCSF), ensuring it is properly structured as a security event.

* It converts the OCSF-formatted data into a standardized OpenTelemetry Event.

* It sends this event to the OpenTelemetry Collector using the OTLP (OpenTelemetry Protocol).

`proofwatch` attributes and body are defined [here](./ATTRIBUTES.md)

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

* Receiving log records from `proofwatch`.

* Running the `truthbeam` log processor on each log record.

* Exporting the processed, enriched logs to a configured backend.

### 3. TruthBeam

**Purpose**: To enrich log records with compliance-related context by querying the `compass` service. This is the core logic that transforms a simple policy check into an actionable compliance event.

**Key Responsibilities**:
> Note: Cache and async patterns are currently unimplemented
* Local Cache: Maintains a local, in-memory cache of previously enriched data for fast lookups.

* Asynchronous API Calls: Puts requests for cache misses into a separate queue and uses a background worker to call the compass API.

* Graceful Degradation: Skips enrichment on API failures, tagging the log record with an enrichment_status: skipped attribute.

### 4. Compass

**Purpose**: A centralized lookup service that provides compliance context. It's the source of truth for mapping policies to standards and risk attributes.

**Key Responsibilities**:

* Receiving an EnrichmentRequest from `truthbeam`.

* Performing a lookup based on policy.id and the policy details.

* Returning an EnrichmentResponse with a compliance result, relevant baselines, and requirements.
