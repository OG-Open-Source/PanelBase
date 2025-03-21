package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server   ServerConfig   `toml:"server"`
	Security SecurityConfig `toml:"security"`
	Users    UsersConfig    `toml:"users"`
	Logging  LoggingConfig  `toml:"logging"`
	UI       UIConfig       `toml:"ui"`
}

type ServerConfig struct {
	Host  string `toml:"host"`
	Port  int    `toml:"port"`
	Entry string `toml:"entry"`
}

type SecurityConfig struct {
	JWTSecret      string `toml:"jwt_secret"`
	JWTExpireHours int    `toml:"jwt_expire_hours"`
}

type UsersConfig struct {
	Credentials []UserCredential `toml:"credentials"`
}

type UserCredential struct {
	Username string `toml:"username"`
	Password string `toml:"password"`
}

type LoggingConfig struct {
	Level      string `toml:"level"`
	File       string `toml:"file"`
	MaxSize    int    `toml:"max_size"`
	MaxBackups int    `toml:"max_backups"`
	MaxAge     int    `toml:"max_age"`
}

type UIConfig struct {
	ThemeFile string `toml:"theme_file"`
}

var GlobalConfig Config

func LoadConfig() error {
	configPath := filepath.Join("configs", "config.toml")

	file, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("error reading config file: %v", err)
	}

	if _, err := toml.Decode(string(file), &GlobalConfig); err != nil {
		return fmt.Errorf("error parsing config file: %v", err)
	}

	return nil
}

func GetConfig() *Config {
	return &GlobalConfig
}
