package models

import (
	"time"
)

// APIToken represents an individual API token for a user.
type APIToken struct {
	Token       string     `json:"token"`
	Description string     `json:"description,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"` // Pointer for optional expiration
	Scopes      []string   `json:"scopes"`               // Scopes granted to this specific token
	LastUsed    *time.Time `json:"last_used,omitempty"`  // Pointer for optional last used time
}

// UserAPISettings holds the API-related settings for a user, specifically their tokens.
type UserAPISettings struct {
	Tokens []APIToken `json:"tokens"` // Array to hold multiple API tokens
}

// User represents a user account in the system.
type User struct {
	IsActive  bool      `json:"is_active"`
	Username  string    `json:"username"`
	Password  string    `json:"password"` // This is the hashed password
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	Scopes    []string  `json:"scopes"` // User's base scopes
	// Update the API field to use the new UserAPISettings struct
	API UserAPISettings `json:"api"`
	// ID string `json:"-"` // The map key in UsersConfig.Users serves as the ID (e.g., "usr_admin123")
}
