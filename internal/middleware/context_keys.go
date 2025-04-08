package middleware

// ContextKey defines a type for context keys to avoid collisions.
type ContextKey string

// Define constants for context keys used throughout the middleware and handlers.
const (
	// ContextKeyUserID holds the user ID after successful authentication.
	ContextKeyUserID ContextKey = "userID"

	// ContextKeyPermissions holds the user's permissions map after successful authentication.
	ContextKeyPermissions ContextKey = "userPermissions"

	// ContextKeyAudience holds the audience claim from the JWT.
	ContextKeyAudience ContextKey = "audience"

	// ContextKeyRequestBody holds the cached request body.
	ContextKeyRequestBody ContextKey = "requestBody"

	// ContextKeyJTI holds the JWT ID (jti claim).
	ContextKeyJTI ContextKey = "tokenJTI"
)
