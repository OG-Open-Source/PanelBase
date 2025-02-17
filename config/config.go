package config

import (
	"os"
	"strconv"
)

type Config struct {
	IP           string
	Port         int
	SecurityEntry string
}

func LoadConfig() *Config {
	port, _ := strconv.Atoi(os.Getenv("PANELBASE_PORT"))
	if port == 0 {
		port = 8080 // 默認端口
	}

	return &Config{
		IP:           getEnv("PANELBASE_IP", "0.0.0.0"),
		Port:         port,
		SecurityEntry: getEnv("PANELBASE_SECURITY_ENTRY", "panelbase"),
	}
}

func getEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}