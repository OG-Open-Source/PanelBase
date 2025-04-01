package models

// UsersConfig represents the structure of the entire users.json file.
type UsersConfig struct {
	// Users map holds all user accounts, keyed by user ID (e.g., "usr_admin123").
	Users map[string]User `json:"users"`

	// JWTSecret is the secret key used for signing JWT tokens.
	// Note: This is loaded separately by the config package,
	// but included here for completeness of the file structure representation.
	JWTSecret string `json:"jwt_secret"`
}

// Note: We will need functions to load, parse, and potentially save this config.
// These functions might live in a different package like `internal/services/users`
// or `internal/storage`. For now, we only define the structure.
