package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"time"
)

// 以下常量定义了系统的主要目录结构
const (
	// 相对于基础路径的目录
	WebDir         = "web"
	ThemeAssetsDir = "theme"
	PluginsDir     = "plugins"
	CommandsDir    = "commands"
	ConfigsDir     = "configs"
)

// 配置文件路径
const (
	// 路由配置文件
	CommandsFile = "configs/commands.json"
	// 用户配置文件
	UsersFile = "configs/users.json"
	// 主题配置文件
	ThemesFile = "configs/themes.json"
	// 插件配置文件
	PluginsFile = "configs/plugins.json"
)

// API版本和路径前缀
const (
	// API版本
	APIVersion = "v1"
	// API路径前缀
	APIPathPrefix = "/api/" + APIVersion
	// 插件API路径前缀
	PluginPathPrefix = APIPathPrefix + "/plugins"
)

// 默认值
const (
	// 默认端口
	DefaultPort = 8080
	// 默认主题
	DefaultTheme = "default"
	// 默认用户
	DefaultUser = "admin"
	// 默认配置文件
	ConfigFile = "config.yaml"
)

// 角色权限映射
var rolePermissions = map[string][]string{
	"admin": {
		"user:read", "user:write", "user:delete",
		"theme:read", "theme:write", "theme:delete",
		"plugin:read", "plugin:write", "plugin:delete",
		"script:read", "script:write", "script:delete",
		"system:read", "system:write",
	},
	"user": {
		"user:read",
		"theme:read",
		"plugin:read",
		"script:read",
	},
	"guest": {
		"theme:read",
	},
}

// HashPassword 对密码进行哈希
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// VerifyPassword 验证密码
func VerifyPassword(password, hashedPassword string) bool {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:]) == hashedPassword
}

// GenerateUUID 生成UUID
func GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GetRolePermissions 获取角色权限
func GetRolePermissions(role string) []string {
	if perms, ok := rolePermissions[role]; ok {
		return perms
	}
	return []string{}
}

// JoinPaths 连接路径
func JoinPaths(base string, parts ...string) string {
	result := base
	for _, part := range parts {
		result = filepath.Join(result, part)
	}
	return result
}
