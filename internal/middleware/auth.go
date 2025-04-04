package middleware

import (
	"fmt"
	"log" // Need log for revocation check errors
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"     // For ErrorResponse
	"github.com/OG-Open-Source/PanelBase/internal/tokenstore" // Import tokenstore
	"github.com/OG-Open-Source/PanelBase/internal/user"       // Import user service
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

// Audience constants for differentiating token types.
const (
	AudienceWeb = "web"
	AudienceAPI = "api"
)

// Context keys for storing information.
const (
	UsernameKey      = "username"      // DEPRECATED? Might not be needed if using UserID directly
	ContextKeyUserID = "userID"        // Key for User ID (sub claim)
	ContextKeyScopes = "scopes"        // Key for token scopes
	ContextKeyAud    = "tokenAudience" // Key for token audience
	ContextKeyJTI    = "tokenJTI"      // Key for token JTI (ID)
)

// --- AuthMiddleware ---

// AuthMiddleware validates JWT tokens, checks revocation status, and sets context.
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. Extract token string (from cookie or header)
		tokenString := ""
		cookie, err := c.Cookie(cfg.Auth.CookieName) // Use cookie name from config
		if err == nil && cookie != "" {
			tokenString = cookie
		} else {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if tokenString == "" {
			server.ErrorResponse(c, http.StatusUnauthorized, "Authorization token required")
			c.Abort()
			return
		}

		// 2. Parse and validate the token, dynamically selecting the secret key
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Validate the signing method first
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}

			// Peek into claims to determine audience and subject (user ID) without full validation yet.
			// Using MapClaims here as we don't know the specific struct type yet.
			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				return nil, fmt.Errorf("invalid token claims format")
			}

			// Get Audience
			var audience []string
			if aud, audOk := claims["aud"]; audOk {
				if audStr, ok := aud.(string); ok {
					audience = []string{audStr}
				} else if audSlice, ok := aud.([]interface{}); ok {
					for _, v := range audSlice {
						if audStr, ok := v.(string); ok {
							audience = append(audience, audStr)
						}
					}
				} else if audClaimStrings, ok := aud.(jwt.ClaimStrings); ok {
					audience = []string(audClaimStrings)
				}
			}
			if len(audience) == 0 {
				return nil, fmt.Errorf("token audience (aud) claim is missing or invalid")
			}

			// Determine Secret based on Audience
			isAPIToken := false
			for _, aud := range audience {
				if aud == AudienceAPI { // Check for "api"
					isAPIToken = true
					break
				}
			}

			if isAPIToken {
				// --- API Token Validation ---
				sub, subOk := claims["sub"].(string)
				if !subOk || sub == "" {
					return nil, fmt.Errorf("API token subject (sub) claim (UserID) is missing or invalid")
				}

				// Fetch user by ID to get their specific secret
				// TODO: Consider caching user data/secrets to avoid repeated lookups
				userInstance, userExists, err := user.GetUserByID(sub)
				if err != nil {
					// Log the internal error, but return a generic validation error
					log.Printf("Error fetching user %s for API token validation: %v", sub, err)
					return nil, fmt.Errorf("failed to verify API token user")
				}
				if !userExists {
					return nil, fmt.Errorf("user specified in API token (sub: %s) not found", sub)
				}
				if userInstance.API.JwtSecret == "" {
					return nil, fmt.Errorf("user-specific JWT secret not configured for user %s", sub)
				}
				return []byte(userInstance.API.JwtSecret), nil

			} else {
				// --- Assume Web Session Token Validation ---
				// Check if "web" audience is present (optional strict check)
				/*
					isWebToken := false
					for _, aud := range audience {
						if aud == AudienceWeb {
							isWebToken = true
							break
						}
					}
					if !isWebToken {
						return nil, fmt.Errorf("token audience is not 'web' for session validation")
					}
				*/

				// Use the global/config secret for web sessions
				if cfg.Auth.JWTSecret == "" {
					return nil, fmt.Errorf("JWT secret is not configured for web sessions")
				}
				return []byte(cfg.Auth.JWTSecret), nil
			}
		})

		// Simplified Error Handling:
		if err != nil {
			// Remove or comment out the redundant log statement
			// log.Printf("Token parsing error: %v", err)
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid or expired token") // Generic message
			c.Abort()
			return
		}

		if !token.Valid {
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		// 3. Extract claims and set context
		if claims, ok := token.Claims.(jwt.MapClaims); ok {
			var userID string
			var tokenJTI string

			// Store userID (from sub claim)
			if sub, subOk := claims["sub"].(string); subOk {
				userID = sub
				c.Set(ContextKeyUserID, userID) // Use exported constant
				// Set the UsernameKey context with the UserID as well for now.
				// c.Set(UsernameKey, userID) // Keep this commented out unless needed
			}

			// Store JTI (token ID)
			if jti, jtiOk := claims["jti"].(string); jtiOk {
				tokenJTI = jti
				c.Set(ContextKeyJTI, tokenJTI) // Store JTI
			}

			// Store audience as []string
			if aud, audOk := claims["aud"].(string); audOk { // Simplified assuming single string audience
				c.Set(ContextKeyAud, jwt.ClaimStrings{aud}) // Use exported constant
			}

			// Store scopes as models.UserPermissions
			if scopesClaim, scopesOk := claims["scopes"]; scopesOk {
				// Attempt direct type assertion and conversion
				scopesMap, mapOk := scopesClaim.(map[string]interface{}) // Assert to generic map first
				if mapOk {
					convertedScopes := make(models.UserPermissions)
					conversionOK := true
					for resource, actionsInterface := range scopesMap {
						actionsSlice, sliceOk := actionsInterface.([]interface{}) // Assert actions to slice of interfaces
						if !sliceOk {
							log.Printf("Warning: Invalid format for scopes actions in JWT for resource '%s' (expected slice). Data: %+v", resource, actionsInterface)
							conversionOK = false
							break
						}
						var actions []string
						for _, actionInterface := range actionsSlice {
							if actionStr, strOk := actionInterface.(string); strOk {
								actions = append(actions, actionStr)
							} else {
								log.Printf("Warning: Non-string action found in JWT scopes for resource '%s'. Data: %+v", resource, actionInterface)
								conversionOK = false
								break // Break inner loop
							}
						}
						if !conversionOK {
							break
						} // Break outer loop if inner failed
						convertedScopes[resource] = actions
					}

					if conversionOK {
						c.Set(ContextKeyScopes, convertedScopes) // Store the correctly typed map
					} else {
						log.Printf("Error: Failed to fully convert JWT 'scopes' claim to models.UserPermissions due to format issues. Claim data: %+v", scopesClaim)
						// Decide how to handle - deny access? For now, proceed without scopes.
					}
				} else {
					// Log error if the claim wasn't a map[string]interface{}
					log.Printf("Error: JWT 'scopes' claim is not a map[string]interface{}. Type: %T, Data: %+v", scopesClaim, scopesClaim)
				}
			} else {
				// Scopes claim doesn't exist
				// Set empty scopes? Or deny access if scopes are mandatory?
				c.Set(ContextKeyScopes, models.UserPermissions{}) // Set empty map
			}

			// 4. Check revocation status using tokenstore
			if tokenJTI != "" {
				isRevoked, err := tokenstore.IsTokenRevoked(tokenJTI)
				if err != nil {
					// Log the error and return 500 Internal Server Error
					log.Printf("ERROR: Failed to check token revocation status for jti %s: %v", tokenJTI, err)
					server.ErrorResponse(c, http.StatusInternalServerError, "Failed to check token status")
					c.Abort()
					return
				}
				if isRevoked {
					// Token is validly signed and not expired, but has been revoked
					server.ErrorResponse(c, http.StatusUnauthorized, "Token has been revoked")
					c.Abort()
					return
				}
			} else {
				// Token is missing JTI claim, should we allow or deny?
				// For now, let's deny as JTI is crucial for revocation.
				server.ErrorResponse(c, http.StatusUnauthorized, "Token missing required ID (jti)")
				c.Abort()
				return
			}

			c.Next() // Proceed to the next handler if token is valid and not revoked
		} else {
			// This should technically not happen if token parsing succeeded
			server.ErrorResponse(c, http.StatusUnauthorized, "Invalid token claims")
			c.Abort()
		}
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
