package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/OG-Open-Source/PanelBase/pkg/executor"
)

type Handler struct {
	routeManager *utils.RouteManager
	themeManager *utils.ThemeManager
	metadata     *utils.MetadataManager
}

// NewHandler 創建新的處理器
func NewHandler(configPath string) *Handler {
	scriptsPath := filepath.Join("internal", "scripts")
	themesPath := filepath.Join("internal", "themes")
	routesConfigPath := filepath.Join(configPath, "routes.json")
	themesConfigPath := filepath.Join(configPath, "themes.json")

	// 創建 MetadataManager
	metadata := utils.NewMetadataManager(
		routesConfigPath,
		themesConfigPath,
		scriptsPath,
		themesPath,
	)

	return &Handler{
		routeManager: utils.NewRouteManager(routesConfigPath, scriptsPath),
		themeManager: utils.NewThemeManager(themesConfigPath, themesPath),
		metadata:     metadata,
	}
}

// Response 通用回應結構
type Response struct {
	Status  string      `json:"status"`  // "success" 或 "failure"
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

// SetupRoutes 設置所有路由
func (h *Handler) SetupRoutes(entry string) *mux.Router {
	r := mux.NewRouter()
	api := r.PathPrefix("/" + entry).Subrouter()

	// 連接測試
	api.HandleFunc("/connect", h.handleConnect).Methods("GET")

	// 執行腳本
	api.HandleFunc("/execute", h.executeHandler).Methods("POST")
	api.HandleFunc("/ws-execute", h.wsExecuteHandler)

	// 主題相關路由
	api.HandleFunc("/theme/install", h.themeInstallHandler).Methods("POST")
	api.HandleFunc("/theme/update", h.themeUpdateHandler).Methods("POST")
	api.HandleFunc("/theme/metadata", h.HandleThemeMetadata).Methods("GET")
	api.HandleFunc("/theme/delete", h.themeDeleteHandler).Methods("POST")

	// 路由相關路由
	api.HandleFunc("/route/install", h.routeInstallHandler).Methods("POST")
	api.HandleFunc("/route/update", h.routeUpdateHandler).Methods("POST")
	api.HandleFunc("/route/metadata", h.HandleRouteMetadata).Methods("GET")
	api.HandleFunc("/route/delete", h.routeDeleteHandler).Methods("POST")

	return r
}

// executeHandler 處理執行請求
func (h *Handler) executeHandler(w http.ResponseWriter, r *http.Request) {
	utils.Debug("Handling execute request")

	// 解析請求體
	var req executor.ExecuteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.Error("Failed to decode request body: [%v]", err)
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	// 檢查請求是否為空
	if len(req.Commands) == 0 {
		utils.Warn("Empty commands list received")
		h.respondJSON(w, Response{Status: "failure", Message: "No commands provided"})
		return
	}

	// 創建執行器，傳入腳本目錄和路由配置文件路徑
	scriptsPath := filepath.Join("internal", "scripts")
	routesConfigPath := filepath.Join("internal", "configs", "routes.json")
	exec := executor.NewExecutor(scriptsPath, routesConfigPath)

	// 檢查執行器是否創建成功
	if exec == nil {
		utils.Error("Failed to create executor")
		h.respondJSON(w, Response{Status: "failure", Message: "Internal server error"})
		return
	}

	// 執行指令
	output, err := exec.Execute(req)
	if err != nil {
		utils.Error("Execution failed: '%v'", err)
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	// 返回成功結果
	h.respondJSON(w, Response{
		Status:  "success",
		Message: "Commands executed successfully",
		Data:    output,
	})
}

func (h *Handler) wsExecuteHandler(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, Response{Status: "success", Message: "WebSocket endpoint is ready"})
}

// 主題處理函數
func (h *Handler) themeInstallHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	if err := h.themeManager.Install(req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	h.respondJSON(w, Response{Status: "success", Message: "Theme installed successfully"})
}

func (h *Handler) themeUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	if err := h.themeManager.Update(req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	h.respondJSON(w, Response{Status: "success", Message: "Theme updated successfully"})
}

func (h *Handler) themeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	if err := h.themeManager.Delete(req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	h.respondJSON(w, Response{Status: "success", Message: "Theme deleted successfully"})
}

// 路由處理函數
func (h *Handler) routeInstallHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	if err := h.routeManager.Install(req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	h.respondJSON(w, Response{Status: "success", Message: "Route installed successfully"})
}

func (h *Handler) routeUpdateHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	if err := h.routeManager.Update(req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	h.respondJSON(w, Response{Status: "success", Message: "Route updated successfully"})
}

func (h *Handler) routeDeleteHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	if err := h.routeManager.Delete(req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	h.respondJSON(w, Response{Status: "success", Message: "Route deleted successfully"})
}

// respondJSON 輔助函數，用於發送 JSON 響應
func (h *Handler) respondJSON(w http.ResponseWriter, response Response) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// HandleRouteMetadata 處理路由元數據請求
func (h *Handler) HandleRouteMetadata(w http.ResponseWriter, r *http.Request) {
	utils.Debug("Handling route metadata request")
	if r.Method != http.MethodGet {
		h.respondJSON(w, Response{Status: "failure", Message: "Method not allowed"})
		return
	}

	var req struct {
		URL    string `json:"url,omitempty"`
		Script string `json:"script,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Either URL or Script must be provided"})
		return
	}

	// 如果提供了 URL，從遠程獲取
	if req.URL != "" {
		metadata, err := h.metadata.FetchRouteMetadata(req.URL, true)
		if err != nil {
			h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
			return
		}
		h.respondJSON(w, Response{Status: "success", Message: "Metadata fetched successfully", Data: metadata})
		return
	}

	// 如果提供了 Script，從本地獲取
	if req.Script != "" {
		metadata, err := h.metadata.FetchRouteMetadata(req.Script, false)
		if err != nil {
			h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
			return
		}
		h.respondJSON(w, Response{Status: "success", Message: "Metadata fetched successfully", Data: metadata})
		return
	}

	h.respondJSON(w, Response{Status: "failure", Message: "Either URL or Script must be provided"})
}

// HandleThemeMetadata 處理主題元數據請求
func (h *Handler) HandleThemeMetadata(w http.ResponseWriter, r *http.Request) {
	utils.Debug("Handling theme metadata request")
	if r.Method != http.MethodGet {
		h.respondJSON(w, Response{Status: "failure", Message: "Method not allowed"})
		return
	}

	var req struct {
		URL  string `json:"url,omitempty"`
		Name string `json:"name,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Either URL or Name must be provided"})
		return
	}

	// 如果提供了 URL，從遠程獲取
	if req.URL != "" {
		metadata, err := h.metadata.FetchThemeMetadata(req.URL, true)
		if err != nil {
			h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
			return
		}
		h.respondJSON(w, Response{Status: "success", Message: "Metadata fetched successfully", Data: metadata})
		return
	}

	// 如果提供了 Name，從本地獲取
	if req.Name != "" {
		metadata, err := h.metadata.FetchThemeMetadata(req.Name, false)
		if err != nil {
			h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
			return
		}
		h.respondJSON(w, Response{Status: "success", Message: "Metadata fetched successfully", Data: metadata})
		return
	}

	h.respondJSON(w, Response{Status: "failure", Message: "Either URL or Name must be provided"})
}

// handleConnect 處理連接測試請求
func (h *Handler) handleConnect(w http.ResponseWriter, r *http.Request) {
	utils.Debug("Handling connect request")
	h.respondJSON(w, Response{Status: "success", Message: "Connected successfully"})
}