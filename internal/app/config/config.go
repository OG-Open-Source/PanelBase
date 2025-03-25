package config

import (
	"fmt"
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config represents the application configuration
type Config struct {
	Logging LoggingConfig `toml:"logging"`
	Server  ServerConfig  `toml:"server"`
}

// LoggingConfig holds logging-related configuration
type LoggingConfig struct {
	File       string `toml:"file"`
	Level      string `toml:"level"`
	MaxSize    int    `toml:"max_size"`    // in MB
	MaxBackups int    `toml:"max_backups"` // number of backups to keep
	MaxAge     int    `toml:"max_age"`     // days to keep old logs
}

// ServerConfig holds server-related configuration
type ServerConfig struct {
	Host  string `toml:"host"`
	Port  int    `toml:"port"`
	Entry string `toml:"entry"` // Secret entry path for admin access
}

// LoadConfig loads and parses the configuration file
func LoadConfig(path string) (*Config, error) {
	// Read file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse TOML
	var config Config
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for any missing values
	applyDefaults(&config)

	return &config, nil
}

// applyDefaults sets default values for config options that weren't specified
func applyDefaults(config *Config) {
	// Logging defaults
	if config.Logging.Level == "" {
		config.Logging.Level = "info"
	}
	if config.Logging.MaxSize == 0 {
		config.Logging.MaxSize = 10 // 10 MB
	}
	if config.Logging.MaxBackups == 0 {
		config.Logging.MaxBackups = 5
	}
	if config.Logging.MaxAge == 0 {
		config.Logging.MaxAge = 30 // 30 days
	}

	// Server defaults
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
}
