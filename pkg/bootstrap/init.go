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
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

const (
	configDir      = "configs"
	webDir         = "web"
	extDir         = "ext"
	themeDir       = "ext/themes"
	pluginDir      = "ext/plugins"
	commandDir     = "ext/commands"
	configFile     = "configs/config.yaml"
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

// configStructure mirrors the YAML structure for parsing
type configStructure struct {
	Server struct {
		Port         int    `yaml:"port"`
		Entry        string `yaml:"entry"`
		Ip           string `yaml:"ip"`
		Mode         string `yaml:"mode"`
		TrustedProxy string `yaml:"trusted_proxy"`
	}
	Auth struct {
		JwtSecret    string `yaml:"jwt_secret"`
		TokenMinutes int    `yaml:"token_minutes"`
		Defaults     struct {
			Scopes map[string]interface{} `yaml:"scopes"`
		} `yaml:"defaults"`
		Rules struct {
			RequireOldPw    bool     `yaml:"require_old_pw"`
			AllowSelfDelete bool     `yaml:"allow_self_delete"`
			ProtectedUsers  []string `yaml:"protected_users"`
		} `yaml:"rules"`
	}
	Features struct {
		Plugins  bool `yaml:"plugins"`
		Commands bool `yaml:"commands"`
		Users    bool `yaml:"users"`
		Themes   bool `yaml:"themes"`
	} `yaml:"features"`
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

// createDefaultConfig generates a default YAML configuration for the server.
func createDefaultConfig(entry string, port int) ([]byte, error) {
	defaultJwtSecret := generateRandomString(32)
	defaultTokenMinutes := 60
	defaultConfig := configStructure{}
	defaultConfig.Server.Ip = defaultIP
	defaultConfig.Server.Port = port
	defaultConfig.Server.Entry = entry
	defaultConfig.Server.Mode = "release"
	defaultConfig.Server.TrustedProxy = ""
	defaultConfig.Auth.JwtSecret = defaultJwtSecret
	defaultConfig.Auth.TokenMinutes = defaultTokenMinutes
	defaultConfig.Auth.Defaults.Scopes = map[string]interface{}{
		"users":   []string{"read", "create", "update", "delete"},
		"account": []string{"profile:read", "profile:update", "password:update", "self_delete", "tokens:create", "tokens:read", "tokens:delete"},
	}
	defaultConfig.Auth.Rules.RequireOldPw = true
	defaultConfig.Auth.Rules.AllowSelfDelete = true
	defaultConfig.Auth.Rules.ProtectedUsers = []string{}
	defaultConfig.Features.Plugins = false
	defaultConfig.Features.Commands = false
	defaultConfig.Features.Users = true
	defaultConfig.Features.Themes = false
	// Marshal to YAML
	yamlBytes, err := yaml.Marshal(defaultConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config to YAML: %w", err)
	}
	return yamlBytes, nil
}

// InitializeProject ensures necessary directories and files exist based on configuration.
func InitializeProject() error {
	// log.Println("Checking project structure...")

	// Step 1: Ensure config directory exists
	if err := ensureDir(configDir); err != nil {
		return err
	}

	// Step 2: Ensure config.yaml exists (create with dynamic defaults if not)
	configAbsPath, _ := filepath.Abs(configFile)
	if _, err := os.Stat(configAbsPath); os.IsNotExist(err) {
		defaultConfigBytes, err := createDefaultConfig(generateRandomString(entryLength), rand.Intn(maxPort-minPort+1)+minPort)
		if err != nil {
			return err
		}
		err = os.WriteFile(configAbsPath, defaultConfigBytes, 0644)
		if err != nil {
			return err
		}
	}

	// Step 3: Load config.yaml into config struct
	var config configStructure
	configBytes, err := os.ReadFile(configAbsPath)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(configBytes, &config)
	if err != nil {
		return err
	}

	// Step 4: Conditionally create ext subdirectories based on enabled features
	// [SECURITY] The ext directory will only be created if at least one feature (themes, plugins, commands) is enabled.
	if config.Features.Themes {
		if err := ensureDir(themeDir); err != nil {
			return err
		}
	}
	if config.Features.Plugins {
		if err := ensureDir(pluginDir); err != nil {
			return err
		}
	}
	if config.Features.Commands {
		if err := ensureDir(commandDir); err != nil {
			return err
		}
	}

	// 5. Ensure ui_settings.json exists (create with default if not)
	uiSettingsAbsPath, _ := filepath.Abs(uiSettingsFile)
	if _, err := os.Stat(uiSettingsAbsPath); os.IsNotExist(err) {
		// log.Printf("Creating file: %s", uiSettingsFile) // Remove creation log
		if errWrite := os.WriteFile(uiSettingsAbsPath, []byte(defaultUISettingsContent), 0664); errWrite != nil {
			log.Printf("WARN: Failed to create file %s: %v", uiSettingsFile, errWrite)
		}
	} else if err != nil {
		log.Printf("WARN: Failed to check file %s: %v", uiSettingsFile, err)
	}

	// 6. Ensure users.json exists and create initial admin user if it does not.
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
			UserID:    adminID,
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

	// 7. Ensure entry-specific web directory exists (using the config.Server.Entry value)
	// [SECURITY] Web content must only be placed in webDir or webDir/<entry>. Fallback to ext is strictly prohibited.
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

	// 8. Conditional directory creation based on config
	if config.Features.Plugins {
		if err := ensureDir(pluginDir); err != nil {
			log.Printf("WARN: Failed during conditional directory creation (plugins): %v", err)
		}
		// Ensure configs/plugins.json exists
		pluginsJsonPath := filepath.Join(configDir, "plugins.json")
		if _, err := os.Stat(pluginsJsonPath); os.IsNotExist(err) {
			pluginsObj := map[string]interface{}{"plugins": map[string]interface{}{}}
			pluginsBytes, err := json.MarshalIndent(pluginsObj, "", "  ")
			if err != nil {
				log.Printf("WARN: Failed to marshal default plugins.json: %v", err)
			} else if err := os.WriteFile(pluginsJsonPath, pluginsBytes, 0664); err != nil {
				log.Printf("WARN: Failed to create plugins.json: %v", err)
			}
		}
	}
	if config.Features.Commands {
		if err := ensureDir(commandDir); err != nil {
			log.Printf("WARN: Failed during conditional directory creation (commands): %v", err)
		}
		// Ensure configs/commands.json exists
		commandsJsonPath := filepath.Join(configDir, "commands.json")
		if _, err := os.Stat(commandsJsonPath); os.IsNotExist(err) {
			commandsObj := map[string]interface{}{"commands": map[string]interface{}{}}
			commandsBytes, err := json.MarshalIndent(commandsObj, "", "  ")
			if err != nil {
				log.Printf("WARN: Failed to marshal default commands.json: %v", err)
			} else if err := os.WriteFile(commandsJsonPath, commandsBytes, 0664); err != nil {
				log.Printf("WARN: Failed to create commands.json: %v", err)
			}
		}
	}
	if config.Features.Themes {
		if err := ensureDir(themeDir); err != nil {
			log.Printf("WARN: Failed during conditional directory creation (themes): %v", err)
		}
		// Ensure configs/themes.json exists
		themesJsonPath := filepath.Join(configDir, "themes.json")
		if _, err := os.Stat(themesJsonPath); os.IsNotExist(err) {
			themesObj := map[string]interface{}{"themes": map[string]interface{}{}}
			themesBytes, err := json.MarshalIndent(themesObj, "", "  ")
			if err != nil {
				log.Printf("WARN: Failed to marshal default themes.json: %v", err)
			} else if err := os.WriteFile(themesJsonPath, themesBytes, 0664); err != nil {
				log.Printf("WARN: Failed to create themes.json: %v", err)
			}
		}
	}

	// 9. Ensure a default index file exists in the target web directory
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
		Scopes map[string]interface{} `yaml:"scopes"`
	}
	type AuthConfigRules struct {
		RequireOldPasswordForUpdate bool     `yaml:"require_old_password_for_update"`
		AllowSelfDelete             bool     `yaml:"allow_self_delete"`
		ProtectedUserIDs            []string `yaml:"protected_user_ids"`
	}
	type DefaultConfig struct {
		Server struct {
			Ip           string `yaml:"ip"`
			Port         int    `yaml:"port"`
			Entry        string `yaml:"entry"`
			Mode         string `yaml:"mode"`
			TrustedProxy string `yaml:"trusted_proxy"`
		} `yaml:"server"` // Add struct tags for top-level keys
		Auth struct {
			JwtSecret            string             `yaml:"jwt_secret"`
			TokenDurationMinutes int                `yaml:"token_duration_minutes"`
			Defaults             AuthConfigDefaults `yaml:"defaults"`
			Rules                AuthConfigRules    `yaml:"rules"`
		} `yaml:"auth"`
		Functions struct {
			Plugins  bool `yaml:"plugins"`
			Commands bool `yaml:"commands"`
			Users    bool `yaml:"users"`
			Themes   bool `yaml:"themes"`
		} `yaml:"features"`
	}

	var parsedConfig DefaultConfig
	err = yaml.Unmarshal(configBytes, &parsedConfig)
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

	// Marshal the updated config back to YAML
	updatedYamlBytes, err := yaml.Marshal(parsedConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal updated config: %w", err)
	}

	// Write the updated config back to the file
	err = os.WriteFile(configPath, updatedYamlBytes, 0664)
	if err != nil {
		return fmt.Errorf("failed to write updated config file: %w", err)
	}

	return nil
}

// [SECURITY] All directory logic and comments are now in English and use webDir for web content paths. No fallback to extDir is allowed.
