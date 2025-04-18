package handlers

import (
	"errors"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time" // Need time package

	"github.com/OG-Open-Source/PanelBase/internal/api/v1/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/auth" // Need auth package for Claims type
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/storage" // Need utils for ID generation
	"github.com/OG-Open-Source/PanelBase/pkg/response"     // Import response package
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
	store     storage.UserStore
	authRules interface{} // Store auth rules as interface{}
}

// NewAccountHandler creates a new AccountHandler.
func NewAccountHandler(store storage.UserStore, authRules interface{}) *AccountHandler {
	return &AccountHandler{
		store:     store,
		authRules: authRules,
	}
}

// Helper function to safely get AuthConfigRules from the interface{} (similar to UserHandler)
func (h *AccountHandler) getAuthConfigRules() (protectedUserIDs []string, allowSelfDelete bool, requireOldPassword bool) {
	// Set defaults in case type assertion fails
	protectedUserIDs = []string{}
	allowSelfDelete = true
	requireOldPassword = true

	if h.authRules == nil {
		log.Printf("WARN: Auth rules are nil in AccountHandler. Using default rules.")
		return
	}

	// Use reflection to access fields without direct import
	val := reflect.ValueOf(h.authRules)
	if val.Kind() != reflect.Struct {
		log.Printf("WARN: Auth rules in AccountHandler are not a struct (%s). Using default rules.", val.Kind())
		return
	}

	preventField := val.FieldByName("ProtectedUserIDs")
	if preventField.IsValid() && preventField.Kind() == reflect.Slice {
		// Iterate through the slice and convert to string
		protectedIDs := []string{}
		for i := 0; i < preventField.Len(); i++ {
			elem := preventField.Index(i)
			if elem.Kind() == reflect.String {
				protectedIDs = append(protectedIDs, elem.String())
			}
		}
		protectedUserIDs = protectedIDs // Assign the successfully extracted slice
	}

	allowField := val.FieldByName("AllowSelfDelete")
	if allowField.IsValid() && allowField.Kind() == reflect.Bool {
		allowSelfDelete = allowField.Bool()
	}

	requireOldField := val.FieldByName("RequireOldPasswordForUpdate")
	if requireOldField.IsValid() && requireOldField.Kind() == reflect.Bool {
		requireOldPassword = requireOldField.Bool()
	}

	return
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
// @Success 200 {object} response.ApiResponse{data=models.UserResponse} "User profile details"
// @Failure 401 {object} response.ApiResponse "Unauthorized"
// @Failure 404 {object} response.ApiResponse "User not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Router /api/v1/account/profile [get]
func (h *AccountHandler) GetProfile(c *gin.Context) {
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error: User ID missing", nil))
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error: Invalid User ID", nil))
		return
	}

	user, err := h.store.GetUserByID(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("WARN: User %s not found in store after successful authentication.", userIDStr)
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found", nil))
		} else {
			log.Printf("ERROR: Failed to retrieve user %s: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to retrieve user profile", nil))
		}
		return
	}

	c.JSON(http.StatusOK, response.Success("Profile retrieved successfully", models.NewUserResponse(user)))
}

// UpdateProfile godoc
// @Summary Update own profile
// @Description Update the profile information (e.g., name, email) of the currently authenticated user.
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param profile body UpdateProfileRequest true "Profile data to update"
// @Success 200 {object} response.ApiResponse{data=models.UserResponse} "Updated user profile details"
// @Failure 400 {object} response.ApiResponse "Invalid request body"
// @Failure 401 {object} response.ApiResponse "Unauthorized"
// @Failure 403 {object} response.ApiResponse "Forbidden (if trying to update forbidden field)"
// @Failure 404 {object} response.ApiResponse "User not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Router /api/v1/account/profile [patch]
func (h *AccountHandler) UpdateProfile(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: User ID missing", nil))
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: Invalid User ID", nil))
		return
	}

	// 2. Bind JSON request body
	var req UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid request body: "+err.Error(), nil))
		return
	}

	// 3. Fetch user from store
	user, err := h.store.GetUserByID(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found", nil))
		} else {
			log.Printf("ERROR: Failed to retrieve user %s for update: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to retrieve user profile", nil))
		}
		return
	}

	// 4. Apply updates
	updated := false
	if req.Name != nil {
		user.Name = *req.Name
		updated = true
	}
	if req.Email != nil {
		user.Email = *req.Email
		updated = true
	}

	// 5. If nothing was actually updated
	if !updated {
		c.JSON(http.StatusOK, response.Success("No profile changes detected", models.NewUserResponse(user)))
		return
	}

	// 6. Save updated user to store
	err = h.store.UpdateUser(c.Request.Context(), user)
	if err != nil {
		log.Printf("ERROR: Failed to update user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to update profile", nil))
		return
	}

	// 7. Return updated UserResponse
	c.JSON(http.StatusOK, response.Success("Profile updated successfully", models.NewUserResponse(user)))
}

// UpdatePassword godoc
// @Summary Change own password
// @Description Change the password for the currently authenticated user.
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param password_data body UpdatePasswordRequest true "Old and new password"
// @Success 200 {object} response.ApiResponse
// @Failure 400 {object} response.ApiResponse "Invalid request body or new password too weak"
// @Failure 401 {object} response.ApiResponse "Unauthorized or incorrect old password"
// @Failure 404 {object} response.ApiResponse "User not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Router /api/v1/account/password [patch]
func (h *AccountHandler) UpdatePassword(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: User ID missing", nil))
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: Invalid User ID", nil))
		return
	}

	// Get rules
	_, _, requireOldPasswordRule := h.getAuthConfigRules()

	// 2. Bind JSON request body
	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid request body: "+err.Error(), nil))
		return
	}

	// 3. Fetch user from store
	user, err := h.store.GetUserByID(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found", nil))
		} else {
			log.Printf("ERROR: Failed to retrieve user %s for password update: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to retrieve user profile", nil))
		}
		return
	}

	// 4. Verify old password IF required by config
	if requireOldPasswordRule {
		if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
			log.Printf("DEBUG: Incorrect old password attempt for user %s (Rule Enabled)", userIDStr)
			c.AbortWithStatusJSON(http.StatusUnauthorized, response.Failure("Incorrect old password", nil))
			return
		}
		log.Printf("DEBUG: Old password verified for user %s (Rule Enabled)", userIDStr)
	} else {
		log.Printf("DEBUG: Skipping old password check for user %s (Rule Disabled)", userIDStr)
	}

	// 6. Hash the new password
	newHashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash new password for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to process new password", nil))
		return
	}

	// 7. Update user.Password
	user.Password = string(newHashedPassword)

	// 8. Save updated user to store
	err = h.store.UpdateUser(c.Request.Context(), user)
	if err != nil {
		log.Printf("ERROR: Failed to save updated password for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to update password", nil))
		return
	}

	// 10. Return success message
	log.Printf("INFO: User %s successfully updated their password.", userIDStr)
	c.JSON(http.StatusOK, response.Success("Password updated successfully", nil))
}

// CreateApiToken godoc
// @Summary Create API token
// @Description Create a new API token for the currently authenticated user.
// @Tags account
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param token_data body CreateApiTokenRequest true "API Token details"
// @Success 201 {object} response.ApiResponse{data=CreateApiTokenResponse} "Created API Token details (including secret - show once)"
// @Failure 400 {object} response.ApiResponse "Invalid request body"
// @Failure 401 {object} response.ApiResponse "Unauthorized"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Router /api/v1/account/tokens [post]
func (h *AccountHandler) CreateApiToken(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: User ID missing", nil))
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: Invalid User ID", nil))
		return
	}

	// 2. Bind request
	var req CreateApiTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid request body: "+err.Error(), nil))
		return
	}

	// 3. Generate secure random secret
	rawSecret, err := utils.GenerateSecureRandomString(apiSecretLength)
	if err != nil {
		log.Printf("ERROR: Failed to generate API token secret for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to generate token secret", nil))
		return
	}

	// 4. Hash secret
	hashedSecret, err := bcrypt.GenerateFromPassword([]byte(rawSecret), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("ERROR: Failed to hash API token secret for user %s: %v", userIDStr, err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to process token secret", nil))
		return
	}

	// 5. Generate token ID (tok_...)
	tokenID := auth.JtiPrefixToken + uuid.NewString()

	// 6. Create models.ApiToken struct
	apiToken := models.ApiToken{
		ID:         tokenID,
		Name:       req.Name,
		SecretHash: string(hashedSecret),
		CreatedAt:  time.Now().UTC().Truncate(time.Second),
		ExpiresAt:  req.ExpiresAt,
	}

	// 7. Call store.AddApiToken
	if err := h.store.AddApiToken(c.Request.Context(), userIDStr, apiToken); err != nil {
		log.Printf("ERROR: Failed to add API token %s for user %s: %v", tokenID, userIDStr, err)
		// Consider specific error for JTI conflict
		if strings.Contains(err.Error(), "already exists") {
			c.AbortWithStatusJSON(http.StatusConflict, response.Failure("API token ID conflict", nil))
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to save API token", nil))
		}
		return
	}

	// 8. Prepare response data (including raw secret)
	respData := CreateApiTokenResponse{
		ID:          apiToken.ID,
		Name:        apiToken.Name,
		CreatedAt:   apiToken.CreatedAt,
		ExpiresAt:   apiToken.ExpiresAt,
		TokenSecret: rawSecret,
	}

	log.Printf("INFO: Created API token '%s' (ID: %s) for user %s", apiToken.Name, apiToken.ID, userIDStr)
	c.JSON(http.StatusCreated, response.Success("API Token created successfully. Secret is shown only once.", respData))
}

// ListApiTokens godoc
// @Summary List API tokens
// @Description List all API tokens for the currently authenticated user.
// @Tags account
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse{data=[]ApiTokenListResponse} "List of API Tokens (secrets omitted)"
// @Failure 401 {object} response.ApiResponse "Unauthorized"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Router /api/v1/account/tokens [get]
func (h *AccountHandler) ListApiTokens(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: User ID missing", nil))
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: Invalid User ID", nil))
		return
	}

	// 2. Call store.GetUserApiTokens
	tokens, err := h.store.GetUserApiTokens(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("WARN: User %s not found in store trying to list tokens.", userIDStr)
			// It's not really a failure if user exists but has no tokens, return success with empty list
			c.JSON(http.StatusOK, response.Success("No API tokens found", []ApiTokenListResponse{}))
		} else {
			log.Printf("ERROR: Failed to get API tokens for user %s: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to retrieve API tokens", nil))
		}
		return
	}

	// 3. Convert []models.ApiToken to []ApiTokenListResponse
	respData := make([]ApiTokenListResponse, len(tokens))
	for i, t := range tokens {
		respData[i] = ApiTokenListResponse{
			ID:         t.ID,
			Name:       t.Name,
			CreatedAt:  t.CreatedAt,
			LastUsedAt: t.LastUsedAt,
			ExpiresAt:  t.ExpiresAt,
		}
	}

	// 4. Return list
	c.JSON(http.StatusOK, response.Success("API tokens retrieved successfully", respData))
}

// DeleteApiToken godoc
// @Summary Delete API token
// @Description Delete a specific API token for the currently authenticated user.
// @Tags account
// @Produce json
// @Security BearerAuth
// @Param tokenId path string true "ID of the token to delete (tok_...)"
// @Success 200 {object} response.ApiResponse "No Content"
// @Failure 401 {object} response.ApiResponse "Unauthorized"
// @Failure 404 {object} response.ApiResponse "Token not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Router /api/v1/account/tokens/{tokenId} [delete]
func (h *AccountHandler) DeleteApiToken(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: User ID missing", nil))
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: Invalid User ID", nil))
		return
	}

	// 2. Get tokenId from path parameter
	tokenID := c.Param("tokenId")
	if tokenID == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Token ID is required", nil))
		return
	}
	if !strings.HasPrefix(tokenID, auth.JtiPrefixToken) {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid token ID format", nil))
		return
	}

	// 3. Call store.DeleteApiToken
	if err := h.store.DeleteApiToken(c.Request.Context(), userIDStr, tokenID); err != nil {
		// TODO: Refine storage layer to return ErrTokenNotFound
		if strings.Contains(err.Error(), "not found") { // Simple check
			log.Printf("DEBUG: API token %s not found for deletion for user %s", tokenID, userIDStr)
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("API token not found", nil))
		} else if errors.Is(err, storage.ErrUserNotFound) {
			log.Printf("WARN: User %s not found trying to delete token %s", userIDStr, tokenID)
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found", nil))
		} else {
			log.Printf("ERROR: Failed to delete API token %s for user %s: %v", tokenID, userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to delete API token", nil))
		}
		return
	}

	// 4. Return 200 OK with Success response
	log.Printf("INFO: Deleted API token %s for user %s", tokenID, userIDStr)
	c.JSON(http.StatusOK, response.Success("API token deleted successfully", nil))
}

// Add a new handler for self-deletion

// DeleteSelf godoc
// @Summary Delete own account
// @Description Permanently delete the currently authenticated user's account.
// @Tags account
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse "Account deleted successfully"
// @Failure 401 {object} response.ApiResponse "Unauthorized"
// @Failure 403 {object} response.ApiResponse "Forbidden (self-delete disabled by config or trying to delete admin)"
// @Failure 404 {object} response.ApiResponse "User not found"
// @Failure 500 {object} response.ApiResponse "Internal server error"
// @Router /api/v1/account/delete [delete] // Define the route path
func (h *AccountHandler) DeleteSelf(c *gin.Context) {
	// 1. Get userID
	userIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: User ID missing", nil))
		return
	}
	userIDStr, ok := userIDVal.(string)
	if !ok || userIDStr == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Auth context error: Invalid User ID", nil))
		return
	}

	// 2. Get rules
	protectedIDsRule, allowSelfDeleteRule, _ := h.getAuthConfigRules()

	// 3. Check if self-delete is allowed by config
	if !allowSelfDeleteRule {
		log.Printf("WARN: Self-delete attempt by user %s blocked by configuration.", userIDStr)
		c.AbortWithStatusJSON(http.StatusForbidden, response.Failure("Account self-deletion is disabled by configuration.", nil))
		return
	}

	// 5. Check if deleting a protected user ID (even if it's self-delete)
	for _, protectedID := range protectedIDsRule {
		if userIDStr == protectedID {
			log.Printf("WARN: Protected user ID (%s) attempted self-delete, blocked by configuration.", userIDStr)
			c.AbortWithStatusJSON(http.StatusForbidden, response.Failure("Deleting this user account is forbidden by configuration, even via self-delete.", nil))
			return
		}
	}

	// 6. Perform deletion
	err := h.store.DeleteUser(c.Request.Context(), userIDStr)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			// Already handled above, but log just in case
			log.Printf("WARN: User %s not found during actual delete operation after checks.", userIDStr)
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found during deletion", nil))
		} else {
			log.Printf("ERROR: Failed to delete user %s: %v", userIDStr, err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to delete account", nil))
		}
		return
	}

	// 7. Return success
	log.Printf("INFO: User %s successfully deleted their own account.", userIDStr)
	c.JSON(http.StatusOK, response.Success("Account deleted successfully", nil))
}

// Note: Consider moving newUserResponse helper to a shared location if used by both UserHandler and AccountHandler.
// For now, assume it might be duplicated or refactored later.
