package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	logger "github.com/OG-Open-Source/PanelBase/internal/logging" // Import logging
	"github.com/OG-Open-Source/PanelBase/internal/models"
)

const usersFileName = "users.json"
const configsDir = "./configs"

var (
	usersConfig *models.UsersConfig
	mu          sync.RWMutex
)

// LoadUsersConfig loads the user configuration from the JSON file.
// Must be called once during application startup.
func LoadUsersConfig() error {
	mu.Lock()
	defer mu.Unlock()

	// Prevent re-loading if already loaded
	if usersConfig != nil {
		return nil
	}

	filePath := filepath.Join(configsDir, usersFileName)
	logger.Printf("USER_SVC", "LOAD", "Reading users config from: %s", filePath)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		logger.Printf("USER_SVC", "LOAD", "File '%s' not found. Will be created by bootstrap if needed.", filePath)
		// File doesn't exist, return empty config and let bootstrap handle creation
		usersConfig = &models.UsersConfig{Users: make(map[string]models.User)}
		return nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		logger.ErrorPrintf("USER_SVC", "LOAD", "Failed to read file '%s': %v", filePath, err)
		return fmt.Errorf("failed to read users config file '%s': %w", filePath, err)
	}

	// If the file is empty, initialize with an empty map
	if len(data) == 0 {
		usersConfig = &models.UsersConfig{Users: make(map[string]models.User)}
		logger.Printf("USER_SVC", "LOAD", "File '%s' was empty. Initialized empty map.", filePath)
		return nil
	}

	if err := json.Unmarshal(data, &usersConfig); err != nil {
		logger.ErrorPrintf("USER_SVC", "LOAD", "Failed to unmarshal '%s': %v", filePath, err)
		return fmt.Errorf("failed to unmarshal users config '%s': %w", filePath, err)
	}

	// Ensure Users map is initialized if file contained `null` or was empty object
	if usersConfig.Users == nil {
		usersConfig.Users = make(map[string]models.User)
	}

	logger.Printf("USER_SVC", "LOAD", "Loaded config with %d users.", len(usersConfig.Users))
	return nil
}

// saveUsersConfig is an internal helper to save the current state.
// Assumes caller holds the necessary lock (usually write lock).
func saveUsersConfig() error {
	if usersConfig == nil {
		return fmt.Errorf("internal error: users config is nil during save attempt")
	}

	filePath := filepath.Join(configsDir, usersFileName)
	data, err := json.MarshalIndent(usersConfig, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users config: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write users file '%s': %w", filePath, err)
	}
	return nil
}

// SaveUsersConfigForBootstrap allows bootstrap to save the initial config.
// Acquires the write lock.
func SaveUsersConfigForBootstrap(config *models.UsersConfig) error {
	mu.Lock()
	defer mu.Unlock()
	usersConfig = config     // Assign the bootstrap config
	return saveUsersConfig() // Call internal save
}

// GetUserByID retrieves a user by their ID (using UserID as map key).
func GetUserByID(id string) (models.User, bool, error) {
	mu.RLock()
	defer mu.RUnlock()

	if usersConfig == nil {
		return models.User{}, false, fmt.Errorf("users config not loaded")
	}

	user, exists := usersConfig.Users[id]
	return user, exists, nil
}

// GetUserByUsername retrieves a user by their username (iterates map values).
func GetUserByUsername(username string) (models.User, bool, error) {
	mu.RLock()
	defer mu.RUnlock()

	if usersConfig == nil {
		return models.User{}, false, fmt.Errorf("users config not loaded")
	}

	for _, user := range usersConfig.Users {
		if user.Username == username {
			return user, true, nil
		}
	}

	return models.User{}, false, nil // Not found
}

// AddUser adds a new user (using UserID as key).
func AddUser(newUser models.User) error {
	mu.Lock()
	defer mu.Unlock()

	if usersConfig == nil {
		return fmt.Errorf("users config not loaded")
	}
	if usersConfig.Users == nil {
		usersConfig.Users = make(map[string]models.User)
	}

	if _, exists := usersConfig.Users[newUser.ID]; exists {
		return fmt.Errorf("user with ID '%s' already exists", newUser.ID)
	}

	for _, existingUser := range usersConfig.Users {
		if existingUser.Username == newUser.Username {
			return fmt.Errorf("username '%s' already exists", newUser.Username)
		}
	}

	usersConfig.Users[newUser.ID] = newUser

	// filePath := filepath.Join(configsDir, usersFileName)
	// Attempt to save the updated config
	if err := saveUsersConfig(); err != nil {
		// Rollback the in-memory addition if save fails
		delete(usersConfig.Users, newUser.ID)
		logger.ErrorPrintf("USER_SVC", "ADD", "Failed to save users config after adding user %s: %v", newUser.ID, err)
		return fmt.Errorf("user added in memory but failed to save config: %w", err)
	}

	return nil
}

// UpdateUser updates an existing user's data (using UserID as key).
func UpdateUser(updatedUser models.User) error {
	mu.Lock()
	defer mu.Unlock()

	if usersConfig == nil {
		return fmt.Errorf("users config not loaded")
	}
	if usersConfig.Users == nil {
		return fmt.Errorf("users map is nil, cannot update")
	}

	originalUser, exists := usersConfig.Users[updatedUser.ID]
	if !exists {
		return fmt.Errorf("user with ID '%s' not found for update", updatedUser.ID)
	}

	for userID, existingUser := range usersConfig.Users {
		if userID != updatedUser.ID && existingUser.Username == updatedUser.Username {
			return fmt.Errorf("cannot update user: username '%s' is already taken by another user", updatedUser.Username)
		}
	}

	usersConfig.Users[updatedUser.ID] = updatedUser

	// filePath := filepath.Join(configsDir, usersFileName)
	// Attempt to save the updated config
	if err := saveUsersConfig(); err != nil {
		// Rollback the in-memory update if save fails
		usersConfig.Users[updatedUser.ID] = originalUser // Restore original
		logger.ErrorPrintf("USER_SVC", "UPDATE", "Failed to save users config after updating user %s: %v", updatedUser.ID, err)
		return fmt.Errorf("user updated in memory but failed to save config: %w", err)
	}

	return nil
}

// DeleteUser removes a user by ID (using UserID as key).
func DeleteUser(id string) error {
	mu.Lock()
	defer mu.Unlock()

	if usersConfig == nil {
		return fmt.Errorf("users config not loaded")
	}
	if usersConfig.Users == nil {
		return fmt.Errorf("users map is nil, cannot delete")
	}

	userToDelete, exists := usersConfig.Users[id]
	if !exists {
		return fmt.Errorf("user with ID '%s' not found for deletion", id)
	}

	delete(usersConfig.Users, id)

	// filePath := filepath.Join(configsDir, usersFileName)
	// Attempt to save the updated config
	if err := saveUsersConfig(); err != nil {
		// Rollback the in-memory deletion if save fails
		usersConfig.Users[id] = userToDelete // Put it back
		logger.ErrorPrintf("USER_SVC", "DELETE", "Failed to save users config after deleting user %s: %v", id, err)
		return fmt.Errorf("user deleted in memory but failed to save config: %w", err)
	}
	return nil
}

// UsernameExists checks if a username already exists (iterates map values).
func UsernameExists(username string) (bool, error) {
	mu.RLock()
	defer mu.RUnlock()

	if usersConfig == nil {
		return false, fmt.Errorf("users config not loaded")
	}

	for _, user := range usersConfig.Users {
		if user.Username == username {
			return true, nil
		}
	}

	return false, nil
}

// GetGlobalJWTSecret retrieves the global JWT secret.
func GetGlobalJWTSecret() (string, error) {
	mu.RLock()
	defer mu.RUnlock()
	if usersConfig == nil {
		return "", fmt.Errorf("users config not loaded")
	}
	return usersConfig.JwtSecret, nil
}

// GetAllUsers retrieves all users (primarily for debug or admin listing).
func GetAllUsers() (map[string]models.User, error) {
	mu.RLock()
	defer mu.RUnlock()

	if usersConfig == nil {
		return nil, fmt.Errorf("users config not loaded")
	}
	// Return a copy to prevent modification of the internal map
	usersCopy := make(map[string]models.User, len(usersConfig.Users))
	for k, v := range usersConfig.Users {
		usersCopy[k] = v
	}
	return usersCopy, nil
}
