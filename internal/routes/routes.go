package routes

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/api_token"
	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/ui_settings"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the main application routes.
func SetupRoutes(router *gin.Engine) {
	// API v1 routes
	apiV1 := router.Group("/api/v1")
	{
		// Authentication routes (public)
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/login", auth.LoginHandler())
			authGroup.POST("/register", auth.RegisterHandler)
		}

		// Protected API routes
		protectedGroup := apiV1.Group("")
		protectedGroup.Use(middleware.AuthMiddleware())
		{
			// Token Refresh Route
			protectedGroup.POST("/auth/token", auth.RefreshTokenHandler())

			// Account Management (Self)
			account := protectedGroup.Group("/account")
			{
				// Placeholder for GET /account, PATCH /account, DELETE /account
				// account.GET("", middleware.RequirePermission("account", "read"), ...)
				// account.PATCH("", middleware.RequirePermission("account", "update"), ...)
				// account.DELETE("", middleware.RequirePermission("account", "delete"), ...)

				// API Token Management Routes (under /account)
				token := account.Group("/token") // Define token group under account
				{
					// Note: Permissions are checked inside handlers now for api tokens
					token.POST("", api_token.CreateTokenHandler)
					token.GET("", api_token.GetTokensHandler)      // List user's tokens (or admin targets another user via ?user_id=...)
					token.GET("/:id", api_token.GetTokensHandler)  // Get specific token by ID (admin can target another user via ?user_id=...)
					token.PATCH("", api_token.UpdateTokenHandler)  // Requires 'id' in body
					token.DELETE("", api_token.DeleteTokenHandler) // Requires 'id' in body
				}
			}

			// User Management (Admin) - Routes are placeholders
			/* // Comment out until handlers are implemented
			users := protectedGroup.Group("/users")
			{
				// Placeholder for admin user management
				// users.GET("", middleware.RequirePermission("users", "read:list"), ...)
				// users.POST("", middleware.RequirePermission("users", "create"), ...)
				// users.PATCH("/:id", middleware.RequirePermission("users", "update"), ...)
				// users.DELETE("/:id", middleware.RequirePermission("users", "delete"), ...)
			}
			*/

			// Settings
			settingsGroup := protectedGroup.Group("/settings")
			{
				// UI Settings
				settingsGroup.GET("/ui", middleware.RequirePermission("settings", "read"), ui_settings.GetSettingsHandler)
				settingsGroup.PATCH("/ui", middleware.RequirePermission("settings", "update"), ui_settings.UpdateSettingsHandler)
			}

			// Commands, Plugins, Themes (Placeholders)
			// commands := protectedGroup.Group("/commands") { ... }
			// plugins := protectedGroup.Group("/plugins") { ... }
			// themes := protectedGroup.Group("/themes") { ... }
		}
	}

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
func serveHTMLTemplate(c *gin.Context, templatePath string) {
	// Get UI settings
	uiSettings, err := ui_settings.GetUISettings()
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
	if err := tmpl.Execute(c.Writer, uiSettings); err != nil {
		log.Printf("%s Error executing template %s: %v", time.Now().UTC().Format(time.RFC3339), templatePath, err)
		c.String(http.StatusInternalServerError, "Error executing template")
		return
	}
}
