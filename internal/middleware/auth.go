package middleware

import (
	"errors" // Need log for revocation check errors
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	logger "github.com/OG-Open-Source/PanelBase/internal/logging" // Import logging
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"      // For ErrorResponse
	"github.com/OG-Open-Source/PanelBase/internal/token_store" // Import token_store
	"github.com/OG-Open-Source/PanelBase/internal/user"        // Import user service
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// --- JWT Claims Structure and Audience Constants ---

// JWTCustomClaims contains custom JWT claims.
type JWTCustomClaims struct {
	Scopes models.UserPermissions `json:"scopes"`
}

// JWTClaims represents the structure of the JWT payload.
type JWTClaims struct {
	UserID   string `json:"sub"` // Standard claim for Subject (User ID)
	Username string `json:"username,omitempty"`
	JWTCustomClaims
	jwt.RegisteredClaims // Embeds standard claims like exp, iat, aud, iss
}

// Constants for Audience values
const (
	AudienceWeb = "web"
	AudienceAPI = "api"
)

// --- AuthMiddleware ---

// AuthMiddleware validates JWT tokens, checks revocation status, and sets context.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string
		var source string

		// 1. Try to get token from Cookie first
		cookie, err := c.Cookie(cfg.Auth.CookieName)
		if err == nil && cookie != "" {
			tokenString = cookie
			source = "Cookie"
		} else {
			// 2. If no cookie, try Authorization Header
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
				source = "Header"
			}
		}

		// If no token found from either source
		if tokenString == "" {
			c.Next() // Allow unauthenticated access to potentially public routes
			return
		}

		// 3. Parse the token
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			// Use the global JWT secret for web sessions, or fetch user-specific for API?
			// For now, assume web session uses global secret.
			// API tokens would require fetching user info based on a claim first...
			// This middleware needs refinement if supporting user-specific API secrets.
			audience := "unknown"
			if claims, ok := token.Claims.(jwt.MapClaims); ok {
				if audClaim, okAud := claims["aud"]; okAud {
					if audStr, okAudStr := audClaim.(string); okAudStr {
						audience = audStr
					}
				}
			}
			// TODO: Logic to determine correct secret based on audience/claims if needed.
			// For now, assume session tokens use global secret.
			if audience == AudienceWeb {
				return []byte(cfg.Auth.JWTSecret), nil
			} else if audience == "api" {
				// If API token, we need the user ID to get their secret.
				// This adds complexity: parse claims *before* validation?
				// Alternative: Use asymmetric keys or a dedicated secret lookup service.
				// Simplified approach: If it's API, fetch user and use their secret.
				if claims, ok := token.Claims.(jwt.MapClaims); ok {
					if subClaim, okSub := claims["sub"]; okSub {
						if userID, okUserID := subClaim.(string); okUserID {
							userInstance, found, userErr := user.GetUserByID(userID)
							if userErr != nil || !found {
								// Log error but return a generic validation error to avoid info leak
								logger.ErrorPrintf("AUTH_MW", "GET_API_SECRET", "Failed to find user %s for API token validation: %v", userID, userErr)
								return nil, fmt.Errorf("user lookup failed for secret retrieval")
							}
							if userInstance.API.JwtSecret == "" {
								logger.ErrorPrintf("AUTH_MW", "GET_API_SECRET", "User %s has no API JWT secret configured", userID)
								return nil, fmt.Errorf("api secret not configured for user")
							}
							return []byte(userInstance.API.JwtSecret), nil
						}
					}
				}
				// Fallback/error if user ID couldn't be extracted for API token
				logger.ErrorPrintf("AUTH_MW", "GET_API_SECRET", "Could not extract user ID (sub) from API token claims")
				return nil, fmt.Errorf("missing user identifier in api token")
			}

			// Fallback if audience is unknown or missing
			logger.ErrorPrintf("AUTH_MW", "GET_SECRET", "Cannot determine JWT secret: Unknown or missing audience in token")
			return nil, fmt.Errorf("cannot determine validation secret")
		})

		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				// Let expired tokens pass through for potential refresh, but don't set user context.
				logger.DebugPrintf("AUTH_MW", "PARSE", "Token expired (source: %s)", source)
			} else {
				// Log other parsing errors
				logger.ErrorPrintf("AUTH_MW", "PARSE", "Token parsing error (source: %s): %v", source, err)
			}
			c.Next()
			return
		}

		// 4. Validate Claims
		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			// Check audience
			aud, _ := claims["aud"].(string)
			if aud != AudienceWeb && aud != "api" { // Adjust expected audiences if needed
				logger.ErrorPrintf("AUTH_MW", "VALIDATE", "Invalid audience: %s", aud)
				c.Next() // Invalid token, treat as unauthenticated
				return
			}

			// Check if token is revoked using JTI
			jti, okJti := claims["jti"].(string)
			if !okJti {
				logger.ErrorPrintf("AUTH_MW", "VALIDATE", "Token missing JTI claim")
				c.Next()
				return
			}
			revoked, errRevoke := token_store.IsTokenRevoked(jti)
			if errRevoke != nil {
				logger.ErrorPrintf("AUTH_MW", "VALIDATE_REVOKE", "Error checking token revocation for JTI %s: %v", jti, errRevoke)
				server.ErrorResponse(c, http.StatusInternalServerError, "Error checking token status")
				c.Abort()
				return
			}
			if revoked {
				logger.Printf("AUTH_MW", "VALIDATE_REVOKE", "Access denied: Token JTI %s is revoked", jti)
				server.ErrorResponse(c, http.StatusUnauthorized, "Token has been revoked")
				c.Abort()
				return
			}

			// Set context keys
			c.Set(string(ContextKeyIsAuthenticated), true)
			c.Set(string(ContextKeyUserID), claims["sub"])
			c.Set(string(ContextKeyUsername), claims["name"]) // Assuming 'name' claim holds username
			c.Set(string(ContextKeyAudience), aud)
			c.Set(string(ContextKeyJTI), jti)

			// Convert scopes if they exist
			if scopesClaim, okScopes := claims["scopes"]; okScopes {
				if scopesMap, okScopesMap := scopesClaim.(map[string]interface{}); okScopesMap {
					perms := models.UserPermissions{}
					for k, v := range scopesMap {
						if vSlice, okSlice := v.([]interface{}); okSlice {
							strSlice := []string{}
							for _, item := range vSlice {
								if strItem, okStr := item.(string); okStr {
									strSlice = append(strSlice, strItem)
								}
							}
							perms[k] = strSlice
						}
					}
					c.Set(string(ContextKeyPermissions), perms)
				} else {
					logger.ErrorPrintf("AUTH_MW", "VALIDATE_SCOPES", "Invalid scopes format in token claims: not map[string]interface{} for user %s", claims["sub"])
				}
			} else {
				// If scopes are missing, maybe set empty permissions or deny access?
				logger.Printf("AUTH_MW", "VALIDATE_SCOPES", "Scopes claim missing in token for user %s", claims["sub"])
				c.Set(string(ContextKeyPermissions), models.UserPermissions{}) // Set empty permissions
			}

			logger.DebugPrintf("AUTH_MW", "VALIDATE", "Token validated successfully (Source: %s, User: %s, Aud: %s, JTI: %s)", source, claims["sub"], aud, jti)
		} else {
			// Token parsed but invalid (e.g., signature mismatch, NBF, etc.)
			logger.ErrorPrintf("AUTH_MW", "VALIDATE", "Invalid token (source: %s)", source)
		}

		c.Next()
	}
}

// Helper function to get claims from context (can be placed here or in a utils package)
// Update this helper if needed, or create a specific one for permissions
/* // Commenting out old GetClaims as it's not directly used now
func GetClaims(c *gin.Context) (*JWTClaims, bool) {
	claims, exists := c.Get("userClaims") // Consider using specific keys like "userID", "userPermissions"
	if !exists {
		return nil, false
	}
	typedClaims, ok := claims.(JWTClaims)
	return &typedClaims, ok
}
*/

// GenerateToken - Placeholder for token generation logic (should be in an auth service/handler)
// Update this if it's actually used anywhere to use the new AudienceAPI constant.
func GenerateToken(cfg *config.Config, userID, username string) (string, error) {
	jwtSecret := []byte(cfg.Auth.JWTSecret)
	expirationTime := time.Now().Add(cfg.Auth.TokenDuration)

	claims := &JWTClaims{
		UserID:   userID,
		Username: username,
		JWTCustomClaims: JWTCustomClaims{
			Scopes: models.UserPermissions{}, // Populate scopes as needed
		},
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "PanelBase",
			Audience:  []string{AudienceAPI}, // Use the updated AudienceAPI constant
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// CheckPermission function definition ...
/*
func CheckPermission(c *gin.Context, resource string, action string) bool {
    // ... implementation ...
}
*/

// CheckReadPermission function definition ...
/*
func CheckReadPermission(c *gin.Context, resource string) bool {
    // ... implementation ...
}
*/

// Remove the redundant declaration below
/*
// RequirePermission is a middleware factory that checks for a specific permission.
// It uses the CheckPermission function.
func RequirePermission(resource string, requiredAction string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CheckPermission(c, resource, requiredAction) {
			// CheckPermission already calls Abort and sends an error response
			return
		}
		c.Next()
	}
}
*/
