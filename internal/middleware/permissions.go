package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/gin-gonic/gin"
)

// Constants for context keys are now defined in context_keys.go

// Helper function to extract ID from request body (Potentially problematic, consider alternatives)
func extractIDFromRequestBody(c *gin.Context) (string, bool) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return "", false
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes)) // Restore body

	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return "", false
	}
	if idVal, ok := data["id"]; ok {
		if idStr, ok := idVal.(string); ok {
			return idStr, true
		}
	}
	return "", false
}

// checkBasicPermission verifies if the user associated with the context
// has the required action permission for the specified resource.
func checkBasicPermission(c *gin.Context, resource string, requiredAction string) bool {
	permissionsVal, exists := c.Get(ContextKeyScopes)
	if !exists {
		log.Printf("[DEBUG PERM CHECK] ContextKeyScopes not found in context for user %s", c.GetString(ContextKeyUserID))
		server.ErrorResponse(c, http.StatusInternalServerError, "User permissions not found in context")
		c.Abort()
		return false
	}

	userPermissions, ok := permissionsVal.(models.UserPermissions)
	if !ok {
		log.Printf("[DEBUG PERM CHECK] Invalid permissions format in context for user %s. Expected models.UserPermissions, got %T. Value: %+v", c.GetString(ContextKeyUserID), permissionsVal, permissionsVal)
		server.ErrorResponse(c, http.StatusInternalServerError, "Invalid user permissions format")
		c.Abort()
		return false
	}

	// Log the permissions being checked against
	log.Printf("[DEBUG PERM CHECK] User: %s, Resource: %s, Required Action: %s, User Permissions: %+v", c.GetString(ContextKeyUserID), resource, requiredAction, userPermissions)

	actions, resourceExists := userPermissions[resource]
	if !resourceExists {
		log.Printf("[DEBUG PERM CHECK] Result: DENIED - Resource '%s' not found in user permissions.", resource)
		server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied: No permissions defined for resource '%s'", resource))
		c.Abort()
		return false
	}

	// Check for the exact required action within the resource's actions
	for _, allowedAction := range actions {
		if allowedAction == requiredAction {
			log.Printf("[DEBUG PERM CHECK] Result: GRANTED - Found matching action '%s' for resource '%s'.", requiredAction, resource)
			return true // Permission granted
		}
	}

	// If loop completes without finding the exact action
	log.Printf("[DEBUG PERM CHECK] Result: DENIED - Action '%s' not found in allowed actions [%+v] for resource '%s'.", requiredAction, actions, resource)
	server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied: Action '%s' not allowed for resource '%s'", requiredAction, resource))
	c.Abort()
	return false
}

// CheckPermission checks if the user has the specified permission scope.
// It's a simple wrapper around checkBasicPermission.
// Ownership checks must be performed by the handler.
func CheckPermission(c *gin.Context, resource string, requiredAction string) bool {
	return checkBasicPermission(c, resource, requiredAction)
}

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

// CacheRequestBody reads the request body and stores it in the context.
// ... rest of CacheRequestBody ...
