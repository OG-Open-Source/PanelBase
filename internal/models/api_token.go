package models

import "time"

// ApiToken represents an API token associated with a user.
type ApiToken struct {
	ID         string     `json:"id"`                     // Token ID (tok_...) - Also the JTI
	Name       string     `json:"name"`                   // User-defined name for the token
	SecretHash string     `json:"secret_hash"`            // Hashed secret (bcrypt)
	CreatedAt  time.Time  `json:"created_at"`             // Creation timestamp
	LastUsedAt *time.Time `json:"last_used_at,omitempty"` // Pointer for optional last used time
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`   // Pointer for optional expiration
	// Scopes are embedded in the JWT.
}
