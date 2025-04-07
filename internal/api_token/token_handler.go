package api_token

import (
	"fmt"
	"net/http"
	"strings"

	logger "github.com/OG-Open-Source/PanelBase/internal/logging"
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
	TokenID string `json:"token_id" binding:"required"`
	UserID  string `json:"user_id"` // Optional: For admin actions targeting a specific user
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

	// Validate target user exists if admin action
	if targetUserID != requestingUserID {
		_, userExists, err := user.GetUserByID(targetUserID)
		if err != nil {
			logger.ErrorPrintf("API_TOKEN", "CREATE_VALIDATE", "Error checking target user %s existence: %v", targetUserID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to validate target user")
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
		logger.DebugPrintf("API_TOKEN", "CREATE_VALIDATE", "Admin action target user %s validated.", targetUserID)
	} else {
		logger.DebugPrintf("API_TOKEN", "CREATE", "Action is for requesting user %s.", targetUserID)
	}

	// Load user data for JWT secret and permission validation
	userInfo, userFound, err := user.GetUserByID(targetUserID)
	if err != nil || !userFound {
		logger.ErrorPrintf("API_TOKEN", "CREATE_LOAD_USER", "Failed to load user data for %s (found: %v): %v", targetUserID, userFound, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data")
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
	tokenID, _, signedToken, err := CreateAPIToken(userInfo, createPayload)
	if err != nil {
		// Handle specific errors from the service
		if strings.Contains(err.Error(), "exceed user permissions") {
			server.ErrorResponse(c, http.StatusBadRequest, "Requested scopes exceed user permissions")
		} else if strings.Contains(err.Error(), "duration is required") {
			server.ErrorResponse(c, http.StatusBadRequest, "Token duration is required")
		} else if strings.Contains(err.Error(), "invalid duration format") {
			server.ErrorResponse(c, http.StatusBadRequest, "Invalid duration format: "+err.Error())
		} else {
			logger.ErrorPrintf("API_TOKEN", "CREATE_SVC_CALL", "Error creating API token for user %s: %v", targetUserID, err)
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
	tokenID := c.Param("token_id") // Try reading from path parameter first
	var targetUserID string

	if tokenID != "" {
		// Case 1: Get Specific Token (token_id from path parameter)
		// For admin actions on specific token, user_id might be needed from query (optional)
		targetUserID = c.Query("user_id") // Still check query for optional user_id for admin override
		logger.DebugPrintf("API_TOKEN", "GET_SPECIFIC", "Path Param token_id: '%s', Query Param user_id: '%s'", tokenID, targetUserID)
	} else {
		// Case 2: List Tokens (token_id from path is empty)
		// user_id comes from query parameter for potential admin action
		targetUserID = c.Query("user_id")
		logger.DebugPrintf("API_TOKEN", "LIST", "Query Param user_id: '%s'", targetUserID)
	}
	var requiredPermission string
	isAdminAction := false

	if targetUserID != "" && targetUserID != requestingUserID {
		isAdminAction = true
		if tokenID != "" {
			requiredPermission = "api:read:item:all"
		} else {
			requiredPermission = "api:read:list:all"
		}
	} else {
		isAdminAction = false
		targetUserID = requestingUserID // Default to self if not specified or same
		if tokenID != "" {
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
			logger.ErrorPrintf("API_TOKEN", "GET_VALIDATE", "Error checking target user %s existence: %v", targetUserID, err)
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
		logger.DebugPrintf("API_TOKEN", "GET_VALIDATE", "Admin action target user %s validated.", targetUserID)
	}

	// --- Call token_store ---
	if tokenID != "" {
		// Get specific token info
		tokenInfo, found, err := token_store.GetTokenInfo(tokenID)
		if err != nil {
			logger.ErrorPrintf("API_TOKEN", "GET_INFO", "Error getting token info for %s: %v", tokenID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details")
			return
		}
		if !found {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found", tokenID))
			return
		}

		// Verify ownership or admin permission
		if tokenInfo.UserID != targetUserID {
			server.ErrorResponse(c, http.StatusForbidden, "Permission denied to access this specific token")
			return
		}

		// Ensure it's an API token
		if tokenInfo.Audience != AudienceAPI {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' is not an API token", tokenID))
			return
		}

		// Add the ID back into the response structure (since TokenInfo doesn't store it)
		response := struct {
			token_store.TokenInfo
			ID string `json:"id"`
		}{
			TokenInfo: tokenInfo,
			ID:        tokenID,
		}
		server.SuccessResponse(c, "API token details retrieved", response)

	} else {
		// List tokens for the target user using the dedicated store function
		tokensInfo, _, err := token_store.GetUserTokens(targetUserID)
		if err != nil {
			logger.ErrorPrintf("API_TOKEN", "LIST_STORE_CALL", "Error listing tokens for user %s: %v", targetUserID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to list API tokens")
			return
		}

		// Note: GetUserTokens already filters for non-revoked API tokens
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
	targetUserID := req.UserID
	tokenID := req.TokenID
	var requiredPermission string
	isAdminAction := false

	if targetUserID != "" && targetUserID != requestingUserID {
		isAdminAction = true
		requiredPermission = "api:update:all"
	} else {
		isAdminAction = false
		targetUserID = requestingUserID // Default to self if not specified or same
		requiredPermission = "api:update"
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// Validate target user exists if admin action
	if isAdminAction {
		_, userExists, err := user.GetUserByID(targetUserID)
		if err != nil {
			logger.ErrorPrintf("API_TOKEN", "UPDATE_VALIDATE", "Error checking target user %s existence: %v", targetUserID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load target user data: "+err.Error())
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
		logger.DebugPrintf("API_TOKEN", "UPDATE_VALIDATE", "Admin action target user %s validated.", targetUserID)
	}

	// Get the current token info
	tokenInfo, found, err := token_store.GetTokenInfo(tokenID)
	if err != nil {
		logger.ErrorPrintf("API_TOKEN", "UPDATE_GET_INFO", "Error getting token info for update %s: %v", tokenID, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details for update")
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found for update", tokenID))
		return
	}

	// Verify ownership or admin permission for the specific token
	if tokenInfo.UserID != targetUserID {
		server.ErrorResponse(c, http.StatusForbidden, "Permission denied to update this specific token")
		return
	}

	// Ensure it's an API token
	if tokenInfo.Audience != AudienceAPI {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' is not an API token", tokenID))
		return
	}

	// Apply updates from the request (only Name supported currently)
	updated := false
	if req.Name != nil && *req.Name != tokenInfo.Name {
		tokenInfo.Name = *req.Name
		updated = true
	}
	/* // Description is not part of TokenInfo in store
	if req.Description != nil { // && *req.Description != tokenInfo.Description {
		// tokenInfo.Description = *req.Description // Cannot update description here
		// updated = true
		log.Printf("Warning: Token description update requested but not supported by current TokenInfo structure.")
	}
	*/

	if !updated {
		// Add the ID back into the response structure even if no changes
		response := struct {
			token_store.TokenInfo
			ID string `json:"id"`
		}{
			TokenInfo: tokenInfo,
			ID:        tokenID,
		}
		server.SuccessResponse(c, "No changes detected in token metadata", response)
		return
	}

	// Store the updated token info
	if err := token_store.StoreToken(tokenID, tokenInfo); err != nil {
		logger.ErrorPrintf("API_TOKEN", "UPDATE_STORE", "Error storing updated token info for %s: %v", tokenID, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to save updated token metadata")
		return
	}

	// Add the ID back into the response structure
	response := struct {
		token_store.TokenInfo
		ID string `json:"id"`
	}{
		TokenInfo: tokenInfo,
		ID:        tokenID,
	}
	server.SuccessResponse(c, "API token updated successfully", response)
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

	if req.TokenID == "" {
		server.ErrorResponse(c, http.StatusBadRequest, "token_id is required")
		return
	}

	targetUserID := req.UserID
	tokenID := req.TokenID
	var requiredPermission string
	isAdminAction := false

	if targetUserID != "" {
		isAdminAction = true
		requiredPermission = "api:delete:all"
	} else {
		targetUserID = requestingUserID // Default to self if not admin action
		requiredPermission = "api:delete"
	}

	if !middleware.CheckPermission(c, "api", requiredPermission) {
		// CheckPermission sends the response, so just return
		return
	}

	// If admin action, validate the target user exists
	if isAdminAction {
		_, userExists, err := user.GetUserByID(targetUserID)
		if err != nil {
			logger.ErrorPrintf("API_TOKEN", "DELETE_VALIDATE", "Error checking target user %s existence: %v", targetUserID, err)
			// Don't necessarily fail here, maybe user was just deleted?
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, "Target user not found")
			return
		}
		logger.DebugPrintf("API_TOKEN", "DELETE_VALIDATE", "Admin action target user %s validated/checked.", targetUserID)
	}

	// Get current token info to verify ownership and type
	tokenInfo, found, err := token_store.GetTokenInfo(tokenID)
	if err != nil {
		logger.ErrorPrintf("API_TOKEN", "DELETE_GET_INFO", "Failed to retrieve token info for delete: %s: %v", tokenID, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token info: "+err.Error())
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, "Token not found")
		return
	}

	// Verify ownership if not admin action
	if !isAdminAction && tokenInfo.UserID != requestingUserID {
		server.ErrorResponse(c, http.StatusForbidden, "You do not own this token")
		return
	}

	// Ensure it's an API token we are deleting, not a web session
	if tokenInfo.Audience != AudienceAPI {
		server.ErrorResponse(c, http.StatusBadRequest, "Cannot delete non-API tokens via this endpoint")
		return
	}

	// If it's an admin action targeting another user, verify the token belongs to the target user
	if isAdminAction && tokenInfo.UserID != targetUserID {
		server.ErrorResponse(c, http.StatusForbidden, "Token does not belong to the specified target user")
		return
	}

	// Revoke the token in the store
	err = token_store.RevokeToken(tokenID)
	if err != nil {
		logger.ErrorPrintf("API_TOKEN", "DELETE_REVOKE", "Failed to revoke token %s: %v", tokenID, err)
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to revoke token")
		return
	}

	logger.Printf("API_TOKEN", "DELETE", "Successfully revoked token ID: %s for user: %s", tokenID, targetUserID)
	server.SuccessResponse(c, "API token deleted successfully", nil)
}

// Removed placeholder CreateAPIToken, UpdateAPIToken, DeleteAPIToken functions
