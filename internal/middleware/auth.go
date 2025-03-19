package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/golang-jwt/jwt"
)

// Key type for context values
type contextKey string

// Context keys
const (
	UserContextKey contextKey = "user"
)

// Claims represents the JWT claims
type Claims struct {
	UserID int           `json:"user_id"`
	Role   user.UserRole `json:"role"`
	jwt.StandardClaims
}

// AuthMiddleware handles authentication and authorization
type AuthMiddleware struct {
	config      *config.Config
	userManager user.UserManager
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(cfg *config.Config, userManager user.UserManager) *AuthMiddleware {
	return &AuthMiddleware{
		config:      cfg,
		userManager: userManager,
	}
}

// Authenticate verifies JWT tokens or API keys and adds the user to the request context
func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check if IP is trusted
		clientIP := getClientIP(r)
		if !m.isIPTrusted(clientIP) {
			logger.Warn("Unauthorized access attempt from IP: " + clientIP)
			http.Error(w, "Unauthorized IP", http.StatusUnauthorized)
			return
		}

		// First try to authenticate with Bearer token (JWT)
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenString := strings.TrimPrefix(authHeader, "Bearer ")

			// Parse and validate the token
			claims := &Claims{}
			token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
				return []byte(m.config.JWTSecret), nil
			})

			if err == nil && token.Valid {
				// Token is valid, get the user
				authenticatedUser, err := m.userManager.GetUser(claims.UserID)
				if err == nil && authenticatedUser.Active {
					// Add user to context
					ctx := context.WithValue(r.Context(), UserContextKey, authenticatedUser)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
		}

		// If JWT authentication failed, try API key
		apiKey := r.Header.Get("X-API-Key")
		if apiKey != "" {
			authenticatedUser, err := m.userManager.GetUserByAPIKey(apiKey)
			if err == nil && authenticatedUser.Active {
				// Add user to context
				ctx := context.WithValue(r.Context(), UserContextKey, authenticatedUser)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}

		// Authentication failed
		logger.Warn("Authentication failed for request from: " + clientIP)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
	})
}

// GenerateToken creates a new JWT token for a user
func (m *AuthMiddleware) GenerateToken(u *user.User) (string, error) {
	// Create expiration time (24 hours)
	expirationTime := time.Now().Add(24 * time.Hour)

	// Create claims
	claims := &Claims{
		UserID: u.ID,
		Role:   u.Role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expirationTime.Unix(),
			IssuedAt:  time.Now().Unix(),
			Subject:   u.Username,
		},
	}

	// Create the token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	// Sign the token with the secret
	tokenString, err := token.SignedString([]byte(m.config.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}

// isIPTrusted checks if an IP is in the trusted list
func (m *AuthMiddleware) isIPTrusted(ip string) bool {
	// If no trusted IPs are configured, trust all
	if len(m.config.TrustedIPs) == 0 {
		return true
	}

	for _, trustedIP := range m.config.TrustedIPs {
		if ip == trustedIP {
			return true
		}
	}
	return false
}

// GetUserFromContext retrieves the user from the request context
func GetUserFromContext(r *http.Request) *user.User {
	user, ok := r.Context().Value(UserContextKey).(*user.User)
	if !ok {
		return nil
	}
	return user
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	// If no X-Forwarded-For, use RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
}
