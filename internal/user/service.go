package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"log" // Added for logging potential errors during save

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
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		log.Printf("File '%s' not found. Will be created by bootstrap if needed.", filePath)
		usersConfig = &models.UsersConfig{
			JwtSecret: "",
			Users:     make(map[string]models.User),
		}
		return nil
	}

	// Read file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read users config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &usersConfig); err != nil {
		return fmt.Errorf("failed to parse users config file: %w", err)
	}

	if usersConfig.Users == nil {
		usersConfig.Users = make(map[string]models.User)
	}

	log.Printf("Loaded users config with %d users.", len(usersConfig.Users))
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

// AddUser adds a new user to the configuration
func AddUser(user models.User) error {
	// Check if user already exists
	if _, exists := usersConfig.Users[user.ID]; exists {
		return fmt.Errorf("user with ID %s already exists", user.ID)
	}

	// Add user to config
	usersConfig.Users[user.ID] = user

	// Save config
	if err := saveUsersConfig(); err != nil {
		// Rollback in memory
		delete(usersConfig.Users, user.ID)
		log.Printf("%s ERROR: User added in memory but failed to save config. Rolled back addition. Err: %v", time.Now().UTC().Format(time.RFC3339), err)
		return fmt.Errorf("failed to save users config: %w", err)
	}

	return nil
}

// UpdateUser updates an existing user in the configuration
func UpdateUser(user models.User) error {
	// Check if user exists
	if _, exists := usersConfig.Users[user.ID]; !exists {
		return fmt.Errorf("user with ID %s does not exist", user.ID)
	}

	// Store old user data for rollback
	oldUser := usersConfig.Users[user.ID]

	// Update user in config
	usersConfig.Users[user.ID] = user

	// Save config
	if err := saveUsersConfig(); err != nil {
		// Rollback in memory
		usersConfig.Users[user.ID] = oldUser
		log.Printf("%s ERROR: User updated in memory but failed to save config. Rolled back update for user %s. Err: %v", time.Now().UTC().Format(time.RFC3339), user.ID, err)
		return fmt.Errorf("failed to save users config: %w", err)
	}

	return nil
}

// DeleteUser deletes a user from the configuration
func DeleteUser(id string) error {
	// Check if user exists
	if _, exists := usersConfig.Users[id]; !exists {
		return fmt.Errorf("user with ID %s does not exist", id)
	}

	// Store old user data for rollback
	oldUser := usersConfig.Users[id]

	// Delete user from config
	delete(usersConfig.Users, id)

	// Save config
	if err := saveUsersConfig(); err != nil {
		// Rollback in memory
		usersConfig.Users[id] = oldUser
		log.Printf("%s ERROR: User deleted in memory but failed to save config. Rolled back deletion for user %s. Err: %v", time.Now().UTC().Format(time.RFC3339), id, err)
		return fmt.Errorf("failed to save users config: %w", err)
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
