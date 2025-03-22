package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/OG-Open-Source/PanelBase/pkg/theme"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

// ThemeHandler handles theme-related requests
type ThemeHandler struct {
	themeConfig *theme.ThemeConfig
}

// NewThemeHandler creates a new theme handler
func NewThemeHandler(themeConfig *theme.ThemeConfig) *ThemeHandler {
	return &ThemeHandler{
		themeConfig: themeConfig,
	}
}

// RegisterRoutes registers theme routes
func (h *ThemeHandler) RegisterRoutes(e *echo.Echo, entryPath string) {
	// Get the theme directory
	themeDir, err := h.themeConfig.GetThemeDirectory()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to get theme directory")
	}

	// Create a new group with the entry path
	g := e.Group(fmt.Sprintf("/%s", entryPath))

	// Serve theme files
	g.Static("/", themeDir)

	// Serve index.html as the root
	g.GET("", func(c echo.Context) error {
		return c.File(filepath.Join(themeDir, "index.html"))
	})

	// Add handler for theme metadata
	g.GET("/theme/info", h.GetThemeInfo)

	// 添加新的主题下载 API 端点
	g.GET("/theme/download", h.DownloadTheme)

	// 添加新的主题元数据检查 API 端点
	g.GET("/theme/metadata", h.CheckThemeMetadata)
}

// GetThemeInfo returns information about the current theme
func (h *ThemeHandler) GetThemeInfo(c echo.Context) error {
	currentTheme, err := h.themeConfig.GetCurrentTheme()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to get current theme: %v", err),
		})
	}

	// 返回完整的主题信息，包括目录和结构
	return c.JSON(http.StatusOK, currentTheme)
}

// DownloadTheme 下载指定 URL 的主题文件
func (h *ThemeHandler) DownloadTheme(c echo.Context) error {
	url := c.QueryParam("url")
	if url == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "URL parameter is required",
		})
	}

	// 检查 URL 是否以 .json 结尾
	if !strings.HasSuffix(url, ".json") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "URL must point to a JSON file",
		})
	}

	// 检查 URL 格式是否有效
	_, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid URL format: %v", err),
		})
	}

	// 下载主题文件
	resp, err := http.Get(url)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to download theme: %v", err),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.JSON(http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("Failed to download theme, status code: %d", resp.StatusCode),
		})
	}

	// 读取并验证 JSON 格式
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to read theme data: %v", err),
		})
	}

	// 验证 JSON 格式
	var themeData map[string]interface{}
	if err := json.Unmarshal(body, &themeData); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid JSON format: %v", err),
		})
	}

	// 检查主题必要字段
	if _, ok := themeData["name"]; !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Theme data missing required field: name",
		})
	}

	if _, ok := themeData["authors"]; !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Theme data missing required field: authors",
		})
	}

	if _, ok := themeData["version"]; !ok {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Theme data missing required field: version",
		})
	}

	// 获取文件名
	urlParts := strings.Split(url, "/")
	fileName := urlParts[len(urlParts)-1]

	// 保存主题文件到临时目录
	tempDir := filepath.Join("themes", "temp")
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to create temp directory: %v", err),
		})
	}

	filePath := filepath.Join(tempDir, fileName)
	if err := os.WriteFile(filePath, body, 0644); err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to save theme file: %v", err),
		})
	}

	// 返回成功消息
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "success",
		"message":    "Theme downloaded successfully",
		"file_path":  filePath,
		"theme_name": themeData["name"],
	})
}

// CheckThemeMetadata 检查指定 URL 的主题元数据
func (h *ThemeHandler) CheckThemeMetadata(c echo.Context) error {
	url := c.QueryParam("url")
	if url == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "URL parameter is required",
		})
	}

	// 检查 URL 是否以 .json 结尾
	if !strings.HasSuffix(url, ".json") {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "URL must point to a JSON file",
		})
	}

	// 检查 URL 格式是否有效
	_, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid URL format: %v", err),
		})
	}

	// 下载主题元数据
	resp, err := http.Get(url)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to fetch theme metadata: %v", err),
		})
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return c.JSON(http.StatusBadGateway, map[string]string{
			"error": fmt.Sprintf("Failed to fetch theme metadata, status code: %d", resp.StatusCode),
		})
	}

	// 读取元数据
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": fmt.Sprintf("Failed to read theme metadata: %v", err),
		})
	}

	// 验证 JSON 格式并返回完整内容
	var themeMetadata map[string]interface{}
	if err := json.Unmarshal(body, &themeMetadata); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": fmt.Sprintf("Invalid JSON format: %v", err),
		})
	}

	// 返回主题元数据
	return c.JSON(http.StatusOK, themeMetadata)
}
