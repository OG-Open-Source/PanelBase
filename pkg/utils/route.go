package utils

import (
	"fmt"
	"path/filepath"
)

// RouteManager 處理路由相關操作
type RouteManager struct {
	metadata *MetadataManager
}

// RouteRequest 路由請求結構
type RouteRequest struct {
	URL  string `json:"url,omitempty"`
	Name string `json:"name,omitempty"`
}

// NewRouteManager 創建新的路由管理器
func NewRouteManager(routesConfigPath, scriptsPath string) *RouteManager {
	return &RouteManager{
		metadata: NewMetadataManager(routesConfigPath, "", scriptsPath, ""),
	}
}

// Install 安裝新路由
func (rm *RouteManager) Install(req RouteRequest) error {
	return rm.metadata.InstallRoute(req.URL)
}

// Update 更新路由
func (rm *RouteManager) Update(req RouteRequest) error {
	return rm.metadata.InstallRoute(req.URL) // 使用相同的安裝邏輯
}

// Delete 刪除路由
func (rm *RouteManager) Delete(req RouteRequest) error {
	return rm.metadata.DeleteRoute(req.Name)
}

// Execute 執行路由腳本
func (rm *RouteManager) Execute(script string, args []string) (string, error) {
	// TODO: 實現腳本執行邏輯
	scriptPath := filepath.Join(rm.metadata.scriptsPath, script)
	// 執行腳本並返回結果
	return fmt.Sprintf("Executing script %s with args %v", scriptPath, args), nil
}
