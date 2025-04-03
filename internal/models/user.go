package models

import (
	"time"
)

// APIToken represents an individual API token for a user.
type APIToken struct {
	Token       string          `json:"token"`
	Description string          `json:"description,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	ExpiresAt   *time.Time      `json:"expires_at,omitempty"` // Pointer for optional expiration
	Scopes      UserPermissions `json:"scopes"`               // Scopes granted to this specific token
	LastUsed    *time.Time      `json:"last_used,omitempty"`  // Pointer for optional last used time
}

// UserAPISettings holds the API-related settings for a user, specifically their tokens.
type UserAPISettings struct {
	Tokens []APIToken `json:"tokens"` // Array to hold multiple API tokens
}

// UserPermissions defines the structure for user permissions.
// It maps resource names (e.g., "commands", "users") to a list of allowed actions
// (e.g., "read:list", "create", "delete").
type UserPermissions map[string][]string

// User represents a user account in the system.
type User struct {
	IsActive  bool            `json:"is_active"`
	Username  string          `json:"username"`
	Password  string          `json:"password"` // This is the hashed password
	Name      string          `json:"name"`
	Email     string          `json:"email"`
	CreatedAt time.Time       `json:"created_at"`
	Scopes    UserPermissions `json:"scopes"` // User's base permissions
	// Update the API field to use the new UserAPISettings struct
	API UserAPISettings `json:"api"`
	// ID string `json:"-"` // The map key in UsersConfig.Users serves as the ID (e.g., "usr_admin123")
}
