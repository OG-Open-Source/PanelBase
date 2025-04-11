package v1

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all routes for the v1 API group
func RegisterRoutes(router *gin.RouterGroup) {
	// Example route within the v1 group
	router.GET("/hello", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Hello from API v1",
		})
	})

	// Add more v1 routes here...
}
