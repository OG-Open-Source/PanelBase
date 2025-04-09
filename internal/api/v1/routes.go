package v1

import (
	// "github.com/OG-Open-Source/PanelBase/internal/api_token"
	// "github.com/OG-Open-Source/PanelBase/internal/auth"
	// Use the new handler location
	"github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers/auth"
	"github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers/settings"
	"github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers/token"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"

	// Removed import for ui_settings as handlers are moved
	// "github.com/OG-Open-Source/PanelBase/internal/ui_settings"
	"github.com/gin-gonic/gin"
)

// SetupV1Routes configures the API v1 routes.
func SetupV1Routes(apiV1 *gin.RouterGroup) {
	// Authentication routes (public)
	authGroup := apiV1.Group("/auth")
	{
		// Use handlers from the new package
		authGroup.POST("/login", auth.LoginHandler())
		authGroup.POST("/register", auth.RegisterHandler)
	}

	// Protected API routes
	protectedGroup := apiV1.Group("")
	protectedGroup.Use(middleware.AuthMiddleware())
	{
		// Token Refresh Route
		// Use handler from the new package
		protectedGroup.POST("/auth/token", auth.RefreshTokenHandler())

		// Account Management (Self)
		account := protectedGroup.Group("/account")
		{
			// Placeholder for GET /account, PATCH /account, DELETE /account
			// account.GET("", middleware.RequirePermission("account", "read"), ...)
			// account.PATCH("", middleware.RequirePermission("account", "update"), ...)
			// account.DELETE("", middleware.RequirePermission("account", "delete"), ...)

			// API Token Management Routes (under /account)
			tokenGroup := account.Group("/token") // Renamed variable for clarity
			{
				// Use handlers from the new token package
				tokenGroup.POST("", token.CreateTokenHandler)       // Create token for self
				tokenGroup.GET("", token.GetTokensHandler)          // List user's tokens
				tokenGroup.GET("/:id", token.GetTokensHandler)      // Get specific token by ID
				tokenGroup.PATCH("/:id", token.UpdateTokenHandler)  // UPDATE: Use path param :id
				tokenGroup.DELETE("/:id", token.DeleteTokenHandler) // UPDATE: Use path param :id
			}
		}

		// User Management (Admin)
		users := protectedGroup.Group("/users/:user_id") // Base path includes user_id
		{
			// Admin API Token Management Routes (under /users/:user_id)
			adminTokenGroup := users.Group("/token")
			{
				// Admin routes require specific permissions checked in handlers
				adminTokenGroup.POST("", token.CreateTokenHandler)       // Create token for user_id
				adminTokenGroup.GET("", token.GetTokensHandler)          // List tokens for user_id
				adminTokenGroup.GET("/:id", token.GetTokensHandler)      // Get specific token for user_id
				adminTokenGroup.PATCH("/:id", token.UpdateTokenHandler)  // Update specific token for user_id
				adminTokenGroup.DELETE("/:id", token.DeleteTokenHandler) // Delete specific token for user_id
			}

			/* // Comment out until other admin user handlers are implemented
			// Placeholder for other admin user management
			users.GET("", middleware.RequirePermission("users", "read:list"), ...)
			users.POST("", middleware.RequirePermission("users", "create"), ...)
			users.PATCH("", middleware.RequirePermission("users", "update"), ...)
			users.DELETE("", middleware.RequirePermission("users", "delete"), ...)
			*/
		}
		// Note: Need to decide if top-level /users is needed for listing all users, etc.

		// Settings
		settingsGroup := protectedGroup.Group("/settings")
		{
			// Use handlers from the new settings package
			settingsGroup.GET("/ui", middleware.RequirePermission("settings", "read"), settings.GetSettingsHandler)
			settingsGroup.PATCH("/ui", middleware.RequirePermission("settings", "update"), settings.UpdateSettingsHandler)
		}

		// Commands, Plugins, Themes (Placeholders)
		// commands := protectedGroup.Group("/commands") { ... }
		// plugins := protectedGroup.Group("/plugins") { ... }
		// themes := protectedGroup.Group("/themes") { ... }
	}
}
