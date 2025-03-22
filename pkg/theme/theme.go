package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ThemeConfig represents the theme configuration
type ThemeConfig struct {
	CurrentTheme string                    `json:"current_theme"`
	Themes       map[string]ThemeStructure `json:"themes"`
}

// ThemeStructure represents the structure of a theme
type ThemeStructure struct {
	Name        string            `json:"name"`
	Authors     string            `json:"authors"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	SourceLink  string            `json:"source_link"`
	Directory   string            `json:"directory"`
	Structure   map[string]string `json:"structure"`
}

// LoadTheme loads theme configuration from file
// If the theme config file doesn't exist, it creates a default one
func LoadTheme(themePath string) (*ThemeConfig, error) {
	// Set default theme path if not provided
	if themePath == "" {
		themePath = filepath.Join("configs", "theme.json")
	}

	// Create the directory if it doesn't exist
	themeDir := filepath.Dir(themePath)
	if _, err := os.Stat(themeDir); os.IsNotExist(err) {
		if err := os.MkdirAll(themeDir, 0755); err != nil {
			return nil, fmt.Errorf("unable to create theme directory: %w", err)
		}
	}

	// Check if the theme file exists, if not create a default one
	if _, err := os.Stat(themePath); os.IsNotExist(err) {
		// Create default web theme directory
		defaultThemeDir := filepath.Join("web", "default")
		if err := os.MkdirAll(defaultThemeDir, 0755); err != nil {
			return nil, fmt.Errorf("unable to create default theme directory: %w", err)
		}

		// Default theme configuration
		defaultTheme := &ThemeConfig{
			CurrentTheme: "default_theme",
			Themes: map[string]ThemeStructure{
				"default_theme": {
					Name:        "Default Theme",
					Authors:     "PanelBase Team",
					Version:     "1.0.0",
					Description: "Default theme for PanelBase",
					SourceLink:  "https://github.com/OG-Open-Source/PanelBase",
					Directory:   "default",
					Structure: map[string]string{
						"index.html": "web/default/index.html",
						"style.css":  "web/default/style.css",
						"script.js":  "web/default/script.js",
					},
				},
			},
		}

		// Marshal the default theme to JSON
		defaultThemeJSON, err := json.MarshalIndent(defaultTheme, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("error marshaling default theme: %w", err)
		}

		// Write the default theme to file
		if err := os.WriteFile(themePath, defaultThemeJSON, 0644); err != nil {
			return nil, fmt.Errorf("unable to create default theme file: %w", err)
		}

		// Create default theme files
		defaultHTML := `<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>PanelBase - Default Theme</title>
	<link rel="stylesheet" href="style.css">
</head>
<body>
	<div class="container">
		<header>
			<h1>PanelBase</h1>
			<p>Welcome to PanelBase Default Theme</p>
		</header>

		<main>
			<section class="info-section">
				<h2>Theme Information</h2>
				<div id="theme-info">
					<p>Loading theme information...</p>
				</div>
			</section>
		</main>

		<footer>
			<p>&copy; 2023-2025 PanelBase. All rights reserved.</p>
		</footer>
	</div>

	<script src="script.js"></script>
</body>
</html>`

		defaultCSS := `/* Base styles */
* {
	margin: 0;
	padding: 0;
	box-sizing: border-box;
}

body {
	font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
	line-height: 1.6;
	color: #333;
	background-color: #f5f5f5;
}

.container {
	max-width: 1200px;
	margin: 0 auto;
	padding: 20px;
}

/* Header styles */
header {
	background-color: #3498db;
	color: white;
	padding: 20px;
	text-align: center;
	border-radius: 5px;
	margin-bottom: 20px;
}

header h1 {
	font-size: 2.5rem;
	margin-bottom: 10px;
}

/* Main content styles */
main {
	background-color: #fff;
	padding: 20px;
	border-radius: 5px;
	box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
}

.info-section {
	margin-bottom: 20px;
}

.info-section h2 {
	border-bottom: 2px solid #3498db;
	padding-bottom: 10px;
	margin-bottom: 15px;
}

#theme-info {
	background-color: #f9f9f9;
	padding: 15px;
	border-radius: 5px;
	border-left: 4px solid #3498db;
}

/* Footer styles */
footer {
	text-align: center;
	padding: 20px;
	margin-top: 20px;
	color: #666;
	font-size: 0.9rem;
}

/* Responsive design */
@media (max-width: 768px) {
	.container {
		padding: 10px;
	}

	header h1 {
		font-size: 2rem;
	}
}`

		defaultJS := `document.addEventListener('DOMContentLoaded', () => {
	// Fetch theme information from the API
	fetchThemeInfo();
});

// Function to fetch theme information
async function fetchThemeInfo() {
	const themeInfoElement = document.getElementById('theme-info');

	try {
		// Get current path to build the correct API endpoint
		const path = window.location.pathname;
		const entryPath = path.split('/')[1]; // Get the entry path from URL

		// Fetch theme info from API
		const response = await fetch("/" + entryPath + "/theme/info");

		if (!response.ok) {
			throw new Error("HTTP error! Status: " + response.status);
		}

		const data = await response.json();

		// Create HTML with theme information
		const themeInfoHTML =
			'<div class="theme-info-content">' +
				'<p><strong>Name:</strong> ' + data.name + '</p>' +
				'<p><strong>Version:</strong> ' + data.version + '</p>' +
				'<p><strong>Authors:</strong> ' + data.authors + '</p>' +
				'<p><strong>Description:</strong> ' + data.description + '</p>' +
				'<p><strong>Source:</strong> <a href="' + data.source_link + '" target="_blank">GitHub Repository</a></p>' +
			'</div>';

		// Update the theme info element
		themeInfoElement.innerHTML = themeInfoHTML;

	} catch (error) {
		console.error('Error fetching theme information:', error);
		themeInfoElement.innerHTML = '<p>Error loading theme information: ' + error.message + '</p>';
	}
}`

		// Write default theme files
		if err := os.WriteFile(filepath.Join(defaultThemeDir, "index.html"), []byte(defaultHTML), 0644); err != nil {
			return nil, fmt.Errorf("unable to create default index.html: %w", err)
		}
		if err := os.WriteFile(filepath.Join(defaultThemeDir, "style.css"), []byte(defaultCSS), 0644); err != nil {
			return nil, fmt.Errorf("unable to create default style.css: %w", err)
		}
		if err := os.WriteFile(filepath.Join(defaultThemeDir, "script.js"), []byte(defaultJS), 0644); err != nil {
			return nil, fmt.Errorf("unable to create default script.js: %w", err)
		}
	}

	// Read the theme configuration file
	bytes, err := os.ReadFile(themePath)
	if err != nil {
		return nil, fmt.Errorf("error reading theme config file: %w", err)
	}

	// Parse the JSON into ThemeConfig struct
	var themeConfig ThemeConfig
	if err := json.Unmarshal(bytes, &themeConfig); err != nil {
		return nil, fmt.Errorf("error parsing theme config: %w", err)
	}

	return &themeConfig, nil
}

// GetCurrentTheme returns the current theme structure
func (t *ThemeConfig) GetCurrentTheme() (*ThemeStructure, error) {
	if theme, ok := t.Themes[t.CurrentTheme]; ok {
		return &theme, nil
	}
	return nil, fmt.Errorf("current theme '%s' not found in theme configuration", t.CurrentTheme)
}

// GetThemeDirectory returns the web directory path for the current theme
func (t *ThemeConfig) GetThemeDirectory() (string, error) {
	currentTheme, err := t.GetCurrentTheme()
	if err != nil {
		return "", err
	}

	return filepath.Join("web", currentTheme.Directory), nil
}
