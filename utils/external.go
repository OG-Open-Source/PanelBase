package utils

import (
	"encoding/json"
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"os"
	"runtime"
	"time"
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
	router.HandleFunc("/{securityEntry}/status", h.statusHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/command", h.commandHandler).Methods("POST")
	router.HandleFunc("/{securityEntry}/routes", h.getRoutesHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/install/theme", h.installThemeHandler).Methods("POST")
	router.HandleFunc("/{securityEntry}/install/route", h.installRouteHandler).Methods("POST")
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
		sendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	output, err := h.routeManager.ExecuteCommand(req.Command, req.Args)
	if err != nil {
		sendJSONError(w, fmt.Sprintf("Command execution failed: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]interface{}{
		"status":  "success",
		"output":  output,
	})
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

	sendJSONResponse(w, map[string]interface{}{
		"status":  "success",
		"routes":  routes,
	})
}

func (h *ExternalHandler) installThemeHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendJSONError(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := h.themeManager.InstallTheme(req.URL); err != nil {
		sendJSONError(w, fmt.Sprintf("Theme installation failed: %v", err), http.StatusInternalServerError)
		return
	}

	sendJSONResponse(w, map[string]interface{}{
		"status":  "success",
		"message": "Theme installed successfully",
	})
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

	sendJSONResponse(w, map[string]interface{}{
		"status":  "success",
		"message": "Route installed successfully",
	})
}

func sendJSONResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Failed to encode JSON response", http.StatusInternalServerError)
	}
}

func sendJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "error",
		"message": message,
	})
}