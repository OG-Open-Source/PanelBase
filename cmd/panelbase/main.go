package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/OG-Open-Source/PanelBase/internal/app/handlers"
	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	"github.com/gin-gonic/gin"
)

func main() {
	// 尝试找到当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		log.Fatalf("获取工作目录失败: %v", err)
	}

	// 首先尝试直接使用当前工作目录下的配置文件
	configPath := filepath.Join(workDir, "configs", "config.yaml")
	log.Printf("尝试加载配置文件: %s", configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// 配置文件不在当前工作目录，尝试使用可执行文件所在目录
		exePath, err := os.Executable()
		if err != nil {
			log.Fatalf("获取可执行文件路径失败: %v", err)
		}
		exeDir := filepath.Dir(exePath)
		configPath = filepath.Join(exeDir, "configs", "config.yaml")
		log.Printf("尝试加载配置文件: %s", configPath)

		// 检查可执行文件目录中的配置文件
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			// 仍然找不到配置，尝试回退到上一级目录（开发环境可能会需要）
			configPath = filepath.Join(workDir, "..", "configs", "config.yaml")
			log.Printf("尝试加载配置文件: %s", configPath)

			if _, err := os.Stat(configPath); os.IsNotExist(err) {
				log.Fatalf("无法找到配置文件，已尝试以下路径:\n1. %s\n2. %s\n3. %s",
					filepath.Join(workDir, "configs", "config.yaml"),
					filepath.Join(exeDir, "configs", "config.yaml"),
					filepath.Join(workDir, "..", "configs", "config.yaml"))
			}
		}
	}

	log.Printf("使用配置文件: %s", configPath)

	// 加载配置
	configService, err := services.NewConfigService(configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 设置Gin模式
	gin.SetMode(configService.Config.Server.Mode)

	// 创建路由器
	router := gin.Default()

	// 设置路由
	handlers.SetupRoutes(router, configService)

	// 启动服务器
	serverAddr := fmt.Sprintf("%s:%d",
		configService.Config.Server.Host,
		configService.Config.Server.Port,
	)
	log.Printf("启动PanelBase服务器于 %s", serverAddr)
	log.Printf("模式: %s", configService.Config.Server.Mode)
	if err := router.Run(serverAddr); err != nil {
		log.Fatalf("启动服务器失败: %v", err)
	}
}
