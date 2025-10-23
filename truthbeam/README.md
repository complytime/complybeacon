# `truthbeam` processor

## Overview

The `truthbeam` custom OpenTelemetry Processor is part of the OpenTelemetry Pipeline. The `truthbeam` processor will ingest check normalized logs for required attributes and formulate an enrichment request to query the `compass` API. The logs will be enriched with compliance context attributes from the `compass` API and `truthbeam` will add those attributes to the original log record.  

## Compatibility

The `truthbeam` processor can be integrated into any OpenTelemetry Collector distribution.

> **Note:** The `truthbeam` processor **gracefully** handles API failures to ensure log records won't be discarded.

## Writing tests for `truthbeam`

> **Disclaimer:** As development progresses, additional tests will be added to increase the coverage.

To write effective unit-tests for the `truthbeam` custom OpenTelemetry Processor follow these guidelines:

1. Unit-tests are located in the `truthbeam` directory. 
2. Prioritize maintainability of the unit-tests. When appropriate, use the [attributes.go](https://github.com/complytime/complybeacon/blob/472aafba724b709ab3c9087c401275ebeb171943/truthbeam/internal/client/attributes.go) _instead_ of hard-coded strings in the unit-tests. 
3. Tests should follow the [go testing style](https://go.dev/doc/tutorial/add-a-test). Review existing tests for readability.

### Existing test files

`config_test.go`: Uses table-driven tests to validate configuration validation and default values for the truthbeam processor.

`factory_test.go`: Tests the processor factory lifecycle including creation, configuration validation, and proper component initialization.

`processor_test.go`: Validates the core processor functionality including log processing, enrichment request formation, and response handling.

`apply_test.go`: Tests the attribute application logic for enriching log records with compliance data from the compass API.
