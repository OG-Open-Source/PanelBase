package middleware

// ContextKey type for context keys to avoid collisions
type ContextKey string

// Constants for context keys
const (
	ContextKeyUserID          ContextKey = "userID"
	ContextKeyUsername        ContextKey = "username"
	ContextKeyPermissions     ContextKey = "permissions"
	ContextKeyAudience        ContextKey = "audience"
	ContextKeyJTI             ContextKey = "jti"
	ContextKeyIsAuthenticated ContextKey = "isAuthenticated"
	ContextKeyRequestBody     ContextKey = "requestBody"
)