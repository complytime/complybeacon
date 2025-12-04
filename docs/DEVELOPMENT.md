# ComplyBeacon Development Guide

This guide provides comprehensive instructions for setting up, building, and testing the ComplyBeacon project.
It complements the [DESIGN.md](./DESIGN.md) document by focusing on the practical aspects of development.

<!-- TOC -->
* [ComplyBeacon Development Guide](#complybeacon-development-guide)
  * [Prerequisites](#prerequisites)
    * [Required Software](#required-software)
  * [Development Environment Setup](#development-environment-setup)
    * [1. Clone the Repository](#1-clone-the-repository)
    * [2. Install podman-compose (if needed)](#2-install-podman-compose-if-needed)
    * [3. Initialize Go Workspace](#3-initialize-go-workspace)
    * [4. Install Dependencies](#4-install-dependencies)
    * [5. Verify Installation](#5-verify-installation)
  * [Project Structure](#project-structure)
  * [Testing](#testing)
    * [Running Tests](#running-tests)
    * [Integration Testing](#integration-testing)
  * [Component Development](#component-development)
    * [1. ProofWatch Development](#1-proofwatch-development)
    * [2. Compass Development](#2-compass-development)
    * [3. TruthBeam Development](#3-truthbeam-development)
    * [4. Beacon Distro Development](#4-beacon-distro-development)
  * [Debugging and Troubleshooting](#debugging-and-troubleshooting)
    * [Debugging Tools](#debugging-tools)
  * [Code Generation](#code-generation)
    * [1. API Code Generation](#1-api-code-generation)
    * [2. OpenTelemetry Semantic Conventions](#2-opentelemetry-semantic-conventions)
    * [3. Manual Code Generation](#3-manual-code-generation)
  * [Deployment and Demo](#deployment-and-demo)
    * [Local Development Demo](#local-development-demo)
  * [Additional Resources](#additional-resources)
<!-- TOC -->

## Prerequisites

### Required Software

- **Go 1.24+**: The project uses Go 1.24.0 with toolchain 1.24.5
- **Podman**: For containerized development and deployment
- **podman-compose**: For orchestrating multi-container development environments
- **Make**: For build automation
- **Git**: For version control
- **openssl** Cryptography toolkit 

## Development Environment Setup

### 1. Clone the Repository

```bash
git clone https://github.com/complytime/complybeacon.git
cd complybeacon
```

### 2. Install podman-compose (if needed)

The project uses `podman-compose` for container orchestration. Install it if you don't have it:

```bash
# Install podman-compose
pip install podman-compose

# alternatively for Fedora:
dnf install podman-compose

# Verify installation
podman-compose --version
```

### 3. Initialize Go Workspace

The project uses Go workspaces to manage multiple modules:

```bash
make workspace
```

This creates a `go.work` file that includes all project modules:
- `./compass`
- `./proofwatch` 
- `./truthbeam`

### 4. Install Dependencies

Dependencies are managed per module. Install them for all modules:

```bash
# Install dependencies for all modules
for module in compass proofwatch truthbeam; do
    cd $module && go mod download && cd ..
done
```

### 5. Verify Installation

```bash
# Run tests to verify everything works
make test

# Build all binaries
make build
```

## Project Structure

```
complybeacon/
├── api.yaml                    # OpenAPI specification for Compass service
├── compose.yaml                # podman-compose configuration for demo environment
├── Makefile                    # Build automation
├── docs/                       # Documentation
│   ├── DESIGN.md              # Architecture and design documentation
│   ├── DEVELOPMENT.md         # This file
│   └── attributes/            # Attribute documentation
├── model/                      # OpenTelemetry semantic conventions
│   ├── attributes.yaml        # Attribute definitions
│   └── entities.yaml          # Entity definitions
├── compass/                    # Compass service module
│   ├── cmd/compass/           # Main application
│   ├── api/                   # Generated API code
│   ├── mapper/                # Enrichment mappers
│   └── service/               # Business logic
├── proofwatch/                 # ProofWatch instrumentation library
│   ├── attributes.go          # Attribute definitions
│   ├── evidence.go            # Evidence types
│   └── proofwatch.go          # Main library
├── truthbeam/                  # TruthBeam processor module
│   ├── internal/              # Internal packages
│   ├── config.go              # Configuration
│   └── processor.go           # Main processor logic
├── beacon-distro/              # OpenTelemetry Collector distribution
│   ├── config.yaml            # Collector configuration
│   └── Containerfile.collector # Container definition
├── hack/                       # Development utilities
│   ├── demo/                  # Demo configurations
│   ├── sampledata/            # Sample data for testing
│   └── self-signed-cert/      # self signed cert, testing/development purpose
└── bin/                        # Built binaries (created by make build)
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests for specific module
cd compass && go test -v ./...
cd proofwatch && go test -v ./...
cd truthbeam && go test -v ./...
```

### Integration Testing

The project includes integration tests using the demo environment:

```bash
# Start the demo environment
make deploy

# Test the pipeline
curl -X POST http://localhost:8088/eventsource/receiver \
  -H "Content-Type: application/json" \
  -d @hack/sampledata/evidence.json

# Check logs in Grafana at http://localhost:3000
# Check Compass API at http://localhost:8081/v1/enrich
```

## Component Development

### 1. ProofWatch Development

ProofWatch is an instrumentation library for emitting compliance evidence.

**Key Files:**
- `proofwatch/proofwatch.go` - Main library interface
- `proofwatch/evidence.go` - Evidence type definition
- `proofwatch/attributes.go` - OpenTelemetry attributes

**Development Workflow:**
```bash
cd proofwatch

# Run tests
go test -v ./...

# Check for linting issues
go vet ./...

# Format code
go fmt ./...
```

### 2. Compass Development

Compass is the enrichment service that provides compliance context.

**Key Files:**
- `compass/cmd/compass/main.go` - Service entry point
- `compass/service/service.go` - Business logic
- `compass/mapper/` - Enrichment mappers
- `api.yaml` - OpenAPI specification

**Development Workflow:**
```bash
cd compass

# Run the service locally
go run ./cmd/compass --config hack/demo/config.yaml --catalog hack/sampledata/osps.yaml --port 8081 --skip-tls

# Test the API
curl -X POST http://localhost:8081/v1/metadata \
  -H "Content-Type: application/json" \
  -d '{"policy": {"policyEngineName": "OPA", "policyRuleId": "deny-root-user"}}'
```

**Adding New Mappers:**
1. Create a new mapper in `compass/mapper/plugins/`
2. Implement the `Mapper` interface
3. Register the mapper in the factory
4. Add configuration options

### 3. TruthBeam Development

TruthBeam is an OpenTelemetry Collector processor for enriching logs.

**Key Files:**
- `truthbeam/processor.go` - Main processor logic
- `truthbeam/config.go` - Configuration structures
- `truthbeam/factory.go` - Processor factory

**Development Workflow:**
```bash
cd truthbeam

# Run tests
go test -v ./...

# Test with collector (requires beacon-distro)
cd ../beacon-distro
# Modify config to use local truthbeam
# Run collector with local processor
```

**Local development config**

If you want locally test the TruthBeam, remember to change the [manifest.yaml](../beacon-distro/manifest.yaml)

Add replace directive at the end of [manifest.yaml](../beacon-distro/manifest.yaml), to make sure collector use your `truthbeam` code. Default collector will use `- gomod: github.com/complytime/complybeacon/truthbeam main`

For example:
```yaml
replaces:
  - github.com/complytime/complybeacon/truthbeam => github.com/AlexXuan233/complybeacon/truthbeam 52e4a76ea0f72a7049e73e7a5d67d988116a3892
```
or
```yaml
replaces:
  - github.com/complytime/complybeacon/truthbeam => github.com/AlexXuan233/complybeacon/truthbeam main
```

### 4. Beacon Distro Development

The Beacon distribution is a custom OpenTelemetry Collector.

**Key Files:**
- `beacon-distro/config.yaml` - Collector configuration
- `beacon-distro/Containerfile.collector` - Container definition

**Development Workflow:**
```bash
cd beacon-distro

# Build the collector image
podman build -f Containerfile.collector -t complybeacon-beacon-distro:latest .

# Test with local configuration
podman run --rm -p 4317:4317 -p 8088:8088 \
  -v $(pwd)/config.yaml:/etc/otel-collector.yaml:Z \
  complybeacon-beacon-distro:latest
```

## Debugging and Troubleshooting

### Debugging Tools

```bash
# View container logs
podman-compose logs -f compass
podman-compose logs -f collector
```

## Code Generation

The project uses several code generation tools:

### 1. API Code Generation

Generate Go code from OpenAPI specification:

```bash
make api-codegen
```

This generates:
- `compass/api/types.gen.go` - Request/response types
- `compass/api/server.gen.go` - Server interfaces

### 2. OpenTelemetry Semantic Conventions

Generate documentation and Go code from semantic convention models:

```bash
# Generate documentation
make weaver-docsgen

# Generate Go code
make weaver-codegen

# Validate models
make weaver-check
```

### 3. Manual Code Generation

If you modify the OpenAPI spec or semantic conventions:

```bash
# Update API spec
vim api.yaml

# Regenerate API code
make api-codegen

# Update semantic conventions
vim model/attributes.yaml
vim model/entities.yaml

# Regenerate semantic convention code
make weaver-codegen
```

## Deployment and Demo

### Local Development Demo

The demo environment uses `podman-compose` to orchestrate multiple containers. Ensure you have `podman-compose` installed before proceeding.

1. **Generate self-signed certificate**

Since compass and truthbeam enabled TLS by default, first we need to generate self-signed certificate for testing/development

```shell
make generate-self-signed-cert
```

2. **Start the full stack:**
```bash
make deploy
```

3. **Test the pipeline:**
```bash
curl -X POST http://localhost:8088/eventsource/receiver \
  -H "Content-Type: application/json" \
  -d @hack/sampledata/evidence.json
```

4. **View results:**
- Grafana: http://localhost:3000

5. **Stop the stack:**
```bash
make undeploy
```

---

## Additional Resources

- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [Go Documentation](https://golang.org/doc/)
- [Podman Documentation](https://docs.podman.io/)
- [Project Design Document](./DESIGN.md)
- [Attribute Documentation](./attributes/)

For questions or support, please open an issue in the GitHub repository.