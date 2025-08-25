package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/goccy/go-yaml"

	"github.com/complytime/complybeacon/proofwatch/agent"
	"github.com/complytime/complybeacon/proofwatch/source"
)

const shutDownTimeout = 10 * time.Second

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Println(err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	var (
		otelEndpoint, configPath string
		skipTLS, skipTLSVerify   bool
	)

	flag.StringVar(&otelEndpoint, "otel-endpoint", "localhost:4317", "Endpoint for the OpenTelemetry Collector")
	flag.StringVar(&configPath, "config", "./watch.yaml", "Path to proofwatch configuration file")
	flag.BoolVar(&skipTLS, "skip-tls", false, "Do not use TLS to communicate with the OpenTelemetry Collector")
	flag.BoolVar(&skipTLSVerify, "skip-tls-verify", false, "Bypass server certificate validation")
	flag.Parse()

	agt := agent.New(otelEndpoint, skipTLS, skipTLSVerify)

	configPath = filepath.Clean(configPath)
	configBytes, err := os.ReadFile(filepath.Clean(configPath))
	if err != nil {
		return fmt.Errorf("error reading config: %w", err)
	}

	config := source.Config{}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		return fmt.Errorf("error unmarshaling config: %w", err)
	}

	if err := agt.Run(ctx, &config); err != nil {
		return err
	}

	// Wait for the context to be canceled (e.g., via a signal)
	<-ctx.Done()

	log.Println("Shutdown received")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), shutDownTimeout)
	defer cancelShutdown()
	return agt.Stop(shutdownCtx)
}
