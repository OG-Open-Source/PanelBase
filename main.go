package main

import (
	"fmt"
	"log"
	"net/http"
	"PanelBase/config"
	"PanelBase/api"
)

func main() {
	// 加載配置
	cfg := config.LoadConfig()

	// 設置路由
	router := api.SetupRoutes()

	// 啟動服務
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("PanelBase agent 正在運行在 http://%s:%d/%s\n", cfg.IP, cfg.Port, cfg.SecurityEntry)
	log.Fatal(http.ListenAndServe(addr, router))
}