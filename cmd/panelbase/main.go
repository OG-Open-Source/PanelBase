package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/OG-Open-Source/PanelBase/internal/handlers"
)

// logMessage 格式化日誌訊息
func logMessage(message string) string {
	return fmt.Sprintf("%s | %s", 
		time.Now().UTC().Format("2006-01-02T15:04:05.000Z"),
		message,
	)
}

func init() {
	// 設置 log 的輸出格式，移除預設的時間戳
	log.SetFlags(0)
}

func main() {
	// 載入環境變數
	if err := godotenv.Load(); err != nil {
		log.Fatal(logMessage("Error loading .env file"))
	}

	// 獲取必要的環境變數
	ip := os.Getenv("IP")
	port := os.Getenv("PORT")
	entry := os.Getenv("ENTRY")

	if ip == "" || port == "" || entry == "" {
		log.Fatal(logMessage("Missing required environment variables: IP, PORT, ENTRY"))
	}

	// 創建配置目錄路徑
	configPath := filepath.Join("internal", "configs")

	// 創建處理器
	handler := handlers.NewHandler(configPath)

	// 設置路由
	router := handler.SetupRoutes(entry)

	// 啟動服務器
	addr := fmt.Sprintf("%s:%s", ip, port)
	log.Print(logMessage(fmt.Sprintf("PanelBase starting on http://%s:%s/%s", ip, port, entry)))
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(logMessage(err.Error()))
	}
}