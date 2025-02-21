package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
)

type ThemeRequest struct {
	URL string `json:"url"`
}

type ThemeManager struct {
	routeManager *RouteManager
}

func NewThemeManager(routeManager *RouteManager) *ThemeManager {
	return &ThemeManager{
		routeManager: routeManager,
	}
}

func InstallThemeHandler(w http.ResponseWriter, r *http.Request) {
	var req ThemeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無效的請求", http.StatusBadRequest)
		return
	}

	// 下載主題
	themeDir := "themes"
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		http.Error(w, "無法創建主題目錄", http.StatusInternalServerError)
		return
	}

	// 下載主題文件
	resp, err := http.Get(req.URL)
	if err != nil {
		http.Error(w, "無法下載主題", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 保存主題文件
	themeFile := filepath.Join(themeDir, "theme.zip")
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "無法讀取主題數據", http.StatusInternalServerError)
		return
	}
	if err := ioutil.WriteFile(themeFile, data, 0644); err != nil {
		http.Error(w, "無法保存主題文件", http.StatusInternalServerError)
		return
	}

	// 解壓主題文件
	if err := unzip(themeFile, themeDir); err != nil {
		http.Error(w, "無法解壓主題文件", http.StatusInternalServerError)
		return
	}

	// 更新 routes.json
	if err := updateRoutesFromTheme(themeDir); err != nil {
		http.Error(w, "無法更新路由", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("主題安裝成功"))
}

func (m *ThemeManager) InstallTheme(url string) error {
	themeDir := "themes"
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		return fmt.Errorf("failed to create theme dir: %v", err)
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %v", err)
	}
	defer resp.Body.Close()

	themeFile := filepath.Join(themeDir, "theme.zip")
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read failed: %v", err)
	}

	if err := ioutil.WriteFile(themeFile, data, 0644); err != nil {
		return fmt.Errorf("save failed: %v", err)
	}

	// 解壓主題文件
	if err := m.unzip(themeFile, themeDir); err != nil {
		return fmt.Errorf("無法解壓主題文件: %v", err)
	}

	// 更新 routes.json
	if err := m.routeManager.UpdateRoutesFromTheme(themeDir); err != nil {
		return fmt.Errorf("無法更新路由: %v", err)
	}

	return nil
}

func (m *ThemeManager) unzip(src, dest string) error {
	// 實現解壓邏輯
	return nil
}

func unzip(src, dest string) error {
	// 實現解壓邏輯
	return nil
}

func updateRoutesFromTheme(themeDir string) error {
	// 從主題目錄更新 routes.json
	return nil
}