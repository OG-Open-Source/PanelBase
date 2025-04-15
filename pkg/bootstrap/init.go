package bootstrap

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/models"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/pelletier/go-toml/v2"
	"golang.org/x/crypto/bcrypt"
)

const (
	configDir      = "configs"
	themeDir       = "themes"
	webDir         = "web"
	pluginDir      = "plugins"
	commandDir     = "commands"
	configFile     = "configs/config.toml"
	usersFile      = "configs/users.json"
	uiSettingsFile = "configs/ui_settings.json"

	minPort          = 1024
	maxPort          = 49151
	portCheckRetries = 20
	entryLength      = 12
	defaultIP        = "0.0.0.0"

	initialUsername       = "admin"
	initialPasswordLength = 16
)

// Default content for ui_settings.json if it doesn't exist
const defaultUISettingsContent = `{
  "site_title": "PanelBase",
  "welcome_message": "Welcome to the Panel!"
}
`

// Default content for index.html if it doesn't exist
const defaultIndexContent = `<!DOCTYPE html>
<html lang="en">

<head>
	<meta charset="UTF-8">
	<title>{{ .site_title }}</title>
</head>

<body>
	<h1>{{ .welcome_message }}</h1>
	<p>Edit this file in your web directory to customize the panel entry point.</p>
</body>

</html>
`

// initialAdminScopes defines the default scopes for the first admin user.
// Using "*" grants all actions for the resource and its sub-resources.
var initialAdminScopes = map[string]interface{}{
	"users":    "*",
	"account":  "*", // Technically covered by users:*, but explicit for clarity
	"themes":   "*",
	"plugins":  "*",
	"commands": "*",
}

// usersFileFormat mirrors the structure expected in users.json
// Duplicated here to avoid circular dependency with storage, consider refactoring.
type bootstrapUsersFileFormat struct {
	Users map[string]*models.User `json:"users"`
}

// configStructure mirrors the TOML structure for parsing
type configStructure struct {
	Server struct {
		Port  int    `toml:"port"`
		Entry string `toml:"entry"`
		Ip    string `toml:"ip"`
		Mode  string `toml:"mode"`
	}
	Auth struct {
		JwtSecret            string `toml:"jwt_secret"`
		TokenDurationMinutes int    `toml:"token_duration_minutes"`
	}
	Functions struct {
		Plugins  bool `toml:"plugins"`
		Commands bool `toml:"commands"`
		Users    bool `toml:"users"`
		Themes   bool `toml:"themes"`
	}
}

// seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano())) // Seed random number generator

func init() {
	rand.Seed(time.Now().UnixNano())
}

// generateRandomString generates a random alphanumeric string of a given length.
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// isPortAvailable checks if a TCP port is available on the default IP.
func isPortAvailable(port int) bool {
	address := net.JoinHostPort(defaultIP, strconv.Itoa(port))
	listener, err := net.Listen("tcp", address)
	if err != nil {
		// Assume error means port is likely occupied or invalid
		return false
	}
	listener.Close() // Close the listener immediately
	return true
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(dirPath string) error {
	absPath, _ := filepath.Abs(dirPath)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		// log.Printf("Creating directory: %s", dirPath) // Remove creation log
		if err := os.MkdirAll(absPath, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dirPath, err)
		}
	} else if err == nil {
		// log.Printf("Directory already exists: %s", dirPath) // Optional: reduce noise
	} else {
		return fmt.Errorf("failed to check directory %s: %w", dirPath, err)
	}
	return nil
}

// createDefaultConfig generates a default TOML configuration for the server.
func createDefaultConfig(entry string, port int) ([]byte, error) {
	// Construct default TOML content dynamically
	// Generate a random secret, default duration 60 min
	defaultJwtSecret := generateRandomString(32)
	defaultTokenDuration := 60

	// Default scopes for newly created users
	defaultScopesMap := map[string]interface{}{
		"account": map[string]interface{}{
			"profile":     []string{"read", "update"},
			"password":    []string{"update"},
			"tokens":      []string{"create", "read", "delete"},
			"self_delete": []string{"execute"},
		},
	}

	// Use a struct to marshal TOML cleanly
	// Define nested structs for Auth defaults and rules
	type AuthConfigDefaults struct {
		Scopes map[string]interface{} `toml:"scopes"`
	}
	type AuthConfigRules struct {
		RequireOldPasswordForUpdate bool     `toml:"require_old_password_for_update"`
		AllowSelfDelete             bool     `toml:"allow_self_delete"`
		ProtectedUserIDs            []string `toml:"protected_user_ids"`
	}
	type DefaultConfig struct {
		Server struct {
			Ip           string `toml:"ip"`
			Port         int    `toml:"port"`
			Entry        string `toml:"entry"`
			Mode         string `toml:"mode"`
			TrustedProxy string `toml:"trusted_proxy"`
		}
		Auth struct {
			JwtSecret            string             `toml:"jwt_secret"`
			TokenDurationMinutes int                `toml:"token_duration_minutes"`
			Defaults             AuthConfigDefaults `toml:"defaults"`
			Rules                AuthConfigRules    `toml:"rules"`
		}
		Functions struct {
			Plugins  bool `toml:"plugins"`
			Commands bool `toml:"commands"`
			Users    bool `toml:"users"`
			Themes   bool `toml:"themes"`
		}
	}

	configData := DefaultConfig{
		Server: struct {
			Ip           string `toml:"ip"`
			Port         int    `toml:"port"`
			Entry        string `toml:"entry"`
			Mode         string `toml:"mode"`
			TrustedProxy string `toml:"trusted_proxy"`
		}{
			Ip:           defaultIP,
			Port:         port,
			Entry:        entry,
			Mode:         "release",
			TrustedProxy: "",
		},
		Auth: struct {
			JwtSecret            string             `toml:"jwt_secret"`
			TokenDurationMinutes int                `toml:"token_duration_minutes"`
			Defaults             AuthConfigDefaults `toml:"defaults"`
			Rules                AuthConfigRules    `toml:"rules"`
		}{
			JwtSecret:            defaultJwtSecret,
			TokenDurationMinutes: defaultTokenDuration,
			Defaults: AuthConfigDefaults{
				Scopes: defaultScopesMap,
			},
			Rules: AuthConfigRules{
				RequireOldPasswordForUpdate: true,       // Default: Require old password for account updates
				AllowSelfDelete:             true,       // Default: Allow users to delete themselves
				ProtectedUserIDs:            []string{}, // Default: No protected users initially
			},
		},
		Functions: struct {
			Plugins  bool `toml:"plugins"`
			Commands bool `toml:"commands"`
			Users    bool `toml:"users"`
			Themes   bool `toml:"themes"`
		}{
			Plugins:  false,
			Commands: false,
			Users:    false,
			Themes:   false,
		},
	}

	tomlBytes, err := toml.Marshal(configData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config to TOML: %w", err)
	}

	return tomlBytes, nil
}

// InitializeProject ensures necessary directories and files exist based on configuration.
func InitializeProject() error {
	// log.Println("Checking project structure...")

	// 1. Ensure mandatory directories exist (EXCLUDING themes now)
	if err := ensureDir(configDir); err != nil {
		return err
	}
	if err := ensureDir(webDir); err != nil {
		return err
	}

	// 2. Ensure config.toml exists (create with dynamic defaults if not)
	configAbsPath, _ := filepath.Abs(configFile)
	if _, err := os.Stat(configAbsPath); os.IsNotExist(err) {
		// log.Printf("Creating configuration file: %s", configFile) // Remove creation log

		// Generate random entry
		entry := generateRandomString(entryLength)

		// Find available port
		var port int = -1
		for i := 0; i < portCheckRetries; i++ {
			testPort := rand.Intn(maxPort-minPort+1) + minPort
			if isPortAvailable(testPort) {
				port = testPort
				// log.Printf("Found available port: %d", port)
				break
			}
		}
		if port == -1 {
			return fmt.Errorf("failed to find an available port after %d retries", portCheckRetries)
		}

		// Write the new config file
		defaultContentBytes, errCreate := createDefaultConfig(entry, port)
		if errCreate != nil {
			return fmt.Errorf("failed to create default config content: %w", errCreate)
		}
		if errWrite := os.WriteFile(configAbsPath, defaultContentBytes, 0664); errWrite != nil {
			return fmt.Errorf("failed to create file %s: %w", configFile, errWrite)
		}

	} else if err != nil {
		return fmt.Errorf("failed to check config file %s: %w", configFile, err)
	}

	// 3. Ensure ui_settings.json exists (create with default if not)
	uiSettingsAbsPath, _ := filepath.Abs(uiSettingsFile)
	if _, err := os.Stat(uiSettingsAbsPath); os.IsNotExist(err) {
		// log.Printf("Creating file: %s", uiSettingsFile) // Remove creation log
		if errWrite := os.WriteFile(uiSettingsAbsPath, []byte(defaultUISettingsContent), 0664); errWrite != nil {
			log.Printf("WARN: Failed to create file %s: %v", uiSettingsFile, errWrite)
		}
	} else if err != nil {
		log.Printf("WARN: Failed to check file %s: %v", uiSettingsFile, err)
	}

	// 4. Ensure users.json exists and create initial admin user if it does not.
	usersAbsPath, _ := filepath.Abs(usersFile)
	if _, err := os.Stat(usersAbsPath); os.IsNotExist(err) {
		// log.Printf("INFO: users.json not found. Creating file and initial admin user...") // Removed this log

		// Generate initial admin credentials
		initialPassword := generateRandomString(initialPasswordLength)
		hashedPassword, hashErr := bcrypt.GenerateFromPassword([]byte(initialPassword), bcrypt.DefaultCost)
		if hashErr != nil {
			return fmt.Errorf("failed to hash initial admin password: %w", hashErr)
		}

		// Generate admin user ID using utils constants
		adminID, idErr := utils.GeneratePrefixedID(utils.UserIDPrefix, utils.UserIDRandomLength)
		if idErr != nil {
			return fmt.Errorf("failed to generate initial admin user ID: %w", idErr)
		}

		initialAdminUser := &models.User{
			ID:        adminID,
			Username:  initialUsername,
			Password:  string(hashedPassword),
			Name:      "Administrator",
			Email:     "",
			CreatedAt: time.Now().UTC().Truncate(time.Second),
			Active:    true,
			Scopes:    initialAdminScopes,
		}

		// Prepare the file content
		usersData := bootstrapUsersFileFormat{
			Users: map[string]*models.User{
				adminID: initialAdminUser,
			},
		}

		jsonData, marshalErr := json.MarshalIndent(usersData, "", "  ")
		if marshalErr != nil {
			return fmt.Errorf("failed to marshal initial user data: %w", marshalErr)
		}

		// Write the new users.json file
		if errWrite := os.WriteFile(usersAbsPath, jsonData, 0664); errWrite != nil {
			return fmt.Errorf("failed to create file %s: %w", usersFile, errWrite)
		}

		// Keep the important credential logging block
		log.Println("###########################################################")
		log.Println("IMPORTANT: Initial administrator account created:")
		log.Printf("  Username: %s", initialUsername)
		log.Printf("  Password: %s", initialPassword)
		log.Println("Please log in immediately and change the password.")
		log.Println("###########################################################")

		// --- Add admin ID to protected list in config ---
		if err := addAdminToProtectedList(configAbsPath, adminID); err != nil {
			// Log warning but don't fail the entire bootstrap process
			log.Printf("WARN: Failed to add initial admin ID (%s) to protected list in config '%s': %v", adminID, configFile, err)
		}
		// --- End adding admin ID ---

	} else if err != nil {
		// Error checking the file (permissions?)
		return fmt.Errorf("failed to check file %s: %w", usersFile, err)
	} else {
		// users.json exists, no action needed for initial user here.
		// log.Printf("INFO: users.json found.")
	}

	// --- Reading config for conditional creation (moved down) ---
	configData, err := os.ReadFile(configFile)
	config := configStructure{} // Default values are false

	if err != nil {
		log.Printf("WARN: Failed to read %s: %v. Assuming all optional features disabled.", configFile, err)
	} else {
		err = toml.Unmarshal(configData, &config)
		if err != nil {
			log.Printf("WARN: Failed to parse %s: %v. Assuming all optional features disabled.", configFile, err)
		}
	}

	// 5. Ensure entry-specific web directory exists (using the config.Server.Entry value)
	targetWebDir := webDir // Default to base web directory
	if config.Server.Entry != "" {
		entryWebDir := filepath.Join(webDir, config.Server.Entry)
		if err := ensureDir(entryWebDir); err != nil {
			log.Printf("WARN: Failed to create entry-specific web directory '%s': %v", entryWebDir, err)
		} else {
			targetWebDir = entryWebDir // Update target if successfully created/exists
		}
	} else {
		// log.Printf("INFO: server.entry is empty, using base web directory '%s'", webDir)
	}

	// 6. Conditional directory creation based on config
	if config.Functions.Plugins {
		if err := ensureDir(pluginDir); err != nil {
			log.Printf("WARN: Failed during conditional directory creation (plugins): %v", err)
		}
	}
	if config.Functions.Commands {
		if err := ensureDir(commandDir); err != nil {
			log.Printf("WARN: Failed during conditional directory creation (commands): %v", err)
		}
	}
	if config.Functions.Themes { // Create themes dir conditionally
		if err := ensureDir(themeDir); err != nil {
			log.Printf("WARN: Failed during conditional directory creation (themes): %v", err)
		}
	}

	// 7. Ensure a default index file exists in the target web directory
	indexPathHtml := filepath.Join(targetWebDir, "index.html")
	indexPathHtm := filepath.Join(targetWebDir, "index.htm")

	_, errHtml := os.Stat(indexPathHtml)
	_, errHtm := os.Stat(indexPathHtm)

	if os.IsNotExist(errHtml) && os.IsNotExist(errHtm) {
		// log.Printf("Creating default index file: %s", indexPathHtml) // Remove creation log
		if errWrite := os.WriteFile(indexPathHtml, []byte(defaultIndexContent), 0664); errWrite != nil {
			log.Printf("WARN: Failed to create default index file %s: %v", indexPathHtml, errWrite)
		}
	} else {
		// log.Printf("Index file (index.html or index.htm) already exists in %s", targetWebDir)
	}

	// log.Println("Project structure check complete.")
	return nil
}

// addAdminToProtectedList reads the config, adds the adminID, and writes it back.
func addAdminToProtectedList(configPath string, adminID string) error {
	// Read the existing config file
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file for update: %w", err)
	}

	// Use the same struct definition as createDefaultConfig for consistency
	// Define nested structs for Auth defaults and rules
	type AuthConfigDefaults struct {
		Scopes map[string]interface{} `toml:"scopes"`
	}
	type AuthConfigRules struct {
		RequireOldPasswordForUpdate bool     `toml:"require_old_password_for_update"`
		AllowSelfDelete             bool     `toml:"allow_self_delete"`
		ProtectedUserIDs            []string `toml:"protected_user_ids"`
	}
	type DefaultConfig struct {
		Server struct {
			Ip           string `toml:"ip"`
			Port         int    `toml:"port"`
			Entry        string `toml:"entry"`
			Mode         string `toml:"mode"`
			TrustedProxy string `toml:"trusted_proxy"`
		} `toml:"server"` // Add struct tags for top-level keys
		Auth struct {
			JwtSecret            string             `toml:"jwt_secret"`
			TokenDurationMinutes int                `toml:"token_duration_minutes"`
			Defaults             AuthConfigDefaults `toml:"defaults"`
			Rules                AuthConfigRules    `toml:"rules"`
		} `toml:"auth"`
		Functions struct {
			Plugins  bool `toml:"plugins"`
			Commands bool `toml:"commands"`
			Users    bool `toml:"users"`
			Themes   bool `toml:"themes"`
		} `toml:"functions"`
	}

	var parsedConfig DefaultConfig
	err = toml.Unmarshal(configBytes, &parsedConfig)
	if err != nil {
		return fmt.Errorf("failed to parse config file for update: %w", err)
	}

	// Add the admin ID if not already present
	found := false
	for _, id := range parsedConfig.Auth.Rules.ProtectedUserIDs {
		if id == adminID {
			found = true
			break
		}
	}
	if !found {
		parsedConfig.Auth.Rules.ProtectedUserIDs = append(parsedConfig.Auth.Rules.ProtectedUserIDs, adminID)
		// log.Printf("INFO: Added initial admin ID %s to protected list in config.", adminID) // Commented out this log
	} else {
		// This case shouldn't happen on initial creation, but good to handle
		log.Printf("INFO: Initial admin ID %s was already in protected list.", adminID)
		return nil // No need to rewrite if already present
	}

	// Marshal the updated config back to TOML
	updatedTomlBytes, err := toml.Marshal(parsedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Write the updated config back to the file
	err = os.WriteFile(configPath, updatedTomlBytes, 0664)
	if err != nil {
		return fmt.Errorf("failed to write updated config file: %w", err)
	}

	return nil
}
