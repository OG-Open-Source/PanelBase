package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
)

// Constants moved to pkg/utils
// const (
// 	userIDPrefix       = "usr_"
// 	userIDRandomLength = 12
// )

// usersFileFormat defines the structure of the entire users JSON file.
type usersFileFormat struct {
	Users map[string]*models.User `json:"users"` // Key is the user ID string (e.g., "usr_...")
}

// JSONUserStore implements the UserStore interface using a JSON file.
// IMPORTANT: This implementation is simple and NOT suitable for production
// due to potential race conditions and performance issues with large files.
// It lacks proper file locking beyond a simple mutex.
type JSONUserStore struct {
	filePath      string
	mu            sync.RWMutex
	users         map[string]*models.User // Key is user ID string
	usernameIndex map[string]string       // Username to user ID string index
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
		// If file is empty or invalid JSON, load() handles initialization and saving.
		// We only return error for other issues like permission problems.
		if !errors.Is(err, ErrStoreEmptyOrInvalid) {
			return nil, fmt.Errorf("failed to load initial user data from %s: %w", filePath, err)
		}
		// If ErrStoreEmptyOrInvalid, the store was initialized, so continue.
	}
	return store, nil
}

// Custom error for signaling that the store was initialized empty
var ErrStoreEmptyOrInvalid = errors.New("user store file was empty or invalid, initialized empty store")

// load reads the user data from the JSON file.
func (s *JSONUserStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	absPath, _ := filepath.Abs(s.filePath)
	data, err := os.ReadFile(absPath)
	if err != nil {
		if os.IsNotExist(err) { // Bootstrap should create it, but handle defensively if called directly
			log.Printf("WARN: User store file (%s) does not exist. Initializing empty store.", s.filePath)
			s.users = make(map[string]*models.User)
			s.usernameIndex = make(map[string]string)
			saveErr := s.saveInternal() // Use internal save without lock
			if saveErr != nil {
				return fmt.Errorf("failed to save initial empty user store: %w", saveErr)
			}
			return ErrStoreEmptyOrInvalid // Signal that it was initialized
		}
		return err // Return other read errors (permissions etc)
	}

	if len(data) == 0 {
		log.Printf("User store file (%s) is empty. Initializing empty store.", s.filePath)
		s.users = make(map[string]*models.User)
		s.usernameIndex = make(map[string]string)
		err = s.saveInternal() // Use internal save without lock
		if err != nil {
			return fmt.Errorf("failed to save initial empty user store: %w", err)
		}
		return ErrStoreEmptyOrInvalid // Signal that it was initialized
	}

	// Unmarshal the entire file structure
	var fileData usersFileFormat
	if err := json.Unmarshal(data, &fileData); err != nil {
		log.Printf("WARN: User store file (%s) contains invalid JSON: %v. Initializing empty store.", s.filePath, err)
		s.users = make(map[string]*models.User)
		s.usernameIndex = make(map[string]string)
		saveErr := s.saveInternal() // Use internal save without lock
		if saveErr != nil {
			return fmt.Errorf("failed to save initial empty user store: %w", saveErr)
		}
		return ErrStoreEmptyOrInvalid // Signal that it was initialized
	}

	// Initialize maps if fileData.Users is nil (e.g., file was `{}`)
	if fileData.Users == nil {
		fileData.Users = make(map[string]*models.User)
	}

	// Rebuild in-memory maps
	s.users = fileData.Users
	s.usernameIndex = make(map[string]string, len(s.users))
	for idStr, u := range s.users {
		// Sanity check for user ID consistency and prefix
		if u.ID != idStr || !strings.HasPrefix(idStr, utils.UserIDPrefix) {
			log.Printf("WARN: User ID mismatch or invalid format in user store file for key '%s'. User object ID: '%s'. Skipping this user.", idStr, u.ID)
			delete(s.users, idStr) // Remove inconsistent entry
			continue
		}
		s.usernameIndex[u.Username] = idStr
	}
	return nil
}

// save writes the current user data back to the JSON file.
// Assumes the caller holds the write lock (s.mu.Lock).
func (s *JSONUserStore) saveInternal() error {
	// Create a deep copy of the users map to avoid modifying the in-memory store directly
	usersToSave := make(map[string]*models.User, len(s.users))
	for id, user := range s.users {
		userCopy := *user                                             // Create a shallow copy of the user struct
		userCopy.CreatedAt = userCopy.CreatedAt.Truncate(time.Second) // Truncate timestamp to seconds
		usersToSave[id] = &userCopy
	}

	// Create the structure expected in the file using the modified copy
	fileData := usersFileFormat{
		Users: usersToSave,
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

	// Generate ID with prefix if not provided
	if user.ID == "" {
		var err error
		user.ID, err = utils.GeneratePrefixedID(utils.UserIDPrefix, utils.UserIDRandomLength)
		if err != nil {
			return fmt.Errorf("failed to generate user ID: %w", err)
		}
		// Extremely unlikely, but check for collision just in case
		for _, exists := s.users[user.ID]; exists; _, exists = s.users[user.ID] {
			log.Printf("WARN: User ID collision detected for %s. Regenerating...", user.ID)
			user.ID, err = utils.GeneratePrefixedID(utils.UserIDPrefix, utils.UserIDRandomLength)
			if err != nil {
				return fmt.Errorf("failed to regenerate user ID after collision: %w", err)
			}
		}
	}

	if user.CreatedAt.IsZero() {
		user.CreatedAt = time.Now().UTC() // Set creation time if not provided
	}

	s.users[user.ID] = user
	s.usernameIndex[user.Username] = user.ID

	if err := s.saveInternal(); err != nil {
		delete(s.users, user.ID)
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

	userCopy := *user // Return a copy to prevent modification of internal map value
	return &userCopy, nil
}

// GetUserByID retrieves a user by their ID string.
func (s *JSONUserStore) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, ErrUserNotFound
	}
	userCopy := *user // Return a copy
	return &userCopy, nil
}

// GetAllUsers retrieves all users.
func (s *JSONUserStore) GetAllUsers(ctx context.Context) ([]models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	userList := make([]models.User, 0, len(s.users))
	for _, user := range s.users {
		userList = append(userList, *user) // Append copies
	}
	// Optional: Sort users, e.g., by CreatedAt or Username
	sort.Slice(userList, func(i, j int) bool {
		return userList[i].CreatedAt.Before(userList[j].CreatedAt)
	})

	return userList, nil
}

// UpdateUser updates user data (excluding password).
func (s *JSONUserStore) UpdateUser(ctx context.Context, user *models.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user.ID == "" {
		return errors.New("cannot update user with empty ID")
	}

	originalUser, exists := s.users[user.ID]
	if !exists {
		return ErrUserNotFound
	}

	// Check for username conflict if username is being changed
	if user.Username != originalUser.Username {
		if _, existsInIndex := s.usernameIndex[user.Username]; existsInIndex {
			// Check if the conflicting index entry points to the same user ID (shouldn't happen ideally)
			// If it points to a *different* user ID, then it's a real conflict.
			conflictingID := s.usernameIndex[user.Username]
			if conflictingID != user.ID {
				return ErrUserExists
			}
		}
		// Update index
		delete(s.usernameIndex, originalUser.Username)
		s.usernameIndex[user.Username] = user.ID
	}

	// Preserve original password hash and creation date - crucial!
	updatedUser := *user // Create a copy to modify
	updatedUser.PasswordHash = originalUser.PasswordHash
	updatedUser.CreatedAt = originalUser.CreatedAt
	s.users[user.ID] = &updatedUser // Store the updated copy

	if err := s.saveInternal(); err != nil {
		// Attempt simple rollback in memory (complex rollback is hard)
		log.Printf("ERROR: Failed to save user update for ID %s: %v. Attempting rollback...", user.ID, err)
		s.users[user.ID] = originalUser
		if user.Username != originalUser.Username {
			delete(s.usernameIndex, user.Username)           // Remove potentially added new username
			s.usernameIndex[originalUser.Username] = user.ID // Restore old username mapping
		}
		return err // Return the save error
	}
	return nil
}

// DeleteUser removes a user by ID string.
func (s *JSONUserStore) DeleteUser(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return ErrUserNotFound // Return specific error
	}

	username := user.Username // Store before deleting user

	delete(s.users, id)
	delete(s.usernameIndex, username)

	if err := s.saveInternal(); err != nil {
		// Attempt rollback
		s.users[id] = user
		s.usernameIndex[username] = id
		log.Printf("ERROR: Failed to save user deletion for ID %s: %v. Rolled back.", id, err)
		return err
	}
	return nil
}

// --- API Token Methods ---

// AddApiToken adds a new API token to a user.
func (s *JSONUserStore) AddApiToken(ctx context.Context, userID string, token models.ApiToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	// Check for duplicate Token ID (JTI) for this user - should be unique
	for _, existingToken := range user.ApiTokens {
		if existingToken.ID == token.ID {
			return fmt.Errorf("API token ID (JTI) %s already exists for user %s", token.ID, userID)
		}
	}

	// Initialize ApiTokens slice if nil
	if user.ApiTokens == nil {
		user.ApiTokens = make([]models.ApiToken, 0)
	}

	user.ApiTokens = append(user.ApiTokens, token)

	if err := s.saveInternal(); err != nil {
		// Attempt rollback in memory
		user.ApiTokens = user.ApiTokens[:len(user.ApiTokens)-1]
		log.Printf("ERROR: Failed to save adding API token %s for user %s: %v. Rolled back.", token.ID, userID, err)
		return err
	}
	return nil
}

// GetUserApiTokens retrieves all API tokens for a specific user.
func (s *JSONUserStore) GetUserApiTokens(ctx context.Context, userID string) ([]models.ApiToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	// Return a copy of the slice to prevent external modification
	if user.ApiTokens == nil {
		return []models.ApiToken{}, nil // Return empty slice, not nil
	}

	tokensCopy := make([]models.ApiToken, len(user.ApiTokens))
	copy(tokensCopy, user.ApiTokens)
	return tokensCopy, nil
}

// DeleteApiToken removes a specific API token from a user.
func (s *JSONUserStore) DeleteApiToken(ctx context.Context, userID string, tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[userID]
	if !exists {
		return ErrUserNotFound
	}

	if user.ApiTokens == nil {
		return fmt.Errorf("API token with ID %s not found for user %s", tokenID, userID) // Or a specific TokenNotFound error
	}

	originalTokens := make([]models.ApiToken, len(user.ApiTokens))
	copy(originalTokens, user.ApiTokens)

	found := false
	newTokenList := make([]models.ApiToken, 0, len(user.ApiTokens))
	for _, token := range user.ApiTokens {
		if token.ID == tokenID {
			found = true
			// Skip adding this token to the new list
		} else {
			newTokenList = append(newTokenList, token)
		}
	}

	if !found {
		return fmt.Errorf("API token with ID %s not found for user %s", tokenID, userID) // Or a specific TokenNotFound error
	}

	user.ApiTokens = newTokenList

	if err := s.saveInternal(); err != nil {
		// Attempt rollback
		user.ApiTokens = originalTokens
		log.Printf("ERROR: Failed to save deletion of API token %s for user %s: %v. Rolled back.", tokenID, userID, err)
		return err
	}
	return nil
}

// GetApiTokenByID retrieves a specific API token by its ID for a given user.
// Useful for JTI validation if not using a separate DB.
func (s *JSONUserStore) GetApiTokenByID(ctx context.Context, userID string, tokenID string) (*models.ApiToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[userID]
	if !exists {
		return nil, ErrUserNotFound
	}

	if user.ApiTokens == nil {
		return nil, fmt.Errorf("API token %s not found", tokenID) // Specific error?
	}

	for _, token := range user.ApiTokens {
		if token.ID == tokenID {
			tokenCopy := token // Return a copy
			return &tokenCopy, nil
		}
	}

	return nil, fmt.Errorf("API token %s not found", tokenID) // Specific error?
}
