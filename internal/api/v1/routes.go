package v1

import (
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/api/v1/handlers"
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes for the v1 API group.
// It requires dependencies for the handlers and middleware.
// allowRegister controls whether the POST /auth/register endpoint is enabled.
func RegisterRoutes(router *gin.RouterGroup, authHandler *handlers.AuthHandler, jwtSecret string, allowRegister bool) {

	// Authentication routes (no auth required)
	authRoutes := router.Group("/auth")
	{
		if allowRegister {
			authRoutes.POST("/register", authHandler.Register)
		} else {
			// Optionally, return a specific status code like 405 Method Not Allowed or 404 Not Found
			authRoutes.POST("/register", func(c *gin.Context) {
				c.JSON(http.StatusNotFound, gin.H{"error": "Registration is disabled"})
			})
		}
		authRoutes.POST("/login", authHandler.Login)
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
