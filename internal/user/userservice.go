package user

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/OG-Open-Source/PanelBase/internal/models" // Assuming models are here
)

const usersFilePath = "configs/users.json"

// fileMutex protects users.json from concurrent read/write operations.
// TODO: Replace with a more robust data storage solution (e.g., database) for production.
var fileMutex sync.RWMutex

// loadUsersConfig reads and parses the users.json file (with read lock).
// Renamed from loadUsersData for clarity.
func loadUsersConfig() (*models.UsersConfig, error) {
	fileMutex.RLock()
	defer fileMutex.RUnlock()

	data, err := os.ReadFile(usersFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Handle case where file doesn't exist - perhaps return empty config?
			return &models.UsersConfig{Users: make(map[string]models.User)}, nil // Return empty but valid config
		}
		return nil, fmt.Errorf("failed to read users file '%s': %w", usersFilePath, err)
	}

	// If file is empty, return empty config
	if len(data) == 0 {
		return &models.UsersConfig{Users: make(map[string]models.User)}, nil
	}

	var config models.UsersConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal users data from '%s': %w", usersFilePath, err)
	}

	// Initialize maps if they are nil after unmarshalling
	if config.Users == nil {
		config.Users = make(map[string]models.User)
	}

	return &config, nil
}

// saveUsersConfig writes the users config back to users.json (with write lock).
// Renamed from saveUsersData for clarity.
func saveUsersConfig(config *models.UsersConfig) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	jsonData, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users config: %w", err)
	}
	if err := os.WriteFile(usersFilePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write users file '%s': %w", usersFilePath, err)
	}
	return nil
}

// GetUserByUsername retrieves a user by their username.
// Returns the user and true if found, otherwise false.
func GetUserByUsername(username string) (models.User, bool, error) {
	config, err := loadUsersConfig()
	if err != nil {
		return models.User{}, false, fmt.Errorf("failed to load user config: %w", err)
	}

	user, exists := config.Users[username]
	return user, exists, nil
}

// GetUserByID retrieves a user by their unique ID (e.g., usr_...).
// Returns the user and true if found, otherwise false.
func GetUserByID(userID string) (models.User, bool, error) {
	config, err := loadUsersConfig()
	if err != nil {
		return models.User{}, false, fmt.Errorf("failed to load user config: %w", err)
	}

	// Iterate through the users map to find the user by ID
	for _, user := range config.Users {
		if user.ID == userID {
			return user, true, nil // User found
		}
	}

	return models.User{}, false, nil // User not found
}

// UpdateUser updates the entire user object in the configuration.
// It reads the current config, updates the specific user, and saves it back.
// TODO: Consider more granular update functions (e.g., UpdateUserPassword, UpdateUserAPI) for safety.
func UpdateUser(user models.User) error {
	config, err := loadUsersConfig()
	if err != nil {
		return fmt.Errorf("failed to load user config for update: %w", err)
	}

	// Check if user exists before updating - use username from the passed user struct
	if _, exists := config.Users[user.Username]; !exists {
		return fmt.Errorf("user '%s' not found for update", user.Username)
	}

	config.Users[user.Username] = user // Update the user in the map

	if err := saveUsersConfig(config); err != nil {
		return fmt.Errorf("failed to save user config after update: %w", err)
	}
	return nil
}

// GetAllUsers retrieves all users.
// TODO: Add methods for creating and deleting users.
func GetAllUsers() (map[string]models.User, error) {
	config, err := loadUsersConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to load user config: %w", err)
	}
	return config.Users, nil
}

// SaveUsersConfigForBootstrap provides a way for the bootstrap process to save the initial config.
// It directly calls the internal save function.
// NOTE: This bypasses the typical UpdateUser flow, use with caution and only during init.
func SaveUsersConfigForBootstrap(config *models.UsersConfig) error {
	// Directly use the internal save function with locking
	if err := saveUsersConfig(config); err != nil {
		return fmt.Errorf("bootstrap failed to save initial users config: %w", err)
	}
	return nil
}
