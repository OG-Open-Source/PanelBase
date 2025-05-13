package config

import (
	"fmt"
	"net" // Added import
	"os"
	"strconv" // Added import

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration.
type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	} `yaml:"server"`
	// Plugins struct { // Removed by Trae AI
	// 	Directory string `yaml:"directory"` // Removed by Trae AI
	// } `yaml:"plugins"` // Removed by Trae AI
	// Panels struct {
	// 	Directory string `yaml:"directory"`
	// } `yaml:"panels"` // Removed Panels struct
}

// Validate checks if the configuration values are valid.
func (c *Config) Validate() error {
	// Validate Server Host
	if c.Server.Host != "0.0.0.0" { // Allow 0.0.0.0
		ip := net.ParseIP(c.Server.Host)
		// Check if it's a valid IP and specifically an IPv4 address
		if ip == nil || ip.To4() == nil {
			return fmt.Errorf("invalid server host: '%s' is not a valid IPv4 address", c.Server.Host)
		}
	}

	// Validate Server Port
	port, err := strconv.Atoi(c.Server.Port)
	if err != nil {
		return fmt.Errorf("invalid server port: '%s' is not a number", c.Server.Port)
	}
	// Check if port is within the dynamic/private port range (excluding well-known and system ports)
	if port < 1024 || port > 49151 {
		return fmt.Errorf("invalid server port: %d is outside the valid range (1024-49151)", port)
	}

	return nil
}

// LoadConfig reads the configuration file from the given path and returns a Config struct.
// It assumes the config file exists and is valid YAML.
func LoadConfig(configPath string) (*Config, error) {
	// Read the configuration file
	data, err := os.ReadFile(configPath)
	if err != nil {
		// Error reading file (could be permission issue, or file doesn't exist)
		return nil, fmt.Errorf("error reading config file '%s': %w", configPath, err)
	}

	// Initialize an empty config struct to unmarshal into
	cfg := &Config{}

	// Unmarshal YAML data into the Config struct
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		// Error parsing YAML
		return nil, fmt.Errorf("error unmarshalling config file '%s': %w", configPath, err)
	}

	// Validate the loaded config
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration in '%s': %w", configPath, err)
	}

	// Return the loaded config
	return cfg, nil
}
