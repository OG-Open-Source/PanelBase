package models

import (
	"time"
)

// UserPermissions defines the structure for user permissions (scopes).
// It maps resource names (string) to a list of allowed actions (string slice).
// Example: {"commands": ["list", "execute"], "users": ["read:self"]}
type UserPermissions map[string][]string

// UserAPISettings holds API-specific settings for a user.
type UserAPISettings struct {
	JwtSecret string              `json:"jwt_secret"` // User-specific secret for signing API tokens (JWTs)
	Tokens    map[string]APIToken `json:"tokens"`     // Map of API Tokens, keyed by Token ID (tok_...)
}

// APIToken represents the metadata for an API token.
// The actual token string (JWT) is not stored here.
// The ID is the key in the UserAPISettings.Tokens map.
type APIToken struct {
	// ID field removed - it's the key in the map
	Name        string          `json:"name"`                  // Required name for the token
	Description string          `json:"description,omitempty"` // Optional description
	Scopes      UserPermissions `json:"scopes"`                // Scopes granted to this token
	CreatedAt   time.Time       `json:"created_at"`            // Timestamp when the token was created
	ExpiresAt   time.Time       `json:"expires_at"`            // Timestamp when the token expires (Required, non-pointer)
	// Token string field removed
	// LastUsed    *time.Time      `json:"last_used,omitempty"` // Optional: Timestamp when the token was last used
}

// User defines the structure for a user account.
type User struct {
	ID        string          `json:"id"`                   // Unique user identifier (e.g., usr_...)
	Username  string          `json:"username"`             // Login username (must be unique)
	Password  string          `json:"password"`             // Hashed password
	Name      string          `json:"name,omitempty"`       // Optional display name
	Email     string          `json:"email,omitempty"`      // Optional email address
	CreatedAt time.Time       `json:"created_at"`           // Timestamp when the user was created
	Active    bool            `json:"active"`               // Whether the user account is active
	LastLogin *time.Time      `json:"last_login,omitempty"` // Pointer to timestamp of last successful login
	Scopes    UserPermissions `json:"scopes"`               // Base permissions granted to the user
	API       UserAPISettings `json:"api"`                  // API-specific settings (secret, tokens)
}

// UsersConfig defines the top-level structure for the users.json file.
type UsersConfig struct {
	JwtSecret string          `json:"jwt_secret"` // Optional global JWT secret (fallback?)
	Users     map[string]User `json:"users"`      // Map of users, keyed by username
}
