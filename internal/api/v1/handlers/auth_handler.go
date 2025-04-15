package handlers

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/storage"
	"github.com/OG-Open-Source/PanelBase/pkg/response"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// AuthHandler handles authentication related requests.
type AuthHandler struct {
	UserStore        storage.UserStore
	JwtSecret        string
	TokenDurationMin int
	DefaultScopes    map[string]interface{}
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(store storage.UserStore, jwtSecret string, tokenDurationMin int, defaultScopes map[string]interface{}) *AuthHandler {
	return &AuthHandler{
		UserStore:        store,
		JwtSecret:        jwtSecret,
		TokenDurationMin: tokenDurationMin,
		DefaultScopes:    defaultScopes,
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
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid request body: "+err.Error(), nil))
		return
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash password: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to process registration", nil))
		return
	}

	// Ensure request scopes are initialized if nil
	requestScopes := req.Scopes
	if requestScopes == nil {
		requestScopes = make(map[string]interface{})
	}

	// Create user model
	user := &models.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Name:     req.Name,
		Email:    req.Email,
		Active:   true,
		Scopes:   h.DefaultScopes,
	}

	// Store the user
	if err := h.UserStore.CreateUser(context.Background(), user); err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			c.AbortWithStatusJSON(http.StatusConflict, response.Failure("Username already exists", nil))
		} else {
			log.Printf("ERROR: Failed to create user: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to register user", nil))
		}
		return
	}

	respData := gin.H{"user_id": user.ID}
	c.JSON(http.StatusCreated, response.Success("User registered successfully", respData))
}

// LoginRequest defines the expected request body for user login.
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login handles user login and returns a JWT.
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid request body: "+err.Error(), nil))
		return
	}

	// Get user from store
	user, err := h.UserStore.GetUserByUsername(context.Background(), req.Username)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("Invalid username or password", nil))
		} else {
			log.Printf("ERROR: Failed to get user by username: %v", err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Login failed (internal error)", nil))
		}
		return
	}

	// Check if user is active
	if !user.Active {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("User account is inactive", nil))
		return
	}

	// Compare password hash
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("Invalid username or password", nil))
		return
	}

	// Generate JWT (Web Token)
	tokenString, err := auth.GenerateToken(user, h.JwtSecret, h.TokenDurationMin, auth.TokenTypeWeb)
	if err != nil {
		log.Printf("ERROR: Failed to generate token: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Login failed (token generation error)", nil))
		return
	}

	respData := gin.H{"token": tokenString}
	c.JSON(http.StatusOK, response.Success("Login successful", respData))
}
