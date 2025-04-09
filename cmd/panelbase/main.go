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
	"strings"
	"syscall"
	"time"

	//"net/http" // No longer needed here directly for routes

	// Import the config package

	"github.com/OG-Open-Source/PanelBase/internal/bootstrap"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/routes"
	"github.com/OG-Open-Source/PanelBase/pkg/config"
	"github.com/OG-Open-Source/PanelBase/pkg/tokenstore"
	"github.com/OG-Open-Source/PanelBase/pkg/uisettings"
	"github.com/OG-Open-Source/PanelBase/pkg/userservice"

	// Remove cors import: "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const logsDir = "logs" // Consistent with bootstrap

func main() {
	// Bootstrap: Ensure config/logs directories exist first
	createdItems, err := bootstrap.Bootstrap()
	if err != nil {
		log.Fatalf("Failed to bootstrap application: %v", err)
	}
	// Log created items if any
	if len(createdItems) > 0 {
		log.Printf("Bootstrap created items: %s", strings.Join(createdItems, " "))
	}

	// Initialize the token store database
	if err := tokenstore.InitStore(); err != nil {
		log.Fatalf("Failed to initialize token store: %v", err)
	}
	defer tokenstore.CloseStore()

	// Load user configuration AFTER bootstrap and token store init
	if err := userservice.LoadUsersConfig(); err != nil {
		log.Fatalf("Failed to load user configuration: %v", err)
	}

	// Load UI settings configuration AFTER bootstrap
	if err := uisettings.LoadUISettings(); err != nil {
		log.Fatalf("Failed to load UI settings configuration: %v", err)
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
	log.SetFlags(0) // Remove default flags to allow custom formatting
	gin.DefaultWriter = multiWriter

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode directly, without extra logging.
	gin.SetMode(cfg.Server.Mode)

	// Initialize Gin engine
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CacheRequestBody())
	router.Use(middleware.CustomLogger())

	// Remove CORS middleware usage
	// router.Use(cors.Default())

	// Setup routes
	routes.SetupRoutes(router)

	// Create the HTTP server instance
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("%s Starting server in %s mode on %s...", time.Now().UTC().Format(time.RFC3339), strings.ToUpper(gin.Mode()), serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("%s Listen and serve error: %s", time.Now().UTC().Format(time.RFC3339), err)
		}
		log.Printf("%s HTTP server shut down gracefully", time.Now().UTC().Format(time.RFC3339))
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	receivedSignal := <-quit
	log.Printf("%s Received signal: %v. Shutting down server...", time.Now().UTC().Format(time.RFC3339), receivedSignal)

	// Graceful shutdown context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("%s Server forced to shutdown uncleanly: %v", time.Now().UTC().Format(time.RFC3339), err)
	}

	log.Printf("%s Server exiting", time.Now().UTC().Format(time.RFC3339))
}
