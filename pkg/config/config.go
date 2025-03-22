package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Server struct {
		Host  string `mapstructure:"host"`
		Port  int    `mapstructure:"port"`
		Entry string `mapstructure:"entry"`
	} `mapstructure:"server"`

	Security struct {
		JWTSecret      string `mapstructure:"jwt_secret"`
		JWTExpireHours int    `mapstructure:"jwt_expire_hours"`
	} `mapstructure:"security"`

	Logging struct {
		Level      string `mapstructure:"level"`
		File       string `mapstructure:"file"`
		MaxSize    int    `mapstructure:"max_size"`
		MaxBackups int    `mapstructure:"max_backups"`
		MaxAge     int    `mapstructure:"max_age"`
	} `mapstructure:"logging"`
}

// LoadConfig loads configuration from file or environment variables
// If the config directory or file doesn't exist, it creates a default one
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default config name and path
	configName := "config"
	if configPath == "" {
		// Look for config in default locations
		configPath = "configs"
	}

	// Create configs directory if it doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.MkdirAll(configPath, 0755); err != nil {
			return nil, fmt.Errorf("unable to create config directory: %w", err)
		}
	}

	configFilePath := filepath.Join(configPath, configName+".toml")

	// Check if config file exists, if not create a default one
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		// Generate default configuration values
		port, err := utils.FindAvailablePort(1024, 49151)
		if err != nil {
			port = 8080 // Fallback to 8080 if random port generation fails
		}

		entry, err := utils.GenerateRandomString(12)
		if err != nil {
			entry = "admin" // Fallback to a simple entry if random generation fails
		}

		jwtSecret, err := utils.GenerateRandomString(64)
		if err != nil {
			jwtSecret = "insecure_jwt_secret_please_change" // Fallback to an insecure secret
		}

		// Create default config content
		defaultConfig := fmt.Sprintf(`[server]
host = "0.0.0.0"
port = %d
entry = "%s"

[security]
jwt_secret = "%s"
jwt_expire_hours = 24

[logging]
level = "info"
file = "logs/panelbase.log"
max_size = 10
max_backups = 5
max_age = 30
`, port, entry, jwtSecret)

		// Write default config to file
		if err := os.WriteFile(configFilePath, []byte(defaultConfig), 0644); err != nil {
			return nil, fmt.Errorf("unable to create default config file: %w", err)
		}
	}

	// Set config file properties
	v.SetConfigName(configName)
	v.SetConfigType("toml")
	v.AddConfigPath(configPath)

	// Read from environment variables that match
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	// Parse the config into our Config struct
	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	// Create logs directory if it doesn't exist
	if config.Logging.File != "" {
		logDir := filepath.Dir(config.Logging.File)
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			if err := os.MkdirAll(logDir, 0755); err != nil {
				return nil, fmt.Errorf("unable to create log directory: %w", err)
			}
		}
	}

	return &config, nil
}

// GetEntryURL returns the formatted entry URL based on configuration
func (c *Config) GetEntryURL() string {
	return fmt.Sprintf("%s:%d/%s/", c.Server.Host, c.Server.Port, c.Server.Entry)
}

// GetServerAddress returns the formatted server address
func (c *Config) GetServerAddress() string {
	return fmt.Sprintf("%s:%d", c.Server.Host, c.Server.Port)
}
