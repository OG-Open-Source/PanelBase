package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/OG-Open-Source/PanelBase/internal/app/handlers"
	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/gin-gonic/gin"
)

func main() {
	// Try to find current working directory
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("Failed to get working directory: %v", err)
	}

	// Initialize configuration
	configInitializer := config.NewConfigInitializer()
	configInitializer.SetBaseDir(workDir)

	// Check and initialize configuration files
	if err := configInitializer.Initialize(); err != nil {
		log.Fatalf("Failed to initialize configuration: %v", err)
	}

	// Build configuration file path
	configPath := filepath.Join(workDir, "configs", "config.yaml")
	log.Printf("Using configuration file: %s", configPath)

	// Load configuration
	configService, err := services.NewConfigService(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode
	gin.SetMode(configService.Config.Server.Mode)

	// Create router
	router := gin.Default()

	// Setup routes
	handlers.SetupRoutes(router, configService)

	// Start server
	serverAddr := fmt.Sprintf("%s:%d",
		configService.Config.Server.Host,
		configService.Config.Server.Port,
	)
	log.Printf("Starting PanelBase server at %s", serverAddr)
	log.Printf("Mode: %s", configService.Config.Server.Mode)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
