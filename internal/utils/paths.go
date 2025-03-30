package utils

import (
	"path/filepath"
	"strings"
)

// GetBaseDir 从配置文件路径获取基础目录
// 例如: "E:/Github Desktop/PanelBase/configs/config.yaml" -> "E:/Github Desktop/PanelBase"
func GetBaseDir(configPath string) (string, error) {
	// 获取配置文件的绝对路径
	absPath, err := filepath.Abs(configPath)
	if err != nil {
		return "", err
	}

	// 如果路径包含configs目录，则返回其父目录
	if strings.Contains(absPath, "configs"+string(filepath.Separator)) {
		return filepath.Dir(filepath.Dir(absPath)), nil
	}

	// 获取目录部分
	return filepath.Dir(absPath), nil
}

// GetThemePath 获取主题资源的完整路径
func GetThemePath(baseDir, themeID, resourcePath string) string {
	return filepath.Join(baseDir, WebDir, themeID, resourcePath)
}

// GetStaticFilePath 获取静态文件的完整路径
func GetStaticFilePath(baseDir, resourcePath string) string {
	return filepath.Join(baseDir, WebDir, resourcePath)
}

// GetPluginPath 获取插件的完整路径
func GetPluginPath(baseDir, pluginID string) string {
	return filepath.Join(baseDir, PluginsDir, pluginID)
}

// GetCommandPath 获取命令的完整路径
func GetCommandPath(baseDir, commandPath string) string {
	// 如果是絕對路徑，直接返回
	if filepath.IsAbs(commandPath) {
		return commandPath
	}

	// 如果是相對路徑，拼接基礎目錄
	return filepath.Join(baseDir, commandPath)
}
