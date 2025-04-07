package bootstrap

import (
	"bytes"
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big" // Use crypto/rand for secure random numbers
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/internal/user" // Import the user service
	"golang.org/x/crypto/bcrypt"
)

const configsDir = "configs"
const logsDir = "logs"

// Configs defines the default content for all configuration files.
type Configs struct {
	Themes     map[string]interface{}
	Commands   map[string]interface{}
	Plugins    map[string]interface{}
	Users      map[string]interface{}
	Config     map[string]interface{}
	UISettings map[string]interface{}
}

// NewConfigs creates the default configuration structure.
func NewConfigs() *Configs {
	// Default configurations remain empty as per previous request
	return &Configs{
		Themes:   map[string]interface{}{"themes": map[string]interface{}{}, "current_theme": ""},
		Commands: map[string]interface{}{"commands": map[string]interface{}{}},
		Plugins:  map[string]interface{}{"plugins": map[string]interface{}{}},
		Users:    map[string]interface{}{"jwt_secret": "", "users": map[string]interface{}{}}, // JWT secret and users filled dynamically
		Config: map[string]interface{}{ // For config.toml
			"server": map[string]interface{}{ // Will be set dynamically during Bootstrap
				"host": "0.0.0.0",
				"port": 0,
				"mode": "release",
			},
			"features": map[string]interface{}{ // Will be set dynamically during Bootstrap
				"commands": false,
				"plugins":  true,
			},
			"auth": map[string]interface{}{ // Will be set dynamically during Bootstrap
				"jwt_expiration": 24,
				"cookie_name":    "panelbase_jwt",
			},
		},
		UISettings: map[string]interface{}{ // Initialize UISettings with defaults
			"title":       "PanelBase",
			"logo_url":    "",
			"favicon_url": "",
			"custom_css":  "",
			"custom_js":   "",
		},
	}
}

// Bootstrap checks and creates necessary configuration files and directories.
// It now returns a list of items created for summary logging.
func Bootstrap() ([]string, error) {
	createdItems := []string{}
	var itemCreated string
	var err error

	// Ensure the configs directory exists
	itemCreated, err = ensureDirExists(configsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure configs directory: %w", err)
	}
	if itemCreated != "" {
		createdItems = append(createdItems, itemCreated)
	}

	// Ensure the logs directory exists
	itemCreated, err = ensureDirExists(logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure logs directory: %w", err)
	}
	if itemCreated != "" {
		createdItems = append(createdItems, itemCreated)
	}

	// Create default config structure (used if files need creation)
	configs := NewConfigs()

	// Check/Create users.json
	itemCreated, err = ensureUsersFile()
	if err != nil {
		return createdItems, err // Return already created items and the error
	}
	if itemCreated != "" {
		createdItems = append(createdItems, itemCreated)
	}

	// Check/Create config.toml
	itemCreated, err = ensureConfigFile(configs)
	if err != nil {
		return createdItems, err
	}
	if itemCreated != "" {
		createdItems = append(createdItems, itemCreated)
	}

	// Check/Create simple JSON files (themes, commands, plugins, ui_settings)
	simpleJsonConfigs := map[string]interface{}{
		filepath.Join(configsDir, "themes.json"):      configs.Themes,
		filepath.Join(configsDir, "commands.json"):    configs.Commands,
		filepath.Join(configsDir, "plugins.json"):     configs.Plugins,
		filepath.Join(configsDir, "ui_settings.json"): configs.UISettings, // Add ui_settings here
	}
	for path, data := range simpleJsonConfigs {
		itemCreated, err = ensureSimpleJsonFile(path, data)
		if err != nil {
			return createdItems, err // Return error if checking/writing fails
		}
		if itemCreated != "" {
			createdItems = append(createdItems, itemCreated)
		}
	}

	return createdItems, nil
}

// ensureSimpleJsonFile checks and creates a simple JSON file if it doesn't exist.
// Returns the path if created, or empty string, and an error.
func ensureSimpleJsonFile(path string, defaultData interface{}) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// log.Printf("File '%s' not found, creating with defaults...", path) // Keep log for creation start
		if err := writeJsonFile(path, defaultData); err != nil {
			return "", err // Return error if writing fails
		}
		return fmt.Sprintf("File '%s'", path), nil // Return created path
	} else if err != nil {
		return "", fmt.Errorf("failed to check config file '%s': %w", path, err)
	}
	return "", nil // File existed
}

// ensureUsersFile handles checking/creating users.json
// Returns the path if created, or empty string, and an error.
func ensureUsersFile() (string, error) {
	usersPath := filepath.Join(configsDir, "users.json")
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		// log.Printf("File '%s' not found, creating with defaults...", usersPath)
		if err := initializeUsersFile(); err != nil {
			return "", fmt.Errorf("failed to initialize users file: %w", err)
		}
		return fmt.Sprintf("File '%s'", usersPath), nil // Return created path
	} else if err != nil {
		return "", fmt.Errorf("failed to check users file '%s': %w", usersPath, err)
	}
	return "", nil // File existed
}

// ensureConfigFile handles checking/creating config.toml
// Returns the path if created, or empty string, and an error.
func ensureConfigFile(configs *Configs) (string, error) {
	configTomlPath := filepath.Join(configsDir, "config.toml")
	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
		// log.Printf("File '%s' not found, creating with defaults...", configTomlPath) // Keep log for creation start
		port, err := findAvailablePort(1024, 49151)
		if err != nil {
			return "", fmt.Errorf("failed to find available port for new config file: %w", err)
		}

		defaultConfigData := configs.Config
		if serverConf, ok := defaultConfigData["server"].(map[string]interface{}); ok {
			serverConf["port"] = port
		} else {
			return "", fmt.Errorf("internal error: invalid structure for default server config")
		}

		if err := writeTomlFile(configTomlPath, defaultConfigData); err != nil {
			return "", err
		}
		return fmt.Sprintf("File '%s'", configTomlPath), nil // Return created path
	} else if err != nil {
		return "", fmt.Errorf("failed to check config file '%s': %w", configTomlPath, err)
	}
	return "", nil // File existed
}

// ensureDirExists checks if a directory exists, and creates it if not.
// Returns the path if created, or empty string, and an error.
func ensureDirExists(dirPath string) (string, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", dirPath, err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// log.Printf("Directory '%s' not found, creating...", absPath) // Keep log for creation start
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory '%s': %w", absPath, err)
		}
		return fmt.Sprintf("Directory '%s'", absPath), nil // Return created path
	} else if err != nil {
		return "", fmt.Errorf("failed to check directory '%s': %w", absPath, err)
	}
	return "", nil // Directory existed
}

// initializeUsersFile creates the initial users.json file with a default admin user.
// This now uses the user service to save the config.
func initializeUsersFile() error {
	// Generate global JWT secret (can be used as fallback or default)
	globalJwtSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("failed to generate global JWT secret: %w", err)
	}

	// Prepare user details (can be extended to multiple default users)
	defaultUserDetails := []struct {
		Username string
		Password string
		Name     string
		Email    string
		Scopes   models.UserPermissions
	}{
		{
			Username: "admin",
			Password: "admin",
			Name:     "Administrator",
			Email:    "admin@example.com",
			Scopes: models.UserPermissions{
				"account":  {"read", "update", "delete"},                             // Manage own account (read, update via PATCH, delete)
				"users":    {"read:list", "read:item", "create", "update", "delete"}, // Manage other users
				"api":      {"read:list", "read:item", "create", "update", "delete", "read:list:all", "read:item:all", "create:all", "update:all", "delete:all"},
				"settings": {"read", "update"}, // Manage global settings
				"commands": {"read:list", "read:item", "install", "execute", "update", "delete"},
				"plugins":  {"read:list", "read:item", "install", "update", "delete"},
				"themes":   {"read:list", "read:item", "install", "update", "delete"},
			},
		},
	}

	// Initialize the UsersConfig structure
	usersConfig := &models.UsersConfig{
		JwtSecret: globalJwtSecret,
		Users:     make(map[string]models.User),
	}

	// Create user entries using the models.User struct
	for _, u := range defaultUserDetails {
		userID, err := generateUserID() // Generate ID for each user
		if err != nil {
			log.Printf("Warning: Failed to generate unique user ID for %s: %v. Skipping user.", u.Username, err)
			continue
		}
		// userID = "usr_" + userID // Prefix is now handled by generateUserID

		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			log.Printf("Warning: Failed to hash password for %s: %v. Skipping user.", u.Username, err)
			continue
		}

		userJwtSecret, err := generateRandomString(32) // Generate user-specific secret
		if err != nil {
			log.Printf("Warning: Failed to generate JWT secret for %s: %v. Using fallback.", u.Username, err)
			userJwtSecret = "fallback_secret_" + userID // Less secure fallback
		}

		userData := models.User{
			ID:        userID,
			Username:  u.Username,
			Password:  string(hashedPassword),
			Name:      u.Name,
			Email:     u.Email,
			CreatedAt: models.RFC3339Time(time.Now().UTC()),
			Active:    true,
			Scopes:    u.Scopes,
			API: models.UserAPISettings{
				JwtSecret: userJwtSecret,
			},
		}
		// usersConfig.Users[u.Username] = userData // Old way: Use username as key
		usersConfig.Users[userData.ID] = userData // New way: Use UserID as the key
	}

	// Save the initialized users config using the user service save function
	if err := user.SaveUsersConfigForBootstrap(usersConfig); err != nil {
		log.Printf("Error initializing users.json via service: %v", err)
		return err
	}

	return nil
}

// generateRandomString remains here for bootstrap purposes
// TODO: Consider moving to a shared utility package if used elsewhere.
func generateRandomString(length int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz_"
	ret := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := crand.Int(crand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		ret[i] = letters[num.Int64()]
	}
	return string(ret), nil
}

// writeJsonFile encodes data to JSON and writes it to the specified file path.
func writeJsonFile(filePath string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ") // Use two spaces for indentation
	if err != nil {
		return fmt.Errorf("failed to marshal data for %s: %w", filePath, err)
	}
	if err := os.WriteFile(filePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	return nil
}

// writeTomlFile encodes data to TOML and writes it to the specified file path.
func writeTomlFile(filePath string, data interface{}) error {
	buf := new(bytes.Buffer)
	if err := toml.NewEncoder(buf).Encode(data); err != nil {
		return fmt.Errorf("failed to encode TOML for %s: %w", filePath, err)
	}
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", filePath, err)
	}
	return nil
}

// findAvailablePort searches for an available TCP port within the specified range.
func findAvailablePort(start, end int) (int, error) {
	if start > end {
		return 0, fmt.Errorf("invalid port range: start (%d) > end (%d)", start, end)
	}
	// Try up to 100 times to find a random port
	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		// Generate a random port within the range (inclusive)
		port, err := crand.Int(crand.Reader, big.NewInt(int64(end-start+1)))
		if err != nil {
			return 0, fmt.Errorf("failed to generate random port number: %w", err)
		}
		testPort := start + int(port.Int64())

		// Check if the port is available
		address := fmt.Sprintf(":%d", testPort)
		listener, err := net.Listen("tcp", address)
		if err == nil {
			// Port is available, close the listener and return the port
			listener.Close()
			return testPort, nil
		}

		// Check if the error indicates the port is already in use
		if !isPortInUseError(err) {
			// If it's another error (e.g., permission denied), log it but continue searching
			// as another random port might work.
			log.Printf("Warning: Unexpected error checking port %d: %v", testPort, err)
		}
		// Port is in use or had an unexpected (but possibly transient) error, try another random port
	}

	return 0, fmt.Errorf("failed to find available port in range %d-%d after %d attempts", start, end, maxAttempts)
}

// isPortInUseError checks if the error indicates a port is already in use.
func isPortInUseError(err error) bool {
	if err == nil {
		return false
	}
	// Check for common error strings across different OS
	errStr := err.Error()
	return strings.Contains(errStr, "address already in use") ||
		strings.Contains(errStr, "bind: address already in use") ||
		// Add other potential OS-specific messages if needed
		strings.Contains(errStr, "Only one usage of each socket address") // Windows
}

// generateUserID creates a unique user identifier (e.g., usr_xxxxxxxxxxxxxxxx)
func generateUserID() (string, error) {
	bytesLength := 8 // 8 bytes = 16 hex chars
	b := make([]byte, bytesLength)
	_, err := crand.Read(b)
	if err != nil {
		return "", fmt.Errorf("failed to read random bytes for user ID: %w", err)
	}
	return "usr_" + hex.EncodeToString(b), nil
}
