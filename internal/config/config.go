package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// Config defines the application configuration
type Config struct {
	Server ServerConfig `toml:"server"`
	Auth   AuthConfig   `toml:"auth"`
}

// ServerConfig defines server settings
type ServerConfig struct {
	Host string `toml:"host"`
	Port int    `toml:"port"`
	Mode string `toml:"mode"`
}

// AuthConfig defines authentication settings
type AuthConfig struct {
	JWTSecret   string `toml:"jwt_secret"`
	TokenExpiry int    `toml:"token_expiry"` // Expiration time in seconds
	CookieName  string `toml:"cookie_name"`
}

// LoadConfig loads configuration from file
func LoadConfig() (*Config, error) {
	configPath := filepath.Join("configs", "config.toml")

	// Create default config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := &Config{
			Server: ServerConfig{
				Host: "0.0.0.0",
				Port: 35960,
				Mode: "release",
			},
			Auth: AuthConfig{
				JWTSecret:   "your-secret-key",
				TokenExpiry: 86400, // 24 hours = 24 * 60 * 60 seconds
				CookieName:  "panelbase_jwt",
			},
		}

		// Ensure config directory exists
		if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
			return nil, err
		}

		// Save default config
		if err := SaveConfig(defaultConfig); err != nil {
			return nil, err
		}

		return defaultConfig, nil
	}

	// Read config file
	var config Config
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return nil, err
	}

	return &config, nil
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config) error {
	configPath := filepath.Join("configs", "config.toml")

	// Create file
	f, err := os.Create(configPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Encode and write TOML
	encoder := toml.NewEncoder(f)
	return encoder.Encode(config)
}
