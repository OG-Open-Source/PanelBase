package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
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
		Mode  string `toml:"mode"`
	}
}

const (
	defaultPort    = 8080
	defaultIP      = "0.0.0.0"
	configFile     = "configs/config.toml"
	uiSettingsFile = "configs/ui_settings.json"
	baseWebDir     = "web"
	defaultMode    = gin.ReleaseMode
)

// handleStaticFileRequest attempts to serve a static file from baseDir based on the requestedFile path,
// applying custom rules and rendering HTML templates with uiData.
func handleStaticFileRequest(c *gin.Context, baseDir string, requestedFile string, uiData map[string]interface{}) {
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
			c.HTML(http.StatusOK, "index.html", uiData) // Render template
			return
		}
		indexPathHtm := filepath.Join(fsRoot, "index.htm")
		if _, err := os.Stat(indexPathHtm); err == nil {
			c.HTML(http.StatusOK, "index.htm", uiData) // Render template
			return
		}
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// 3. Try serving <cleanedPath>.html
	templateNameHtml := cleanedPath + ".html"
	filePathHtml := filepath.Join(fsRoot, templateNameHtml)
	if _, err := os.Stat(filePathHtml); err == nil {
		c.HTML(http.StatusOK, templateNameHtml, uiData) // Render template
		return
	}

	// 4. Try serving <cleanedPath>.htm
	templateNameHtm := cleanedPath + ".htm"
	filePathHtm := filepath.Join(fsRoot, templateNameHtm)
	if _, err := os.Stat(filePathHtm); err == nil {
		c.HTML(http.StatusOK, templateNameHtm, uiData) // Render template
		return
	}

	// 5. Try serving the file directly (for CSS, JS, images, etc.)
	directPath := filepath.Join(fsRoot, cleanedPath)
	if fi, err := os.Stat(directPath); err == nil && !fi.IsDir() {
		c.File(directPath) // Serve non-template file directly
		return
	}

	// 6. Not found
	c.AbortWithStatus(http.StatusNotFound)
}

// loadTemplates manually finds and loads .html and .htm files from a directory.
func loadTemplates(router *gin.Engine, baseDir string) error {
	var templateFiles []string
	baseDirAbs, _ := filepath.Abs(baseDir)

	log.Printf("Scanning for templates (.html, .htm) in: %s", baseDirAbs)

	walkErr := filepath.WalkDir(baseDirAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("WARN: Error accessing path %q: %v\n", path, err)
			return err
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".html" || ext == ".htm" {
				templateFiles = append(templateFiles, path)
			}
		}
		return nil
	})

	if walkErr != nil {
		return fmt.Errorf("error walking template directory %s: %w", baseDirAbs, walkErr)
	}

	if len(templateFiles) == 0 {
		log.Printf("WARN: No template files (.html, .htm) found in %s", baseDirAbs)
		return nil
	}

	log.Printf("Loading %d template files...", len(templateFiles))
	router.LoadHTMLFiles(templateFiles...)

	return nil
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
			Mode  string `toml:"mode"`
		}{Port: defaultPort, Ip: defaultIP, Mode: defaultMode},
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
	listenAddr := fmt.Sprintf("%s:%d", config.Server.Ip, config.Server.Port)
	// --- End Load Configuration ---

	// --- Set Gin Mode BEFORE creating router ---
	ginMode := gin.DebugMode // Default for logic
	if strings.ToLower(config.Server.Mode) == gin.ReleaseMode {
		ginMode = gin.ReleaseMode
	}
	gin.SetMode(ginMode)
	// --- End Set Gin Mode ---

	// --- Load UI Settings AFTER setting mode, BEFORE router ---
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

	// Create a gin router
	router := gin.New()
	router.Use(gin.Recovery())
	logConfig := gin.LoggerConfig{
		Formatter: logger.CustomLogFormatter,
		Output:    logWriter,
	}
	router.Use(gin.LoggerWithConfig(logConfig))

	// --- Load HTML Templates ---
	templateBaseDir := baseWebDir
	if config.Server.Entry != "" {
		templateBaseDir = filepath.Join(baseWebDir, config.Server.Entry)
	}
	// Load templates manually
	if err := loadTemplates(router, templateBaseDir); err != nil {
		log.Printf("ERROR: Failed to load HTML templates: %v", err)
	}
	// --- End Load Templates ---

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
	adminEntryPath := "Disabled (server.entry is empty in config)"
	if config.Server.Entry != "" {
		// Serve entry-specific directory at /<entry>/
		entryWebDir := filepath.Join(baseWebDir, config.Server.Entry)
		entryRoutePath := "/" + config.Server.Entry
		router.GET(entryRoutePath+"/*filepath", func(c *gin.Context) {
			handleStaticFileRequest(c, entryWebDir, c.Param("filepath"), uiSettingsData)
		})
		adminEntryPath = entryRoutePath
	} else {
		// Handle NoRoute when entry is empty (try serving from baseWebDir)
		router.NoRoute(func(c *gin.Context) {
			handleStaticFileRequest(c, baseWebDir, c.Request.URL.Path, uiSettingsData)
		})
	}
	// --- End Static File Serving ---

	// Start the server with combined log message
	log.Printf("Starting server | Mode: %s | Addr: %s | Admin Entry: %s",
		ginMode,
		listenAddr,
		adminEntryPath,
	)
	if err := router.Run(listenAddr); err != nil {
		log.Fatalf("Failed to run server: %v", err)
	}
}
