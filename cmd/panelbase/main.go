package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/server"
)

func main() {
	// Initialize logger
	logger.Init()
	defer logger.Cleanup()

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to load config: %v", err))
	}
	logger.Info("Configuration loaded successfully")

	// Create and start server
	srv, err := server.New(cfg)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to initialize server: %v", err))
	}

	logger.Info(fmt.Sprintf("Starting server on %s:%d", cfg.IP, cfg.Port))
	logger.Info(fmt.Sprintf("Web interface available at http://localhost:%d/%s", cfg.Port, cfg.EntryPoint))
	logger.Info(fmt.Sprintf("API endpoint available at http://localhost:%d/%s/api", cfg.Port, cfg.EntryPoint))
	logger.Info("Default admin user created with username 'admin' and password 'admin'")
	logger.Info("IMPORTANT: Please change the default admin password immediately!")

	// Start server in a new goroutine
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal(fmt.Sprintf("Server failed: %v", err))
		}
	}()

	// Wait for termination signal
	<-sigChan
	logger.Info("Shutting down server...")
}
