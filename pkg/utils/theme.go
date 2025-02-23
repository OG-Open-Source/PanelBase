package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
)

type Theme struct {
	Name       string            `json:"name"`
	Version    string            `json:"version"`
	Colors     map[string]string `json:"colors"`
	Components map[string]string `json:"components"`
}

var currentTheme *Theme

func (t *Theme) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("theme name is required")
	}
	if t.Version == "" {
		return fmt.Errorf("theme version is required")
	}
	if len(t.Colors) == 0 {
		return fmt.Errorf("theme must define at least one color")
	}
	if len(t.Components) == 0 {
		return fmt.Errorf("theme must define at least one component")
	}
	return nil
}

func LoadTheme(themeName string) error {
	themePath := filepath.Join("themes", themeName+".json")
	data, err := ioutil.ReadFile(themePath)
	if err != nil {
		Log(EROR, "Failed to read theme file: %v", err)
		return err
	}

	theme := &Theme{}
	if err := json.Unmarshal(data, theme); err != nil {
		Log(EROR, "Failed to parse theme file: %v", err)
		return err
	}

	if err := theme.Validate(); err != nil {
		Log(EROR, "Invalid theme configuration: %v", err)
		return err
	}

	currentTheme = theme
	Log(INFO, "Theme loaded: %s (v%s)", theme.Name, theme.Version)
	return nil
}

func GetTheme() *Theme {
	if currentTheme == nil {
		if err := LoadTheme("default"); err != nil {
			Log(EROR, "Failed to load default theme: %v", err)
			return nil
		}
	}
	return currentTheme
}

// TODO: Implement theme system