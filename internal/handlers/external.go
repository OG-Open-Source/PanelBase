package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
)

type ExternalHandler struct {
	routeManager *utils.RouteManager
}

type APIResponse struct {
	Output  string `json:"output,omitempty"`
	Message string `json:"message,omitempty"`
	Status  string `json:"status"`
}

func NewExternalHandler() *ExternalHandler {
	return &ExternalHandler{
		routeManager: utils.NewRouteManager(),
	}
}

func (h *ExternalHandler) Init() error {
	if err := h.routeManager.LoadRoutes("internal/config/routes.json"); err != nil {
		utils.Log(utils.EROR, "Failed to initialize handler: %v", err)
		return err
	}
	return nil
}

func (h *ExternalHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	remoteIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		remoteIP = strings.Split(forwarded, ",")[0]
	}
	clientIP := strings.Split(remoteIP, ":")[0]


	utils.Log(utils.INFO, "[%s] %s %s", clientIP, r.Method, r.URL.Path)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	apiPath := strings.TrimPrefix(r.URL.Path, "/"+os.Getenv("ENTRY")+"/")

	if apiPath == "connect" {
		h.handleConnect(w, r)
		return
	}

	route := h.routeManager.GetRoute(apiPath)
	if route == nil {
		utils.Log(utils.WARN, "Route not found: %s", apiPath)
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	switch r.Method {
	case "GET":
		h.handleGet(w, r, route)
	case "POST":
		h.handlePost(w, r, route)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

func (h *ExternalHandler) handleGet(w http.ResponseWriter, r *http.Request, route *utils.Route) {
	args := make(map[string]string)
	for key, values := range r.URL.Query() {
		if len(values) > 0 {
			args[key] = values[0]
		}
	}

	output, err := h.routeManager.ExecuteCommand(route, args)
	if err != nil {
		utils.Log(utils.EROR, "Command execution failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Message: "Internal Server Error",
			Status:  "error",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Output: output,
		Status: "success",
	})
}

func (h *ExternalHandler) handlePost(w http.ResponseWriter, r *http.Request, route *utils.Route) {
	var args map[string]string
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		utils.Log(utils.EROR, "Failed to parse request body: %v", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	output, err := h.routeManager.ExecuteCommand(route, args)
	if err != nil {
		utils.Log(utils.EROR, "Command execution failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(APIResponse{
			Message: "Internal Server Error",
			Status:  "error",
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Output: output,
		Status: "success",
	})
}

func (h *ExternalHandler) handleConnect(w http.ResponseWriter, r *http.Request) {
	remoteIP := r.RemoteAddr
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		remoteIP = strings.Split(forwarded, ",")[0]
	}
	clientIP := strings.Split(remoteIP, ":")[0]

	utils.Log(utils.INFO, "[%s] Connection attempt", clientIP)

	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == "OPTIONS" {
		return
	}

	response := APIResponse{
		Message: "Successfully connected to PanelBase",
		Status:  "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// TODO: Implement external request handlers