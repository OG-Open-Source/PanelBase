package models

import "time"

// Plugin represents a plugin in the system
type Plugin struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Version         string                 `json:"version"`
	Author          string                 `json:"author"`
	Enabled         bool                   `json:"enabled"`
	APIRoutes       map[string]string      `json:"api_routes"`           // API路由映射 (路径 -> 处理函数名)
	BackgroundTasks []string               `json:"background_tasks"`     // 后台任务函数列表
	Config          map[string]interface{} `json:"config"`               // 插件配置
	StartedAt       time.Time              `json:"started_at,omitempty"` // 插件启动时间
}

// PluginAPIResponse 插件API响应
type PluginAPIResponse struct {
	Status  string                 `json:"status"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

// PluginAPIRequest 插件API请求
type PluginAPIRequest struct {
	Method  string                 `json:"method"`
	Path    string                 `json:"path"`
	Query   map[string]string      `json:"query,omitempty"`
	Body    map[string]interface{} `json:"body,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
}

// BackgroundTaskStatus 后台任务状态
type BackgroundTaskStatus struct {
	TaskID      string    `json:"task_id"`
	PluginID    string    `json:"plugin_id"`
	TaskName    string    `json:"task_name"`
	Status      string    `json:"status"` // "running", "completed", "failed"
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time,omitempty"`
	Progress    int       `json:"progress,omitempty"` // 0-100
	Message     string    `json:"message,omitempty"`
	LastUpdated time.Time `json:"last_updated"`
}
