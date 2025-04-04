package routes

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/api_token"
	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/server"
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
					// Use the new CheckReadPermission function
					if !middleware.CheckReadPermission(c, "users") {
						return // CheckReadPermission handles the error response and aborts
					}
					// TODO: Implement actual logic to fetch user(s) based on whether ID was in body
					server.SuccessResponse(c, "GET Users (List/Item) endpoint needs implementation", nil)
				})

				// POST /users - Create user (Action: create)
				userGroup.POST("", func(c *gin.Context) {
					// Use the regular CheckPermission for non-read actions
					if !middleware.CheckPermission(c, "users", "create") {
						return // CheckPermission handles the error response and aborts
					}
					// TODO: Implement actual logic to parse body and create user
					server.SuccessResponse(c, "POST Users (Create) endpoint needs implementation", nil)
				})

				// PUT /users - Update user (Action: update, requires {"id": ...} in body)
				userGroup.PUT("", func(c *gin.Context) {
					// TODO: Parse body *first* to get the target ID for potential ownership checks later.
					// targetID, idProvided := extractIDFromRequestBody(c) // Example, maybe bind struct
					// if !idProvided { /* handle error */ }

					// Check basic permission
					if !middleware.CheckPermission(c, "users", "update") {
						return
					}
					// TODO: Add logic here to check if user is updating self vs other (if needed)
					// using targetID and userID from context.
					// TODO: Implement actual logic to update user
					server.SuccessResponse(c, "PUT Users (Update) endpoint needs implementation", nil)
				})

				// DELETE /users - Delete user (Action: delete, requires {"id": ...} in body)
				userGroup.DELETE("", func(c *gin.Context) {
					// TODO: Parse body *first* to get the target ID.
					// targetID, idProvided := extractIDFromRequestBody(c)
					// if !idProvided { /* handle error */ }

					// Check basic permission
					if !middleware.CheckPermission(c, "users", "delete") {
						return
					}
					// TODO: Add logic here to check if user is deleting self vs other (if needed)
					// using targetID and userID from context.
					// TODO: Implement actual logic to delete user
					server.SuccessResponse(c, "DELETE Users (Delete) endpoint needs implementation", nil)
				})

				// --- User's Own API Token Management ---
				// Mounted under /api/v1/users/token
				selfTokenGroup := userGroup.Group("/token")
				{
					// GET /api/v1/users/token - List own API tokens or get specific one via body ID
					selfTokenGroup.GET("", api_token.GetTokensHandler)

					// POST /api/v1/users/token - Create a new API token for the user
					selfTokenGroup.POST("", middleware.RequirePermission("api", "create"), api_token.CreateTokenHandler)

					// PUT /api/v1/users/token - Update own token metadata (name, description)
					selfTokenGroup.PUT("", middleware.RequirePermission("api", "update"), api_token.UpdateTokenHandler)

					// DELETE /api/v1/users/token - Delete own token
					selfTokenGroup.DELETE("", middleware.RequirePermission("api", "delete"), api_token.DeleteTokenHandler)
				}
			}
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
