package middleware

import (
	"errors"
	"fmt"
	"log" // Need log for revocation check errors
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/pkg/models"
	"github.com/OG-Open-Source/PanelBase/pkg/serverutils" // UPDATED PATH
	"github.com/OG-Open-Source/PanelBase/pkg/tokenstore"  // Ensure this is the correct path
	"github.com/OG-Open-Source/PanelBase/pkg/userservice" // UPDATED PATH // Import user service
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

// Constants for audience checks
const (
	AudienceWeb = "web"
	AudienceAPI = "api"
)

// AuthMiddleware creates a middleware handler for JWT authentication.
// It checks for the token in the Authorization header (Bearer) or a cookie.
func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ExtractToken(c)
		if tokenString == "" {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Authorization token required")
			c.Abort()
			return
		}

		// Need the user's JWT secret to verify the token.
		// This is tricky because we don't know the user yet.
		// Option 1: Decode claims without verification first to get user ID (sub).
		// Option 2: Use a global secret (less secure if compromised).
		// Option 3: Pass the user ID somehow (e.g., in another header? Risky).
		// Let's go with Option 1 for now, acknowledging the initial unverified step.

		// --- Initial Parse (Unverified) to get User ID --- //
		parser := jwt.Parser{}
		unverifiedToken, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
		if err != nil {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid token format")
			c.Abort()
			return
		}

		claims, ok := unverifiedToken.Claims.(jwt.MapClaims)
		if !ok {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid token claims")
			c.Abort()
			return
		}

		subject, _ := claims.GetSubject()
		if subject == "" {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Token subject (user ID) missing")
			c.Abort()
			return
		}

		// --- Load User and Secret --- //
		userInstance, userExists, err := userservice.GetUserByID(subject)
		if err != nil || !userExists {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid token: User not found")
			c.Abort()
			return
		}
		if userInstance.API.JwtSecret == "" {
			log.Printf("JWT Secret not configured for user %s", userInstance.ID)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Internal token validation error")
			c.Abort()
			return
		}
		jwtSecret := []byte(userInstance.API.JwtSecret)

		// --- Parse and Verify Token --- //
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil {
			if err == jwt.ErrSignatureInvalid {
				serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid token signature")
			} else if errors.Is(err, jwt.ErrTokenExpired) {
				serverutils.ErrorResponse(c, http.StatusUnauthorized, "Token has expired")
			} else {
				serverutils.ErrorResponse(c, http.StatusUnauthorized, fmt.Sprintf("Invalid token: %v", err))
			}
			c.Abort()
			return
		}

		if !token.Valid {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid token")
			c.Abort()
			return
		}

		// --- Extract Verified Claims --- //
		verifiedClaims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Invalid token claims format after verification")
			c.Abort()
			return
		}

		// Check required claims: aud, sub, jti
		audience, _ := verifiedClaims.GetAudience()
		jtiValue, jtiExists := verifiedClaims["jti"] // Access jti directly
		jti := ""
		if jtiExists {
			jti, _ = jtiValue.(string)
		}
		userID, _ := verifiedClaims.GetSubject()

		if len(audience) == 0 || jti == "" || userID == "" {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Token missing required claims (aud, sub, jti)")
			c.Abort()
			return
		}

		// Verify the subject claim matches the user loaded earlier
		if userID != userInstance.ID {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Token subject mismatch")
			c.Abort()
			return
		}

		// Check token revocation status using JTI
		isRevoked, err := tokenstore.IsTokenRevoked(jti) // Use correct package name
		if err != nil {
			log.Printf("Error checking token revocation for JTI %s: %v", jti, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to verify token status")
			c.Abort()
			return
		}
		if isRevoked {
			serverutils.ErrorResponse(c, http.StatusUnauthorized, "Token has been revoked")
			c.Abort()
			return
		}

		// Extract permissions (scopes) from token
		var tokenPermissions models.UserPermissions
		if scopesClaim, ok := verifiedClaims["scopes"]; ok {
			if scopesMap, ok := scopesClaim.(map[string]interface{}); ok {
				tokenPermissions = convertClaimScopes(scopesMap)
			} else {
				log.Printf("Invalid scopes format in token for user %s", userID)
				tokenPermissions = make(models.UserPermissions)
			}
		} else {
			// Use user's base permissions if not in token?
			// Or require scopes in token? Let's use user's base for now.
			tokenPermissions = userInstance.Scopes
		}

		// Store extracted information in context
		c.Set(string(ContextKeyUserID), userID)
		c.Set(string(ContextKeyPermissions), tokenPermissions)
		c.Set(string(ContextKeyAudience), audience[0]) // Assuming single audience
		c.Set(string(ContextKeyJTI), jti)

		c.Next()
	}
}

// ExtractToken extracts the JWT token from the Authorization header or cookie.
func ExtractToken(c *gin.Context) string {
	// Check Authorization header first
	header := c.GetHeader("Authorization")
	if header != "" {
		parts := strings.Split(header, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// Fallback to cookie
	// TODO: Make cookie name configurable
	cookieName := "panelbase_jwt" // Get from config
	cookie, err := c.Cookie(cookieName)
	if err == nil {
		return cookie
	}

	return ""
}

// convertClaimScopes converts the generic map[string]interface{} from JWT claims
// into the specific models.UserPermissions type.
func convertClaimScopes(claimScopes map[string]interface{}) models.UserPermissions {
	permissions := make(models.UserPermissions)
	for resource, actionsInterface := range claimScopes {
		if actionsSlice, ok := actionsInterface.([]interface{}); ok {
			var actions []string
			for _, actionInterface := range actionsSlice {
				if actionStr, ok := actionInterface.(string); ok {
					actions = append(actions, actionStr)
				}
			}
			if len(actions) > 0 {
				permissions[resource] = actions
			}
		}
	}
	return permissions
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
func GenerateToken(userID, username string, jwtSecret string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)

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
			Audience:  []string{AudienceAPI},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(jwtSecret))
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
