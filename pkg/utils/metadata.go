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

// 錯誤定義
type MetadataError struct {
	Type    string
	Message string
	Err     error
}

func (e *MetadataError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Type, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// 錯誤類型常量
const (
	ErrFailedToDownloadTheme      = "FailedToDownloadTheme"
	ErrFailedToParseThemeMetadata = "FailedToParseThemeMetadata"
	ErrInvalidThemeMetadata       = "InvalidThemeMetadata"
	ErrFailedToCreateThemeDir     = "FailedToCreateThemeDirectory"
)

// 錯誤建構函數
func NewErrorFailedToDownloadTheme(err error) error {
	return &MetadataError{
		Type:    ErrFailedToDownloadTheme,
		Message: "Failed to download theme",
		Err:     err,
	}
}

func NewErrorFailedToParseThemeMetadata(err error) error {
	return &MetadataError{
		Type:    ErrFailedToParseThemeMetadata,
		Message: "Failed to parse theme metadata",
		Err:     err,
	}
}

func NewErrorInvalidThemeMetadata(msg string) error {
	return &MetadataError{
		Type:    ErrInvalidThemeMetadata,
		Message: msg,
	}
}

func NewErrorFailedToCreateThemeDirectory(err error) error {
	return &MetadataError{
		Type:    ErrFailedToCreateThemeDir,
		Message: "Failed to create theme directory",
		Err:     err,
	}
}

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
type ThemeConfig map[string]ThemeStructure

// ThemeStructure 表示主題結構
type ThemeStructure struct {
	Name        string                 `json:"name"`
	Authors     string                 `json:"authors"`
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	Structure   map[string]interface{} `json:"structure"`
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
		return nil, fmt.Errorf("Failed to read routes config: '%v'", err)
	}

	var config RouteConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Failed to parse routes config: '%v'", err)
	}

	return config, nil
}

// SaveRouteConfig 保存路由配置
func (mm *MetadataManager) SaveRouteConfig(config RouteConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal routes config: '%v'", err)
	}

	if err := ioutil.WriteFile(mm.routesConfigPath, data, 0644); err != nil {
		return fmt.Errorf("Failed to write routes config: '%v'", err)
	}

	return nil
}

// LoadThemeConfig 載入主題配置
func (mm *MetadataManager) LoadThemeConfig() (ThemeConfig, error) {
	data, err := os.ReadFile(mm.themesConfigPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read themes config: '%v'", err)
	}

	var config ThemeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("Failed to parse themes config: '%v'", err)
	}

	return config, nil
}

// SaveThemeConfig 保存主題配置
func (mm *MetadataManager) SaveThemeConfig(config ThemeConfig) error {
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to marshal themes config: '%v'", err)
	}

	if err := os.WriteFile(mm.themesConfigPath, data, 0644); err != nil {
		return fmt.Errorf("Failed to write themes config: '%v'", err)
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

// FetchRouteMetadata 從文件或URL獲取路由元數據
func (mm *MetadataManager) FetchRouteMetadata(urlOrPath string, isURL bool) (*RouteMetadata, error) {
	if isURL {
		Debug("Fetching route metadata from URL: '%s'", urlOrPath)
	} else {
		Debug("Fetching route metadata from file: '%s'", urlOrPath)
	}
	var reader io.Reader

	if isURL {
		resp, err := http.Get(urlOrPath)
		if err != nil {
			return nil, fmt.Errorf("Failed to fetch URL: '%v'", err)
		}
		defer resp.Body.Close()
		reader = resp.Body
	} else {
		file, err := os.Open(filepath.Join(mm.scriptsPath, urlOrPath))
		if err != nil {
			return nil, fmt.Errorf("Failed to open script file: '%v'", err)
		}
		defer file.Close()
		reader = file
	}

	metadata, err := parseRouteMetadata(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to read metadata: '%v'", err)
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
		return nil, fmt.Errorf("Failed to read metadata: '%v'", err)
	}

	requiredMetadata := []string{"script", "pkg_managers", "dependencies", "authors", "version", "description"}
	for _, key := range requiredMetadata {
		if !foundMetadata[key] {
			return nil, fmt.Errorf("Required metadata @%s not found in first 10 lines", key)
		}
	}

	return result, nil
}

// FetchThemeMetadata 獲取主題元數據
func (mm *MetadataManager) FetchThemeMetadata(url string, validate bool) (*ThemeStructure, error) {
	// 下載主題配置
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Failed to download theme metadata: '%v'", err)
	}
	defer resp.Body.Close()

	// 解析主題配置
	var themeConfig map[string]ThemeStructure
	if err := json.NewDecoder(resp.Body).Decode(&themeConfig); err != nil {
		return nil, fmt.Errorf("Failed to parse theme metadata: '%v'", err)
	}

	// 確保只有一個主題
	if len(themeConfig) != 1 {
		return nil, fmt.Errorf("Theme config must contain exactly one theme")
	}

	// 獲取主題結構
	var themeStructure ThemeStructure
	for _, structure := range themeConfig {
		themeStructure = structure
		break
	}

	// 驗證主題結構
	if validate {
		if themeStructure.Name == "" {
			return nil, fmt.Errorf("missing name field")
		}
		if themeStructure.Authors == "" {
			return nil, fmt.Errorf("missing authors field")
		}
		if themeStructure.Version == "" {
			return nil, fmt.Errorf("missing version field")
		}
		if themeStructure.Description == "" {
			return nil, fmt.Errorf("missing description field")
		}
		if themeStructure.Structure == nil {
			return nil, fmt.Errorf("missing structure field")
		}
	}

	return &themeStructure, nil
}

// ValidateThemeInfo 驗證主題信息
func (mm *MetadataManager) ValidateThemeInfo(theme string, structure *ThemeStructure) error {
	if !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(theme) {
		return fmt.Errorf("Invalid theme name format: '%s'", theme)
	}
	if structure.Name == "" {
		return fmt.Errorf("missing name field")
	}
	if structure.Authors == "" {
		return fmt.Errorf("missing authors field")
	}
	if structure.Version == "" {
		return fmt.Errorf("missing version field")
	}
	if structure.Description == "" {
		return fmt.Errorf("missing description field")
	}
	if structure.Structure == nil {
		return fmt.Errorf("missing structure field")
	}
	return nil
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
		return fmt.Errorf("Route [%s] already exists", metadata.Script)
	}

	// TODO: 下載腳本到 scripts 目錄

	// 更新配置
	config[metadata.Script] = filepath.Base(url)
	return mm.SaveRouteConfig(config)
}

// InstallTheme 安裝主題
func (mm *MetadataManager) InstallTheme(themeURL string) error {
	// 下載主題配置
	resp, err := http.Get(themeURL)
	if err != nil {
		return NewErrorFailedToDownloadTheme(err)
	}
	defer resp.Body.Close()

	// 解析主題配置
	var themeConfig map[string]ThemeStructure
	if err := json.NewDecoder(resp.Body).Decode(&themeConfig); err != nil {
		return NewErrorFailedToParseThemeMetadata(err)
	}

	// 確保只有一個主題
	if len(themeConfig) != 1 {
		return NewErrorInvalidThemeMetadata("Theme config must contain exactly one theme")
	}

	// 獲取主題名稱和結構
	var themeName string
	var themeStructure ThemeStructure
	for name, structure := range themeConfig {
		themeName = name
		themeStructure = structure
		break
	}

	// 創建主題目錄
	themeDir := filepath.Join("internal", "themes", themeName)
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		return NewErrorFailedToCreateThemeDirectory(err)
	}

	// 下載並安裝文件
	if err := downloadThemeFiles(themeDir, "", themeStructure.Structure); err != nil {
		return err
	}

	// 更新主題配置
	return mm.SaveThemeConfig(themeConfig)
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
		return fmt.Errorf("Route [%s] not found", name)
	}

	// 刪除腳本文件
	scriptPath := filepath.Join(mm.scriptsPath, scriptName)
	if err := os.Remove(scriptPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to delete script file: '%v'", err)
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
		return fmt.Errorf("Theme [%s] not found", name)
	}

	// 刪除主題目錄
	themePath := filepath.Join(mm.themesPath, name)
	if err := os.RemoveAll(themePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Failed to delete theme directory: '%v'", err)
	}

	// 從配置中移除
	delete(config, name)
	return mm.SaveThemeConfig(config)
}

// ValidateThemeStructure 驗證主題結構
func (mm *MetadataManager) ValidateThemeStructure(themeName string, structure ThemeStructure) error {
	if !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(themeName) {
		return fmt.Errorf("Invalid theme name format: '%s'", themeName)
	}
	if structure.Name == "" {
		return fmt.Errorf("missing name field")
	}
	if structure.Authors == "" {
		return fmt.Errorf("missing authors field")
	}
	if structure.Version == "" {
		return fmt.Errorf("missing version field")
	}
	if structure.Description == "" {
		return fmt.Errorf("missing description field")
	}
	if structure.Structure == nil {
		return fmt.Errorf("missing structure field")
	}

	// 獲取主題目錄
	themeDir := filepath.Join("internal", "themes", themeName)
	
	// 遍歷結構並驗證
	return validateStructure(themeDir, "", structure.Structure)
}

// validateStructure 遞歸驗證結構
func validateStructure(baseDir string, currentPath string, structure interface{}) error {
	switch v := structure.(type) {
	case map[string]interface{}:
		// 處理目錄
		for key, value := range v {
			newPath := filepath.Join(currentPath, key)
			if err := validateStructure(baseDir, newPath, value); err != nil {
				return err
			}
		}
	case string:
		// 處理文件
		fullPath := filepath.Join(baseDir, currentPath)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return fmt.Errorf("Missing required file: %s", fullPath)
		}
	default:
		return fmt.Errorf("Invalid structure type for path %s", currentPath)
	}
	return nil
}

// downloadThemeFiles 遞歸下載主題文件
func downloadThemeFiles(baseDir string, currentPath string, structure interface{}) error {
	switch v := structure.(type) {
	case map[string]interface{}:
		// 處理目錄
		for key, value := range v {
			newPath := filepath.Join(currentPath, key)
			dirPath := filepath.Join(baseDir, currentPath, key)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return fmt.Errorf("Failed to create directory %s: %v", dirPath, err)
			}
			if err := downloadThemeFiles(baseDir, newPath, value); err != nil {
				return err
			}
		}
	case string:
		// 處理文件
		fileURL := v
		filePath := filepath.Join(baseDir, currentPath)
		if err := downloadFile(fileURL, filePath); err != nil {
			return fmt.Errorf("Failed to download file %s: %v", filePath, err)
		}
	}
	return nil
}

// downloadFile 下載單個文件
func downloadFile(url string, filepath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// UpdateTheme 更新主題
func (mm *MetadataManager) UpdateTheme(name string, structure *ThemeStructure) error {
	// 載入當前配置
	config, err := mm.LoadThemeConfig()
	if err != nil {
		return err
	}

	// 檢查主題是否存在
	if _, exists := config[name]; !exists {
		return fmt.Errorf("Theme [%s] not found", name)
	}

	// 更新配置
	config[name] = *structure
	return mm.SaveThemeConfig(config)
}