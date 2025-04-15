package handlers

import (
	"errors"
	"log"
	"net/http"
	"reflect" // Import reflect for type assertion

	"github.com/OG-Open-Source/PanelBase/internal/api/v1/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/auth"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/storage"
	"github.com/OG-Open-Source/PanelBase/pkg/response"

	// Import the config rules struct definition location (assuming it's in main for now)
	// This might need adjustment if the config struct is refactored.
	// main "github.com/OG-Open-Source/PanelBase/cmd/server"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

// Removed defaultUserScopes as it now comes from config
/*
var defaultUserScopes = map[string]interface{}{
	"account": map[string]interface{}{
		"profile": []string{"read", "update"},
		// TODO: Add password update scope here when implemented
		// "password": []string{"update"},
	},
}
*/

// UserHandler handles user management API requests.
type UserHandler struct {
	store         storage.UserStore
	defaultScopes map[string]interface{} // Store default scopes from config
	authRules     interface{}            // Store auth rules as interface{}
}

// NewUserHandler creates a new UserHandler.
// Accept authRules as interface{} to avoid import cycle.
func NewUserHandler(store storage.UserStore, defaultScopes map[string]interface{}, authRules interface{}) *UserHandler {
	return &UserHandler{
		store:         store,
		defaultScopes: defaultScopes,
		authRules:     authRules,
	}
}

// Helper function to safely get AuthConfigRules from the interface{}
func (h *UserHandler) getAuthConfigRules() (protectedUserIDs []string, allowSelfDelete bool, requireOldPassword bool) {
	// Set defaults in case type assertion fails
	protectedUserIDs = []string{} // Default to empty slice
	allowSelfDelete = true
	requireOldPassword = true

	if h.authRules == nil {
		log.Printf("WARN: Auth rules are nil in UserHandler. Using default rules.")
		return
	}

	// Use reflection to access fields without direct import
	val := reflect.ValueOf(h.authRules)
	if val.Kind() != reflect.Struct {
		log.Printf("WARN: Auth rules in UserHandler are not a struct (%s). Using default rules.", val.Kind())
		return
	}

	preventField := val.FieldByName("ProtectedUserIDs") // Read the new field
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

// --- DTOs (Data Transfer Objects) ---

type CreateUserRequest struct {
	Username string                 `json:"username" binding:"required"`
	Password string                 `json:"password" binding:"required,min=8"` // Add password validation
	Name     string                 `json:"name" binding:"required"`
	Email    string                 `json:"email" binding:"required,email"`
	Active   bool                   `json:"active"` // Default to true if omitted? Handler decides.
	Scopes   map[string]interface{} `json:"scopes"` // Changed to map[string]interface{}
}

type UpdateUserRequest struct {
	// Allow updating Name, Email, Active status, and Scopes
	// Username updates might be complex, consider disallowing or handling carefully
	Name   *string                 `json:"name"` // Use pointers to distinguish between empty and not provided
	Email  *string                 `json:"email" binding:"omitempty,email"`
	Active *bool                   `json:"active"`
	Scopes *map[string]interface{} `json:"scopes"` // Changed to *map[string]interface{}
	// Password changes should have a dedicated endpoint
	ApiTokens *[]models.ApiToken `json:"api_tokens"` // Allow updating API tokens (carefully!)
}

// UserResponse struct is now defined in models package
// Helper function newUserResponse is now models.NewUserResponse

// --- Handlers ---

// GetAllUsers godoc
// @Summary List all users
// @Description Get a list of all registered users.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Success 200 {object} response.ApiResponse{data=[]models.UserResponse}
// @Failure 500 {object} response.ApiResponse
// @Router /api/v1/users [get]
func (h *UserHandler) GetAllUsers(c *gin.Context) {
	users, err := h.store.GetAllUsers(c.Request.Context())
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to retrieve users", nil))
		return
	}

	respData := make([]models.UserResponse, len(users))
	for i, u := range users {
		respData[i] = models.NewUserResponse(&u)
	}
	c.JSON(http.StatusOK, response.Success("Users retrieved successfully", respData))
}

// CreateUser godoc
// @Summary Create a new user
// @Description Register a new user account.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param user body CreateUserRequest true "User data"
// @Success 201 {object} response.ApiResponse{data=models.UserResponse}
// @Failure 400 {object} response.ApiResponse
// @Failure 409 {object} response.ApiResponse
// @Failure 500 {object} response.ApiResponse
// @Router /api/v1/users [post]
func (h *UserHandler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid request body: "+err.Error(), nil))
		return
	}

	// Get claims of the user making the request
	claimsValue, exists := c.Get(middleware.UserClaimsKey)
	if !exists {
		log.Printf("ERROR: User claims not found in context during CreateUser.")
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error", nil))
		return
	}
	requesterClaims, ok := claimsValue.(*auth.Claims)
	if !ok || requesterClaims == nil {
		log.Printf("ERROR: Invalid claims type or nil claims in context during CreateUser.")
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error", nil))
		return
	}

	// Determine scopes for the new user
	var finalScopes map[string]interface{}
	canUpdateScopes := middleware.HasAction(requesterClaims.Scopes, "users:update", "scopes")

	if canUpdateScopes && req.Scopes != nil {
		// Admin can set scopes, use provided scopes
		finalScopes = req.Scopes
		log.Printf("DEBUG: Admin creating user with custom scopes: %v", finalScopes)
	} else {
		// Use default scopes from config if admin doesn't provide specific ones or if creator lacks permission
		finalScopes = h.defaultScopes // Use handler's default scopes
		if req.Scopes != nil && !canUpdateScopes {
			log.Printf("WARN: User creation request included scopes, but creator lacks 'users:update:scopes' permission. Applying default scopes.")
		}
		log.Printf("DEBUG: Creating user with default scopes: %v", finalScopes)
	}
	if finalScopes == nil {
		finalScopes = make(map[string]interface{})
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to hash password", nil))
		return
	}

	newUser := &models.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Name:     req.Name,
		Email:    req.Email,
		Active:   req.Active,
		Scopes:   finalScopes,
	}

	err = h.store.CreateUser(c.Request.Context(), newUser)
	if err != nil {
		if errors.Is(err, storage.ErrUserExists) {
			c.AbortWithStatusJSON(http.StatusConflict, response.Failure("Username already exists", nil))
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to create user", nil))
		}
		return
	}

	c.JSON(http.StatusCreated, response.Success("User created successfully", models.NewUserResponse(newUser)))
}

// GetUserByID godoc
// @Summary Get a user by ID
// @Description Retrieve information for a specific user.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (e.g., usr_...)"
// @Success 200 {object} response.ApiResponse{data=models.UserResponse}
// @Failure 404 {object} response.ApiResponse
// @Failure 500 {object} response.ApiResponse
// @Router /api/v1/users/{id} [get]
func (h *UserHandler) GetUserByID(c *gin.Context) {
	userID := c.Param("id")
	// // Basic validation for prefix - Removed, let storage handle not found
	// if !strings.HasPrefix(userID, userIDPrefix) {
	//     c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
	//     return
	// }

	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found", nil))
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to retrieve user", nil))
		}
		return
	}
	c.JSON(http.StatusOK, response.Success("User retrieved successfully", models.NewUserResponse(user)))
}

// UpdateUser godoc
// @Summary Update a user
// @Description Update information for an existing user (Name, Email, Active, Scopes).
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (e.g., usr_...)"
// @Param user body UpdateUserRequest true "User data to update"
// @Success 200 {object} response.ApiResponse{data=models.UserResponse}
// @Failure 400 {object} response.ApiResponse
// @Failure 404 {object} response.ApiResponse
// @Failure 409 {object} response.ApiResponse
// @Failure 500 {object} response.ApiResponse
// @Router /api/v1/users/{id} [patch]
func (h *UserHandler) UpdateUser(c *gin.Context) {
	userID := c.Param("id")
	// // Basic validation for prefix - Removed
	// if !strings.HasPrefix(userID, userIDPrefix) {
	//     c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID format"})
	//     return
	// }

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, response.Failure("Invalid request body: "+err.Error(), nil))
		return
	}

	// Fetch the existing user
	user, err := h.store.GetUserByID(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found", nil))
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to retrieve user for update", nil))
		}
		return
	}

	// Apply updates from request (only if field is present in request)
	// Note: Permission checks for each field are handled by the middleware chain
	updated := false
	if req.Name != nil {
		user.Name = *req.Name
		updated = true
	}
	if req.Email != nil {
		user.Email = *req.Email
		updated = true
	}
	if req.Active != nil {
		user.Active = *req.Active
		updated = true
	}

	// Scopes update is allowed here because the middleware chain already verified
	// the necessary permission ("users:update:scopes") before reaching the handler.
	if req.Scopes != nil {
		user.Scopes = *req.Scopes // Assign the new map[string]interface{}
		updated = true
	}

	// API Tokens update (Requires users:update:api_tokens scope, checked by middleware chain)
	if req.ApiTokens != nil {
		// **Warning:** Replacing the entire list can be dangerous.
		// Consider if separate add/delete endpoints for tokens under /users/:id/tokens are better.
		// For now, we allow replacing the whole slice if the scope is present.
		user.ApiTokens = *req.ApiTokens
		updated = true
		log.Printf("DEBUG: Updating API tokens for user %s (requires users:update:api_tokens scope)", userID)
	}

	if !updated {
		c.JSON(http.StatusOK, response.Success("No changes detected", models.NewUserResponse(user))) // Nothing to update, return current state
		return
	}

	// Attempt to update in store (UpdateUser handles preserving password etc.)
	err = h.store.UpdateUser(c.Request.Context(), user)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) { // Should not happen if GetUserByID succeeded, but check anyway
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found during update", nil))
		} else if errors.Is(err, storage.ErrUserExists) { // Handle potential username conflicts if username updates were allowed
			c.AbortWithStatusJSON(http.StatusConflict, response.Failure("Username conflict", nil))
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to update user", nil))
		}
		return
	}

	c.JSON(http.StatusOK, response.Success("User updated successfully", models.NewUserResponse(user)))
}

// DeleteUser godoc
// @Summary Delete a user
// @Description Permanently delete a user account.
// @Tags users
// @Produce json
// @Security BearerAuth
// @Param id path string true "User ID (e.g., usr_...)"
// @Success 200 {object} response.ApiResponse
// @Failure 400 {object} response.ApiResponse
// @Failure 404 {object} response.ApiResponse
// @Failure 500 {object} response.ApiResponse
// @Router /api/v1/users/{id} [delete]
func (h *UserHandler) DeleteUser(c *gin.Context) {
	userID := c.Param("id")

	// Get ID of the user making the request
	requesterIDVal, exists := c.Get(middleware.UserIDKey)
	if !exists {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error: User ID missing", nil))
		return
	}
	requesterID, ok := requesterIDVal.(string)
	if !ok || requesterID == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Authentication context error: Invalid User ID", nil))
		return
	}

	// Get rules using helper
	protectedIDsRule, _, _ := h.getAuthConfigRules()

	// Check if trying to delete self
	if userID == requesterID {
		// Check if self-delete is allowed via this endpoint (usually not, handled by account handler)
		c.AbortWithStatusJSON(http.StatusForbidden, response.Failure("Cannot delete own account via this endpoint. Use account management endpoints.", nil))
		return
	}

	// Check if deleting a protected user ID
	for _, protectedID := range protectedIDsRule {
		if userID == protectedID {
			log.Printf("WARN: Attempt to delete protected user ID (%s) blocked by configuration.", userID)
			c.AbortWithStatusJSON(http.StatusForbidden, response.Failure("Deleting this user is forbidden by configuration.", nil))
			return
		}
	}

	err := h.store.DeleteUser(c.Request.Context(), userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, response.Failure("User not found", nil))
		} else {
			c.AbortWithStatusJSON(http.StatusInternalServerError, response.Failure("Failed to delete user", nil))
		}
		return
	}

	c.JSON(http.StatusOK, response.Success("User deleted successfully", nil))
}

// Need to define models.Timestamp or remove its usage if not available
// Example placeholder if not defined elsewhere:
/*
package models
import "time"

type Timestamp struct {
    time.Time
}

func (t Timestamp) MarshalJSON() ([]byte, error) {
    return []byte(t.Time.Format(`"` + time.RFC3339Nano + `"`)), nil
}
*/
