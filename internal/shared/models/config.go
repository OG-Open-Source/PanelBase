package models

// Config represents the application configuration
type Config struct {
	Server  ServerConfig         `yaml:"server"`
	Auth    AuthConfig           `yaml:"auth"`
	Logging LoggingConfig        `yaml:"logging"`
	Plugins PluginsConfig        `yaml:"plugins"`
	Routes  SystemCommandsConfig `yaml:"routes"`
}

// ServerConfig represents server-related configuration
type ServerConfig struct {
	Host    string `yaml:"host"`
	Port    int    `yaml:"port"`
	Timeout int    `yaml:"timeout"`
	Mode    string `yaml:"mode"`
}

// AuthConfig represents authentication-related configuration
type AuthConfig struct {
	JWTExpiration int    `yaml:"jwt_expiration"`
	CookieName    string `yaml:"cookie_name"`
}

// LoggingConfig represents logging-related configuration
type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

// PluginsConfig represents plugin-related configuration
type PluginsConfig struct {
	Enabled   bool `yaml:"enabled"`
	AutoStart bool `yaml:"auto_start"`
}

// SystemCommandsConfig represents command-related configuration
type SystemCommandsConfig struct {
	Enabled bool                     `json:"enabled" yaml:"enabled"`
	Routes  map[string]CommandConfig `json:"routes" yaml:"routes"`
}

// CommandConfig represents a command configuration
type CommandConfig struct {
	Path        string            `json:"path" yaml:"path"`
	Method      string            `json:"method" yaml:"method"`
	Script      string            `json:"script" yaml:"script"`
	Description string            `json:"description" yaml:"description"`
	Params      map[string]string `json:"params" yaml:"params"`
}

// ThemesConfig represents themes configuration
type ThemesConfig struct {
	Themes       map[string]ThemeInfo `json:"themes"`
	CurrentTheme string               `json:"current_theme"`
}

// ThemeInfo represents a theme information
type ThemeInfo struct {
	Name        string                 `json:"name"`
	Authors     string                 `json:"authors"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	SourceLink  string                 `json:"source_link"`
	Directory   string                 `json:"directory"`
	Structure   map[string]interface{} `json:"structure"`
}

// GetCommandPath returns the script path for a command
func (c *SystemCommandsConfig) GetCommandPath(commandName string) string {
	if command, ok := c.Routes[commandName]; ok {
		return command.Script
	}
	return ""
}
