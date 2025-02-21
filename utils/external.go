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
			sendGeneralResponse(w, "error", "Access denied")
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (h *ExternalHandler) statusHandler(w http.ResponseWriter, r *http.Request) {
	hostname, _ := os.Hostname()
	goVersion := runtime.Version()
	numCPU := runtime.NumCPU()
	memStats := &runtime.MemStats{}
	runtime.ReadMemStats(memStats)

	response := map[string]interface{}{
		"status":      "running",
		"version":     "0.1.0.1",
		"hostname":    hostname,
		"go_version":  goVersion,
		"cpu_cores":   numCPU,
		"memory_usage": map[string]interface{}{
			"alloc":      memStats.Alloc,
			"total_alloc": memStats.TotalAlloc,
			"sys":        memStats.Sys,
			"num_gc":     uint64(memStats.NumGC),
		},
		"uptime":      time.Since(startTime).String(),
		"last_update": lastUpdateTime.Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Unable to encode JSON response", http.StatusInternalServerError)
	}
}

func (h *ExternalHandler) commandHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendCommandResponse(w, "error", "Invalid request")
		return
	}

	output, err := h.routeManager.ExecuteCommand(req.Command, req.Args)
	if err != nil {
		sendCommandResponse(w, "error", output)
		return
	}

	sendCommandResponse(w, "success", output)
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
		sendJSONError(w, "Failed to read routes file", http.StatusInternalServerError)
		return
	}

	var routes interface{}
	if err := json.Unmarshal(data, &routes); err != nil {
		sendJSONError(w, "Failed to parse routes file", http.StatusInternalServerError)
		return
	}

	sendGeneralResponse(w, "success", "Routes retrieved successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"routes": routes,
	})
}

func (h *ExternalHandler) installThemeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendGeneralResponse(w, "error", "Invalid request")
		return
	}

	if err := h.themeManager.InstallTheme(req.URL); err != nil {
		sendGeneralResponse(w, "error", fmt.Sprintf("Theme installation failed: %v", err))
		return
	}

	sendGeneralResponse(w, "success", "Theme installed successfully")
}

func (h *ExternalHandler) installRouteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.routeManager.InstallRoute(req.URL); err != nil {
		sendJSONError(w, fmt.Sprintf("Route installation failed: %v", err), http.StatusInternalServerError)
		return
	}

	sendGeneralResponse(w, "success", "Route installed successfully")
}

func (h *ExternalHandler) getRouteMetadataHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendGeneralResponse(w, "error", "Invalid request")
		return
	}

	metadata, err := h.routeManager.GetRouteMetadata(req.URL)
	if err != nil {
		sendGeneralResponse(w, "error", fmt.Sprintf("無法獲取路由指令文件元數據: %v", err))
		return
	}

	sendGeneralResponse(w, "success", "Route metadata retrieved successfully")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"metadata": metadata,
	})
}

// 用於 /{securityEntry}/command 路徑的響應模板
func sendCommandResponse(w http.ResponseWriter, status string, output string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"output":  output,
	})
}

// 用於其他路徑的響應模板
func sendGeneralResponse(w http.ResponseWriter, status string, message string) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  status,
		"message": message,
	})
}

func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}