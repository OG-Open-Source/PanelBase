package handlers

import (
	"fmt"
	"os"

	"github.com/OG-Open-Source/PanelBase/internal/app/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	"github.com/gin-gonic/gin"
)

// SetupRoutes 设置应用程序的路由
func SetupRoutes(router *gin.Engine, configService *services.ConfigService) {
	// 创建处理器
	authHandler := NewAuthHandler(configService)
	pluginHandler := NewPluginHandler(configService)
	commandHandler := NewCommandHandler(configService)
	systemHandler := NewSystemHandler(configService)

	// Auth相關路由
	router.POST("/api/v1/auth/login", authHandler.LoginHandler)

	// API 路由组 (需要认证)
	apiGroup := router.Group("/api/v1")
	apiGroup.Use(middleware.AuthMiddleware(configService))
	{
		// API Token管理
		apiGroup.POST("/auth/token", authHandler.CreateAPITokenHandler)

		// 系统相关API
		apiGroup.GET("/system/info", systemHandler.GetSystemInfo)
		apiGroup.GET("/system/status", systemHandler.GetSystemStatus)

		// 插件相关API
		apiGroup.GET("/plugins", pluginHandler.HandlePluginsV1)
		apiGroup.POST("/plugins", pluginHandler.HandlePluginsV1)
		apiGroup.PUT("/plugins", pluginHandler.HandlePluginsV1)
		apiGroup.PATCH("/plugins", pluginHandler.HandlePluginsV1)
		apiGroup.DELETE("/plugins", pluginHandler.HandlePluginsV1)

		// 插件API调用
		apiGroup.Any("/plugins/:plugin_id/*route", pluginHandler.HandlePluginAPI)

		// 命令相关API
		apiGroup.GET("/commands", commandHandler.HandleCommandsV1)
		apiGroup.POST("/commands", commandHandler.HandleCommandsV1)
		apiGroup.PUT("/commands", commandHandler.HandleCommandsV1)
		apiGroup.PATCH("/commands", commandHandler.HandleCommandsV1)
		apiGroup.DELETE("/commands", commandHandler.HandleCommandsV1)

		// 命令执行API
		apiGroup.POST("/execute", commandHandler.ExecuteHandler)
	}

	// 静态文件服务
	router.Static("/assets", "./web/assets")
	router.Static("/themes", "./web/themes")

	// 添加當前主題的靜態文件路由
	currentTheme := configService.ThemesConfig.CurrentTheme
	themeInfo, exists := configService.ThemesConfig.Themes[currentTheme]
	if exists {
		// 直接从主题目录提供 CSS 和 JS 文件
		router.StaticFile("/style.css", fmt.Sprintf("./web/%s/style.css", themeInfo.Directory))
		router.StaticFile("/script.js", fmt.Sprintf("./web/%s/script.js", themeInfo.Directory))
	} else {
		// 使用默认主题
		router.StaticFile("/style.css", "./web/default/style.css")
		router.StaticFile("/script.js", "./web/default/script.js")
	}

	// 默认路由到主页
	router.GET("/", func(c *gin.Context) {
		// 获取当前主题
		currentTheme := configService.ThemesConfig.CurrentTheme
		themeInfo, exists := configService.ThemesConfig.Themes[currentTheme]

		if !exists {
			// 如果当前主题不存在，使用默认主题
			themeInfo, exists = configService.ThemesConfig.Themes["default_theme"]
			if !exists {
				c.JSON(500, gin.H{
					"error": "无法找到任何主题",
				})
				return
			}
		}

		// 从主题目录获取 index.html
		themeDir := themeInfo.Directory
		indexPath := fmt.Sprintf("./web/%s/index.html", themeDir)

		// 检查文件是否存在
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			c.JSON(404, gin.H{
				"error": "主题首页不存在",
				"path":  indexPath,
			})
			return
		}

		// 直接显示主题首页
		c.File(indexPath)
	})

	// 主题模板文件
	router.StaticFile("/index.html", "./web/default/index.html")
	router.StaticFile("/login.html", "./web/default/login.html")
}
