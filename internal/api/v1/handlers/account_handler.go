package handlers

import (
	"errors"
	"log"
	"net/http"
	"strings"
	"time" // Need time package

	"github.com/OG-Open-Source/PanelBase/internal/api/v1/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/auth" // Need auth package for Claims type
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/storage" // Need utils for ID generation
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt" // Need bcrypt

	// bcrypt needed for password update later
	"github.com/google/uuid"
)

const (
	apiSecretLength = 32 // Define desired length for generated API secrets
)

// AccountHandler handles API requests related to the user's own account.
type AccountHandler struct {
	store storage.UserStore
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(store storage.UserStore) *AccountHandler {
	return &AccountHandler{store: store}
}

// --- DTOs ---

// UpdateProfileRequest defines the structure for updating the user's own profile.
// Only include fields the user is allowed to change about themselves.
type UpdateProfileRequest struct {
	Name  *string `json:"name"` // Pointer to distinguish not provided vs empty
	Email *string `json:"email" binding:"omitempty,email"`
}

// UpdatePasswordRequest defines the structure for changing the user's own password.
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// CreateApiTokenRequest defines the structure for creating a new API token.
type CreateApiTokenRequest struct {
	Name      string     `json:"name" binding:"required"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"` // Optional expiration
	// Scopes will be inherited from the user at the time of use (if using ID/Secret auth)
}

// CreateApiTokenResponse defines the structure returned after creating an API token.
// It includes the raw secret which is only shown once.
type CreateApiTokenResponse struct {
	ID          string     `json:"id"`                   // Token ID (tok_...)
	Name        string     `json:"name"`                 // User-defined name
	CreatedAt   time.Time  `json:"created_at"`           // Creation timestamp
	ExpiresAt   *time.Time `json:"expires_at,omitempty"` // Optional expiration
	TokenSecret string     `json:"token_secret"`         // The raw secret (show once!)
}

// ApiTokenListResponse defines the structure for listing API tokens (omits secret).
type ApiTokenListResponse struct {
	ID         string     `json:"id"`         // Token ID (tok_...)
	Name       string     `json:"name"`       // User-defined name
	CreatedAt  time.Time  `json:"created_at"` // Creation timestamp
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

// --- Handlers ---

// GetProfile godoc
// @Summary Get own profile
// @Description Retrieve the profile information of the currently authenticated user.
// @Tags account
// @Produce json
// @Security BearerAuth
// @Success 200 {object} UserResponse "User profile details"
// @Failure 401 {object} gin.H{"error":string} "Unauthorized"
// @Failure 404 {object} gin.H{"error":string} "User not found"
// @Failure 500 {object} gin.H{"error":string} "Internal server error"
// @Router /api/v1/account/profile [get]
func (h *AccountHandler) GetProfile(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		log.Printf("ERROR: User ID not found in context during GetProfile.")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Authentication context error"})
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		log.Printf("ERROR: Invalid User ID type or empty in context during GetProfile.")
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Authentication context error"})
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("WARN: User %s not found in store after successful authentication.", userIDStr)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			log.Printf("ERROR: Failed to retrieve user %s: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		}
		return
	}

	c.JSON(http.StatusOK, models.NewUserResponse(user))
}

// UpdateProfile godoc
// @Summary Update own profile
// @Description Update the profile information (e.g., name, email) of the currently authenticated user.
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body UpdateProfileRequest true "Profile data to update"
// @Success 200 {object} UserResponse "Updated user profile details"
// @Failure 400 {object} gin.H{"error":string} "Invalid request body"
// @Failure 401 {object} gin.H{"error":string} "Unauthorized"
// @Failure 403 {object} gin.H{"error":string} "Forbidden (if trying to update forbidden field)"
// @Failure 404 {object} gin.H{"error":string} "User not found"
// @Failure 500 {object} gin.H{"error":string} "Internal server error"
// @Router /api/v1/account/profile [patch]
func (h *AccountHandler) UpdateProfile(c *gin.Context) {
	// 1. Get userID and claims from context
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: User ID missing"})
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: Invalid User ID"})
		return
	}

	claimsVal, exists := c.Get(middleware.UserClaimsKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: Claims missing"})
		return
	}
	claims, ok := claimsVal.(*auth.Claims)
	if !ok || claims == nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: Invalid Claims"})
		return
	}

	// 2. Bind JSON request body
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 3. Fetch user from store
	user, err := h.store.GetUserByID(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			log.Printf("ERROR: Failed to retrieve user %s for update: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		}
		return
	}

	// 4. Apply updates based on request and check permissions for each field
	updated := false
	// Check Name update
	if req.Name != nil {
		// Permission check for name update (already done by middleware for this route)
		// if !middleware.HasAction(claims.Scopes, "account:update", "name") { ... }
		user.Name = *req.Name
		updated = true
	}
	// Check Email update
	if req.Email != nil {
		// Permission check for email update (already done by middleware for this route)
		// if !middleware.HasAction(claims.Scopes, "account:update", "email") { ... }
		user.Email = *req.Email
		updated = true
	}

	// If nothing was actually updated (e.g., request body was empty or identical)
	if !updated {
		c.JSON(http.StatusOK, models.NewUserResponse(user))
		return
	}

	// 6. Save updated user to store
	err = h.store.UpdateUser(c.Request.Context(), user)
	if err != nil {
		// UpdateUser in storage should handle ErrUserExists if username was changeable
		log.Printf("ERROR: Failed to update user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// 8. Return updated UserResponse
	c.JSON(http.StatusOK, models.NewUserResponse(user))
}

// UpdatePassword godoc
// @Summary Change own password
// @Description Change the password for the currently authenticated user.
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param password_data body UpdatePasswordRequest true "Old and new password"
// @Success 200 {object} gin.H{"message":string} "Password updated successfully"
// @Failure 400 {object} gin.H{"error":string} "Invalid request body or new password too weak"
// @Failure 401 {object} gin.H{"error":string} "Unauthorized or incorrect old password"
// @Failure 404 {object} gin.H{"error":string} "User not found"
// @Failure 500 {object} gin.H{"error":string} "Internal server error"
// @Router /api/v1/account/password [patch]
func (h *AccountHandler) UpdatePassword(c *gin.Context) {
	// 1. Get userID from context
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: User ID missing"})
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: Invalid User ID"})
		return
	}

	// 2. Bind JSON request body
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 3. Fetch user from store
	user, err := h.store.GetUserByID(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			log.Printf("ERROR: Failed to retrieve user %s for password update: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve user profile"})
		}
		return
	}

	// 4. Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		// If bcrypt.ErrMismatchedHashAndPassword or other error
		log.Printf("DEBUG: Incorrect old password attempt for user %s", userIDStr)
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Incorrect old password"})
		return
	}

	// 6. Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash new password for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to process new password"})
		return
	}

	// 7. Update user.PasswordHash
	user.PasswordHash = string(newHashedPassword)

	// 8. Save updated user to store
	err = h.store.UpdateUser(c.Request.Context(), user)
	if err != nil {
		log.Printf("ERROR: Failed to save updated password for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to update password"})
		return
	}

	// 10. Return success message
	log.Printf("INFO: User %s successfully updated their password.", userIDStr)
	c.JSON(http.StatusOK, gin.H{"message": "Password updated successfully"})
}

// CreateApiToken godoc
// @Summary Create API token
// @Description Create a new API token for the currently authenticated user.
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param token_data body CreateApiTokenRequest true "API Token details"
// @Success 201 {object} CreateApiTokenResponse "Created API Token details (including secret - show once)"
// @Failure 400 {object} gin.H{"error":string} "Invalid request body"
// @Failure 401 {object} gin.H{"error":string} "Unauthorized"
// @Failure 500 {object} gin.H{"error":string} "Internal server error"
// @Router /api/v1/account/tokens [post]
func (h *AccountHandler) CreateApiToken(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: User ID missing"})
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: Invalid User ID"})
		return
	}

	// 2. Bind request
	var req CreateApiTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	// 3. Generate secure random secret
	rawSecret, err := utils.GenerateSecureRandomString(apiSecretLength)
	if err != nil {
		log.Printf("ERROR: Failed to generate API token secret for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token secret"})
		return
	}

	// 4. Hash secret
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash API token secret for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to process token secret"})
		return
	}

	// 5. Generate token ID (tok_...)
	// Note: We don't use GeneratePrefixedID here as the prefix is part of the standard JTI format for API tokens
	tokenID := auth.JtiPrefixToken + uuid.NewString() // Assuming google/uuid is imported in auth or utils

	// 6. Create models.ApiToken struct
	apiToken := models.ApiToken{
		ID:         tokenID,
		Name:       req.Name,
		SecretHash: string(hashedSecret),
		CreatedAt:  time.Now().UTC().Truncate(time.Second), // Use RFC3339 format
		ExpiresAt:  req.ExpiresAt,                          // Assign optional expiration from request
		// LastUsedAt will be nil initially
	}

	// 7. Call store.AddApiToken
	if err := h.store.AddApiToken(c.Request.Context(), userIDStr, apiToken); err != nil {
		log.Printf("ERROR: Failed to add API token %s for user %s: %v", tokenID, userIDStr, err)
		// Handle potential duplicate JTI error specifically?
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to save API token"})
		return
	}

	// 8. Return CreateApiTokenResponse with raw secret
	response := CreateApiTokenResponse{
		ID:          apiToken.ID,
		Name:        apiToken.Name,
		CreatedAt:   apiToken.CreatedAt,
		ExpiresAt:   apiToken.ExpiresAt,
		TokenSecret: rawSecret, // IMPORTANT: Show this only once!
	}

	log.Printf("INFO: Created API token '%s' (ID: %s) for user %s", apiToken.Name, apiToken.ID, userIDStr)
	c.JSON(http.StatusCreated, response)
}

// ListApiTokens godoc
// @Summary List API tokens
// @Description List all API tokens for the currently authenticated user.
// @Tags account
// @Produce json
// @Security BearerAuth
// @Success 200 {array} ApiTokenListResponse "List of API Tokens (secrets omitted)"
// @Failure 401 {object} gin.H{"error":string} "Unauthorized"
// @Failure 500 {object} gin.H{"error":string} "Internal server error"
// @Router /api/v1/account/tokens [get]
func (h *AccountHandler) ListApiTokens(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: User ID missing"})
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: Invalid User ID"})
		return
	}

	// 2. Call store.GetUserApiTokens
	tokens, err := h.store.GetUserApiTokens(c.Request.Context(), userIDStr)
	if err != nil {
		// Handle case where user doesn't exist (shouldn't happen after auth)
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("WARN: User %s not found in store trying to list tokens.", userIDStr)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			log.Printf("ERROR: Failed to get API tokens for user %s: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve API tokens"})
		}
		return
	}

	// 3. Convert []models.ApiToken to []ApiTokenListResponse (omitting secret hash)
	resp := make([]ApiTokenListResponse, len(tokens))
	for i, t := range tokens {
		resp[i] = ApiTokenListResponse{
			ID:         t.ID,
			Name:       t.Name,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
			ExpiresAt:  t.ExpiresAt,
		}
	}

	// 4. Return list
	c.JSON(http.StatusOK, resp)
}

// DeleteApiToken godoc
// @Summary Delete API token
// @Description Delete a specific API token for the currently authenticated user.
// @Tags account
// @Produce json
// @Security BearerAuth
// @Param tokenId path string true "ID of the token to delete (tok_...)"
// @Success 204 "No Content"
// @Failure 401 {object} gin.H{"error":string} "Unauthorized"
// @Failure 404 {object} gin.H{"error":string} "Token not found"
// @Failure 500 {object} gin.H{"error":string} "Internal server error"
// @Router /api/v1/account/tokens/{tokenId} [delete]
func (h *AccountHandler) DeleteApiToken(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: User ID missing"})
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Auth context error: Invalid User ID"})
		return
	}

	// 2. Get tokenId from path parameter
	tokenID := c.Param("tokenId")
	if tokenID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Token ID is required"})
		return
	}
	// Basic format check for JTI might be useful here
	if !strings.HasPrefix(tokenID, auth.JtiPrefixToken) {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid token ID format"})
		return
	}

	// 3. Call store.DeleteApiToken
	if err := h.store.DeleteApiToken(c.Request.Context(), userIDStr, tokenID); err != nil {
		// Check if the error is because the token wasn't found
		// TODO: Refine storage layer to return a specific ErrTokenNotFound
		if strings.Contains(err.Error(), "not found") { // Simple check for now
			log.Printf("DEBUG: API token %s not found for deletion for user %s", tokenID, userIDStr)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "API token not found"})
		} else if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("WARN: User %s not found trying to delete token %s", userIDStr, tokenID)
			c.AbortWithStatusJSON(http.StatusNotFound, gin.H{"error": "User not found"})
		} else {
			log.Printf("ERROR: Failed to delete API token %s for user %s: %v", tokenID, userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete API token"})
		}
		return
	}

	// 5. Return 204 No Content on success
	log.Printf("INFO: Deleted API token %s for user %s", tokenID, userIDStr)
	c.Status(http.StatusNoContent)
}

// Note: Consider moving newUserResponse helper to a shared location if used by both UserHandler and AccountHandler.
// For now, assume it might be duplicated or refactored later.
