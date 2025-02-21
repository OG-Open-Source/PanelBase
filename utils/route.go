package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
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
	var routes struct {
		Commands   map[string]string `json:"commands"`
		Variables  map[string]string `json:"variables"`
	}
	if err := json.Unmarshal(routesData, &routes); err != nil {
		return "", fmt.Errorf("無法解析 routes.json: %v", err)
	}

	// 查找命令對應的文件
	cmdFile, ok := routes.Commands[command]
	if !ok {
		return "", fmt.Errorf("未找到命令: %s", command)
	}

	// 讀取命令文件
	data, err := ioutil.ReadFile(fmt.Sprintf("commands/%s", cmdFile))
	if err != nil {
		return "", fmt.Errorf("無法讀取命令文件: %v", err)
	}

	// 替換變量
	content := string(data)
	for i, arg := range args {
		// 替換 *#ARG_N#* 格式的變量
		content = strings.ReplaceAll(content, fmt.Sprintf("*#ARG_%d#*", i+1), arg)
	}

	// 自動生成 ARG_數字 變量
	re := regexp.MustCompile(`\*#ARG_\d+#\*`)
	content = re.ReplaceAllStringFunc(content, func(match string) string {
		// 提取數字部分
		num := strings.TrimPrefix(strings.TrimSuffix(match, "#*"), "*#ARG_")
		index, _ := strconv.Atoi(num)
		if index > 0 && index <= len(args) {
			return args[index-1]
		}
		return ""
	})

	// 替換所有未匹配的變量為空字符串
	re = regexp.MustCompile(`\*#[A-Za-z0-9_]+#\*`)
	content = re.ReplaceAllString(content, "")

	// 創建臨時文件
	tmpFile, err := ioutil.TempFile("", "cmd_")
	if err != nil {
		return "", fmt.Errorf("無法創建臨時文件: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// 寫入替換後的內容
	if _, err := tmpFile.WriteString(content); err != nil {
		return "", fmt.Errorf("無法寫入臨時文件: %v", err)
	}
	tmpFile.Close()

	// 解析註解
	pkgManagers, dependencies, commands, author, version, description := parseMetadata(content, cmdFile)

	// 輸出元數據信息
	var output strings.Builder
	output.WriteString(fmt.Sprintf("作者: %s\n", author))
	output.WriteString(fmt.Sprintf("版本: %s\n", version))
	output.WriteString(fmt.Sprintf("介紹: %s\n", description))

	// 檢查系統是否支援指定的套件管理器
	if !isPackageManagerSupported(pkgManagers) {
		return "", fmt.Errorf("系統不支援套件管理器: %v", pkgManagers)
	}

	// 檢查依賴套件是否已安裝
	for _, dep := range dependencies {
		if !isPackageInstalled(pkgManagers[0], dep) {
			return "", fmt.Errorf("依賴套件未安裝: %s", dep)
		}
	}

	// 如果有新的commands，更新routes.json
	if len(commands) > 0 {
		if err := m.updateCommands(commands); err != nil {
			return "", fmt.Errorf("failed to update commands: %v", err)
		}
	}

	// 根據文件類型執行
	var cmd *exec.Cmd
	switch filepath.Ext(cmdFile) {
	case ".sh":
		cmd = exec.Command("bash", tmpFile.Name())
	case ".py":
		cmd = exec.Command("python3", tmpFile.Name())
	case ".go":
		cmd = exec.Command("go", "run", tmpFile.Name())
	default:
		return "", fmt.Errorf("不支援的文件類型: %s", filepath.Ext(cmdFile))
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		// 返回完整的錯誤信息，包括命令輸出
		return string(out), fmt.Errorf("%v: %s", err, out)
	}

	return string(out), nil
}

func (m *RouteManager) GetRoutes() ([]byte, error) {
	return ioutil.ReadFile("routes.json")
}

func (m *RouteManager) InstallRoute(url string) error {
	// 下載路由指令文件
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法下載路由指令文件: %v"}`, err)
	}
	defer resp.Body.Close()

	// 保存路由指令文件
	routeFile := filepath.Join("commands", filepath.Base(url))
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法讀取路由指令文件數據: %v"}`, err)
	}
	if err := ioutil.WriteFile(routeFile, data, 0644); err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法保存路由指令文件: %v"}`, err)
	}

	// 解析並處理註解
	if err := m.processRouteMetadata(routeFile, string(data)); err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法處理路由指令文件元數據: %v"}`, err)
	}

	// 更新 routes.json
	if err := m.updateRoutes(routeFile); err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法更新路由: %v"}`, err)
	}

	return nil
}

func (m *RouteManager) updateRoutes(routeFile string) error {
	// 讀取文件內容
	data, err := ioutil.ReadFile(routeFile)
	if err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法讀取路由指令文件: %v"}`, err)
	}

	// 解析並處理註解
	if err := m.processRouteMetadata(routeFile, string(data)); err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法處理路由指令文件元數據: %v"}`, err)
	}

	// 更新 routes.json
	routesData, err := ioutil.ReadFile("routes.json")
	if err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法讀取 routes.json: %v"}`, err)
	}

	var routes struct {
		Commands  map[string]string `json:"commands"`
		Variables map[string]string `json:"variables,omitempty"`
	}
	if err := json.Unmarshal(routesData, &routes); err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法解析 routes.json: %v"}`, err)
	}

	// 如果Variables為nil，初始化為空map
	if routes.Variables == nil {
		routes.Variables = make(map[string]string)
	}

	// 添加新命令
	fileName := filepath.Base(routeFile)
	routes.Commands[strings.TrimSuffix(fileName, filepath.Ext(fileName))] = fileName

	// 寫回 routes.json
	newData, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法編碼 routes.json: %v"}`, err)
	}

	if err := ioutil.WriteFile("routes.json", newData, 0644); err != nil {
		return fmt.Errorf(`{"status": "error", "message": "無法寫入 routes.json: %v"}`, err)
	}

	return nil
}

func (m *RouteManager) processRouteMetadata(filePath, content string) error {
	// 解析註解
	pkgManagers, dependencies, commands, _, _, _ := parseMetadata(content, filepath.Base(filePath))

	// 處理 commands
	if len(commands) > 0 {
		if err := m.updateCommands(commands); err != nil {
			return fmt.Errorf(`{"status": "error", "message": "無法更新 commands: %v"}`, err)
		}
	}

	// 如果有依賴，才進行檢查
	if len(dependencies) > 0 && len(pkgManagers) > 0 {
		// 檢查是否有支援的包管理器
		supported := false
		for _, pkgManager := range pkgManagers {
			if isPackageManagerSupported([]string{pkgManager}) {
				supported = true
				break
			}
		}

		if !supported {
			return fmt.Errorf(`{"status": "error", "message": "系統不支援任何指定的套件管理器: %v"}`, pkgManagers)
		}

		for _, pkgManager := range pkgManagers {
			if !isPackageManagerSupported([]string{pkgManager}) {
				continue
			}

			for _, dep := range dependencies {
				if !isPackageInstalled(pkgManager, dep) {
					// 這裡可以添加自動安裝依賴的邏輯
					return fmt.Errorf(`{"status": "error", "message": "依賴套件未安裝: %s"}`, dep)
				}
			}
		}
	}

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

	// 下載並安裝路由指令文件
	if err := NewRouteManager().InstallRoute(req.URL); err != nil {
		http.Error(w, fmt.Sprintf("路由指令文件安裝失敗: %v", err), http.StatusInternalServerError)
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
func parseMetadata(data, filePath string) ([]string, []string, map[string]string, string, string, string) {
	var pkgManagers []string
	var dependencies []string
	var commands map[string]string = make(map[string]string)
	var author string
	var version string
	var description string

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		// 處理 # @ 和 // @ 格式的註解
		if strings.HasPrefix(line, "# @") || strings.HasPrefix(line, "// @") {
			// 移除註解符號
			line = strings.TrimPrefix(line, "# @")
			line = strings.TrimPrefix(line, "// @")

			if strings.HasPrefix(line, "commands:") {
				// 解析commands，格式為 command1,command2
				cmdNames := strings.Split(strings.TrimSpace(strings.Split(line, ":")[1]), ",")
				for _, cmd := range cmdNames {
					commands[strings.TrimSpace(cmd)] = filePath
				}
			} else if strings.HasPrefix(line, "pkg_manager:") {
				pkgManagers = strings.Split(strings.TrimSpace(strings.Split(line, ":")[1]), ",")
				for i := range pkgManagers {
					pkgManagers[i] = strings.TrimSpace(pkgManagers[i])
				}
			} else if strings.HasPrefix(line, "dependencies:") {
				deps := strings.TrimSpace(strings.Split(line, ":")[1])
				if deps != "null" {
					dependencies = strings.Split(deps, ",")
					for i := range dependencies {
						dependencies[i] = strings.TrimSpace(dependencies[i])
					}
				}
			} else if strings.HasPrefix(line, "author:") {
				author = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.HasPrefix(line, "version:") {
				version = strings.TrimSpace(strings.Split(line, ":")[1])
			} else if strings.HasPrefix(line, "description:") {
				description = strings.TrimSpace(strings.Split(line, ":")[1])
			}
		}
	}

	return pkgManagers, dependencies, commands, author, version, description
}

// 檢查系統是否支援指定的套件管理器
func isPackageManagerSupported(packageManagers []string) bool {
	for _, packageManager := range packageManagers {
		switch packageManager {
		case "apk":
			if runtime.GOOS == "linux" && isCommandAvailable("apk") {
				return true
			}
		case "apt":
			if runtime.GOOS == "linux" && isCommandAvailable("apt-get") {
				return true
			}
		case "opkg":
			if runtime.GOOS == "linux" && isCommandAvailable("opkg") {
				return true
			}
		case "pacman":
			if runtime.GOOS == "linux" && isCommandAvailable("pacman") {
				return true
			}
		case "yum":
			if runtime.GOOS == "linux" && isCommandAvailable("yum") {
				return true
			}
		case "zypper":
			if runtime.GOOS == "linux" && isCommandAvailable("zypper") {
				return true
			}
		case "dnf":
			if runtime.GOOS == "linux" && isCommandAvailable("dnf") {
				return true
			}
		}
	}
	return false
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
	case "apk":
		checkCmd = fmt.Sprintf("apk info -e %s &>/dev/null", packageName)
	case "apt":
		checkCmd = fmt.Sprintf("dpkg-query -W -f='${Status}' %s 2>/dev/null | grep -q 'ok installed'", packageName)
	case "opkg":
		checkCmd = fmt.Sprintf("opkg list-installed | grep -q '^%s '", packageName)
	case "pacman":
		checkCmd = fmt.Sprintf("pacman -Qi %s &>/dev/null", packageName)
	case "yum", "dnf":
		checkCmd = fmt.Sprintf("%s list installed %s &>/dev/null", packageManager, packageName)
	case "zypper":
		checkCmd = fmt.Sprintf("zypper se -i -x %s &>/dev/null", packageName)
	default:
		return false
	}

	err := exec.Command("bash", "-c", checkCmd).Run()
	return err == nil
}

// 替換命令中的參數
func replaceArgs(cmd string, args []string) string {
	for i, arg := range args {
		// 使用 {ARG_1}, {ARG_2}, ... 作為通用變量格式
		cmd = strings.ReplaceAll(cmd, fmt.Sprintf("{ARG_%d}", i+1), arg)
	}
	return cmd
}

func (m *RouteManager) updateCommands(commands map[string]string) error {
	// 讀取現有的routes.json
	data, err := ioutil.ReadFile("routes.json")
	if err != nil {
		return err
	}

	var routes struct {
		Commands  map[string]string `json:"commands"`
		Variables map[string]string `json:"variables"`
	}
	if err := json.Unmarshal(data, &routes); err != nil {
		return err
	}

	// 更新commands
	for cmd, file := range commands {
		routes.Commands[cmd] = file
	}

	// 寫回routes.json
	newData, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile("routes.json", newData, 0644)
}

func (m *RouteManager) GetRouteMetadata(url string) (map[string]interface{}, error) {
	// 下載路由指令文件
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("無法下載路由指令文件: %v", err)
	}
	defer resp.Body.Close()

	// 讀取文件數據
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("無法讀取路由指令文件數據: %v", err)
	}

	// 解析元數據
	pkgManagers, dependencies, commands, author, version, description := parseMetadata(string(data), filepath.Base(url))

	return map[string]interface{}{
		"commands":     commands,
		"pkg_manager":  pkgManagers,
		"dependencies": dependencies,
		"author":       author,
		"version":      version,
		"description":  description,
	}, nil
}