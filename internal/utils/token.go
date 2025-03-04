package utils

import (
	"time"
	"github.com/golang-jwt/jwt"
	"github.com/OG-Open-Source/PanelBase/internal/config"
)

func GenerateToken(cfg *config.Config) (string, error) {
	// 創建一個新的令牌對象，指定簽名方法和聲明
	token := jwt.New(jwt.SigningMethodHS256)

	// 設置聲明
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix() // 24小時後過期
	claims["iat"] = time.Now().Unix()

	// 使用密鑰簽名並獲得完整的編碼令牌作為字符串
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
} 