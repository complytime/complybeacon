package proofwatch

import (
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

type config struct {
	LogProvider    log.LoggerProvider
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

type OptionFunc func(*config)

// WithTracerProvider specifies a tracer provider to use for creating a tracer.
// If none is specified, the global TracerProvider is used.
func WithTracerProvider(provider trace.TracerProvider) OptionFunc {
	return OptionFunc(func(cfg *config) {
		if provider != nil {
			cfg.TracerProvider = provider
		}
	})
}

// WithMeterProvider specifies a meter provider to use for creating a meter.
// If none is specified, the global MeterProvider is used.
func WithMeterProvider(provider metric.MeterProvider) OptionFunc {
	return OptionFunc(func(cfg *config) {
		if provider != nil {
			cfg.MeterProvider = provider
		}
	})
}

// WithLoggerProvider specifies a logger provider to use for creating a logger.
// If none is specified, the global LoggerProvider is used.
func WithLoggerProvider(provider log.LoggerProvider) OptionFunc {
	return OptionFunc(func(cfg *config) {
		if provider != nil {
			cfg.LogProvider = provider
		}
	})
}
