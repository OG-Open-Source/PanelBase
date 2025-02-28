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
	Theme       string                 `json:"theme"`       // 主題目錄名
	Name        string                 `json:"name"`        // 主題顯示名稱
	Authors     string                 `json:"authors"`     // 作者
	Version     string                 `json:"version"`     // 版本
	Description string                 `json:"description"` // 描述
	Structure   map[string]interface{} `json:"structure"`   // 文件結構
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
	// 獲取當前工作目錄
	cwd, err := os.Getwd()
	Debug("Current working directory: '%s'", cwd)
	Debug("Loading theme config from: '%s'", mm.themesConfigPath)

	// 檢查配置路徑
	if mm.themesConfigPath == "" {
		return nil, fmt.Errorf("Theme config path is empty")
	}

	// 檢查文件是否存在
	fileInfo, err := os.Stat(mm.themesConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			Debug("Theme config file does not exist at: '%s'", mm.themesConfigPath)
			// 如果文件不存在，創建一個空的配置
			emptyConfig := make(ThemeConfig)
			if err := mm.SaveThemeConfig(emptyConfig); err != nil {
				return nil, fmt.Errorf("Failed to create empty theme config at '%s': %v", mm.themesConfigPath, err)
			}
			return emptyConfig, nil
		}
		return nil, fmt.Errorf("Failed to stat theme config file: '%v'", err)
	}

	Debug("Theme config file size: %d bytes", fileInfo.Size())

	// 讀取配置文件
	data, err := os.ReadFile(mm.themesConfigPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to read themes config from '%s': %v", mm.themesConfigPath, err)
	}

	Debug("Read %d bytes from theme config file", len(data))
	Debug("File content: '%s'", string(data))

	// 首先解析為通用的 map 結構
	var rawConfig map[string]map[string]interface{}
	if err := json.Unmarshal(data, &rawConfig); err != nil {
		Debug("Failed to parse JSON: '%v'", err)
		return nil, fmt.Errorf("Failed to parse themes config from '%s': %v", mm.themesConfigPath, err)
	}

	// 轉換為 ThemeConfig
	config := make(ThemeConfig)
	for themeName, rawTheme := range rawConfig {
		// 只處理有效的主題配置
		if rawTheme["name"] != nil && rawTheme["authors"] != nil && 
		   rawTheme["version"] != nil && rawTheme["description"] != nil {
			theme := ThemeStructure{
				Name:        rawTheme["name"].(string),
				Authors:     rawTheme["authors"].(string),
				Version:     rawTheme["version"].(string),
				Description: rawTheme["description"].(string),
			}
			
			// 處理 structure 字段
			if structure, ok := rawTheme["structure"].(map[string]interface{}); ok {
				theme.Structure = structure
				config[themeName] = theme
			}
		}
	}

	Debug("Successfully loaded theme config with %d themes", len(config))
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
	Debug("Fetching theme metadata from URL: '%s'", url)
	
	// 下載主題配置
	resp, err := http.Get(url)
	if err != nil {
		return nil, NewErrorFailedToDownloadTheme(err)
	}
	defer resp.Body.Close()

	// 解析主題配置
	var themeConfig map[string]ThemeStructure
	if err := json.NewDecoder(resp.Body).Decode(&themeConfig); err != nil {
		return nil, NewErrorFailedToParseThemeMetadata(err)
	}

	// 確保只有一個主題
	if len(themeConfig) != 1 {
		return nil, NewErrorInvalidThemeMetadata("Theme config must contain exactly one theme")
	}

	// 獲取主題名稱和結構
	var themeName string
	var themeStructure ThemeStructure
	for name, structure := range themeConfig {
		themeName = name
		themeStructure = structure
		// 設置主題目錄名
		themeStructure.Theme = themeName
		break
	}

	Debug("Theme name: '%s'", themeName)
	Debug("Theme structure: %+v", themeStructure)

	// 驗證主題結構
	if validate {
		if err := mm.ValidateThemeStructure(themeName, themeStructure); err != nil {
			return nil, err
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
	Debug("Installing theme from URL: '%s'", themeURL)
	
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
		// 設置主題目錄名
		themeStructure.Theme = themeName
		break
	}

	Debug("Installing theme: '%s'", themeName)

	// 載入當前配置
	currentConfig, err := mm.LoadThemeConfig()
	if err != nil {
		return fmt.Errorf("Failed to load current theme config: '%v'", err)
	}

	// 檢查主題是否已存在於配置中
	if _, exists := currentConfig[themeName]; exists {
		// 如果存在於配置中，先刪除舊的主題目錄
		themeDir := filepath.Join(mm.themesPath, themeName)
		if err := os.RemoveAll(themeDir); err != nil {
			return fmt.Errorf("Failed to remove existing theme directory: '%v'", err)
		}
		Debug("Removed existing theme directory: '%s'", themeDir)
	}

	// 創建主題目錄
	themeDir := filepath.Join(mm.themesPath, themeName)
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		return NewErrorFailedToCreateThemeDirectory(err)
	}

	Debug("Created theme directory: '%s'", themeDir)

	// 下載並安裝文件
	if err := downloadThemeFiles(themeDir, "", themeStructure.Structure); err != nil {
		// 如果下載失敗，清理已創建的目錄
		os.RemoveAll(themeDir)
		return err
	}

	Debug("Downloaded theme files successfully")

	// 更新主題配置
	currentConfig[themeName] = themeStructure
	return mm.SaveThemeConfig(currentConfig)
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
	// 驗證主題名稱格式
	if !regexp.MustCompile(`^[a-z0-9_-]+$`).MatchString(themeName) {
		return fmt.Errorf("Invalid theme name format: '%s'", themeName)
	}

	// 驗證必要字段
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

	// 驗證結構格式
	return validateStructureFormat("", structure.Structure)
}

// validateStructureFormat 只驗證結構格式，不檢查文件是否存在
func validateStructureFormat(currentPath string, structure interface{}) error {
	switch v := structure.(type) {
	case map[string]interface{}:
		// 處理目錄
		for key, value := range v {
			newPath := filepath.Join(currentPath, key)
			if err := validateStructureFormat(newPath, value); err != nil {
				return err
			}
		}
	case string:
		// 驗證 URL 格式
		if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
			return fmt.Errorf("Invalid URL format for path %s: %s", currentPath, v)
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
			
			// 只有當值是 map 時才創建目錄
			if _, isMap := value.(map[string]interface{}); isMap {
				if err := os.MkdirAll(dirPath, 0755); err != nil {
					return fmt.Errorf("Failed to create directory %s: %v", dirPath, err)
				}
			}
			
			if err := downloadThemeFiles(baseDir, newPath, value); err != nil {
				return err
			}
		}
	case string:
		// 處理文件
		fileURL := v
		filePath := filepath.Join(baseDir, currentPath)
		
		// 確保父目錄存在
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("Failed to create parent directory for %s: %v", filePath, err)
		}
		
		// 下載文件
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