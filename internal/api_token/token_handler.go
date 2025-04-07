package api_token

import (
	"fmt"
	"log"
	"net/http"
	"strings"

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
type GetTokensRequest struct {
	UserID  string `json:"user_id"`  // Optional: For admin actions targeting a specific user
	TokenID string `json:"token_id"` // Optional: To get a specific token by ID
}

// CreateTokenHandler handles creating an API token, potentially for another user if admin.
func CreateTokenHandler(c *gin.Context) {
	var req CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	authUserID, _ := c.Get(middleware.ContextKeyUserID)
	requestingUserID := authUserID.(string)
	targetUserID := req.UserID
	var requiredPermission string
	var isAdminAction bool // Keep for check below
	if targetUserID != "" && targetUserID != requestingUserID {
		isAdminAction = true // Needed for user check
		requiredPermission = "api:create:all"
	} else {
		isAdminAction = false
		targetUserID = requestingUserID
		requiredPermission = "api:create"
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// Validate target user exists if admin action
	if isAdminAction {
		// var targetUserInstance models.User // Not needed for just existence check
		_, userExists, err := user.GetUserByID(targetUserID)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load target user data: "+err.Error())
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
	}

	log.Printf("[Placeholder] CreateTokenHandler: Action permitted for user %s targeting user %s (admin: %t) with name: %s, perm: %s", requestingUserID, targetUserID, isAdminAction, req.Name, requiredPermission)

	// Placeholder Response
	server.SuccessResponse(c, "Placeholder: Token created successfully", gin.H{"id": "tkn_placeholder", "user_id": targetUserID, "name": req.Name, "token": "jwt_placeholder"})
}

// GetTokensHandler handles retrieving API token metadata.
// Reads optional 'id' from the JSON request body.
// - If no 'id' is provided, lists tokens for the current user (requires 'api:read:list').
// - If 'id' is provided:
//   - If the token belongs to the current user, requires 'api:read:item'.
//   - If the token belongs to another user, requires 'api:read:item:all'.
func GetTokensHandler(c *gin.Context) {
	authUserID, _ := c.Get(middleware.ContextKeyUserID)
	requestingUserID := authUserID.(string)
	var req GetTokensRequest
	_ = c.ShouldBindJSON(&req)
	targetUserID := req.UserID
	tokenID := req.TokenID
	var requiredPermission string
	if targetUserID != "" && targetUserID != requestingUserID {
		if tokenID != "" {
			requiredPermission = "api:read:item:all"
		} else {
			requiredPermission = "api:read:list:all"
		}
	} else {
		targetUserID = requestingUserID
		if tokenID != "" {
			requiredPermission = "api:read:item"
		} else {
			requiredPermission = "api:read:list"
		}
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	log.Printf("[Placeholder] GetTokensHandler: Action permitted for user %s targeting user %s (token: %s) with permission %s", requestingUserID, targetUserID, tokenID, requiredPermission)

	// Placeholder Response - Replace with actual service call later
	if tokenID != "" {
		server.SuccessResponse(c, "Placeholder: Token details would be here", gin.H{"id": tokenID, "user_id": targetUserID, "name": "Placeholder Token"})
	} else {
		server.SuccessResponse(c, "Placeholder: Token list would be here", []gin.H{{"id": "placeholder_tkn_1", "user_id": targetUserID, "name": "Placeholder Token 1"}})
	}
}

// UpdateTokenHandler handles updating API token metadata, supporting admin actions.
func UpdateTokenHandler(c *gin.Context) {
	var req UpdateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	authUserID, _ := c.Get(middleware.ContextKeyUserID)
	requestingUserID := authUserID.(string)
	targetUserID := req.UserID
	tokenID := req.TokenID
	var requiredPermission string
	if targetUserID != "" && targetUserID != requestingUserID {
		requiredPermission = "api:update:all"
	} else {
		targetUserID = requestingUserID
		requiredPermission = "api:update"
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// TODO: Add pre-checks here (fetch token, check ownership/target, check revocation)

	log.Printf("[Placeholder] UpdateTokenHandler: Action permitted for user %s targeting user %s (token: %s) with perm: %s. Update Name: %v, Desc: %v",
		requestingUserID, targetUserID, tokenID, requiredPermission, req.Name != nil, req.Description != nil)

	// Placeholder Response
	updatedName := "Original Name"
	if req.Name != nil {
		updatedName = *req.Name
	}
	server.SuccessResponse(c, "Placeholder: Token updated successfully", gin.H{"id": tokenID, "user_id": targetUserID, "name": updatedName})
}

// DeleteTokenHandler handles deleting (revoking) an API token, supporting admin actions.
func DeleteTokenHandler(c *gin.Context) {
	var req DeleteTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}
	authUserID, _ := c.Get(middleware.ContextKeyUserID)
	requestingUserID := authUserID.(string)
	targetUserID := req.UserID
	tokenID := req.TokenID
	var requiredPermission string
	if targetUserID != "" && targetUserID != requestingUserID {
		requiredPermission = "api:delete:all"
	} else {
		targetUserID = requestingUserID
		requiredPermission = "api:delete"
	}
	if !middleware.CheckPermission(c, "api", requiredPermission) {
		return
	}

	// TODO: Add pre-checks here (fetch token, check ownership/target)

	log.Printf("[Placeholder] DeleteTokenHandler: Action permitted for user %s targeting user %s (token: %s) with perm: %s", requestingUserID, targetUserID, tokenID, requiredPermission)

	// Call actual revoke from token_store
	err := token_store.RevokeToken(tokenID)
	if err != nil {
		// Handle not found gracefully
		if strings.Contains(strings.ToLower(err.Error()), "not found") {
			log.Printf("Token %s already deleted or never existed.", tokenID)
			server.SuccessResponse(c, "API token deleted successfully (or was already deleted)", nil)
		} else {
			log.Printf("Error revoking token %s: %v", tokenID, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete token: "+err.Error())
		}
		return
	}

	server.SuccessResponse(c, "API token deleted successfully", nil)
}

// Removed placeholder CreateAPIToken, UpdateAPIToken, DeleteAPIToken functions
