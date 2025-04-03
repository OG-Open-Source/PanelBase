package apitoken

import (
	"net/http"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/gin-gonic/gin"
)

// CreateTokenHandler handles the creation of a new API token for the authenticated user.
// POST /api/v1/users/token
func CreateTokenHandler(c *gin.Context) {
	// 1. Get User ID from authenticated context (set by AuthMiddleware from 'sub' claim)
	userIDVal, exists := c.Get("userID") // Get userID set by middleware
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
	var payload models.CreateAPITokenPayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		server.ErrorResponse(c, http.StatusBadRequest, "Invalid request payload: "+err.Error())
		return
	}

	// 3. Load user data using the user service with User ID
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
	tokenID, apiTokenMeta, signedTokenString, err := CreateAPIToken(userInstance, payload)
	if err != nil {
		// Handle specific errors like token limit reached, invalid scopes etc.
		// For now, return a generic internal server error or bad request depending on the error type.
		// TODO: Improve error handling based on the type of error from CreateAPIToken.
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to create API token: "+err.Error())
		return
	}

	// 5. Add the new token metadata to the user's token map
	if userInstance.API.Tokens == nil {
		userInstance.API.Tokens = make(map[string]models.APIToken)
	}
	userInstance.API.Tokens[tokenID] = apiTokenMeta // Add the metadata using tokenID as key

	// 6. Save the updated user data using the user service
	if err := user.UpdateUser(userInstance); err != nil {
		// TODO: Consider how to handle this failure. Maybe log it but still return the token?
		// For now, return an error as the state is inconsistent.
		server.ErrorResponse(c, http.StatusInternalServerError, "Failed to save updated user data: "+err.Error())
		return
	}

	// 7. Prepare and return the success response
	responseData := map[string]interface{}{
		"id":          tokenID, // Use the generated tokenID as the response ID
		"name":        apiTokenMeta.Name,
		"description": apiTokenMeta.Description,
		"scopes":      apiTokenMeta.Scopes,
		"created_at":  apiTokenMeta.CreatedAt.Format(time.RFC3339),
		"expires_at":  apiTokenMeta.ExpiresAt.Format(time.RFC3339),
		"token":       signedTokenString, // The actual JWT string for the user
	}

	server.SuccessResponse(c, "API token created successfully", responseData)
}

// GetTokensHandler retrieves all API tokens for the authenticated user.
// GET /api/v1/users/token
// ... existing code ...
