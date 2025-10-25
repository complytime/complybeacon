# `proofwatch` instrumentation kit

## Overview

Proofwatch captures, normalizes, and emits formatted compliance-data logs using the OpenTelemetry format (OTLP).

## Usage 

Proofwatch is an OpenTelemetry instrumentation library that enable applications to log compliance evidence and policy evaluation events using OpenTelemetry's structured logging format.

> **Note:** Proofwatch is commonly used with the [`Beacon`](https://github.com/complytime/complybeacon/blob/ba4106b36dd25b08c15d134650843fb4bffc4e0e/beacon-distro) Collector which leverages [`truthbeam`](https://github.com/complytime/complybeacon/blob/953910cf8a7b1c8c44b8e21630bbb112461d30f0/truthbeam) and [`compass`](https://github.com/complytime/complybeacon/blob/ba4106b36dd25b08c15d134650843fb4bffc4e0e/compass) for processing, enrichment, and export of logs.

### Example Code Snippet

```go
import (
    "context"
    "log"
    
    "go.opentelemetry.io/otel/log"
    "github.com/complytime/complybeacon/proofwatch"
)

// Create a new ProofWatch instance
pw, err := proofwatch.NewProofWatch()
if err != nil {
    log.Fatal(err)
}

// Create evidence (example with GemaraEvidence)
evidence := proofwatch.GemaraEvidence{
    // ... populate evidence fields
}

// Log evidence with default severity
err = pw.Log(ctx, evidence)
if err != nil {
    return fmt.Errorf("error logging evidence: %w", err)
}

// Or log with specific severity
err = pw.LogWithSeverity(ctx, evidence, olog.SeverityWarn)
```

> Review guidelines for writing tests in the [DEVELOPMENT.md](https://github.com/complytime/complybeacon/blob/main/docs/DEVELOPMENT.md).