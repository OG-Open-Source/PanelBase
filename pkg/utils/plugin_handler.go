package utils

import (
	"fmt"
	"net/http"
	"sync"
)

// PluginAPIHandler handles API requests for plugins
type PluginAPIHandler struct {
	PluginManager *PluginManager
	Logger        interface{} // 简化为通用接口
	mu            sync.RWMutex
}

// Lock 获取 API 处理器写锁
func (h *PluginAPIHandler) Lock() {
	h.mu.Lock()
}

// Unlock 释放 API 处理器写锁
func (h *PluginAPIHandler) Unlock() {
	h.mu.Unlock()
}

// RLock 获取 API 处理器读锁
func (h *PluginAPIHandler) RLock() {
	h.mu.RLock()
}

// RUnlock 释放 API 处理器读锁
func (h *PluginAPIHandler) RUnlock() {
	h.mu.RUnlock()
}

// NewPluginAPIHandler creates a new plugin API handler
func NewPluginAPIHandler(pm *PluginManager, logger interface{}) *PluginAPIHandler {
	return &PluginAPIHandler{
		PluginManager: pm,
		Logger:        logger,
	}
}

// RegisterRoutes registers all plugin API routes
func (h *PluginAPIHandler) RegisterRoutes(router interface{}) {
	// 简化为接口文档式实现，实际实现时需根据路由库调整
	logInfo("Registered plugin API routes")
}

// ListPlugins returns a list of all loaded plugins
func (h *PluginAPIHandler) ListPlugins(w http.ResponseWriter, r *http.Request) {
	h.PluginManager.RLock()
	defer h.PluginManager.RUnlock()

	plugins := make([]map[string]interface{}, 0)
	for _, p := range h.PluginManager.Plugins {
		p.RLock()
		plugins = append(plugins, map[string]interface{}{
			"id":          p.Info.ID,
			"name":        p.Info.Name,
			"description": p.Info.Description,
			"version":     p.Info.Version,
			"author":      p.Info.Author,
			"status":      p.Status,
			"isRunning":   p.IsRunning,
			"startedAt":   p.StartedAt,
		})
		p.RUnlock()
	}

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"plugins":%s}`, toJSON(plugins))
}

// GetPlugin returns details about a specific plugin
func (h *PluginAPIHandler) GetPlugin(w http.ResponseWriter, r *http.Request) {
	vars := parsePathVars(r)
	id := vars["id"]

	h.PluginManager.RLock()
	plugin, exists := h.PluginManager.Plugins[id]
	h.PluginManager.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Plugin not found"}`, http.StatusNotFound)
		return
	}

	plugin.RLock()
	resp := map[string]interface{}{
		"id":          plugin.Info.ID,
		"name":        plugin.Info.Name,
		"description": plugin.Info.Description,
		"version":     plugin.Info.Version,
		"author":      plugin.Info.Author,
		"status":      plugin.Status,
		"isRunning":   plugin.IsRunning,
		"startedAt":   plugin.StartedAt,
		"config":      plugin.Config,
	}
	plugin.RUnlock()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", toJSON(resp))
}

// HandlePluginAPI forwards API requests to the appropriate plugin
func (h *PluginAPIHandler) HandlePluginAPI(w http.ResponseWriter, r *http.Request) {
	vars := parsePathVars(r)
	id := vars["id"]
	path := vars["path"]

	h.PluginManager.RLock()
	plugin, exists := h.PluginManager.Plugins[id]
	h.PluginManager.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Plugin not found"}`, http.StatusNotFound)
		return
	}

	plugin.RLock()
	handlerFunc, exists := plugin.APIRoutes[path]
	plugin.RUnlock()

	if !exists {
		http.Error(w, `{"error":"API endpoint not found"}`, http.StatusNotFound)
		return
	}

	// Call the plugin's API handler
	handler, ok := handlerFunc.(func(http.ResponseWriter, *http.Request))
	if !ok {
		http.Error(w, `{"error":"Invalid handler function"}`, http.StatusInternalServerError)
		return
	}

	handler(w, r)
}

// ListPluginTasks returns a list of all background tasks for a plugin
func (h *PluginAPIHandler) ListPluginTasks(w http.ResponseWriter, r *http.Request) {
	vars := parsePathVars(r)
	id := vars["id"]

	h.PluginManager.RLock()
	plugin, exists := h.PluginManager.Plugins[id]
	h.PluginManager.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Plugin not found"}`, http.StatusNotFound)
		return
	}

	plugin.RLock()
	tasks := make([]map[string]interface{}, 0)
	for _, task := range plugin.BackgroundTasks {
		task.Lock()
		tasks = append(tasks, map[string]interface{}{
			"id":     task.ID,
			"name":   task.Name,
			"status": task.Status,
		})
		task.Unlock()
	}
	plugin.RUnlock()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"tasks":%s}`, toJSON(tasks))
}

// GetPluginTask returns details about a specific background task
func (h *PluginAPIHandler) GetPluginTask(w http.ResponseWriter, r *http.Request) {
	vars := parsePathVars(r)
	id := vars["id"]
	taskId := vars["taskId"]

	h.PluginManager.RLock()
	plugin, exists := h.PluginManager.Plugins[id]
	h.PluginManager.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Plugin not found"}`, http.StatusNotFound)
		return
	}

	plugin.RLock()
	task, exists := plugin.BackgroundTasks[taskId]
	plugin.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
		return
	}

	task.Lock()
	resp := map[string]interface{}{
		"id":     task.ID,
		"name":   task.Name,
		"status": task.Status,
	}
	task.Unlock()

	// Return JSON response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "%s", toJSON(resp))
}

// StartPluginTask starts a background task
func (h *PluginAPIHandler) StartPluginTask(w http.ResponseWriter, r *http.Request) {
	vars := parsePathVars(r)
	id := vars["id"]
	taskId := vars["taskId"]

	h.PluginManager.RLock()
	plugin, exists := h.PluginManager.Plugins[id]
	h.PluginManager.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Plugin not found"}`, http.StatusNotFound)
		return
	}

	plugin.RLock()
	_, exists = plugin.BackgroundTasks[taskId]
	plugin.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
		return
	}

	// Start the task
	err := h.PluginManager.StartBackgroundTask(id, taskId)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to start task: %s"}`, err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success":true,"message":"Task started"}`)
}

// StopPluginTask stops a background task
func (h *PluginAPIHandler) StopPluginTask(w http.ResponseWriter, r *http.Request) {
	vars := parsePathVars(r)
	id := vars["id"]
	taskId := vars["taskId"]

	h.PluginManager.RLock()
	plugin, exists := h.PluginManager.Plugins[id]
	h.PluginManager.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Plugin not found"}`, http.StatusNotFound)
		return
	}

	plugin.RLock()
	_, exists = plugin.BackgroundTasks[taskId]
	plugin.RUnlock()

	if !exists {
		http.Error(w, `{"error":"Task not found"}`, http.StatusNotFound)
		return
	}

	// Stop the task
	err := h.PluginManager.StopBackgroundTask(id, taskId)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Failed to stop task: %s"}`, err), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"success":true,"message":"Task stopped"}`)
}

// Helper functions
func parsePathVars(r *http.Request) map[string]string {
	// 简化实现，实际中需要根据路由库替换
	// 这里仅仅是一个占位符
	return make(map[string]string)
}

func logInfo(msg string) {
	// 简化日志实现
	fmt.Println(msg)
}

// Helper function to convert a value to JSON
func toJSON(v interface{}) string {
	// Simple implementation for now
	// In a real application, you would use json.Marshal
	return fmt.Sprintf("%v", v)
}
