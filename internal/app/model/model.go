package model

import (
	"github.com/golang-jwt/jwt/v5"
)

// User represents a user in the system
type User struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Password  string `json:"-"` // Password hash, not returned in JSON
	Role      string `json:"role"`
	Name      string `json:"name,omitempty"`
	Email     string `json:"email,omitempty"`
	Created   string `json:"created_at,omitempty"`
	LastLogin string `json:"last_login,omitempty"`
}

// Claims represents the JWT claims
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}

// Theme represents a UI theme configuration
type Theme struct {
	Name        string                 `json:"name"`
	Authors     interface{}            `json:"authors"` // Can be string or []string
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	SourceLink  string                 `json:"source_link"`
	Directory   string                 `json:"directory"`
	Structure   map[string]interface{} `json:"structure"`
}

// ThemeConfig represents the theme configuration file
type ThemeConfig struct {
	CurrentTheme string           `json:"current_theme"`
	Themes       map[string]Theme `json:"themes"`
}

// UsersConfig represents the users configuration file
type UsersConfig struct {
	Users       map[string]User `json:"users"`
	DefaultRole string          `json:"default_role"`
	JWTSecret   string          `json:"jwt_secret"`
}

// RoutesConfig represents the routes configuration file
type RoutesConfig struct {
	Routes map[string]string `json:"routes"`
}

// SystemInfo represents system information
type SystemInfo struct {
	Version   string `json:"version"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Uptime    string `json:"uptime"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token     string `json:"token"`
	ExpiresAt string `json:"expires_at"`
	User      User   `json:"user"`
}
