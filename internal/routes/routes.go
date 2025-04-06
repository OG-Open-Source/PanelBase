package routes

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/api_token"
	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/OG-Open-Source/PanelBase/internal/ui_settings"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the main application routes.
func SetupRoutes(router *gin.Engine, cfg *config.Config) {
	// API v1 routes
	apiV1 := router.Group("/api/v1")
	{
		// Authentication routes (public)
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/login", auth.LoginHandler(cfg))
			authGroup.POST("/register", auth.RegisterHandler)
		}

		// Protected API routes
		protectedGroup := apiV1.Group("")
		protectedGroup.Use(middleware.AuthMiddleware(cfg))
		{
			// Token Refresh Route
			protectedGroup.POST("/auth/token", auth.RefreshTokenHandler(cfg))

			// Commands (only if enabled)
			if cfg.Features.Commands {
				cmdGroup := protectedGroup.Group("/commands")
				{
					cmdGroup.GET("", func(c *gin.Context) {
						server.SuccessResponse(c, "GET Commands (List) endpoint not implemented yet", nil)
					})
					cmdGroup.POST("", func(c *gin.Context) {
						server.SuccessResponse(c, "POST Commands (Create) endpoint not implemented yet", nil)
					})
					cmdGroup.PUT("", func(c *gin.Context) {
						server.SuccessResponse(c, "PUT Commands (Update - requires {'id': ...} in body) endpoint not implemented yet", nil)
					})
					cmdGroup.DELETE("", func(c *gin.Context) {
						server.SuccessResponse(c, "DELETE Commands (Delete - requires {'id': ...} in body) endpoint not implemented yet", nil)
					})
				}
			}

			// Plugins (only if enabled)
			if cfg.Features.Plugins {
				pluginGroup := protectedGroup.Group("/plugins")
				{
					pluginGroup.GET("", func(c *gin.Context) {
						server.SuccessResponse(c, "GET Plugins (List) endpoint not implemented yet", nil)
					})
					// POST might use {"url": "..."} in body to install from URL
					pluginGroup.POST("", func(c *gin.Context) {
						server.SuccessResponse(c, "POST Plugins (Install/Create - may use {'url': ...} in body) endpoint not implemented yet", nil)
					})
					// PUT requires {"id": "..."} in body
					pluginGroup.PUT("", func(c *gin.Context) {
						server.SuccessResponse(c, "PUT Plugins (Update - requires {'id': ...} in body) endpoint not implemented yet", nil)
					})
					// DELETE requires {"id": "..."} in body
					pluginGroup.DELETE("", func(c *gin.Context) {
						server.SuccessResponse(c, "DELETE Plugins (Uninstall - requires {'id': ...} in body) endpoint not implemented yet", nil)
					})
				}
			}

			// Themes
			themeGroup := protectedGroup.Group("/themes")
			{
				themeGroup.GET("", func(c *gin.Context) { server.SuccessResponse(c, "GET Themes (List) endpoint not implemented yet", nil) })
				// POST might use {"url": "..."} in body to install from URL
				themeGroup.POST("", func(c *gin.Context) {
					server.SuccessResponse(c, "POST Themes (Install/Create - may use {'url': ...} in body) endpoint not implemented yet", nil)
				})
				// PUT requires {"id": "..."} in body
				themeGroup.PUT("", func(c *gin.Context) {
					server.SuccessResponse(c, "PUT Themes (Update - requires {'id': ...} in body) endpoint not implemented yet", nil)
				})
				// DELETE requires {"id": "..."} in body
				themeGroup.DELETE("", func(c *gin.Context) {
					server.SuccessResponse(c, "DELETE Themes (Uninstall - requires {'id': ...} in body) endpoint not implemented yet", nil)
				})
			}

			// General User Management (/api/v1/users)
			userGroup := protectedGroup.Group("/users")
			{
				// GET /users - List users OR Get specific user by ID in body
				userGroup.GET("", func(c *gin.Context) {
					// Permission check (read:list or read:item) and ownership check
					// should be implemented INSIDE the handler logic, using middleware.CheckPermission
					// and comparing target ID with context user ID.
					// Removed: if !middleware.CheckReadPermission(c, "users") { return }
					server.SuccessResponse(c, "GET Users (List/Item) endpoint needs permission checks and implementation", nil)
				})

				// POST /users - Create user (Action: create)
				userGroup.POST("", middleware.RequirePermission("users", "create"), func(c *gin.Context) {
					// Handler needs implementation
					server.SuccessResponse(c, "POST Users (Create) endpoint needs implementation", nil)
				})

				// PUT /users - Update user (Action: update, requires {"id": ...} in body)
				// PATCH /users - Update user (Action: update)
				userGroup.PATCH("", middleware.RequirePermission("users", "update"), func(c *gin.Context) {
					// Handler needs to perform ownership check before updating.
					// Note: Using PATCH on collection is non-standard. Consider PATCH /users/{id}
					server.SuccessResponse(c, "PATCH Users (Update) endpoint needs ownership check and implementation", nil)
				})

				// DELETE /users - Delete user (Action: delete, requires {"id": ...} in body)
				userGroup.DELETE("", middleware.RequirePermission("users", "delete"), func(c *gin.Context) {
					// Handler needs to perform ownership check before deleting.
					server.SuccessResponse(c, "DELETE Users (Delete) endpoint needs ownership check and implementation", nil)
				})

				// --- User's Own API Token Management (and Admin) ---
				// Mounted under /api/v1/users/token
				selfTokenGroup := userGroup.Group("/token")
				{
					// GET /api/v1/users/token - List/Get tokens (self or admin)
					// Permission checks are handled inside GetTokensHandler.
					selfTokenGroup.GET("", api_token.GetTokensHandler)

					// POST /api/v1/users/token - Create token (self or admin)
					// Permission check (api:create or api:create:all) is inside CreateTokenHandler
					selfTokenGroup.POST("", api_token.CreateTokenHandler)

					// PUT /api/v1/users/token - Update token (self or admin)
					// Permission check (api:update or api:update:all) is inside UpdateTokenHandler
					// PATCH /api/v1/users/token - Update token (self or admin)
					// Permission check (api:update or api:update:all) is inside UpdateTokenHandler
					selfTokenGroup.PATCH("", api_token.UpdateTokenHandler)

					// DELETE /api/v1/users/token - Delete token (self or admin)
					// Permission check (api:delete or api:delete:all) is inside DeleteTokenHandler
					selfTokenGroup.DELETE("", api_token.DeleteTokenHandler)
				}
			}

			// Settings Routes
			settingsGroup := protectedGroup.Group("/settings")
			{
				// UI Settings
				settingsGroup.GET("/ui", middleware.RequirePermission("settings", "read"), ui_settings.GetSettingsHandler)
				// settingsGroup.PUT("/ui", middleware.RequirePermission("settings", "update"), uisettings.UpdateSettingsHandler) // Old PUT
				settingsGroup.PATCH("/ui", middleware.RequirePermission("settings", "update"), ui_settings.UpdateSettingsHandler)
			}
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
	// Load UI settings
	settings, err := ui_settings.GetUISettings()
	if err != nil {
		log.Printf("Error getting UI settings for template %s: %v", templatePath, err)
		// Fallback to default settings or render a basic error page
		settings = &models.UISettings{Title: "PanelBase Error"} // Basic fallback
	}

	// Parse the specified HTML template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		log.Printf("Error parsing template %s: %v", templatePath, err)
		c.String(http.StatusInternalServerError, "Error loading page template")
		return
	}

	// Prepare data for the template, explicitly marking CSS and JS
	templateData := struct {
		Title      string
		LogoURL    string
		FaviconURL string
		CustomCSS  template.CSS // Mark as safe CSS
		CustomJS   template.JS  // Mark as safe JS
	}{
		Title:      settings.Title,
		LogoURL:    settings.LogoURL,
		FaviconURL: settings.FaviconURL,
		CustomCSS:  template.CSS(settings.CustomCSS),
		CustomJS:   template.JS(settings.CustomJS),
	}

	// Explicitly set status code to 200 OK
	c.Status(http.StatusOK)
	// Set content type header
	c.Header("Content-Type", "text/html; charset=utf-8")
	// Execute the template with the UI settings data
	err = tmpl.Execute(c.Writer, templateData)
	if err != nil {
		log.Printf("Error executing template %s: %v", templatePath, err)
		// Avoid writing error string if headers already sent
	}
}
