package models

// PluginEndpoint defines the plugin endpoint configuration
type PluginEndpoint struct {
	Path         string            `json:"path"`
	Methods      []string          `json:"methods"`
	InputFormat  map[string]string `json:"input_format"`
	OutputFormat map[string]string `json:"output_format"`
}

// PluginInfo plugin information
type PluginInfo struct {
	Name        string           `json:"name"`
	Authors     string           `json:"authors"`
	Version     string           `json:"version"`
	Description string           `json:"description"`
	SourceLink  string           `json:"source_link"`
	APIVersion  string           `json:"api_version"`
	Endpoints   []PluginEndpoint `json:"endpoints"`
	Directory   string           `json:"directory"`
}

// PluginsConfigJSON plugin JSON configuration
type PluginsConfigJSON struct {
	Plugins map[string]PluginInfo `json:"plugins"`
}

// GetPlugin gets plugin information by name
func (c *PluginsConfigJSON) GetPlugin(name string) *PluginInfo {
	if plugin, exists := c.Plugins[name]; exists {
		return &plugin
	}
	return nil
}

// GetEndpoint gets plugin endpoint by path
func (p *PluginInfo) GetEndpoint(path string) *PluginEndpoint {
	for i := range p.Endpoints {
		if p.Endpoints[i].Path == path {
			return &p.Endpoints[i]
		}
	}
	return nil
}

// ValidMethods defines the allowed HTTP methods
var ValidMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE"}

// ValidateMethods checks if all methods in the endpoint are valid
func (e *PluginEndpoint) ValidateMethods() bool {
	for _, method := range e.Methods {
		valid := false
		for _, validMethod := range ValidMethods {
			if method == validMethod {
				valid = true
				break
			}
		}
		if !valid {
			return false
		}
	}
	return true
}

// SupportsMethod checks if the endpoint supports the specified HTTP method
func (e *PluginEndpoint) SupportsMethod(method string) bool {
	for _, m := range e.Methods {
		if m == method {
			return true
		}
	}
	return false
}
