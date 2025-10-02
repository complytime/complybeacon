# ComplyBeacon

**ComplyBeacon** is an open-source observability toolkit designed to collect, normalize, and enrich compliance evidence, extending the OpenTelemetry (OTEL) standard.

By bridging the gap between raw policy scanner output and modern logging pipelines, it provides a unified, enriched, and auditable data stream for security and compliance analysis.

---

⚠️ **WARNING:** All components are under initial development and are **not** ready for production use.

---

## The ComplyBeacon Architecture

ComplyBeacon is a policy-driven observability toolkit composed of four main components that work together to process and enrich compliance data.

### 1. ProofWatch

An instrumentation library that accepts and emits pre-normalized compliance evidence as an OpenTelemetry log stream, while also instrumenting metrics for real-time observability.

### 2. Beacon

A custom OpenTelemetry Collector distribution that acts as the pipeline's host, receiving log records from ProofWatch and preparing them for the next stage of enrichment.

### 3. TruthBeam

A custom OpenTelemetry Collector processor that enriches log records with compliance and risk data by integrating with the Compass service.

### 4. Compass

A central enrichment service that provides risk, threat, and compliance framework attributes based on policy lookup data.

#### Supported Compass Mappers

| Mapper  | Description                                        |
|---------|----------------------------------------------------|
| `basic` | Maps to the `gemara` model based on log attributes |

## Quick Start

Follow these steps to deploy the infrastructure and test the pipeline.

1.  **Deploy the Stack:**
    This command builds and starts the full infrastructure, including Grafana, Loki, the custom collector (`Beacon`), and the `Compass` service.
    ```bash
    podman-compose up --build
    ```

2.  **Test the Pipeline:**
    Send sample compliance data to the webhook receiver to test the pipeline's functionality.
    ```bash
    curl -X POST http://localhost:8088/eventsource/receiver -H "Content-Type: application/json" -d @hack/sampledata/evidence.json
    ```

## Project Design

For additional details on the planned design and roadmap, see [`DESIGN.md`](./docs/DESIGN.md).

## Updating the Semantic Conventions

Update semantic convention under `model/`

Validate with `make weaver-check`

Update docs and code:
`make weaver-docsgen`
`make weaver-codegen`

---