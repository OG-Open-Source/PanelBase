package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/gin-gonic/gin"
)

// Removed local constants, use exported constants from auth.go
// const userPermissionsKey = "userPermissions"
// const userIDKey = "userID"

// Helper function to extract ID from request body
// Note: This reads the body, so it might interfere if the handler needs to read it again.
// Consider alternative ways like binding to a struct with only ID or passing parsed body.
func extractIDFromRequestBody(c *gin.Context) (string, bool) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		// Handle error, maybe log it. Cannot determine ID.
		return "", false
	}
	// IMPORTANT: Restore the body so the handler can read it later
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var data map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		// Handle error, not valid JSON or doesn't fit map.
		return "", false
	}

	if idVal, ok := data["id"]; ok {
		if idStr, ok := idVal.(string); ok {
			return idStr, true
		}
	}
	return "", false
}

// checkBasicPermission is an internal helper performing the core permission check.
func checkBasicPermission(c *gin.Context, resource string, requiredAction string) bool {
	permissionsVal, exists := c.Get(ContextKeyScopes) // Use exported constant
	if !exists {
		// Log or handle the error appropriately. Avoid fmt.Printf in production.
		// log.Printf("Error: Permissions not found in context for user %s", c.GetString(ContextKeyUserID))
		server.ErrorResponse(c, http.StatusInternalServerError, "User permissions not found in context")
		c.Abort()
		return false
	}

	userPermissions, ok := permissionsVal.(models.UserPermissions)
	if !ok {
		// log.Printf("Error: Invalid permissions format in context for user %s (%T)", c.GetString(ContextKeyUserID), permissionsVal)
		server.ErrorResponse(c, http.StatusInternalServerError, "Invalid user permissions format")
		c.Abort()
		return false
	}

	actions, resourceExists := userPermissions[resource]
	// Remove wildcard resource check
	// wildcardActions, wildcardResourceExists := userPermissions["*"]

	if !resourceExists /* && !wildcardResourceExists */ {
		// Update error message to only mention the specific resource
		server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied: No permissions defined for resource '%s'", resource))
		c.Abort()
		return false
	}

	// Remove logic combining or checking wildcard resource actions

	// Check only the specific resource actions for an exact match
	if resourceExists { // Technically redundant now, as !resourceExists causes early return
		for _, allowedAction := range actions {
			if allowedAction == requiredAction /* || allowedAction == "*" */ { // Remove wildcard action check
				return true // Permission granted
			}
		}
	}

	// Remove second loop checking wildcard actions

	// If loop completes without finding the exact action
	server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied: Action '%s' not allowed for resource '%s'", requiredAction, resource))
	c.Abort()
	return false
}

// CheckPermission checks non-read permissions (create, update, delete, execute, install).
// For read operations, use CheckReadPermission.
func CheckPermission(c *gin.Context, resource string, requiredAction string) bool {
	// For now, it directly calls the basic check. Can add more logic later if needed.
	return checkBasicPermission(c, resource, requiredAction)
}

// CheckReadPermission handles the specific logic for read operations, differentiating
// between listing resources ('read:list') and accessing a specific item ('read:item').
// It checks if an 'id' is provided in the request body to determine the required action.
func CheckReadPermission(c *gin.Context, resource string) bool {
	// Determine if an ID is provided in the request body
	targetID, idProvided := extractIDFromRequestBody(c)

	if idProvided {
		// Request is for a specific item
		requiredAction := "read:item"
		// Check basic permission first
		if !checkBasicPermission(c, resource, requiredAction) {
			return false
		}

		// Now, check if the user can read *this specific* item
		loggedInUserIDVal, exists := c.Get(ContextKeyUserID) // Use exported constant
		if !exists {
			// log.Printf("Error: UserID not found in context during read item check")
			server.ErrorResponse(c, http.StatusInternalServerError, "User ID not found in context")
			c.Abort()
			return false
		}
		loggedInUserID, ok := loggedInUserIDVal.(string)
		if !ok {
			// log.Printf("Error: Invalid UserID format in context (%T)", loggedInUserIDVal)
			server.ErrorResponse(c, http.StatusInternalServerError, "Invalid UserID format in context")
			c.Abort()
			return false
		}

		// If the target ID is the user's own ID, 'read:item' is sufficient.
		if targetID == loggedInUserID {
			return true // Allowed to read own item
		}

		// If the target ID is different, the user needs 'read:list' permission
		// to view *other* users' specific items.
		if checkBasicPermission(c, resource, "read:list") {
			return true // Has list permission, can read specific other item
		} else {
			server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Permission denied: Action 'read:item' on resource '%s' requires 'read:list' permission to view other items", resource))
			c.Abort()
			return false
		}

	} else {
		// Request is for the list
		requiredAction := "read:list"
		return checkBasicPermission(c, resource, requiredAction)
	}
}

// TODO: Refine extractIDFromRequestBody - reading body here might be problematic.
// Consider requiring handlers to parse the body and pass the ID to permission checks.
