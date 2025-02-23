package config

type Config struct {
	Port  string
	Entry string
}

func LoadConfig() (*Config, error) {
	// TODO: Load configuration from .env file
	return &Config{}, nil
} 