package config

import (
	"os"
	"testing"
)

func TestApplyEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		initial  Config
		expected Config
	}{
		{
			name: "applies token from environment",
			envVars: map[string]string{
				"IONOS_TOKEN": "test-token-123",
			},
			initial: Config{},
			expected: Config{
				IONOS: IONOSConfig{
					Token: "test-token-123",
				},
			},
		},
		{
			name: "applies username and password from environment",
			envVars: map[string]string{
				"IONOS_USERNAME": "test-user",
				"IONOS_PASSWORD": "test-pass",
			},
			initial: Config{},
			expected: Config{
				IONOS: IONOSConfig{
					Username: "test-user",
					Password: "test-pass",
				},
			},
		},
		{
			name: "environment overrides config file values",
			envVars: map[string]string{
				"IONOS_TOKEN": "env-token",
			},
			initial: Config{
				IONOS: IONOSConfig{
					Token: "file-token",
				},
			},
			expected: Config{
				IONOS: IONOSConfig{
					Token: "env-token",
				},
			},
		},
		{
			name:    "no environment variables leaves config unchanged",
			envVars: map[string]string{},
			initial: Config{
				IONOS: IONOSConfig{
					Token: "file-token",
				},
			},
			expected: Config{
				IONOS: IONOSConfig{
					Token: "file-token",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear environment
			_ = os.Unsetenv("IONOS_TOKEN")
			_ = os.Unsetenv("IONOS_USERNAME")
			_ = os.Unsetenv("IONOS_PASSWORD")
			_ = os.Unsetenv("IONOS_API_URL")

			// Set test environment variables
			for key, value := range tt.envVars {
				_ = os.Setenv(key, value)
			}

			// Apply environment
			cfg := tt.initial
			cfg.ApplyEnvironment()

			// Check results
			if cfg.IONOS.Token != tt.expected.IONOS.Token {
				t.Errorf("Token = %v, want %v", cfg.IONOS.Token, tt.expected.IONOS.Token)
			}
			if cfg.IONOS.Username != tt.expected.IONOS.Username {
				t.Errorf("Username = %v, want %v", cfg.IONOS.Username, tt.expected.IONOS.Username)
			}
			if cfg.IONOS.Password != tt.expected.IONOS.Password {
				t.Errorf("Password = %v, want %v", cfg.IONOS.Password, tt.expected.IONOS.Password)
			}

			// Cleanup
			for key := range tt.envVars {
				_ = os.Unsetenv(key)
			}
		})
	}
}
