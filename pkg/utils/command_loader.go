package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CommandScript represents a script-based command
type CommandScript struct {
	Path       string
	ScriptPath string
}

// CommandManager manages script-based commands
type CommandManager struct {
	Commands map[string]*CommandScript
	BaseDir  string
}

// NewCommandManager creates a new command manager
func NewCommandManager(baseDir string) *CommandManager {
	return &CommandManager{
		Commands: make(map[string]*CommandScript),
		BaseDir:  baseDir,
	}
}

// LoadCommand loads a command script
func (cm *CommandManager) LoadCommand(commandPath string, scriptPath string) (*CommandScript, error) {
	// 检查是否使用了简化脚本引用（仅提供脚本名称）
	// 如果是简化引用方式，自动添加commands目录前缀
	var fullScriptPath string

	// 如果scriptPath不以/commands开头，则假定它是一个简化引用
	if !strings.HasPrefix(scriptPath, "/commands") {
		// 将脚本路径转换为commands目录下的完整路径
		fullScriptPath = filepath.Join(cm.BaseDir, "commands", scriptPath)
	} else {
		// 否则使用提供的完整路径（移除开头的斜杠）
		fullScriptPath = filepath.Join(cm.BaseDir, scriptPath[1:])
	}

	// 检查脚本文件是否存在
	_, err := os.Stat(fullScriptPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("script not found: %s", fullScriptPath)
	}

	// 创建命令对象
	command := &CommandScript{
		Path:       commandPath,
		ScriptPath: fullScriptPath,
	}

	// 存储命令
	cm.Commands[commandPath] = command

	return command, nil
}

// ExecuteScript executes a script and returns the output
func (cm *CommandManager) ExecuteScript(command *CommandScript, args ...string) (string, error) {
	var cmd *exec.Cmd

	// 根据扩展名决定如何执行脚本
	ext := strings.ToLower(filepath.Ext(command.ScriptPath))
	switch ext {
	case ".py":
		cmdArgs := append([]string{command.ScriptPath}, args...)
		cmd = exec.Command("python", cmdArgs...)
	case ".js":
		cmdArgs := append([]string{command.ScriptPath}, args...)
		cmd = exec.Command("node", cmdArgs...)
	case ".sh":
		cmdArgs := append([]string{command.ScriptPath}, args...)
		cmd = exec.Command("bash", cmdArgs...)
	case ".php":
		cmdArgs := append([]string{command.ScriptPath}, args...)
		cmd = exec.Command("php", cmdArgs...)
	default:
		// 对于可执行文件或未知类型，直接运行
		cmdArgs := append([]string{}, args...)
		cmd = exec.Command(command.ScriptPath, cmdArgs...)
	}

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to execute script: %w\nOutput: %s", err, string(output))
	}

	return string(output), nil
}
