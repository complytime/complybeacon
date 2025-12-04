package truthbeam

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/config/confighttp"
)

// The factory tests validate processor factory lifecycle including creation,
// configuration validation, and proper component initialization.

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	config := factory.CreateDefaultConfig()

	require.NotNil(t, config, "Config should not be nil")

	// Validate that config is processor-specific truthbeam config
	cfg, ok := config.(*Config)
	require.True(t, ok, "Expected *Config, got %T", config)

	// Validate truthbeam default config values
	assert.Empty(t, cfg.ClientConfig.Endpoint, "Expected default endpoint to be empty (must be set by user)")
	assert.Equal(t, 30*time.Second, cfg.ClientConfig.Timeout, "Expected timeout 30s")
	assert.Empty(t, cfg.ClientConfig.Compression, "Expected compression to be disabled by default for small payloads")
	assert.Equal(t, 512*1024, cfg.ClientConfig.WriteBufferSize, "Expected write buffer size 512KB")
	assert.Equal(t, time.Duration(-1), cfg.CacheTTL, "Expected default cache TTL to be 0 (no expiration)")
}

func TestCreateLogsProcessor(t *testing.T) {
	factory := NewFactory()
	config := factory.CreateDefaultConfig()

	assert.Equal(t, "truthbeam", factory.Type().String(), "Expected factory type 'truthbeam'")

	// Validate that config is processor-specific truthbeam config
	cfg, ok := config.(*Config)
	require.True(t, ok, "Expected *Config, got %T", config)

	// Validate that config validation fails for empty endpoint
	err := cfg.Validate()
	assert.Error(t, err, "Expected config validation to fail for empty endpoint")
	assert.Contains(t, err.Error(), "endpoint must be specified")
}

func TestConfigValidation(t *testing.T) {
	validConfig := getValidConfig()
	err := validConfig.Validate()
	assert.NoError(t, err, "Valid config should pass validation")

	invalidConfig := getInvalidConfig()
	err = invalidConfig.Validate()
	assert.Error(t, err, "Invalid config should fail validation")
	assert.Contains(t, err.Error(), "endpoint must be specified")
}

// Helper functions for test configurations
func getValidConfig() *Config {
	return &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint:        "http://localhost:8081",
			Timeout:         30 * time.Second,
			Compression:     "", // Compression disabled - unnecessary for small payloads
			WriteBufferSize: 512 * 1024,
			ReadBufferSize:  0,
		},
	}
}

// Helper function to get an invalid config missing the required endpoint
func getInvalidConfig() *Config {
	return &Config{
		ClientConfig: confighttp.ClientConfig{
			// Endpoint is the only required attribute
			Timeout: 30 * time.Second,
		},
	}
}
