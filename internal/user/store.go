package user

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"golang.org/x/crypto/bcrypt"
)

// Errors
var (
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("user already exists")
)

// FileStore implements UserManager interface with file-based storage
type FileStore struct {
	users    map[int]*User
	nextID   int
	filePath string
	mutex    sync.RWMutex
}

// NewFileStore creates a new file-based user store
func NewFileStore(dataDir string) (*FileStore, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	filePath := filepath.Join(dataDir, "users.json")
	store := &FileStore{
		users:    make(map[int]*User),
		nextID:   1,
		filePath: filePath,
	}

	// Load existing users if file exists
	if _, err := os.Stat(filePath); err == nil {
		if err := store.load(); err != nil {
			return nil, err
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}

	// Create root user if no users exist
	if len(store.users) == 0 {
		logger.Info("No users found. Creating default root user.")
		// Default password is "admin" - should be changed immediately
		if _, err := store.CreateUser("admin", "admin", RoleRoot); err != nil {
			return nil, err
		}
	}

	return store, nil
}

// load users from file
func (s *FileStore) load() error {
	data, err := ioutil.ReadFile(s.filePath)
	if err != nil {
		return fmt.Errorf("failed to read users file: %w", err)
	}

	var users []*User
	if err := json.Unmarshal(data, &users); err != nil {
		return fmt.Errorf("failed to parse users file: %w", err)
	}

	s.users = make(map[int]*User)
	for _, user := range users {
		if user.Password == "" {
			logger.Warn(fmt.Sprintf("User %s (ID: %d) has no password. Setting default password.", user.Username, user.ID))
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
			if err != nil {
				return fmt.Errorf("failed to hash default password: %w", err)
			}
			user.Password = string(hashedPassword)
		}
		s.users[user.ID] = user
		if user.ID >= s.nextID {
			s.nextID = user.ID + 1
		}
	}

	if err := s.save(); err != nil {
		return fmt.Errorf("failed to save updated users: %w", err)
	}

	return nil
}

// save users to file
func (s *FileStore) save() error {
	var users []*User
	for _, user := range s.users {
		users = append(users, user)
	}

	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users: %w", err)
	}

	if err := ioutil.WriteFile(s.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write users file: %w", err)
	}

	return nil
}

// GetUser retrieves a user by ID
func (s *FileStore) GetUser(id int) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// GetUserByUsername retrieves a user by username
func (s *FileStore) GetUserByUsername(username string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, user := range s.users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

// GetUserByAPIKey retrieves a user by API key
func (s *FileStore) GetUserByAPIKey(apiKey string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for _, user := range s.users {
		if user.APIKey == apiKey {
			return user, nil
		}
	}
	return nil, ErrUserNotFound
}

// ListUsers returns all users
func (s *FileStore) ListUsers() ([]*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

// CreateUser creates a new user
func (s *FileStore) CreateUser(username, password string, role UserRole) (*User, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Check if username already exists
	for _, user := range s.users {
		if user.Username == username {
			return nil, ErrUserAlreadyExists
		}
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	user := &User{
		ID:       s.nextID,
		Username: username,
		Password: string(hashedPassword),
		Role:     role,
		Active:   true,
	}

	s.users[user.ID] = user
	s.nextID++

	if err := s.save(); err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("Created user: %s (ID: %d, Role: %s)", username, user.ID, role))
	return user, nil
}

// UpdateUser updates a user
func (s *FileStore) UpdateUser(id int, updates map[string]interface{}) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	user, exists := s.users[id]
	if !exists {
		return ErrUserNotFound
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "username":
			// Check if new username is already taken
			if username, ok := value.(string); ok {
				for _, u := range s.users {
					if u.ID != id && u.Username == username {
						return ErrUserAlreadyExists
					}
				}
				user.Username = username
			}
		case "password":
			if password, ok := value.(string); ok {
				hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
				if err != nil {
					return fmt.Errorf("failed to hash password: %w", err)
				}
				user.Password = string(hashedPassword)
			}
		case "role":
			if role, ok := value.(UserRole); ok {
				user.Role = role
			} else if roleStr, ok := value.(string); ok {
				user.Role = UserRole(roleStr)
			}
		case "active":
			if active, ok := value.(bool); ok {
				user.Active = active
			}
		case "api_key":
			if apiKey, ok := value.(string); ok {
				user.APIKey = apiKey
			}
		}
	}

	if err := s.save(); err != nil {
		return err
	}

	return nil
}

// DeleteUser deletes a user
func (s *FileStore) DeleteUser(id int) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if _, exists := s.users[id]; !exists {
		return ErrUserNotFound
	}

	// Don't allow deleting the root user (ID 1)
	if id == 1 {
		return errors.New("cannot delete root user")
	}

	delete(s.users, id)
	return s.save()
}

// Authenticate verifies username and password
func (s *FileStore) Authenticate(username, password string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Find user by username
	var user *User
	for _, u := range s.users {
		if u.Username == username {
			user = u
			break
		}
	}

	if user == nil {
		return nil, ErrInvalidCredentials
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Check if user is active
	if !user.Active {
		return nil, errors.New("user account is inactive")
	}

	return user, nil
}

// ValidateAPIKey validates an API key
func (s *FileStore) ValidateAPIKey(apiKey string) (*User, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Find user by API key
	for _, user := range s.users {
		if user.APIKey == apiKey && user.Active {
			return user, nil
		}
	}

	return nil, ErrInvalidCredentials
}

// HasPermission checks if a user has a specific permission
func HasPermission(user *User, permission Permission) bool {
	permissions, ok := RolePermissions[user.Role]
	if !ok {
		return false
	}

	for _, p := range permissions {
		if p == permission {
			return true
		}
	}

	return false
}
