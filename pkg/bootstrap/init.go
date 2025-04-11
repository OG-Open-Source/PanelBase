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
	configDir  = "configs"
	themeDir   = "themes"
	webDir     = "web"
	pluginDir  = "plugins"
	commandDir = "commands"
	configFile = "configs/config.toml"
	usersFile  = "configs/users.json"

	minPort          = 1024
	maxPort          = 49151
	portCheckRetries = 20
	entryLength      = 12
	defaultIP        = "0.0.0.0"
)

// configStructure mirrors the TOML structure for parsing
type configStructure struct {
	Server struct {
		Port  int    `toml:"port"`
		Entry string `toml:"entry"`
		Ip    string `toml:"ip"`
	}
	Functions struct {
		Plugins  bool `toml:"plugins"`
		Commands bool `toml:"commands"`
		Users    bool `toml:"users"`
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
		log.Printf("Creating directory: %s", dirPath)
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
	log.Println("Checking project structure...")

	// 1. Ensure mandatory directories exist
	if err := ensureDir(configDir); err != nil {
		return err
	}
	if err := ensureDir(themeDir); err != nil {
		return err
	}
	if err := ensureDir(webDir); err != nil {
		return err
	}

	// 2. Ensure config.toml exists (create with dynamic defaults if not)
	configAbsPath, _ := filepath.Abs(configFile)
	if _, err := os.Stat(configAbsPath); os.IsNotExist(err) {
		log.Printf("Creating configuration file: %s", configFile)

		// Generate random entry
		entry := generateRandomString(entryLength)

		// Find available port
		var port int = -1
		for i := 0; i < portCheckRetries; i++ {
			testPort := rand.Intn(maxPort-minPort+1) + minPort
			if isPortAvailable(testPort) {
				port = testPort
				log.Printf("Found available port: %d", port)
				break
			}
		}
		if port == -1 {
			return fmt.Errorf("failed to find an available port after %d retries", portCheckRetries)
		}

		// Construct default TOML content dynamically
		defaultContent := fmt.Sprintf(`
[server]
ip    = "%s"
port  = %d
entry = "%s"

[functions]
plugins = false
commands = false
users = false
`, defaultIP, port, entry)

		// Write the new config file
		if errWrite := os.WriteFile(configAbsPath, []byte(defaultContent), 0664); errWrite != nil {
			return fmt.Errorf("failed to create file %s: %w", configFile, errWrite)
		}

	} else if err != nil {
		// Error checking the file (other than NotExist)
		return fmt.Errorf("failed to check config file %s: %w", configFile, err)
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

	// 4. Ensure entry-specific web directory exists (using the config.Server.Entry value)
	if config.Server.Entry != "" {
		entryWebDir := filepath.Join(webDir, config.Server.Entry)
		if err := ensureDir(entryWebDir); err != nil {
			log.Printf("WARN: Failed to create entry-specific web directory '%s': %v", entryWebDir, err)
		}
	} else {
		log.Printf("WARN: Server entry is empty in config, cannot create entry-specific web directory.")
	}

	// 5. Conditional creation based on config
	if config.Functions.Plugins {
		if err := ensureDir(pluginDir); err != nil {
			log.Printf("WARN: Failed during conditional creation: %v", err) // Log warning but continue
		}
	}
	if config.Functions.Commands {
		if err := ensureDir(commandDir); err != nil {
			log.Printf("WARN: Failed during conditional creation: %v", err)
		}
	}
	if config.Functions.Users {
		usersAbsPath, _ := filepath.Abs(usersFile)
		if _, err := os.Stat(usersAbsPath); os.IsNotExist(err) {
			log.Printf("Creating file: %s", usersFile)
			if err := os.WriteFile(usersAbsPath, []byte("{}\n"), 0664); err != nil {
				log.Printf("WARN: Failed to create file %s: %v", usersFile, err)
			}
		} // No need for ensureFile complex logic here
	}

	log.Println("Project structure check complete.")
	return nil
}
