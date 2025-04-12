package storage

import (
	"context"
	"errors"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/google/uuid"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("username already exists")
)

// UserStore defines the interface for user data storage operations.
type UserStore interface {
	// CreateUser adds a new user to the store.
	CreateUser(ctx context.Context, user *models.User) error
	// GetUserByUsername retrieves a user by their username.
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	// GetUserByID retrieves a user by their ID.
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	// UpdateUser updates an existing user's data (e.g., permissions).
	// Note: Password updates should likely have a separate, more specific method.
	UpdateUser(ctx context.Context, user *models.User) error
	// DeleteUser removes a user from the store.
	DeleteUser(ctx context.Context, id uuid.UUID) error
}
 