package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	v1 "github.com/OG-Open-Source/PanelBase/internal/api/v1"
	v1handlers "github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers"
	"github.com/OG-Open-Source/PanelBase/internal/storage"
	"github.com/OG-Open-Source/PanelBase/internal/webserver"
	"github.com/OG-Open-Source/PanelBase/pkg/bootstrap"
	"github.com/OG-Open-Source/PanelBase/pkg/logger"
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	// "golang.org/x/time/rate" // No longer needed
)

// AppConfig holds application configuration read from config.yaml
type AppConfig struct {
	Server struct {
		Port         int    `yaml:"port"`
		Entry        string `yaml:"entry"`
		Ip           string `yaml:"ip"`
		Mode         string `yaml:"mode"`
		TrustedProxy string `yaml:"trusted_proxy"`
	}
	Auth struct {
		JwtSecret    string `yaml:"jwt_secret"`
		TokenMinutes int    `yaml:"token_minutes"`
		Defaults     struct {
			Scopes map[string]interface{} `yaml:"scopes"`
		} `yaml:"defaults"`
		Rules struct {
			RequireOldPw    bool     `yaml:"require_old_pw"`
			AllowSelfDelete bool     `yaml:"allow_self_delete"`
			ProtectedUsers  []string `yaml:"protected_users"`
		} `yaml:"rules"`
	}
	Features struct {
		Plugins  bool `yaml:"plugins"`
		Commands bool `yaml:"commands"`
		Users    bool `yaml:"users"`
		Themes   bool `yaml:"themes"`
	} `yaml:"features"`
}

const (
	defaultPort         = 8080
	defaultIP           = "0.0.0.0"
	configFile          = "configs/config.yaml"
	usersFile           = "configs/users.json"
	uiSettingsFile      = "configs/ui_settings.json"
	defaultMode         = gin.ReleaseMode
	defaultJwtSecret    = "change-this-in-config-yaml" // Default secret (should be changed)
	defaultTokenMinutes = 60                           // Default duration in minutes
	defaultTrustedProxy = ""                           // Default: No trusted proxy
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
			Port         int    `yaml:"port"`
			Entry        string `yaml:"entry"`
			Ip           string `yaml:"ip"`
			Mode         string `yaml:"mode"`
			TrustedProxy string `yaml:"trusted_proxy"`
		}{Port: defaultPort, Ip: defaultIP, Mode: defaultMode, TrustedProxy: defaultTrustedProxy},
		Auth: struct {
			JwtSecret    string `yaml:"jwt_secret"`
			TokenMinutes int    `yaml:"token_minutes"`
			Defaults     struct {
				Scopes map[string]interface{} `yaml:"scopes"`
			} `yaml:"defaults"`
			Rules struct {
				RequireOldPw    bool     `yaml:"require_old_pw"`
				AllowSelfDelete bool     `yaml:"allow_self_delete"`
				ProtectedUsers  []string `yaml:"protected_users"`
			} `yaml:"rules"`
		}{JwtSecret: defaultJwtSecret, TokenMinutes: defaultTokenMinutes},
		Features: struct {
			Plugins  bool `yaml:"plugins"`
			Commands bool `yaml:"commands"`
			Users    bool `yaml:"users"`
			Themes   bool `yaml:"themes"`
		}{Users: false},
	}
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("WARN: Failed to read config file '%s': %v. Using default settings.", configFile, err)
	} else {
		err = yaml.Unmarshal(configData, &config)
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
			if config.Auth.JwtSecret == "" {
				config.Auth.JwtSecret = defaultJwtSecret
			}
			if config.Auth.TokenMinutes <= 0 {
				config.Auth.TokenMinutes = defaultTokenMinutes
			}
			// Features.Users defaults to false
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
	if config.Auth.JwtSecret == "" || config.Auth.JwtSecret == defaultJwtSecret {
		config.Auth.JwtSecret = defaultJwtSecret
		log.Printf("WARN: Using default JWT secret. Change 'jwt_secret' in %s for production!", configFile)
	}
	if config.Auth.TokenMinutes <= 0 {
		config.Auth.TokenMinutes = defaultTokenMinutes
		log.Printf("WARN: Invalid token duration, defaulting to %d minutes", defaultTokenMinutes)
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
	router := gin.New()

	// --- Global Middleware ---
	// Custom Logger (using config)
	logConfig := gin.LoggerConfig{
		Formatter: logger.CustomLogFormatter,
		Output:    logWriter, // Use the multi-writer
	}
	router.Use(gin.LoggerWithConfig(logConfig))

	// Recovery Middleware (built-in)
	router.Use(gin.Recovery())

	// Rate Limiting (REMOVED)
	// rateLimiter := rate.NewLimiter(rate.Limit(config.Server.RateLimitR), config.Server.RateLimitB)
	// router.Use(middleware.RateLimitMiddleware(rateLimiter))

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
	authHandler := v1handlers.NewAuthHandler(userStore, config.Auth.JwtSecret, config.Auth.TokenMinutes, config.Auth.Defaults.Scopes)
	userHandler := v1handlers.NewUserHandler(userStore, config.Auth.Defaults.Scopes, nil)
	accountHandler := v1handlers.NewAccountHandler(userStore, nil) // Create AccountHandler
	// --- End Setup Handlers ---

	// --- Register API Routes (Dynamic Prefix) ---
	apiPrefix := "/api"
	if config.Server.Entry != "" {
		var err error
		apiPrefix, err = url.JoinPath("/", config.Server.Entry, "api")
		if err != nil {
			log.Fatalf("CRITICAL: Failed to construct API path prefix: %v", err)
		}
	}

	apiGroup := router.Group(apiPrefix)
	{
		v1Group := apiGroup.Group("/v1") // v1 is relative to the dynamic apiGroup
		{
			v1.RegisterRoutes(
				v1Group,
				authHandler,
				userHandler,
				accountHandler,
				config.Auth.JwtSecret,
				config.Features.Users,
			)
		}
		// Register other API versions relative to apiGroup here...
	}
	// --- End Register API Routes ---

	// --- Register Web Server Handlers ---
	// Pass the determined apiPrefix to RegisterHandlers
	webServerConfig := webserver.AppConfig{
		Server: struct {
			Entry string `yaml:"entry"`
			Mode  string
		}{
			Entry: config.Server.Entry,
			Mode:  config.Server.Mode,
		},
	}
	webserver.RegisterHandlers(router, webServerConfig, uiSettingsData, uiSettingsFile, apiPrefix)
	// --- End Web Server Handlers ---

	// --- Debug Endpoint (Remains at root) ---
	if debugModeActive { // Use the existing debugModeActive flag
		debug := router.Group("/debug") // Register under root router
		{
			debug.GET("/ping", func(c *gin.Context) {
				c.JSON(200, gin.H{"message": "pong"})
			})
			// Add other debug routes as needed
		}
	}
	// --- End Debug Endpoint ---

	// --- Start Server with Graceful Shutdown ---
	srv := &http.Server{
		Addr:    listenAddr,
		Handler: router,
	}

	// Initial startup message
	startupMsg := fmt.Sprintf("Starting PanelBase | Mode: %s | Addr: %s", gin.Mode(), listenAddr)
	if config.Server.Entry != "" {
		startupMsg += fmt.Sprintf(" | Entry: /%s/", config.Server.Entry)
	}
	log.Println(startupMsg)

	// Start server in a goroutine so that it doesn't block.
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with a timeout.
	quit := make(chan os.Signal, 1) // Buffer of 1
	// kill (no param) default send syscall.SIGTERM
	// kill -2 is syscall.SIGINT
	// kill -9 is syscall.SIGKILL but can't be caught, so don't need to add it
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // Block until a signal is received
	log.Println("Shutting down server...")

	// The context is used to inform the server it has 5 seconds to finish
	// the requests it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown: ", err)
	}

	log.Println("Server exiting")
}
