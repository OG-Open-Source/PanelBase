package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	v1 "github.com/OG-Open-Source/PanelBase/internal/api/v1"
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
	}
}

const (
	defaultPort = 8080
	defaultIP   = "0.0.0.0"
	configFile  = "configs/config.toml"
	baseWebDir  = "web"
)

// handleStaticFileRequest attempts to serve a static file from baseDir based on the requestedFile path,
// applying custom rules (deny .html/.htm, allow clean URLs).
func handleStaticFileRequest(c *gin.Context, baseDir string, requestedFile string) {
	fsRoot, _ := filepath.Abs(baseDir)

	// Remove leading/trailing slashes for consistency
	cleanedPath := strings.Trim(requestedFile, "/")

	// 1. Deny direct .html/.htm access in the original request path
	if strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".html") || strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".htm") {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 2. Handle root path (serve index.html or index.htm)
	if cleanedPath == "" || cleanedPath == "." {
		indexPathHtml := filepath.Join(fsRoot, "index.html")
		if _, err := os.Stat(indexPathHtml); err == nil {
			c.File(indexPathHtml)
			return
		}
		indexPathHtm := filepath.Join(fsRoot, "index.htm")
		if _, err := os.Stat(indexPathHtm); err == nil {
			c.File(indexPathHtm)
			return
		}
		// If no index file, fall through to 404
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 3. Try serving <cleanedPath>.html
	filePathHtml := filepath.Join(fsRoot, cleanedPath+".html")
	if _, err := os.Stat(filePathHtml); err == nil {
		c.File(filePathHtml)
		return
	}

	// 4. Try serving <cleanedPath>.htm
	filePathHtm := filepath.Join(fsRoot, cleanedPath+".htm")
	if _, err := os.Stat(filePathHtm); err == nil {
		c.File(filePathHtm)
		return
	}

	// 5. Try serving the file directly (for CSS, JS, images, etc.)
	directPath := filepath.Join(fsRoot, cleanedPath)
	if fi, err := os.Stat(directPath); err == nil && !fi.IsDir() {
		c.File(directPath)
		return
	}

	// 6. Not found
	c.AbortWithStatus(http.StatusNotFound)
}

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
		}{Port: defaultPort, Ip: defaultIP}, // Set defaults
	}
	configData, err := os.ReadFile(configFile)
	if err != nil {
		log.Printf("WARN: Failed to read config file '%s': %v. Using default server settings.", configFile, err)
	} else {
		err = toml.Unmarshal(configData, &config)
		if err != nil {
			log.Printf("WARN: Failed to parse config file '%s': %v. Using default server settings.", configFile, err)
			config.Server.Port = defaultPort // Reset to default on parse error
			config.Server.Ip = defaultIP
		}
	}
	if config.Server.Ip == "" {
		config.Server.Ip = defaultIP
		log.Printf("WARN: Server IP not set in config, defaulting to %s", defaultIP)
	}
	listenAddr := fmt.Sprintf("%s:%d", config.Server.Ip, config.Server.Port)
	// --- End Load Configuration ---

	// Create a gin router without default middleware
	router := gin.New()
	router.Use(gin.Recovery())
	logConfig := gin.LoggerConfig{
		Formatter: logger.CustomLogFormatter,
		Output:    logWriter,
	}
	router.Use(gin.LoggerWithConfig(logConfig))

	// --- Register Explicit Routes FIRST ---
	// API v1 group
	apiV1 := router.Group("/api/v1")
	v1.RegisterRoutes(apiV1)

	// Define a simple route for testing
	router.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	// --- End Explicit Routes ---

	// --- Setup Static File Serving / NoRoute Handler ---
	if config.Server.Entry != "" {
		// Serve entry-specific directory at /<entry>/
		entryWebDir := filepath.Join(baseWebDir, config.Server.Entry)
		entryRoutePath := "/" + config.Server.Entry
		log.Printf("Serving files from '%s' at URL path '%s/*' with custom handler", entryWebDir, entryRoutePath)
		router.GET(entryRoutePath+"/*filepath", func(c *gin.Context) {
			handleStaticFileRequest(c, entryWebDir, c.Param("filepath"))
		})
		log.Printf("Admin Entry: %s", entryRoutePath)

		// Handle NoRoute when entry exists (likely a real 404)
		router.NoRoute(func(c *gin.Context) {
			c.AbortWithStatus(http.StatusNotFound)
		})

	} else {
		// Handle NoRoute when entry is empty (try serving from baseWebDir)
		log.Printf("Serving files from base '%s' at root path '%s' via NoRoute handler", baseWebDir, "/")
		router.NoRoute(func(c *gin.Context) {
			handleStaticFileRequest(c, baseWebDir, c.Request.URL.Path)
		})
		log.Printf("Admin Entry: Disabled (server.entry is empty in config)")
	}
	// --- End Static File Serving ---

	// Start the server
	log.Printf("Starting server on %s", listenAddr)
	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
