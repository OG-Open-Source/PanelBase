package webserver

import (
	"encoding/json"
	"fmt"
	"html/template"
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

// handleStaticFileRequest attempts to serve a static file or render a template dynamically.
// Internal to the webserver package.
func handleStaticFileRequest(c *gin.Context, baseDir string, requestedFile string, uiData map[string]interface{}, config AppConfig) {
	fsRoot, _ := filepath.Abs(baseDir)
	cleanedPath := filepath.Clean(strings.TrimPrefix(requestedFile, "/")) // Clean the path
	requestedPathAbs := filepath.Join(fsRoot, cleanedPath)

	// Prevent directory traversal attacks
	if !strings.HasPrefix(requestedPathAbs, fsRoot) {
		log.Printf("WARN: Directory traversal attempt blocked: %s (requested %s)", requestedPathAbs, requestedFile)
		handleErrorResponse(c, http.StatusNotFound, config, uiData)
		return
	}

	// Prevent direct access to the templates directory
	if strings.HasPrefix(cleanedPath, "templates/") || cleanedPath == "templates" {
		log.Printf("INFO: Direct access to templates directory blocked: %s", requestedFile)
		handleErrorResponse(c, http.StatusNotFound, config, uiData)
		return
	}

	// Deny direct .html/.htm access in the URL path itself - This check might be redundant now but kept for clarity
	if strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".html") || strings.HasSuffix(strings.ToLower(c.Request.URL.Path), ".htm") {
		log.Printf("INFO: Direct access to .html/.htm via URL blocked: %s", c.Request.URL.Path)
		handleErrorResponse(c, http.StatusNotFound, config, uiData)
		return
	}

	// --- Dynamic Template Rendering ---

	// Determine the template file base name (e.g., "index" for "/", "market" for "/market")
	templateBaseName := cleanedPath
	if cleanedPath == "" || cleanedPath == "." || requestedFile == "/" {
		templateBaseName = "index" // Special case for root requests
	}

	// Check for .html first
	templatePathHtml := filepath.Join(fsRoot, templateBaseName+".html")
	if _, err := os.Stat(templatePathHtml); err == nil {
		renderDynamicTemplate(c, templatePathHtml, uiData, config)
		return
	}

	// Check for .htm if .html not found
	templatePathHtm := filepath.Join(fsRoot, templateBaseName+".htm")
	if _, err := os.Stat(templatePathHtm); err == nil {
		renderDynamicTemplate(c, templatePathHtm, uiData, config)
		return
	}

	// --- Static File Serving (Fallback) ---

	// If no matching template found, try serving as a direct static file (CSS, JS, images, etc.)
	directPath := requestedPathAbs // Use the absolute path we calculated earlier
	if fi, err := os.Stat(directPath); err == nil && !fi.IsDir() {
		// Use http.ServeFile for correct content type and caching headers
		http.ServeFile(c.Writer, c.Request, directPath)
		return
	}

	// --- Not Found ---

	// If neither a template nor a static file is found, return 404
	// log.Printf("INFO: File or template not found for request: %s (checked %s, %s, %s)",
	// 	requestedFile, templatePathHtml, templatePathHtm, directPath)
	handleErrorResponse(c, http.StatusNotFound, config, uiData)
}

// renderDynamicTemplate parses and executes a single template file.
func renderDynamicTemplate(c *gin.Context, templatePath string, uiData map[string]interface{}, config AppConfig) {
	// log.Printf("Rendering template: %s", templatePath) // Remove rendering log
	templateContent, err := os.ReadFile(templatePath)
	if err != nil {
		log.Printf("ERROR: Failed to read template file '%s': %v", templatePath, err)
		handleErrorResponse(c, http.StatusInternalServerError, config, uiData)
		return
	}

	// Use the file name as the template name for uniqueness
	templateName := filepath.Base(templatePath)
	tmpl, err := template.New(templateName).Parse(string(templateContent))
	if err != nil {
		log.Printf("ERROR: Failed to parse template '%s': %v", templatePath, err)
		handleErrorResponse(c, http.StatusInternalServerError, config, uiData)
		return
	}

	c.Header("Content-Type", "text/html; charset=utf-8")
	err = tmpl.Execute(c.Writer, uiData)
	if err != nil {
		log.Printf("ERROR: Failed to execute template '%s': %v", templatePath, err)
		// Don't call handleErrorResponse here to avoid infinite loop if error template fails
		c.String(http.StatusInternalServerError, "%d: %s", http.StatusInternalServerError, httpStatusMessages[http.StatusInternalServerError])
	}
}

// loadTemplates is deprecated as templates are loaded dynamically.
/*
func loadTemplates(router *gin.Engine, baseDir string) error {
	// ... old logic ...
}
*/

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

// robotsHandler generates and serves the robots.txt file.
func robotsHandler(config AppConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		var content strings.Builder
		content.WriteString("User-agent: *\n")

		if config.Server.Entry != "" {
			// If entry is set, disallow crawling the entire site starting from root.
			// This effectively hides the entry path without revealing it.
			content.WriteString("Disallow: /\n")
		} else {
			// If entry is not set, disallow common sensitive paths
			content.WriteString("Disallow: /api/\n")
			if strings.ToLower(config.Server.Mode) == gin.DebugMode {
				content.WriteString("Disallow: /debug/\n")
			}
			// Add other Disallow rules if needed
		}

		c.Header("Content-Type", "text/plain; charset=utf-8")
		c.String(http.StatusOK, content.String())
	}
}

// RegisterHandlers sets up all web-related routes and handlers.
func RegisterHandlers(router *gin.Engine, config AppConfig, uiData map[string]interface{}, uiSettingsFile string) {
	baseDir := baseWebDir
	entryPath := "/"
	if config.Server.Entry != "" {
		entryDirAbs := filepath.Join(baseDir, config.Server.Entry)
		if _, err := os.Stat(entryDirAbs); err == nil {
			baseDir = entryDirAbs
			entryPath = "/" + config.Server.Entry + "/"
		} else {
			log.Printf("WARN: Entry directory '%s' not found, serving from base '%s' at root path.", entryDirAbs, baseWebDir)
			baseDir = baseWebDir // Fallback to base web dir
		}
	}

	// Serve files from the determined base directory
	// fsRoot, _ := filepath.Abs(baseDir) // fsRoot is not directly used by NoRoute handler
	// fileServer := http.FileServer(http.Dir(fsRoot)) // fileServer is not used directly

	// Static file serving needs careful handling due to custom rules
	// For paths starting with the entry path (or root if no entry)
	router.NoRoute(func(c *gin.Context) {
		// Check if the request path matches the expected entry path or root
		pathMatchesEntry := false
		if config.Server.Entry != "" && strings.HasPrefix(c.Request.URL.Path, entryPath) {
			pathMatchesEntry = true
		} else if config.Server.Entry == "" && (c.Request.URL.Path == "/" || !strings.HasPrefix(c.Request.URL.Path, "/api/")) {
			pathMatchesEntry = true
		}

		if pathMatchesEntry {
			// Extract the relative file path within the entry/web directory
			relativeRequestPath := c.Request.URL.Path
			if config.Server.Entry != "" {
				relativeRequestPath = strings.TrimPrefix(relativeRequestPath, entryPath)
			}
			// Ensure it's always relative
			relativeRequestPath = strings.TrimPrefix(relativeRequestPath, "/")

			handleStaticFileRequest(c, baseDir, relativeRequestPath, uiData, config)
		} else {
			// If the path doesn't match the entry/root structure or is an API path not handled elsewhere
			handleErrorResponse(c, http.StatusNotFound, config, uiData)
		}
	})

	// Explicitly handle GET requests for the entry path root (e.g., /entry/)
	if config.Server.Entry != "" {
		router.GET(entryPath, func(c *gin.Context) {
			handleStaticFileRequest(c, baseDir, "index", uiData, config) // Serve index template
		})
	}

	// Register other specific web handlers if needed (e.g., /robots.txt)
	router.GET("/robots.txt", robotsHandler(config))

	// Debug Info Handler (already handled in main.go based on mode)
	// if config.Server.Mode == gin.DebugMode {
	// 	router.GET("/debug/info", debugInfoHandler(config))
	// }

	// log.Println("Web server handlers registered.") // Removed registration log
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
