package config

import (
	"os"
	"strconv"
)

type Config struct {
	Port          int
	SecurityEntry string
}

func LoadConfig() *Config {
	port, _ := strconv.Atoi(os.Getenv("PANELBASE_PORT"))
	if port == 0 {
		port = 8080 // 默認端口
	}
	
	return &Config{
		Port:          port,
		SecurityEntry: getEnv("PANELBASE_SECURITY_ENTRY", "default-entry"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}