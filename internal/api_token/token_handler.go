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
}

// UpdateTokenPayload defines the structure for updating token metadata.
// Name, Description, and Scopes can be updated.
type UpdateTokenPayload struct {
	ID          string                 `json:"id" binding:"required"`
	Name        *string                `json:"name,omitempty"`
	Description *string                `json:"description,omitempty"`
	Scopes      models.UserPermissions `json:"scopes,omitempty"` // Allow updating scopes
}

// DeleteTokenPayload defines the structure for deleting a token.
type DeleteTokenPayload struct {
	TokenID string `json:"token_id" binding:"required"`
}

// CreateTokenHandler handles the creation of a new API token for the authenticated user.
// POST /api/v1/users/token
func CreateTokenHandler(c *gin.Context) {
	// 1. Get User ID from authenticated context (set by AuthMiddleware from 'sub' claim)
	userIDVal, exists := c.Get(middleware.ContextKeyUserID) // Use exported constant
	if !exists {
		server.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		server.ErrorResponse(c, http.StatusUnauthorized, "Invalid User ID format in context")
		return
	}

	// 2. Bind request payload
	var payload models.CreateAPITokenPayload // Use models.CreateAPITokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// 3. Load user data (needed for JWT secret and scope validation)
	userInstance, userExists, err := user.GetUserByID(userIDStr) // Use GetUserByID
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data: "+err.Error())
		return
	}
	if !userExists {
		server.ErrorResponse(c, http.StatusNotFound, "Authenticated user not found") // Correct error message now
		return
	}

	// 4. Call the token service to create the token metadata and JWT
	// This now also handles storing the metadata in tokenstore
	tokenID, apiTokenMeta, signedTokenString, err := CreateAPIToken(userInstance, payload)
	if err != nil {
		if strings.Contains(err.Error(), "exceed user permissions") || strings.Contains(err.Error(), "invalid duration format") {
			server.ErrorResponse(c, http.StatusBadRequest, err.Error())
		} else if strings.Contains(err.Error(), "failed to store token metadata") {
			log.Printf("ERROR: Failed to store token metadata for user %s: %v", userIDStr, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to store token metadata")
		} else {
			log.Printf("ERROR: Failed to create API token for user %s: %v", userIDStr, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to create API token: "+err.Error())
		}
		return
	}

	// 7. Prepare and return the success response
	responseData := map[string]interface{}{ // Use map[string]interface{} for flexibility
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

// GetTokensHandler handles retrieving all or a specific API token metadata for the user.
// Reads optional 'id' from request body to determine mode.
// Checks 'api:read:list' for all tokens, 'api:read:item' for a specific token.
func GetTokensHandler(c *gin.Context) {
	// 1. Get User ID from authenticated context
	userIDVal, exists := c.Get(middleware.ContextKeyUserID) // Use exported constant
	if !exists {
		server.ErrorResponse(c, http.StatusUnauthorized, "User ID not found in context")
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		server.ErrorResponse(c, http.StatusUnauthorized, "Invalid User ID format in context")
		return
	}

	// 2. Try to read and bind the request body to get optional ID
	var payload GetTokenPayload
	// Note: Binding JSON for GET is non-standard. Consider query params later.
	if err := c.ShouldBindJSON(&payload); err != nil && err.Error() != "EOF" { // Allow empty body (EOF)
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload format: "+err.Error())
		return
	}

	// 3. Determine action based on presence of ID in payload
	if payload.ID != "" {
		// --- Get Specific Token ---

		// 3a. Check permission for reading specific item
		if !middleware.CheckPermission(c, "api", "read:item") { // Corrected permission check
			return
		}

		// 3b. Get token info from store
		tokenInfo, found, err := tokenstore.GetTokenInfo(payload.ID)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve token info: "+err.Error())
			return
		}
		if !found {
			server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("API token with ID '%s' not found", payload.ID))
			return
		}

		// 3c. Verify ownership
		if tokenInfo.UserID != userIDStr {
			server.ErrorResponse(c, http.StatusForbidden, "You do not have permission to view this token")
			return
		}

		// 3d. Check if revoked
		isRevoked, err := tokenstore.IsTokenRevoked(payload.ID)
		if err != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to check token revocation status: "+err.Error())
			return
		}
		if isRevoked {
			server.ErrorResponse(c, http.StatusNotFound, "Token not found (or has been revoked)")
			return
		}

		// 3e. Prepare and return response for the single token
		responseData := map[string]interface{}{
			"id":   payload.ID,
			"name": tokenInfo.Name,
			// "description": tokenInfo.Description, // TODO: Add Description to TokenInfo
			"scopes":     tokenInfo.Scopes,
			"created_at": tokenInfo.CreatedAt.Time().Format(time.RFC3339),
			"expires_at": tokenInfo.ExpiresAt.Time().Format(time.RFC3339),
		}
		server.SuccessResponse(c, "API token retrieved successfully", responseData)

	} else {
		// --- Get All Tokens (List) ---

		// 4a. Check permission for listing items
		if !middleware.CheckPermission(c, "api", "read:list") {
			return
		}

		// 4b. Call tokenstore to get user's tokens
		tokensInfo, tokenIDs, err := tokenstore.GetUserTokens(userIDStr)
		if err != nil {
			log.Printf("Error retrieving tokens for user %s: %v", userIDStr, err)
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve tokens")
			return
		}

		// 4c. Prepare the response data list
		tokenList := make([]map[string]interface{}, len(tokensInfo))
		for i, info := range tokensInfo {
			tokenList[i] = map[string]interface{}{
				"id":   tokenIDs[i],
				"name": info.Name,
				// "description": info.Description, // TODO
				"scopes":     info.Scopes,
				"created_at": info.CreatedAt.Time().Format(time.RFC3339),
				"expires_at": info.ExpiresAt.Time().Format(time.RFC3339),
			}
		}

		// 4d. Return the list
		server.SuccessResponse(c, "API tokens retrieved successfully", tokenList)
	}
}

// UpdateTokenHandler handles updating an API token's metadata.
// PUT /api/v1/users/token
func UpdateTokenHandler(c *gin.Context) {
	// 1. Get User ID from context
	userIDVal, exists := c.Get(middleware.ContextKeyUserID) // Use exported constant
	if !exists {                                            /* ... error handling ... */
		server.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" { /* ... error handling ... */
		server.ErrorResponse(c, http.StatusUnauthorized, "Invalid User ID format in context")
		return
	}

	// 2. Bind request payload
	var payload UpdateTokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// 3. Permission Check (Handled by RequirePermission middleware)

	// 4. Load user data (needed ONLY if validating scopes)
	var userInstance models.User // Declare userInstance variable
	var userErr error
	var userExists bool
	if payload.Scopes != nil && len(payload.Scopes) > 0 {
		userInstance, userExists, userErr = user.GetUserByID(userIDStr)
		if userErr != nil {
			server.ErrorResponse(c, http.StatusInternalServerError, "Failed to load user data for scope validation: "+userErr.Error())
			return
		}
		if !userExists {
			server.ErrorResponse(c, http.StatusNotFound, "Authenticated user not found for scope validation")
			return
		}
	}

	// 5. Get current token info, check existence and ownership
	currentTokenInfo, found, err := tokenstore.GetTokenInfo(payload.ID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve current token info: "+err.Error())
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, fmt.Sprintf("API token with ID '%s' not found", payload.ID))
		return
	}
	if currentTokenInfo.UserID != userIDStr {
		server.ErrorResponse(c, http.StatusForbidden, "You do not have permission to update this token")
		return
	}

	// 6. Check if revoked
	isRevoked, err := tokenstore.IsTokenRevoked(payload.ID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to check token revocation status: "+err.Error())
		return
	}
	if isRevoked {
		server.ErrorResponse(c, http.StatusConflict, "Cannot update a revoked token")
		return
	}

	// 7. Prepare updates and validate scopes if necessary
	updatedInfo := currentTokenInfo // Start with current info
	needsResave := false
	// Removed: Re-issuing JWT logic is complex and likely not needed just for metadata update.
	// The original JWT remains valid.

	if payload.Name != nil {
		updatedInfo.Name = *payload.Name
		needsResave = true
	}
	// TODO: Add Description to TokenInfo struct and handle update here
	// if payload.Description != nil { updatedInfo.Description = *payload.Description; needsResave = true }

	if payload.Scopes != nil && len(payload.Scopes) > 0 {
		// Use the validateScopes function from token_service implicitly via CreateAPIToken or explicitly?
		// For now, let's use the local helper, assuming it will be replaced.
		if !validateScopesHelper(payload.Scopes, userInstance.Scopes) { // Use helper
			server.ErrorResponse(c, http.StatusBadRequest, "Requested scopes exceed user permissions")
			return
		}
		updatedInfo.Scopes = payload.Scopes
		needsResave = true
	}

	if !needsResave {
		server.ErrorResponse(c, http.StatusBadRequest, "No updateable fields provided (name, description, scopes allowed)")
		return
	}

	// 8. Save updated info back to tokenstore
	if err := tokenstore.StoreToken(payload.ID, updatedInfo); err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to update token metadata: "+err.Error())
		return
	}

	// 9. Removed: Saving updated user data (user object wasn't modified)

	// 10. Return success response with updated metadata
	responseData := map[string]interface{}{ // Use map
		"id":   payload.ID,
		"name": updatedInfo.Name,
		// "description": updatedInfo.Description, // TODO: Add when description is handled
		"scopes":     updatedInfo.Scopes,
		"created_at": updatedInfo.CreatedAt.Time().Format(time.RFC3339),
		"expires_at": updatedInfo.ExpiresAt.Time().Format(time.RFC3339),
	}
	server.SuccessResponse(c, "API token metadata updated successfully", responseData)
}

// DeleteTokenHandler handles deleting (revoking) an API token.
// Requires "api:delete" permission.
func DeleteTokenHandler(c *gin.Context) {
	// 1. Extract user info from context (set by AuthMiddleware)
	userIDVal, exists := c.Get(middleware.ContextKeyUserID) // Use exported constant
	if !exists {
		server.ErrorResponse(c, http.StatusUnauthorized, "User ID missing from context")
		return
	}
	userID := userIDVal.(string) // Assuming User ID is always string

	// 2. Bind the request payload
	var payload DeleteTokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// 3. Check if the token belongs to the user making the request
	tokenInfo, found, err := tokenstore.GetTokenInfo(payload.TokenID)
	if err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Error checking token ownership: "+err.Error())
		return
	}
	if !found {
		server.ErrorResponse(c, http.StatusNotFound, "Token not found")
		return
	}
	if tokenInfo.UserID != userID {
		server.ErrorResponse(c, http.StatusForbidden, "You do not have permission to delete this token")
		return
	}

	// 4. Call tokenstore to revoke the token
	if err := tokenstore.RevokeToken(payload.TokenID); err != nil {
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to revoke token: "+err.Error())
		return
	}

	// 5. Return success response
	server.SuccessResponse(c, "API token revoked successfully", nil) // No data needed on success
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
