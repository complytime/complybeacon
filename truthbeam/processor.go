package truthbeam

import (
	"context"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/processor"
	"go.uber.org/zap"

	"github.com/complytime/complybeacon/truthbeam/internal/applier"
	"github.com/complytime/complybeacon/truthbeam/internal/client"
)

type truthBeamProcessor struct {
	telemetry component.TelemetrySettings
	config    *Config

	logger *zap.Logger

	client  *client.CacheableClient
	applier *applier.Applier
}

func newTruthBeamProcessor(conf component.Config, set processor.Settings) (*truthBeamProcessor, error) {
	cfg, ok := conf.(*Config)
	if !ok {
		return nil, errors.New("invalid configuration provided")
	}

	return &truthBeamProcessor{
		config:    cfg,
		telemetry: set.TelemetrySettings,
		logger:    set.Logger,
		client:    nil,
		applier:   applier.NewApplier(set.Logger),
	}, nil
}

func (t *truthBeamProcessor) processLogs(ctx context.Context, ld plog.Logs) (plog.Logs, error) {
	rl := ld.ResourceLogs()
	for i := 0; i < rl.Len(); i++ {
		rs := rl.At(i)
		ilss := rs.ScopeLogs()
		for j := 0; j < ilss.Len(); j++ {
			ils := ilss.At(j)
			logs := ils.LogRecords()
			for k := 0; k < logs.Len(); k++ {
				logRecord := logs.At(k)

				policy, status, err := t.applier.Extract(logRecord)
				if err != nil {
					t.logger.Error("Failed to extract evidence from log record", zap.Error(err))
					continue
				}

				// Get cached data
				enrichment, err := t.client.Retrieve(ctx, policy)
				if err != nil {
					// We don't want to return an error here to ensure the evidence
					// is not dropped. It will just be unmapped.

					t.logger.Error("failed to get enrichment",
						zap.String("policy_id", policy.PolicyRuleId),
						zap.Error(err))
					continue
				}

				err = t.applier.Apply(logRecord, enrichment, status)
				if err != nil {
					t.logger.Error("failed to apply enrichment",
						zap.String("policy_id", policy.PolicyRuleId),
						zap.Error(err))
				}
			}
		}
	}
	return ld, nil
}

// start will add HTTP client and pre-fetch any policy data
func (t *truthBeamProcessor) start(ctx context.Context, host component.Host) error {
	httpClient, err := t.config.ClientConfig.ToClient(ctx, host, t.telemetry)
	if err != nil {
		return err
	}

	baseClient, err := client.NewClient(t.config.ClientConfig.Endpoint, client.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}

	t.client = client.NewCacheableClient(baseClient, t.logger, t.config.CacheTTL)

	return nil
}
