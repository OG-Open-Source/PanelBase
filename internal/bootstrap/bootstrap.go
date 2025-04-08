package bootstrap

import (
	crand "crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"math/big" // Use crypto/rand for secure random numbers
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/OG-Open-Source/PanelBase/internal/models" // Import the user service
	"golang.org/x/crypto/bcrypt"
)

func init() {
	// Initialize the random number generator
	rand.Seed(time.Now().UnixNano())
}

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
		createdItems = append(createdItems, fmt.Sprintf("Directory '%s'", configsDir))
	}

	// Ensure the logs directory exists
	itemCreated, err = ensureDirExists(logsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to ensure logs directory: %w", err)
	}
	if itemCreated != "" {
		createdItems = append(createdItems, fmt.Sprintf("Directory '%s'", logsDir))
	}

	// Create default config structure (used if files need creation)
	configs := NewConfigs()

	// Check/Create users.json
	itemCreated, err = ensureUsersFile()
	if err != nil {
		return createdItems, err
	}
	if itemCreated != "" {
		createdItems = append(createdItems, fmt.Sprintf("File '%s'", filepath.Join(configsDir, "users.json")))
	}

	// Check/Create config.toml
	itemCreated, err = ensureConfigFile(configs)
	if err != nil {
		return createdItems, err
	}
	if itemCreated != "" {
		createdItems = append(createdItems, fmt.Sprintf("File '%s'", filepath.Join(configsDir, "config.toml")))
	}

	// Check/Create simple JSON files (themes, commands, plugins, ui_settings)
	simpleJsonConfigs := map[string]interface{}{
		filepath.Join(configsDir, "themes.json"):      configs.Themes,
		filepath.Join(configsDir, "commands.json"):    configs.Commands,
		filepath.Join(configsDir, "plugins.json"):     configs.Plugins,
		filepath.Join(configsDir, "ui_settings.json"): configs.UISettings,
	}
	for path, data := range simpleJsonConfigs {
		itemCreated, err = ensureSimpleJsonFile(path, data)
		if err != nil {
			return createdItems, err
		}
		if itemCreated != "" {
			createdItems = append(createdItems, fmt.Sprintf("File '%s'", path))
		}
	}

	return createdItems, nil
}

// ensureSimpleJsonFile checks and creates a simple JSON file if it doesn't exist.
// Returns the path if created, or empty string, and an error.
func ensureSimpleJsonFile(path string, defaultData interface{}) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := writeJsonFile(path, defaultData); err != nil {
			return "", err
		}
		return path, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to check config file '%s': %w", path, err)
	}
	return "", nil
}

// ensureUsersFile handles checking/creating users.json
// Returns the path if created, or empty string, and an error.
func ensureUsersFile() (string, error) {
	usersPath := filepath.Join(configsDir, "users.json")
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		if err := initializeUsersFile(); err != nil {
			return "", fmt.Errorf("failed to initialize users file: %w", err)
		}
		return usersPath, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to check users file '%s': %w", usersPath, err)
	}
	return "", nil
}

// ensureConfigFile handles checking/creating config.toml
// Returns the path if created, or empty string, and an error.
func ensureConfigFile(configs *Configs) (string, error) {
	configTomlPath := filepath.Join(configsDir, "config.toml")
	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
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
		return configTomlPath, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to check config file '%s': %w", configTomlPath, err)
	}
	return "", nil
}

// ensureDirExists checks if a directory exists, and creates it if not.
// Returns the path if created, or empty string, and an error.
func ensureDirExists(dirPath string) (string, error) {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path for %s: %w", dirPath, err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return "", fmt.Errorf("failed to create directory '%s': %w", absPath, err)
		}
		return dirPath, nil
	} else if err != nil {
		return "", fmt.Errorf("failed to check directory '%s': %w", absPath, err)
	}
	return "", nil
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
			Password: "", // Will be generated
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
		// Generate user ID
		userID, err := generateUserID(u.Username)
		if err != nil {
			return fmt.Errorf("failed to generate user ID for %s: %w", u.Username, err)
		}

		// Generate random password if not provided
		password := u.Password
		if password == "" {
			password, err = generateRandomString(12)
			if err != nil {
				return fmt.Errorf("failed to generate random password for %s: %w", u.Username, err)
			}
		}

		// Hash the password
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("failed to hash password for %s: %w", u.Username, err)
		}

		// Create user with creation time
		now := time.Now().UTC()

		// Generate user-specific API JWT secret
		apiJwtSecret, err := generateRandomString(32)
		if err != nil {
			return fmt.Errorf("failed to generate API JWT secret for %s: %w", u.Username, err)
		}

		usersConfig.Users[userID] = models.User{
			ID:        userID,
			Username:  u.Username,
			Password:  string(hashedPassword),
			Name:      u.Name,
			Email:     u.Email,
			Scopes:    u.Scopes,
			CreatedAt: models.RFC3339Time(now),
			Active:    true,
			API: models.UserAPISettings{
				JwtSecret: apiJwtSecret,
			},
		}

		// Log the generated password for admin user
		if u.Username == "admin" {
			log.Printf("Generated admin password: %s", password)
		}
	}

	// Save the users configuration
	if err := saveUsersConfig(usersConfig); err != nil {
		return fmt.Errorf("failed to save users configuration: %w", err)
	}

	return nil
}

// generateRandomString generates a random string of specified length
func generateRandomString(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		num, err := crand.Int(crand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", fmt.Errorf("failed to generate random number: %w", err)
		}
		result[i] = charset[num.Int64()]
	}
	return string(result), nil
}

// writeJsonFile writes a JSON file with pretty formatting
func writeJsonFile(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON data: %w", err)
	}
	return os.WriteFile(path, jsonData, 0644)
}

// writeTomlFile writes a TOML file
func writeTomlFile(path string, data interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create TOML file: %w", err)
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode TOML data: %w", err)
	}
	return nil
}

// findAvailablePort finds an available port in the specified range
func findAvailablePort(start, end int) (int, error) {
	// Calculate the range size
	rangeSize := end - start + 1

	// Create a slice of all possible ports
	ports := make([]int, rangeSize)
	for i := range ports {
		ports[i] = start + i
	}

	// Shuffle the ports
	for i := range ports {
		j := i + rand.Intn(rangeSize-i)
		ports[i], ports[j] = ports[j], ports[i]
	}

	// Try each port in random order
	for _, port := range ports {
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available ports found in range %d-%d", start, end)
}

// generateUserID generates a unique user ID
func generateUserID(username string) (string, error) {
	// Generate a random hex string
	randomBytes := make([]byte, 4)
	if _, err := crand.Read(randomBytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	randomHex := hex.EncodeToString(randomBytes)

	// Combine username and random hex
	userID := fmt.Sprintf("usr_%s_%s", username, randomHex)
	return userID, nil
}

// saveUsersConfig saves the provided user config to the file.
func saveUsersConfig(config *models.UsersConfig) error {
	filePath := filepath.Join(configsDir, "users.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal users config: %w", err)
	}
	return os.WriteFile(filePath, data, 0644)
}
