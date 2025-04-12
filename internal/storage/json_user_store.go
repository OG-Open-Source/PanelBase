package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/google/uuid"
)

// usersFileFormat defines the structure of the entire users JSON file.
type usersFileFormat struct {
	// Ignoring top-level jwt_secret from example, handled in config.toml
	Users map[string]*models.User `json:"users"` // Key is string representation of UUID
}

// JSONUserStore implements the UserStore interface using a JSON file.
// IMPORTANT: This implementation is simple and NOT suitable for production
// due to potential race conditions and performance issues with large files.
// It lacks proper file locking beyond a simple mutex.
type JSONUserStore struct {
	filePath      string
	mu            sync.RWMutex            // Read/Write mutex to protect access to the file/data
	users         map[string]*models.User // Use string UUID as key to match file format
	usernameIndex map[string]string       // Username to string UUID index
}

// NewJSONUserStore creates a new JSONUserStore and loads initial data.
func NewJSONUserStore(filePath string) (*JSONUserStore, error) {
	store := &JSONUserStore{
		filePath:      filePath,
		users:         make(map[string]*models.User),
		usernameIndex: make(map[string]string),
	}
	if err := store.load(); err != nil {
		// If file not found, it's now an error because bootstrap should create it.
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("user store file '%s' not found, initialization might have failed: %w", filePath, err)
		}
		return nil, fmt.Errorf("failed to load initial user data from %s: %w", filePath, err)
	}
	return store, nil
}

// load reads the user data from the JSON file.
func (s *JSONUserStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	absPath, _ := filepath.Abs(s.filePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		return err
	}

	if len(data) == 0 || string(data) == "[]\n" || string(data) == "[]" {
		// Handle empty file or file initialized as empty array `[]`
		log.Printf("User store file (%s) is empty or invalid, initializing empty store.", s.filePath)
		// Initialize with the expected structure: an object with a "users" map
		s.users = make(map[string]*models.User)
		s.usernameIndex = make(map[string]string)
		return s.save() // Save the initial empty object structure
	}

	// Unmarshal the entire file structure
	var fileData usersFileFormat
	if err := json.Unmarshal(data, &fileData); err != nil {
		return fmt.Errorf("failed to unmarshal users file data: %w", err)
	}

	// Initialize maps if fileData.Users is nil (e.g., file was `{}`)
	if fileData.Users == nil {
		fileData.Users = make(map[string]*models.User)
	}

	// Rebuild in-memory maps
	s.users = fileData.Users
	s.usernameIndex = make(map[string]string, len(s.users))
	for idStr, u := range s.users {
		// Ensure user ID matches the map key (optional sanity check)
		if u.ID.String() != idStr {
			log.Printf("WARN: User ID mismatch in user store file for key '%s'. Using user object ID '%s'.", idStr, u.ID.String())
			// Potentially correct the map key here if desired, but might indicate deeper issues.
		}
		s.usernameIndex[u.Username] = idStr
	}
	log.Printf("Loaded %d users from %s", len(s.users), s.filePath)
	return nil
}

// save writes the current user data back to the JSON file.
func (s *JSONUserStore) save() error {
	// Create the structure expected in the file
	fileData := usersFileFormat{
		Users: s.users,
	}

	data, err := json.MarshalIndent(fileData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal user data: %w", err)
	}

	absPath, _ := filepath.Abs(s.filePath)
	if err := os.WriteFile(absPath, data, 0664); err != nil {
		return fmt.Errorf("failed to write user data to %s: %w", s.filePath, err)
	}
	return nil
}

// CreateUser adds a new user.
func (s *JSONUserStore) CreateUser(ctx context.Context, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.usernameIndex[user.Username]; exists {
		return ErrUserExists
	}
	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}
	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC() // Set creation time if not provided
	}
	// Default Active to true if not explicitly set (or handle as needed)
	// user.Active = true

	idStr := user.ID.String()
	s.users[idStr] = user
	s.usernameIndex[user.Username] = idStr

	if err := s.save(); err != nil {
		delete(s.users, idStr)
		delete(s.usernameIndex, user.Username)
		return err
	}
	return nil
}

// GetUserByUsername retrieves a user by username.
func (s *JSONUserStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idStr, exists := s.usernameIndex[username]
	if !exists {
		return nil, ErrUserNotFound
	}
	user, userExists := s.users[idStr]
	if !userExists {
		log.Printf("ERROR: Username index points to non-existent user ID string: %s", idStr)
		return nil, ErrUserNotFound
	}

	userCopy := *user
	return &userCopy, nil
}

// GetUserByID retrieves a user by their ID.
func (s *JSONUserStore) GetUserByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	idStr := id.String()
	user, exists := s.users[idStr]
	if !exists {
		return nil, ErrUserNotFound
	}
	userCopy := *user
	return &userCopy, nil
}

// UpdateUser updates user data (excluding password).
func (s *JSONUserStore) UpdateUser(ctx context.Context, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idStr := user.ID.String()
	originalUser, exists := s.users[idStr]
	if !exists {
		return ErrUserNotFound
	}

	if user.Username != originalUser.Username {
		if _, exists := s.usernameIndex[user.Username]; exists {
			return ErrUserExists
		}
		delete(s.usernameIndex, originalUser.Username)
		s.usernameIndex[user.Username] = idStr
	}

	// Preserve original password hash and creation date
	user.PasswordHash = originalUser.PasswordHash
	user.CreatedAt = originalUser.CreatedAt
	s.users[idStr] = user

	if err := s.save(); err != nil {
		// Rollback is complex, log and return error
		log.Printf("ERROR: Failed to save user update for ID %s: %v", idStr, err)
		return err
	}
	return nil
}

// DeleteUser removes a user.
func (s *JSONUserStore) DeleteUser(ctx context.Context, id uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	idStr := id.String()
	user, exists := s.users[idStr]
	if !exists {
		return ErrUserNotFound
	}

	delete(s.users, idStr)
	delete(s.usernameIndex, user.Username)

	if err := s.save(); err != nil {
		// Attempt rollback
		s.users[idStr] = user
		s.usernameIndex[user.Username] = idStr
		log.Printf("ERROR: Failed to save user deletion for ID %s: %v", idStr, err)
		return err
	}
	return nil
}
