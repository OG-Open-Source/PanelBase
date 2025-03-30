package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	appmodels "github.com/OG-Open-Source/PanelBase/internal/app/models"
	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	sharedmodels "github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/OG-Open-Source/PanelBase/internal/utils"
	"github.com/gin-gonic/gin"
)

// ThemeHandler 主题处理器
type ThemeHandler struct {
	configService *services.ConfigService
}

// NewThemeHandler 创建新的主题处理器
func NewThemeHandler(configService *services.ConfigService) *ThemeHandler {
	return &ThemeHandler{
		configService: configService,
	}
}

// GetThemes 获取所有主题
func (h *ThemeHandler) GetThemes(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"themes": h.configService.ThemesConfig.Themes})
}

// InstallTheme 安装新主题
func (h *ThemeHandler) InstallTheme(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数"})
		return
	}

	// 验证URL
	if !h.validateURL(request.URL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的URL"})
		return
	}

	// 获取主题配置
	themeData, err := h.fetchThemeFromURL(request.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取主题配置", "details": err.Error()})
		return
	}

	// 解析主题配置
	var themeConfig map[string]interface{}
	if err := json.Unmarshal(themeData, &themeConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题配置格式错误", "details": err.Error()})
		return
	}

	// 提取主题信息
	themeData, err = h.extractThemeFromConfig(themeConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法提取主题信息", "details": err.Error()})
		return
	}

	// 解析主题对象
	var theme sharedmodels.Theme
	if err := json.Unmarshal(themeData, &theme); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题信息格式错误", "details": err.Error()})
		return
	}

	// 验证主题数据完整性
	if !theme.ValidateTheme() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题数据不完整", "details": "缺少必要字段"})
		return
	}

	// 获取主题ID
	themeID := theme.Directory
	if themeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的主题ID"})
		return
	}

	// 检查主题是否已存在
	if _, exists := h.configService.ThemesConfig.Themes[themeID]; exists {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题已存在", "theme_id": themeID})
		return
	}

	// 创建主题目录
	themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
	if err := os.MkdirAll(themePath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建主题目录", "details": err.Error()})
		return
	}

	// 下载主题文件
	if err := h.downloadThemeFiles(&theme, themePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法下载主题文件", "details": err.Error()})
		return
	}

	// 清除结构信息，避免将其保存到themes.json中
	clonedTheme := theme
	clonedTheme.Structure = nil

	// 存储主题信息到配置
	if h.configService.ThemesConfig.Themes == nil {
		h.configService.ThemesConfig.Themes = make(map[string]sharedmodels.ThemeInfo)
	}

	themeInfo := sharedmodels.ThemeInfo{
		Name:        clonedTheme.Name,
		Authors:     clonedTheme.Authors,
		Version:     clonedTheme.Version,
		Description: clonedTheme.Description,
		SourceLink:  clonedTheme.SourceLink,
		Directory:   clonedTheme.Directory,
		Structure:   clonedTheme.Structure,
	}

	h.configService.ThemesConfig.Themes[themeID] = themeInfo

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存主题配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "主题安装成功",
		"theme":   themeInfo,
	})
}

// UpdateTheme 更新主题
func (h *ThemeHandler) UpdateTheme(c *gin.Context) {
	themeID := c.Param("id")

	// 检查主题是否存在
	if _, exists := h.configService.ThemesConfig.Themes[themeID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "主题不存在", "id": themeID})
		return
	}

	// 获取当前主题
	theme := h.configService.ThemesConfig.Themes[themeID]

	// 从URL获取最新的主题配置
	themeData, err := h.fetchThemeFromURL(theme.SourceLink)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取主题配置", "details": err.Error()})
		return
	}

	// 解析主题信息
	var themeConfig map[string]interface{}
	if err := json.Unmarshal(themeData, &themeConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题配置格式错误", "details": err.Error()})
		return
	}

	// 提取主题信息
	themeData, err = h.extractThemeFromConfig(themeConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法提取主题信息", "details": err.Error()})
		return
	}

	// 解析更新后的主题对象
	var updatedTheme sharedmodels.Theme
	if err := json.Unmarshal(themeData, &updatedTheme); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题信息格式错误", "details": err.Error()})
		return
	}

	// 验证主题数据完整性
	if !updatedTheme.ValidateTheme() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题数据不完整", "details": "缺少必要字段"})
		return
	}

	// 清空主题目录
	themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
	if err := os.RemoveAll(themePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法清空主题目录", "details": err.Error()})
		return
	}

	// 重新创建主题目录
	if err := os.MkdirAll(themePath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建主题目录", "details": err.Error()})
		return
	}

	// 下载主题文件
	if err := h.downloadThemeFiles(&updatedTheme, themePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法下载主题文件", "details": err.Error()})
		return
	}

	// 清除结构信息，避免将其保存到themes.json中
	updatedTheme.Structure = nil

	// 更新主题信息
	h.configService.ThemesConfig.Themes[themeID] = updatedTheme.ToThemeInfo()

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存主题配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "主题更新成功",
		"theme":   updatedTheme.ToThemeInfo(),
	})
}

// DeleteTheme 删除主题
func (h *ThemeHandler) DeleteTheme(c *gin.Context) {
	themeID := c.Param("id")

	// 检查主题是否存在
	if _, exists := h.configService.ThemesConfig.Themes[themeID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "主题不存在", "id": themeID})
		return
	}

	// 检查是否为当前使用的主题
	if h.configService.ThemesConfig.CurrentTheme == themeID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法删除当前使用的主题", "id": themeID})
		return
	}

	// 删除主题目录
	themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
	if err := os.RemoveAll(themePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法删除主题目录", "details": err.Error()})
		return
	}

	// 从配置中移除主题
	delete(h.configService.ThemesConfig.Themes, themeID)

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存主题配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "主题删除成功",
	})
}

// ActivateTheme 激活指定主题
func (h *ThemeHandler) ActivateTheme(c *gin.Context) {
	themeID := c.Param("id")

	// 检查主题是否存在
	if _, exists := h.configService.ThemesConfig.Themes[themeID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "主题不存在", "id": themeID})
		return
	}

	// 更新当前主题
	h.configService.ThemesConfig.CurrentTheme = themeID

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存主题配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "主题激活成功",
		"theme":   h.configService.ThemesConfig.Themes[themeID],
	})
}

// ThemeV1Handler V1版本的主题处理程序
func (h *ThemeHandler) ThemeV1Handler(c *gin.Context) {
	// 根据请求方法分发到对应的处理函数
	switch c.Request.Method {
	case "GET":
		h.GetThemesV1(c)
	case "POST":
		h.InstallThemeV1(c)
	case "PUT":
		// 更新全部主题列表
		h.UpdateAllThemesV1(c)
	case "PATCH":
		// 更新指定主题的全部内容
		h.UpdateSingleThemeV1(c)
	case "DELETE":
		h.DeleteThemeV1(c)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "不支持的请求方法"})
	}
}

// GetThemesV1 获取主题信息 (V1 API)
func (h *ThemeHandler) GetThemesV1(c *gin.Context) {
	// 直接返回themes.json的内容
	c.JSON(http.StatusOK, h.configService.ThemesConfig)
}

// InstallThemeV1 安装新主题 (V1 API)
func (h *ThemeHandler) InstallThemeV1(c *gin.Context) {
	var request appmodels.ThemeRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	// URL格式验证
	if !h.validateURL(request.URL) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的URL格式"})
		return
	}

	// 从URL获取主题配置
	themeData, err := h.fetchThemeFromURL(request.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取主题配置", "details": err.Error()})
		return
	}

	// 解析主题信息
	var themeConfig map[string]interface{}
	if err := json.Unmarshal(themeData, &themeConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题配置格式错误", "details": err.Error()})
		return
	}

	// 提取主题信息
	themeData, err = h.extractThemeFromConfig(themeConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法提取主题信息", "details": err.Error()})
		return
	}

	// 解析主题对象
	var theme sharedmodels.Theme
	if err := json.Unmarshal(themeData, &theme); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题信息格式错误", "details": err.Error()})
		return
	}

	// 验证主题数据完整性
	if !theme.ValidateTheme() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主题数据不完整", "details": "缺少必要字段"})
		return
	}

	// 创建主题目录
	themeID := theme.Directory
	themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
	if err := os.MkdirAll(themePath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建主题目录", "details": err.Error()})
		return
	}

	// 下载主题文件
	if err := h.downloadThemeFiles(&theme, themePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法下载主题文件", "details": err.Error()})
		return
	}

	// 清除结构信息，避免将其保存到themes.json中
	structureCopy := theme.Structure
	theme.Structure = nil

	// 存储主题信息到配置
	if h.configService.ThemesConfig.Themes == nil {
		h.configService.ThemesConfig.Themes = make(map[string]sharedmodels.ThemeInfo)
	}
	h.configService.ThemesConfig.Themes[themeID] = theme.ToThemeInfo()

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存主题配置", "details": err.Error()})
		return
	}

	// 恢复结构信息用于响应
	theme.Structure = structureCopy

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "主题安装成功",
		"theme":   theme.ToThemeInfo(),
	})
}

// UpdateAllThemesV1 更新全部主題列表 (V1 API)
func (h *ThemeHandler) UpdateAllThemesV1(c *gin.Context) {
	// 更新全部主題列表
	themesToUpdate := make(map[string]sharedmodels.ThemeInfo)

	// 遍歷所有主題，檢查更新
	for themeID, theme := range h.configService.ThemesConfig.Themes {
		// 獲取源鏈接
		sourceLink := theme.SourceLink
		if sourceLink == "" {
			log.Printf("主題 %s 未設置源鏈接，跳過更新", themeID)
			themesToUpdate[themeID] = theme
			continue
		}

		// 從URL獲取最新的主題配置
		themeData, err := h.fetchThemeFromURL(sourceLink)
		if err != nil {
			log.Printf("無法獲取主題 %s 的配置: %v", themeID, err)
			themesToUpdate[themeID] = theme
			continue
		}

		// 解析主題信息
		var themeConfig map[string]interface{}
		if err := json.Unmarshal(themeData, &themeConfig); err != nil {
			log.Printf("主題 %s 配置格式錯誤: %v", themeID, err)
			themesToUpdate[themeID] = theme
			continue
		}

		// 提取主題信息
		themeData, err = h.extractThemeFromConfig(themeConfig)
		if err != nil {
			log.Printf("無法提取主題 %s 信息: %v", themeID, err)
			themesToUpdate[themeID] = theme
			continue
		}

		// 解析更新後的主題對象
		var updatedTheme sharedmodels.Theme
		if err := json.Unmarshal(themeData, &updatedTheme); err != nil {
			log.Printf("主題 %s 信息格式錯誤: %v", themeID, err)
			themesToUpdate[themeID] = theme
			continue
		}

		// 比較版本號
		if updatedTheme.Version == theme.Version {
			log.Printf("主題 %s 已是最新版本: %s", themeID, theme.Version)
			themesToUpdate[themeID] = theme
			continue
		}

		log.Printf("主題 %s 有更新: %s -> %s", themeID, theme.Version, updatedTheme.Version)

		// 驗證主題數據完整性
		if !updatedTheme.ValidateTheme() {
			log.Printf("主題 %s 數據不完整", themeID)
			themesToUpdate[themeID] = theme
			continue
		}

		// 創建主題目錄
		themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
		if err := os.MkdirAll(themePath, 0755); err != nil {
			log.Printf("無法創建主題 %s 目錄: %v", themeID, err)
			themesToUpdate[themeID] = theme
			continue
		}

		// 下載主題文件
		if err := h.downloadThemeFiles(&updatedTheme, themePath); err != nil {
			log.Printf("無法下載主題 %s 文件: %v", themeID, err)
			themesToUpdate[themeID] = theme
			continue
		}

		// 清除結構信息，避免將其保存到themes.json中
		structureCopy := updatedTheme.Structure
		updatedTheme.Structure = nil

		// 更新主題信息
		themesToUpdate[themeID] = updatedTheme.ToThemeInfo()

		// 恢復結構信息
		updatedTheme.Structure = structureCopy

		log.Printf("主題 %s 更新成功", themeID)
	}

	// 更新全部主題列表
	h.configService.ThemesConfig.Themes = themesToUpdate

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存主題配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "全部主題更新檢查完成",
		"themes":  themesToUpdate,
	})
}

// UpdateSingleThemeV1 更新指定主題 (V1 API)
func (h *ThemeHandler) UpdateSingleThemeV1(c *gin.Context) {
	var request struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求參數", "details": err.Error()})
		return
	}

	themeID := request.ID

	// 檢查主題是否存在
	theme, exists := h.configService.ThemesConfig.Themes[themeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "主題不存在", "id": themeID})
		return
	}

	// 獲取源鏈接
	sourceLink := theme.SourceLink
	if sourceLink == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主題未設置源鏈接", "id": themeID})
		return
	}

	// 從URL獲取最新的主題配置
	themeData, err := h.fetchThemeFromURL(sourceLink)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法獲取主題配置", "details": err.Error()})
		return
	}

	// 解析主題信息
	var themeConfig map[string]interface{}
	if err := json.Unmarshal(themeData, &themeConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主題配置格式錯誤", "details": err.Error()})
		return
	}

	// 提取主題信息
	themeData, err = h.extractThemeFromConfig(themeConfig)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法提取主題信息", "details": err.Error()})
		return
	}

	// 解析更新後的主題對象
	var updatedTheme sharedmodels.Theme
	if err := json.Unmarshal(themeData, &updatedTheme); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主題信息格式錯誤", "details": err.Error()})
		return
	}

	// 比較版本號
	if updatedTheme.Version == theme.Version {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "主題已是最新版本",
			"theme":   theme,
		})
		return
	}

	// 驗證主題數據完整性
	if !updatedTheme.ValidateTheme() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "主題數據不完整", "details": "缺少必要字段"})
		return
	}

	// 創建主題目錄
	themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
	if err := os.MkdirAll(themePath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法創建主題目錄", "details": err.Error()})
		return
	}

	// 下載主題文件
	if err := h.downloadThemeFiles(&updatedTheme, themePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法下載主題文件", "details": err.Error()})
		return
	}

	// 清除結構信息，避免將其保存到themes.json中
	structureCopy := updatedTheme.Structure
	updatedTheme.Structure = nil

	// 更新主題信息
	h.configService.ThemesConfig.Themes[themeID] = updatedTheme.ToThemeInfo()

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存主題配置", "details": err.Error()})
		return
	}

	// 恢復結構信息
	updatedTheme.Structure = structureCopy

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "主題更新成功",
		"theme":   updatedTheme.ToThemeInfo(),
	})
}

// DeleteThemeV1 删除主题 (V1 API)
func (h *ThemeHandler) DeleteThemeV1(c *gin.Context) {
	themeID := c.Query("id")
	if themeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少id参数"})
		return
	}

	// 检查主题是否存在
	if _, exists := h.configService.ThemesConfig.Themes[themeID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "主题不存在", "id": themeID})
		return
	}

	// 检查是否为当前使用的主题
	if h.configService.ThemesConfig.CurrentTheme == themeID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法删除当前使用的主题", "id": themeID})
		return
	}

	// 删除主题目录
	themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
	if err := os.RemoveAll(themePath); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法删除主题目录", "details": err.Error()})
		return
	}

	// 从配置中移除主题
	delete(h.configService.ThemesConfig.Themes, themeID)

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存主题配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "主题删除成功",
	})
}

// GetThemeByIDV1 获取指定主题的信息 (V1 API)
func (h *ThemeHandler) GetThemeByIDV1(c *gin.Context) {
	themeID := c.Param("theme_id")
	if themeID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少theme_id參數"})
		return
	}

	// 檢查主題是否存在
	theme, exists := h.configService.ThemesConfig.Themes[themeID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "主題不存在", "theme_id": themeID})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"theme":  theme,
	})
}

// HandleThemeV1 處理主題接口請求 (V1 API)
func (h *ThemeHandler) HandleThemeV1(c *gin.Context) {
	method := c.Request.Method
	switch method {
	case "GET":
		// 判斷是獲取所有主題還是特定主題
		var request struct {
			ID string `json:"id"`
		}
		if err := c.ShouldBindJSON(&request); err == nil && request.ID != "" {
			// 獲取特定主題
			c.Set("theme_id", request.ID)
			h.GetThemeByIDV1(c)
		} else {
			// 獲取所有主題
			h.GetThemesV1(c)
		}
	case "POST":
		// 判斷是安裝新主題還是切換主題
		var request struct {
			URL    string `json:"url"`
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求參數", "details": err.Error()})
			return
		}

		if request.URL != "" {
			// 安裝新主題
			h.InstallThemeV1(c)
		} else if request.ID != "" && request.Status == "switch" {
			// 切換主題
			c.Set("theme_id", request.ID)
			h.ActivateTheme(c)
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求參數", "details": "需要url或id+status=switch"})
		}
	case "PUT":
		// 更新全部主題列表
		h.UpdateAllThemesV1(c)
	case "PATCH":
		// 更新指定主題
		h.UpdateSingleThemeV1(c)
	case "DELETE":
		// 判斷是刪除所有主題還是特定主題
		var request struct {
			ID string `json:"id"`
		}
		if err := c.ShouldBindJSON(&request); err == nil && request.ID != "" {
			// 刪除特定主題
			c.Set("theme_id", request.ID)
			h.DeleteThemeV1(c)
		} else {
			// 刪除所有主題
			h.DeleteAllThemesV1(c)
		}
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "不支持的請求方法"})
	}
}

// DeleteAllThemesV1 刪除所有主題 (V1 API)
func (h *ThemeHandler) DeleteAllThemesV1(c *gin.Context) {
	// 獲取當前主題
	currentTheme := h.configService.ThemesConfig.CurrentTheme

	// 遍歷所有主題，除了當前主題
	for themeID := range h.configService.ThemesConfig.Themes {
		if themeID == currentTheme {
			continue
		}

		// 刪除主題目錄
		themePath := filepath.Join(h.configService.BaseDir, utils.WebDir, themeID)
		if err := os.RemoveAll(themePath); err != nil {
			log.Printf("無法刪除主題 %s 目錄: %v", themeID, err)
		}

		// 從配置中刪除主題
		delete(h.configService.ThemesConfig.Themes, themeID)
	}

	// 保存配置
	if err := h.configService.SaveThemesConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存主題配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "已刪除所有非當前主題",
		"themes":  h.configService.ThemesConfig.Themes,
	})
}

// 辅助方法

// fetchThemeFromURL 从URL获取主题配置
func (h *ThemeHandler) fetchThemeFromURL(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求URL失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("请求返回错误状态码: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// extractThemeFromConfig 从配置中提取主题信息
func (h *ThemeHandler) extractThemeFromConfig(config map[string]interface{}) ([]byte, error) {
	// 检查主题数据格式
	themeData, ok := config["theme"]
	if !ok {
		// 尝试直接解析整个配置
		if _, ok := config["name"]; ok {
			return json.Marshal(config)
		}
		return nil, fmt.Errorf("主题配置中不包含theme字段")
	}

	// 如果theme是对象，且包含多个主题，取第一个
	if themeMap, ok := themeData.(map[string]interface{}); ok {
		if len(themeMap) > 0 {
			for _, v := range themeMap {
				if themeObj, ok := v.(map[string]interface{}); ok {
					return json.Marshal(themeObj)
				}
			}
		}
	}

	return nil, fmt.Errorf("无法提取主题信息")
}

// downloadThemeFiles 下载主题文件
func (h *ThemeHandler) downloadThemeFiles(theme *sharedmodels.Theme, themePath string) error {
	// 递归处理结构
	return h.processThemeStructure(theme.Structure, themePath)
}

// processThemeStructure 处理主题结构
func (h *ThemeHandler) processThemeStructure(structure map[string]interface{}, basePath string) error {
	for key, value := range structure {
		switch v := value.(type) {
		case string:
			// 为文件创建必要的目录
			filePath := filepath.Join(basePath, key)
			dir := filepath.Dir(filePath)
			if err := os.MkdirAll(dir, 0755); err != nil {
				return err
			}

			// 下载文件
			if err := h.downloadFile(v, filePath); err != nil {
				return err
			}
		case map[string]interface{}:
			// 创建目录
			dirPath := filepath.Join(basePath, key)
			if err := os.MkdirAll(dirPath, 0755); err != nil {
				return err
			}

			// 递归处理子目录
			if err := h.processThemeStructure(v, dirPath); err != nil {
				return err
			}
		}
	}
	return nil
}

// downloadFile 下载单个文件
func (h *ThemeHandler) downloadFile(url string, filePath string) error {
	// 确保目录存在
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("无法创建目录: %w", err)
	}

	// 下载文件
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载文件返回错误状态码: %d", resp.StatusCode)
	}

	// 创建文件
	out, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer out.Close()

	// 写入文件
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}

// validateURL 验证URL格式是否有效
func (h *ThemeHandler) validateURL(url string) bool {
	// 简单验证URL格式
	return strings.HasPrefix(url, "http://") || strings.HasPrefix(url, "https://")
}
