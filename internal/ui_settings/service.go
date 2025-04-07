package ui_settings

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	logger "github.com/OG-Open-Source/PanelBase/internal/logging"
	"github.com/OG-Open-Source/PanelBase/internal/models"
)

// Define constants locally within this package
const uiSettingsFileName = "ui_settings.json"
const configsDir = "./configs"

var (
	uiSettings *models.UISettings
	// fileMutex sync.RWMutex // Use standard name `mu`
	mu sync.RWMutex
)

// DefaultUISettings returns the default UI configuration.
func DefaultUISettings() *models.UISettings {
	return &models.UISettings{
		Title:      "PanelBase",
		LogoURL:    "/assets/logo.png", // Example path
		FaviconURL: "/assets/favicon.ico",
		CustomCSS:  "",
		CustomJS:   "",
	}
}

// loadUISettings reads and parses the ui_settings.json file.
// Returns the loaded settings or default settings if the file is missing/empty.
func loadUISettings() (*models.UISettings, error) {
	filePath := filepath.Join(configsDir, uiSettingsFileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Printf("UI_SVC", "LOAD", "Settings file '%s' not found. Using defaults.", filePath)
			return DefaultUISettings(), nil
		}
		// Use ErrorPrintf for actual file read errors
		logger.ErrorPrintf("UI_SVC", "LOAD", "Failed to read ui settings file '%s': %v", filePath, err)
		return nil, fmt.Errorf("failed to read ui settings file '%s': %w", filePath, err)
	}

	// If the file is empty, use defaults
	if len(data) == 0 {
		logger.Printf("UI_SVC", "LOAD", "Settings file '%s' was empty. Using defaults.", filePath)
		return DefaultUISettings(), nil
	}

	// Unmarshal the settings into a temporary variable first
	loadedSettings := &models.UISettings{}
	if err := json.Unmarshal(data, loadedSettings); err != nil {
		logger.ErrorPrintf("UI_SVC", "LOAD", "Failed to unmarshal UI settings '%s': %v", filePath, err)
		return nil, fmt.Errorf("failed to unmarshal ui settings file '%s': %w", filePath, err)
	}

	logger.Printf("UI_SVC", "LOAD", "Loaded UI settings (Title: %s)", loadedSettings.Title)
	return loadedSettings, nil
}

// GetUISettings retrieves the current UI settings.
func GetUISettings() (*models.UISettings, error) {
	mu.RLock()
	defer mu.RUnlock()
	if uiSettings == nil {
		// Maybe attempt to load here? Or rely on LoadUISettings being called at startup.
		// Let's return an error for now if not loaded.
		return nil, fmt.Errorf("UI settings have not been loaded")
	}
	// Return a copy to prevent modification through the pointer
	settingsCopy := *uiSettings
	return &settingsCopy, nil
}

// UpdateUISettings updates the UI settings and saves them.
// It performs a partial update based on the keys present in the updateData map.
func UpdateUISettings(updateData map[string]interface{}) (*models.UISettings, error) {
	mu.Lock()
	defer mu.Unlock()

	if uiSettings == nil {
		return nil, fmt.Errorf("UI settings have not been loaded, cannot update")
	}

	originalSettings := *uiSettings // Create a copy for potential rollback (though not implemented)
	changed := false

	// Apply partial updates safely
	if title, ok := updateData["title"]; ok {
		if titleStr, okStr := title.(string); okStr {
			if uiSettings.Title != titleStr {
				uiSettings.Title = titleStr
				changed = true
			}
		}
	}
	if logoURL, ok := updateData["logo_url"]; ok {
		if logoURLStr, okStr := logoURL.(string); okStr {
			if uiSettings.LogoURL != logoURLStr {
				uiSettings.LogoURL = logoURLStr
				changed = true
			}
		}
	}
	if faviconURL, ok := updateData["favicon_url"]; ok {
		if faviconURLStr, okStr := faviconURL.(string); okStr {
			if uiSettings.FaviconURL != faviconURLStr {
				uiSettings.FaviconURL = faviconURLStr
				changed = true
			}
		}
	}
	if customCSS, ok := updateData["custom_css"]; ok {
		if customCSSStr, okStr := customCSS.(string); okStr {
			if uiSettings.CustomCSS != customCSSStr {
				uiSettings.CustomCSS = customCSSStr
				changed = true
			}
		}
	}
	if customJS, ok := updateData["custom_js"]; ok {
		if customJSStr, okStr := customJS.(string); okStr {
			if uiSettings.CustomJS != customJSStr {
				uiSettings.CustomJS = customJSStr
				changed = true
			}
		}
	}

	if !changed {
		settingsCopy := *uiSettings // Return a copy even if no changes
		return &settingsCopy, nil
	}

	// Save the updated settings using the internal helper that doesn't lock again
	filePath := filepath.Join(configsDir, uiSettingsFileName)
	if err := saveUISettingsToFile(filePath, uiSettings); err != nil {
		// Rollback in-memory change on save failure
		*uiSettings = originalSettings
		return nil, fmt.Errorf("settings updated in memory but failed to save (rolled back): %w", err)
	}

	settingsCopy := *uiSettings // Return a copy of the successfully saved state
	return &settingsCopy, nil
}

// LoadUISettings loads the UI settings from the JSON file into the package variable.
// Should be called once at startup.
func LoadUISettings() error {
	mu.Lock()
	defer mu.Unlock()

	if uiSettings != nil {
		return nil // Already loaded
	}

	var err error
	// Load settings using the helper, which handles defaults
	loadedSettings, err := loadUISettings()
	if err != nil {
		// If loading fails (e.g., read error, unmarshal error), return the error
		return fmt.Errorf("failed to load initial ui settings: %w", err)
	}
	// Assign the successfully loaded (or default) settings to the global variable
	uiSettings = loadedSettings
	return nil
}

// saveUISettingsToFile saves the settings to the specified file path.
// Assumes caller holds lock if necessary.
func saveUISettingsToFile(filePath string, settings *models.UISettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ui settings: %w", err)
	}
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write ui settings file '%s': %w", filePath, err)
	}
	return nil
}

// InitUISettings initializes the UI settings service.
func InitUISettings() error {
	mu.Lock()
	defer mu.Unlock()
	if uiSettings != nil {
		return nil // Already initialized
	}

	loaded, err := loadUISettings() // Call internal loader
	if err != nil {
		return err // Error already logged by loadUISettings
	}
	uiSettings = loaded // Assign the successfully loaded (or default) settings
	return nil
}
