package bootstrap

import (
	crand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	mrand "math/rand"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
	"golang.org/x/crypto/bcrypt"
)

// Configs 定义所有配置文件的默认内容
type Configs struct {
	Themes   map[string]interface{}
	Commands map[string]interface{}
	Plugins  map[string]interface{}
	Users    map[string]interface{}
	Config   map[string]interface{}
}

// NewConfigs 创建默认配置
func NewConfigs() *Configs {
	return &Configs{
		Themes: map[string]interface{}{
			"themes":        map[string]interface{}{},
			"current_theme": "",
		},
		Commands: map[string]interface{}{
			"commands": map[string]interface{}{},
		},
		Plugins: map[string]interface{}{
			"plugins": map[string]interface{}{},
		},
		Users: map[string]interface{}{
			"jwt_secret": "", // 将在 Bootstrap 时设置
			"users":      map[string]interface{}{},
		},
		Config: map[string]interface{}{
			"server": map[string]interface{}{
				"host": "0.0.0.0",
				"port": 0, // 将在 Bootstrap 时设置
				"mode": "debug",
			},
			"features": map[string]interface{}{
				"commands": false,
				"plugins":  true,
			},
			"auth": map[string]interface{}{
				"jwt_expiration": 24,
				"cookie_name":    "panelbase_jwt",
			},
		},
	}
}

// Bootstrap 检查并创建必要的配置文件
func Bootstrap() error {
	// 确保 configs 目录存在
	if err := ensureConfigsDir(); err != nil {
		return fmt.Errorf("failed to ensure configs directory: %w", err)
	}

	// 创建默认配置
	configs := NewConfigs()

	// 生成随机 JWT 密钥
	jwtSecret, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("failed to generate JWT secret: %w", err)
	}
	configs.Users["jwt_secret"] = jwtSecret

	// 生成随机端口 (1024-49151) 并检查是否被占用
	port, err := findAvailablePort(1024, 49151)
	if err != nil {
		return fmt.Errorf("failed to find available port: %w", err)
	}
	configs.Config["server"].(map[string]interface{})["port"] = port

	// 生成随机用户ID
	userID, err := generateRandomString(8)
	if err != nil {
		return fmt.Errorf("failed to generate user ID: %w", err)
	}
	userID = "usr_" + userID

	// 设置管理员密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash admin password: %w", err)
	}

	// 添加管理员用户
	configs.Users["users"].(map[string]interface{})[userID] = map[string]interface{}{
		"is_active":  true,
		"username":   "admin",
		"password":   string(hashedPassword),
		"name":       "Administrator",
		"email":      "admin@example.com",
		"created_at": time.Now().Format(time.RFC3339),
		"scopes":     []string{"*:*:*"},
		"api": map[string]interface{}{
			"tokens": []map[string]interface{}{},
		},
	}

	// 创建配置文件
	if err := createConfigFile("themes.json", configs.Themes); err != nil {
		return err
	}
	if err := createConfigFile("commands.json", configs.Commands); err != nil {
		return err
	}
	if err := createConfigFile("plugins.json", configs.Plugins); err != nil {
		return err
	}
	if err := createConfigFile("users.json", configs.Users); err != nil {
		return err
	}
	if err := createConfigFile("config.toml", configs.Config); err != nil {
		return err
	}

	return nil
}

// ensureConfigsDir 确保 configs 目录存在
func ensureConfigsDir() error {
	configsDir := "configs"
	if _, err := os.Stat(configsDir); os.IsNotExist(err) {
		return os.MkdirAll(configsDir, 0755)
	}
	return nil
}

// createConfigFile 创建配置文件
func createConfigFile(filename string, data interface{}) error {
	filepath := filepath.Join("configs", filename)

	// 检查文件是否已存在
	if _, err := os.Stat(filepath); err == nil {
		return nil // 文件已存在，跳过
	}

	// 创建文件
	file, err := os.Create(filepath)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", filename, err)
	}
	defer file.Close()

	// 根据文件类型选择编码方式
	switch filename {
	case "config.toml":
		encoder := toml.NewEncoder(file)
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	default:
		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ") // 使用两个空格缩进
		if err := encoder.Encode(data); err != nil {
			return fmt.Errorf("failed to write %s: %w", filename, err)
		}
	}

	return nil
}

// generateRandomString 生成随机字符串
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	n, err := crand.Read(bytes)
	if err != nil {
		return "", err
	}
	if n != length {
		return "", fmt.Errorf("failed to generate random string: expected %d bytes, got %d", length, n)
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

// findAvailablePort 在指定范围内查找可用端口
func findAvailablePort(start, end int) (int, error) {
	// 设置随机种子
	mrand.Seed(time.Now().UnixNano())

	// 最多尝试 100 次
	maxAttempts := 100
	for i := 0; i < maxAttempts; i++ {
		// 生成随机端口
		port := start + mrand.Intn(end-start+1)

		// 检查端口是否被占用
		addr := fmt.Sprintf(":%d", port)
		listener, err := net.Listen("tcp", addr)
		if err == nil {
			// 端口可用,关闭监听器并返回端口号
			listener.Close()
			return port, nil
		}

		// 如果错误不是端口被占用,则返回错误
		if !isPortInUseError(err) {
			return 0, fmt.Errorf("failed to check port %d: %w", port, err)
		}

		// 端口被占用,继续尝试下一个
		continue
	}

	return 0, fmt.Errorf("failed to find available port after %d attempts", maxAttempts)
}

// isPortInUseError 检查错误是否为端口被占用
func isPortInUseError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "address already in use") ||
		strings.Contains(err.Error(), "bind: address already in use")
}
