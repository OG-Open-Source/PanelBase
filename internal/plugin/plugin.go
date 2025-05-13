package plugin

// Plugin defines the interface that all plugins must implement.
type Plugin interface {
	// ID returns the unique identifier for the plugin.
	ID() string

	// Name returns the human-readable name of the plugin.
	Name() string

	// Initialize performs any setup required by the plugin.
	// It might receive configuration or access to shared resources.
	Initialize(config map[string]interface{}) error

	// Execute performs the main action of the plugin.
	// The specific parameters and return value will depend on the plugin's purpose.
	Execute(args ...interface{}) (interface{}, error)

	// TODO: Add other common methods if needed, e.g., Version(), Shutdown()
}
