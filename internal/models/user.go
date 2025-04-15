package models

import (
	"time"
)

// User represents a user in the system.
type User struct {
	ID        string                 `json:"id"`
	Username  string                 `json:"username"`
	Password  string                 `json:"password"`
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	CreatedAt time.Time              `json:"created_at"`
	Active    bool                   `json:"active"`
	Scopes    map[string]interface{} `json:"scopes,omitempty"`
	ApiTokens []ApiToken             `json:"api_tokens,omitempty"`
}

// UserResponse represents a user in API responses (omits password hash).
type UserResponse struct {
	ID        string                 `json:"id"`
	Username  string                 `json:"username"`
	Name      string                 `json:"name"`
	Email     string                 `json:"email"`
	CreatedAt string                 `json:"created_at"` // Format as RFC3339
	Active    bool                   `json:"active"`
	Scopes    map[string]interface{} `json:"scopes"`
}

// NewUserResponse converts a models.User to UserResponse for safe API exposure.
func NewUserResponse(user *User) UserResponse {
	// Handle potential nil Scopes map from the model
	scopes := user.Scopes
	if scopes == nil {
		scopes = make(map[string]interface{}) // Ensure response scope is not nil
	}
	return UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.UTC().Format(time.RFC3339), // Format as RFC3339
		Active:    user.Active,
		Scopes:    scopes,
	}
}
