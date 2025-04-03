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

// --- JWT Claims Structure and Audience Constants ---

// JWTCustomClaims contains custom JWT claims.
type JWTCustomClaims struct {
	Scopes []string `json:"scopes"`
}

// JWTClaims represents the structure of the JWT payload.
type JWTClaims struct {
	UserID   string `json:"sub"` // Standard claim for Subject (User ID)
	Username string `json:"username,omitempty"`
	JWTCustomClaims
	jwt.RegisteredClaims // Embeds standard claims like exp, iat, aud, iss
}

// Audience constants for differentiating token types.
const (
	AudienceWebSession = "web_session"
	AudienceAPIAccess  = "api_access"
)

// --- AuthMiddleware ---

// AuthMiddleware validates JWT tokens from either Cookie or Authorization header.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var tokenSource string // "cookie" or "header"

		// 1. Try to get token from Cookie first
		cookie, err := c.Cookie(cfg.Auth.CookieName)
		if err == nil && cookie != "" {
			tokenString = cookie
			tokenSource = "cookie"
		} else {
			// 2. If not in cookie, try Authorization header (Bearer token)
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				tokenSource = "header"
			}
		}

		// If token is not found in either location
		if tokenString == "" {
			server.ErrorResponse(c, http.StatusUnauthorized, "Authorization token required")
			c.Abort()
			return
		}

		// 3. Parse and validate the token
		token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
			// Ensure the signing method is HMAC
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Return the secret key for validation
			return []byte(cfg.Auth.JWTSecret), nil
		})

		if err != nil {
			// Handle parsing errors (expired, malformed, signature mismatch etc.)
			server.ErrorResponse(c, http.StatusUnauthorized, fmt.Sprintf("Invalid or expired token: %v", err))
			c.Abort()
			return
		}

		// 4. Check if the token and claims are valid
		if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
			// 5. Validate Audience based on token source
			var expectedAudience string
			if tokenSource == "cookie" {
				expectedAudience = AudienceWebSession
			} else { // tokenSource == "header"
				expectedAudience = AudienceAPIAccess
			}

			// Check if the expected audience is present in the token's audience list
			audienceMatch := false
			for _, aud := range claims.Audience {
				if aud == expectedAudience {
					audienceMatch = true
					break
				}
			}

			if !audienceMatch {
				server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Invalid token audience for %s access", tokenSource))
				c.Abort()
				return
			}

			// 6. Validation successful, store user info in context
			c.Set("userID", claims.UserID)
			c.Set("username", claims.Username)        // Store username if present
			c.Set("scopes", claims.Scopes)            // Store scopes (full or subset)
			c.Set("token_audience", expectedAudience) // Store audience for potential use

			// Continue to the next handler
			c.Next()
		} else {
			// General invalid token case (should ideally be caught by ParseWithClaims err)
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
		}
	}
}

// Helper function to get claims from context (can be placed here or in a utils package)
func GetClaims(c *gin.Context) (*JWTClaims, bool) {
	claims, exists := c.Get("userClaims")
	if !exists {
		return nil, false
	}
	typedClaims, ok := claims.(JWTClaims)
	return &typedClaims, ok
}

// GenerateToken - Placeholder for token generation logic (should be in an auth service/handler)
func GenerateToken(cfg *config.Config, userID, username string) (string, error) {
	jwtSecret := []byte(cfg.Auth.JWTSecret)
	expirationTime := time.Now().Add(cfg.Auth.TokenDuration)

	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		JWTCustomClaims: JWTCustomClaims{
			Scopes: []string{}, // Populate scopes as needed
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "PanelBase",                 // Optional: Identify the issuer
			Audience:  []string{AudienceAPIAccess}, // Assuming API access
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}
