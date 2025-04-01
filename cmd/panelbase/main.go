package main

import (
	"fmt"
	"log"

	//"net/http" // No longer needed here directly for routes

	// Import the config package
	"github.com/OG-Open-Source/PanelBase/internal/bootstrap"
	"github.com/OG-Open-Source/PanelBase/internal/config" // Use the correct module path
	"github.com/OG-Open-Source/PanelBase/internal/routes" // Import the routes package
	"github.com/gin-gonic/gin"
)

func main() {
	// 初始化配置文件
	if err := bootstrap.Bootstrap(); err != nil {
		log.Printf("Warning: Failed to bootstrap configs: %v", err)
	}

	// 加载配置
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 设置 Gin 模式
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 创建 Gin 引擎
	router := gin.Default()

	// 设置路由
	routes.SetupRoutes(router, cfg)

	// 构建服务器地址
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	// 启动服务器
	log.Printf("Server starting on %s", addr)
	if err := router.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
