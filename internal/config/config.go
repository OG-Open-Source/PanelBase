package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"github.com/joho/godotenv"
)

type Config struct {
	IP             string   `json:"ip"`
	Port           int      `json:"port"`
	AllowedOrigins string   `json:"allowed_origins"`
	ProxyTarget    string   `json:"proxy_target"`
	EntryPoint     string   `json:"entry_point"`
	JWTSecret      string   `json:"jwt_secret"`
	TrustedIPs     []string `json:"trusted_ips"`
	WorkDir        string   `json:"work_dir"`
}

func Load() (*Config, error) {
	// 加載 .env 文件
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	// 檢查必要的環境變量
	requiredEnvs := map[string]string{
		"IP":         getEnvOrDefault("IP", ""),
		"PORT":       getEnvOrDefault("PORT", ""),
		"ENTRY":      getEnvOrDefault("ENTRY", ""),
		"JWT_SECRET": getEnvOrDefault("JWT_SECRET", ""),
		"WORK_DIR":   getEnvOrDefault("WORK_DIR", ""),
	}

	// 檢查是否有空值
	var missingEnvs []string
	for key, value := range requiredEnvs {
		if value == "" {
			missingEnvs = append(missingEnvs, key)
		}
	}

	if len(missingEnvs) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %v", missingEnvs)
	}

	port, err := strconv.Atoi(requiredEnvs["PORT"])
	if err != nil {
		return nil, fmt.Errorf("invalid PORT value: %v", err)
	}

	return &Config{
		IP:             requiredEnvs["IP"],
		Port:           port,
		AllowedOrigins: "*",
		ProxyTarget:    "",
		EntryPoint:     requiredEnvs["ENTRY"],
		JWTSecret:      requiredEnvs["JWT_SECRET"],
		TrustedIPs:     getTrustedIPs(),
		WorkDir:        requiredEnvs["WORK_DIR"],
	}, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getTrustedIPs() []string {
	ips := os.Getenv("PANEL_TRUSTED_IPS")
	if ips == "" {
		return []string{"127.0.0.1"}
	}
	return strings.Split(ips, ",")
}