package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	"github.com/gorilla/mux"
)

type CommandRequest struct {
	Command string `json:"command"`
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
		output, err := executeCommand(command)
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

func executeCommand(command string) (string, error) {
	// 解析命令
	commands := strings.Split(command, "&&")
	var output strings.Builder

	for _, cmd := range commands {
		cmd = strings.TrimSpace(cmd)
		// 檢查是否為命令文件
		if strings.HasSuffix(cmd, ".sh") || strings.HasSuffix(cmd, "") {
			// 讀取命令文件
			data, err := ioutil.ReadFile(fmt.Sprintf("commands/%s", cmd))
			if err != nil {
				return "", err
			}
			// 執行文件中的命令
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "---") || line == "" {
					continue
				}
				out, err := exec.Command("sh", "-c", line).CombinedOutput()
				if err != nil {
					return "", fmt.Errorf("%v: %s", err, out)
				}
				output.WriteString(fmt.Sprintf("%s\n", out))
			}
		} else {
			// 直接執行命令
			out, err := exec.Command("sh", "-c", cmd).CombinedOutput()
			if err != nil {
				return "", fmt.Errorf("%v: %s", err, out)
			}
			output.WriteString(fmt.Sprintf("%s\n", out))
		}
	}

	return output.String(), nil
}