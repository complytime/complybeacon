# `proofwatch` instrumentation kit

## Overview

Proofwatch captures, normalizes, and emits formatted compliance-data logs using the OpenTelemetry format (OTLP).

## Compatibility 

Proofwatch is a distinctive microservice that can be used any standard OpenTelemetry Collector to receive and process compliance-data logs.

> **Note:** Proofwatch is commonly used with the [`Beacon`](https://github.com/complytime/complybeacon/blob/ba4106b36dd25b08c15d134650843fb4bffc4e0e/beacon-distro) Collector which leverages [`truthbeam`](https://github.com/complytime/complybeacon/blob/953910cf8a7b1c8c44b8e21630bbb112461d30f0/truthbeam) and [`compass`](https://github.com/complytime/complybeacon/blob/ba4106b36dd25b08c15d134650843fb4bffc4e0e/compass) for processing, enrichment, and export of logs.

## Writing unit-tests for `proofwatch`

> **Disclaimer:** As development progresses, additional tests will be added to increase the coverage.

To write effective unit-tests for the `proofwatch` instrumentation kit follow these guidelines:

1. Unit-tests are located in the `proofwatch` directory. 
2. Prioritize maintainability of the unit-tests. When appropriate, use the [attributes.go](https://github.com/complytime/complybeacon/blob/ba4106b36dd25b08c15d134650843fb4bffc4e0e/proofwatch/attributes.go) _instead_ of hard-coded strings in the unit-tests.
3. Tests should follow the [go testing style](https://go.dev/doc/tutorial/add-a-test). Review existing tests for readability.

> Example: [gemara.go](https://github.com/complytime/complybeacon/blob/ba4106b36dd25b08c15d134650843fb4bffc4e0e/proofwatch/gemara.go) is properly tested by the [gemara_test.go](https://github.com/complytime/complybeacon/blob/ba4106b36dd25b08c15d134650843fb4bffc4e0e/proofwatch/gemara_test.go).
