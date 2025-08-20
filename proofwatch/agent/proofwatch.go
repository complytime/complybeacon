package agent

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/complytime/complybeacon/proofwatch/source"
)

// Global configuration w/ OpenTelemetry SDK

var otelShutdown func(ctx context.Context) error

// ProofWatch handles processing raw evidence, generating claims, and exporting data.
type ProofWatch struct {
	// Options
	otelEndpoint string
	sources      []source.Source
}

func New(otelEndpoint string) *ProofWatch {
	return &ProofWatch{
		otelEndpoint: otelEndpoint,
	}
}

// Run begins scraping for raw evidence and processing it.
func (s *ProofWatch) Run(ctx context.Context, config *source.Config) error {
	log.Printf("Configuring log exporting to %s", s.otelEndpoint)
	var err error
	conn, err := grpc.NewClient(s.otelEndpoint,
		// FIXME(jpower432): Configure secure credential options
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("failed to create gRPC connection to collector: %v", err)
	}
	otelShutdown, err = otelSDKSetup(ctx, conn)
	if err != nil {
		log.Fatalf("error with instrumentation: %v", err)
	}

	observer := metricsConfigure()
	config.SetupObserver(observer)

	s.sources, err = source.Deploy(ctx, *config)
	return err
}

// Stop signals the agent to gracefully shut down.
func (s *ProofWatch) Stop(ctx context.Context) error {
	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(2)

	eg.Go(func() error {
		return source.Shutdown(ctx, s.sources)
	})

	if otelShutdown != nil {
		eg.Go(func() error {
			otelCtx, otelCancel := context.WithTimeout(egCtx, 5*time.Second)
			defer otelCancel()
			if err := otelShutdown(otelCtx); err != nil {
				return fmt.Errorf("error during opentelemetry shutdown: %w", err)
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Println("Timed out during graceful shutdown. Some cleanup operations might not have completed.")
			return nil
		}
		return err
	}

	log.Println("Graceful shutdown complete...")
	return nil
}
