package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	//"net/http" // No longer needed here directly for routes

	// Import the config package
	"github.com/OG-Open-Source/PanelBase/internal/bootstrap"
	"github.com/OG-Open-Source/PanelBase/internal/config"     // Use the correct module path
	"github.com/OG-Open-Source/PanelBase/internal/middleware" // Import middleware
	"github.com/OG-Open-Source/PanelBase/internal/routes"     // Import the routes package
	"github.com/OG-Open-Source/PanelBase/internal/token_store" // Import token store
	"github.com/OG-Open-Source/PanelBase/internal/user"       // Import user package

	// Remove cors import: "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const logsDir = "logs" // Consistent with bootstrap

func main() {
	// Bootstrap: Ensure config/logs directories exist first
	if err := bootstrap.Bootstrap(); err != nil {
		log.Fatalf("Failed to bootstrap application: %v", err)
	}

	// Initialize the token store database
	if err := token_store.InitStore(); err != nil {
		log.Fatalf("Failed to initialize token store: %v", err)
	}
	defer token_store.CloseStore()

	// Load user configuration AFTER bootstrap and token store init
	if err := user.LoadUsersConfig(); err != nil {
		log.Fatalf("Failed to load user configuration: %v", err)
	}

	// --- Setup Logging (Reduced Output) ---
	// Generate timestamped log filename
	timestamp := time.Now().UTC().Format("2006-01-02T15_04_05Z") // Use underscores for filename compatibility
	logFileName := fmt.Sprintf("%s.log", timestamp)
	logFilePath := filepath.Join(logsDir, logFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("Failed to open log file %s: %v", logFilePath, err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(log.LstdFlags) // Keep date and time
	gin.DefaultWriter = multiWriter

	// Remove verbose logging setup message
	// log.Println("Logging configured to output to console and", logFilePath)
	// --- End Logging Setup ---

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Remove logging of loaded mode
	// log.Printf("[Config] Loaded server mode from config.toml: '%s'", cfg.Server.Mode)

	// Set Gin mode directly, without extra logging.
	gin.SetMode(cfg.Server.Mode)
	// Remove logging of mode setting
	// log.Printf("[Main] Set Gin mode based on config: '%s'", cfg.Server.Mode)

	// Initialize Gin engine
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CacheRequestBody())
	router.Use(middleware.CustomLogger())

	// Remove CORS middleware usage
	// router.Use(cors.Default())

	// Setup routes
	routes.SetupRoutes(router, cfg)

	// Create the HTTP server instance
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		// Keep the essential starting server message
		log.Printf("Starting server on %s in %s mode...", serverAddr, gin.Mode())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen and serve error: %s\n", err)
		}
		// Keep the server stopped message
		log.Println("HTTP server shut down gracefully.")
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	receivedSignal := <-quit
	// Keep shutdown messages
	log.Printf("Received signal: %v. Shutting down server...", receivedSignal)

	// Graceful shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown uncleanly: ", err)
	}

	// Keep the final exiting message
	log.Println("Server exiting.")
}
