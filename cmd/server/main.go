package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	v1 "github.com/OG-Open-Source/PanelBase/internal/api/v1"
	v1handlers "github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers"
	"github.com/OG-Open-Source/PanelBase/internal/api/v1/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/storage"
	"github.com/OG-Open-Source/PanelBase/internal/webserver"
	"github.com/OG-Open-Source/PanelBase/pkg/bootstrap"
	"github.com/OG-Open-Source/PanelBase/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/time/rate"
)

// AppConfig holds application configuration read from config.toml
type AppConfig struct {
	Server struct {
		Port         int     `toml:"port"`
		Entry        string  `toml:"entry"`
		Ip           string  `toml:"ip"`
		Mode         string  `toml:"mode"`
		RateLimitR   float64 `toml:"rate_limit_r"`  // Requests per second
		RateLimitB   int     `toml:"rate_limit_b"`  // Burst size
		TrustedProxy string  `toml:"trusted_proxy"` // Single trusted proxy IP or CIDR
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
	defaultRateLimitR    = 10.0                         // Default rate limit: 10 req/sec
	defaultRateLimitB    = 20                           // Default burst size: 20
	defaultTrustedProxy  = ""                           // Default: No trusted proxy
)

func main() {
	// --- Initialize Project Structure (Handles initial user creation) ---
	if err := bootstrap.InitializeProject(); err != nil {
		log.Printf("WARNING: Project initialization failed: %v", err)
		// Consider if this should be fatal depending on the error
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
			Port         int     `toml:"port"`
			Entry        string  `toml:"entry"`
			Ip           string  `toml:"ip"`
			Mode         string  `toml:"mode"`
			RateLimitR   float64 `toml:"rate_limit_r"`
			RateLimitB   int     `toml:"rate_limit_b"`
			TrustedProxy string  `toml:"trusted_proxy"`
		}{Port: defaultPort, Ip: defaultIP, Mode: defaultMode, RateLimitR: defaultRateLimitR, RateLimitB: defaultRateLimitB, TrustedProxy: defaultTrustedProxy},
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
		log.Printf("WARN: Failed to read config file '%s': %v. Using default settings.", configFile, err)
	} else {
		err = toml.Unmarshal(configData, &config)
		if err != nil {
			log.Printf("WARN: Failed to parse config file '%s': %v. Using default/incomplete settings.", configFile, err)
			// Reset to defaults if parsing failed for specific fields
			if config.Server.Port == 0 {
				config.Server.Port = defaultPort
			}
			if config.Server.Ip == "" {
				config.Server.Ip = defaultIP
			}
			if config.Server.Mode == "" {
				config.Server.Mode = defaultMode
			}
			if config.Server.RateLimitR <= 0 {
				config.Server.RateLimitR = defaultRateLimitR
			}
			if config.Server.RateLimitB <= 0 {
				config.Server.RateLimitB = defaultRateLimitB
			}
			// TrustedProxy can be empty
			if config.Auth.JwtSecret == "" {
				config.Auth.JwtSecret = defaultJwtSecret
			}
			if config.Auth.TokenDurationMinutes <= 0 {
				config.Auth.TokenDurationMinutes = defaultTokenDuration
			}
			// Functions.Users defaults to false
		}
	}
	// Final checks/defaults after potential partial parse
	if config.Server.Ip == "" {
		config.Server.Ip = defaultIP
		log.Printf("WARN: Server IP not set, defaulting to %s", defaultIP)
	}
	if config.Server.Mode == "" {
		config.Server.Mode = defaultMode
		log.Printf("WARN: Server Mode not set, defaulting to %s", defaultMode)
	}
	if config.Server.RateLimitR <= 0 {
		config.Server.RateLimitR = defaultRateLimitR
		log.Printf("WARN: Invalid RateLimitR, defaulting to %.2f", defaultRateLimitR)
	}
	if config.Server.RateLimitB <= 0 {
		config.Server.RateLimitB = defaultRateLimitB
		log.Printf("WARN: Invalid RateLimitB, defaulting to %d", defaultRateLimitB)
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
	listenAddr := fmt.Sprintf("%s:%d", config.Server.Ip, config.Server.Port)
	// --- End Load Configuration ---

	// --- Initialize User Store ---
	userStore, err := storage.NewJSONUserStore(usersFile)
	if err != nil {
		log.Printf("CRITICAL: Failed to initialize user store from '%s': %v. User/Account functions will likely fail.", usersFile, err)
		userStore = nil
	}
	// --- End User Store Initialization ---

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

	// --- Setup Gin Router ---
	router := gin.New() // Use New instead of Default to have more control

	// --- Global Middleware ---
	// Custom Logger (using config)
	logConfig := gin.LoggerConfig{
		Formatter: logger.CustomLogFormatter,
		Output:    logWriter, // Use the multi-writer
	}
	router.Use(gin.LoggerWithConfig(logConfig)) // Use configured logger

	// Recovery Middleware (built-in)
	router.Use(gin.Recovery())

	// Rate Limiting
	rateLimiter := rate.NewLimiter(rate.Limit(config.Server.RateLimitR), config.Server.RateLimitB)
	router.Use(middleware.RateLimitMiddleware(rateLimiter)) // Use the exported middleware

	// Trusted Proxies
	if config.Server.TrustedProxy != "" {
		err := router.SetTrustedProxies([]string{config.Server.TrustedProxy})
		if err != nil {
			log.Printf("WARN: Failed to set trusted proxy '%s': %v", config.Server.TrustedProxy, err)
		}
	} else {
		_ = router.SetTrustedProxies(nil) // Explicitly clear trusted proxies
	}

	// --- End Global Middleware ---

	// --- Setup Handlers ---
	// Ensure userStore is usable before creating handlers that depend on it
	if userStore == nil {
		log.Fatalf("CRITICAL: UserStore is nil, cannot proceed with handler setup.")
		// Or implement a fallback mechanism / limited functionality mode
	}
	authHandler := v1handlers.NewAuthHandler(userStore, config.Auth.JwtSecret, config.Auth.TokenDurationMinutes)
	userHandler := v1handlers.NewUserHandler(userStore)
	accountHandler := v1handlers.NewAccountHandler(userStore) // Create AccountHandler
	// --- End Setup Handlers ---

	// --- Register API Routes ---
	api := router.Group("/api")
	{
		v1Group := api.Group("/v1")
		{
			v1.RegisterRoutes(
				v1Group,
				authHandler,
				userHandler,
				accountHandler, // Pass AccountHandler
				config.Auth.JwtSecret,
				config.Functions.Users, // Use config flag to allow/disallow registration
			)
		}
		// Register other API versions here (e.g., v2)
	}
	// --- End Register API Routes ---

	// --- Register Web Server Handlers (Static files, Templates, Error pages) ---
	// Create webserver config struct needed by RegisterHandlers
	webServerConfig := webserver.AppConfig{
		Server: struct {
			Entry string `toml:"entry"`
			Mode  string
		}{
			Entry: config.Server.Entry,
			Mode:  config.Server.Mode,
		},
	}
	webserver.RegisterHandlers(router, webServerConfig, uiSettingsData, uiSettingsFile) // Pass correct arguments
	// --- End Web Server Handlers ---

	// --- Debug Endpoint (if enabled) ---
	if debugModeActive {
		debug := router.Group("/debug")
		{
			debug.GET("/ping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "pong"})
			})
			// Add other debug routes as needed
		}
	}
	// --- End Debug Endpoint ---

	// --- Start Server ---
	// Build the startup log message
	startupMsg := fmt.Sprintf("Starting PanelBase | Mode: %s | Addr: %s",
		gin.Mode(),
		listenAddr,
	)
	if config.Server.Entry != "" {
		startupMsg += fmt.Sprintf(" | Entry: /%s/", config.Server.Entry)
	}
	log.Println(startupMsg) // Print the combined message

	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
