# `truthbeam`

The `truthbeam` custom OpenTelemetry Processor is part of the OpenTelemetry Pipeline. The `truthbeam` processor will ingest check normalized logs for required attributes and formulate an enrichment request to query the `compass` API. The logs will be enriched with compliance context attributes from the `compass` API and `truthbeam` will add those attributes to the original log record.  

> **Note** The `truthbeam` processor **gracefully** handles API failures to ensure log records won't be discarded.

## Writing tests for `truthbeam`

To write effective tests for the `truthbeam` custom OpenTelemetry Processor follow these guidelines:

1. Preserve maintainability whenever possible

   * Instead of using hard-coded strings, leverage the [attributes.go](https://github.com/complytime/complybeacon/blob/472aafba724b709ab3c9087c401275ebeb171943/truthbeam/internal/client/attributes.go) to ensure naming conventions don't require manual updates.

2. Follow the go-language recommendations for updating existing unit-tests

   * [go-language testing doc](https://go.dev/doc/tutorial/add-a-test)

### Existing test files

`config_test.go`: Uses table-driven tests to validate configuration validation and default values for the truthbeam processor.

`factory_test.go`: Tests the processor factory lifecycle including creation, configuration validation, and proper component initialization.

`processor_test.go`: Validates the core processor functionality including log processing, enrichment request formation, and response handling.

`apply_test.go`: Tests the attribute application logic for enriching log records with compliance data from the compass API.
