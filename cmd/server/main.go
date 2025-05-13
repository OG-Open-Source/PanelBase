package main

import (
	"fmt"
	"log"
	"math/rand" // Added import
	"net"
	"os"
	"path/filepath"
	"time" // Added import

	"github.com/OG-Open-Source/PanelBase/internal/config"
	// "github.com/OG-Open-Source/PanelBase/internal/plugin" // Removed plugin import
	"github.com/OG-Open-Source/PanelBase/internal/server"

	"gopkg.in/yaml.v3"
)

// findAvailablePort tries to find an available port within the range 1024-49151.
func findAvailablePort() (string, error) {
	rand.Seed(time.Now().UnixNano()) // Seed the random number generator
	for i := 0; i < 100; i++ {       // Try up to 100 times
		port := rand.Intn(49151-1024+1) + 1024 // Generate random port in range [1024, 49151]
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return fmt.Sprintf("%d", port), nil // Port is available
		}
	}
	// Fallback to OS-assigned port if random attempts fail
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", fmt.Errorf("failed to find an available port after multiple attempts: %w", err)
	}
	defer listener.Close()
	addrInfo := listener.Addr().(*net.TCPAddr)
	return fmt.Sprintf("%d", addrInfo.Port), nil
}

// ensureConfigFile checks if the config file exists. If not, it creates a default one.
func ensureConfigFile(configPath string) error {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Printf("Config file '%s' not found. Creating default config...", configPath)

		// Ensure the directory exists
		dir := filepath.Dir(configPath)
		if err = os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create config directory '%s': %w", dir, err)
		}

		var defaultPort string // Declare defaultPort
		// Find an available port for the default config
		defaultPort, err = findAvailablePort()
		if err != nil {
			log.Printf("Warning: could not find an available port automatically, using default 8080: %v", err)
			defaultPort = "8080" // Fallback port
		}

		// Create default configuration content
		defaultCfg := config.Config{
			Server: struct {
				Host string `yaml:"host"`
				Port string `yaml:"port"`
			}{
				Host: "0.0.0.0", // Bind to all interfaces by default
				Port: defaultPort,
			},
		}

		// Marshal default config to YAML
		data, err := yaml.Marshal(&defaultCfg)
		if err != nil {
			return fmt.Errorf("failed to marshal default config: %w", err)
		}

		// Write the default config file
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to write default config file '%s': %w", configPath, err)
		}
		log.Printf("Default config file created at '%s' with port %s", configPath, defaultPort)

	} else if err != nil {
		// Another error occurred during stat
		return fmt.Errorf("error checking config file '%s': %w", configPath, err)
	}
	return nil
}

func main() {
	configPath := "configs/config.yaml"

	// Ensure config file exists or create a default one
	if err := ensureConfigFile(configPath); err != nil {
		log.Fatalf("Failed to ensure config file: %v", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Construct server address from config
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port)

	// Plugin loading logic removed

	// Start the web server
	// Allow overriding port via environment variable (e.g., for Heroku/Cloud Run)
	port := os.Getenv("PORT")
	if port != "" {
		serverAddr = fmt.Sprintf("%s:%s", cfg.Server.Host, port) // Use configured host, override port
	}

	server.StartServer(serverAddr)
}
