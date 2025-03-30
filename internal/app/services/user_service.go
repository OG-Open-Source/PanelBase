package services

import (
	"errors"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/models"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// UserService handles user-related operations
type UserService struct {
	usersConfig *models.UsersConfig
	jwtSecret   string
}

// NewUserService creates a new UserService
func NewUserService(usersConfig *models.UsersConfig) *UserService {
	return &UserService{
		usersConfig: usersConfig,
		jwtSecret:   usersConfig.JWTSecret,
	}
}

// Authenticate authenticates a user with username and password
func (s *UserService) Authenticate(username, password string) (*models.User, error) {
	for _, user := range s.usersConfig.Users {
		if user.Username == username {
			err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
			if err == nil {
				// Update last login time
				user.LastLogin = time.Now()
				return user, nil
			}
			return nil, errors.New("invalid password")
		}
	}
	return nil, errors.New("user not found")
}

// GenerateJWT generates a JWT token for a user
func (s *UserService) GenerateJWT(user *models.User, userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   userID,
		"user": user.Username,
		"role": user.Role,
		"exp":  time.Now().Add(time.Hour * 24).Unix(),
	})

	return token.SignedString([]byte(s.jwtSecret))
}

// GetUserByID gets a user by ID
func (s *UserService) GetUserByID(id string) (*models.User, error) {
	if user, ok := s.usersConfig.Users[id]; ok {
		return user, nil
	}
	return nil, errors.New("user not found")
}

// GetUserByUsername gets a user by username
func (s *UserService) GetUserByUsername(username string) (*models.User, error) {
	for _, user := range s.usersConfig.Users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, errors.New("user not found")
}
