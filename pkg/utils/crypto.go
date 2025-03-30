package utils

import (
	"crypto/sha256"
	"encoding/hex"

	"golang.org/x/crypto/bcrypt"
)

// CryptoUtils 加密工具結構體
type CryptoUtils struct{}

// NewCryptoUtils 創建一個新的加密工具實例
func NewCryptoUtils() *CryptoUtils {
	return &CryptoUtils{}
}

// HashPassword 使用 bcrypt 對密碼進行加密
// 如果 bcrypt 加密失敗，回退到 SHA-256
func (c *CryptoUtils) HashPassword(password string) string {
	// 使用 bcrypt 生成更安全的密碼哈希
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// 如果 bcrypt 失敗，回退到 SHA-256
		hash := sha256.Sum256([]byte(password))
		return hex.EncodeToString(hash[:])
	}
	return string(hashedBytes)
}

// VerifyPassword 驗證密碼是否匹配
// 自動檢測密碼哈希類型（bcrypt 或 SHA-256）
func (c *CryptoUtils) VerifyPassword(password, hashedPassword string) bool {
	// 檢測是否為 bcrypt 格式的密碼（以 $2a$、$2b$ 或 $2y$ 開頭）
	if len(hashedPassword) > 4 && (hashedPassword[:4] == "$2a$" || hashedPassword[:4] == "$2b$" || hashedPassword[:4] == "$2y$") {
		// 使用 bcrypt 驗證
		err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
		return err == nil
	}

	// 默認使用 SHA-256 驗證
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:]) == hashedPassword
}

// SimpleHash 簡單的 SHA-256 哈希，用於非敏感數據
func (c *CryptoUtils) SimpleHash(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// HashWithSalt 使用自定義鹽值進行哈希
func (c *CryptoUtils) HashWithSalt(data, salt string) string {
	hash := sha256.Sum256([]byte(data + salt))
	return hex.EncodeToString(hash[:])
}

// GenerateHash 生成指定長度的哈希字符串
func (c *CryptoUtils) GenerateHash(data string, length int) string {
	if length <= 0 {
		length = 32 // 默認長度
	}

	hash := sha256.Sum256([]byte(data))
	hexHash := hex.EncodeToString(hash[:])

	// 如果請求的長度大於 SHA-256 哈希的長度，則重複哈希
	if length > len(hexHash) {
		return hexHash
	}

	return hexHash[:length]
}
