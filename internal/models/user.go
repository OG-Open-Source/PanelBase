package models

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system.
type User struct {
	ID           uuid.UUID           `json:"id"`
	Username     string              `json:"username"`
	PasswordHash string              `json:"password"`
	Name         string              `json:"name"`
	Email        string              `json:"email"`
	CreatedAt    time.Time           `json:"created_at"`
	Active       bool                `json:"active"`
	Scopes       map[string][]string `json:"scopes"`
}
