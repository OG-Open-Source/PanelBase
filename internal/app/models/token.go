package models

import (
	"fmt"
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
	ID            string   `json:"id"`              // API token的唯一标识
	Name          string   `json:"name"`            // API token的名称
	Description   string   `json:"description"`     // API token的描述
	Permissions   []string `json:"permissions"`     // API token的权限列表
	RateLimit     int      `json:"rate_limit"`      // 速率限制（每分钟请求数）
	Expiration    int      `json:"expiration"`      // 过期时间（分钟）
	CreatedAt     string   `json:"created_at"`      // 创建时间
	LastUsed      string   `json:"last_used"`       // 最后使用时间
	UsageCount    int      `json:"usage_count"`     // 使用次数
	IsActive      bool     `json:"is_active"`       // 是否激活
	AllowedIPs    []string `json:"allowed_ips"`     // 允许的IP地址列表
	AllowedHosts  []string `json:"allowed_hosts"`   // 允许的主机名列表
	MaxRequests   int      `json:"max_requests"`    // 最大请求次数（0表示无限制）
	LastResetTime string   `json:"last_reset_time"` // 上次重置时间
}

// GenerateJWTToken 生成JWT token
func GenerateJWTToken(user *User, secret string, expiration time.Duration) (string, error) {
	claims := &JWTClaims{
		UserID:   user.ID,
		Role:     user.Role,
		Name:     user.Name,
		Email:    user.Email,
		Password: user.Password,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateAPIToken 生成API token
func GenerateAPIToken(userID, apiID, secret string, expiration time.Duration) (string, error) {
	claims := &APIClaims{
		UserID: userID,
		APIID:  apiID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// ValidateJWTToken 验证JWT token
func ValidateJWTToken(tokenString, secret string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateAPIToken 验证API token
func ValidateAPIToken(tokenString, secret string) (*APIClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &APIClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*APIClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// ValidateToken 通用token验证方法
func ValidateToken(tokenString, secret string) (*TokenClaims, error) {
	// 直接尝试作为JWT token解析，但忽略过期错误
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithoutClaimsValidation())

	if err == nil {
		if claims, ok := token.Claims.(*JWTClaims); ok {
			// 只检查签名有效性，忽略过期
			if token.Valid {
				return &TokenClaims{
					UserID:           claims.UserID,
					Type:             TokenTypeJWT,
					RegisteredClaims: claims.RegisteredClaims,
				}, nil
			}
		}
	}

	// JWT解析失败，尝试API token
	apiClaims, apiErr := ValidateAPIToken(tokenString, secret)
	if apiErr == nil {
		// API token解析成功
		return &TokenClaims{
			UserID:           apiClaims.UserID,
			Type:             TokenTypeAPI,
			RegisteredClaims: apiClaims.RegisteredClaims,
		}, nil
	}

	// 如果两种都失败，返回JWT错误
	return nil, fmt.Errorf("invalid token: %v", err)
}

// IsAPITokenValid 检查API token是否有效
func IsAPITokenValid(token *APIToken) bool {
	if !token.IsActive {
		return false
	}

	// 检查是否超过最大请求次数
	if token.MaxRequests > 0 && token.UsageCount >= token.MaxRequests {
		return false
	}

	// 检查是否过期
	lastReset, err := time.Parse(time.RFC3339, token.LastResetTime)
	if err != nil {
		return false
	}

	expiration := time.Duration(token.Expiration) * time.Minute
	if time.Since(lastReset) > expiration {
		return false
	}

	return true
}

// ResetAPITokenUsage 重置API token使用统计
func ResetAPITokenUsage(token *APIToken) {
	token.UsageCount = 0
	token.LastResetTime = time.Now().Format(time.RFC3339)
}

// IncrementAPITokenUsage 增加API token使用次数
func IncrementAPITokenUsage(token *APIToken) {
	token.UsageCount++
	token.LastUsed = time.Now().Format(time.RFC3339)
}
