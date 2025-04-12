package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/OG-Open-Source/PanelBase/internal/api/v1"
	v1handlers "github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers"
	"github.com/OG-Open-Source/PanelBase/internal/storage"
	"github.com/OG-Open-Source/PanelBase/internal/webserver"
	"github.com/OG-Open-Source/PanelBase/pkg/bootstrap"
	"github.com/OG-Open-Source/PanelBase/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
)

// AppConfig holds application configuration read from config.toml
type AppConfig struct {
	Server struct {
		Port  int    `toml:"port"`
		Entry string `toml:"entry"`
		Ip    string `toml:"ip"`
		Mode  string `toml:"mode"`
	}
	Auth struct {
		JwtSecret            string `toml:"jwt_secret"`
		TokenDurationMinutes int    `toml:"token_duration_minutes"`
	}
	Functions struct {
		Users bool `toml:"users"`
	}
}

const (
	defaultPort          = 8080
	defaultIP            = "0.0.0.0"
	configFile           = "configs/config.toml"
	usersFile            = "configs/users.json"
	uiSettingsFile       = "configs/ui_settings.json"
	defaultMode          = gin.ReleaseMode
	defaultJwtSecret     = "change-this-in-config-toml" // Default secret (should be changed)
	defaultTokenDuration = 60                           // Default duration in minutes
)

func main() {
	// --- Initialize Project Structure ---
	if err := bootstrap.InitializeProject(); err != nil {
		log.Printf("WARNING: Project initialization failed: %v", err)
	}
	// --- End Initialization ---

	// --- Setup Logging ---
	logWriter, logFile, err := logger.SetupLogWriter("logs")
	if err != nil {
		log.Fatalf("Failed to set up logger: %v", err)
	}
	if logFile != nil {
		defer logFile.Close()
	}
	log.SetOutput(logWriter)
	gin.DefaultWriter = logWriter
	gin.ForceConsoleColor()
	// --- End Logging Setup ---

	// --- Load Configuration ---
	config := AppConfig{
		Server: struct {
			Port  int    `toml:"port"`
			Entry string `toml:"entry"`
			Ip    string `toml:"ip"`
			Mode  string `toml:"mode"`
		}{Port: defaultPort, Ip: defaultIP, Mode: defaultMode},
		Auth: struct {
			JwtSecret            string `toml:"jwt_secret"`
			TokenDurationMinutes int    `toml:"token_duration_minutes"`
		}{JwtSecret: defaultJwtSecret, TokenDurationMinutes: defaultTokenDuration},
		Functions: struct {
			Users bool `toml:"users"`
		}{Users: false},
	}
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("WARN: Failed to read config file '%s': %v. Using default server settings.", configFile, err)
	} else {
		err = toml.Unmarshal(configData, &config)
		if err != nil {
			log.Printf("WARN: Failed to parse config file '%s': %v. Using default/incomplete server settings.", configFile, err)
			if config.Server.Port == 0 {
				config.Server.Port = defaultPort
			}
			if config.Server.Ip == "" {
				config.Server.Ip = defaultIP
			}
			if config.Server.Mode == "" {
				config.Server.Mode = defaultMode
			}
			if config.Auth.JwtSecret == "" {
				config.Auth.JwtSecret = defaultJwtSecret
			}
			if config.Auth.TokenDurationMinutes <= 0 {
				config.Auth.TokenDurationMinutes = defaultTokenDuration
			}
			if !config.Functions.Users {
				config.Functions.Users = false
			}
		}
	}
	if config.Server.Ip == "" {
		config.Server.Ip = defaultIP
		log.Printf("WARN: Server IP not set in config, defaulting to %s", defaultIP)
	}
	if config.Server.Mode == "" {
		config.Server.Mode = defaultMode
		log.Printf("WARN: Server Mode not set in config, defaulting to %s", defaultMode)
	}
	if config.Auth.JwtSecret == "" || config.Auth.JwtSecret == defaultJwtSecret {
		config.Auth.JwtSecret = defaultJwtSecret
		log.Printf("WARN: Using default JWT secret. Change 'jwt_secret' in %s for production!", configFile)
	}
	if config.Auth.TokenDurationMinutes <= 0 {
		config.Auth.TokenDurationMinutes = defaultTokenDuration
		log.Printf("WARN: Invalid token duration, defaulting to %d minutes", defaultTokenDuration)
	}

	// --- Mode-Specific Overrides ---
	debugModeActive := strings.ToLower(config.Server.Mode) == gin.DebugMode
	if debugModeActive {
		const debugPort = 32768
		if config.Server.Port != debugPort {
			log.Printf("INFO: Debug mode active, overriding server port to %d (ignoring config value %d)", debugPort, config.Server.Port)
			config.Server.Port = debugPort
		}
	}
	// --- End Mode-Specific Overrides ---

	listenAddr := fmt.Sprintf("%s:%d", config.Server.Ip, config.Server.Port)
	// --- End Load Configuration ---

	// --- Set Gin Mode BEFORE creating router ---
	ginMode := gin.DebugMode // Default for logic
	if !debugModeActive {    // Use the already checked variable
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)
	// --- End Set Gin Mode ---

	// --- Load UI Settings ---
	uiSettingsData := make(map[string]interface{})
	uiSettingsBytes, err := os.ReadFile(uiSettingsFile)
	if err != nil {
		log.Printf("WARN: Failed to read UI settings file '%s': %v. UI data will be empty.", uiSettingsFile, err)
	} else {
		err = json.Unmarshal(uiSettingsBytes, &uiSettingsData)
		if err != nil {
			log.Printf("WARN: Failed to parse UI settings file '%s': %v. UI data will be empty.", uiSettingsFile, err)
			uiSettingsData = make(map[string]interface{}) // Ensure it's an empty map on error
		}
	}
	// --- End Load UI Settings ---

	// --- Initialize User Store ---
	userStore, err := storage.NewJSONUserStore(usersFile)
	if err != nil {
		// Allow server to start even if users.json is missing/corrupt, but log critical error
		log.Printf("CRITICAL: Failed to initialize user store from '%s': %v. User functions will likely fail.", usersFile, err)
		// Assign a nil store or a dummy store if needed, depending on how handlers cope
		userStore = nil // Or potentially a dummy implementation that always returns errors
	}
	// --- End User Store Initialization ---

	// Create a gin router
	router := gin.New()
	router.Use(gin.Recovery())
	logConfig := gin.LoggerConfig{
		Formatter: logger.CustomLogFormatter,
		Output:    logWriter,
	}
	router.Use(gin.LoggerWithConfig(logConfig))

	// --- Register Explicit Routes FIRST ---
	// Initialize Handlers
	authHandler := v1handlers.NewAuthHandler(userStore, config.Auth.JwtSecret, config.Auth.TokenDurationMinutes)

	// API v1 group
	apiV1 := router.Group("/api/v1")
	v1.RegisterRoutes(apiV1, authHandler, config.Auth.JwtSecret, config.Functions.Users)

	// --- Setup Web Server Handlers (Static Files, Templates, Errors) ---
	// Pass the necessary parts of AppConfig and uiSettingsData
	webConfig := webserver.AppConfig{
		Server: struct {
			Entry string `toml:"entry"`
			Mode  string // Pass Mode to webserver for debug endpoint
		}{Entry: config.Server.Entry, Mode: config.Server.Mode},
	}
	webserver.RegisterHandlers(router, webConfig, uiSettingsData, uiSettingsFile)
	// --- End Web Server Handlers ---

	// Determine adminEntryPath for logging *after* webserver handlers are registered
	adminEntryPathForLog := ""
	if config.Server.Entry != "" {
		// Check if the entry directory actually exists, similar to how webserver does
		checkEntryDir := filepath.Join("web", config.Server.Entry)
		if _, err := os.Stat(checkEntryDir); err == nil {
			adminEntryPathForLog = fmt.Sprintf(" | Admin Entry: /%s/", config.Server.Entry)
		} else {
			adminEntryPathForLog = " | Admin Entry: (Configured but dir not found)"
		}
	} else {
		adminEntryPathForLog = " | Admin Entry: (Not Configured)"
	}

	// Start the server with combined log message
	log.Printf("Starting server | Mode: %s | Addr: %s%s",
		ginMode,
		listenAddr,
		adminEntryPathForLog, // Use the determined path for logging
	)
	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
