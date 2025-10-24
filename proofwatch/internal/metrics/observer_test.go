package metrics

import (
	"context"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
)

type evidenceObserverTestFixture struct {
	observer *EvidenceObserver
	reader   *sdkmetric.ManualReader
	t        *testing.T
}

func setupEvidenceObserverTest(t *testing.T) *evidenceObserverTestFixture {
	t.Helper()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

	meter := mp.Meter("test-meter")
	observer, err := NewEvidenceObserver(meter)
	require.NoError(t, err)

	return &evidenceObserverTestFixture{
		observer: observer,
		reader:   reader,
		t:        t,
	}
}

func (f *evidenceObserverTestFixture) collectMetrics(ctx context.Context) metricdata.ResourceMetrics {
	f.t.Helper()

	var rm metricdata.ResourceMetrics
	err := f.reader.Collect(ctx, &rm)
	require.NoError(f.t, err)
	return rm
}

func (f *evidenceObserverTestFixture) assertMetricsRecorded(ctx context.Context) {
	f.t.Helper()

	rm := f.collectMetrics(ctx)
	assert.NotEmpty(f.t, rm.ScopeMetrics)
}

func TestNewEvidenceObserver(t *testing.T) {
	t.Run("constructs successfully", func(t *testing.T) {
		mp := sdkmetric.NewMeterProvider()
		t.Cleanup(func() { _ = mp.Shutdown(context.Background()) })

		meter := mp.Meter("test-meter")
		observer, err := NewEvidenceObserver(meter)

		require.NoError(t, err)
		require.NotNil(t, observer)
		assert.NotNil(t, observer.meter)
		assert.NotNil(t, observer.droppedCounter)
		assert.NotNil(t, observer.processedCount)
	})

	t.Run("constructs with manual reader", func(t *testing.T) {
		fixture := setupEvidenceObserverTest(t)
		require.NotNil(t, fixture.observer)
	})
}

func TestEvidenceObserverCounters_TableDriven(t *testing.T) {
	type kind string
	const (
		processed kind = "processed"
		dropped   kind = "dropped"
	)

	tests := []struct {
		name       string
		k          kind
		iterations int
		attrsFn    func(i int) []attribute.KeyValue
	}{
		{
			name:       "processed: single event",
			k:          processed,
			iterations: 1,
			attrsFn: func(i int) []attribute.KeyValue {
				return []attribute.KeyValue{attribute.String("test", "value")}
			},
		},
		{
			name:       "processed: multiple events with iteration attr",
			k:          processed,
			iterations: 5,
			attrsFn: func(i int) []attribute.KeyValue {
				return []attribute.KeyValue{attribute.String("iteration", strconv.Itoa(i))}
			},
		},
		{
			name:       "processed: multiple attributes",
			k:          processed,
			iterations: 1,
			attrsFn: func(i int) []attribute.KeyValue {
				return []attribute.KeyValue{
					attribute.String("policy.id", "test-policy"),
					attribute.String("policy.source", "test-source"),
					attribute.String("policy.evaluation.status", "pass"),
				}
			},
		},
		{
			name:       "processed: no attributes",
			k:          processed,
			iterations: 1,
			attrsFn:    func(i int) []attribute.KeyValue { return nil },
		},
		{
			name:       "dropped: single event",
			k:          dropped,
			iterations: 1,
			attrsFn: func(i int) []attribute.KeyValue {
				return []attribute.KeyValue{attribute.String("reason", "validation_failed")}
			},
		},
		{
			name:       "dropped: multiple events with reason attr",
			k:          dropped,
			iterations: 3,
			attrsFn: func(i int) []attribute.KeyValue {
				reasons := []string{"validation_failed", "processing_error", "timeout"}
				return []attribute.KeyValue{attribute.String("reason", reasons[i%len(reasons)])}
			},
		},
		{
			name:       "dropped: no attributes",
			k:          dropped,
			iterations: 1,
			attrsFn:    func(i int) []attribute.KeyValue { return nil },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := setupEvidenceObserverTest(t)
			ctx := context.Background()

			for i := 0; i < tt.iterations; i++ {
				attrs := tt.attrsFn(i)
				switch tt.k {
				case processed:
					if attrs == nil {
						fixture.observer.Processed(ctx)
					} else {
						fixture.observer.Processed(ctx, attrs...)
					}
				case dropped:
					if attrs == nil {
						fixture.observer.Dropped(ctx)
					} else {
						fixture.observer.Dropped(ctx, attrs...)
					}
				default:
					require.FailNow(t, "unknown kind")
				}
			}

			fixture.assertMetricsRecorded(ctx)
		})
	}
}

func TestEvidenceObserverBothMetrics(t *testing.T) {
	t.Run("records both processed and dropped", func(t *testing.T) {
		fixture := setupEvidenceObserverTest(t)
		ctx := context.Background()

		// processed
		fixture.observer.Processed(ctx, attribute.String("policy.id", "policy-1"))
		fixture.observer.Processed(ctx, attribute.String("policy.id", "policy-2"))
		fixture.observer.Processed(ctx, attribute.String("policy.id", "policy-3"))

		// dropped
		fixture.observer.Dropped(ctx, attribute.String("reason", "error"))
		fixture.observer.Dropped(ctx, attribute.String("reason", "timeout"))

		rm := fixture.collectMetrics(ctx)
		require.NotEmpty(t, rm.ScopeMetrics)

		found := map[string]bool{}
		for _, sm := range rm.ScopeMetrics {
			for _, m := range sm.Metrics {
				found[m.Name] = true
			}
		}

		assert.True(t, found["evidence_processed_count"], "expected processed metric to be present")
		assert.True(t, found["evidence_dropped_count"], "expected dropped metric to be present")
	})
}

func TestEvidenceObserverConcurrentRecording(t *testing.T) {
	fixture := setupEvidenceObserverTest(t)
	ctx := context.Background()

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			fixture.observer.Processed(ctx, attribute.String("goroutine", "1"))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 10; i++ {
			fixture.observer.Dropped(ctx, attribute.String("goroutine", "2"))
		}
	}()

	wg.Wait()
	fixture.assertMetricsRecorded(ctx)
}

func TestEvidenceObserverWithContext(t *testing.T) {
	tests := []struct {
		name string
		ctx  func() context.Context
	}{
		{
			name: "with cancelled context",
			ctx: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
		},
		{
			name: "with background context",
			ctx:  func() context.Context { return context.Background() },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := setupEvidenceObserverTest(t)
			ctx := tt.ctx()

			// Should not panic for either context
			fixture.observer.Processed(ctx, attribute.String("test", "value"))
			fixture.observer.Dropped(ctx, attribute.String("test", "value"))

			// Only assert metrics for non-cancelled context (cancelled may or may not record)
			if tt.name != "with cancelled context" {
				fixture.assertMetricsRecorded(ctx)
			}
		})
	}
}
