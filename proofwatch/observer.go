package proofwatch

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// EvidenceObserver handles observing and pushing evidence processing metrics.
type EvidenceObserver struct {
	meter          *metric.Meter
	droppedCounter metric.Int64Counter
	processedCount metric.Int64Counter
}

// NewEvidenceObserver creates a new EvidenceObserver and registers the callback.
func NewEvidenceObserver(meter metric.Meter) (*EvidenceObserver, error) {
	co := &EvidenceObserver{
		meter: &meter,
	}

	var err error
	// Create and register the new counter.
	co.droppedCounter, err = meter.Int64Counter(
		"evidence_dropped_count",
		metric.WithDescription("The total number of evidence items dropped due to processing failures."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create dropped counter: %w", err)
	}

	co.processedCount, err = meter.Int64Counter(
		"evidence_processed_count",
		metric.WithDescription("The total number of evidence items processed successfully."),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create processed counter: %w", err)
	}

	return co, nil
}

func (e *EvidenceObserver) Dropped(ctx context.Context, attrs ...attribute.KeyValue) {
	e.droppedCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

func (e *EvidenceObserver) Processed(ctx context.Context, attrs ...attribute.KeyValue) {
	e.processedCount.Add(ctx, 1, metric.WithAttributes(attrs...))
}
