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
	// 確保程序結束時關閉日誌
	defer utils.Close()

	// 載入環境變數
	if err := godotenv.Load(); err != nil {
		utils.Fatal("Error loading .env file")
	}

	// 獲取必要的環境變數
	ip := os.Getenv("IP")
	port := os.Getenv("PORT")
	entry := os.Getenv("ENTRY")

	if ip == "" || port == "" || entry == "" {
		utils.Fatal("Missing required environment variables: IP, PORT, ENTRY")
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
	if err := http.ListenAndServe(addr, router); err != nil {
		utils.Fatal(err.Error())
	}
}