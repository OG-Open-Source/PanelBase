package utils

// ThemeManager 處理主題相關操作
type ThemeManager struct {
	metadata *MetadataManager
}

// ThemeRequest 主題請求結構
type ThemeRequest struct {
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

// NewThemeManager 創建新的主題管理器
func NewThemeManager(themesConfigPath, themesPath string) *ThemeManager {
	return &ThemeManager{
		metadata: NewMetadataManager("", themesConfigPath, "", themesPath),
	}
}

// Install 安裝新主題
func (tm *ThemeManager) Install(req ThemeRequest) error {
	return tm.metadata.InstallTheme(req.URL)
}

// Update 更新主題
func (tm *ThemeManager) Update(req ThemeRequest) error {
	return tm.metadata.InstallTheme(req.URL) // 使用相同的安裝邏輯
}

// Delete 刪除主題
func (tm *ThemeManager) Delete(req ThemeRequest) error {
	return tm.metadata.DeleteTheme(req.Name)
}
