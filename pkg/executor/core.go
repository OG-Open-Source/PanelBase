package executor

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/OG-Open-Source/PanelBase/pkg/utils"
)

// Command 指令結構
type Command struct {
	Name string   `json:"name"`  // 路由名稱（對應 routes.json 中的鍵）
	Args []string `json:"args"`  // 參數列表
}

// ExecuteRequest 執行請求結構
type ExecuteRequest struct {
	Commands []Command `json:"commands"` // 指令列表
}

// Executor 執行器結構
type Executor struct {
	scriptsPath     string              // 腳本目錄路徑
	routesConfig    utils.RouteConfig   // 路由配置
}

// NewExecutor 創建新的執行器
func NewExecutor(scriptsPath, routesConfigPath string) *Executor {
	// 載入路由配置
	metadata := utils.NewMetadataManager(routesConfigPath, "", scriptsPath, "")
	config, err := metadata.LoadRouteConfig()
	if err != nil {
		utils.Error("Failed to load routes config: '%v'", err)
		return nil
	}

	return &Executor{
		scriptsPath:  scriptsPath,
		routesConfig: config,
	}
}

// Execute 執行指令列表
func (e *Executor) Execute(req ExecuteRequest) (string, error) {
	var results []string
	var lastOutput string

	utils.Debug("Starting execution of [%d] commands", len(req.Commands))

	// 創建臨時目錄作為沙盒
	tmpDir, err := os.MkdirTemp("", "panelbase-sandbox-*")
	if err != nil {
		return "", fmt.Errorf("Failed to create sandbox directory: '%v'", err)
	}
	defer os.RemoveAll(tmpDir)

	for i, cmd := range req.Commands {
		// 從路由配置中獲取腳本名稱
		scriptName, exists := e.routesConfig[cmd.Name]
		if !exists {
			return "", fmt.Errorf("Route [%s] not found in configuration", cmd.Name)
		}

		utils.Debug("Executing command [%d]: route [%s] (script: '%s') with args %v", 
			i+1, cmd.Name, scriptName, cmd.Args)

		// 獲取原始腳本路徑
		originalScript := filepath.Join("internal", "scripts", scriptName)

		// 複製腳本到沙盒並處理變量
		sandboxScript := filepath.Join(tmpDir, scriptName)
		if err := copyAndProcessScript(originalScript, sandboxScript, cmd.Args, lastOutput); err != nil {
			return "", fmt.Errorf("Failed to prepare script: '%v'", err)
		}

		// 根據腳本類型選擇執行器
		var execCmd *exec.Cmd
		switch filepath.Ext(scriptName) {
		case ".sh":
			execCmd = exec.Command("bash", sandboxScript)
		case ".py":
			execCmd = exec.Command("python3", sandboxScript)
		case ".go":
			execCmd = exec.Command("go", "run", sandboxScript)
		default:
			return "", fmt.Errorf("Unsupported script type: '%s'", filepath.Ext(scriptName))
		}

		// 設置工作目錄為沙盒目錄
		execCmd.Dir = tmpDir

		// 捕獲輸出
		var stdout, stderr bytes.Buffer
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr

		// 執行指令
		if err := execCmd.Run(); err != nil {
			return "", fmt.Errorf("Failed to execute route [%s] (script: '%s'): '%v'\nStderr: '%s'", 
				cmd.Name, scriptName, err, stderr.String())
		}

		// 保存輸出
		output := stdout.String()
		lastOutput = output
		results = append(results, output)

		utils.Debug("Command [%d] completed successfully", i+1)
	}

	// 合併所有輸出
	finalOutput := strings.Join(results, "\n")
	utils.Debug("All commands completed successfully")
	return finalOutput, nil
}

// copyAndProcessScript 複製腳本到沙盒並處理變量
func copyAndProcessScript(src, dst string, args []string, lastOutput string) error {
	// 讀取原始腳本
	content, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("Failed to read script: '%v'", err)
	}

	// 處理變量替換
	processedContent := string(content)
	for i, arg := range args {
		// 替換 *#ARG_n#* 格式的變量
		placeholder := fmt.Sprintf("*#ARG_%d#*", i+1)
		processedContent = strings.ReplaceAll(processedContent, placeholder, arg)
	}

	// 替換 *#LAST_OUTPUT#* 變量
	processedContent = strings.ReplaceAll(processedContent, "*#LAST_OUTPUT#*", lastOutput)

	// 寫入處理後的腳本到沙盒
	if err := os.WriteFile(dst, []byte(processedContent), 0755); err != nil {
		return fmt.Errorf("Failed to write processed script: '%v'", err)
	}

	return nil
}