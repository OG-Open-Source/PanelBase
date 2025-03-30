package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/types"
	"github.com/OG-Open-Source/PanelBase/internal/common"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID        string           `json:"id"`
	IsActive  bool             `json:"is_active"`
	Username  string           `json:"username"`
	Password  string           `json:"password"`
	Role      string           `json:"role"`
	Name      string           `json:"name"`
	Email     string           `json:"email"`
	CreatedAt time.Time        `json:"created_at"`
	LastLogin time.Time        `json:"last_login"`
	API       []types.APIToken `json:"api"`
}

// UsersConfig holds the users configuration
type UsersConfig struct {
	Users       map[string]*User `json:"users"`
	DefaultRole string           `json:"default_role"`
	JWTSecret   string           `json:"jwt_secret"`
}

// NewUsersConfig 创建新的用户配置
func NewUsersConfig() *UsersConfig {
	return &UsersConfig{
		Users: make(map[string]*User),
	}
}

// LoadUsersConfig 从文件加载用户配置
func LoadUsersConfig(baseDir string) (*UsersConfig, error) {
	config := NewUsersConfig()
	configPath := filepath.Join(baseDir, common.ConfigDir, common.UsersConfigFile)

	// 如果配置文件不存在，创建默认管理员用户
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config.Users["admin"] = &User{
			ID:        "admin",
			IsActive:  true,
			Username:  "admin",
			Password:  common.HashPassword("admin"), // 默认密码
			Role:      "admin",
			Name:      "Administrator",
			Email:     "admin@example.com",
			CreatedAt: time.Now(),
			LastLogin: time.Now(),
		}
		return config, config.Save(baseDir)
	}

	// 读取配置文件
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("读取用户配置文件失败: %w", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析用户配置文件失败: %w", err)
	}

	return config, nil
}

// Save 保存用户配置到文件
func (c *UsersConfig) Save(baseDir string) error {
	configPath := filepath.Join(baseDir, common.ConfigDir, common.UsersConfigFile)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化用户配置失败: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("保存用户配置文件失败: %w", err)
	}

	return nil
}

// GetUser 获取用户信息
func (c *UsersConfig) GetUser(username string) (*User, error) {
	if user, exists := c.Users[username]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("用户不存在: %s", username)
}

// AddUser 添加新用户
func (c *UsersConfig) AddUser(user *User) error {
	if _, exists := c.Users[user.Username]; exists {
		return fmt.Errorf("用户已存在: %s", user.Username)
	}

	c.Users[user.Username] = user
	return nil
}

// UpdateUser 更新用户信息
func (c *UsersConfig) UpdateUser(user *User) error {
	if _, exists := c.Users[user.Username]; !exists {
		return fmt.Errorf("用户不存在: %s", user.Username)
	}

	c.Users[user.Username] = user
	return nil
}

// DeleteUser 删除用户
func (c *UsersConfig) DeleteUser(username string) error {
	if _, exists := c.Users[username]; !exists {
		return fmt.Errorf("用户不存在: %s", username)
	}

	delete(c.Users, username)
	return nil
}

// CreateAPIToken creates a new API token for the user
func (u *User) CreateAPIToken(name string, permissions []string, expiresIn int, rateLimit int) (string, error) {
	token := &types.APIToken{
		ID:            common.GenerateUUID(),
		Name:          name,
		Permissions:   permissions,
		RateLimit:     rateLimit,
		Expiration:    expiresIn,
		CreatedAt:     time.Now(),
		LastUsed:      time.Now(),
		UsageCount:    0,
		IsActive:      true,
		LastResetTime: time.Now(),
	}

	// 生成JWT token
	claims := &types.APIClaims{
		UserID: u.ID,
		APIID:  token.ID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Duration(expiresIn) * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := jwtToken.SignedString([]byte(common.APISecretKey)) // 使用固定密钥，实际应用中应使用配置文件中的密钥
	if err != nil {
		return "", err
	}

	token.Token = tokenString
	u.API = append(u.API, *token)
	return tokenString, nil
}

// GetAPITokens returns all API tokens for the user
func (u *User) GetAPITokens() []types.APIToken {
	return u.API
}

// UpdateAPIToken updates an existing API token
func (u *User) UpdateAPIToken(tokenID string, name string, permissions []string, expiresIn int, rateLimit int) error {
	for i := range u.API {
		if u.API[i].ID == tokenID {
			u.API[i].Name = name
			u.API[i].Permissions = permissions
			u.API[i].Expiration = expiresIn
			u.API[i].RateLimit = rateLimit
			return nil
		}
	}
	return errors.New("API token not found")
}

// DeleteAPIToken deletes an API token
func (u *User) DeleteAPIToken(tokenID string) error {
	for i := range u.API {
		if u.API[i].ID == tokenID {
			u.API = append(u.API[:i], u.API[i+1:]...)
			return nil
		}
	}
	return errors.New("API token not found")
}

// ResetAPITokenUsage resets the usage statistics of an API token
func (u *User) ResetAPITokenUsage(tokenID string) error {
	for i := range u.API {
		if u.API[i].ID == tokenID {
			u.API[i].UsageCount = 0
			u.API[i].LastResetTime = time.Now()
			return nil
		}
	}
	return errors.New("API token not found")
}

// VerifyPassword verifies the user's password
func (u *User) VerifyPassword(password string) bool {
	// 检测是否为bcrypt格式的密码 (以$2a$、$2b$或$2y$开头)
	if len(u.Password) > 4 && (u.Password[:4] == "$2a$" || u.Password[:4] == "$2b$" || u.Password[:4] == "$2y$") {
		// 使用golang.org/x/crypto/bcrypt包验证bcrypt格式密码
		err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
		return err == nil
	}

	// 默认使用SHA-256验证
	return common.VerifyPassword(password, u.Password)
}

// GenerateToken generates a JWT token for the user
func (u *User) GenerateToken(secret string) (string, error) {
	claims := &types.JWTClaims{
		UserID: u.ID,
		Role:   u.Role,
		Name:   u.Name,
		Email:  u.Email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// HasPermission 检查用户是否具有指定权限
func (u *User) HasPermission(permission string) bool {
	// 管理员拥有所有权限
	if u.Role == "admin" {
		return true
	}

	// 检查用户角色权限
	rolePermissions := common.GetRolePermissions(u.Role)
	for _, p := range rolePermissions {
		if p == permission {
			return true
		}
	}

	return false
}
