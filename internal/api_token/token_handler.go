package api_token

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/OG-Open-Source/PanelBase/internal/tokenstore"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/gin-gonic/gin"
)

// Constants
// const AudienceAPI = "api" // Removed: Defined in token_service.go

// Struct to bind optional ID from GET request body
type GetTokenPayload struct {
	ID string `json:"id,omitempty"` // Optional: ID (JTI) of the specific token to fetch
	// Username string `json:"username,omitempty"` // Removed: Username for GET should be query param if needed, but we handle via permissions
}

// UpdateTokenPayload defines the structure for updating token metadata.
// Name, Description, and Scopes can be updated.
// Username is optional for admin actions.
type UpdateTokenPayload struct {
	ID          string                 `json:"id" binding:"required"`
	Username    *string                `json:"username,omitempty"` // Optional: Target username for admin actions
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Scopes      models.UserPermissions `json:"scopes,omitempty"`
}

// DeleteTokenPayload defines the structure for deleting a token.
// Username is optional for admin actions.
type DeleteTokenPayload struct {
	TokenID  string  `json:"token_id" binding:"required"`
	Username *string `json:"username,omitempty"` // Optional: Target username for admin actions
}

// Helper to get target user ID and check admin permission if needed
func getTargetUserID(c *gin.Context, requestedUsername *string) (targetUserID string, isAdminAction bool, err error) {
	currentUserID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		return "", false, fmt.Errorf("current user ID not found in context")
	}
	currentUserIDStr := currentUserID.(string)

	if requestedUsername != nil && *requestedUsername != "" {
		// Admin action requested
		targetUser, userExists, userErr := user.GetUserByUsername(*requestedUsername)
		if userErr != nil {
			return "", true, fmt.Errorf("failed to lookup target user '%s': %w", *requestedUsername, userErr)
		}
		if !userExists {
			return "", true, fmt.Errorf("target user '%s' not found", *requestedUsername)
		}
		// If target is self, treat as non-admin action for permission check later
		if targetUser.ID == currentUserIDStr {
			return currentUserIDStr, false, nil
		}
		return targetUser.ID, true, nil
	} else {
		// Action is for the current user
		return currentUserIDStr, false, nil
	}
}

// CreateTokenHandler handles creating an API token, potentially for another user if admin.
func CreateTokenHandler(c *gin.Context) {
	var payload models.CreateAPITokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// Determine target user and check if it's an admin action
	targetUserID, isAdminAction, err := getTargetUserID(c, payload.Username)
	if err != nil {
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "not found") {
			statusCode = http.StatusNotFound
		} else if strings.Contains(err.Error(), "context") {
			statusCode = http.StatusUnauthorized
		}
		server.ErrorResponse(c, statusCode, err.Error())
		return
	}

	// Check permission
	requiredAction := "create"
	if isAdminAction {
		requiredAction = "create:all"
	}
	if !middleware.CheckPermission(c, "api", requiredAction) {
		return
	}

	// Load the *target* user data (needed for JWT secret and scope validation)
	targetUserInstance, targetUserExists, err := user.GetUserByID(targetUserID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load target user data: "+err.Error())
		return
	}
	if !targetUserExists {
		// Should be unlikely if getTargetUserID worked, but check anyway
		server.ErrorResponse(c, http.StatusNotFound, "Target user not found after permission check")
		return
	}

	// Call the token service with the target user and payload
	tokenID, apiTokenMeta, signedTokenString, err := CreateAPIToken(targetUserInstance, payload)
	if err != nil {
		if strings.Contains(err.Error(), "exceed user permissions") || strings.Contains(err.Error(), "invalid duration format") {
			server.ErrorResponse(c, http.StatusBadRequest, err.Error())
		} else if strings.Contains(err.Error(), "failed to store token metadata") {
			log.Printf("ERROR: Failed to store token metadata for user %s: %v", targetUserID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to store token metadata")
		} else {
			log.Printf("ERROR: Failed to create API token for user %s: %v", targetUserID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to create API token: "+err.Error())
		}
		return
	}

	// Prepare and return the success response
	responseData := map[string]interface{}{
		"id":          tokenID,
		"name":        apiTokenMeta.Name,
		"description": apiTokenMeta.Description,
		"scopes":      apiTokenMeta.Scopes,
		"created_at":  apiTokenMeta.CreatedAt.Time().Format(time.RFC3339),
		"expires_at":  apiTokenMeta.ExpiresAt.Time().Format(time.RFC3339),
		"token":       signedTokenString,
	}
	server.SuccessResponse(c, "API token created successfully", responseData)
}

// GetTokensHandler handles retrieving API token metadata.
// Reads optional 'id' from the JSON request body.
// - If no 'id' is provided, lists tokens for the current user (requires 'api:read:list').
// - If 'id' is provided:
//   - If the token belongs to the current user, requires 'api:read:item'.
//   - If the token belongs to another user, requires 'api:read:item:all'.
func GetTokensHandler(c *gin.Context) {
	// 1. Get current User ID from authenticated context
	userIDVal, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		server.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	currentUserIDStr := userIDVal.(string)

	// 2. Try to bind optional payload from request body
	var payload GetTokenPayload
	// Allow empty body (EOF) or binding errors if no body is sent for list action
	if err := c.ShouldBindJSON(&payload); err != nil && err.Error() != "EOF" {
		// If there's a body but it's malformed JSON
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload format: "+err.Error())
		return
	}
	targetTokenID := payload.ID // Get ID from body payload

	// 3. Determine action based on presence of targetTokenID
	if targetTokenID != "" {
		// --- Get Specific Token ---

		// 3a. Get token info from store FIRST
		tokenInfo, found, err := tokenstore.GetTokenInfo(targetTokenID)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token info: "+err.Error())
			return
		}
		if !found {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("API token with ID '%s' not found", targetTokenID))
			return
		}

		// 3b. Check ownership and permissions
		isOwner := tokenInfo.UserID == currentUserIDStr
		requiredAction := "read:item"
		if !isOwner {
			requiredAction = "read:item:all"
		}
		if !middleware.CheckPermission(c, "api", requiredAction) {
			return
		}

		// 3c. Check if revoked (after permission check)
		isRevoked, err := tokenstore.IsTokenRevoked(targetTokenID)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to check token revocation status: "+err.Error())
			return
		}
		if isRevoked {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("API token with ID '%s' not found (or revoked)", targetTokenID))
			return
		}

		// 3d. Prepare and return response
		responseData := map[string]interface{}{
			"id":         targetTokenID,
			"name":       tokenInfo.Name,
			"scopes":     tokenInfo.Scopes,
			"created_at": tokenInfo.CreatedAt.Time().Format(time.RFC3339),
			"expires_at": tokenInfo.ExpiresAt.Time().Format(time.RFC3339),
		}
		server.SuccessResponse(c, "API token retrieved successfully", responseData)

	} else {
		// --- Get Token List (for current user) ---
		requiredAction := "read:list"
		if !middleware.CheckPermission(c, "api", requiredAction) {
			return
		}

		// Get tokens for the currentUserIDStr
		tokensInfo, tokenIDs, err := tokenstore.GetUserTokens(currentUserIDStr)
		if err != nil {
			log.Printf("Error retrieving tokens for user %s: %v", currentUserIDStr, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve tokens")
			return
		}

		tokenList := make([]map[string]interface{}, len(tokensInfo))
		for i, info := range tokensInfo {
			tokenList[i] = map[string]interface{}{
				"id":         tokenIDs[i],
				"name":       info.Name,
				"scopes":     info.Scopes,
				"created_at": info.CreatedAt.Time().Format(time.RFC3339),
				"expires_at": info.ExpiresAt.Time().Format(time.RFC3339),
			}
		}

		server.SuccessResponse(c, "API tokens retrieved successfully", tokenList)
	}
}

// UpdateTokenHandler handles updating API token metadata, supporting admin actions.
func UpdateTokenHandler(c *gin.Context) {
	var payload UpdateTokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	currentUserID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		server.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
		return
	}
	currentUserIDStr := currentUserID.(string)

	// 1. Get Token Info FIRST to determine ownership
	targetTokenInfo, found, err := tokenstore.GetTokenInfo(payload.ID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve current token info: "+err.Error())
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("API token with ID '%s' not found", payload.ID))
		return
	}

	// 2. Determine target user and required permission
	targetUserID := targetTokenInfo.UserID
	isOwner := targetUserID == currentUserIDStr
	isExplicitAdminAction := payload.Username != nil && *payload.Username != ""
	var requiredAction string
	var effectiveTargetUser models.User // User whose context we operate under
	var targetUserExists bool

	if isExplicitAdminAction {
		// Admin explicitly specified target user via payload.Username
		lookupUsername := *payload.Username
		targetUserLookup, userLookupExists, userLookupErr := user.GetUserByUsername(lookupUsername)
		if userLookupErr != nil {
			server.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Failed to lookup target user '%s': %v", lookupUsername, userLookupErr))
			return
		}
		if !userLookupExists {
			server.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Target user '%s' not found", lookupUsername))
			return
		}

		// Verify the specified username matches the token's owner ID
		if targetUserLookup.ID != targetUserID {
			server.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Mismatch: Token '%s' belongs to user ID '%s', not specified username '%s' (ID '%s')", payload.ID, targetUserID, lookupUsername, targetUserLookup.ID))
			return
		}

		requiredAction = "update:all"
		effectiveTargetUser = targetUserLookup
		targetUserExists = true
	} else {
		// No username specified, assume action on own token OR admin implicit action
		if isOwner {
			requiredAction = "update" // Standard self-update
			effectiveTargetUser, targetUserExists, err = user.GetUserByID(currentUserIDStr)
			if err != nil {
				server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data for scope validation: "+err.Error())
				return
			}
			if !targetUserExists {
				server.ErrorResponse(c, http.StatusNotFound, "Authenticated user not found for scope validation")
				return
			}
		} else {
			// Trying to update someone else's token without specifying username? Requires admin perm.
			requiredAction = "update:all"
			// We need the *owner's* user data for scope validation
			effectiveTargetUser, targetUserExists, err = user.GetUserByID(targetUserID)
			if err != nil {
				server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data for scope validation: "+err.Error())
				return
			}
			if !targetUserExists {
				server.ErrorResponse(c, http.StatusNotFound, "Target user not found for scope validation")
				return
			}
		}
	}

	// 3. Check Permission
	if !middleware.CheckPermission(c, "api", requiredAction) {
		return
	}

	// 4. Check if revoked (can't update revoked)
	isRevoked, err := tokenstore.IsTokenRevoked(payload.ID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to check token revocation status: "+err.Error())
		return
	}
	if isRevoked {
		server.ErrorResponse(c, http.StatusConflict, "Cannot update a revoked token")
		return
	}

	// 5. Prepare updates and validate scopes (using effectiveTargetUser)
	updatedInfo := targetTokenInfo
	needsResave := false
	if payload.Name != nil {
		updatedInfo.Name = *payload.Name
		needsResave = true
	}
	// TODO: Description update
	if payload.Scopes != nil && len(payload.Scopes) > 0 {
		if !validateScopesHelper(payload.Scopes, effectiveTargetUser.Scopes) {
			server.ErrorResponse(c, http.StatusBadRequest, "Requested scopes exceed target user permissions")
			return
		}
		updatedInfo.Scopes = payload.Scopes
		needsResave = true
	}
	if !needsResave {
		server.ErrorResponse(c, http.StatusBadRequest, "No updateable fields provided (name, description, scopes allowed)")
		return
	}

	// 6. Save updated info back to tokenstore
	if err := tokenstore.StoreToken(payload.ID, updatedInfo); err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to update token metadata: "+err.Error())
		return
	}

	// 7. Return success response
	responseData := map[string]interface{}{
		"id":         payload.ID,
		"name":       updatedInfo.Name,
		"scopes":     updatedInfo.Scopes,
		"created_at": updatedInfo.CreatedAt.Time().Format(time.RFC3339),
		"expires_at": updatedInfo.ExpiresAt.Time().Format(time.RFC3339),
	}
	server.SuccessResponse(c, "API token metadata updated successfully", responseData)
}

// DeleteTokenHandler handles deleting (revoking) an API token, supporting admin actions.
func DeleteTokenHandler(c *gin.Context) {
	var payload DeleteTokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	currentUserID, exists := c.Get(middleware.ContextKeyUserID)
	if !exists {
		server.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
		return
	}
	currentUserIDStr := currentUserID.(string)

	// 1. Get Token Info FIRST to determine ownership
	targetTokenInfo, found, err := tokenstore.GetTokenInfo(payload.TokenID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Error checking token ownership: "+err.Error())
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, "Token not found")
		return
	}

	// 2. Determine required permission based on ownership and explicit admin request
	targetUserID := targetTokenInfo.UserID
	isOwner := targetUserID == currentUserIDStr
	isExplicitAdminAction := payload.Username != nil && *payload.Username != ""
	var requiredAction string

	if isExplicitAdminAction {
		// Admin explicitly specified target user via payload.Username
		lookupUsername := *payload.Username
		targetUserLookup, userLookupExists, userLookupErr := user.GetUserByUsername(lookupUsername)
		if userLookupErr != nil {
			server.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Failed to lookup target user '%s': %v", lookupUsername, userLookupErr))
			return
		}
		if !userLookupExists {
			server.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Target user '%s' not found", lookupUsername))
			return
		}

		// Verify the specified username matches the token's owner ID
		if targetUserLookup.ID != targetUserID {
			server.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Mismatch: Token '%s' belongs to user ID '%s', not specified username '%s' (ID '%s')", payload.TokenID, targetUserID, lookupUsername, targetUserLookup.ID))
			return
		}
		requiredAction = "delete:all"
	} else {
		if isOwner {
			requiredAction = "delete"
		} else {
			// Trying to delete someone else's token without specifying username? Requires admin perm.
			requiredAction = "delete:all"
		}
	}

	// 3. Check Permission
	if !middleware.CheckPermission(c, "api", requiredAction) {
		return
	}

	// 4. Call tokenstore to revoke the token
	if err := tokenstore.RevokeToken(payload.TokenID); err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to revoke token: "+err.Error())
		return
	}

	// 5. Return success response
	server.SuccessResponse(c, "API token revoked successfully", nil)
}

// validateScopesHelper is a temporary helper until validation logic is consolidated.
// TODO: Remove this and use the service's validation logic directly or indirectly.
func validateScopesHelper(requested models.UserPermissions, userScopes models.UserPermissions) bool {
	for resource, reqActions := range requested {
		userActions, ok := userScopes[resource]
		if !ok {
			return false // User doesn't have access to this resource at all
		}
		userActionSet := make(map[string]bool)
		for _, action := range userActions {
			userActionSet[action] = true
		}
		for _, reqAction := range reqActions {
			if !userActionSet[reqAction] {
				return false // Requested action not allowed for the user on this resource
			}
		}
	}
	return true
}
