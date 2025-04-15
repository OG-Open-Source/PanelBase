package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/OG-Open-Source/PanelBase/pkg/response"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeaderKey = "Authorization"
	AuthTypeBearer         = "bearer"
	UserClaimsKey          = "userClaims"
	UserIDKey              = "userID"
)

// Helper function to recursively check for a scope within the nested map.
func hasScope(userScopes map[string]interface{}, requiredScope string) bool {
	if userScopes == nil {
		return false
	}

	parts := strings.Split(requiredScope, ":")
	currentLevel := userScopes

	for i, part := range parts {
		value, ok := currentLevel[part]
		if !ok {
			return false // Path does not exist
		}

		if i == len(parts)-1 {
			// Last part, should be the action (string or list of strings)
			switch v := value.(type) {
			case string:
				return v == "*" // Check for wildcard action at the end of the path
			case []interface{}: // JSON arrays decode to []interface{}
				for _, item := range v {
					if strAction, ok := item.(string); ok && strAction == "*" {
						return true // Wildcard action found in list
					}
				}
				return false // Action not found or no wildcard
			case []string: // Handle case where it might already be []string
				for _, action := range v {
					if action == "*" {
						return true // Wildcard action found
					}
				}
				return false
			default:
				return false // Unexpected type at the end
			}
		} else {
			// Not the last part, should be a map for the next level
			nextLevel, ok := value.(map[string]interface{})
			if !ok {
				// Check for wildcard scope higher up the chain
				switch v := value.(type) {
				case string:
					if v == "*" {
						return true
					}
				case []interface{}:
					for _, item := range v {
						if strAction, ok := item.(string); ok && strAction == "*" {
							return true
						}
					}
				case []string:
					for _, action := range v {
						if action == "*" {
							return true
						}
					}
				}
				return false // Path part exists but is not a map and not a wildcard
			}
			currentLevel = nextLevel
		}
	}
	// Should not be reached if requiredScope has parts, but return false just in case.
	return false
}

// HasAction checks if the user scopes contain the required action within the scope path.
func HasAction(userScopes map[string]interface{}, requiredScopePath string, requiredAction string) bool {
	if userScopes == nil {
		return false
	}

	parts := strings.Split(requiredScopePath, ":")
	currentLevel := userScopes

	for i, part := range parts {
		value, ok := currentLevel[part]
		if !ok {
			// Check for wildcard at this level before declaring path non-existent
			if wildcardValue, wildcardOk := currentLevel["*"]; wildcardOk {
				// If wildcard is just "*", grant access
				if ws, ok := wildcardValue.(string); ok && ws == "*" {
					return true
				}
				// If wildcard is a list containing "*", grant access
				if wList, ok := wildcardValue.([]interface{}); ok {
					for _, item := range wList {
						if sItem, ok := item.(string); ok && sItem == "*" {
							return true
						}
					}
				} else if wStrList, ok := wildcardValue.([]string); ok {
					for _, action := range wStrList {
						if action == "*" {
							return true
						}
					}
				}
			}
			return false // Path does not exist and no applicable wildcard
		}

		if i == len(parts)-1 {
			// Reached the target resource level. Now check actions.
			switch actions := value.(type) {
			case string:
				// If a string, check if it's the action or a wildcard
				return actions == requiredAction || actions == "*"
			case []interface{}:
				for _, item := range actions {
					if strAction, ok := item.(string); ok {
						if strAction == requiredAction || strAction == "*" {
							return true // Action or wildcard found
						}
					}
				}
				return false // Action not found in list
			case []string: // Handle case where it might already be []string
				for _, action := range actions {
					if action == requiredAction || action == "*" {
						return true // Action or wildcard found
					}
				}
				return false
			default:
				return false // Unexpected type at the action level
			}
		} else {
			// Intermediate level, must be a map or wildcard string/list
			switch v := value.(type) {
			case map[string]interface{}:
				currentLevel = v
			case string:
				if v == "*" {
					return true // Wildcard at intermediate level grants access
				}
				return false // String that is not a wildcard at intermediate level blocks path
			default:
				// Check for wildcard in list at intermediate level
				if actionsList, ok := value.([]interface{}); ok {
					for _, item := range actionsList {
						if strAction, ok := item.(string); ok && strAction == "*" {
							return true // Wildcard found
						}
					}
				} else if strActionsList, ok := value.([]string); ok {
					for _, action := range strActionsList {
						if action == "*" {
							return true // Wildcard found
						}
					}
				}
				return false // Not a map and not a wildcard string/list
			}
		}
	}
	return false // Should not be reached
}

// RequireAuth creates a middleware function that validates the JWT.
func RequireAuth(jwtSecret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if len(authHeader) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("Authorization header is missing", nil))
			return
		}

		fields := strings.Fields(authHeader)
		if len(fields) < 2 || !strings.EqualFold(fields[0], AuthTypeBearer) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("Invalid authorization header format", nil))
			return
		}

		accessToken := fields[1]
		claims, err := auth.ValidateToken(accessToken, jwtSecret, auth.TokenTypeAPI)
		if err != nil {
			log.Printf("DEBUG: Token validation failed: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("Invalid or expired token", nil))
			return
		}

		// Extract Subject (User ID) and store it in the context
		userID := claims.Subject
		if userID == "" {
			log.Printf("ERROR: Valid token received but Subject (UserID) is empty.")
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("Invalid token claims (missing user ID)", nil))
				return
		}
		c.Set(UserIDKey, userID)

		// Store the full claims object as well, needed by RequireScope and handlers
		c.Set(UserClaimsKey, claims)

		c.Next()
	}
}

// RequireScope creates a middleware function that checks if the authenticated user
// (whose claims should already be in the context via RequireAuth) has the required scope.
// The requiredScope format is colon-separated, e.g., "users:read" or "account:profile:update".
func RequireScope(requiredScope string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Retrieve claims from context
		claimsValue, exists := c.Get(UserClaimsKey)
		if !exists {
			log.Printf("ERROR: User claims not found in context. Ensure RequireAuth runs before RequireScope.")
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error", nil))
			return
		}

		claims, ok := claimsValue.(*auth.Claims)
		if !ok || claims == nil {
			log.Printf("ERROR: Invalid claims type or nil claims in context.")
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error", nil))
			return
		}

		// Separate path and action
		parts := strings.Split(requiredScope, ":")
		if len(parts) < 2 {
			log.Printf("ERROR: Invalid requiredScope format: %s", requiredScope)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Internal server configuration error (invalid scope format)", nil))
			return
		}

		requiredAction := parts[len(parts)-1]
		requiredScopePath := strings.Join(parts[:len(parts)-1], ":")

		// Check permission
		if !HasAction(claims.Scopes, requiredScopePath, requiredAction) {
			log.Printf("DEBUG: Permission check failed for scope '%s:%s'. User scopes: %v", requiredScopePath, requiredAction, claims.Scopes)
			c.AbortWithStatusJSON(http.StatusForbidden, response.Failure("Insufficient permissions", nil))
			return
		}

		c.Next()
	}
}
 