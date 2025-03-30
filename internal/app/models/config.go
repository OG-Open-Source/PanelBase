package models

// Config represents the application configuration
type Config struct {
	Server  ServerConfig  `yaml:"server"`
	Auth    AuthConfig    `yaml:"auth"`
	Logging LoggingConfig `yaml:"logging"`
	Plugins PluginsConfig `yaml:"plugins"`
	Routes  RoutesConfig  `yaml:"routes"`
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
