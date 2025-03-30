package types

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// TokenType 定义token类型
type TokenType string

const (
	TokenTypeJWT TokenType = "jwt"
	TokenTypeAPI TokenType = "api"
)

// TokenClaims 定义token的通用claims
type TokenClaims struct {
	UserID string    `json:"user_id"`
	Type   TokenType `json:"type"`
	jwt.RegisteredClaims
}

// JWTClaims 定义JWT token的claims
type JWTClaims struct {
	UserID   string `json:"user_id"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
	jwt.RegisteredClaims
}

// APIClaims 定义API token的claims
type APIClaims struct {
	UserID string `json:"user_id"`
	APIID  string `json:"api_id"`
	jwt.RegisteredClaims
}

// APIToken 定义API token配置
type APIToken struct {
	ID            string    `json:"id"`              // API token的唯一标识
	Name          string    `json:"name"`            // API token的名称
	Description   string    `json:"description"`     // API token的描述
	Token         string    `json:"token"`           // API token的JWT字符串
	Permissions   []string  `json:"permissions"`     // API token的权限列表
	RateLimit     int       `json:"rate_limit"`      // 速率限制（每分钟请求数）
	Expiration    int       `json:"expiration"`      // 过期时间（分钟）
	CreatedAt     time.Time `json:"created_at"`      // 创建时间
	LastUsed      time.Time `json:"last_used"`       // 最后使用时间
	UsageCount    int       `json:"usage_count"`     // 使用次数
	IsActive      bool      `json:"is_active"`       // 是否激活
	AllowedIPs    []string  `json:"allowed_ips"`     // 允许的IP地址列表
	AllowedHosts  []string  `json:"allowed_hosts"`   // 允许的主机名列表
	MaxRequests   int       `json:"max_requests"`    // 最大请求次数（0表示无限制）
	LastResetTime time.Time `json:"last_reset_time"` // 上次重置时间
}
