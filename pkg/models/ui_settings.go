package models

// UISettings defines the structure for global UI settings.
type UISettings struct {
	Title      string `json:"title"`                 // Website title displayed in the browser tab
	LogoURL    string `json:"logo_url,omitempty"`    // URL for the main logo displayed on the page
	FaviconURL string `json:"favicon_url,omitempty"` // URL for the favicon
	CustomCSS  string `json:"custom_css,omitempty"`  // Custom CSS rules to be injected
	CustomJS   string `json:"custom_js,omitempty"`   // Custom JavaScript code to be injected
}
