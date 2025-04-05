package models

import "time"

// CreateAPITokenPayload defines the structure for the request body when creating a new API token.
// Username is optional for admin actions.
type CreateAPITokenPayload struct {
	Username    *string         `json:"username,omitempty"`          // Optional: Target username for admin actions
	Name        string          `json:"name" binding:"required"`     // Required: A unique name for this token
	Description string          `json:"description,omitempty"`       // Optional: A longer description
	Scopes      UserPermissions `json:"scopes,omitempty"`            // Optional: If empty, inherit target user's scopes. Validation happens in service.
	Duration    string          `json:"duration" binding:"required"` // Required: Duration in ISO 8601 format (Date part only, e.g., "P7D", "P1M", "P1Y6M")
}

// UpdateAPITokenPayload defines the structure for the request body when updating an API token.
type UpdateAPITokenPayload struct {
	ID          string           `json:"id" binding:"required"` // The ID of the token to update
	Description *string          `json:"description,omitempty"` // Optional: New description
	Scopes      *UserPermissions `json:"scopes,omitempty"`      // Optional: New permission subset
	ExpiresAt   *time.Time       `json:"expires_at,omitempty"`  // Optional: New expiration time (use pointer to allow clearing)
}

// DeleteAPITokenPayload defines the structure for the request body when deleting an API token.
type DeleteAPITokenPayload struct {
	ID string `json:"id" binding:"required"` // The ID of the token to delete
}
