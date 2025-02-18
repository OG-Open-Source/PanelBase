package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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
	// 加載 routes.json
	routesData, err := ioutil.ReadFile("routes.json")
	if err != nil {
		return "", fmt.Errorf("無法讀取 routes.json: %v", err)
	}

	// 解析 routes.json
	var routes map[string]string
	if err := json.Unmarshal(routesData, &routes); err != nil {
		return "", fmt.Errorf("無法解析 routes.json: %v", err)
	}

	// 查找命令對應的文件
	cmdFile, ok := routes[command]
	if !ok {
		return "", fmt.Errorf("未找到命令: %s", command)
	}

	// 讀取命令文件
	data, err := ioutil.ReadFile(fmt.Sprintf("commands/%s", cmdFile))
	if err != nil {
		return "", fmt.Errorf("無法讀取命令文件: %v", err)
	}

	// 解析註解
	pkgManager, dependencies, author, version, description := parseMetadata(string(data))

	// 輸出元數據信息
	var output strings.Builder
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

		// 根據文件擴展名選擇解釋器
		interpreter := "sh" // 默認為 sh
		switch filepath.Ext(cmdFile) {
		case ".py":
			interpreter = "python3"
		case ".go":
			interpreter = "go run"
		case ".sh":
			interpreter = "bash"
		default:
			interpreter = "sh"
		}

		out, err := exec.Command(interpreter, "-c", line).CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%v: %s", err, out)
		}
		output.WriteString(fmt.Sprintf("%s\n", out))
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

// 解析註解
func parseMetadata(data string) (string, []string, string, string, string) {
	var pkgManager string
	var dependencies []string
	var author string
	var version string
	var description string

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# @pkg_manager:") {
			pkgManager = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "# @dependencies:") {
			deps := strings.TrimSpace(strings.Split(line, ":")[1])
			dependencies = strings.Split(deps, ",")
			for i := range dependencies {
				dependencies[i] = strings.TrimSpace(dependencies[i])
			}
		} else if strings.HasPrefix(line, "# @author:") {
			author = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "# @version:") {
			version = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "# @description:") {
			description = strings.TrimSpace(strings.Split(line, ":")[1])
		}
	}

	return pkgManager, dependencies, author, version, description
}

// 檢查系統是否支援指定的套件管理器
func isPackageManagerSupported(packageManager string) bool {
	switch packageManager {
	case "apt":
		// 檢查是否為 Debian/Ubuntu 系統
		return runtime.GOOS == "linux" && isCommandAvailable("apt-get")
	case "yum":
		// 檢查是否為 CentOS/RHEL 系統
		return runtime.GOOS == "linux" && isCommandAvailable("yum")
	default:
		return false
	}
}

// 檢查命令是否可用
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

// 檢查套件是否已安裝
func isPackageInstalled(packageManager, packageName string) bool {
	var checkCmd string
	switch packageManager {
	case "apt":
		checkCmd = fmt.Sprintf("dpkg-query -W -f='${Status}' %s 2>/dev/null | grep -q 'ok installed'", packageName)
	case "yum":
		checkCmd = fmt.Sprintf("rpm -q %s", packageName)
	default:
		return false
	}

	err := exec.Command("bash", "-c", checkCmd).Run()
	return err == nil
}

// 替換命令中的參數
func replaceArgs(cmd string, args []string) string {
	for i, arg := range args {
		cmd = strings.ReplaceAll(cmd, fmt.Sprintf("$%d", i+1), arg)
	}
	return cmd
}