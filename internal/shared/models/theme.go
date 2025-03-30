package models

// Theme 主题模型
type Theme struct {
	Name        string                 `json:"name"`
	Authors     string                 `json:"authors"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	SourceLink  string                 `json:"source_link"`
	Directory   string                 `json:"directory"`
	Structure   map[string]interface{} `json:"structure,omitempty"`
}

// ValidateTheme 验证主题数据完整性
func (t *Theme) ValidateTheme() bool {
	return t.Name != "" && t.Directory != "" && t.SourceLink != ""
}

// ToThemeInfo 将Theme转换为ThemeInfo
func (t *Theme) ToThemeInfo() ThemeInfo {
	return ThemeInfo{
		Name:        t.Name,
		Authors:     t.Authors,
		Version:     t.Version,
		Description: t.Description,
		SourceLink:  t.SourceLink,
		Directory:   t.Directory,
		Structure:   t.Structure,
	}
}

// FromTheme 从Theme创建ThemeInfo
func FromTheme(theme Theme) ThemeInfo {
	return ThemeInfo{
		Name:        theme.Name,
		Authors:     theme.Authors,
		Version:     theme.Version,
		Description: theme.Description,
		SourceLink:  theme.SourceLink,
		Directory:   theme.Directory,
		Structure:   theme.Structure,
	}
}
