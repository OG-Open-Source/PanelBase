package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	"github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/OG-Open-Source/PanelBase/internal/utils"
	"github.com/gin-gonic/gin"
)

// CommandHandler 命令處理程序
type CommandHandler struct {
	configService *services.ConfigService
}

// NewCommandHandler 創建新的命令處理程序
func NewCommandHandler(configService *services.ConfigService) *CommandHandler {
	return &CommandHandler{
		configService: configService,
	}
}

// 定義命令結構
type Command struct {
	Command string        `json:"command"`
	Args    []interface{} `json:"args"`
}

// ExecuteHandler 處理批量命令執行請求
func (h *CommandHandler) ExecuteHandler(c *gin.Context) {
	// 解析請求體中的命令列表
	var commands []Command
	if err := c.ShouldBindJSON(&commands); err != nil {
		log.Printf("解析命令列表失敗: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求數據", "details": err.Error()})
		return
	}

	if len(commands) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "命令列表為空"})
		return
	}

	log.Printf("收到執行請求，包含 %d 個命令\n", len(commands))

	// 執行結果
	results := make([]map[string]interface{}, 0, len(commands))

	// 執行每個命令
	for i, cmd := range commands {
		log.Printf("執行命令 %d: %s\n", i+1, cmd.Command)

		// 查找對應的命令腳本路徑
		scriptPath := h.configService.CommandsConfig.GetCommandPath(cmd.Command)
		if scriptPath == "" {
			log.Printf("未找到命令: %s\n", cmd.Command)
			results = append(results, map[string]interface{}{
				"command": cmd.Command,
				"success": false,
				"error":   "未找到該命令對應的腳本配置",
			})
			continue
		}

		// 構建完整命令腳本路徑
		fullScriptPath := filepath.Join(h.configService.BaseDir, scriptPath)

		// 確保目錄存在
		scriptDir := filepath.Dir(fullScriptPath)
		if err := os.MkdirAll(scriptDir, 0755); err != nil {
			log.Printf("創建命令目錄失敗: %v\n", err)
			results = append(results, map[string]interface{}{
				"command": cmd.Command,
				"success": false,
				"error":   fmt.Sprintf("創建命令目錄失敗: %v", err),
			})
			continue
		}

		// 檢查命令腳本文件是否存在
		if _, err := os.Stat(fullScriptPath); os.IsNotExist(err) {
			// 創建簡單的Python腳本示例
			if err := createExampleScript(fullScriptPath); err != nil {
				log.Printf("創建示例腳本失敗: %v\n", err)
				results = append(results, map[string]interface{}{
					"command": cmd.Command,
					"success": false,
					"error":   fmt.Sprintf("腳本不存在且創建示例腳本失敗: %v", err),
				})
				continue
			}
			log.Printf("已創建示例腳本: %s\n", fullScriptPath)
		}

		// 構建Python命令行參數
		cmdArgs := []string{fullScriptPath}

		// 添加參數
		for i, arg := range cmd.Args {
			// 格式化參數名稱 (arg1, arg2, ...)
			paramName := fmt.Sprintf("arg%d", i+1)
			// 添加參數
			cmdArgs = append(cmdArgs, fmt.Sprintf("--%s=%v", paramName, arg))
		}

		// 執行腳本
		result, err := executeScript(cmdArgs)
		if err != nil {
			log.Printf("執行腳本失敗: %v\n", err)
			results = append(results, map[string]interface{}{
				"command": cmd.Command,
				"success": false,
				"error":   err.Error(),
			})
			continue
		}

		// 解析JSON結果
		var jsonResult map[string]interface{}
		if err := json.Unmarshal([]byte(result), &jsonResult); err != nil {
			// 如果不是有效的JSON，則作為純文本返回
			results = append(results, map[string]interface{}{
				"command": cmd.Command,
				"success": true,
				"result":  result,
			})
		} else {
			// 如果是有效的JSON，則直接返回解析後的JSON
			jsonResult["command"] = cmd.Command
			jsonResult["success"] = true
			results = append(results, jsonResult)
		}
	}

	// 返回所有命令的執行結果
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"results": results,
	})
}

// executeScript 執行腳本並返回結果
func executeScript(args []string) (string, error) {
	// 確定Python可執行文件路徑
	pythonPath := "python"
	if utils.IsWindows() {
		pythonPath = "python.exe"
	}

	// 創建命令
	cmd := exec.Command(pythonPath, args...)

	// 捕獲標準輸出和錯誤
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// 執行命令
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("執行腳本失敗: %v\n%s", err, stderr.String())
	}

	return stdout.String(), nil
}

// createExampleScript 創建示例Python腳本
func createExampleScript(scriptPath string) error {
	// 確保目錄存在
	dir := filepath.Dir(scriptPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 示例腳本內容
	script := `#!/usr/bin/env python3
import json
import argparse
import sys

# 解析命令行參數
parser = argparse.ArgumentParser(description='示例腳本')
# 根據腳本類型添加不同的參數
if 'get_user' in sys.argv[0]:
		parser.add_argument('--user_id', required=True, help='用戶ID')
elif 'update_product' in sys.argv[0]:
		parser.add_argument('--product_id', required=True, help='產品ID')
		parser.add_argument('--name', required=True, help='產品名稱')
		parser.add_argument('--price', required=True, help='產品價格')
elif 'delete_comment' in sys.argv[0]:
		parser.add_argument('--comment_id', required=True, help='評論ID')
else:
		# 通用參數
		parser.add_argument('--param1', help='參數1')
		parser.add_argument('--param2', help='參數2')

args = parser.parse_args()

# 示例響應
if 'get_user' in sys.argv[0]:
		response = {
				"user": {
						"id": args.user_id,
						"username": f"user_{args.user_id}",
						"email": f"user_{args.user_id}@example.com",
						"created_at": "2023-08-01T10:00:00Z"
				}
		}
elif 'update_product' in sys.argv[0]:
		response = {
				"product": {
						"id": args.product_id,
						"name": args.name,
						"price": args.price,
						"updated_at": "2023-08-01T10:00:00Z"
				}
		}
elif 'delete_comment' in sys.argv[0]:
		response = {
				"message": f"評論 {args.comment_id} 已刪除",
				"deleted_at": "2023-08-01T10:00:00Z"
		}
else:
		response = {
				"message": "示例腳本執行成功",
				"args": vars(args)
		}

# 輸出JSON格式的結果
print(json.dumps(response, ensure_ascii=False))
`

	// 寫入文件
	return os.WriteFile(scriptPath, []byte(script), 0755)
}

// HandleCommandsV1 處理命令接口請求 (V1 API)
func (h *CommandHandler) HandleCommandsV1(c *gin.Context) {
	method := c.Request.Method
	switch method {
	case "GET":
		// 判斷是獲取所有命令還是特定命令
		var request struct {
			ID string `json:"id"`
		}
		if err := c.ShouldBindJSON(&request); err == nil && request.ID != "" {
			// 獲取特定命令
			h.GetCommandByIDV1(c, request.ID)
		} else {
			// 獲取所有命令
			h.GetCommandsV1(c)
		}
	case "POST":
		// 安裝新命令
		h.InstallCommandV1(c)
	case "PUT":
		// 更新全部命令列表
		h.UpdateAllCommandsV1(c)
	case "PATCH":
		// 更新指定命令
		h.UpdateSingleCommandV1(c)
	case "DELETE":
		// 判斷是刪除所有命令還是特定命令
		var request struct {
			ID string `json:"id"`
		}
		if err := c.ShouldBindJSON(&request); err == nil && request.ID != "" {
			// 刪除特定命令
			h.DeleteCommandV1(c, request.ID)
		} else {
			// 刪除所有命令
			h.DeleteAllCommandsV1(c)
		}
	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "不支持的請求方法"})
	}
}

// GetCommandsV1 處理GET方法獲取所有命令
func (h *CommandHandler) GetCommandsV1(c *gin.Context) {
	commandsList := make([]map[string]interface{}, 0)

	// 遍歷命令配置
	for id, commandConfig := range h.configService.CommandsConfig.Routes {
		// 添加命令信息
		commandsList = append(commandsList, map[string]interface{}{
			"id":          id,
			"path":        commandConfig.Path,
			"method":      commandConfig.Method,
			"script":      commandConfig.Script,
			"description": commandConfig.Description,
			"params":      commandConfig.Params,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"total":   len(commandsList),
		"data":    commandsList,
	})
}

// GetCommandByIDV1 獲取特定ID的命令 (V1 API)
func (h *CommandHandler) GetCommandByIDV1(c *gin.Context, commandID string) {
	// 檢查命令是否存在
	commandConfig, exists := h.configService.CommandsConfig.Routes[commandID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "未找到該命令",
		})
		return
	}

	// 返回命令信息
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": map[string]interface{}{
			"id":          commandID,
			"path":        commandConfig.Path,
			"method":      commandConfig.Method,
			"script":      commandConfig.Script,
			"description": commandConfig.Description,
			"params":      commandConfig.Params,
		},
	})
}

// InstallCommandV1 安裝新命令 (V1 API)
func (h *CommandHandler) InstallCommandV1(c *gin.Context) {
	var request struct {
		URL string `json:"url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求參數", "details": err.Error()})
		return
	}

	// 驗證URL
	if !strings.HasPrefix(request.URL, "http://") && !strings.HasPrefix(request.URL, "https://") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的URL", "details": "URL必須以http://或https://開頭"})
		return
	}

	// 獲取命令腳本
	resp, err := http.Get(request.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法下載命令腳本", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法下載命令腳本", "details": fmt.Sprintf("伺服器回應: %s", resp.Status)})
		return
	}

	// 讀取腳本內容
	scriptData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法讀取命令腳本", "details": err.Error()})
		return
	}

	// 解析腳本元數據
	script := string(scriptData)
	lines := strings.Split(script, "\n")
	if len(lines) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的命令腳本格式", "details": "腳本太短，無法解析元數據"})
		return
	}

	// 解析命令ID和其他元數據
	commandID := ""
	for i := 0; i < min(10, len(lines)); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "# @command ") {
			commandID = strings.TrimSpace(line[len("# @command "):])
			break
		}
	}

	if commandID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的命令腳本格式", "details": "腳本必須包含 # @command 命令ID 標記"})
		return
	}

	// 創建插件目錄
	pluginsDir := filepath.Join(h.configService.BaseDir, "plugins")
	if err := os.MkdirAll(pluginsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法創建插件目錄", "details": err.Error()})
		return
	}

	// 生成腳本路徑
	scriptPath := filepath.Join(pluginsDir, commandID+".sh")

	// 保存腳本
	if err := os.WriteFile(scriptPath, scriptData, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存命令腳本", "details": err.Error()})
		return
	}

	// 更新配置
	relPath, err := filepath.Rel(h.configService.BaseDir, scriptPath)
	if err != nil {
		relPath = scriptPath
	}

	// 確保Routes映射已初始化
	if h.configService.CommandsConfig.Routes == nil {
		h.configService.CommandsConfig.Routes = make(map[string]models.CommandConfig)
	}

	// 添加新命令
	h.configService.CommandsConfig.Routes[commandID] = models.CommandConfig{
		Path:        "/api/command/" + commandID,
		Method:      "POST",
		Script:      relPath,
		Description: "從URL安裝的命令",
		Params:      make(map[string]string),
	}

	// 保存配置
	commandsPath := filepath.Join(h.configService.BaseDir, utils.CommandsFile)
	commandsData, err := json.MarshalIndent(h.configService.CommandsConfig, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法序列化命令配置", "details": err.Error()})
		return
	}

	if err := os.WriteFile(commandsPath, commandsData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存命令配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "命令安裝成功",
		"command": map[string]interface{}{
			"id":     commandID,
			"path":   "/api/command/" + commandID,
			"method": "POST",
			"script": relPath,
		},
	})
}

// UpdateAllCommandsV1 更新全部命令列表 (V1 API)
func (h *CommandHandler) UpdateAllCommandsV1(c *gin.Context) {
	// 更新全部命令列表
	updatedCommands := make(map[string]interface{})

	// 遍歷所有命令，檢查更新
	for commandID, commandConfig := range h.configService.CommandsConfig.Routes {
		scriptPath := commandConfig.Script

		// 獲取腳本完整路徑
		fullScriptPath := filepath.Join(h.configService.BaseDir, scriptPath)

		// 檢查腳本是否存在
		if _, err := os.Stat(fullScriptPath); os.IsNotExist(err) {
			log.Printf("命令 %s 的腳本 %s 不存在，跳過更新", commandID, scriptPath)
			updatedCommands[commandID] = "未更新: 腳本文件不存在"
			continue
		}

		// 讀取腳本內容
		scriptData, err := os.ReadFile(fullScriptPath)
		if err != nil {
			log.Printf("無法讀取命令 %s 的腳本: %v", commandID, err)
			updatedCommands[commandID] = "未更新: 無法讀取腳本文件"
			continue
		}

		// 解析腳本元數據
		script := string(scriptData)
		lines := strings.Split(script, "\n")

		// 解析元數據
		sourceLink := ""
		version := ""
		for i := 0; i < min(10, len(lines)); i++ {
			line := strings.TrimSpace(lines[i])
			if strings.HasPrefix(line, "# @source_link ") {
				sourceLink = strings.TrimSpace(line[len("# @source_link "):])
			} else if strings.HasPrefix(line, "# @version ") {
				version = strings.TrimSpace(line[len("# @version "):])
			}
		}

		if sourceLink == "" {
			log.Printf("命令 %s 未設置源鏈接，跳過更新", commandID)
			updatedCommands[commandID] = "未更新: 未設置源鏈接"
			continue
		}

		if version == "" {
			log.Printf("命令 %s 未設置版本號，跳過更新", commandID)
			updatedCommands[commandID] = "未更新: 未設置版本號"
			continue
		}

		// 從URL獲取最新的命令腳本
		resp, err := http.Get(sourceLink)
		if err != nil {
			log.Printf("無法獲取命令 %s 的更新: %v", commandID, err)
			updatedCommands[commandID] = "未更新: 無法獲取更新"
			continue
		}

		if resp.StatusCode != http.StatusOK {
			log.Printf("無法下載命令 %s 的腳本: %s", commandID, resp.Status)
			resp.Body.Close()
			updatedCommands[commandID] = "未更新: 下載失敗"
			continue
		}

		// 讀取新腳本內容
		newScriptData, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			log.Printf("無法讀取命令 %s 的新腳本: %v", commandID, err)
			updatedCommands[commandID] = "未更新: 無法讀取新腳本"
			continue
		}

		// 解析新腳本元數據
		newScript := string(newScriptData)
		newLines := strings.Split(newScript, "\n")

		// 解析新版本
		newVersion := ""
		for i := 0; i < min(10, len(newLines)); i++ {
			line := strings.TrimSpace(newLines[i])
			if strings.HasPrefix(line, "# @version ") {
				newVersion = strings.TrimSpace(line[len("# @version "):])
				break
			}
		}

		if newVersion == "" {
			log.Printf("命令 %s 的新腳本未設置版本號，跳過更新", commandID)
			updatedCommands[commandID] = "未更新: 新腳本未設置版本號"
			continue
		}

		// 比較版本號
		if newVersion == version {
			log.Printf("命令 %s 已是最新版本: %s", commandID, version)
			updatedCommands[commandID] = "未更新: 已是最新版本"
			continue
		}

		log.Printf("命令 %s 有更新: %s -> %s", commandID, version, newVersion)

		// 保存新腳本
		if err := os.WriteFile(fullScriptPath, newScriptData, 0755); err != nil {
			log.Printf("無法保存命令 %s 的更新腳本: %v", commandID, err)
			updatedCommands[commandID] = "未更新: 無法保存新腳本"
			continue
		}

		// 更新命令版本信息
		updatedCommands[commandID] = map[string]string{
			"status":      "已更新",
			"old_version": version,
			"new_version": newVersion,
		}

		log.Printf("命令 %s 更新成功", commandID)
	}

	// 保存配置
	commandsPath := filepath.Join(h.configService.BaseDir, utils.CommandsFile)
	commandsData, err := json.MarshalIndent(h.configService.CommandsConfig, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法序列化命令配置", "details": err.Error()})
		return
	}

	if err := os.WriteFile(commandsPath, commandsData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存命令配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":   "success",
		"message":  "全部命令更新檢查完成",
		"commands": updatedCommands,
	})
}

// UpdateSingleCommandV1 更新指定命令 (V1 API)
func (h *CommandHandler) UpdateSingleCommandV1(c *gin.Context) {
	var request struct {
		ID string `json:"id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無效的請求參數", "details": err.Error()})
		return
	}

	commandID := request.ID

	// 檢查命令是否存在
	commandConfig, exists := h.configService.CommandsConfig.Routes[commandID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "命令不存在", "id": commandID})
		return
	}

	scriptPath := commandConfig.Script

	// 獲取腳本完整路徑
	fullScriptPath := filepath.Join(h.configService.BaseDir, scriptPath)

	// 檢查腳本是否存在
	if _, err := os.Stat(fullScriptPath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "命令腳本不存在", "path": scriptPath})
		return
	}

	// 讀取腳本內容
	scriptData, err := os.ReadFile(fullScriptPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法讀取命令腳本", "details": err.Error()})
		return
	}

	// 解析腳本元數據
	script := string(scriptData)
	lines := strings.Split(script, "\n")

	// 解析元數據
	sourceLink := ""
	version := ""
	for i := 0; i < min(10, len(lines)); i++ {
		line := strings.TrimSpace(lines[i])
		if strings.HasPrefix(line, "# @source_link ") {
			sourceLink = strings.TrimSpace(line[len("# @source_link "):])
		} else if strings.HasPrefix(line, "# @version ") {
			version = strings.TrimSpace(line[len("# @version "):])
		}
	}

	if sourceLink == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "命令未設置源鏈接", "id": commandID})
		return
	}

	// 從URL獲取最新的命令腳本
	resp, err := http.Get(sourceLink)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法獲取命令更新", "details": err.Error()})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法下載命令腳本", "details": fmt.Sprintf("伺服器回應: %s", resp.Status)})
		return
	}

	// 讀取新腳本內容
	newScriptData, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "無法讀取命令新腳本", "details": err.Error()})
		return
	}

	// 解析新腳本元數據
	newScript := string(newScriptData)
	newLines := strings.Split(newScript, "\n")

	// 解析新版本
	newVersion := ""
	for i := 0; i < min(10, len(newLines)); i++ {
		line := strings.TrimSpace(newLines[i])
		if strings.HasPrefix(line, "# @version ") {
			newVersion = strings.TrimSpace(line[len("# @version "):])
			break
		}
	}

	if newVersion == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "命令新腳本未設置版本號", "id": commandID})
		return
	}

	// 比較版本號
	if version != "" && newVersion == version {
		c.JSON(http.StatusOK, gin.H{
			"status":  "success",
			"message": "命令已是最新版本",
			"command": map[string]string{
				"id":      commandID,
				"script":  scriptPath,
				"version": version,
			},
		})
		return
	}

	// 保存新腳本
	if err := os.WriteFile(fullScriptPath, newScriptData, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存命令更新腳本", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "命令更新成功",
		"command": map[string]string{
			"id":          commandID,
			"script":      scriptPath,
			"old_version": version,
			"new_version": newVersion,
		},
	})
}

// DeleteCommandV1 刪除指定命令 (V1 API)
func (h *CommandHandler) DeleteCommandV1(c *gin.Context, commandID string) {
	// 檢查命令是否存在
	commandConfig, exists := h.configService.CommandsConfig.Routes[commandID]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "命令不存在", "id": commandID})
		return
	}

	scriptPath := commandConfig.Script

	// 刪除腳本文件
	fullScriptPath := filepath.Join(h.configService.BaseDir, scriptPath)
	if err := os.Remove(fullScriptPath); err != nil && !os.IsNotExist(err) {
		log.Printf("無法刪除命令 %s 的腳本: %v", commandID, err)
	}

	// 從配置中刪除命令
	delete(h.configService.CommandsConfig.Routes, commandID)

	// 保存配置
	commandsPath := filepath.Join(h.configService.BaseDir, utils.CommandsFile)
	commandsData, err := json.MarshalIndent(h.configService.CommandsConfig, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法序列化命令配置", "details": err.Error()})
		return
	}

	if err := os.WriteFile(commandsPath, commandsData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存命令配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "命令刪除成功",
		"id":      commandID,
	})
}

// DeleteAllCommandsV1 刪除所有命令 (V1 API)
func (h *CommandHandler) DeleteAllCommandsV1(c *gin.Context) {
	// 遍歷所有命令
	for commandID, commandConfig := range h.configService.CommandsConfig.Routes {
		scriptPath := commandConfig.Script

		// 刪除腳本文件
		fullScriptPath := filepath.Join(h.configService.BaseDir, scriptPath)
		if err := os.Remove(fullScriptPath); err != nil && !os.IsNotExist(err) {
			log.Printf("無法刪除命令 %s 的腳本: %v", commandID, err)
		}
	}

	// 清空命令列表
	h.configService.CommandsConfig.Routes = make(map[string]models.CommandConfig)

	// 保存配置
	commandsPath := filepath.Join(h.configService.BaseDir, utils.CommandsFile)
	commandsData, err := json.MarshalIndent(h.configService.CommandsConfig, "", "  ")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法序列化命令配置", "details": err.Error()})
		return
	}

	if err := os.WriteFile(commandsPath, commandsData, 0644); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "無法保存命令配置", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "已刪除所有命令",
	})
}

// min 返回兩個整數中較小的一個
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
