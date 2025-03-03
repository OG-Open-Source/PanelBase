package config

import (
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
    Domain         string   `json:"domain"`
    JWTSecret      string   `json:"jwt_secret"`
    TrustedIPs     []string `json:"trusted_ips"`
}

func Load() (*Config, error) {
    // 加載 .env 文件
    if err := godotenv.Load(); err != nil {
        return nil, err
    }

    port, err := strconv.Atoi(getEnvOrDefault("PORT", "8080"))
    if err != nil {
        port = 8080
    }

    return &Config{
        IP:             getEnvOrDefault("IP", "0.0.0.0"),
        Port:           port,
        AllowedOrigins: "*",
        ProxyTarget:    "",
        EntryPoint:     getEnvOrDefault("ENTRY", "panel"),
        Domain:         getEnvOrDefault("DOMAIN", ""),
        JWTSecret:      getEnvOrDefault("JWT_SECRET", "your-secret-key"),
        TrustedIPs:     getTrustedIPs(),
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