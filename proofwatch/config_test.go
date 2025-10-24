package proofwatch

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// providerCase exercises the three provider options against the same contract.
type providerCase struct {
	name                  string
	newProvider           func() any
	option                func(any) OptionFunc
	getProviderFromConfig func(*config) any
}

func (pc providerCase) run(t *testing.T) {
	t.Helper()

	t.Run(pc.name, func(t *testing.T) {
		t.Run("sets when non-nil", func(t *testing.T) {
			custom := pc.newProvider()
			cfg := &config{}

			pc.option(custom)(cfg)

			require.NotNil(t, pc.getProviderFromConfig(cfg))
			assert.Equal(t, custom, pc.getProviderFromConfig(cfg))
		})

		t.Run("no-op on nil", func(t *testing.T) {
			original := pc.newProvider()
			cfg := &config{}

			pc.option(original)(cfg) // set initial
			pc.option(nil)(cfg)      // should not override

			assert.Equal(t, original, pc.getProviderFromConfig(cfg))
		})
	})
}

func TestProviderOptions_ConfigOnly(t *testing.T) {
	cases := []providerCase{
		{
			name:        "WithMeterProvider",
			newProvider: func() any { return sdkmetric.NewMeterProvider() },
			option: func(p any) OptionFunc {
				// Make the assertion nil-safe by passing a typed nil when p is nil.
				var mp metric.MeterProvider
				if p != nil {
					mp = p.(metric.MeterProvider)
				}
				return WithMeterProvider(mp)
			},
			getProviderFromConfig: func(c *config) any { return c.MeterProvider },
		},
		{
			name:        "WithLoggerProvider",
			newProvider: func() any { return noop.NewLoggerProvider() },
			option: func(p any) OptionFunc {
				var lp log.LoggerProvider
				if p != nil {
					lp = p.(log.LoggerProvider)
				}
				return WithLoggerProvider(lp)
			},
			getProviderFromConfig: func(c *config) any { return c.LoggerProvider },
		},
		{
			name:        "WithTracerProvider",
			newProvider: func() any { return sdktrace.NewTracerProvider() },
			option: func(p any) OptionFunc {
				var tp trace.TracerProvider
				if p != nil {
					tp = p.(trace.TracerProvider)
				}
				return WithTracerProvider(tp)
			},
			getProviderFromConfig: func(c *config) any { return c.TracerProvider },
		},
	}

	for _, c := range cases {
		c.run(t)
	}
}

func TestOptionOrdering_ConfigOnly(t *testing.T) {
	first := sdkmetric.NewMeterProvider()
	second := sdkmetric.NewMeterProvider()

	cfg := &config{}
	WithMeterProvider(first)(cfg)
	WithMeterProvider(second)(cfg)

	assert.Equal(t, second, cfg.MeterProvider)
}

func TestOptionFunc_CustomSetter(t *testing.T) {
	cfg := &config{}
	custom := sdkmetric.NewMeterProvider()

	opt := OptionFunc(func(c *config) {
		c.MeterProvider = custom
	})

	opt(cfg)
	assert.Equal(t, custom, cfg.MeterProvider)
}

func TestOptionFunc_ModifyAllFields(t *testing.T) {
	cfg := &config{}
	meter := sdkmetric.NewMeterProvider()
	logger := noop.NewLoggerProvider()
	tracer := sdktrace.NewTracerProvider()

	opt := OptionFunc(func(c *config) {
		c.MeterProvider = meter
		c.LoggerProvider = logger
		c.TracerProvider = tracer
	})

	opt(cfg)
	assert.Equal(t, meter, cfg.MeterProvider)
	assert.Equal(t, logger, cfg.LoggerProvider)
	assert.Equal(t, tracer, cfg.TracerProvider)
}
