package ui_settings

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/OG-Open-Source/PanelBase/internal/models"
)

const uiSettingsFilePath = "configs/ui_settings.json"

var fileMutex sync.RWMutex

// loadUISettings reads and parses the ui_settings.json file.
func loadUISettings() (*models.UISettings, error) {
	fileMutex.RLock()
	defer fileMutex.RUnlock()

	data, err := os.ReadFile(uiSettingsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default settings if file doesn't exist
			return &models.UISettings{Title: "PanelBase"}, nil
		}
		return nil, fmt.Errorf("failed to read ui settings file '%s': %w", uiSettingsFilePath, err)
	}

	if len(data) == 0 {
		// Return default settings if file is empty
		return &models.UISettings{Title: "PanelBase"}, nil
	}

	var settings models.UISettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal ui settings from '%s': %w", uiSettingsFilePath, err)
	}

	// Ensure default title if empty after loading
	if settings.Title == "" {
		settings.Title = "PanelBase"
	}

	return &settings, nil
}

// saveUISettings writes the UI settings back to the file.
func saveUISettings(settings *models.UISettings) error {
	fileMutex.Lock()
	defer fileMutex.Unlock()

	jsonData, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal ui settings: %w", err)
	}
	if err := os.WriteFile(uiSettingsFilePath, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write ui settings file '%s': %w", uiSettingsFilePath, err)
	}
	return nil
}

// GetUISettings retrieves the current UI settings.
func GetUISettings() (*models.UISettings, error) {
	return loadUISettings()
}

// UpdateUISettings updates the stored UI settings.
// It performs a partial update: only fields present in the input `updates` are changed.
func UpdateUISettings(updates models.UISettings) (*models.UISettings, error) {
	currentSettings, err := loadUISettings()
	if err != nil {
		return nil, fmt.Errorf("failed to load current ui settings for update: %w", err)
	}

	// Apply updates - only update fields if they are provided in the input
	// Note: This simple approach means empty strings in `updates` will overwrite existing values.
	// A more robust approach might use pointers in the `updates` struct or check field presence.
	if updates.Title != "" { // Allow setting title to empty? Maybe not.
		currentSettings.Title = updates.Title
	}
	// For URLs, CSS, JS, allow setting to empty string to clear them.
	currentSettings.LogoURL = updates.LogoURL
	currentSettings.FaviconURL = updates.FaviconURL
	currentSettings.CustomCSS = updates.CustomCSS
	currentSettings.CustomJS = updates.CustomJS

	if err := saveUISettings(currentSettings); err != nil {
		return nil, fmt.Errorf("failed to save updated ui settings: %w", err)
	}
	return currentSettings, nil
}

// EnsureUISettingsFile ensures the settings file exists, creating it with defaults if not.
// This is intended to be called by the bootstrap process.
func EnsureUISettingsFile() error {
	fileMutex.Lock() // Use write lock for check-and-create
	defer fileMutex.Unlock()

	_, err := os.Stat(uiSettingsFilePath)
	if os.IsNotExist(err) {
		fmt.Printf("File '%s' not found, creating with defaults...\n", uiSettingsFilePath)
		defaultSettings := &models.UISettings{
			Title: "PanelBase", // Default title
			// Other fields default to empty strings
		}
		jsonData, jsonErr := json.MarshalIndent(defaultSettings, "", "  ")
		if jsonErr != nil {
			return fmt.Errorf("failed to marshal default ui settings: %w", jsonErr)
		}
		if writeErr := os.WriteFile(uiSettingsFilePath, jsonData, 0644); writeErr != nil {
			return fmt.Errorf("failed to write default ui settings file '%s': %w", uiSettingsFilePath, writeErr)
		}
		fmt.Printf("Successfully created default file: %s\n", uiSettingsFilePath)
	} else if err != nil {
		return fmt.Errorf("failed to check ui settings file '%s': %w", uiSettingsFilePath, err)
	}
	// File exists, do nothing
	return nil
}
