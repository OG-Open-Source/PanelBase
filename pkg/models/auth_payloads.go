package models

// LoginPayload defines the expected JSON body for login requests.
type LoginPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterPayload defines the expected JSON body for registration requests.
type RegisterPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email"` // Optional
	Name     string `json:"name" binding:"required"`
}

// CreateAPITokenPayload defines the structure for creating an API token via the service layer.
// This is used internally, not directly bound from the API request in this specific format.
type CreateAPITokenPayload struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Duration    string          `json:"duration"` // ISO 8601 Duration string
	Scopes      UserPermissions `json:"scopes"`   // Use map[string][]string for scopes
}
