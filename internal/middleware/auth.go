package middleware

import (
	"fmt"
	// "log" // Ensure this line is commented out or removed
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server" // For ErrorResponse
	"github.com/OG-Open-Source/PanelBase/internal/user"   // Import user service
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

// UsernameKey is the key used to store the authenticated username in the Gin context.
const UsernameKey = "username"

// --- AuthMiddleware ---

// AuthMiddleware validates JWT tokens, selecting the correct secret based on audience.
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
					// log.Printf("Error fetching user %s for API token validation: %v", sub, err)
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
			// Log the specific error for debugging if needed
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

			// Store userID (from sub claim)
			if sub, subOk := claims["sub"].(string); subOk {
				userID = sub
				c.Set("userID", userID) // Set the key RefreshTokenHandler expects
				// Set the UsernameKey context with the UserID as well for now.
				c.Set(UsernameKey, userID)
			}

			// Store audience as []string
			if aud, audOk := claims["aud"]; audOk {
				var audienceList []string
				if audStr, ok := aud.(string); ok {
					audienceList = []string{audStr}
				} else if audSlice, ok := aud.([]interface{}); ok {
					for _, v := range audSlice {
						if audStr, ok := v.(string); ok {
							audienceList = append(audienceList, audStr)
						}
					}
				}
				c.Set("tokenAudience", jwt.ClaimStrings(audienceList)) // Set the key RefreshTokenHandler expects, using compatible type
			}

			// Store scopes as models.UserPermissions
			if scopes, scopesOk := claims["scopes"]; scopesOk {
				var perms models.UserPermissions = make(models.UserPermissions)
				if scopesMap, ok := scopes.(map[string]interface{}); ok {
					// Convert map[string]interface{} to map[string][]string
					for k, v := range scopesMap {
						if actionsInterface, ok := v.([]interface{}); ok {
							actions := []string{}
							for _, act := range actionsInterface {
								if actStr, ok := act.(string); ok {
									actions = append(actions, actStr)
								}
							}
							perms[k] = actions
						} // else: handle unexpected scope format?
					}
				} else if scopesPerms, ok := scopes.(models.UserPermissions); ok {
					// If it's already the correct type
					perms = scopesPerms
				}
				c.Set("userPermissions", perms) // Set the key RefreshTokenHandler expects
			}

		} else {
			server.ErrorResponse(c, http.StatusInternalServerError, "Invalid token claims format")
			c.Abort()
			return
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

// RequirePermission returns a Gin middleware handler that checks if the user
// has the required permission (action) for a specific resource.
// It utilizes the CheckPermission function.
func RequirePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// CheckPermission already handles sending the error response and aborting.
		if CheckPermission(c, resource, action) {
			// Permission granted, continue to the next handler.
			c.Next()
		}
		// If CheckPermission returns false, it has already aborted the context,
		// so we don't need to explicitly call c.Abort() here again.
	}
}
