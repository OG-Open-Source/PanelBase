package bootstrap

import (
	"bytes"
	crand "crypto/rand"
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
	Themes   map[string]interface{}
	Commands map[string]interface{}
	Plugins  map[string]interface{}
	Users    map[string]interface{}
	Config   map[string]interface{}
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
	}
}

// Bootstrap checks and creates necessary configuration files and directories.
func Bootstrap() error {
	// Ensure the configs directory exists
	if err := ensureDirExists(configsDir); err != nil {
		return fmt.Errorf("failed to ensure configs directory: %w", err)
	}

	// Ensure the logs directory exists
	if err := ensureDirExists(logsDir); err != nil {
		return fmt.Errorf("failed to ensure logs directory: %w", err)
	}

	// Create default config structure (used if files need creation)
	configs := NewConfigs()

	// --- Check/Create individual config files if they don't exist ---

	// Check/Create users.json (handles dynamic values)
	usersPath := filepath.Join(configsDir, "users.json")
	if _, err := os.Stat(usersPath); os.IsNotExist(err) {
		log.Printf("File '%s' not found, creating with defaults...", usersPath)
		if err := initializeUsersFile(); err != nil {
			return fmt.Errorf("failed to initialize users file: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check users file '%s': %w", usersPath, err)
	}

	// Check/Create config.toml (handles dynamic port)
	configTomlPath := filepath.Join(configsDir, "config.toml")
	if _, err := os.Stat(configTomlPath); os.IsNotExist(err) {
		log.Printf("File '%s' not found, creating with defaults...", configTomlPath)
		// Generate dynamic port ONLY if creating the file
		port, err := findAvailablePort(1024, 49151)
		if err != nil {
			return fmt.Errorf("failed to find available port for new config file: %w", err)
		}

		// Start with the default static config structure
		defaultConfigData := configs.Config // Get map[string]interface{} from NewConfigs()

		// Inject the dynamic port
		if serverConf, ok := defaultConfigData["server"].(map[string]interface{}); ok {
			serverConf["port"] = port
		} else {
			// This should not happen if NewConfigs is correct
			return fmt.Errorf("internal error: invalid structure for default server config")
		}

		if err := writeTomlFile(configTomlPath, defaultConfigData); err != nil {
			return err
		}
	} else if err != nil {
		return fmt.Errorf("failed to check config file '%s': %w", configTomlPath, err)
	}

	// Check/Create other simple JSON files (themes, commands, plugins)
	simpleJsonConfigs := map[string]interface{}{
		filepath.Join(configsDir, "themes.json"):   configs.Themes,
		filepath.Join(configsDir, "commands.json"): configs.Commands,
		filepath.Join(configsDir, "plugins.json"):  configs.Plugins,
	}
	for path, data := range simpleJsonConfigs {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			log.Printf("File '%s' not found, creating with defaults...", path)
			if err := writeJsonFile(path, data); err != nil {
				return err // Return error if writing fails
			}
		} else if err != nil {
			return fmt.Errorf("failed to check config file '%s': %w", path, err)
		}
	}

	return nil
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
			Scopes: models.UserPermissions{ // Define admin scopes directly
				"commands": {"read:list", "read:item", "install", "execute", "update", "delete"},
				"plugins":  {"read:list", "read:item", "install", "update", "delete"},
				"themes":   {"read:list", "read:item", "install", "update", "delete"},
				"users":    {"read:list", "read:item", "create", "update", "delete"},
				"api":      {"read:list", "read:item", "create", "update", "delete"},
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
		userID, err := generateRandomString(8) // Generate ID for each user
		if err != nil {
			log.Printf("Warning: Failed to generate unique user ID for %s: %v. Skipping user.", u.Username, err)
			continue
		}
		userID = "usr_" + userID

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
			CreatedAt: time.Now().UTC(),
			Active:    true,
			Scopes:    u.Scopes,
			API: models.UserAPISettings{
				JwtSecret: userJwtSecret,
				Tokens:    make(map[string]models.APIToken), // Initialize correctly
			},
		}
		usersConfig.Users[u.Username] = userData // Add user to the map using username as key
	}

	// Save the initialized users config using the user service save function
	// Need to access the save function from the user service.
	// For now, assume user package has a SaveConfig function (needs adding to userservice.go)
	if err := user.SaveUsersConfigForBootstrap(usersConfig); err != nil { // Placeholder name
		log.Printf("Error initializing users.json via service: %v", err)
		return err
	}

	log.Println("users.json initialized successfully via service.")
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

// ensureDirExists checks if a directory exists, and creates it if not.
func ensureDirExists(dirPath string) error {
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path for %s: %w", dirPath, err)
	}
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		log.Printf("Directory '%s' not found, creating...", absPath)
		// Create the directory with permissions 0755
		// 0755 means owner can read/write/execute, group and others can read/execute
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", absPath, err)
		}
		log.Printf("Directory '%s' created successfully.", absPath)
	} else if err != nil {
		// Handle other potential errors from os.Stat (e.g., permission denied)
		return fmt.Errorf("failed to check directory '%s': %w", absPath, err)
	}
	return nil
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
	log.Printf("Successfully created default file: %s", filePath)
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
	log.Printf("Successfully created default file: %s", filePath)
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
