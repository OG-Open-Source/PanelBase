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
	"github.com/OG-Open-Source/PanelBase/internal/config" // Use the correct module path

	// Import the new logging package
	logger "github.com/OG-Open-Source/PanelBase/internal/logging" // Import with logger alias
	"github.com/OG-Open-Source/PanelBase/internal/middleware"     // Import middleware
	"github.com/OG-Open-Source/PanelBase/internal/routes"         // Import the routes package
	"github.com/OG-Open-Source/PanelBase/internal/token_store"    // Import token store
	"github.com/OG-Open-Source/PanelBase/internal/ui_settings"    // Import ui_settings package
	"github.com/OG-Open-Source/PanelBase/internal/user"           // Import user package

	// Remove cors import: "github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const logsDir = "logs" // Consistent with bootstrap

func main() {
	// --- Early Logging Setup ---
	// Ensure logs directory exists BEFORE trying to open log file
	if _, err := os.Stat(logsDir); os.IsNotExist(err) {
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			log.Fatalf("FATAL: Failed to create logs directory '%s': %v", logsDir, err)
		}
	}
	timestamp := time.Now().UTC().Format("2006-01-02T15_04_05Z")
	logFileName := fmt.Sprintf("%s.log", timestamp)
	logFilePath := filepath.Join(logsDir, logFileName)
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		// Use standard log here as logging package might not be fully ready
		log.Fatalf("FATAL: Failed to open log file %s: %v", logFilePath, err)
	}
	defer logFile.Close()

	multiWriter := io.MultiWriter(os.Stdout, logFile)
	log.SetOutput(multiWriter)
	log.SetFlags(0) // IMPORTANT: Set flags to 0 BEFORE any logging calls
	gin.DefaultWriter = multiWriter
	// --- End Early Logging Setup ---

	logger.Printf("MAIN", "START", "Starting PanelBase...")

	// --- Bootstrap FIRST ---
	// Run bootstrap early to ensure config files and directories exist before loading.
	logger.Printf("MAIN", "BOOTSTRAP", "Running bootstrap process...")
	createdItems, err := bootstrap.Bootstrap()
	if err != nil {
		// Use standard log for critical bootstrap failure before full logging is configured
		log.Printf("FATAL: Failed to bootstrap application: %v", err)
		os.Exit(1)
	}
	if len(createdItems) > 0 {
		// Log created items *after* setting debug mode potentially
		// We'll log this later, after config load and debug mode set.
		// logger.Printf("MAIN", "BOOTSTRAP", "Bootstrap created items: %v", createdItems)
	}
	logger.Printf("MAIN", "BOOTSTRAP", "Bootstrap completed.")

	// --- Configuration Loading ---
	logger.Printf("MAIN", "CONFIG", "Loading configuration...")
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.ErrorPrintf("MAIN", "CONFIG", "Failed to load configuration: %v", err)
		os.Exit(1)
	}
	logger.Printf("MAIN", "CONFIG", "Configuration loaded: %+v", cfg.Server)

	// --- Initialize Logging Level (After config is loaded) ---
	logger.SetDebugMode(cfg.Server.Mode == "debug") // Set debug mode based on server mode

	// Log created bootstrap items *now* that logging level is set
	if len(createdItems) > 0 {
		logger.Printf("MAIN", "BOOTSTRAP", "Bootstrap created items: %v", createdItems)
	}

	// Initialize the token store database
	logger.Printf("MAIN", "TOKEN_STORE", "Initializing token store...")
	if err := token_store.InitStore(); err != nil {
		logger.ErrorPrintf("MAIN", "TOKEN_STORE", "Failed to initialize token store: %v", err)
		os.Exit(1)
	}
	defer token_store.CloseStore()
	logger.Printf("MAIN", "TOKEN_STORE", "Token store initialized.")

	// Load user configuration AFTER bootstrap and token store init
	logger.Printf("MAIN", "USER_SVC", "Loading user configuration...")
	if err := user.LoadUsersConfig(); err != nil {
		logger.ErrorPrintf("MAIN", "USER_SVC", "Failed to load user configuration: %v", err)
		os.Exit(1)
	}
	logger.Printf("MAIN", "USER_SVC", "User configuration loaded.")

	// Load UI settings configuration AFTER bootstrap
	logger.Printf("MAIN", "UI_SVC", "Loading UI settings...")
	if err := ui_settings.LoadUISettings(); err != nil {
		logger.ErrorPrintf("MAIN", "UI_SVC", "Failed to load UI settings configuration: %v", err)
		os.Exit(1)
	}
	logger.Printf("MAIN", "UI_SVC", "UI settings loaded.")

	// --- Gin Setup ---
	logger.Printf("MAIN", "GIN", "Setting up Gin router...")
	gin.SetMode(cfg.Server.Mode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(middleware.CacheRequestBody())
	router.Use(middleware.CustomLogger()) // CustomLogger already uses RFC3339
	routes.SetupRoutes(router, cfg)
	logger.Printf("MAIN", "GIN", "Gin router setup complete.")

	// --- Start Server ---
	serverAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    serverAddr,
		Handler: router,
	}

	go func() {
		logger.Printf("MAIN", "HTTP_SERVER", "Starting server in %s mode on %s...", strings.ToUpper(gin.Mode()), serverAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorPrintf("MAIN", "HTTP_SERVER", "Listen and serve error: %s", err)
			os.Exit(1)
		}
		logger.Printf("MAIN", "HTTP_SERVER", "HTTP server shut down gracefully.")
	}()

	// --- Shutdown Handling ---
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	receivedSignal := <-quit
	logger.Printf("MAIN", "SHUTDOWN", "Received signal: %v. Shutting down server...", receivedSignal)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.ErrorPrintf("MAIN", "SHUTDOWN", "Server forced to shutdown uncleanly: %v", err)
		os.Exit(1)
	}

	logger.Printf("MAIN", "SHUTDOWN", "Server exiting.")
}
