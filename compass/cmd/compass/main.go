package main

import (
	"flag"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"

	"github.com/complytime/complybeacon/compass/cmd/compass/server"
	"github.com/complytime/complybeacon/compass/internal/logging"
	compass "github.com/complytime/complybeacon/compass/service"
)

func main() {

	var (
		port, catalogPath, configPath string
		logLevel                      string
		skipTLS                       bool
	)

	flag.StringVar(&port, "port", "8080", "Port for HTTP server")
	flag.BoolVar(&skipTLS, "skip-tls", false, "Run without TLS")
	flag.StringVar(&logLevel, "log-level", "info", "Log level: debug|info|warn|error")

	// TODO: This needs to become Layer 3 policy and complete resolution on startup
	flag.StringVar(&catalogPath, "catalog", "./hack/sampledata/osps.yaml", "Path to Layer 2 catalog")
	flag.StringVar(&configPath, "config", "./docs/config.yaml", "Path to compass config file")
	flag.Parse()

	_, err := logging.Init(logLevel)
	if err != nil {
		slog.Error("failed to initialize logging", "err", err)
		os.Exit(1)
	}

	slog.Info("starting compass service",
		slog.String("port", port),
		slog.String("catalog", catalogPath),
		slog.String("config", configPath),
		slog.Bool("skip_tls", skipTLS),
	)

	catalogPath = filepath.Clean(catalogPath)
	scope, err := server.NewScopeFromCatalogPath(catalogPath)
	if err != nil {
		slog.Error("failed to load catalog", "path", catalogPath, "err", err)
		os.Exit(1)
	}

	var cfg server.Config
	configPath = filepath.Clean(configPath)
	content, err := os.ReadFile(configPath)
	if err != nil {
		slog.Error("failed to read config file", "path", configPath, "err", err)
		os.Exit(1)
	}

	err = yaml.Unmarshal(content, &cfg)
	if err != nil {
		slog.Error("failed to parse config file", "path", configPath, "err", err)
		os.Exit(1)
	}

	transformers, err := server.NewMapperSet(&cfg)
	if err != nil {
		slog.Error("failed to initialize plugin mappers", "err", err)
		os.Exit(1)
	}

	service := compass.NewService(transformers, scope)

	s := server.NewGinServer(service, port)

	if skipTLS {
		slog.Warn("insecure connections permitted; TLS is highly recommended for production")
		if err := s.ListenAndServe(); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	} else {
		cert, key := server.SetupTLS(s, cfg)
		if err := s.ListenAndServeTLS(cert, key); err != nil {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}
}
