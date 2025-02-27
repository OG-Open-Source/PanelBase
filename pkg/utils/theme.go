package utils

import (
	"fmt"
)

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

// NewError 創建新的錯誤
func NewError(message string) error {
	return fmt.Errorf(message)
}

// NewErrorWithFormat 創建新的帶格式化的錯誤
func NewErrorWithFormat(format string, args ...interface{}) error {
	return fmt.Errorf(format, args...)
}

// NewErrorThemeAlreadyExists 創建主題已存在的錯誤
func NewErrorThemeAlreadyExists(theme string) error {
	return fmt.Errorf("Theme %s already exists", theme)
}

// NewErrorThemeNotFound 創建主題未找到的錯誤
func NewErrorThemeNotFound(theme string) error {
	return fmt.Errorf("Theme %s not found", theme)
}

// NewErrorFailedToDeleteThemeDirectory 創建刪除主題目錄失敗的錯誤
func NewErrorFailedToDeleteThemeDirectory(err error) error {
	return fmt.Errorf("Failed to delete theme directory: %v", err)
}

// NewErrorFailedToValidateThemeMetadata 創建驗證主題元數據失敗的錯誤
func NewErrorFailedToValidateThemeMetadata(err error) error {
	return fmt.Errorf("Failed to validate theme metadata: %v", err)
}
