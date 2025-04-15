package v1

import (
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers"
	"github.com/OG-Open-Source/PanelBase/internal/api/v1/middleware"
	"github.com/OG-Open-Source/PanelBase/pkg/response"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes for the v1 API group.
// It requires dependencies for the handlers and middleware.
// allowRegister controls whether the POST /auth/register endpoint is enabled.
func RegisterRoutes(
	router *gin.RouterGroup,
	authHandler *handlers.AuthHandler,
	userHandler *handlers.UserHandler,
	accountHandler *handlers.AccountHandler,
	jwtSecret string,
	allowRegister bool,
) {

	// Authentication routes (no auth required)
	authRoutes := router.Group("/auth")
	{
		if allowRegister {
			authRoutes.POST("/register", authHandler.Register)
		} else {
			// Use the standard response format for disabled registration
			authRoutes.POST("/register", func(c *gin.Context) {
				c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("Registration is disabled", nil))
			})
		}
		authRoutes.POST("/login", authHandler.Login)
	}

	// User Management routes (requires authentication and specific scopes)
	userRoutes := router.Group("/users")
	userRoutes.Use(middleware.RequireAuth(jwtSecret)) // All user routes require authentication
	{
		// GET /api/v1/users - List all users (requires users:read scope)
		userRoutes.GET("", middleware.RequireScope("users:read"), userHandler.GetAllUsers)

		// POST /api/v1/users - Create a new user (requires users:create scope)
		userRoutes.POST("", middleware.RequireScope("users:create"), userHandler.CreateUser)

		// GET /api/v1/users/:id - Get a specific user (requires users:read scope)
		userRoutes.GET("/:id", middleware.RequireScope("users:read"), userHandler.GetUserByID)

		// PATCH /api/v1/users/:id - Update a specific user
		// Requires specific update scopes for each field that *might* be updated.
		// Handler assumes permission is granted if request reaches it.
		userRoutes.PATCH("/:id",
			// middleware.RequireScope("users:update:password"), // Example if password change was allowed here
			middleware.RequireScope("users:update:name"),
			middleware.RequireScope("users:update:email"),
			middleware.RequireScope("users:update:active"),
			middleware.RequireScope("users:update:scopes"),
			middleware.RequireScope("users:update:api_tokens"),
			userHandler.UpdateUser, // Handler logic remains simpler
		)

		// DELETE /api/v1/users/:id - Delete a specific user (requires users:delete scope)
		userRoutes.DELETE("/:id", middleware.RequireScope("users:delete"), userHandler.DeleteUser)
	}

	// Account Management routes (requires authentication and specific account scopes)
	accountRoutes := router.Group("/account")
	accountRoutes.Use(middleware.RequireAuth(jwtSecret)) // All account routes require authentication
	{
		// GET /api/v1/account/profile (requires account:profile:read)
		accountRoutes.GET("/profile", middleware.RequireScope("account:profile:read"), accountHandler.GetProfile)

		// PATCH /api/v1/account/profile (requires account:update:name, account:update:email)
		// Apply middleware chain for each field allowed to be updated by the user themselves
		accountRoutes.PATCH("/profile",
			middleware.RequireScope("account:update:name"),
			middleware.RequireScope("account:update:email"),
			accountHandler.UpdateProfile,
		)

		// PATCH /api/v1/account/password (requires account:password:update)
		// Define the scope string as needed, e.g., "account:password:update"
		accountRoutes.PATCH("/password", middleware.RequireScope("account:password:update"), accountHandler.UpdatePassword)

		// DELETE /api/v1/account/delete (requires account:self_delete:execute scope)
		accountRoutes.DELETE("/delete", middleware.RequireScope("account:self_delete:execute"), accountHandler.DeleteSelf)

		// API Tokens
		tokenRoutes := accountRoutes.Group("/tokens")
		{
			// Requires scope to manage tokens, e.g., "account:tokens:create"
			tokenRoutes.POST("", middleware.RequireScope("account:tokens:create"), accountHandler.CreateApiToken)
			// Requires scope to list tokens, e.g., "account:tokens:read"
			tokenRoutes.GET("", middleware.RequireScope("account:tokens:read"), accountHandler.ListApiTokens)
			// Requires scope to delete tokens, e.g., "account:tokens:delete"
			tokenRoutes.DELETE("/:tokenId", middleware.RequireScope("account:tokens:delete"), accountHandler.DeleteApiToken)
		}
	}

	// Example protected route group - REMOVED
	/*
		protected := router.Group("/protected")
		protected.Use(middleware.RequireAuth(jwtSecret))
		{
			protected.GET("/me", func(c *gin.Context) {
				// ... me handler ...
			})

			adminOnly := protected.Group("/admin")
			adminOnly.Use(middleware.RequireAuth(jwtSecret, "admin"))
			{
				adminOnly.GET("/users", func(c *gin.Context) {
					// ... users handler ...
				})
			}
		}
	*/

	// Add more v1 routes here...
}
