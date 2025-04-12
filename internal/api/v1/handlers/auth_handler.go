package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/storage"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication related requests.
type AuthHandler struct {
	UserStore        storage.UserStore
	JwtSecret        string
	TokenDurationMin int
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(store storage.UserStore, jwtSecret string, tokenDurationMin int) *AuthHandler {
	return &AuthHandler{
		UserStore:        store,
		JwtSecret:        jwtSecret,
		TokenDurationMin: tokenDurationMin,
	}
}

// RegisterRequest defines the expected request body for user registration.
type RegisterRequest struct {
	Username string              `json:"username" binding:"required,min=3"`
	Password string              `json:"password" binding:"required,min=6"`
	Name     string              `json:"name"`                            // Added Name
	Email    string              `json:"email" binding:"omitempty,email"` // Added Email (optional, validate format)
	Scopes   map[string][]string `json:"scopes"`                          // Changed from Permissions
}

// Register handles user registration.
func (h *AuthHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash password: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process registration"})
		return
	}

	// Create user model
	user := &models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,   // Set Name
		Email:        req.Email,  // Set Email
		Active:       true,       // Default new users to active
		Scopes:       req.Scopes, // Use provided scopes or nil/empty map
		// CreatedAt will be set by the store
	}
	// Ensure Scopes map is initialized if nil from request
	if user.Scopes == nil {
		user.Scopes = make(map[string][]string)
	}

	// Store the user
	if err := h.UserStore.CreateUser(context.Background(), user); err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			c.JSON(http.StatusConflict, gin.H{"error": "Username already exists"})
		} else {
			log.Printf("ERROR: Failed to create user: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to register user"})
		}
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user_id": user.ID})
}

// LoginRequest defines the expected request body for user login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse defines the response body for successful login.
type LoginResponse struct {
	Token string `json:"token"`
}

// Login handles user login and returns a JWT.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// Get user from store
	user, err := h.UserStore.GetUserByUsername(context.Background(), req.Username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		} else {
			log.Printf("ERROR: Failed to get user by username: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		}
		return
	}

	// Check if user is active
	if !user.Active {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User account is inactive"})
		return
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		// If err is bcrypt.ErrMismatchedHashAndPassword, return unauthorized
		// Otherwise, it might be an internal error (e.g., invalid hash)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid username or password"})
		return
	}

	// Generate JWT
	tokenString, err := auth.GenerateToken(user, h.JwtSecret, h.TokenDurationMin)
	if err != nil {
		log.Printf("ERROR: Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
}
 