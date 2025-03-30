package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/models"
	"github.com/google/uuid"
)

// PluginInfo contains information about a plugin
type PluginInfo struct {
	ID          string
	Name        string
	Description string
	Version     string
	Author      string
}

// Plugin represents a loaded plugin
type Plugin struct {
	Info            *PluginInfo
	Handle          *plugin.Plugin
	APIRoutes       map[string]plugin.Symbol   // API路由处理函数
	BackgroundTasks map[string]*BackgroundTask // 后台任务
	Config          map[string]interface{}
	IsRunning       bool
	StartedAt       time.Time
	Status          string
	mu              sync.RWMutex // 为每个插件添加单独的锁
}

// Lock 获取插件写锁
func (p *Plugin) Lock() {
	p.mu.Lock()
}

// Unlock 释放插件写锁
func (p *Plugin) Unlock() {
	p.mu.Unlock()
}

// RLock 获取插件读锁
func (p *Plugin) RLock() {
	p.mu.RLock()
}

// RUnlock 释放插件读锁
func (p *Plugin) RUnlock() {
	p.mu.RUnlock()
}

// BackgroundTask 表示一个后台任务
type BackgroundTask struct {
	ID       string
	Name     string
	Function plugin.Symbol
	Status   *models.BackgroundTaskStatus
	StopChan chan struct{}
	mu       sync.Mutex
}

// Lock 获取任务锁
func (t *BackgroundTask) Lock() {
	t.mu.Lock()
}

// Unlock 释放任务锁
func (t *BackgroundTask) Unlock() {
	t.mu.Unlock()
}

// PluginManager manages plugins
type PluginManager struct {
	Plugins         map[string]*Plugin
	PluginsPath     string
	BackgroundTasks map[string]*BackgroundTask // 所有正在运行的后台任务
	mu              sync.RWMutex               // 保护并发访问
}

// Lock 获取管理器写锁
func (pm *PluginManager) Lock() {
	pm.mu.Lock()
}

// Unlock 释放管理器写锁
func (pm *PluginManager) Unlock() {
	pm.mu.Unlock()
}

// RLock 获取管理器读锁
func (pm *PluginManager) RLock() {
	pm.mu.RLock()
}

// RUnlock 释放管理器读锁
func (pm *PluginManager) RUnlock() {
	pm.mu.RUnlock()
}

// NewPluginManager creates a new plugin manager
func NewPluginManager(pluginsPath string) *PluginManager {
	return &PluginManager{
		Plugins:         make(map[string]*Plugin),
		PluginsPath:     pluginsPath,
		BackgroundTasks: make(map[string]*BackgroundTask),
	}
}

// LoadPlugin loads a plugin from the given path
func (pm *PluginManager) LoadPlugin(pluginID string) (*Plugin, error) {
	// 注意：这里我们不能在函数开始就锁定，否则会导致死锁
	// 先检查插件文件是否存在，这不需要锁
	pluginPath := filepath.Join(pm.PluginsPath, pluginID, fmt.Sprintf("%s.so", pluginID))

	// Check if the plugin file exists
	_, err := os.Stat(pluginPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("plugin not found: %s", pluginPath)
	}

	// Open the plugin
	p, err := plugin.Open(pluginPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin: %w", err)
	}

	// Look up the GetPluginInfo function
	infoSymbol, err := p.Lookup("GetPluginInfo")
	if err != nil {
		return nil, fmt.Errorf("plugin does not export GetPluginInfo: %w", err)
	}

	// Call the GetPluginInfo function
	getPluginInfo, ok := infoSymbol.(func() *PluginInfo)
	if !ok {
		return nil, fmt.Errorf("GetPluginInfo has wrong type signature")
	}

	info := getPluginInfo()
	if info == nil {
		return nil, fmt.Errorf("GetPluginInfo returned nil")
	}

	// 创建插件实例
	plug := &Plugin{
		Info:            info,
		Handle:          p,
		APIRoutes:       make(map[string]plugin.Symbol),
		BackgroundTasks: make(map[string]*BackgroundTask),
		Config:          make(map[string]interface{}),
		IsRunning:       false,
		Status:          "loaded",
	}

	// 加载API路由
	if apiRoutesSymbol, err := p.Lookup("GetAPIRoutes"); err == nil {
		if getAPIRoutes, ok := apiRoutesSymbol.(func() map[string]string); ok {
			apiRoutes := getAPIRoutes()
			for path, funcName := range apiRoutes {
				if sym, err := p.Lookup(funcName); err == nil {
					plug.APIRoutes[path] = sym
				}
			}
		}
	}

	// 加载后台任务
	if tasksSymbol, err := p.Lookup("GetBackgroundTasks"); err == nil {
		if getTasks, ok := tasksSymbol.(func() []string); ok {
			tasks := getTasks()
			for _, taskName := range tasks {
				if sym, err := p.Lookup(taskName); err == nil {
					taskID := uuid.New().String()
					task := &BackgroundTask{
						ID:       taskID,
						Name:     taskName,
						Function: sym,
						Status: &models.BackgroundTaskStatus{
							TaskID:      taskID,
							PluginID:    info.ID,
							TaskName:    taskName,
							Status:      "registered",
							LastUpdated: time.Now(),
						},
						StopChan: make(chan struct{}),
					}
					plug.BackgroundTasks[taskName] = task
				}
			}
		}
	}

	// 现在是存储插件的时候，需要获取锁
	pm.Lock()
	defer pm.Unlock()

	// Store the plugin
	pm.Plugins[pluginID] = plug

	return plug, nil
}

// LoadAllPlugins loads all plugins from the plugins directory
func (pm *PluginManager) LoadAllPlugins() error {
	// 先读取目录，这不需要锁
	// Check if the plugins directory exists
	_, err := os.Stat(pm.PluginsPath)
	if os.IsNotExist(err) {
		return fmt.Errorf("plugins directory not found: %s", pm.PluginsPath)
	}

	// Read the directory
	entries, err := os.ReadDir(pm.PluginsPath)
	if err != nil {
		return fmt.Errorf("failed to read plugins directory: %w", err)
	}

	// 读取完目录后再遍历加载插件，每个插件的加载过程中LoadPlugin会自己处理锁
	for _, entry := range entries {
		if entry.IsDir() {
			pluginID := entry.Name()
			_, err := pm.LoadPlugin(pluginID)
			if err != nil {
				fmt.Printf("Failed to load plugin %s: %v\n", pluginID, err)
				continue
			}
		}
	}

	return nil
}

// StartPlugin starts a plugin and its background tasks
func (pm *PluginManager) StartPlugin(pluginID string) error {
	// 先获取读锁检查插件是否存在
	pm.RLock()
	plug, exists := pm.Plugins[pluginID]
	pm.RUnlock()

	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	// 获取插件的锁
	plug.Lock()
	defer plug.Unlock()

	if plug.IsRunning {
		return fmt.Errorf("plugin is already running: %s", pluginID)
	}

	// Look up and call the Start function if it exists
	if startSym, err := plug.Handle.Lookup("Start"); err == nil {
		if startFunc, ok := startSym.(func() error); ok {
			if err := startFunc(); err != nil {
				return fmt.Errorf("failed to start plugin: %w", err)
			}
		}
	}

	// Start all background tasks
	for _, task := range plug.BackgroundTasks {
		taskFunc, ok := task.Function.(func(chan struct{}) error)
		if !ok {
			continue
		}

		task.Lock()
		task.Status.Status = "running"
		task.Status.StartTime = time.Now()
		task.Status.LastUpdated = time.Now()
		task.Unlock()

		// 添加到全局任务列表需要获取管理器的写锁
		pm.Lock()
		pm.BackgroundTasks[task.ID] = task
		pm.Unlock()

		// 启动后台任务
		go func(t *BackgroundTask) {
			if err := taskFunc(t.StopChan); err != nil {
				t.Lock()
				t.Status.Status = "failed"
				t.Status.Message = err.Error()
				t.Status.EndTime = time.Now()
				t.Status.LastUpdated = time.Now()
				t.Unlock()
			} else {
				t.Lock()
				t.Status.Status = "completed"
				t.Status.EndTime = time.Now()
				t.Status.Progress = 100
				t.Status.LastUpdated = time.Now()
				t.Unlock()
			}
		}(task)
	}

	plug.IsRunning = true
	plug.StartedAt = time.Now()
	plug.Status = "running"

	return nil
}

// StopPlugin stops a plugin and all its background tasks
func (pm *PluginManager) StopPlugin(pluginID string) error {
	// 先获取读锁检查插件是否存在
	pm.RLock()
	plug, exists := pm.Plugins[pluginID]
	pm.RUnlock()

	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	// 获取插件的锁
	plug.Lock()
	defer plug.Unlock()

	if !plug.IsRunning {
		return fmt.Errorf("plugin is not running: %s", pluginID)
	}

	// Stop all background tasks
	for _, task := range plug.BackgroundTasks {
		// 发送停止信号
		close(task.StopChan)

		// 从全局任务列表中移除
		pm.Lock()
		delete(pm.BackgroundTasks, task.ID)
		pm.Unlock()

		task.Lock()
		if task.Status.Status == "running" {
			task.Status.Status = "stopped"
			task.Status.EndTime = time.Now()
			task.Status.LastUpdated = time.Now()
		}
		task.Unlock()
	}

	// Look up and call the Stop function if it exists
	if stopSym, err := plug.Handle.Lookup("Stop"); err == nil {
		if stopFunc, ok := stopSym.(func() error); ok {
			if err := stopFunc(); err != nil {
				return fmt.Errorf("failed to stop plugin: %w", err)
			}
		}
	}

	plug.IsRunning = false
	plug.Status = "stopped"

	return nil
}

// GetTaskStatus returns the status of a background task
func (pm *PluginManager) GetTaskStatus(taskID string) (*models.BackgroundTaskStatus, error) {
	pm.RLock()
	task, exists := pm.BackgroundTasks[taskID]
	pm.RUnlock()

	if !exists {
		return nil, fmt.Errorf("task not found: %s", taskID)
	}

	task.Lock()
	// 创建状态副本
	status := *task.Status
	task.Unlock()

	return &status, nil
}

// UpdateTaskProgress 更新任务进度
func (pm *PluginManager) UpdateTaskProgress(taskID string, progress int, message string) error {
	pm.RLock()
	task, exists := pm.BackgroundTasks[taskID]
	pm.RUnlock()

	if !exists {
		return fmt.Errorf("task not found: %s", taskID)
	}

	task.Lock()
	task.Status.Progress = progress
	task.Status.Message = message
	task.Status.LastUpdated = time.Now()
	task.Unlock()

	return nil
}

// ExecuteAPIRoute 执行插件的API路由
func (pm *PluginManager) ExecuteAPIRoute(pluginID, route string, request *models.PluginAPIRequest) (*models.PluginAPIResponse, error) {
	pm.RLock()
	plug, exists := pm.Plugins[pluginID]
	pm.RUnlock()

	if !exists {
		return nil, fmt.Errorf("plugin not found: %s", pluginID)
	}

	plug.RLock()
	running := plug.IsRunning
	plug.RUnlock()

	if !running {
		return nil, fmt.Errorf("plugin is not running: %s", pluginID)
	}

	plug.RLock()
	handler, exists := plug.APIRoutes[route]
	plug.RUnlock()

	if !exists {
		return nil, fmt.Errorf("API route not found: %s", route)
	}

	// 执行API处理函数
	apiFunc, ok := handler.(func(*models.PluginAPIRequest) *models.PluginAPIResponse)
	if !ok {
		return nil, fmt.Errorf("API handler has wrong type signature")
	}

	return apiFunc(request), nil
}

// StartBackgroundTask starts a background task for a plugin
func (pm *PluginManager) StartBackgroundTask(pluginID string, taskID string) error {
	pm.Lock()
	defer pm.Unlock()

	plugin, exists := pm.Plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	plugin.Lock()
	task, exists := plugin.BackgroundTasks[taskID]
	if !exists {
		plugin.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}

	// If the task is already running, return an error
	if task.Status != nil && task.Status.Status == "running" {
		plugin.Unlock()
		return fmt.Errorf("task is already running")
	}

	// Create a stop channel for the task
	task.StopChan = make(chan struct{})

	// Create a status object for the task
	task.Status = &models.BackgroundTaskStatus{
		TaskID:      taskID,
		PluginID:    pluginID,
		TaskName:    task.Name,
		Status:      "running",
		StartTime:   time.Now(),
		LastUpdated: time.Now(),
	}

	// Add the task to the global background tasks
	pm.BackgroundTasks[taskID] = task

	plugin.Unlock()

	// Start the task in a goroutine
	go func() {
		// Get the task function
		taskFunc, ok := task.Function.(func(<-chan struct{}) error)
		if !ok {
			// Update the status
			task.Lock()
			task.Status.Status = "error"
			task.Status.Message = "invalid task function"
			task.Status.EndTime = time.Now()
			task.Status.LastUpdated = time.Now()
			task.Unlock()
			return
		}

		// Call the task function
		err := taskFunc(task.StopChan)

		// Update the status
		task.Lock()
		if err != nil {
			task.Status.Status = "error"
			task.Status.Message = err.Error()
		} else {
			task.Status.Status = "stopped"
		}
		task.Status.EndTime = time.Now()
		task.Status.LastUpdated = time.Now()
		task.Unlock()

		// Remove the task from the global background tasks when it's done
		pm.Lock()
		delete(pm.BackgroundTasks, taskID)
		pm.Unlock()
	}()

	return nil
}

// StopBackgroundTask stops a background task for a plugin
func (pm *PluginManager) StopBackgroundTask(pluginID string, taskID string) error {
	pm.Lock()
	defer pm.Unlock()

	plugin, exists := pm.Plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	plugin.Lock()
	task, exists := plugin.BackgroundTasks[taskID]
	if !exists {
		plugin.Unlock()
		return fmt.Errorf("task not found: %s", taskID)
	}

	// If the task is not running, return an error
	if task.Status == nil || task.Status.Status != "running" {
		plugin.Unlock()
		return fmt.Errorf("task is not running")
	}

	// Send the stop signal
	close(task.StopChan)

	// Update the status
	task.Status.Status = "stopping"

	plugin.Unlock()

	return nil
}

// IsPluginLoaded checks if a plugin is loaded
func (pm *PluginManager) IsPluginLoaded(pluginID string) bool {
	pm.RLock()
	defer pm.RUnlock()
	_, exists := pm.Plugins[pluginID]
	return exists
}

// IsPluginRunning checks if a plugin is running
func (pm *PluginManager) IsPluginRunning(pluginID string) bool {
	pm.RLock()
	defer pm.RUnlock()
	plugin, exists := pm.Plugins[pluginID]
	if !exists {
		return false
	}
	plugin.RLock()
	defer plugin.RUnlock()
	return plugin.IsRunning
}

// UnloadPlugin unloads a plugin
func (pm *PluginManager) UnloadPlugin(pluginID string) error {
	pm.Lock()
	defer pm.Unlock()

	plugin, exists := pm.Plugins[pluginID]
	if !exists {
		return fmt.Errorf("plugin not found: %s", pluginID)
	}

	// 检查插件是否在运行
	plugin.RLock()
	isRunning := plugin.IsRunning
	plugin.RUnlock()

	if isRunning {
		return fmt.Errorf("cannot unload running plugin: %s, stop it first", pluginID)
	}

	// 删除插件
	delete(pm.Plugins, pluginID)
	return nil
}
