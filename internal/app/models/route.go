package models

// RoutesConfig holds the routes configuration
type RoutesConfig struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Routes  map[string]string `json:"routes"`
}
