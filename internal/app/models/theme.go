package models

// Theme represents a UI theme
type Theme struct {
	Name        string                 `json:"name"`
	Authors     string                 `json:"authors"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	SourceLink  string                 `json:"source_link"`
	Directory   string                 `json:"directory"`
	Structure   map[string]interface{} `json:"structure,omitempty"`
}

// ThemesConfig holds the themes configuration
type ThemesConfig struct {
	CurrentTheme string            `json:"current_theme"`
	Themes       map[string]*Theme `json:"themes"`
}

// ThemeRequest 主题请求体
type ThemeRequest struct {
	URL string `json:"url" binding:"required,url"`
}

// ValidateTheme 验证主题数据完整性
func (t *Theme) ValidateTheme() bool {
	return t.Name != "" &&
		t.Authors != "" &&
		t.Version != "" &&
		t.Description != "" &&
		t.SourceLink != "" &&
		t.Directory != "" &&
		t.Structure != nil
}
