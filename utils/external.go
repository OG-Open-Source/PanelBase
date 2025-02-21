package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"os"
	"runtime"
	"time"
	"strings"
)

var (
	startTime      = time.Now()
	lastUpdateTime = time.Now()
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
	router.Use(h.checkAccess)
	router.HandleFunc("/{securityEntry}/status", h.statusHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/command", h.commandHandler).Methods("POST")
	router.HandleFunc("/{securityEntry}/routes", h.getRoutesHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/theme/install", h.installThemeHandler).Methods("POST")
	router.HandleFunc("/{securityEntry}/route/install", h.installRouteHandler).Methods("POST")
	router.HandleFunc("/{securityEntry}/route/metadata", h.getRouteMetadataHandler).Methods("POST")
}

func (h *ExternalHandler) checkAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clientIP := strings.Split(r.RemoteAddr, ":")[0]

		allowedIPs := map[string]bool{
			"127.0.0.1": true,
			"localhost": true,
		}

		host := r.Host

		if !allowedIPs[clientIP] && host != "panel.ogtt.tk" {
			sendJSONResponse(w, "error", "Access denied", http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *ExternalHandler) statusHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	data := map[string]interface{}{
		"version":     "0.1.0.1",
		"hostname":    hostname,
		"go_version":  runtime.Version(),
		"cpu_cores":   runtime.NumCPU(),
		"memory":      memStats.Alloc,
		"uptime":      time.Since(startTime).String(),
		"last_update": lastUpdateTime.Format(time.RFC3339),
	}

	sendJSONResponse(w, "success", data, http.StatusOK)
}

func (h *ExternalHandler) commandHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, "error", "Invalid request", http.StatusBadRequest)
		return
	}

	output, err := h.routeManager.ExecuteCommand(req.Command, req.Args)
	if err != nil {
		sendJSONResponse(w, "error", err.Error(), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", output, http.StatusOK)
}

// 判斷是否為PanelBase管理操作
func isManagementCommand(command string) bool {
	managementCommands := map[string]bool{
		"install/theme": true,
		"install/route": true,
		"update/theme":  true,
		"update/route":  true,
	}
	return managementCommands[command]
}

func (h *ExternalHandler) getRoutesHandler(w http.ResponseWriter, r *http.Request) {
	data, err := h.routeManager.GetRoutes()
	if err != nil {
		sendJSONResponse(w, "error", "Failed to read routes file", http.StatusInternalServerError)
		return
	}

	var routes interface{}
	if err := json.Unmarshal(data, &routes); err != nil {
		sendJSONResponse(w, "error", "Failed to parse routes file", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", map[string]interface{}{
		"routes": routes,
	}, http.StatusOK)
}

func (h *ExternalHandler) installThemeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, "error", "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.themeManager.InstallTheme(req.URL); err != nil {
		sendJSONResponse(w, "error", fmt.Sprintf("Theme installation failed: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", "Theme installed successfully", http.StatusOK)
}

func (h *ExternalHandler) installRouteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, "error", "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.routeManager.InstallRoute(req.URL); err != nil {
		sendJSONResponse(w, "error", fmt.Sprintf("Route installation failed: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", "Route installed successfully", http.StatusOK)
}

func (h *ExternalHandler) getRouteMetadataHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, "error", "Invalid request", http.StatusBadRequest)
		return
	}

	metadata, err := h.routeManager.GetRouteMetadata(req.URL)
	if err != nil {
		sendJSONResponse(w, "error", fmt.Sprintf("無法獲取路由指令文件元數據: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", map[string]interface{}{
		"metadata": metadata,
	}, http.StatusOK)
}

func sendJSONResponse(w http.ResponseWriter, status string, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": status,
		"data":   data,
	})
}