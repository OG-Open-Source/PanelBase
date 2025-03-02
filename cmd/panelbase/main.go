package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/OG-Open-Source/PanelBase/internal/handlers"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
)

func main() {
	// 初始化日誌系統
	if err := utils.InitLogger(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer utils.CloseLogger()

	// 載入環境變數
	err := godotenv.Load()
	if err != nil {
		utils.Error("Error loading .env file: %v", err)
		os.Exit(1)
	}

	// 獲取 API 路徑
	entry := os.Getenv("ENTRY")
	if entry == "" {
		entry = "api"
		utils.Warn("ENTRY not set, using default: %s", entry)
	}

	// 獲取端口
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		utils.Warn("PORT not set, using default: %s", port)
	}

	// 獲取 IP
	ip := os.Getenv("IP")
	if ip == "" {
		ip = "0.0.0.0"
		utils.Warn("IP not set, using default: %s", ip)
	}

	// 創建配置目錄路徑
	configPath := filepath.Join("internal", "configs")

	// 創建處理器
	handler := handlers.NewHandler(configPath)

	// 設置路由
	router := handler.SetupRoutes(entry)

	// 啟動服務器
	addr := fmt.Sprintf("%s:%s", ip, port)
	utils.Info("PanelBase starting on http://%s:%s/%s", ip, port, entry)
	err = http.ListenAndServe(addr, router)
	if err != nil {
		utils.Error("Server error: %v", err)
		os.Exit(1)
	}
}