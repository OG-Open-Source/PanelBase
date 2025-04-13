package models

// Predefined scope constants for user management
const (
	// ScopeUsersRead grants permission to read user information (list all, get by ID).
	ScopeUsersRead = "users:read"
	// ScopeUsersWrite grants permission to create, update, and delete users.
	ScopeUsersWrite = "users:write"
	// ScopeAccountRead grants permission to read the authenticated user's own account information.
	ScopeAccountRead = "account:read"
	// ScopeAccountWrite grants permission to update the authenticated user's own account information.
	ScopeAccountWrite = "account:write"
)
