package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"os"
	"strings"
	"fmt"

	"github.com/gorilla/mux"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/OG-Open-Source/PanelBase/pkg/executor"
)

type Handler struct {
	routeManager *utils.RouteManager
	themeManager *utils.ThemeManager
	metadata     *utils.MetadataManager
	wsManager    *utils.WebSocketManager
}

// NewHandler 創建新的處理器
func NewHandler(configPath string) *Handler {
	// 打印路徑以進行調試
	utils.Debug("Config path: '%s'", configPath)
	
	scriptsPath := filepath.Join("internal", "scripts")
	themesPath := filepath.Join("internal", "themes")
	routesConfigPath := filepath.Join(configPath, "routes.json")
	themesConfigPath := filepath.Join(configPath, "themes.json")

	// 打印所有路徑以進行調試
	utils.Debug("Scripts path: '%s'", scriptsPath)
	utils.Debug("Themes path: '%s'", themesPath)
	utils.Debug("Routes config path: '%s'", routesConfigPath)
	utils.Debug("Themes config path: '%s'", themesConfigPath)

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
		wsManager:    utils.NewWebSocketManager(),
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
	
	// 添加根目錄的 main.js 路由
	r.HandleFunc("/main.js", h.mainJSHandler).Methods("GET")

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

	// 靜態文件服務
	api.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		utils.Debug("Handling static file request: %s", r.URL.Path)
		
		// 獲取當前使用的主題
		theme := os.Getenv("THEME")
		utils.Debug("Current theme: %s", theme)
		
		if theme == "" {
			// 如果沒有設置主題，檢查是否有可用主題
			themesDir := filepath.Join("internal", "themes")
			entries, err := os.ReadDir(themesDir)
			if err != nil {
				utils.Error("Failed to read themes directory: %v", err)
				http.Error(w, "No theme available", http.StatusInternalServerError)
				return
			}
			if len(entries) == 0 {
				utils.Error("No themes found in directory")
				http.Error(w, "No theme available", http.StatusInternalServerError)
				return
			}
			// 使用第一個主題作為默認主題
			theme = entries[0].Name()
			utils.Debug("Using default theme: %s", theme)
			// 更新 .env 文件
			if err := h.updateEnvTheme(theme); err != nil {
				utils.Error("Failed to set default theme: %v", err)
				http.Error(w, "Failed to set default theme", http.StatusInternalServerError)
				return
			}
		}

		// 處理根路徑請求
		path := strings.TrimPrefix(r.URL.Path, "/"+entry)
		if path == "" || path == "/" {
			path = "/index.html"
		}
		utils.Debug("Processed path: %s", path)

		// 如果請求的是 main.js，直接使用根目錄的文件
		if strings.HasSuffix(path, "/main.js") {
			http.ServeFile(w, r, "main.js")
			return
		}

		// 從主題目錄提供其他文件
		filePath := filepath.Join("internal", "themes", theme, strings.TrimPrefix(path, "/"))
		utils.Debug("Serving file: %s", filePath)

		http.ServeFile(w, r, filePath)
	})

	return r
}

// updateEnvTheme 更新 .env 文件中的主題設置
func (h *Handler) updateEnvTheme(theme string) error {
	envFile := ".env"
	
	// 讀取當前的 .env 文件
	content, err := os.ReadFile(envFile)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	lines := strings.Split(string(content), "\n")
	themeFound := false
	
	// 更新 THEME 變量
	for i, line := range lines {
		if strings.HasPrefix(line, "THEME=") {
			lines[i] = "THEME=" + theme
			themeFound = true
			break
		}
	}

	// 如果沒有找到 THEME 變量，添加它
	if !themeFound {
		lines = append(lines, "THEME="+theme)
	}

	// 寫回文件
	return os.WriteFile(envFile, []byte(strings.Join(lines, "\n")), 0644)
}

// executeHandler 處理執行請求
func (h *Handler) executeHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	// 創建命令執行器，使用默認的腳本和路由配置路徑
	scriptsPath := filepath.Join("internal", "scripts")
	routesConfigPath := filepath.Join("internal", "configs", "routes.json")
	exec := executor.NewExecutor(scriptsPath, routesConfigPath)
	
	// 設置輸出回調
	exec.SetOutputCallback(func(output string) {
		// 實時發送命令輸出
		h.wsManager.Broadcast(utils.WebSocketMessage{
			Status:  "running",
			Data:    output,
			Command: req.Commands[0].Args[0],
		})
	})

	// 轉換請求格式
	execReq := executor.ExecuteRequest{
		Commands: make([]executor.Command, len(req.Commands)),
	}
	for i, cmd := range req.Commands {
		execReq.Commands[i] = executor.Command{
			Name: cmd.Name,
			Args: cmd.Args,
		}
	}

	// 執行命令
	output, err := exec.Execute(execReq)
	
	// 發送最終結果
	if err != nil {
		h.wsManager.Broadcast(utils.WebSocketMessage{
			Status:  "failure",
			Message: err.Error(),
			Command: req.Commands[0].Args[0],
		})
		return
	}

	h.wsManager.Broadcast(utils.WebSocketMessage{
		Status:  "success",
		Message: "Command executed successfully",
		Data:    output,
		Command: req.Commands[0].Args[0],
	})
}

func (h *Handler) wsExecuteHandler(w http.ResponseWriter, r *http.Request) {
	if err := h.wsManager.HandleConnection(w, r); err != nil {
		utils.Error("WebSocket connection failed: %v", err)
		return
	}
}

// themeInstallHandler 處理主題安裝
func (h *Handler) themeInstallHandler(w http.ResponseWriter, r *http.Request) {
	var req utils.ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Invalid request body"})
		return
	}

	// 從 URL 中提取主題配置
	themeConfig, err := h.metadata.FetchThemeMetadata(req.URL, true)
	if err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	// 檢查主題是否已存在
	currentConfig, err := h.metadata.LoadThemeConfig()
	if err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Failed to load theme config"})
		return
	}

	// 使用主題配置中的 theme 字段作為目錄名
	themeDirName := themeConfig.Theme
	if _, exists := currentConfig[themeDirName]; exists {
		h.respondJSON(w, Response{Status: "failure", Message: fmt.Sprintf("Theme [%s] already exists", themeDirName)})
		return
	}

	// 安裝主題
	if err := h.themeManager.Install(req); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: err.Error()})
		return
	}

	// 更新 .env 文件中的主題設置
	if err := h.updateEnvTheme(themeDirName); err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Failed to update theme in .env"})
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

// mainJSHandler 處理 main.js 請求
func (h *Handler) mainJSHandler(w http.ResponseWriter, r *http.Request) {
	// 讀取根目錄的 main.js 文件
	content, err := os.ReadFile("main.js")
	if err != nil {
		h.respondJSON(w, Response{Status: "failure", Message: "Failed to read main.js"})
		return
	}

	// 設置正確的 Content-Type
	w.Header().Set("Content-Type", "application/javascript")
	w.Write(content)
}