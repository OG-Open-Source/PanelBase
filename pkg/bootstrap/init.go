package bootstrap

import (
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pelletier/go-toml/v2"
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

		// Construct default TOML content dynamically
		defaultContent := fmt.Sprintf(`[server]
ip    = "%s"
port  = %d
entry = "%s"
mode  = "release"

[auth]
jwt_secret = "%s"
token_duration_minutes = %d

[functions]
plugins = false
commands = false
users = false
themes = false
`, defaultIP, port, entry, generateRandomString(32), 60) // Generate a random secret, default duration 60 min

		// Write the new config file
		if errWrite := os.WriteFile(configAbsPath, []byte(defaultContent), 0664); errWrite != nil {
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

	// 4. Ensure users.json exists (create if not)
	usersAbsPath, _ := filepath.Abs(usersFile)
	if _, err := os.Stat(usersAbsPath); os.IsNotExist(err) {
		// log.Printf("Creating file: %s", usersFile) // Remove creation log
		if errWrite := os.WriteFile(usersAbsPath, []byte("[]\n"), 0664); errWrite != nil {
			log.Printf("WARN: Failed to create file %s: %v", usersFile, errWrite)
		}
	} else if err != nil {
		log.Printf("WARN: Failed to check file %s: %v", usersFile, err)
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
