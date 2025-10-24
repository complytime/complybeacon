package proofwatch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	olog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

// proofWatchTestFixture provides test infrastructure for ProofWatch behavioral tests
type proofWatchTestFixture struct {
	pw            *ProofWatch
	exporter      *tracetest.InMemoryExporter
	metricsReader *sdkmetric.ManualReader
	t             *testing.T
}

// setupProofWatchTest creates a test fixture with configured providers and exporters
func setupProofWatchTest(t *testing.T) *proofWatchTestFixture {
	exporter := tracetest.NewInMemoryExporter()
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSyncer(exporter),
	)
	reader := sdkmetric.NewManualReader()
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)

	pw, err := NewProofWatch(
		WithTracerProvider(tracerProvider),
		WithMeterProvider(meterProvider),
		WithLoggerProvider(noop.NewLoggerProvider()),
	)
	require.NoError(t, err)

	return &proofWatchTestFixture{
		pw:            pw,
		exporter:      exporter,
		metricsReader: reader,
		t:             t,
	}
}

// assertSpanCreated verifies that exactly one span with the expected name was created
func (f *proofWatchTestFixture) assertSpanCreated(spanName string) {
	spans := f.exporter.GetSpans()
	assert.Len(f.t, spans, 1)
	assert.Equal(f.t, spanName, spans[0].Name)
}

// assertSpanEvent verifies that the span has exactly one event with the expected name
func (f *proofWatchTestFixture) assertSpanEvent(eventName string) {
	spans := f.exporter.GetSpans()
	require.Len(f.t, spans, 1)
	assert.Len(f.t, spans[0].Events, 1)
	assert.Equal(f.t, eventName, spans[0].Events[0].Name)
}

// assertSpanCreatedWithEvent verifies that a span and event were both created with expected names
func (f *proofWatchTestFixture) assertSpanCreatedWithEvent(spanName, eventName string) {
	f.assertSpanCreated(spanName)
	f.assertSpanEvent(eventName)
}

// collectMetrics collects and returns metrics from the test fixture
func (f *proofWatchTestFixture) collectMetrics(ctx context.Context) metricdata.ResourceMetrics {
	var rm metricdata.ResourceMetrics
	err := f.metricsReader.Collect(ctx, &rm)
	require.NoError(f.t, err)
	return rm
}

func TestNewProofWatch(t *testing.T) {
	t.Run("default options", func(t *testing.T) {
		pw, err := NewProofWatch()
		require.NoError(t, err)
		assert.NotNil(t, pw)
		assert.NotNil(t, pw.logger)
		assert.NotNil(t, pw.tracer)
		assert.NotNil(t, pw.observer)
		assert.Equal(t, olog.SeverityInfo, pw.levelSeverity)
	})

	t.Run("with custom providers", func(t *testing.T) {
		meterProvider := sdkmetric.NewMeterProvider()
		loggerProvider := noop.NewLoggerProvider()
		tracerProvider := sdktrace.NewTracerProvider()

		pw, err := NewProofWatch(
			WithMeterProvider(meterProvider),
			WithLoggerProvider(loggerProvider),
			WithTracerProvider(tracerProvider),
		)
		require.NoError(t, err)
		assert.NotNil(t, pw)
	})

	t.Run("with nil providers", func(t *testing.T) {
		// Should not panic with nil providers - they fall back to global providers
		pw, err := NewProofWatch(
			WithMeterProvider(nil),
			WithLoggerProvider(nil),
			WithTracerProvider(nil),
		)
		require.NoError(t, err)
		assert.NotNil(t, pw)
	})
}

func TestProofWatchLog(t *testing.T) {
	t.Run("log with default severity", func(t *testing.T) {
		fixture := setupProofWatchTest(t)
		evidence := createTestEvidence()

		ctx := context.Background()
		err := fixture.pw.Log(ctx, evidence)
		require.NoError(t, err)

		fixture.assertSpanCreatedWithEvent("evidence.log_evidence", "evidence.logged")
		fixture.collectMetrics(ctx)
	})

	t.Run("log with invalid evidence", func(t *testing.T) {
		pw, err := NewProofWatch()
		require.NoError(t, err)

		evidence := &invalidEvidence{}
		err = pw.Log(context.Background(), evidence)
		assert.Error(t, err)
	})
}

func TestProofWatchLogWithSeverity(t *testing.T) {
	tests := []struct {
		name     string
		severity olog.Severity
	}{
		{
			name:     "debug severity",
			severity: olog.SeverityDebug,
		},
		{
			name:     "info severity",
			severity: olog.SeverityInfo,
		},
		{
			name:     "warn severity",
			severity: olog.SeverityWarn,
		},
		{
			name:     "error severity",
			severity: olog.SeverityError,
		},
		{
			name:     "fatal severity",
			severity: olog.SeverityFatal,
		},
		{
			name:     "unspecified",
			severity: olog.Severity(0),
		},
		{
			name:     "negative",
			severity: olog.Severity(-1),
		},
		{
			name:     "out of range",
			severity: olog.Severity(999),
		},
		{
			name:     "max valid",
			severity: olog.SeverityFatal4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := setupProofWatchTest(t)
			evidence := createTestEvidence()

			err := fixture.pw.LogWithSeverity(context.Background(), evidence, tt.severity)
			require.NoError(t, err)

			fixture.assertSpanCreatedWithEvent("evidence.log_evidence", "evidence.logged")
		})
	}
}

func TestVersion(t *testing.T) {
	version := Version()
	assert.NotEmpty(t, version)
	assert.Equal(t, "0.1.0", version)
}

func TestToLogKeyValues(t *testing.T) {
	attrs := []attribute.KeyValue{
		attribute.String("key1", "value1"),
		attribute.Int("key2", 42),
		attribute.Bool("key3", true),
	}

	logAttrs := ToLogKeyValues(attrs)

	assert.Equal(t, len(attrs), len(logAttrs))
	for i, logAttr := range logAttrs {
		assert.Equal(t, string(attrs[i].Key), logAttr.Key)
	}
}

// createTestEvidence is defined in ocsf_test.go and shared across test files
// createTestGemaraEvidence is defined in gemara_test.go
// invalidEvidence is a test implementation that fails JSON marshaling
type invalidEvidence struct{}

func (e *invalidEvidence) ToJSON() ([]byte, error) {
	return nil, assert.AnError
}

func (e *invalidEvidence) Attributes() []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("test", "value"),
	}
}

func (e *invalidEvidence) Timestamp() time.Time {
	return time.Now()
}
