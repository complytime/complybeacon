package truthbeam

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/config/confighttp"
)

// The config tests are table-driven tests to validate configuration validation
//and default values for the truthbeam processor.

// TestConfigValidate tests the Validate method of the Config struct
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name:        "empty config should fail",
			config:      &Config{},
			expectError: true,
			errorMsg:    "must be specified",
		},
		{
			name: "valid endpoint should pass",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "http://example.com",
				},
			},
			expectError: false,
		},
		{
			name: "https endpoint should pass",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "https://api.example.com:8080",
				},
			},
			expectError: false,
		},
		{
			name: "endpoint with path should pass",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "http://localhost:8081/v1",
				},
			},
			expectError: false,
		},
		{
			name: "empty string endpoint should fail",
			config: &Config{
				ClientConfig: confighttp.ClientConfig{
					Endpoint: "",
				},
			},
			expectError: true,
			errorMsg:    "must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err, "Expected validation error")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Expected no validation error")
			}
		})
	}
}

func TestConfigStruct(t *testing.T) {
	// Test that Config struct can be created and accessed
	cfg := &Config{
		ClientConfig: confighttp.ClientConfig{
			Endpoint: "http://localhost:8081",
		},
	}

	// Test that we can access the embedded ClientConfig
	assert.Equal(t, "http://localhost:8081", cfg.ClientConfig.Endpoint)

	// Test that validation passes
	err := cfg.Validate()
	assert.NoError(t, err, "Config with valid endpoint should pass validation")
}
