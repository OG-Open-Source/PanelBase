package webserver

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/pelletier/go-toml/v2" // Required for AppConfig definition

	// Import runtime for debug info (Placeholder - might need more specific packages later)
	"runtime"
)

// --- Configuration (Copied from main.go, consider refactoring later) ---

// AppConfig holds application configuration read from config.toml
// TODO: Consider passing only necessary fields instead of the whole struct
type AppConfig struct {
	Server struct {
		Entry string `toml:"entry"`
		Mode  string // Added Mode field
		// Add other fields if needed by webserver logic (e.g., Port, IP for logging?)
	}
	// Add other sections like Auth or Functions if directly needed here
}

const (
	baseWebDir = "web" // Define constants used by the moved functions
)

// --- End Configuration ---

// httpStatusMessages holds the standard reason phrases for HTTP status codes.
var httpStatusMessages = map[int]string{
	http.StatusContinue:                     "Continue",
	http.StatusSwitchingProtocols:           "Switching Protocols",
	http.StatusOK:                           "OK",
	http.StatusCreated:                      "Created",
	http.StatusAccepted:                     "Accepted",
	http.StatusNonAuthoritativeInfo:         "Non-Authoritative Information",
	http.StatusNoContent:                    "No Content",
	http.StatusResetContent:                 "Reset Content",
	http.StatusPartialContent:               "Partial Content",
	http.StatusMultipleChoices:              "Multiple Choices",
	http.StatusMovedPermanently:             "Moved Permanently",
	http.StatusFound:                        "Found",
	http.StatusSeeOther:                     "See Other",
	http.StatusNotModified:                  "Not Modified",
	http.StatusUseProxy:                     "Use Proxy",
	http.StatusTemporaryRedirect:            "Temporary Redirect",
	http.StatusBadRequest:                   "Bad Request",
	http.StatusUnauthorized:                 "Unauthorized",
	http.StatusPaymentRequired:              "Payment Required",
	http.StatusForbidden:                    "Forbidden",
	http.StatusNotFound:                     "Not Found",
	http.StatusMethodNotAllowed:             "Method Not Allowed",
	http.StatusNotAcceptable:                "Not Acceptable",
	http.StatusProxyAuthRequired:            "Proxy Authentication Required",
	http.StatusRequestTimeout:               "Request Timeout",
	http.StatusConflict:                     "Conflict",
	http.StatusGone:                         "Gone",
	http.StatusLengthRequired:               "Length Required",
	http.StatusPreconditionFailed:           "Precondition Failed",
	http.StatusRequestEntityTooLarge:        "Payload Too Large",
	http.StatusRequestURITooLong:            "URI Too Long",
	http.StatusUnsupportedMediaType:         "Unsupported Media Type",
	http.StatusRequestedRangeNotSatisfiable: "Range Not Satisfiable",
	http.StatusExpectationFailed:            "Expectation Failed",
	http.StatusUpgradeRequired:              "Upgrade Required",
	http.StatusInternalServerError:          "Internal Server Error",
	http.StatusNotImplemented:               "Not Implemented",
	http.StatusBadGateway:                   "Bad Gateway",
	http.StatusServiceUnavailable:           "Service Unavailable",
	http.StatusGatewayTimeout:               "Gateway Timeout",
	http.StatusHTTPVersionNotSupported:      "HTTP Version Not Supported",
}

// handleErrorResponse handles serving custom error pages based on HTTP status code.
// This function is now internal to the webserver package.
func handleErrorResponse(c *gin.Context, statusCode int, config AppConfig, uiData map[string]interface{}) {
	webRoot := baseWebDir
	entryDir := ""
	if config.Server.Entry != "" {
		entryDir = filepath.Join(baseWebDir, config.Server.Entry)
		if _, err := os.Stat(entryDir); err == nil {
			webRoot = entryDir
		} else {
			log.Printf("WARN: Entry directory '%s' not found, falling back to '%s' for error pages.", entryDir, baseWebDir)
		}
	}

	reasonPhrase := httpStatusMessages[statusCode]
	if reasonPhrase == "" {
		reasonPhrase = "Unknown Status"
	}

	// Priority 1: Check for /web/<entry>/templates/<status_code>.html(.htm)
	templateDir := filepath.Join(webRoot, "templates")
	specificTemplateBase := filepath.Join(templateDir, strconv.Itoa(statusCode))
	specificTemplatePathHtml := specificTemplateBase + ".html"
	specificTemplatePathHtm := specificTemplateBase + ".htm"

	var templateContent []byte
	var err error
	var isSpecificTemplate bool

	if _, statErr := os.Stat(specificTemplatePathHtml); statErr == nil {
		templateContent, err = os.ReadFile(specificTemplatePathHtml)
		isSpecificTemplate = true
	} else if _, statErr = os.Stat(specificTemplatePathHtm); statErr == nil {
		templateContent, err = os.ReadFile(specificTemplatePathHtm)
		isSpecificTemplate = true
	}

	if isSpecificTemplate && err == nil {
		// log.Printf("Serving specific error template for %d from %s", statusCode, webRoot) // Commented out
		tmpl, parseErr := template.New(filepath.Base(specificTemplatePathHtml)).Parse(string(templateContent)) // Use unique name based on path
		if parseErr != nil {
			log.Printf("ERROR: Failed to parse specific error template for %d ('%s'): %v", statusCode, specificTemplatePathHtml, parseErr)
		} else {
			c.Status(statusCode)
			c.Header("Content-Type", "text/html; charset=utf-8")
			execErr := tmpl.Execute(c.Writer, uiData)
			if execErr != nil {
				log.Printf("ERROR: Failed to execute specific error template for %d: %v", statusCode, execErr)
				c.String(http.StatusInternalServerError, "%d: %s", http.StatusInternalServerError, httpStatusMessages[http.StatusInternalServerError])
			}
			c.Abort()
			return
		}
	} else if isSpecificTemplate && err != nil {
		log.Printf("ERROR: Failed to read specific error template file for %d ('%s'): %v", statusCode, specificTemplatePathHtml, err)
	}

	// Priority 2: Check for /web/<entry>/error.html(.htm)
	generalErrorPathHtml := filepath.Join(webRoot, "error.html")
	generalErrorPathHtm := filepath.Join(webRoot, "error.htm")
	var isGeneralTemplate bool

	// Reset templateContent and err for reading general template
	templateContent = nil
	err = nil

	if _, statErr := os.Stat(generalErrorPathHtml); statErr == nil {
		templateContent, err = os.ReadFile(generalErrorPathHtml)
		isGeneralTemplate = true
	} else if _, statErr = os.Stat(generalErrorPathHtm); statErr == nil {
		templateContent, err = os.ReadFile(generalErrorPathHtm)
		isGeneralTemplate = true
	}

	if isGeneralTemplate && err == nil {
		// log.Printf("Serving general error template from %s for status %d", webRoot, statusCode) // Commented out
		tmpl, parseErr := template.New("error.html").Parse(string(templateContent))
		if parseErr != nil {
			log.Printf("ERROR: Failed to parse general error template ('%s'): %v", generalErrorPathHtml, parseErr)
		} else {
			errorData := gin.H{
				"http_status_code":    statusCode,
				"http_status_message": reasonPhrase,
			}
			for k, v := range uiData {
				if _, exists := errorData[k]; !exists {
					errorData[k] = v
				}
			}

			c.Status(statusCode)
			c.Header("Content-Type", "text/html; charset=utf-8")
			execErr := tmpl.Execute(c.Writer, errorData)
			if execErr != nil {
				log.Printf("ERROR: Failed to execute general error template: %v", execErr)
				c.String(http.StatusInternalServerError, "%d: %s", http.StatusInternalServerError, httpStatusMessages[http.StatusInternalServerError])
			}
			c.Abort()
			return
		}
	} else if isGeneralTemplate && err != nil {
		log.Printf("ERROR: Failed to read general error template file ('%s'): %v", generalErrorPathHtml, err)
	}

	// Priority 3: Plain text response
	// log.Printf("Serving plain text error for status %d", statusCode) // Commented out as requested
	// Ensure content type is text/plain for fallback
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(statusCode, "%d: %s", statusCode, reasonPhrase)
	c.Abort()
}

// handleStaticFileRequest attempts to serve a static file or render a template.
// Internal to the webserver package.
func handleStaticFileRequest(c *gin.Context, baseDir string, requestedFile string, uiData map[string]interface{}, config AppConfig) {
	fsRoot, _ := filepath.Abs(baseDir)
	cleanedPath := strings.Trim(requestedFile, "/")

	// Deny direct .html/.htm access in the URL path itself
	if strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".html") || strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".htm") {
		// Delegate to NoRoute handler, which will call handleErrorResponse
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Handle root path (serve index.html or index.htm)
	if cleanedPath == "" || cleanedPath == "." {
		// Use router's built-in template rendering
		if _, err := os.Stat(filepath.Join(fsRoot, "index.html")); err == nil {
			c.HTML(http.StatusOK, "index.html", uiData)
			return
		}
		if _, err := os.Stat(filepath.Join(fsRoot, "index.htm")); err == nil {
			c.HTML(http.StatusOK, "index.htm", uiData)
			return
		}
		// If no index file, delegate to NoRoute
		c.AbortWithStatus(http.StatusNotFound)
		return
	}

	// Try serving <cleanedPath>.html as a template
	templateNameHtml := cleanedPath + ".html"
	filePathHtml := filepath.Join(fsRoot, templateNameHtml)
	if _, err := os.Stat(filePathHtml); err == nil {
		c.HTML(http.StatusOK, templateNameHtml, uiData)
		return
	}

	// Try serving <cleanedPath>.htm as a template
	templateNameHtm := cleanedPath + ".htm"
	filePathHtm := filepath.Join(fsRoot, templateNameHtm)
	if _, err := os.Stat(filePathHtm); err == nil {
		c.HTML(http.StatusOK, templateNameHtm, uiData)
		return
	}

	// Try serving the file directly (CSS, JS, images, etc.)
	directPath := filepath.Join(fsRoot, cleanedPath)
	if fi, err := os.Stat(directPath); err == nil && !fi.IsDir() {
		// Use http.ServeFile for correct content type and caching headers
		http.ServeFile(c.Writer, c.Request, directPath)
		return
	}

	// Not found - call handleErrorResponse directly
	handleErrorResponse(c, http.StatusNotFound, config, uiData)
}

// loadTemplates manually finds and loads .html and .htm files from a directory.
// Internal to the webserver package.
func loadTemplates(router *gin.Engine, baseDir string) error {
	var templateFiles []string
	baseDirAbs, _ := filepath.Abs(baseDir)

	// log.Printf("Scanning for templates (.html, .htm) in: %s", baseDirAbs) // Simplified

	walkErr := filepath.WalkDir(baseDirAbs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("WARN: Error accessing path %q during template scan: %v\n", path, err)
			return fs.SkipDir
		}
		if !d.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".html" || ext == ".htm" {
				// log.Printf("Found template: %s", path) // Simplified
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

	log.Printf("Loading %d template(s) from %s", len(templateFiles), baseDirAbs) // Keep this summary log
	router.LoadHTMLFiles(templateFiles...)                                       // LoadHTMLFiles handles parsing

	return nil
}

// debugInfoHandler provides debug information about the server.
// Accessible only when server.mode is "debug".
func debugInfoHandler(config AppConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Basic runtime stats
		var m runtime.MemStats
		runtime.ReadMemStats(&m)

		debugData := gin.H{
			"server_config": config.Server, // Show server part of config
			"runtime_stats": gin.H{
				"num_goroutine": runtime.NumGoroutine(),
				"memory_alloc":  m.Alloc,
				"memory_total":  m.TotalAlloc,
				"memory_sys":    m.Sys,
				"num_gc":        m.NumGC,
			},
			// TODO: Add more debug info as needed (e.g., full config, loaded routes, user store status)
		}
		c.JSON(http.StatusOK, debugData)
	}
}

// RegisterHandlers sets up the routes and handlers for serving web content.
func RegisterHandlers(router *gin.Engine, config AppConfig, uiData map[string]interface{}, uiSettingsFile string) {
	webRoot := baseWebDir
	entryDir := ""
	entryRoute := ""

	if config.Server.Entry != "" {
		checkEntryDir := filepath.Join(baseWebDir, config.Server.Entry)
		if _, err := os.Stat(checkEntryDir); err == nil {
			entryDir = checkEntryDir
			webRoot = entryDir // Override webRoot if entry exists
			entryRoute = "/" + config.Server.Entry
			log.Printf("INFO: Using entry directory: %s", entryDir)
		} else {
			log.Printf("WARN: Entry directory '%s' specified but not found. Static files served from '%s'.", checkEntryDir, baseWebDir)
			// Keep webRoot as baseWebDir, entryRoute remains empty
		}
	}

	// Add Debug route if in debug mode
	if strings.ToLower(config.Server.Mode) == gin.DebugMode {
		log.Println("INFO: Registering /debug API endpoint.")
		router.GET("/debug", debugInfoHandler(config)) // Pass config to the handler
	}

	// Load templates from the determined web root
	if err := loadTemplates(router, webRoot); err != nil {
		log.Printf("WARN: Failed to load templates from '%s': %v", webRoot, err)
	}

	// Setup NoRoute handler to manage 404s and custom error pages
	router.NoRoute(func(c *gin.Context) {
		handleErrorResponse(c, http.StatusNotFound, config, uiData)
	})

	// --- Static File Serving --- Priority: Entry Dir -> Base Dir

	// 1. Serve from Entry Directory (if configured and exists)
	if entryDir != "" {
		// Serve index/specific templates from the entry directory
		router.GET(entryRoute+"/*filepath", func(c *gin.Context) {
			requestedFile := c.Param("filepath")
			handleStaticFileRequest(c, entryDir, requestedFile, uiData, config)
		})
		// Handle root of the entry directory
		router.GET(entryRoute, func(c *gin.Context) {
			handleStaticFileRequest(c, entryDir, "/", uiData, config)
		})
		// If entryDir is set, DO NOT register base dir routes to avoid conflict.
	} else {
		// 2. Serve from Base Directory (web/) ONLY if entryDir is NOT set
		log.Printf("INFO: Serving static files from base directory: %s", baseWebDir)
		// Handle root path ("/")
		router.GET("/", func(c *gin.Context) {
			handleStaticFileRequest(c, baseWebDir, "/", uiData, config)
		})

		// Serve other files from the base directory
		router.GET("/*filepath", func(c *gin.Context) {
			requestedFile := c.Param("filepath")
			handleStaticFileRequest(c, baseWebDir, requestedFile, uiData, config)
		})
	}

	log.Println("Web server handlers registered.")
}

// LoadAppConfig is needed if AppConfig definition stays here and is used by RegisterHandlers
// Alternatively, pass individual config values or define a specific struct.
// For now, we assume AppConfig is defined/passed from main.
func LoadAppConfig(filePath string) (AppConfig, error) {
	config := AppConfig{
		// Set defaults if needed, or rely on main's defaults
	}
	configData, err := os.ReadFile(filePath)
	if err != nil {
		return config, fmt.Errorf("failed to read config file '%s': %w", filePath, err)
	}
	err = toml.Unmarshal(configData, &config)
	if err != nil {
		return config, fmt.Errorf("failed to parse config file '%s': %w", filePath, err)
	}
	return config, nil
}

// LoadUISettings is needed if uiData loading moves here
func LoadUISettings(filePath string) (map[string]interface{}, error) {
	uiSettingsData := make(map[string]interface{})
	uiSettingsBytes, err := os.ReadFile(filePath)
	if err != nil {
		// It might be acceptable for this file to be missing
		log.Printf("WARN: Failed to read UI settings file '%s': %v. UI data will be empty.", filePath, err)
		return uiSettingsData, nil // Return empty map, not an error? Or return err? Let's return nil err for now.
	}
	err = json.Unmarshal(uiSettingsBytes, &uiSettingsData)
	if err != nil {
		log.Printf("WARN: Failed to parse UI settings file '%s': %v. UI data will be empty.", filePath, err)
		// Return empty map on parse error as well
		return make(map[string]interface{}), nil // Return empty map
	}
	return uiSettingsData, nil
}
