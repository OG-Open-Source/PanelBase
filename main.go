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

	// 初始化路由管理器和主題管理器
	routeManager := api.NewRouteManager()
	themeManager := api.NewThemeManager(routeManager)

	// 初始化對外接口
	externalHandler := api.NewExternalHandler(themeManager, routeManager)

	// 設置路由
	router := externalHandler.SetupRoutes()

	// 啟動服務
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("PanelBase agent 正在運行在 http://%s:%d/%s\n", cfg.IP, cfg.Port, cfg.SecurityEntry)
	log.Fatal(http.ListenAndServe(addr, router))
}