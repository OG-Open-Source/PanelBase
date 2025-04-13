package storage

import (
	"context"
	"errors"

	"github.com/OG-Open-Source/PanelBase/internal/models"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("username already exists")
)

// UserStore defines the interface for user data storage operations.
type UserStore interface {
	// User CRUD operations
	CreateUser(ctx context.Context, user *models.User) error
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
	GetUserByID(ctx context.Context, id string) (*models.User, error)
	GetAllUsers(ctx context.Context) ([]models.User, error)
	UpdateUser(ctx context.Context, user *models.User) error
	DeleteUser(ctx context.Context, id string) error

	// API Token operations
	AddApiToken(ctx context.Context, userID string, token models.ApiToken) error
	GetUserApiTokens(ctx context.Context, userID string) ([]models.ApiToken, error)
	DeleteApiToken(ctx context.Context, userID string, tokenID string) error
	GetApiTokenByID(ctx context.Context, userID string, tokenID string) (*models.ApiToken, error) // For JTI lookup
}
