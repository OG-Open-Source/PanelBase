package common

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// Constants
const (
	// ConfigDir 配置文件目录
	ConfigDir = "configs"

	// UsersConfigFile 用户配置文件
	UsersConfigFile = "users.json"

	// ThemesConfigFile 主题配置文件
	ThemesConfigFile = "themes.json"

	// PluginsConfigFile 插件配置文件
	PluginsConfigFile = "plugins.json"

	// ConfigFile 主配置文件
	ConfigFile = "config.yaml"

	// WebDir Web目录
	WebDir = "web"

	// ThemeAssetsDir 主题资源目录
	ThemeAssetsDir = "theme"

	// APISecretKey API密钥
	APISecretKey = "api_secret_key" // 实际应用中应从配置文件读取
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
	// 使用bcrypt生成更安全的密码哈希
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// 如果bcrypt失败，回退到SHA-256
		hash := sha256.Sum256([]byte(password))
		return hex.EncodeToString(hash[:])
	}
	return string(hashedBytes)
}

// VerifyPassword 验证密码
func VerifyPassword(password, hashedPassword string) bool {
	// 检测是否为bcrypt格式的密码 (以$2a$、$2b$或$2y$开头)
	if len(hashedPassword) > 4 && (hashedPassword[:4] == "$2a$" || hashedPassword[:4] == "$2b$" || hashedPassword[:4] == "$2y$") {
		// 使用bcrypt验证
		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
		return err == nil
	}

	// 默认使用SHA-256验证
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
