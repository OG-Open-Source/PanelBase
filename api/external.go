package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/mux"
)

type ExternalHandler struct {
	themeManager  *ThemeManager
	routeManager  *RouteManager
}

func NewExternalHandler(themeManager *ThemeManager, routeManager *RouteManager) *ExternalHandler {
	return &ExternalHandler{
		themeManager: themeManager,
		routeManager: routeManager,
	}
}

func (h *ExternalHandler) SetupRoutes(router *mux.Router) {
	// 對外接口
	router.HandleFunc("/{securityEntry}/status", h.statusHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/command", h.commandHandler).Methods("POST")
	router.HandleFunc("/{securityEntry}/routes", h.getRoutesHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/install/theme", h.installThemeHandler).Methods("POST")
	router.HandleFunc("/{securityEntry}/install/route", h.installRouteHandler).Methods("POST")
}

func (h *ExternalHandler) statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("PanelBase agent 正在運行"))
}

func (h *ExternalHandler) commandHandler(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無效的請求", http.StatusBadRequest)
		return
	}

	// 執行命令
	output, err := h.routeManager.ExecuteCommand(req.Command, req.Args)
	if err != nil {
		http.Error(w, fmt.Sprintf("命令執行失敗: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(output))
}

func (h *ExternalHandler) getRoutesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := h.routeManager.GetRoutes()
	if err != nil {
		http.Error(w, "無法讀取路由文件", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *ExternalHandler) installThemeHandler(w http.ResponseWriter, r *http.Request) {
	var req ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無效的請求", http.StatusBadRequest)
		return
	}

	if err := h.themeManager.InstallTheme(req.URL); err != nil {
		http.Error(w, fmt.Sprintf("主題安裝失敗: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("主題安裝成功"))
}

func (h *ExternalHandler) installRouteHandler(w http.ResponseWriter, r *http.Request) {
	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無效的請求", http.StatusBadRequest)
		return
	}

	if err := h.routeManager.InstallRoute(req.URL); err != nil {
		http.Error(w, fmt.Sprintf("路由指令文件安裝失敗: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("路由指令文件安裝成功"))
} 