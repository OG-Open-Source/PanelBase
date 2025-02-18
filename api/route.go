package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"os/exec"
	"strings"
	"runtime"
)

type RouteManager struct{}

func NewRouteManager() *RouteManager {
	return &RouteManager{}
}

func (m *RouteManager) ExecuteCommand(command string, args []string) (string, error) {
	// 解析命令
	commands := strings.Split(command, "&&")
	var output strings.Builder

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		// 檢查是否為命令文件
		if !strings.Contains(cmd, " ") { // 假設沒有空格的是文件
			// 讀取命令文件
			data, err := ioutil.ReadFile(fmt.Sprintf("commands/%s", cmd))
			if err != nil {
				return "", err
			}

			// 解析註解
			pkgManager, dependencies, author, version, description := parseMetadata(string(data))

			// 輸出元數據信息
			output.WriteString(fmt.Sprintf("作者: %s\n", author))
			output.WriteString(fmt.Sprintf("版本: %s\n", version))
			output.WriteString(fmt.Sprintf("介紹: %s\n", description))

			// 檢查系統是否支援指定的套件管理器
			if !isPackageManagerSupported(pkgManager) {
				return "", fmt.Errorf("系統不支援套件管理器: %s", pkgManager)
			}

			// 檢查依賴套件是否已安裝
			for _, dep := range dependencies {
				if !isPackageInstalled(pkgManager, dep) {
					return "", fmt.Errorf("依賴套件未安裝: %s", dep)
				}
			}

			// 執行文件中的命令
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "#") || line == "" {
					continue // 跳過註解和空行
				}
				// 將參數傳遞給命令
				line = replaceArgs(line, args)
				out, err := exec.Command("sh", "-c", line).CombinedOutput()
				if err != nil {
					return "", fmt.Errorf("%v: %s", err, out)
				}
				output.WriteString(fmt.Sprintf("%s\n", out))
			}
		} else {
			// 直接執行命令
			cmd = replaceArgs(cmd, args)
			out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("%v: %s", err, out)
			}
			output.WriteString(fmt.Sprintf("%s\n", out))
		}
	}

	return output.String(), nil
}

func (m *RouteManager) GetRoutes() ([]byte, error) {
	return ioutil.ReadFile("routes.json")
}

func (m *RouteManager) InstallRoute(url string) error {
	// 下載路由指令文件
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("無法下載路由指令文件: %v", err)
	}
	defer resp.Body.Close()

	// 保存路由指令文件
	routeFile := filepath.Join("commands", filepath.Base(url))
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("無法讀取路由指令文件數據: %v", err)
	}
	if err := ioutil.WriteFile(routeFile, data, 0644); err != nil {
		return fmt.Errorf("無法保存路由指令文件: %v", err)
	}

	// 更新 routes.json
	if err := m.updateRoutes(routeFile); err != nil {
		return fmt.Errorf("無法更新路由: %v", err)
	}

	return nil
}

func (m *RouteManager) updateRoutes(routeFile string) error {
	// 更新 routes.json
	return nil
}

func (m *RouteManager) UpdateRoutesFromTheme(themeDir string) error {
	// 從主題目錄更新 routes.json
	return nil
}

type RouteRequest struct {
	URL string `json:"url"`
}

func InstallRouteHandler(w http.ResponseWriter, r *http.Request) {
	var req RouteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無效的請求", http.StatusBadRequest)
		return
	}

	// 下載路由指令文件
	resp, err := http.Get(req.URL)
	if err != nil {
		http.Error(w, "無法下載路由指令文件", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// 保存路由指令文件
	routeFile := filepath.Join("commands", filepath.Base(req.URL))
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "無法讀取路由指令文件數據", http.StatusInternalServerError)
		return
	}
	if err := ioutil.WriteFile(routeFile, data, 0644); err != nil {
		http.Error(w, "無法保存路由指令文件", http.StatusInternalServerError)
		return
	}

	// 更新 routes.json
	if err := updateRoutes(routeFile); err != nil {
		http.Error(w, "無法更新路由", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("路由指令文件安裝成功"))
}

func updateRoutes(routeFile string) error {
	// 更新 routes.json
	return nil
}