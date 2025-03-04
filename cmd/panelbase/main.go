package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/server"
	"github.com/OG-Open-Source/PanelBase/internal/utils"
)

func main() {
	// 初始化日誌
	logger.Init()
	defer logger.Cleanup()

	// 設置信號處理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 加載配置
	cfg, err := config.Load()
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to load config: %v", err))
	}
	logger.Info("Configuration loaded successfully")

	// 生成 JWT Token
	token, err := utils.GenerateToken(cfg)
	if err != nil {
		logger.Fatal(fmt.Sprintf("Failed to generate token: %v", err))
	}
	logger.Info("Generated JWT Token for API access:")
	logger.Info(token)
	
	// 創建並啟動服務器
	srv := server.New(cfg)
	logger.Info(fmt.Sprintf("Starting server on %s:%d", cfg.IP, cfg.Port))
	logger.Info(fmt.Sprintf("Entry point set to /%s", cfg.EntryPoint))

	// 在新的 goroutine 中啟動服務器
	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal(fmt.Sprintf("Server failed: %v", err))
		}
	}()

	// 等待信號
	<-sigChan
	logger.Info("Shutting down server...")
} 