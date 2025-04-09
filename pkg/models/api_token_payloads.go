package models

import "time"

// RequestPayload defines the structure for identifying a resource, potentially for a specific user (admin).
type RequestPayload struct {
	ID     string `json:"id" binding:"required"`
	UserID string `json:"user_id,omitempty"` // Optional: Target user ID for admin actions
}

// CreateAPITokenRequest defines the expected JSON body for creating a token via the API handler.
type CreateAPITokenRequest struct {
	// UserID is NOT part of the request body, it's derived from the URL path for admin routes.
	Name        string   `json:"name" binding:"required"`
	Description string   `json:"description,omitempty"`       // Optional
	Duration    string   `json:"duration" binding:"required"` // Required: ISO 8601 Duration string (e.g., "P30D")
	Scopes      []string `json:"scopes,omitempty"`            // Optional: List of specific scope strings (e.g., "users:read", "settings:update")
}

// UpdateTokenPayload defines the structure for updating token metadata.
// Name and Description can be updated. Scopes update might need specific handling.
// Uses pointers to differentiate between empty and not provided.
type UpdateTokenPayload struct {
	ID          string  `json:"id" binding:"required"` // ID is required in body for clarity, but path param is authoritative
	UserID      string  `json:"user_id,omitempty"`     // Optional: Target user ID for admin actions
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// DeleteTokenPayload defines the structure for deleting a token.
// Body is optional, primarily uses URL params, but binding struct helps if body is sent.
type DeleteTokenPayload struct {
	ID     string `json:"id,omitempty"`      // Optional: ID from body (path param is authoritative)
	UserID string `json:"user_id,omitempty"` // Optional: Target user ID for admin actions
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
