package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	IONOS      IONOSConfig `yaml:"ionos"`
	Kubeconfig string      `yaml:"kubeconfig,omitempty"`
}

type IONOSConfig struct {
	Token    string `yaml:"token,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
	APIURL   string `yaml:"api_url,omitempty"`
}

func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ionos-cloud-watchdog"), nil
}

func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath) //nolint:gosec // configPath is from trusted GetConfigPath()
	if err != nil {
		if os.IsNotExist(err) {
			return &Config{}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &cfg, nil
}

func Save(cfg *Config) error {
	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func (c *Config) ApplyEnvironment() {
	if token := os.Getenv("IONOS_TOKEN"); token != "" {
		c.IONOS.Token = token
	}
	if username := os.Getenv("IONOS_USERNAME"); username != "" {
		c.IONOS.Username = username
	}
	if password := os.Getenv("IONOS_PASSWORD"); password != "" {
		c.IONOS.Password = password
	}
	if apiURL := os.Getenv("IONOS_API_URL"); apiURL != "" {
		c.IONOS.APIURL = apiURL
	}
}
