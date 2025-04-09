package token

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	// Use the middleware package
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	// Import the new service and models packages
	"github.com/OG-Open-Source/PanelBase/pkg/apitokenservice"
	"github.com/OG-Open-Source/PanelBase/pkg/models"
	"github.com/OG-Open-Source/PanelBase/pkg/serverutils"
	"github.com/OG-Open-Source/PanelBase/pkg/tokenstore"
	"github.com/OG-Open-Source/PanelBase/pkg/userservice"
	"github.com/gin-gonic/gin"
)

// Constants
const AudienceAPI = "api" // Keep AudienceAPI constant here or in service, needs consistency

// ... existing structs (GetTokenPayload, UpdateTokenPayload, DeleteTokenPayload etc.) ...
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
	TokenID  string  `json:"token_id" binding:"required"` // Keep for backward compatibility? Prefer "id"
	Username *string `json:"username,omitempty"`          // Optional: Target username for admin actions
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

// ... existing handlers (CreateTokenHandler, GetTokensHandler, etc.) ...
// CreateTokenHandler handles creating an API token, potentially for another user if admin.
// POST /api/v1/account/token
// POST /api/v1/users/:user_id/token (Admin)
func CreateTokenHandler(c *gin.Context) {
	var req models.CreateAPITokenRequest // Use request struct from models
	if err := c.ShouldBindJSON(&req); err != nil {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)

	// Determine target user and required permission
	targetUserID := c.Param("user_id") // Check URL param first (admin route)
	var requiredPermissionScope string
	isAdminRoute := targetUserID != ""

	if isAdminRoute {
		requiredPermissionScope = "api:create:all"
	} else {
		// If not admin route, target user is self
		targetUserID = requestingUserID
		requiredPermissionScope = "api:create"
	}

	// Check permission
	if !middleware.CheckPermission(c, "api", requiredPermissionScope) {
		return
	}

	// Fetch the target user data (always needed)
	targetUserInstance, userExists, err := userservice.GetUserByID(targetUserID)
	if err != nil {
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to load target user data: "+err.Error())
		return
	}
	if !userExists {
		serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
		return
	}

	// Prepare payload for the service function
	createPayload := models.CreateAPITokenPayload{
		Name:        req.Name,
		Description: req.Description,
		Duration:    req.Duration,                                 // Service will handle default/validation
		Scopes:      models.ScopeStringsToPermissions(req.Scopes), // Convert []string to map[string][]string
	}

	// Call the service function from the pkg/apitokenservice package
	tokenID, _, signedToken, err := apitokenservice.CreateAPIToken(targetUserInstance, createPayload)
	if err != nil {
		// Handle specific errors from the service
		if strings.Contains(err.Error(), "exceed user permissions") {
			serverutils.ErrorResponse(c, http.StatusBadRequest, "Requested scopes exceed user permissions")
		} else if strings.Contains(err.Error(), "duration is required") {
			serverutils.ErrorResponse(c, http.StatusBadRequest, "Token duration is required")
		} else if strings.Contains(err.Error(), "invalid duration format") {
			serverutils.ErrorResponse(c, http.StatusBadRequest, "Invalid duration format: "+err.Error())
		} else {
			log.Printf("%s Error creating API token for user %s: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to create API token: "+err.Error())
		}
		return
	}

	// Return the actual token ID and JWT string
	serverutils.SuccessResponse(c, "API token created successfully", gin.H{
		"id":      tokenID, // Actual token ID (e.g., tok_...)
		"user_id": targetUserID,
		"name":    req.Name,
		"token":   signedToken, // The generated JWT
	})
}

// GetTokensHandler handles retrieving API token metadata.
// GET /api/v1/account/token
// GET /api/v1/account/token/:id
// GET /api/v1/users/:user_id/token (Admin, list)
// GET /api/v1/users/:user_id/token/:id (Admin, specific)
func GetTokensHandler(c *gin.Context) {
	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)

	tokenID := c.Param("id")
	targetUserIDFromURL := c.Param("user_id")

	var targetUserID string
	var requiredPermissionScope string
	isAdminRoute := targetUserIDFromURL != ""
	isSpecificGet := tokenID != ""

	if isAdminRoute {
		targetUserID = targetUserIDFromURL
		if isSpecificGet {
			requiredPermissionScope = "api:read:item:all"
		} else {
			requiredPermissionScope = "api:read:list:all"
		}
	} else {
		targetUserID = requestingUserID
		if isSpecificGet {
			requiredPermissionScope = "api:read:item"
		} else {
			requiredPermissionScope = "api:read:list"
		}
	}

	// Check permission
	if !middleware.CheckPermission(c, "api", requiredPermissionScope) {
		return
	}

	// Validate target user exists if admin action (redundant? CheckPermission might imply user existence? Let's keep it for clarity)
	if isAdminRoute {
		_, userExists, err := userservice.GetUserByID(targetUserID)
		if err != nil {
			log.Printf("%s Error checking target user %s existence: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
			// Don't fail here, maybe log and continue? Or return error?
			// Let's return error for now, as subsequent logic depends on user existing.
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Error checking target user existence")
			return
		}
		if !userExists {
			serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
	}

	// --- Call token_store --- depends on specificGet
	if isSpecificGet {
		// Get specific token info
		tokenInfo, found, err := tokenstore.GetTokenInfo(tokenID)
		if err != nil {
			log.Printf("%s Error getting token info for %s: %v", time.Now().UTC().Format(time.RFC3339), tokenID, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details")
			return
		}
		if !found {
			serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found", tokenID))
			return
		}

		// Verify ownership
		if tokenInfo.UserID != targetUserID {
			serverutils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Token '%s' does not belong to target user '%s'", tokenID, targetUserID))
			return
		}

		// Ensure it's an API token
		if tokenInfo.Audience != apitokenservice.AudienceAPI { // Use constant from service
			serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' is not an API token", tokenID))
			return
		}

		// Add the ID back into the response structure
		response := struct {
			tokenstore.TokenInfo
			ID string `json:"id"`
		}{
			TokenInfo: tokenInfo,
			ID:        tokenID, // Use the ID from the path
		}
		serverutils.SuccessResponse(c, "API token details retrieved", response)

	} else {
		// List tokens for the target user
		tokensInfo, tokenIDs, err := tokenstore.GetUserTokens(targetUserID)
		if err != nil {
			log.Printf("%s Error listing tokens for user %s: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve tokens")
			return
		}

		// Filter for API tokens only and add ID to each
		apiTokens := []interface{}{}
		for i, info := range tokensInfo {
			if info.Audience == apitokenservice.AudienceAPI { // Use constant from service
				responseToken := struct {
					tokenstore.TokenInfo
					ID string `json:"id"`
				}{
					TokenInfo: info,
					ID:        tokenIDs[i], // Use the ID from the parallel slice
				}
				apiTokens = append(apiTokens, responseToken)
			}
		}

		serverutils.SuccessResponse(c, "API tokens retrieved successfully", apiTokens)
	}
}

// UpdateTokenHandler handles updating API token metadata (Name, Description).
// PATCH /api/v1/account/token/:id
// PATCH /api/v1/users/:user_id/token/:id (Admin)
func UpdateTokenHandler(c *gin.Context) {
	var req models.UpdateTokenPayload // Use payload struct from models
	if err := c.ShouldBindJSON(&req); err != nil {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)

	tokenID := c.Param("id") // Get token ID from path
	if tokenID == "" {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Token ID is required in the URL path")
		return
	}
	// Ensure the ID from the path matches the ID in the body if provided (sanity check)
	if req.ID != "" && req.ID != tokenID {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Token ID in URL path does not match ID in request body")
		return
	}
	req.ID = tokenID // Use ID from path as the authoritative one

	targetUserIDFromURL := c.Param("user_id")

	var targetUserID string
	var requiredPermissionScope string
	isAdminRoute := targetUserIDFromURL != ""

	if isAdminRoute {
		targetUserID = targetUserIDFromURL
		requiredPermissionScope = "api:update:all"
		// Ensure user_id in body (if provided) matches URL
		if req.UserID != "" && req.UserID != targetUserID {
			serverutils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("user_id '%s' in request body does not match user_id '%s' in URL for admin action", req.UserID, targetUserID))
			return
		}
	} else {
		targetUserID = requestingUserID
		requiredPermissionScope = "api:update"
		// Disallow user_id in body for non-admin routes
		if req.UserID != "" {
			serverutils.ErrorResponse(c, http.StatusBadRequest, "user_id cannot be specified in request body for self-update")
			return
		}
	}

	// Check permission
	if !middleware.CheckPermission(c, "api", requiredPermissionScope) {
		return
	}

	// Validate target user exists if admin action
	if isAdminRoute {
		_, userExists, err := userservice.GetUserByID(targetUserID)
		if err != nil {
			log.Printf("%s Error checking target user %s existence: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Error checking target user existence")
			return
		}
		if !userExists {
			serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
	}

	// 1. Get current token info from token_store
	currentTokenInfo, found, err := tokenstore.GetTokenInfo(req.ID)
	if err != nil {
		log.Printf("%s Error getting current token info for %s: %v", time.Now().UTC().Format(time.RFC3339), req.ID, err)
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details for update")
		return
	}
	if !found {
		serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found", req.ID))
		return
	}

	// 2. Verify ownership
	if currentTokenInfo.UserID != targetUserID {
		serverutils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Token '%s' does not belong to target user '%s'", req.ID, targetUserID))
		return
	}

	// 3. Ensure it's an API token
	if currentTokenInfo.Audience != apitokenservice.AudienceAPI { // Use constant from service
		serverutils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Token with ID '%s' is not an API token and cannot be updated via this endpoint", req.ID))
		return
	}

	// 4. Apply updates (Name, Description)
	updated := false
	if req.Name != nil && *req.Name != currentTokenInfo.Name {
		currentTokenInfo.Name = *req.Name
		updated = true
	}
	// Description is not stored in TokenInfo currently, so cannot update it here.
	// If Description needs to be updatable, TokenInfo struct and storage logic needs changes.
	if req.Description != nil {
		log.Printf("%s Info: Update request for token %s included description, but it's not stored/updatable in current implementation.", time.Now().UTC().Format(time.RFC3339), req.ID)
		// Optionally return a specific message or ignore silently
		// serverutils.ErrorResponse(c, http.StatusBadRequest, "Updating token description is not currently supported")
		// return
	}

	if !updated {
		// Add ID back for the response even if not updated
		response := struct {
			tokenstore.TokenInfo
			ID string `json:"id"`
		}{
			TokenInfo: currentTokenInfo,
			ID:        req.ID,
		}
		serverutils.SuccessResponse(c, "No changes detected, token not updated", response)
		return
	}

	// 5. Store updated token info back into token_store
	if err := tokenstore.StoreToken(req.ID, currentTokenInfo); err != nil {
		log.Printf("%s Error storing updated token info for %s: %v", time.Now().UTC().Format(time.RFC3339), req.ID, err)
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save updated token details")
		return
	}

	// Add the ID back into the response structure
	response := struct {
		tokenstore.TokenInfo
		ID string `json:"id"`
	}{
		TokenInfo: currentTokenInfo,
		ID:        req.ID,
	}
	serverutils.SuccessResponse(c, "API token updated successfully", response)
}

// DeleteTokenHandler handles deleting an API token.
// DELETE /api/v1/account/token/:id
// DELETE /api/v1/users/:user_id/token/:id (Admin)
func DeleteTokenHandler(c *gin.Context) {
	var req models.DeleteTokenPayload // Use payload struct from models
	// Body is optional for DELETE, but useful for admin specifying user_id?
	// Let's bind optionally, but primarily rely on URL params.
	_ = c.ShouldBindJSON(&req)

	authUserID, _ := c.Get(string(middleware.ContextKeyUserID))
	requestingUserID := authUserID.(string)

	tokenID := c.Param("id") // Get token ID from path
	if tokenID == "" {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Token ID is required in the URL path")
		return
	}
	// Ensure the ID from the path matches the ID in the body if provided
	if req.ID != "" && req.ID != tokenID {
		serverutils.ErrorResponse(c, http.StatusBadRequest, "Token ID in URL path does not match ID in request body")
		return
	}
	req.ID = tokenID // Use ID from path

	targetUserIDFromURL := c.Param("user_id")

	var targetUserID string
	var requiredPermissionScope string
	isAdminRoute := targetUserIDFromURL != ""

	if isAdminRoute {
		targetUserID = targetUserIDFromURL
		requiredPermissionScope = "api:delete:all"
		// Ensure user_id in body (if provided) matches URL
		if req.UserID != "" && req.UserID != targetUserID {
			serverutils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("user_id '%s' in request body does not match user_id '%s' in URL for admin action", req.UserID, targetUserID))
			return
		}
	} else {
		targetUserID = requestingUserID
		requiredPermissionScope = "api:delete"
		// Disallow user_id in body for non-admin routes
		if req.UserID != "" {
			serverutils.ErrorResponse(c, http.StatusBadRequest, "user_id cannot be specified in request body for self-delete")
			return
		}
	}

	// Check permission
	if !middleware.CheckPermission(c, "api", requiredPermissionScope) {
		return
	}

	// Validate target user exists if admin action
	if isAdminRoute {
		_, userExists, err := userservice.GetUserByID(targetUserID)
		if err != nil {
			log.Printf("%s Error checking target user %s existence: %v", time.Now().UTC().Format(time.RFC3339), targetUserID, err)
			serverutils.ErrorResponse(c, http.StatusInternalServerError, "Error checking target user existence")
			return
		}
		if !userExists {
			serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Target user with ID '%s' not found", targetUserID))
			return
		}
	}

	// 1. Get current token info to verify ownership/existence/type
	tokenInfo, found, err := tokenstore.GetTokenInfo(req.ID)
	if err != nil {
		log.Printf("%s Error getting token info for deletion %s: %v", time.Now().UTC().Format(time.RFC3339), req.ID, err)
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token details before deletion")
		return
	}
	if !found {
		serverutils.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("Token with ID '%s' not found", req.ID))
		return
	}

	// 2. Verify ownership
	if tokenInfo.UserID != targetUserID {
		serverutils.ErrorResponse(c, http.StatusForbidden, fmt.Sprintf("Token '%s' does not belong to target user '%s'", req.ID, targetUserID))
		return
	}

	// 3. Ensure it's an API token (cannot delete web session tokens this way)
	if tokenInfo.Audience != apitokenservice.AudienceAPI { // Use constant from service
		serverutils.ErrorResponse(c, http.StatusBadRequest, fmt.Sprintf("Token with ID '%s' is not an API token and cannot be deleted via this endpoint", req.ID))
		return
	}

	// 4. Revoke/Delete the token from the token_store
	if err := tokenstore.RevokeToken(req.ID); err != nil {
		log.Printf("%s Error revoking token %s: %v", time.Now().UTC().Format(time.RFC3339), req.ID, err)
		serverutils.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete token")
		return
	}

	serverutils.SuccessResponse(c, "API token deleted successfully", nil)
}
