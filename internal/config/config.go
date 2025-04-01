package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

// Config holds the application configuration.
type Config struct {
	Server struct {
		Host string `toml:"host"`
		Port int    `toml:"port"`
		Mode string `toml:"mode"` // debug or release
	} `toml:"server"`
	Features struct {
		Commands bool `toml:"commands"`
		Plugins  bool `toml:"plugins"`
	} `toml:"features"`
	Auth struct {
		JWTSecret          string        `toml:"-"`              // Now loaded from users.json
		TokenDuration      time.Duration `toml:"-"`              // Calculated from jwt_expiration
		JWTExpirationHours int           `toml:"jwt_expiration"` // Raw value from TOML (hours)
		CookieName         string        `toml:"cookie_name"`    // Name of the JWT cookie
	} `toml:"auth"`
}

// Path to the main configuration file
const ConfigFilePath = "configs/config.toml"
const UsersConfigPath = "configs/users.json"

// Temp struct to unmarshal only the JWT secret from users.json
type usersAuthTemp struct {
	JWTSecret string `json:"jwt_secret"`
}

// LoadConfig reads the configuration files.
func LoadConfig() (*Config, error) {
	cfg := &Config{}

	// 1. Read main config file (config.toml)
	tomlFile, err := os.ReadFile(ConfigFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %s: %w", ConfigFilePath, err)
	}
	err = toml.Unmarshal(tomlFile, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config TOML: %w", err)
	}

	// 2. Read users config file (users.json) to get JWT secret
	usersFile, err := os.ReadFile(UsersConfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read users config file %s: %w", UsersConfigPath, err)
	}

	var usersAuth usersAuthTemp
	err = json.Unmarshal(usersFile, &usersAuth)
	if err != nil {
		// Check if it's just that the key is missing, or a real JSON error
		if _, ok := err.(*json.UnmarshalTypeError); !ok && err.Error() != "json: unknown field \"users\"" && !strings.Contains(err.Error(), "cannot unmarshal") {
			// If it's a real JSON syntax error, fail
			return nil, fmt.Errorf("failed to unmarshal JWT secret from %s: %w", UsersConfigPath, err)
		}
		// If key missing or other fields present, try to proceed but check secret below
	}

	// 3. Assign JWT Secret
	cfg.Auth.JWTSecret = usersAuth.JWTSecret
	if cfg.Auth.JWTSecret == "" {
		// Ensure the secret was actually loaded
		return nil, fmt.Errorf("jwt_secret field is missing or empty in %s", UsersConfigPath)
	}

	// Calculate Token Duration
	if cfg.Auth.JWTExpirationHours <= 0 {
		// Default to 24 hours if not set or invalid
		cfg.Auth.JWTExpirationHours = 24
	}
	cfg.Auth.TokenDuration = time.Duration(cfg.Auth.JWTExpirationHours) * time.Hour

	return cfg, nil
}
