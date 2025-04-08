package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

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

// CheckPermission is a helper function to check if the user has the required permission.
// It returns true if allowed, false otherwise.
// It also handles sending the 403 response if permission is denied.
func CheckPermission(c *gin.Context, resource string, requiredAction string) bool {
	// Use string(Constant) to get the key for context lookup
	permissions, exists := c.Get(string(ContextKeyPermissions))
	if !exists {
		server.ErrorResponse(c, http.StatusInternalServerError, "User permissions not found in context")
		return false
	}

	userPermissions, ok := permissions.(models.UserPermissions)
	if !ok {
		server.ErrorResponse(c, http.StatusInternalServerError, "Invalid user permissions format")
		return false
	}

	// Use string(Constant) for user ID lookup as well
	_, userExists := c.Get(string(ContextKeyUserID))
	if !userExists {
		server.ErrorResponse(c, http.StatusInternalServerError, "Authenticated User ID not found in context")
		return false
	}

	// Extract the actual action from the requiredAction string (e.g., "create" from "api:create")
	actionParts := strings.Split(requiredAction, ":")
	actualAction := requiredAction // Default to the full string if no colon
	if len(actionParts) > 1 {
		actualAction = strings.Join(actionParts[1:], ":") // Handle actions like "read:list:all"
	}

	allowedActions, resourceExists := userPermissions[resource]

	if !resourceExists {
		server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied: Resource '%s' access not configured", resource))
		return false
	}

	// Check if the actual action is in the allowed list for the resource
	for _, allowedAction := range allowedActions {
		if allowedAction == actualAction {
			return true // Permission granted
		}
	}

	// If loop finishes, the action was not found
	server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied: Action '%s' not allowed for resource '%s'", requiredAction, resource))
	return false
}

// RequirePermission returns a Gin middleware function that checks for the specified permission.
func RequirePermission(resource string, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !CheckPermission(c, resource, action) {
			c.Abort() // Stop further processing if permission denied
			return
		}
		c.Next() // Continue processing if permission granted
	}
}

// CacheRequestBody reads the request body and stores it in the context.
// ... rest of CacheRequestBody ...
