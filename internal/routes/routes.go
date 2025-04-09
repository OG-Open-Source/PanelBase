package routes

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	// Remove old direct handler imports if they are only used by v1 routes
	// "github.com/OG-Open-Source/PanelBase/internal/api_token"
	// "github.com/OG-Open-Source/PanelBase/internal/auth"
	// Keep if used elsewhere
	v1 "github.com/OG-Open-Source/PanelBase/internal/api/v1" // Import the new v1 routes package
	"github.com/OG-Open-Source/PanelBase/pkg/uisettings"     // UPDATED PATH // Keep for serveHTMLTemplate
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the main application routes.
func SetupRoutes(router *gin.Engine) {
	// Setup API v1 routes
	apiV1Group := router.Group("/api/v1")
	v1.SetupV1Routes(apiV1Group) // Call the new setup function

	// API v2 routes (Placeholder - to be added later)
	// apiV2Group := router.Group("/api/v2")
	// v2.SetupV2Routes(apiV2Group)

	// Serve static files specifically from the web/assets directory
	router.Static("/assets", "./web/assets")

	// Handle all other requests (potential frontend pages or 404s)
	router.NoRoute(func(c *gin.Context) {
		// Skip if it's an API call (already handled or should be 404)
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.JSON(http.StatusNotFound, gin.H{"error": "API route not found"})
			return
		}

		// Determine the target file path based on the request URL
		requestedPath := c.Request.URL.Path
		filePath := filepath.Join("web", requestedPath)

		// Handle root path specifically -> serve index.html
		if requestedPath == "/" || requestedPath == "/index.html" {
			filePath = filepath.Join("web", "index.html")
		} else {
			// Clean the path to prevent directory traversal
			filePath = filepath.Clean(filePath)
			// Ensure the path still starts with "web/" after cleaning
			if !strings.HasPrefix(filePath, "web"+string(filepath.Separator)) {
				c.String(http.StatusBadRequest, "Invalid path")
				return
			}
		}

		// Check if the requested file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			// If file doesn't exist, try serving index.html for client-side routing
			// (This assumes a single-page application structure)
			indexPath := filepath.Join("web", "index.html")
			if _, indexErr := os.Stat(indexPath); indexErr == nil {
				serveHTMLTemplate(c, indexPath)
			} else {
				c.String(http.StatusNotFound, "Resource not found")
			}
			return
		}

		// Check if it's an HTML file that needs template rendering
		if strings.HasSuffix(strings.ToLower(filePath), ".html") || strings.HasSuffix(strings.ToLower(filePath), ".htm") {
			serveHTMLTemplate(c, filePath)
		} else {
			// Serve other static files directly
			c.File(filePath)
		}
	})
}

// serveHTMLTemplate loads UI settings, parses and executes an HTML template
// This function remains here as it's not API specific
func serveHTMLTemplate(c *gin.Context, templatePath string) {
	// Get UI settings
	uiSettingsData, err := uisettings.GetUISettings() // UPDATED package name
	if err != nil {
		log.Printf("%s Error getting UI settings for template %s: %v", time.Now().UTC().Format(time.RFC3339), templatePath, err)
		c.String(http.StatusInternalServerError, "Error loading UI settings")
		return
	}

	// Parse template
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("%s Error parsing template %s: %v", time.Now().UTC().Format(time.RFC3339), templatePath, err)
		c.String(http.StatusInternalServerError, "Error parsing template")
		return
	}

	// Execute template
	if err := tmpl.Execute(c.Writer, uiSettingsData); err != nil { // Use uiSettingsData
		log.Printf("%s Error executing template %s: %v", time.Now().UTC().Format(time.RFC3339), templatePath, err)
		c.String(http.StatusInternalServerError, "Error executing template")
		return
	}
}
