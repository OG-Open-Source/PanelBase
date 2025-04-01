package middleware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/server" // For ErrorResponse
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims defines the structure of the JWT claims used in this application.
// Add fields relevant to your user model (e.g., Role).
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	// Role     string `json:"role"` // Example: Add user role if needed
	jwt.RegisteredClaims // Includes standard claims like Issuer, ExpiresAt, etc.
}

const (
	AuthHeaderKey  = "Authorization"
	BearerSchema   = "Bearer "
	ContextUserKey = "userClaims" // Key to store claims in Gin context
)

// AuthMiddleware creates a Gin middleware for JWT authentication.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	// Convert JWT secret string to []byte for the library
	jwtSecret := []byte(cfg.Auth.JWTSecret)

	return func(c *gin.Context) {
		tokenString := extractToken(c, cfg.Auth.CookieName)
		if tokenString == "" {
			server.ErrorResponse(c, http.StatusUnauthorized, "Authorization token required.")
			return // Abort middleware chain
		}

		// Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing algorithm
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			errMsg := "Invalid or expired token."
			// Log the actual error for debugging, but don't expose details to the client
			fmt.Printf("Token parsing error: %v\n", err) // Replace with proper logging
			// Distinguish between expired and other invalid tokens if needed
			if strings.Contains(err.Error(), "token is expired") {
				errMsg = "Token has expired."
			}
			server.ErrorResponse(c, http.StatusUnauthorized, errMsg)
			return
		}

		if claims, ok := token.Claims.(*Claims); ok && token.Valid {
			// Token is valid, store claims in context for later handlers
			c.Set(ContextUserKey, claims)
			c.Next() // Proceed to the next handler
		} else {
			fmt.Println("Token claims invalid or token is not valid.") // Replace with proper logging
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid token.")
			return
		}
	}
}

// extractToken tries to extract the JWT from the Authorization header or a cookie.
func extractToken(c *gin.Context, cookieName string) string {
	// 1. Check Authorization header
	authHeader := c.GetHeader(AuthHeaderKey)
	if authHeader != "" && strings.HasPrefix(authHeader, BearerSchema) {
		return strings.TrimPrefix(authHeader, BearerSchema)
	}

	// 2. Check cookie
	cookie, err := c.Cookie(cookieName)
	if err == nil && cookie != "" {
		return cookie
	}

	return "" // No token found
}

// Helper function to get claims from context (can be placed here or in a utils package)
func GetClaims(c *gin.Context) (*Claims, bool) {
	claims, exists := c.Get(ContextUserKey)
	if !exists {
		return nil, false
	}
	typedClaims, ok := claims.(*Claims)
	return typedClaims, ok
}

// GenerateToken - Placeholder for token generation logic (should be in an auth service/handler)
func GenerateToken(cfg *config.Config, userID, username string) (string, error) {
	jwtSecret := []byte(cfg.Auth.JWTSecret)
	expirationTime := time.Now().Add(cfg.Auth.TokenDuration)

	claims := &Claims{
		UserID:   userID,
		Username: username,
		// Role:     "admin", // Example
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "PanelBase", // Optional: Identify the issuer
			// Subject: userID, // Optional: Subject of the token
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
