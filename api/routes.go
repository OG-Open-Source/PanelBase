package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"
	"regexp"

	"github.com/gorilla/mux"
	"gopkg.in/yaml.v2"
)

type CommandRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
}

// 新增 CommandFile 結構
type CommandFile struct {
	Metadata struct {
		Name           string   `yaml:"name"`
		Description    string   `yaml:"description"`
		PackageManager string   `yaml:"package_manager"`
		Dependencies   []string `yaml:"dependencies"`
	} `yaml:"metadata"`
	Commands []string `yaml:"commands"`
}

func SetupRoutes() *mux.Router {
	router := mux.NewRouter()

	// 安全入口路由
	router.HandleFunc("/{securityEntry}/status", statusHandler).Methods("GET")
	router.HandleFunc("/{securityEntry}/command", commandHandler).Methods("POST")

	return router
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("PanelBase agent 正在運行"))
}

func commandHandler(w http.ResponseWriter, r *http.Request) {
	var req CommandRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "無效的請求", http.StatusBadRequest)
		return
	}

	// 加載路由文件
	routes, err := loadRoutes()
	if err != nil {
		http.Error(w, "無法加載路由文件", http.StatusInternalServerError)
		return
	}

	// 檢查命令是否存在於路由中
	if command, ok := routes[req.Command]; ok {
		// 執行命令
		output, err := executeCommand(command, req.Args)
		if err != nil {
			http.Error(w, fmt.Sprintf("命令執行失敗: %v", err), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(output))
	} else {
		http.Error(w, "未知的命令", http.StatusNotFound)
	}
}

func loadRoutes() (map[string]string, error) {
	data, err := ioutil.ReadFile("routes.json")
	if err != nil {
		return nil, err
	}

	var routes map[string]string
	if err := json.Unmarshal(data, &routes); err != nil {
		return nil, err
	}

	return routes, nil
}

func executeCommand(command string, args []string) (string, error) {
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
			packageManager, dependencies := parseMetadata(string(data))

			// 安裝依賴套件
			if len(dependencies) > 0 {
				installCmd := fmt.Sprintf("%s install -y %s", packageManager, strings.Join(dependencies, " "))
				out, err := exec.Command("sh", "-c", installCmd).CombinedOutput()
				if err != nil {
					return "", fmt.Errorf("安裝依賴失敗: %v: %s", err, out)
				}
				output.WriteString(fmt.Sprintf("安裝依賴成功: %s\n", out))
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

// 替換命令中的參數
func replaceArgs(cmd string, args []string) string {
	for i, arg := range args {
		cmd = strings.ReplaceAll(cmd, fmt.Sprintf("$%d", i+1), arg)
	}
	return cmd
}

// 解析註解
func parseMetadata(data string) (string, []string) {
	var packageManager string
	var dependencies []string

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# @package_manager:") {
			packageManager = strings.TrimSpace(strings.Split(line, ":")[1])
		} else if strings.HasPrefix(line, "# @dependencies:") {
			deps := strings.TrimSpace(strings.Split(line, ":")[1])
			dependencies = strings.Split(deps, ",")
			for i := range dependencies {
				dependencies[i] = strings.TrimSpace(dependencies[i])
			}
		}
	}

	return packageManager, dependencies
}