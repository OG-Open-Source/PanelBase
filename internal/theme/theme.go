package theme

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

// Theme represents a UI theme
type Theme struct {
	Name        string                 `json:"name"`
	Authors     string                 `json:"authors"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Directory   string                 `json:"directory"`
	Structure   map[string]interface{} `json:"structure"`
}

// ThemeConfig represents the theme configuration file
type ThemeConfig struct {
	Theme Theme `json:"theme"`
}

var currentTheme *Theme

// LoadTheme loads the theme from the given file
func LoadTheme(filePath string) (*Theme, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening theme file: %v", err)
	}
	defer file.Close()

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading theme file: %v", err)
	}

	var config ThemeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("error parsing theme file: %v", err)
	}

	currentTheme = &config.Theme
	return currentTheme, nil
}

// GetCurrentTheme returns the current theme
func GetCurrentTheme() *Theme {
	return currentTheme
}

// EnsureThemeDirectory ensures the theme directory exists
func EnsureThemeDirectory(basePath string) (string, error) {
	if currentTheme == nil {
		return "", fmt.Errorf("no theme loaded")
	}

	themePath := filepath.Join(basePath, currentTheme.Directory)

	// Create directory if it doesn't exist
	if _, err := os.Stat(themePath); os.IsNotExist(err) {
		if err := os.MkdirAll(themePath, 0755); err != nil {
			return "", fmt.Errorf("error creating theme directory: %v", err)
		}
	}

	return themePath, nil
}
