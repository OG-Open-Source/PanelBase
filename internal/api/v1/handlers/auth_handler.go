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
	Username string `json:"username" binding:"required,min=3"`
	Password string `json:"password" binding:"required,min=6"`
	Name     string `json:"name"`
	Email    string `json:"email" binding:"omitempty,email"`
	// Allow registering with scopes, but they might be overridden by CreateUser logic
	Scopes map[string]interface{} `json:"scopes"` // Changed to map[string]interface{}
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

	// Ensure request scopes are initialized if nil
	requestScopes := req.Scopes
	if requestScopes == nil {
		requestScopes = make(map[string]interface{}) // Use interface{} type
	}

	// Create user model
	user := &models.User{
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		Name:         req.Name,
		Email:        req.Email,
		Active:       true,
		Scopes:       requestScopes, // Assign map[string]interface{}
	}

	// Note: The CreateUser handler will potentially override these scopes
	// with default scopes if the registration endpoint itself doesn't
	// inherently grant 'users:update:scopes' permission (which it shouldn't).

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

	// Generate JWT (Web Token)
	tokenString, err := auth.GenerateToken(user, h.JwtSecret, h.TokenDurationMin, auth.TokenTypeWeb) // Specify token type
	if err != nil {
		log.Printf("ERROR: Failed to generate token: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	c.JSON(http.StatusOK, LoginResponse{Token: tokenString})
}
