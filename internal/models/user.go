package models

import (
	"time"
)

// APIToken represents an API token associated with a user.
type APIToken struct {
	Name      string     `json:"name"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"` // Consider using pointer for optional expiration
	LastUsed  *time.Time `json:"last_used"`  // Use pointer for nullable time
	Scopes    []string   `json:"scopes"`     // Scopes specific to this token
	// Token string `json:"-"` // The actual token string is usually not stored directly here
	// ID string `json:"-"` // The map key in User.API serves as the ID
}

// User represents a user account in the system.
type User struct {
	IsActive  bool                `json:"is_active"`
	Username  string              `json:"username"`
	Password  string              `json:"password"` // Hashed password
	Name      string              `json:"name"`
	Email     string              `json:"email"`
	CreatedAt time.Time           `json:"created_at"`
	LastLogin *time.Time          `json:"last_login"` // Use pointer for nullable time
	Scopes    []string            `json:"scopes"`     // List of scopes granted to the user
	API       map[string]APIToken `json:"api"`        // API tokens, map key is the token ID (e.g., "tok_abc123")
	// ID string `json:"-"` // The map key in UsersConfig.Users serves as the ID (e.g., "usr_admin123")
}
