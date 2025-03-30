package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"strings"
	"time"
)

// 定義不同類型的隨機生成器常量
const (
	TypeUserID    = "user"
	TypeJWTSecret = "jwt"
	TypeAPIToken  = "api"
	TypePassword  = "pwd"
)

// 字符集常量
const (
	CharsetAlphaNumeric = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	CharsetAlphaLower   = "abcdefghijklmnopqrstuvwxyz"
	CharsetAlphaUpper   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	CharsetNumeric      = "0123456789"
	CharsetSpecial      = "!@#$%^&*()-_=+[]{}|;:,.<>?"
)

// RandomGenerator 隨機值生成器
type RandomGenerator struct{}

// NewRandomGenerator 創建一個新的隨機生成器
func NewRandomGenerator() *RandomGenerator {
	return &RandomGenerator{}
}

// GenerateRandomString 生成指定長度的隨機字符串
func (g *RandomGenerator) GenerateRandomString(length int, charset string) string {
	if charset == "" {
		charset = CharsetAlphaNumeric
	}

	b := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))

	for i := range b {
		n, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			// 如果加密隨機數生成失敗，使用當前時間作為備選
			return fmt.Sprintf("%x", time.Now().UnixNano())
		}
		b[i] = charset[n.Int64()]
	}

	return string(b)
}

// GenerateUserID 生成用戶ID
func (g *RandomGenerator) GenerateUserID() string {
	prefix := "usr_"
	randomPart := g.GenerateRandomString(16, CharsetAlphaNumeric)
	return prefix + randomPart
}

// GenerateJWTSecret 生成JWT密鑰
func (g *RandomGenerator) GenerateJWTSecret() string {
	return g.GenerateRandomString(32, CharsetAlphaNumeric)
}

// GenerateAPIToken 生成API令牌
func (g *RandomGenerator) GenerateAPIToken() string {
	prefix := "tok_"
	randomPart := g.GenerateRandomString(24, CharsetAlphaNumeric)
	return prefix + randomPart
}

// GenerateUUID 生成UUID格式的字符串
func (g *RandomGenerator) GenerateUUID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// GenerateRandomPort 生成指定範圍內的隨機端口號
func (g *RandomGenerator) GenerateRandomPort(min, max int) (int, error) {
	// 確保最小值不小於註冊端口範圍下限
	if min < 1024 {
		min = 1024
	}

	// 確保最大值不超過註冊端口範圍上限
	if max > 49151 {
		max = 49151
	}

	// 檢查範圍有效性
	if min >= max {
		return 0, fmt.Errorf("無效的端口範圍: %d-%d", min, max)
	}

	// 生成隨機端口
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	if err != nil {
		return 0, err
	}

	port := int(n.Int64()) + min
	return port, nil
}

// GenerateSecurePassword 生成安全密碼
func (g *RandomGenerator) GenerateSecurePassword(length int) string {
	if length < 8 {
		length = 8 // 最低8位密碼
	}

	// 混合多種字符集確保密碼安全性
	password := g.GenerateRandomString(length/4, CharsetAlphaLower) +
		g.GenerateRandomString(length/4, CharsetAlphaUpper) +
		g.GenerateRandomString(length/4, CharsetNumeric) +
		g.GenerateRandomString(length-3*(length/4), CharsetSpecial)

	// 將密碼字符打亂順序
	passwordRunes := []rune(password)
	for i := range passwordRunes {
		j, _ := rand.Int(rand.Reader, big.NewInt(int64(len(passwordRunes))))
		passwordRunes[i], passwordRunes[j.Int64()] = passwordRunes[j.Int64()], passwordRunes[i]
	}

	return string(passwordRunes)
}

// IsValidRegisteredPort 檢查端口是否在註冊端口範圍內 (1024-49151)
func (g *RandomGenerator) IsValidRegisteredPort(port int) bool {
	return port >= 1024 && port <= 49151
}

// IsPortAvailable 檢查指定端口是否可用
func (g *RandomGenerator) IsPortAvailable(port int) bool {
	// 首先檢查端口範圍是否有效
	if !g.IsValidRegisteredPort(port) {
		return false
	}

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	listener.Close()
	return true
}

// FindAvailablePort 查找可用端口
func (g *RandomGenerator) FindAvailablePort(min, max int, attempts int) (int, error) {
	for i := 0; i < attempts; i++ {
		port, err := g.GenerateRandomPort(min, max)
		if err != nil {
			continue
		}

		if g.IsPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("無法在 %d 次嘗試內找到可用端口", attempts)
}

// GenerateRandomBytes 生成指定長度的隨機字節
func (g *RandomGenerator) GenerateRandomBytes(length int) ([]byte, error) {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// GenerateHexString 生成指定長度的十六進制隨機字符串
func (g *RandomGenerator) GenerateHexString(length int) string {
	bytes, err := g.GenerateRandomBytes(length / 2)
	if err != nil {
		return g.GenerateRandomString(length, CharsetAlphaNumeric)
	}
	return hex.EncodeToString(bytes)
}

// GenerateFriendlyID 生成友好可讀的 ID（如 blue-dog-123）
func (g *RandomGenerator) GenerateFriendlyID() string {
	adjectives := []string{"red", "blue", "green", "happy", "quick", "silent", "bold", "calm", "brave", "bright"}
	nouns := []string{"cat", "dog", "bird", "fish", "tiger", "lion", "eagle", "fox", "wolf", "bear"}

	adj, _ := rand.Int(rand.Reader, big.NewInt(int64(len(adjectives))))
	noun, _ := rand.Int(rand.Reader, big.NewInt(int64(len(nouns))))
	num, _ := rand.Int(rand.Reader, big.NewInt(1000))

	return fmt.Sprintf("%s-%s-%d",
		adjectives[adj.Int64()],
		nouns[noun.Int64()],
		num.Int64())
}

// SanitizeIDString 清理字符串使其成為有效的 ID
func SanitizeIDString(input string) string {
	// 移除特殊字符，只保留字母、數字和下劃線
	sanitized := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' || r == '-' {
			return r
		}
		return '_'
	}, input)

	// 確保 ID 不以數字開頭
	if len(sanitized) > 0 && sanitized[0] >= '0' && sanitized[0] <= '9' {
		sanitized = "id_" + sanitized
	}

	// 如果結果為空，返回默認 ID
	if sanitized == "" {
		return "default_id"
	}

	return sanitized
}
