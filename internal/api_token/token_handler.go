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
	"github.com/OG-Open-Source/PanelBase/internal/token_store"
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

// Common request structure for identifying a token, potentially for a specific user (admin)
type targetTokenRequest struct {
	ID     string `json:"id" binding:"required"`
	UserID string `json:"user_id"` // Optional: For admin actions targeting a specific user
}

// CreateTokenRequest defines the expected JSON body for creating a token.
type CreateTokenRequest struct {
	UserID      string   `json:"user_id"` // Optional: For admin actions targeting a specific user
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description"` // Optional
	Duration    string   `json:"duration"`    // Optional: ISO 8601 Duration string (e.g., "P30D")
	Scopes      []string `json:"scopes"`      // Optional: List of specific scope strings
}

// UpdateTokenRequest defines the expected JSON body for updating a token.
// Uses targetTokenRequest for identification.
type UpdateTokenRequest struct {
	targetTokenRequest
	Name        *string `json:"name"`        // Use pointers to distinguish between empty and not provided
	Description *string `json:"description"` // Use pointers
	// Scopes update via PATCH might be complex, omitted for now
}

// DeleteTokenRequest defines the expected JSON body for deleting a token.
// Uses targetTokenRequest for identification.
type DeleteTokenRequest struct {
	targetTokenRequest
}

// GetTokensRequest defines the optional JSON body for getting tokens (admin or specific token).
// Deprecated for GET, use query parameters instead.
/*
type GetTokensRequest struct {
	UserID  string `json:"user_id"`  // Optional: For admin actions targeting a specific user
	TokenID string `json:"token_id"` // Optional: To get a specific token by ID
}
*/

// CreateTokenHandler handles creating an API token, potentially for another user if admin.
func CreateTokenHandler(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)
	targetUserID := req.UserID
	var requiredPermission string
	if targetUserID != "" && targetUserID != requestingUserID {
		requiredPermission = "api:create:all"
	} else {
		targetUserID = requestingUserID
		requiredPermission = "api:create"
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// Fetch the target user data
	targetUserInstance, userExists, err := user.GetUserByID(targetUserID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load target user data: "+err.Error())
		return
	}
	if !userExists {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
		return
	}

	// Prepare payload for the service function
	// Note: Scope validation happens inside CreateAPIToken
	createPayload := models.CreateAPITokenPayload{
		Name:        req.Name,
		Description: req.Description,
		Duration:    req.Duration,                                 // Service will handle default/validation
		Scopes:      models.ScopeStringsToPermissions(req.Scopes), // Convert []string to map[string][]string
	}

	// Call the service function to create the token
	tokenID, _, signedToken, err := CreateAPIToken(targetUserInstance, createPayload)
	if err != nil {
		// Handle specific errors from the service
		if strings.Contains(err.Error(), "exceed user permissions") {
			server.ErrorResponse(c, http.StatusBadRequest, "Requested scopes exceed user permissions")
		} else if strings.Contains(err.Error(), "duration is required") {
			server.ErrorResponse(c, http.StatusBadRequest, "Token duration is required")
		} else if strings.Contains(err.Error(), "invalid duration format") {
			server.ErrorResponse(c, http.StatusBadRequest, "Invalid duration format: "+err.Error())
		} else {
			log.Printf("%s Error creating API token for user %s: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to create API token: "+err.Error())
		}
		return
	}

	// Return the actual token ID and JWT string
	server.SuccessResponse(c, "API token created successfully", gin.H{
		"id":      tokenID, // Actual token ID (e.g., tok_...)
		"user_id": targetUserID,
		"name":    req.Name,
		"token":   signedToken, // The generated JWT
	})
}

// GetTokensHandler handles retrieving API token metadata.
func GetTokensHandler(c *gin.Context) {
	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)

	// --- Differentiate between Get Specific (path param) and List (query param) ---
	tokenIDFromPath := c.Param("id") // Changed parameter name from token_id to id
	var targetUserID string

	if tokenIDFromPath != "" {
		// Case 1: Get Specific Token (id from path parameter)
		targetUserID = c.Query("user_id")
	} else {
		// Case 2: List Tokens (id from path is empty)
		targetUserID = c.Query("user_id")
	}
	var requiredPermission string
	isAdminAction := false

	if targetUserID != "" && targetUserID != requestingUserID {
		isAdminAction = true
		if tokenIDFromPath != "" {
			requiredPermission = "api:read:item:all"
		} else {
			requiredPermission = "api:read:list:all"
		}
	} else {
		isAdminAction = false
		targetUserID = requestingUserID // Default to self if not specified or same
		if tokenIDFromPath != "" {
			requiredPermission = "api:read:item"
		} else {
			requiredPermission = "api:read:list"
		}
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// Validate target user exists if admin action
	if isAdminAction {
		_, userExists, err := user.GetUserByID(targetUserID)
		if err != nil {
			log.Printf("%s Error checking target user %s existence: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
	}

	// --- Call token_store ---
	if tokenIDFromPath != "" {
		// Get specific token info using the ID from the path
		tokenInfo, found, err := token_store.GetTokenInfo(tokenIDFromPath)
		if err != nil {
			log.Printf("%s Error getting token info for %s: %v", time.Now().UTC().Format(time.RFC3339), tokenIDFromPath, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details")
			return
		}
		if !found {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found", tokenIDFromPath))
			return
		}

		// Verify ownership or admin permission
		// IMPORTANT: targetUserID here is the USER we are allowed to look at,
		// either self or the one specified in admin action.
		if tokenInfo.UserID != targetUserID && !isAdminAction {
			// If not admin action, token's user must match requesting user (who is targetUserID here)
			server.ErrorResponse(c, http.StatusForbidden, "Permission denied to access this specific token")
			return
		}
		if isAdminAction && tokenInfo.UserID != targetUserID {
			// If admin action, token's user must match the target user specified
			server.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Token '%s' does not belong to target user '%s'", tokenIDFromPath, targetUserID))
			return
		}

		// Ensure it's an API token
		if tokenInfo.Audience != AudienceAPI {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' is not an API token", tokenIDFromPath))
			return
		}

		// Add the ID back into the response structure
		response := struct {
			token_store.TokenInfo
			ID string `json:"id"`
		}{
			TokenInfo: tokenInfo,
			ID:        tokenIDFromPath, // Use the ID from the path
		}
		server.SuccessResponse(c, "API token details retrieved", response)

	} else {
		// List tokens for the target user
		tokensInfo, _, err := token_store.GetUserTokens(targetUserID)
		if err != nil {
			log.Printf("%s Error listing tokens for user %s: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to list API tokens")
			return
		}
		server.SuccessResponse(c, "API tokens listed successfully", tokensInfo)
	}
}

// UpdateTokenHandler handles updating API token metadata, supporting admin actions.
func UpdateTokenHandler(c *gin.Context) {
	var req UpdateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)
	// req.UserID is the target user for admin actions
	// req.ID is the token ID to update
	targetUserIDForAction := req.UserID
	tokenIDToUpdate := req.ID
	var requiredPermission string
	isAdminAction := false

	if targetUserIDForAction != "" && targetUserIDForAction != requestingUserID {
		isAdminAction = true
		requiredPermission = "api:update:all"
	} else {
		isAdminAction = false
		// If not admin action, or admin targeting self, the effective target user is the requester
		targetUserIDForAction = requestingUserID
		requiredPermission = "api:update"
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// Validate target user exists if admin action specifies a different user
	if isAdminAction {
		_, userExists, err := user.GetUserByID(targetUserIDForAction)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to check target user data: "+err.Error())
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserIDForAction))
			return
		}
	}

	// Get the current token info using the token ID from the request
	tokenInfo, found, err := token_store.GetTokenInfo(tokenIDToUpdate)
	if err != nil {
		log.Printf("%s Error getting token info for update %s: %v", time.Now().UTC().Format(time.RFC3339), tokenIDToUpdate, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details for update")
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found for update", tokenIDToUpdate))
		return
	}

	// Verify ownership or admin permission for the specific token
	// The token's UserID must match the user we are allowed to act upon (targetUserIDForAction)
	if tokenInfo.UserID != targetUserIDForAction {
		server.ErrorResponse(c, http.StatusForbidden, "Permission denied to update this specific token")
		return
	}

	// Ensure it's an API token
	if tokenInfo.Audience != AudienceAPI {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' is not an API token", tokenIDToUpdate))
		return
	}

	// Apply updates from the request (only Name supported currently)
	if req.Name == nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Name field is required for token update")
		return
	}

	if *req.Name == tokenInfo.Name {
		// No changes detected
		response := struct {
			token_store.TokenInfo
			ID string `json:"id"`
		}{
			TokenInfo: tokenInfo,
			ID:        tokenIDToUpdate, // Use the token ID from the request
		}
		server.SuccessResponse(c, "No changes detected in token name", response)
		return
	}

	// Update the token name
	tokenInfo.Name = *req.Name

	// Store the updated token info
	if err := token_store.StoreToken(tokenIDToUpdate, tokenInfo); err != nil {
		log.Printf("%s Error storing updated token info for %s: %v", time.Now().UTC().Format(time.RFC3339), tokenIDToUpdate, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to save updated token name")
		return
	}

	// Return the updated info
	response := struct {
		token_store.TokenInfo
		ID string `json:"id"`
	}{
		TokenInfo: tokenInfo,
		ID:        tokenIDToUpdate, // Use the token ID from the request
	}
	server.SuccessResponse(c, "API token name updated successfully", response)
}

// DeleteTokenHandler handles deleting/revoking an API token.
func DeleteTokenHandler(c *gin.Context) {
	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)

	var req DeleteTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// req.UserID is the target user for admin actions
	// req.ID is the token ID to delete
	targetUserIDForAction := req.UserID
	tokenIDToDelete := req.ID

	if tokenIDToDelete == "" {
		server.ErrorResponse(c, http.StatusBadRequest, "id field (token ID) is required")
		return
	}

	var requiredPermission string
	isAdminAction := false

	if targetUserIDForAction != "" && targetUserIDForAction != requestingUserID {
		isAdminAction = true
		requiredPermission = "api:delete:all"
	} else {
		isAdminAction = false
		// If not admin action, or admin targeting self, the effective target user is the requester
		targetUserIDForAction = requestingUserID
		requiredPermission = "api:delete"
	}

	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// If admin action specifies a different user, validate the target user exists
	if isAdminAction {
		_, userExists, err := user.GetUserByID(targetUserIDForAction)
		if err != nil {
			log.Printf("%s Error checking target user %s existence: %v", time.Now().UTC().Format(time.RFC3339), targetUserIDForAction, err)
			// Don't expose internal error, just proceed to token check
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserIDForAction))
			return
		}
	}

	// Get token info to verify ownership before deleting
	tokenInfo, found, err := token_store.GetTokenInfo(tokenIDToDelete)
	if err != nil {
		log.Printf("%s Error getting token info for delete %s: %v", time.Now().UTC().Format(time.RFC3339), tokenIDToDelete, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details for deletion")
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found for deletion", tokenIDToDelete))
		return
	}

	// Verify ownership or admin permission for the specific token
	// The token's UserID must match the user we are allowed to act upon (targetUserIDForAction)
	if tokenInfo.UserID != targetUserIDForAction {
		server.ErrorResponse(c, http.StatusForbidden, "Permission denied to delete this specific token")
		return
	}

	// Ensure it's an API token (though Revoke works on any JTI, this adds context)
	if tokenInfo.Audience != AudienceAPI {
		log.Printf("%s Warning: Attempting to delete token '%s' which is not marked as an API token (Audience: %s)", time.Now().UTC().Format(time.RFC3339), tokenIDToDelete, tokenInfo.Audience)
		// Proceed with deletion anyway, as revocation is by JTI
	}

	// Call the service function to revoke the token
	if err := token_store.RevokeToken(tokenIDToDelete); err != nil {
		log.Printf("%s Error revoking token %s: %v", time.Now().UTC().Format(time.RFC3339), tokenIDToDelete, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete API token")
		return
	}

	server.SuccessResponse(c, "API token deleted successfully", nil)
}

// Removed placeholder CreateAPIToken, UpdateAPIToken, DeleteAPIToken functions
