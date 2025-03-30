package models

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// 持续时间部分的正则表达式
var (
	durationRegex = regexp.MustCompile(`^P(?:(\d+)Y)?(?:(\d+)M)?(?:(\d+)D)?(?:T(?:(\d+)H)?(?:(\d+)M)?(?:(\d+(?:\.\d+)?)S)?)?$`)
	weekRegex     = regexp.MustCompile(`^P(\d+)W$`)
)

// ParseISO8601Duration 解析ISO 8601格式的持续时间字符串
// 支持格式如: P1Y2M3DT4H5M6S, PT1H, P1D, P1W等
// 如果无法解析，则返回错误
func ParseISO8601Duration(durationStr string) (time.Duration, error) {
	// 处理空字符串
	if durationStr == "" {
		return 0, fmt.Errorf("空的持续时间字符串")
	}

	// 处理周格式 (PnW)
	if match := weekRegex.FindStringSubmatch(durationStr); match != nil {
		weeks, _ := strconv.Atoi(match[1])
		return time.Duration(weeks) * 7 * 24 * time.Hour, nil
	}

	// 处理标准格式 (PnYnMnDTnHnMnS)
	match := durationRegex.FindStringSubmatch(durationStr)
	if match == nil {
		return 0, fmt.Errorf("无效的ISO 8601持续时间格式: %s", durationStr)
	}

	// 解析各个部分
	var duration time.Duration

	// 年
	if match[1] != "" {
		years, _ := strconv.Atoi(match[1])
		duration += time.Duration(years) * 365 * 24 * time.Hour
	}

	// 月
	if match[2] != "" {
		months, _ := strconv.Atoi(match[2])
		duration += time.Duration(months) * 30 * 24 * time.Hour
	}

	// 日
	if match[3] != "" {
		days, _ := strconv.Atoi(match[3])
		duration += time.Duration(days) * 24 * time.Hour
	}

	// 时
	if match[4] != "" {
		hours, _ := strconv.Atoi(match[4])
		duration += time.Duration(hours) * time.Hour
	}

	// 分
	if match[5] != "" {
		minutes, _ := strconv.Atoi(match[5])
		duration += time.Duration(minutes) * time.Minute
	}

	// 秒
	if match[6] != "" {
		seconds, _ := strconv.ParseFloat(match[6], 64)
		duration += time.Duration(seconds * float64(time.Second))
	}

	return duration, nil
}

// GetDurationOrDefault 从字符串解析持续时间，如果解析失败则返回默认值（秒）
func GetDurationOrDefault(durationStr string, defaultSeconds int) time.Duration {
	if durationStr == "" {
		return time.Duration(defaultSeconds) * time.Second
	}

	// 尝试解析ISO 8601格式
	duration, err := ParseISO8601Duration(durationStr)
	if err == nil {
		return duration
	}

	// 尝试解析Go的持续时间格式 (如: "1h30m")
	duration, err = time.ParseDuration(durationStr)
	if err == nil {
		return duration
	}

	// 最后尝试解析为秒
	if seconds, err := strconv.Atoi(durationStr); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// 返回默认值
	return time.Duration(defaultSeconds) * time.Second
}

// User represents a user in the system
type User struct {
	ID        string     `json:"id"`
	IsActive  bool       `json:"is_active"`
	Username  string     `json:"username"`
	Password  string     `json:"password"`
	Role      string     `json:"role"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	CreatedAt time.Time  `json:"created_at"`
	LastLogin time.Time  `json:"last_login"`
	API       []APIToken `json:"api"`
}

// UsersConfig holds the users configuration
type UsersConfig struct {
	Users       map[string]*User `json:"users"`
	DefaultRole string           `json:"default_role"`
	JWTSecret   string           `json:"jwt_secret"`
}

// GetUser 获取用户信息
func (c *UsersConfig) GetUser(username string) (*User, error) {
	if user, exists := c.Users[username]; exists {
		return user, nil
	}
	return nil, fmt.Errorf("用户不存在: %s", username)
}

// Save 保存用户配置到文件
func (c *UsersConfig) Save(baseDir string) error {
	// 在这里实现保存逻辑，例如将用户配置写入文件
	// 这里只是占位，实际实现会涉及到文件IO
	return nil
}

// VerifyPassword 验证用户密码
func (u *User) VerifyPassword(password string) bool {
	// 检测是否为bcrypt格式的密码 (以$2a$、$2b$或$2y$开头)
	if len(u.Password) > 4 && (u.Password[:4] == "$2a$" || u.Password[:4] == "$2b$" || u.Password[:4] == "$2y$") {
		// 使用golang.org/x/crypto/bcrypt包验证bcrypt格式密码
		err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
		return err == nil
	}

	// 默认使用SHA-256验证
	hash := sha256.Sum256([]byte(password))
	hashedPassword := hex.EncodeToString(hash[:])
	return hashedPassword == u.Password
}

// GenerateToken 生成用户JWT token
func (u *User) GenerateToken(secret string, expiration string) (string, error) {
	// 解析过期时间
	expirationDuration := GetDurationOrDefault(expiration, 86400) // 默认24小时

	claims := &JWTTokenClaims{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expirationDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   u.ID,
			// 明确添加token类型
			ID: "jwt",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// FormatDuration 将time.Duration格式化为ISO 8601持续时间格式
func FormatDuration(d time.Duration) string {
	var parts []string

	// 提取各个时间单位
	days := int(d.Hours()) / 24
	hours := int(d.Hours()) % 24
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60
	milliseconds := int(d.Milliseconds()) % 1000

	// 构建ISO 8601格式
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dD", days))
	}

	var timeParts []string
	if hours > 0 {
		timeParts = append(timeParts, fmt.Sprintf("%dH", hours))
	}
	if minutes > 0 {
		timeParts = append(timeParts, fmt.Sprintf("%dM", minutes))
	}
	if seconds > 0 || milliseconds > 0 {
		if milliseconds > 0 {
			timeParts = append(timeParts, fmt.Sprintf("%d.%03dS", seconds, milliseconds))
		} else {
			timeParts = append(timeParts, fmt.Sprintf("%dS", seconds))
		}
	}

	// 组合结果
	result := "P"
	if len(parts) > 0 {
		result += strings.Join(parts, "")
	}

	if len(timeParts) > 0 {
		result += "T" + strings.Join(timeParts, "")
	} else if len(parts) == 0 {
		// 如果没有天和时间部分，至少添加0秒
		result += "T0S"
	}

	return result
}

// CreateAPIToken 创建新的API token
func (u *User) CreateAPIToken(name string, permissions []string, expiration string, rateLimit int) (string, error) {
	// 创建新的API token
	tokenID := generateUUID()

	// 解析过期时间
	expirationDuration := GetDurationOrDefault(expiration, 3600) // 默认1小时
	expirationMinutes := int(expirationDuration.Minutes())
	if expirationMinutes <= 0 {
		expirationMinutes = 60 // 至少1小时
	}

	// 存储格式化后的ISO 8601格式
	formattedExpiration := expiration
	if expiration == "" || expiration == "0" {
		formattedExpiration = FormatDuration(expirationDuration)
	} else {
		// 尝试解析并重新格式化以确保标准格式
		parsedDuration, err := ParseISO8601Duration(expiration)
		if err == nil {
			formattedExpiration = FormatDuration(parsedDuration)
		}
	}

	apiToken := APIToken{
		ID:            tokenID,
		Name:          name,
		Permissions:   permissions,
		RateLimit:     rateLimit,
		Expiration:    formattedExpiration,
		CreatedAt:     time.Now(),
		LastUsed:      time.Now(),
		UsageCount:    0,
		IsActive:      true,
		LastResetTime: time.Now(),
	}

	// 生成JWT token
	tokenString, err := generateAPITokenJWT(u.ID, tokenID, expirationMinutes)
	if err != nil {
		return "", err
	}

	apiToken.Token = tokenString
	u.API = append(u.API, apiToken)
	return tokenString, nil
}

// GetAPITokens 获取用户所有API token
func (u *User) GetAPITokens() []APIToken {
	return u.API
}

// UpdateAPIToken 更新API token
func (u *User) UpdateAPIToken(tokenID string, name string, permissions []string, expiration string, rateLimit int) error {
	for i := range u.API {
		if u.API[i].ID == tokenID {
			u.API[i].Name = name
			u.API[i].Permissions = permissions
			u.API[i].Expiration = expiration
			u.API[i].RateLimit = rateLimit
			return nil
		}
	}
	return fmt.Errorf("API token不存在: %s", tokenID)
}

// DeleteAPIToken 删除API token
func (u *User) DeleteAPIToken(tokenID string) error {
	for i := range u.API {
		if u.API[i].ID == tokenID {
			u.API = append(u.API[:i], u.API[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("API token不存在: %s", tokenID)
}

// ResetAPITokenUsage 重置API token使用统计
func (u *User) ResetAPITokenUsage(tokenID string) error {
	for i := range u.API {
		if u.API[i].ID == tokenID {
			u.API[i].UsageCount = 0
			u.API[i].LastResetTime = time.Now()
			return nil
		}
	}
	return fmt.Errorf("API token不存在: %s", tokenID)
}

// 生成UUID
func generateUUID() string {
	// 简单实现，实际应使用uuid库
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// 生成API Token的JWT
func generateAPITokenJWT(userID, tokenID string, expiresInMinutes int) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"token_id": tokenID,
		"exp":      time.Now().Add(time.Duration(expiresInMinutes) * time.Minute).Unix(),
		"iat":      time.Now().Unix(),
		"type":     "api", // 明确添加token类型
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte("api_secret_key")) // 实际应使用配置的密钥
}

// APIToken 定义API token配置
type APIToken struct {
	ID            string    `json:"id"`              // API token的唯一标识
	Name          string    `json:"name"`            // API token的名称
	Description   string    `json:"description"`     // API token的描述
	Token         string    `json:"token"`           // API token的JWT字符串
	Permissions   []string  `json:"permissions"`     // API token的权限列表
	RateLimit     int       `json:"rate_limit"`      // 速率限制（每分钟请求数）
	Expiration    string    `json:"expiration"`      // 过期时间（ISO 8601持续时间格式）
	CreatedAt     time.Time `json:"created_at"`      // 创建时间
	LastUsed      time.Time `json:"last_used"`       // 最后使用时间
	UsageCount    int       `json:"usage_count"`     // 使用次数
	IsActive      bool      `json:"is_active"`       // 是否激活
	AllowedIPs    []string  `json:"allowed_ips"`     // 允许的IP地址列表
	AllowedHosts  []string  `json:"allowed_hosts"`   // 允许的主机名列表
	MaxRequests   int       `json:"max_requests"`    // 最大请求次数（0表示无限制）
	LastResetTime time.Time `json:"last_reset_time"` // 上次重置时间
}
