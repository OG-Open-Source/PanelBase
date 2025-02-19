package main

import (
	"fmt"
	"log"
	"net/http"
	"PanelBase/config"
	"PanelBase/utils"
	"github.com/gorilla/mux"
)

func main() {
	cfg := config.LoadConfig()

	routeManager := utils.NewRouteManager()
	themeManager := utils.NewThemeManager(routeManager)

	externalHandler := utils.NewExternalHandler(themeManager, routeManager)

	router := mux.NewRouter()

	externalHandler.SetupRoutes(router)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("PanelBase agent is running on http://%s:%d/%s\n", cfg.IP, cfg.Port, cfg.SecurityEntry)
	log.Fatal(http.ListenAndServe(addr, router))
}