package handler

import (
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/config"
	"github.com/OG-Open-Source/PanelBase/internal/app/model"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/sirupsen/logrus"
)

// Handler handles HTTP requests
type Handler struct {
	config *config.Config
	// Add other dependencies like database, services etc.
}

// NewHandler creates a new handler
func NewHandler(cfg *config.Config) *Handler {
	return &Handler{
		config: cfg,
	}
}

// AuthMiddleware handles authentication for protected routes
func (h *Handler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get JWT token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			return
		}

		// Check if the format is "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization format"})
			return
		}

		// Validate JWT token
		tokenString := parts[1]
		claims := &model.Claims{}

		// Parse the JWT token
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			// TODO: Get JWT secret from user config file
			return []byte("your-secret-key"), nil
		})

		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Store user information in context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// Login handles user authentication and returns a JWT token
func (h *Handler) Login(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request data"})
		return
	}

	// TODO: Validate credentials against user database
	// This is just a placeholder implementation
	if loginData.Username == "admin" && loginData.Password == "admin" {
		// Create a new token
		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &model.Claims{
			UserID:   "1",
			Username: loginData.Username,
			Role:     "admin",
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		// TODO: Get JWT secret from user config file
		tokenString, err := token.SignedString([]byte("your-secret-key"))
		if err != nil {
			logrus.Errorf("Failed to create JWT token: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
			return
		}

		c.JSON(http.StatusOK, gin.H{"token": tokenString, "expiresAt": expirationTime})
	} else {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
	}
}

// GetSystemInfo returns information about the system
func (h *Handler) GetSystemInfo(c *gin.Context) {
	// TODO: Implement actual system info retrieval
	info := map[string]interface{}{
		"version":   "1.0.0",
		"goVersion": "1.19",
		"platform":  "PanelBase",
		"uptime":    "10h 30m",
	}

	c.JSON(http.StatusOK, info)
}

// IndexPage renders the home page
func (h *Handler) IndexPage(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "PanelBase",
	})
}

// LoginPage renders the login page
func (h *Handler) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "Login - PanelBase",
	})
}

// AdminEntryPage renders the admin entry page
func (h *Handler) AdminEntryPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admin.html", gin.H{
		"title": "Admin - PanelBase",
	})
}

// ListUsers returns a list of all users
func (h *Handler) ListUsers(c *gin.Context) {
	// TODO: Get users from database
	users := []model.User{
		{ID: "1", Username: "admin", Role: "admin"},
		{ID: "2", Username: "user", Role: "user"},
	}

	c.JSON(http.StatusOK, users)
}

// GetUser returns details for a specific user
func (h *Handler) GetUser(c *gin.Context) {
	id := c.Param("id")
	// TODO: Get user from database
	user := model.User{
		ID:       id,
		Username: "admin",
		Role:     "admin",
	}

	c.JSON(http.StatusOK, user)
}

// CreateUser creates a new user
func (h *Handler) CreateUser(c *gin.Context) {
	var user model.User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user data"})
		return
	}

	// TODO: Save user to database
	user.ID = "3" // Placeholder ID

	c.JSON(http.StatusCreated, user)
}

// UpdateUser updates an existing user
func (h *Handler) UpdateUser(c *gin.Context) {
	id := c.Param("id")

	var userData model.User
	if err := c.ShouldBindJSON(&userData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user data"})
		return
	}

	// TODO: Update user in database
	userData.ID = id

	c.JSON(http.StatusOK, userData)
}

// DeleteUser deletes a user
func (h *Handler) DeleteUser(c *gin.Context) {
	id := c.Param("id")

	// TODO: Delete user from database
	logrus.Infof("Deleted user with ID: %s", id)

	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}
