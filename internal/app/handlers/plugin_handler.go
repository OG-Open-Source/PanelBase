package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"log"

	"github.com/OG-Open-Source/PanelBase/internal/app/models"
	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	sharedmodels "github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/OG-Open-Source/PanelBase/internal/utils"
	pkgutils "github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/gin-gonic/gin"
)

// PluginHandler 处理插件API请求
type PluginHandler struct {
	configService *services.ConfigService
	pluginManager *pkgutils.PluginManager
}

// NewPluginHandler 创建插件处理器
func NewPluginHandler(configService *services.ConfigService) *PluginHandler {
	// 创建插件管理器
	pluginsPath := filepath.Join(configService.BaseDir, utils.PluginsDir)
	pluginManager := pkgutils.NewPluginManager(pluginsPath)

	// 加载所有插件
	err := pluginManager.LoadAllPlugins()
	if err != nil {
		// 在实际应用中应该记录日志而不是忽略错误
		// 这里我们只是继续处理
	}

	return &PluginHandler{
		configService: configService,
		pluginManager: pluginManager,
	}
}

// HandlePluginAPI handles plugin API requests
func (h *PluginHandler) HandlePluginAPI(c *gin.Context) {
	pluginID := c.Param("plugin_id")
	route := c.Param("route")

	// Check if the plugin exists
	plugin, exists := h.pluginManager.Plugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found", "plugin_id": pluginID})
		return
	}

	// Check if the plugin is running
	if !plugin.IsRunning {
		// Auto-start the plugin
		if err := h.pluginManager.StartPlugin(pluginID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start plugin", "details": err.Error()})
			return
		}
	}

	// Validate that the method is one of the allowed ones (GET, POST, PUT, PATCH, DELETE)
	method := c.Request.Method
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	validMethod := false
	for _, m := range validMethods {
		if method == m {
			validMethod = true
			break
		}
	}

	if !validMethod {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":           "Method not allowed",
			"plugin":          pluginID,
			"path":            route,
			"method":          method,
			"allowed_methods": validMethods,
		})
		return
	}

	// Build API request object
	request := &models.PluginAPIRequest{
		Method:  method,
		Path:    route,
		Query:   make(map[string]string),
		Headers: make(map[string]string),
	}

	// Get query parameters
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			request.Query[key] = values[0]
		}
	}

	// Get request headers
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			request.Headers[key] = values[0]
		}
	}

	// For POST/PUT/PATCH requests, process the request body
	if method == "POST" || method == "PUT" || method == "PATCH" {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err == nil {
			request.Body = body
		}
	}

	// Execute plugin API
	response, err := h.pluginManager.ExecuteAPIRoute(pluginID, route, request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute plugin API", "details": err.Error()})
		return
	}

	// Return plugin response
	c.JSON(http.StatusOK, response)
}

// ListPlugins lists all plugins
func (h *PluginHandler) ListPlugins(c *gin.Context) {
	plugins := make([]map[string]interface{}, 0)

	for id, plugin := range h.pluginManager.Plugins {
		plugins = append(plugins, map[string]interface{}{
			"id":          id,
			"name":        plugin.Info.Name,
			"description": plugin.Info.Description,
			"version":     plugin.Info.Version,
			"author":      plugin.Info.Author,
			"status":      plugin.Status,
			"is_running":  plugin.IsRunning,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"plugins": plugins,
		},
	})
}

// GetPluginInfo gets plugin information
func (h *PluginHandler) GetPluginInfo(c *gin.Context) {
	pluginID := c.Param("plugin_id")

	// Check if the plugin exists
	plugin, exists := h.pluginManager.Plugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found", "plugin_id": pluginID})
		return
	}

	// Get task information
	tasks := make([]map[string]interface{}, 0)
	for _, task := range plugin.BackgroundTasks {
		tasks = append(tasks, map[string]interface{}{
			"id":      task.ID,
			"name":    task.Name,
			"status":  task.Status.Status,
			"message": task.Status.Message,
		})
	}

	// Get API routes
	routes := make([]string, 0)
	for route := range plugin.APIRoutes {
		routes = append(routes, route)
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"id":          pluginID,
			"name":        plugin.Info.Name,
			"description": plugin.Info.Description,
			"version":     plugin.Info.Version,
			"author":      plugin.Info.Author,
			"status":      plugin.Status,
			"is_running":  plugin.IsRunning,
			"tasks":       tasks,
			"api_routes":  routes,
		},
	})
}

// StartPlugin starts a plugin
func (h *PluginHandler) StartPlugin(c *gin.Context) {
	pluginID := c.Param("plugin_id")

	// Check if the plugin exists
	_, exists := h.pluginManager.Plugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found", "plugin_id": pluginID})
		return
	}

	// Start the plugin
	if err := h.pluginManager.StartPlugin(pluginID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to start plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Plugin started",
	})
}

// StopPlugin stops a plugin
func (h *PluginHandler) StopPlugin(c *gin.Context) {
	pluginID := c.Param("plugin_id")

	// Check if the plugin exists
	_, exists := h.pluginManager.Plugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found", "plugin_id": pluginID})
		return
	}

	// Stop the plugin
	if err := h.pluginManager.StopPlugin(pluginID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to stop plugin", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Plugin stopped",
	})
}

// GetTaskStatus gets task status
func (h *PluginHandler) GetTaskStatus(c *gin.Context) {
	taskID := c.Param("task_id")

	// Get task status
	status, err := h.pluginManager.GetTaskStatus(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   status,
	})
}

// HandlePluginV1 handles plugin API requests for V1 version
func (h *PluginHandler) HandlePluginV1(c *gin.Context) {
	pluginName := c.Param("name")
	pluginPath := c.Param("path")

	// Check if the plugin exists in the configuration
	pluginInfo := h.configService.PluginsConfig.GetPlugin(pluginName)
	if pluginInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found", "plugin": pluginName})
		return
	}

	// Check API version
	if pluginInfo.APIVersion != utils.APIVersion {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "API version mismatch",
			"plugin":   pluginName,
			"expected": utils.APIVersion,
			"actual":   pluginInfo.APIVersion,
		})
		return
	}

	// Check if the plugin exists in the plugin manager
	plugin, exists := h.pluginManager.Plugins[pluginName]
	if !exists {
		// If the plugin exists in the configuration but is not loaded, try to load it
		log.Printf("Plugin %s exists in config but not loaded, attempting to load\n", pluginName)
		var err error
		plugin, err = h.pluginManager.LoadPlugin(pluginName)
		if err != nil {
			log.Printf("Failed to load plugin %s: %v\n", pluginName, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to load plugin",
				"plugin":  pluginName,
				"details": err.Error(),
			})
			return
		}

		// Check again if the plugin is loaded
		if plugin == nil {
			log.Printf("Plugin %s load failed, no valid plugin instance returned\n", pluginName)
			// Return error but don't break the service
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Plugin load failed",
				"plugin":  pluginName,
				"details": "Could not get a valid plugin instance",
			})
			return
		}
	}

	// If the plugin doesn't exist or isn't properly loaded, return response without continuing
	if plugin == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Plugin not loaded",
			"plugin":  pluginName,
			"details": "Plugin may not exist or failed to load",
		})
		return
	}

	// Check if the plugin is running
	if !plugin.IsRunning {
		// Try to start the plugin
		if err := h.pluginManager.StartPlugin(pluginName); err != nil {
			log.Printf("Failed to start plugin %s: %v\n", pluginName, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to start plugin",
				"plugin":  pluginName,
				"details": err.Error(),
			})
			return
		}
	}

	// Find the endpoint configuration
	endpoint := pluginInfo.GetEndpoint(pluginPath)
	if endpoint == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Plugin endpoint not found",
			"plugin": pluginName,
			"path":   pluginPath,
		})
		return
	}

	// Check if the request method is supported
	method := c.Request.Method

	// Validate that the method is one of the allowed ones (GET, POST, PUT, PATCH, DELETE)
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	validMethod := false
	for _, m := range validMethods {
		if method == m {
			validMethod = true
			break
		}
	}

	if !validMethod {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":           "Method not allowed",
			"plugin":          pluginName,
			"path":            pluginPath,
			"method":          method,
			"allowed_methods": validMethods,
		})
		return
	}

	// Check if the endpoint supports the method
	if !endpoint.SupportsMethod(method) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":             "Method not supported by this endpoint",
			"plugin":            pluginName,
			"path":              pluginPath,
			"method":            method,
			"supported_methods": endpoint.Methods,
		})
		return
	}

	// Build API request object
	request := &models.PluginAPIRequest{
		Method:  method,
		Path:    pluginPath,
		Query:   make(map[string]string),
		Headers: make(map[string]string),
	}

	// Get query parameters
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			request.Query[key] = values[0]
		}
	}

	// Get request headers
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			request.Headers[key] = values[0]
		}
	}

	// For POST/PUT/PATCH requests, process the request body
	if method == "POST" || method == "PUT" || method == "PATCH" {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err == nil {
			request.Body = body
		}
	}

	// Execute plugin API
	response, err := h.pluginManager.ExecuteAPIRoute(pluginName, pluginPath, request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to execute plugin API",
			"plugin":  pluginName,
			"path":    pluginPath,
			"details": err.Error(),
		})
		return
	}

	// Return plugin response
	c.JSON(http.StatusOK, response)
}

// HandlePluginV2 handles plugin API requests for V2 version, replacing HandlePluginV1
func (h *PluginHandler) HandlePluginV2(c *gin.Context) {
	pluginName := c.Param("plugin_id")
	pluginPath := c.Param("api_path")

	log.Printf("Processing plugin API request: plugin=%s, path=%s\n", pluginName, pluginPath)

	// Check if the plugin exists in the configuration
	pluginInfo := h.configService.PluginsConfig.GetPlugin(pluginName)
	if pluginInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin not found", "plugin": pluginName})
		return
	}

	// Check API version
	if pluginInfo.APIVersion != utils.APIVersion {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":    "API version mismatch",
			"plugin":   pluginName,
			"expected": utils.APIVersion,
			"actual":   pluginInfo.APIVersion,
		})
		return
	}

	// Check if the plugin exists in the plugin manager
	plugin, exists := h.pluginManager.Plugins[pluginName]
	if !exists {
		// If the plugin exists in the configuration but is not loaded, try to load it
		log.Printf("Plugin %s exists in config but not loaded, attempting to load\n", pluginName)
		var err error
		plugin, err = h.pluginManager.LoadPlugin(pluginName)
		if err != nil {
			log.Printf("Failed to load plugin %s: %v\n", pluginName, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to load plugin",
				"plugin":  pluginName,
				"details": err.Error(),
			})
			return
		}

		// Check again if the plugin is loaded
		if plugin == nil {
			log.Printf("Plugin %s load failed, no valid plugin instance returned\n", pluginName)
			// Return error but don't break the service
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Plugin load failed",
				"plugin":  pluginName,
				"details": "Could not get a valid plugin instance",
			})
			return
		}
	}

	// If the plugin doesn't exist or isn't properly loaded, return response without continuing
	if plugin == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error":   "Plugin not loaded",
			"plugin":  pluginName,
			"details": "Plugin may not exist or failed to load",
		})
		return
	}

	// Check if the plugin is running
	if !plugin.IsRunning {
		// Try to start the plugin
		if err := h.pluginManager.StartPlugin(pluginName); err != nil {
			log.Printf("Failed to start plugin %s: %v\n", pluginName, err)
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Failed to start plugin",
				"plugin":  pluginName,
				"details": err.Error(),
			})
			return
		}
	}

	// Find the endpoint configuration
	endpoint := pluginInfo.GetEndpoint(pluginPath)
	if endpoint == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error":  "Plugin endpoint not found",
			"plugin": pluginName,
			"path":   pluginPath,
		})
		return
	}

	// Check if the request method is supported
	method := c.Request.Method

	// Validate that the method is one of the allowed ones (GET, POST, PUT, PATCH, DELETE)
	validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	validMethod := false
	for _, m := range validMethods {
		if method == m {
			validMethod = true
			break
		}
	}

	if !validMethod {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":           "Method not allowed",
			"plugin":          pluginName,
			"path":            pluginPath,
			"method":          method,
			"allowed_methods": validMethods,
		})
		return
	}

	// Check if the endpoint supports the method
	if !endpoint.SupportsMethod(method) {
		c.JSON(http.StatusMethodNotAllowed, gin.H{
			"error":             "Method not supported by this endpoint",
			"plugin":            pluginName,
			"path":              pluginPath,
			"method":            method,
			"supported_methods": endpoint.Methods,
		})
		return
	}

	// Build API request object
	request := &models.PluginAPIRequest{
		Method:  method,
		Path:    pluginPath,
		Query:   make(map[string]string),
		Headers: make(map[string]string),
	}

	// Get query parameters
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			request.Query[key] = values[0]
		}
	}

	// Get request headers
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			request.Headers[key] = values[0]
		}
	}

	// For POST/PUT/PATCH requests, process the request body
	if method == "POST" || method == "PUT" || method == "PATCH" {
		var body map[string]interface{}
		if err := c.ShouldBindJSON(&body); err == nil {
			request.Body = body
		} else {
			log.Printf("Failed to parse request body: %v\n", err)
		}
	}

	// Execute plugin API
	response, err := h.pluginManager.ExecuteAPIRoute(pluginName, pluginPath, request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to execute plugin API",
			"plugin":  pluginName,
			"path":    pluginPath,
			"details": err.Error(),
		})
		return
	}

	// Return plugin response
	c.JSON(http.StatusOK, response)
}

// InstallPluginV1 從URL安裝新插件 (V1 API)
func (h *PluginHandler) InstallPluginV1(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request parameters", "details": err.Error()})
		return
	}

	// 验证URL
	if !strings.HasPrefix(request.URL, "http://") && !strings.HasPrefix(request.URL, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid URL format"})
		return
	}

	// 从URL获取插件配置
	resp, err := http.Get(request.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to fetch plugin configuration", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "URL returned error status", "status": resp.StatusCode})
		return
	}

	pluginData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read plugin data", "details": err.Error()})
		return
	}

	// 解析插件信息
	var pluginConfig map[string]interface{}
	if err := json.Unmarshal(pluginData, &pluginConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin format", "details": err.Error()})
		return
	}

	// 提取插件信息
	pluginObj, ok := pluginConfig["plugin"]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing plugin object in configuration"})
		return
	}

	// 安装插件
	pluginMap, ok := pluginObj.(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin object format"})
		return
	}

	if len(pluginMap) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Empty plugin configuration"})
		return
	}

	// 定义临时结构来解析插件信息
	type TempPluginEndpoint struct {
		Path         string            `json:"path"`
		Methods      []string          `json:"methods"`
		InputFormat  map[string]string `json:"input_format"`
		OutputFormat map[string]string `json:"output_format"`
	}

	type TempPluginInfo struct {
		Name        string               `json:"name"`
		Authors     string               `json:"authors"`
		Version     string               `json:"version"`
		Description string               `json:"description"`
		SourceLink  string               `json:"source_link"`
		APIVersion  string               `json:"api_version"`
		Endpoints   []TempPluginEndpoint `json:"endpoints"`
	}

	// 处理所有插件
	for pluginID, pluginInfoInterface := range pluginMap {
		pluginInfoMap, ok := pluginInfoInterface.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin info format", "plugin_id": pluginID})
			return
		}

		// 转换为JSON以便解析成临时结构
		jsonData, err := json.Marshal(pluginInfoMap)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process plugin info", "details": err.Error()})
			return
		}

		// 解析成临时结构
		var tempPluginInfo TempPluginInfo
		if err := json.Unmarshal(jsonData, &tempPluginInfo); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid plugin format", "details": err.Error()})
			return
		}

		// 验证插件基本数据
		if tempPluginInfo.Name == "" || tempPluginInfo.Version == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Incomplete plugin data", "details": "Missing required fields"})
			return
		}

		// 转换为sharedmodels.PluginInfo以添加到配置中
		pluginInfo := sharedmodels.PluginInfo{
			Name:        tempPluginInfo.Name,
			Authors:     tempPluginInfo.Authors,
			Version:     tempPluginInfo.Version,
			Description: tempPluginInfo.Description,
			SourceLink:  tempPluginInfo.SourceLink,
			APIVersion:  tempPluginInfo.APIVersion,
			Endpoints:   make([]sharedmodels.PluginEndpoint, 0),
		}

		// 转换端点
		for _, tempEndpoint := range tempPluginInfo.Endpoints {
			endpoint := sharedmodels.PluginEndpoint{
				Path:         tempEndpoint.Path,
				Methods:      make([]string, 0),
				InputFormat:  tempEndpoint.InputFormat,
				OutputFormat: tempEndpoint.OutputFormat,
			}

			// 验证方法
			validMethods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
			for _, method := range tempEndpoint.Methods {
				isValid := false
				for _, validMethod := range validMethods {
					if method == validMethod {
						isValid = true
						break
					}
				}
				if isValid {
					endpoint.Methods = append(endpoint.Methods, method)
				}
			}

			// 如果没有有效方法，默认使用GET
			if len(endpoint.Methods) == 0 {
				endpoint.Methods = []string{"GET"}
			}

			pluginInfo.Endpoints = append(pluginInfo.Endpoints, endpoint)
		}

		// 添加到配置
		if h.configService.PluginsConfig.Plugins == nil {
			h.configService.PluginsConfig.Plugins = make(map[string]sharedmodels.PluginInfo)
		}
		h.configService.PluginsConfig.Plugins[pluginID] = pluginInfo
	}

	// 构建插件配置文件路径
	pluginsFilePath := filepath.Join(h.configService.BaseDir, "configs", "plugins.json")

	// 保存配置
	if err := utils.SavePluginsConfig(pluginsFilePath, h.configService.PluginsConfig); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save plugin configuration", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Plugins installed successfully",
		"plugins": h.configService.PluginsConfig.Plugins,
	})
}

// UpdateAllPluginsV1 更新全部插件列表 (V1 API)
func (h *PluginHandler) UpdateAllPluginsV1(c *gin.Context) {
	// 更新全部插件列表
	pluginsToUpdate := make(map[string]sharedmodels.PluginInfo)

	// 遍历所有插件，检查更新
	for pluginID, plugin := range h.configService.PluginsConfig.Plugins {
		// 获取源链接
		sourceLink := plugin.SourceLink
		if sourceLink == "" {
			log.Printf("插件 %s 未设置源链接，跳过更新", pluginID)
			pluginsToUpdate[pluginID] = plugin
			continue
		}

		// 从URL获取最新的插件配置
		pluginData, err := fetchFromURL(sourceLink)
		if err != nil {
			log.Printf("无法获取插件 %s 的配置: %v", pluginID, err)
			pluginsToUpdate[pluginID] = plugin
			continue
		}

		// 解析插件信息
		var pluginConfig map[string]interface{}
		if err := json.Unmarshal(pluginData, &pluginConfig); err != nil {
			log.Printf("插件 %s 配置格式错误: %v", pluginID, err)
			pluginsToUpdate[pluginID] = plugin
			continue
		}

		// 提取插件信息
		var updatedPlugin sharedmodels.PluginInfo
		if err := json.Unmarshal(pluginData, &updatedPlugin); err != nil {
			log.Printf("插件 %s 信息格式错误: %v", pluginID, err)
			pluginsToUpdate[pluginID] = plugin
			continue
		}

		// 比较版本号
		if updatedPlugin.Version == plugin.Version {
			log.Printf("插件 %s 已是最新版本: %s", pluginID, plugin.Version)
			pluginsToUpdate[pluginID] = plugin
			continue
		}

		log.Printf("插件 %s 有更新: %s -> %s", pluginID, plugin.Version, updatedPlugin.Version)

		// 验证插件数据完整性
		if updatedPlugin.Name == "" || updatedPlugin.Version == "" || updatedPlugin.Authors == "" {
			log.Printf("插件 %s 数据不完整", pluginID)
			pluginsToUpdate[pluginID] = plugin
			continue
		}

		// 确保支持的HTTP方法有效
		for i := range updatedPlugin.Endpoints {
			endpoint := &updatedPlugin.Endpoints[i]
			// 如果有非法方法，过滤掉
			validMethods := make([]string, 0)
			for _, method := range endpoint.Methods {
				for _, validMethod := range sharedmodels.ValidMethods {
					if method == validMethod {
						validMethods = append(validMethods, method)
						break
					}
				}
			}
			// 如果没有有效方法，默认使用GET
			if len(validMethods) == 0 {
				log.Printf("插件 %s 端点 %s 无效HTTP方法，默认使用GET", pluginID, endpoint.Path)
				endpoint.Methods = []string{"GET"}
			} else {
				endpoint.Methods = validMethods
			}
		}

		// 创建插件目录
		pluginsPath := filepath.Join(h.configService.BaseDir, utils.PluginsDir)
		pluginPath := filepath.Join(pluginsPath, pluginID)
		if err := os.MkdirAll(pluginPath, 0755); err != nil {
			log.Printf("无法创建插件 %s 目录: %v", pluginID, err)
			pluginsToUpdate[pluginID] = plugin
			continue
		}

		// 下载插件文件
		// 这里需要根据实际情况实现插件文件的下载逻辑
		// 与主题类似，可能需要处理插件结构信息中的文件

		// 更新插件信息
		pluginsToUpdate[pluginID] = updatedPlugin

		log.Printf("插件 %s 更新成功", pluginID)
	}

	// 更新全部插件列表
	h.configService.PluginsConfig.Plugins = pluginsToUpdate

	// 保存配置
	if err := h.configService.SavePluginsConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存插件配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "全部插件更新检查完成",
		"plugins": pluginsToUpdate,
	})
}

// UpdateSinglePluginV1 更新指定插件 (V1 API)
func (h *PluginHandler) UpdateSinglePluginV1(c *gin.Context) {
	var request struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	pluginID := request.ID

	// 检查插件是否存在
	plugin, exists := h.configService.PluginsConfig.Plugins[pluginID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在", "id": pluginID})
		return
	}

	// 获取源链接
	sourceLink := plugin.SourceLink
	if sourceLink == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件未设置源链接", "id": pluginID})
		return
	}

	// 从URL获取最新的插件配置
	pluginData, err := fetchFromURL(sourceLink)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法获取插件配置", "details": err.Error()})
		return
	}

	// 解析插件信息
	var pluginConfig map[string]interface{}
	if err := json.Unmarshal(pluginData, &pluginConfig); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件配置格式错误", "details": err.Error()})
		return
	}

	// 提取插件信息
	var updatedPlugin sharedmodels.PluginInfo
	if err := json.Unmarshal(pluginData, &updatedPlugin); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件信息格式错误", "details": err.Error()})
		return
	}

	// 比较版本号
	if updatedPlugin.Version == plugin.Version {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "插件已是最新版本",
			"plugin":  plugin,
		})
		return
	}

	// 验证插件数据完整性
	if updatedPlugin.Name == "" || updatedPlugin.Version == "" || updatedPlugin.Authors == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件数据不完整", "details": "缺少必要字段"})
		return
	}

	// 确保支持的HTTP方法有效
	for i := range updatedPlugin.Endpoints {
		endpoint := &updatedPlugin.Endpoints[i]
		// 如果有非法方法，过滤掉
		validMethods := make([]string, 0)
		for _, method := range endpoint.Methods {
			for _, validMethod := range sharedmodels.ValidMethods {
				if method == validMethod {
					validMethods = append(validMethods, method)
					break
				}
			}
		}
		// 如果没有有效方法，默认使用GET
		if len(validMethods) == 0 {
			endpoint.Methods = []string{"GET"}
		} else {
			endpoint.Methods = validMethods
		}
	}

	// 创建插件目录
	pluginsPath := filepath.Join(h.configService.BaseDir, utils.PluginsDir)
	pluginPath := filepath.Join(pluginsPath, pluginID)
	if err := os.MkdirAll(pluginPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法创建插件目录", "details": err.Error()})
		return
	}

	// 下载插件文件
	// 这里需要根据实际情况实现插件文件的下载逻辑
	// 与主题类似，可能需要处理插件结构信息中的文件

	// 更新插件信息
	h.configService.PluginsConfig.Plugins[pluginID] = updatedPlugin

	// 保存配置
	if err := h.configService.SavePluginsConfig(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "无法保存插件配置", "details": err.Error()})
		return
	}

	// 重启插件（如果正在运行）
	if plugin, exists := h.pluginManager.Plugins[pluginID]; exists && plugin.IsRunning {
		// 先停止
		if err := h.pluginManager.StopPlugin(pluginID); err != nil {
			log.Printf("停止插件 %s 失败: %v", pluginID, err)
		}
		// 卸载
		if err := h.pluginManager.UnloadPlugin(pluginID); err != nil {
			log.Printf("卸载插件 %s 失败: %v", pluginID, err)
		}
		// 重新加载
		if _, err := h.pluginManager.LoadPlugin(pluginID); err != nil {
			log.Printf("重新加载插件 %s 失败: %v", pluginID, err)
		} else {
			// 启动
			if err := h.pluginManager.StartPlugin(pluginID); err != nil {
				log.Printf("启动插件 %s 失败: %v", pluginID, err)
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "插件更新成功",
		"plugin":  updatedPlugin,
	})
}

// HandlePluginsV1 处理插件接口请求 (V1 API)
func (h *PluginHandler) HandlePluginsV1(c *gin.Context) {
	method := c.Request.Method
	switch method {
	case "GET":
		h.ListPluginsV1(c)
	case "POST":
		// 判断是安装新插件还是控制插件状态
		var request struct {
			URL    string `json:"url"`
			ID     string `json:"id"`
			Status string `json:"status"`
		}
		if err := c.ShouldBindJSON(&request); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": err.Error()})
			return
		}

		if request.URL != "" {
			// 安装新插件
			h.InstallPluginV1(c)
		} else if request.ID != "" && request.Status != "" {
			// 控制插件状态
			if request.Status == "start" {
				h.StartPluginV1(c)
			} else if request.Status == "stop" {
				h.StopPluginV1(c)
			} else {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的状态值", "details": "status只能是start或stop"})
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": "需要url或id+status"})
		}
	case "PUT":
		// 更新全部插件列表
		h.UpdateAllPluginsV1(c)
	case "PATCH":
		// 更新指定插件
		h.UpdateSinglePluginV1(c)
	case "DELETE":
		h.DeletePluginV1(c)
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "不支持的请求方法"})
	}
}

// fetchFromURL 从URL获取数据
func fetchFromURL(url string) ([]byte, error) {
	// 发送GET请求
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("无法从URL获取数据: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("服务器返回非200状态码: %s", resp.Status)
	}

	// 读取响应体
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应体失败: %w", err)
	}

	return data, nil
}

// ListPluginsV1 列出所有插件 (V1 API)
func (h *PluginHandler) ListPluginsV1(c *gin.Context) {
	// 获取所有插件
	plugins := h.configService.PluginsConfig.Plugins

	// 插件列表
	pluginsList := make([]map[string]interface{}, 0, len(plugins))

	// 遍历插件
	for id, pluginInfo := range plugins {
		// 获取插件状态
		status := "stopped"
		if h.pluginManager.IsPluginLoaded(id) {
			if h.pluginManager.IsPluginRunning(id) {
				status = "running"
			} else {
				status = "loaded"
			}
		}

		// 添加到列表
		pluginsList = append(pluginsList, map[string]interface{}{
			"id":          id,
			"name":        pluginInfo.Name,
			"description": pluginInfo.Description,
			"version":     pluginInfo.Version,
			"status":      status,
			"api_version": pluginInfo.APIVersion,
			"endpoints":   pluginInfo.Endpoints,
			"source_link": pluginInfo.SourceLink,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   len(pluginsList),
		"data":    pluginsList,
	})
}

// StartPluginV1 启动插件 (V1 API)
func (h *PluginHandler) StartPluginV1(c *gin.Context) {
	var request struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	pluginID := request.ID

	// 检查插件是否存在
	if _, exists := h.configService.PluginsConfig.Plugins[pluginID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在", "id": pluginID})
		return
	}

	// 如果插件没有加载，先加载
	if !h.pluginManager.IsPluginLoaded(pluginID) {
		if _, err := h.pluginManager.LoadPlugin(pluginID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载插件失败", "details": err.Error()})
			return
		}
	}

	// 启动插件
	if err := h.pluginManager.StartPlugin(pluginID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "启动插件失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "插件已启动",
		"id":      pluginID,
		"status":  "running",
	})
}

// StopPluginV1 停止插件 (V1 API)
func (h *PluginHandler) StopPluginV1(c *gin.Context) {
	var request struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数", "details": err.Error()})
		return
	}

	pluginID := request.ID

	// 检查插件是否存在
	if _, exists := h.configService.PluginsConfig.Plugins[pluginID]; !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在", "id": pluginID})
		return
	}

	// 检查插件是否已加载
	if !h.pluginManager.IsPluginLoaded(pluginID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件未加载", "id": pluginID})
		return
	}

	// 检查插件是否在运行
	if !h.pluginManager.IsPluginRunning(pluginID) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "插件未运行", "id": pluginID})
		return
	}

	// 停止插件
	if err := h.pluginManager.StopPlugin(pluginID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "停止插件失败", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "插件已停止",
		"id":      pluginID,
		"status":  "stopped",
	})
}

// DeletePluginV1 删除插件 (V1 API)
func (h *PluginHandler) DeletePluginV1(c *gin.Context) {
	var request struct {
		ID string `json:"id"`
	}

	if err := c.ShouldBindJSON(&request); err == nil && request.ID != "" {
		// 删除特定插件
		pluginID := request.ID

		// 检查插件是否存在
		if _, exists := h.configService.PluginsConfig.Plugins[pluginID]; !exists {
			c.JSON(http.StatusNotFound, gin.H{"error": "插件不存在", "id": pluginID})
			return
		}

		// 如果插件正在运行，先停止
		if h.pluginManager.IsPluginRunning(pluginID) {
			if err := h.pluginManager.StopPlugin(pluginID); err != nil {
				log.Printf("停止插件 %s 失败: %v", pluginID, err)
			}
		}

		// 如果插件已加载，卸载
		if h.pluginManager.IsPluginLoaded(pluginID) {
			// 使用通用方法卸载
			h.pluginManager.StopPlugin(pluginID)
		}

		// 删除插件目录
		pluginInfo := h.configService.PluginsConfig.Plugins[pluginID]
		if pluginInfo.Directory != "" {
			pluginDir := filepath.Join(h.configService.BaseDir, pluginInfo.Directory)
			if err := os.RemoveAll(pluginDir); err != nil {
				log.Printf("删除插件 %s 目录失败: %v", pluginID, err)
			}
		}

		// 从配置中删除插件
		delete(h.configService.PluginsConfig.Plugins, pluginID)

		// 保存配置
		if err := h.configService.SavePluginsConfig(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存插件配置失败", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "插件已删除",
			"id":      pluginID,
		})
	} else {
		// 删除所有插件
		for pluginID, pluginInfo := range h.configService.PluginsConfig.Plugins {
			// 如果插件正在运行，先停止
			if h.pluginManager.IsPluginRunning(pluginID) {
				if err := h.pluginManager.StopPlugin(pluginID); err != nil {
					log.Printf("停止插件 %s 失败: %v", pluginID, err)
				}
			}

			// 如果插件已加载，卸载
			if h.pluginManager.IsPluginLoaded(pluginID) {
				// 使用通用方法卸载
				h.pluginManager.StopPlugin(pluginID)
			}

			// 删除插件目录
			if pluginInfo.Directory != "" {
				pluginDir := filepath.Join(h.configService.BaseDir, pluginInfo.Directory)
				if err := os.RemoveAll(pluginDir); err != nil {
					log.Printf("删除插件 %s 目录失败: %v", pluginID, err)
				}
			}
		}

		// 清空插件列表
		h.configService.PluginsConfig.Plugins = make(map[string]sharedmodels.PluginInfo)

		// 保存配置
		if err := h.configService.SavePluginsConfig(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "保存插件配置失败", "details": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "所有插件已删除",
		})
	}
}
