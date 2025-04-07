package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	logger "github.com/OG-Open-Source/PanelBase/internal/logging"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/gin-gonic/gin"
)

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

// CheckPermission checks if the user has the required permission for a resource and action.
// It retrieves user permissions from the Gin context (set by AuthMiddleware).
// requiredPermission format: "resource:action" or "resource:action:scope" (e.g., "users:read:all")
func CheckPermission(c *gin.Context, resource string, requiredPermission string) bool {
	permissionsVal, exists := c.Get(string(ContextKeyPermissions))
	if !exists {
		logger.ErrorPrintf("PERMISSIONS", "CHECK_NO_CTX", "Permissions not found in context for user trying to access %s with %s", resource, requiredPermission)
		server.ErrorResponse(c, http.StatusInternalServerError, "Permissions context missing")
		return false
	}

	userPermissions, ok := permissionsVal.(models.UserPermissions)
	if !ok {
		logger.ErrorPrintf("PERMISSIONS", "CHECK_INVALID_CTX", "Invalid permissions format in context for user trying to access %s with %s", resource, requiredPermission)
		server.ErrorResponse(c, http.StatusInternalServerError, "Invalid permissions context format")
		return false
	}

	// Extract user ID for logging purposes
	userIDVal, _ := c.Get(string(ContextKeyUserID))
	userID := "[unknown]"
	if uidStr, okUid := userIDVal.(string); okUid {
		userID = uidStr
	}

	// Split required permission into parts (e.g., "api:read:list:all" -> ["api", "read", "list", "all"])
	// We primarily care about the resource (first part) and the full permission string
	requiredParts := strings.Split(requiredPermission, ":")
	if len(requiredParts) < 2 {
		logger.ErrorPrintf("PERMISSIONS", "CHECK_INVALID_REQ", "Invalid required permission format: %s (must be resource:action[:scope])", requiredPermission)
		server.ErrorResponse(c, http.StatusInternalServerError, "Internal permission format error")
		return false
	}
	// Note: We use the *full* requiredPermission string for matching, not just the first two parts

	logger.DebugPrintf("PERMISSIONS", "CHECK", "User: %s, Resource: %s, Required: %s (Checking: %s), User Perms: %+v", userID, resource, requiredPermission, requiredPermission, userPermissions)

	// Get the list of actions allowed for the specific resource
	allowedActions, resourceFound := userPermissions[resource]
	if !resourceFound {
		logger.DebugPrintf("PERMISSIONS", "CHECK", "Result: DENIED - Resource '%s' not found in user permissions.", resource)
		server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Access denied to resource: %s", resource))
		return false
	}

	// Check if the required action (full string) exists in the allowed actions
	for _, allowedAction := range allowedActions {
		if allowedAction == requiredPermission {
			logger.DebugPrintf("PERMISSIONS", "CHECK", "Result: ALLOWED - Action '%s' found for '%s'.", requiredPermission, resource)
			return true // Permission granted
		}
	}

	// If the loop finishes without finding the permission
	logger.DebugPrintf("PERMISSIONS", "CHECK", "Result: DENIED - Required action '%s' not found in allowed actions for '%s'.", requiredPermission, resource)
	server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied for action: %s", requiredPermission))
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
