package models

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// JsonTime 自定义时间类型，用于处理JSON序列化时的零值时间
type JsonTime time.Time

// MarshalJSON 自定义时间序列化方法，处理零值时间
func (t JsonTime) MarshalJSON() ([]byte, error) {
	tt := time.Time(t)
	if tt.IsZero() || tt.Year() == 1 {
		return []byte("null"), nil
	}
	return json.Marshal(tt)
}

// UnmarshalJSON 自定义时间反序列化方法
func (t *JsonTime) UnmarshalJSON(data []byte) error {
	// 处理null值
	if string(data) == "null" {
		*t = JsonTime(time.Time{})
		return nil
	}

	tt := time.Time{}
	err := json.Unmarshal(data, &tt)
	*t = JsonTime(tt)
	return err
}

// Time 将JsonTime转换为time.Time
func (t JsonTime) Time() time.Time {
	return time.Time(t)
}

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

// GetDurationOrDefault 从字符串解析持续时间，如果解析失败则返回默认值
// 仅支持ISO 8601格式，如: PT1H, P1D, P1W, P1Y2M3DT4H5M6S等
func GetDurationOrDefault(durationStr string, defaultSeconds int) time.Duration {
	defaultDuration := time.Duration(defaultSeconds) * time.Second

	if durationStr == "" {
		return defaultDuration
	}

	// 只尝试解析ISO 8601格式
	duration, err := ParseISO8601Duration(durationStr)
	if err == nil {
		return duration
	}

	// 解析失败，返回默认值
	return defaultDuration
}

// User represents a user in the system
type User struct {
	// ID 欄位被移除，因為使用 map 中的 key 作為用戶ID
	IsActive  bool                `json:"is_active"`
	Username  string              `json:"username"`
	Password  string              `json:"password"`
	Role      string              `json:"role"`
	Name      string              `json:"name"`
	Email     string              `json:"email"`
	CreatedAt JsonTime            `json:"created_at"`
	LastLogin JsonTime            `json:"last_login"`
	API       map[string]APIToken `json:"api"`
}

// UsersConfig holds the users configuration
type UsersConfig struct {
	Users       map[string]*User `json:"users"`
	DefaultRole string           `json:"default_role"`
	JWTSecret   string           `json:"jwt_secret"`
}

// GetUser 获取用户信息
func (c *UsersConfig) GetUser(username string) (*User, error) {
	// 遍历所有用户，查找匹配的用户名
	for _, user := range c.Users {
		if user.Username == username {
			return user, nil
		}
	}
	return nil, fmt.Errorf("user not found: %s", username)
}

// Save 保存用户配置到文件
func (c *UsersConfig) Save(baseDir string) error {
	// Import internally to avoid import cycles
	saveFn := func(config *UsersConfig, dir string) error {
		configPath := filepath.Join(dir, "configs", "users.json")

		// Serialize the configuration
		data, err := json.MarshalIndent(config, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize users configuration: %w", err)
		}

		// Ensure the directory exists
		dirPath := filepath.Dir(configPath)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return fmt.Errorf("failed to create config directory: %w", err)
		}

		// Write to file
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return fmt.Errorf("failed to save users configuration: %w", err)
		}

		return nil
	}

	return saveFn(c, baseDir)
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

// GenerateToken 生成用戶JWT token
func (u *User) GenerateToken(secret string, expiration string, userID string) (string, error) {
	// 默认持续时间（如果未提供或解析失败）
	defaultExpirationDuration := 24 * time.Hour // 默认24小时

	// 解析過期時間
	var expirationDuration time.Duration
	if expiration == "" {
		// 使用默認過期時間
		expirationDuration = defaultExpirationDuration
	} else {
		// 嘗試解析為ISO 8601格式
		parsedDuration, err := ParseISO8601Duration(expiration)
		if err == nil {
			// 使用解析结果（无最小/最大限制）
			expirationDuration = parsedDuration
		} else {
			// 解析失败，使用默认值
			expirationDuration = defaultExpirationDuration
		}
	}

	// 获取当前时间
	now := time.Now()

	// 计算过期时间
	expiresAt := now.Add(expirationDuration)

	// 使用MapClaims以便添加自定义字段
	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": u.Username,
		"role":     u.Role,
		"type":     "jwt", // 明确添加token类型
		"exp":      expiresAt.Unix(),
		"iat":      now.Unix(),
		"sub":      userID,
		"jti":      generateUUID(),   // 使用UUID作为JWT ID
		"iss":      "panelbase-auth", // 发行者标识
		"aud":      "panelbase-api",  // 接收者标识
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	// 返回令牌
	return tokenString, nil
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
func (u *User) CreateAPIToken(name string, permissions []string, expiration string, rateLimit int, secret string) (string, error) {
	// 创建新的API token
	tokenID := generateUUID()

	// 默认持续时间（如果未提供或解析失败）
	defaultExpirationDuration := time.Hour // 默认1小时

	// 解析过期时间
	var expirationDuration time.Duration
	if expiration == "" || expiration == "0" {
		// 使用默认过期时间
		expirationDuration = defaultExpirationDuration
	} else {
		// 尝试解析为ISO 8601格式
		parsedDuration, err := ParseISO8601Duration(expiration)
		if err == nil {
			// 使用解析结果（无最小/最大限制）
			expirationDuration = parsedDuration
		} else {
			// 解析失败，使用默认值
			expirationDuration = defaultExpirationDuration
		}
	}

	// 存储格式化后的ISO 8601格式
	formattedExpiration := FormatDuration(expirationDuration)

	apiToken := APIToken{
		Name:          name,
		Permissions:   permissions,
		RateLimit:     rateLimit,
		Expiration:    formattedExpiration,
		CreatedAt:     JsonTime(time.Now()),
		LastUsed:      JsonTime(time.Time{}), // 使用零值表示从未使用
		UsageCount:    0,
		IsActive:      true,
		LastResetTime: JsonTime(time.Time{}), // 使用零值表示从未重置
	}

	// 获取当前时间
	now := time.Now()

	// 计算过期时间（以分钟为单位）
	expirationMinutes := int(expirationDuration.Minutes())

	// 计算过期时间
	_ = now.Add(expirationDuration)

	// 生成JWT token
	tokenString, err := generateAPITokenJWT(u.Username, tokenID, expirationMinutes, secret)
	if err != nil {
		return "", err
	}

	apiToken.Token = tokenString

	// 初始化API map（如果为nil）
	if u.API == nil {
		u.API = make(map[string]APIToken)
	}

	u.API[tokenID] = apiToken
	return tokenString, nil
}

// CreateAPITokenWithSecret 创建新的API token，使用指定的JWT密钥
func (u *User) CreateAPITokenWithSecret(name string, permissions []string, expiration string, rateLimit int, secret string) (string, error) {
	// 创建新的API token
	tokenID := generateUUID()

	// 默认持续时间（如果未提供或解析失败）
	defaultExpirationDuration := time.Hour // 默认1小时

	// 解析过期时间
	var expirationDuration time.Duration
	if expiration == "" || expiration == "0" {
		// 使用默认过期时间
		expirationDuration = defaultExpirationDuration
	} else {
		// 尝试解析为ISO 8601格式
		parsedDuration, err := ParseISO8601Duration(expiration)
		if err == nil {
			// 使用解析结果（无最小/最大限制）
			expirationDuration = parsedDuration
		} else {
			// 解析失败，使用默认值
			expirationDuration = defaultExpirationDuration
		}
	}

	// 存储格式化后的ISO 8601格式
	formattedExpiration := FormatDuration(expirationDuration)

	apiToken := APIToken{
		Name:          name,
		Permissions:   permissions,
		RateLimit:     rateLimit,
		Expiration:    formattedExpiration,
		CreatedAt:     JsonTime(time.Now()),
		LastUsed:      JsonTime(time.Time{}), // 使用零值表示从未使用
		UsageCount:    0,
		IsActive:      true,
		LastResetTime: JsonTime(time.Time{}), // 使用零值表示从未重置
	}

	// 使用传入的secret生成JWT token
	// 获取当前时间
	now := time.Now()

	// 计算过期时间
	expiresAt := now.Add(expirationDuration)

	claims := jwt.MapClaims{
		"user_id":  u.Username,
		"token_id": tokenID,
		"exp":      expiresAt.Unix(),
		"iat":      now.Unix(),
		"type":     "api",            // 明确添加token类型
		"jti":      tokenID,          // 使用tokenID作为JWT ID
		"iss":      "panelbase-auth", // 发行者标识
		"aud":      "panelbase-api",  // 接收者标识
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	apiToken.Token = tokenString

	// 初始化API map（如果为nil）
	if u.API == nil {
		u.API = make(map[string]APIToken)
	}

	u.API[tokenID] = apiToken
	return tokenString, nil
}

// GetAPITokens 获取用户所有API token
func (u *User) GetAPITokens() []APIToken {
	var tokens []APIToken
	for _, token := range u.API {
		tokens = append(tokens, token)
	}
	return tokens
}

// UpdateAPIToken 更新API token
func (u *User) UpdateAPIToken(tokenID string, name string, permissions []string, expiration string, rateLimit int) error {
	token, exists := u.API[tokenID]
	if !exists {
		return fmt.Errorf("API token不存在: %s", tokenID)
	}
	token.Name = name
	token.Permissions = permissions
	token.Expiration = expiration
	token.RateLimit = rateLimit
	u.API[tokenID] = token
	return nil
}

// DeleteAPIToken 删除API token
func (u *User) DeleteAPIToken(tokenID string) error {
	_, exists := u.API[tokenID]
	if !exists {
		return fmt.Errorf("API token不存在: %s", tokenID)
	}
	delete(u.API, tokenID)
	return nil
}

// ResetAPITokenUsage 重置API token使用统计
func (u *User) ResetAPITokenUsage(tokenID string) error {
	token, exists := u.API[tokenID]
	if !exists {
		return fmt.Errorf("API token不存在: %s", tokenID)
	}
	token.UsageCount = 0
	token.LastResetTime = JsonTime(time.Now())
	u.API[tokenID] = token
	return nil
}

// UpdateAPITokenUsage 更新API token使用情况
func (u *User) UpdateAPITokenUsage(tokenID string) error {
	token, exists := u.API[tokenID]
	if !exists {
		return fmt.Errorf("API token不存在: %s", tokenID)
	}

	// 增加使用次数
	token.UsageCount++

	// 更新最后使用时间
	token.LastUsed = JsonTime(time.Now())

	// 更新token
	u.API[tokenID] = token
	return nil
}

// 生成UUID
func generateUUID() string {
	// 简单实现，实际应使用uuid库
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// 生成API Token的JWT
func generateAPITokenJWT(userID, tokenID string, expiresInMinutes int, secret string) (string, error) {
	// 获取当前时间
	now := time.Now()

	// 计算过期时间
	expiresAt := now.Add(time.Duration(expiresInMinutes) * time.Minute)

	claims := jwt.MapClaims{
		"user_id":  userID,
		"token_id": tokenID,
		"exp":      expiresAt.Unix(),
		"iat":      now.Unix(),
		"type":     "api",            // 明确添加token类型
		"jti":      tokenID,          // 使用tokenID作为JWT ID
		"iss":      "panelbase-auth", // 发行者标识
		"aud":      "panelbase-api",  // 接收者标识
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("生成JWT令牌失败: %w", err)
	}

	return tokenString, nil
}

// APIToken 定義API token配置
type APIToken struct {
	Name          string   `json:"name"`            // API token的名称
	Description   string   `json:"description"`     // API token的描述
	Token         string   `json:"token"`           // API token的JWT字符串
	Permissions   []string `json:"permissions"`     // API token的权限列表
	RateLimit     int      `json:"rate_limit"`      // 速率限制（每分钟请求数）
	Expiration    string   `json:"expiration"`      // 过期时间（ISO 8601持续时间格式）
	CreatedAt     JsonTime `json:"created_at"`      // 创建时间
	LastUsed      JsonTime `json:"last_used"`       // 最后使用时间
	UsageCount    int      `json:"usage_count"`     // 使用次数
	IsActive      bool     `json:"is_active"`       // 是否激活
	AllowedIPs    []string `json:"allowed_ips"`     // 允许的IP地址列表
	AllowedHosts  []string `json:"allowed_hosts"`   // 允许的主机名列表
	MaxRequests   int      `json:"max_requests"`    // 最大请求次数（0表示无限制）
	LastResetTime JsonTime `json:"last_reset_time"` // 上次重置时间
}
