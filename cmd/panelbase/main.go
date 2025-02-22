package main

import (
	"fmt"
	"log"
	"net/http"
	"github.com/OG-Open-Source/PanelBase/config"
	"github.com/OG-Open-Source/PanelBase/utils"
	"github.com/gorilla/mux"
	"github.com/OG-Open-Source/PanelBase/internal/commands/clean"
	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func main() {
	// 初始化日誌
	logger.SetFormatter(&logrus.JSONFormatter{})
	logger.Info("Starting PanelBase agent")

	cfg := config.LoadConfig()

	// 執行清理命令示例
	if err := clean.Execute(nil); err != nil {
		panic(err)
	}

	routeManager := utils.NewRouteManager()
	themeManager := utils.NewThemeManager(routeManager)

	externalHandler := utils.NewExternalHandler(themeManager, routeManager)

	router := mux.NewRouter()

	externalHandler.SetupRoutes(router)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("PanelBase agent is running on http://%s:%d/%s\n", cfg.IP, cfg.Port, cfg.SecurityEntry)
	log.Fatal(http.ListenAndServe(addr, router))
}