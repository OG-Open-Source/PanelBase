package routes

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/gin-gonic/gin"
)

// SetupRoutes configures the main application routes.
func SetupRoutes(router *gin.Engine, cfg *config.Config) {
	// Set Gin mode based on configuration
	gin.SetMode(cfg.Server.Mode)

	// API v1 routes
	apiV1 := router.Group("/api/v1")
	{
		// Authentication routes (public)
		authGroup := apiV1.Group("/auth")
		{
			authGroup.POST("/login", func(c *gin.Context) { server.SuccessResponse(c, "Login endpoint not implemented yet", nil) })
			authGroup.POST("/token", func(c *gin.Context) { server.SuccessResponse(c, "Token refresh endpoint not implemented yet", nil) })
		}

		// Protected API routes
		protectedGroup := apiV1.Group("")
		protectedGroup.Use(middleware.AuthMiddleware(cfg))
		{
			// Commands (only if enabled)
			if cfg.Features.Commands {
				protectedGroup.GET("/commands", func(c *gin.Context) { server.SuccessResponse(c, "Commands endpoint not implemented yet", nil) })
			}

			// Plugins (only if enabled)
			if cfg.Features.Plugins {
				protectedGroup.GET("/plugins", func(c *gin.Context) { server.SuccessResponse(c, "Plugins endpoint not implemented yet", nil) })
			}

			// Themes
			protectedGroup.GET("/themes", func(c *gin.Context) { server.SuccessResponse(c, "Themes endpoint not implemented yet", nil) })

			// Users
			protectedGroup.GET("/users", func(c *gin.Context) { server.SuccessResponse(c, "Users endpoint not implemented yet", nil) })
		}
	}

	// Custom NoRoute handler to serve static files from './web'
	// and fallback to index.html for SPA support.
	router.NoRoute(func(c *gin.Context) {
		// 1. Check if it's an API route
		if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			server.ErrorResponse(c, http.StatusNotFound, "API route not found")
			return
		}

		// 2. Get absolute path for the web directory
		webDir, err := filepath.Abs("./web")
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Internal server error (web dir)")
			return
		}

		// 3. Construct and clean the potential file path
		requestedPath := filepath.Join(webDir, filepath.Clean(c.Request.URL.Path))

		// 4. Security Check: Ensure the cleaned absolute path is still within the web directory
		if !strings.HasPrefix(requestedPath, webDir) {
			server.ErrorResponse(c, http.StatusBadRequest, "Invalid path (security check failed)")
			return
		}

		// 5. Check if the file exists and is not a directory
		if fileInfo, err := os.Stat(requestedPath); err == nil {
			if !fileInfo.IsDir() {
				// Serve the existing file
				http.ServeFile(c.Writer, c.Request, requestedPath)
				return
			}
			// If it's a directory, fall through to serving index.html (SPA behavior)
		}

		// 6. If file doesn't exist or is a directory, serve index.html (SPA fallback)
		indexPath := filepath.Join(webDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(c.Writer, c.Request, indexPath)
		} else {
			// If index.html doesn't exist either, return a proper 404
			server.ErrorResponse(c, http.StatusNotFound, "Resource not found")
		}
	})
}
