package middleware

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/gin-gonic/gin"
)

const (
	AuthorizationHeaderKey = "Authorization"
	AuthTypeBearer         = "bearer"
	ContextUserClaimsKey   = "userClaims"
)

// RequireAuth creates a middleware function that validates JWT and checks permissions/scopes.
func RequireAuth(jwtSecret string, requiredPermissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader(AuthorizationHeaderKey)
		if len(authHeader) == 0 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is missing"})
			return
		}

		fields := strings.Fields(authHeader)
		if len(fields) < 2 || !strings.EqualFold(fields[0], AuthTypeBearer) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			return
		}

		accessToken := fields[1]
		claims, err := auth.ValidateToken(accessToken, jwtSecret)
		if err != nil {
			log.Printf("DEBUG: Token validation failed: %v", err) // Log details for debugging
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			return
		}

		// Store claims in context for later use
		c.Set(ContextUserClaimsKey, claims)

		// Check permissions if any are required
		if len(requiredPermissions) > 0 {
			if !hasRequiredScopes(claims.Scopes, requiredPermissions) {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
				return
			}
		}

		c.Next()
	}
}

// hasRequiredScopes checks if the user has all the required permissions/scopes.
// Permissions can be simple strings ("admin") or scope:action ("users:create").
func hasRequiredScopes(userScopes map[string][]string, requiredPermissions []string) bool {
	// Build a set of all permissions the user has for quick lookup.
	userPermsSet := make(map[string]struct{})
	if userScopes != nil {
		for scope, actions := range userScopes {
			// Add the scope itself as a permission (e.g., having "users" scope implies basic access)
			userPermsSet[scope] = struct{}{}
			for _, action := range actions {
				// Add scope:action permission
				userPermsSet[fmt.Sprintf("%s:%s", scope, action)] = struct{}{}
				// Optionally, add just the action as a global permission if needed?
				// userPermsSet[action] = struct{}{}
			}
		}
	}
	// Allow a global "admin" scope/permission to bypass other checks
	if _, isAdmin := userPermsSet["admin"]; isAdmin {
		return true
	}

	for _, reqPerm := range requiredPermissions {
		if _, ok := userPermsSet[reqPerm]; !ok {
			log.Printf("DEBUG: Permission check failed. User missing required permission: %s", reqPerm)
			return false // Missing a required permission
		}
	}
	return true
}
 