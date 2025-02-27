package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// MetadataManager 處理元數據管理
type MetadataManager struct {
	routesConfigPath string // routes.json 的路徑
	themesConfigPath string // themes.json 的路徑
	scriptsPath     string // scripts 目錄的路徑
	themesPath      string // themes 目錄的路徑
}

// RouteConfig 路由配置結構
type RouteConfig map[string]string // 路由名稱 -> 腳本名稱

// ThemeConfig 主題配置結構
type ThemeConfig map[string]ThemeInfo // 主題名稱 -> 主題信息

// ThemeInfo 主題信息結構
type ThemeInfo struct {
	Name        string `json:"name"`        // 主題顯示名稱
	Authors     string `json:"authors"`     // 作者
	Version     string `json:"version"`     // 版本
	Description string `json:"description"` // 描述
	File        string `json:"file"`       // 主題文件連結
}

// RouteMetadata 路由元數據結構
type RouteMetadata struct {
	Script       string   `json:"script"`
	PkgManagers  []string `json:"pkg_managers"`
	Dependencies []string `json:"dependencies"`
	Authors      string   `json:"authors"`
	Version      string   `json:"version"`
	Description  string   `json:"description"`
}

// NewMetadataManager 創建新的元數據管理器
func NewMetadataManager(routesConfigPath, themesConfigPath, scriptsPath, themesPath string) *MetadataManager {
	return &MetadataManager{
		routesConfigPath: routesConfigPath,
		themesConfigPath: themesConfigPath,
		scriptsPath:     scriptsPath,
		themesPath:      themesPath,
	}
}

// LoadRouteConfig 載入路由配置
func (mm *MetadataManager) LoadRouteConfig() (RouteConfig, error) {
	data, err := ioutil.ReadFile(mm.routesConfigPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read routes config: %v", err)
	}

	var config RouteConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Failed to parse routes config: %v", err)
	}

	return config, nil
}

// SaveRouteConfig 保存路由配置
func (mm *MetadataManager) SaveRouteConfig(config RouteConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal routes config: %v", err)
	}

	if err := ioutil.WriteFile(mm.routesConfigPath, data, 0644); err != nil {
		return fmt.Errorf("Failed to write routes config: %v", err)
	}

	return nil
}

// LoadThemeConfig 載入主題配置
func (mm *MetadataManager) LoadThemeConfig() (ThemeConfig, error) {
	data, err := ioutil.ReadFile(mm.themesConfigPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read themes config: %v", err)
	}

	var config ThemeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Failed to parse themes config: %v", err)
	}

	return config, nil
}

// SaveThemeConfig 保存主題配置
func (mm *MetadataManager) SaveThemeConfig(config ThemeConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal themes config: %v", err)
	}

	if err := ioutil.WriteFile(mm.themesConfigPath, data, 0644); err != nil {
		return fmt.Errorf("Failed to write themes config: %v", err)
	}

	return nil
}

// ValidateRouteMetadata 驗證路由元數據
func (mm *MetadataManager) ValidateRouteMetadata(metadata *RouteMetadata) error {
	if metadata.Script == "" {
		return fmt.Errorf("missing script field")
	}
	if len(metadata.PkgManagers) == 0 {
		return fmt.Errorf("missing pkg_managers field")
	}
	if len(metadata.Dependencies) == 0 {
		return fmt.Errorf("missing dependencies field")
	}
	if metadata.Authors == "" {
		return fmt.Errorf("missing authors field")
	}
	if metadata.Version == "" {
		return fmt.Errorf("missing version field")
	}
	if metadata.Description == "" {
		return fmt.Errorf("missing description field")
	}
	return nil
}

// ValidateThemeInfo 驗證主題信息
func (mm *MetadataManager) ValidateThemeInfo(theme string, info *ThemeInfo) error {
	if !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(theme) {
		return fmt.Errorf("invalid theme name format")
	}
	if info.Name == "" {
		return fmt.Errorf("missing name field")
	}
	if info.Authors == "" {
		return fmt.Errorf("missing authors field")
	}
	if info.Version == "" {
		return fmt.Errorf("missing version field")
	}
	if info.Description == "" {
		return fmt.Errorf("missing description field")
	}
	if info.File == "" {
		return fmt.Errorf("missing file field")
	}
	return nil
}

// FetchRouteMetadata 從文件或URL獲取路由元數據
func (mm *MetadataManager) FetchRouteMetadata(urlOrPath string, isURL bool) (*RouteMetadata, error) {
	var reader io.Reader

	if isURL {
		resp, err := http.Get(urlOrPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to fetch URL: %v", err)
		}
		defer resp.Body.Close()
		reader = resp.Body
	} else {
		file, err := os.Open(filepath.Join(mm.scriptsPath, urlOrPath))
		if err != nil {
			return nil, fmt.Errorf("Failed to open script file: %v", err)
		}
		defer file.Close()
		reader = file
	}

	metadata, err := parseRouteMetadata(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse route metadata: %v", err)
	}

	return metadata, nil
}

// parseRouteMetadata 解析路由元數據
func parseRouteMetadata(reader io.Reader) (*RouteMetadata, error) {
	scanner := bufio.NewScanner(reader)
	result := &RouteMetadata{}
	foundMetadata := make(map[string]bool)
	lineCount := 0

	commentPattern := regexp.MustCompile(`^[#/]+\s*@(\w+):\s*(.+)$`)

	for scanner.Scan() && lineCount < 10 {
		line := scanner.Text()
		lineCount++

		if matches := commentPattern.FindStringSubmatch(line); len(matches) == 3 {
			key := matches[1]
			value := strings.TrimSpace(matches[2])

			switch key {
			case "script":
				result.Script = value
				foundMetadata["script"] = true
			case "pkg_managers":
				result.PkgManagers = strings.Split(value, ",")
				foundMetadata["pkg_managers"] = true
			case "dependencies":
				result.Dependencies = strings.Split(value, ",")
				foundMetadata["dependencies"] = true
			case "authors":
				result.Authors = value
				foundMetadata["authors"] = true
			case "version":
				result.Version = value
				foundMetadata["version"] = true
			case "description":
				result.Description = value
				foundMetadata["description"] = true
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Failed to read metadata: %v", err)
	}

	requiredMetadata := []string{"script", "pkg_managers", "dependencies", "authors", "version", "description"}
	for _, key := range requiredMetadata {
		if !foundMetadata[key] {
			return nil, fmt.Errorf("Required metadata @%s not found in first 10 lines", key)
		}
	}

	return result, nil
}

// FetchThemeMetadata 從文件或URL獲取主題元數據
func (mm *MetadataManager) FetchThemeMetadata(urlOrPath string, isURL bool) (*ThemeInfo, error) {
	var reader io.Reader

	if isURL {
		resp, err := http.Get(urlOrPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to fetch URL: %v", err)
		}
		defer resp.Body.Close()
		reader = resp.Body
	} else {
		file, err := os.Open(filepath.Join(mm.themesPath, urlOrPath, "theme.json"))
		if err != nil {
			return nil, fmt.Errorf("Failed to open theme file: %v", err)
		}
		defer file.Close()
		reader = file
	}

	var themeInfo ThemeInfo
	if err := json.NewDecoder(reader).Decode(&themeInfo); err != nil {
		return nil, fmt.Errorf("Failed to parse theme metadata: %v", err)
	}

	return &themeInfo, nil
}

// InstallRoute 安裝路由
func (mm *MetadataManager) InstallRoute(url string) error {
	// 獲取並驗證元數據
	metadata, err := mm.FetchRouteMetadata(url, true)
	if err != nil {
		return err
	}

	// 載入當前配置
	config, err := mm.LoadRouteConfig()
	if err != nil {
		return err
	}

	// 檢查路由是否已存在
	if _, exists := config[metadata.Script]; exists {
		return fmt.Errorf("Route %s already exists", metadata.Script)
	}

	// TODO: 下載腳本到 scripts 目錄

	// 更新配置
	config[metadata.Script] = filepath.Base(url)
	return mm.SaveRouteConfig(config)
}

// InstallTheme 安裝主題
func (mm *MetadataManager) InstallTheme(url string) error {
	// 獲取並驗證元數據
	themeInfo, err := mm.FetchThemeMetadata(url, true)
	if err != nil {
		return err
	}

	// 載入當前配置
	config, err := mm.LoadThemeConfig()
	if err != nil {
		return err
	}

	// 檢查主題是否已存在
	if _, exists := config[themeInfo.Name]; exists {
		return fmt.Errorf("Theme %s already exists", themeInfo.Name)
	}

	// TODO: 下載主題文件到 themes 目錄

	// 更新配置
	config[themeInfo.Name] = *themeInfo
	return mm.SaveThemeConfig(config)
}

// DeleteRoute 刪除路由
func (mm *MetadataManager) DeleteRoute(name string) error {
	// 載入當前配置
	config, err := mm.LoadRouteConfig()
	if err != nil {
		return err
	}

	// 檢查路由是否存在
	scriptName, exists := config[name]
	if !exists {
		return fmt.Errorf("Route %s not found", name)
	}

	// 刪除腳本文件
	scriptPath := filepath.Join(mm.scriptsPath, scriptName)
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to delete script file: %v", err)
	}

	// 從配置中移除
	delete(config, name)
	return mm.SaveRouteConfig(config)
}

// DeleteTheme 刪除主題
func (mm *MetadataManager) DeleteTheme(name string) error {
	// 載入當前配置
	config, err := mm.LoadThemeConfig()
	if err != nil {
		return err
	}

	// 檢查主題是否存在
	if _, exists := config[name]; !exists {
		return fmt.Errorf("Theme %s not found", name)
	}

	// 刪除主題目錄
	themePath := filepath.Join(mm.themesPath, name)
	if err := os.RemoveAll(themePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to delete theme directory: %v", err)
	}

	// 從配置中移除
	delete(config, name)
	return mm.SaveThemeConfig(config)
}