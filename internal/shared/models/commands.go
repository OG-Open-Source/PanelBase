package models

// CommandsConfig 命令配置映射，键为命令名称，值为脚本路径
type CommandsConfig map[string]string

// GetCommandPath 获取命令对应的脚本路径
func (c CommandsConfig) GetCommandPath(command string) string {
	if path, ok := c[command]; ok {
		return path
	}
	return ""
}

// HasCommand 检查命令是否存在
func (c CommandsConfig) HasCommand(command string) bool {
	_, exists := c[command]
	return exists
}

// GetCommands 获取所有命令名称
func (c CommandsConfig) GetCommands() []string {
	commands := make([]string, 0, len(c))
	for cmd := range c {
		commands = append(commands, cmd)
	}
	return commands
}
