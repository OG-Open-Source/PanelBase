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
	router.HandleFunc("/{securityEntry}/login", h.loginHandler).Methods("POST")
}

func (h *ExternalHandler) checkAccess(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取并验证cookie中的设置
		settingsCookie, err := r.Cookie("panelbase_settings")
		if err != nil || !validateSettings(settingsCookie.Value) {
			sendJSONResponse(w, "error", "Invalid session", http.StatusUnauthorized)
			return
		}

		// 其他验证逻辑...
		next.ServeHTTP(w, r)
	})
}

func validateSettings(settings string) bool {
	// 实现具体的验证逻辑
	return true
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

	sendJSONResponse(w, "success", map[string]interface{}{
		"message": "System status retrieved",
		"data":    data,
	}, http.StatusOK)
}

func (h *ExternalHandler) commandHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string   `json:"command"`
		Args    []string `json:"args"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, "error", map[string]interface{}{
			"output": "Invalid request format",
		}, http.StatusBadRequest)
		return
	}

	startTime := time.Now()
	output, err := h.routeManager.ExecuteCommand(req.Command, req.Args)
	executionTime := time.Since(startTime).String()

	if err != nil {
		sendJSONResponse(w, "error", map[string]interface{}{
			"output": err.Error(),
		}, http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", map[string]interface{}{
		"output":         output,
		"execution_time": executionTime,
	}, http.StatusOK)
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
		sendJSONResponse(w, "error", "Failed to retrieve route information", http.StatusInternalServerError)
		return
	}

	var routes interface{}
	if err := json.Unmarshal(data, &routes); err != nil {
		sendJSONResponse(w, "error", "Failed to parse routes file", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", map[string]interface{}{
		"message": "Route list retrieved",
		"routes":  routes,
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
		sendJSONResponse(w, "error", fmt.Sprintf("Theme installation error: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", "Theme package installed successfully", http.StatusOK)
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
		sendJSONResponse(w, "error", fmt.Sprintf("Route installation error: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", "Route package installed successfully", http.StatusOK)
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
		sendJSONResponse(w, "error", "Failed to fetch route metadata", http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, "success", map[string]interface{}{
		"message":  "Route metadata retrieved",
		"metadata": metadata,
	}, http.StatusOK)
}

func (h *ExternalHandler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Address  string `json:"address"`
		Port     string `json:"port"`
		Entrance string `json:"entrance"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONResponse(w, "error", "Invalid request", http.StatusBadRequest)
		return
	}

	// 验证逻辑(可扩展)
	if req.Address == "" || req.Port == "" || req.Entrance == "" {
		sendJSONResponse(w, "error", "Missing parameters", http.StatusBadRequest)
		return
	}

	sendJSONResponse(w, "success", map[string]interface{}{
		"message": "Login successful",
		"dashboard": fmt.Sprintf("/dashboard?address=%s&port=%s&entrance=%s", 
			req.Address, req.Port, req.Entrance),
	}, http.StatusOK)
}

func sendJSONResponse(w http.ResponseWriter, status string, data interface{}, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	
	response := map[string]interface{}{
		"status": status,
	}
	
	switch v := data.(type) {
	case string:
		response["message"] = v
	case map[string]interface{}:
		for key, val := range v {
			if key == "execution_time" {
				response["execution_time"] = val
				continue
			}
			response[key] = val
		}
	default:
		response["data"] = v
	}
	
	json.NewEncoder(w).Encode(response)
}