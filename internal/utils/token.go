package utils

import (
	"time"
	"github.com/golang-jwt/jwt"
	"github.com/OG-Open-Source/PanelBase/internal/config"
)

func GenerateToken(cfg *config.Config) (string, error) {
	token := jwt.New(jwt.SigningMethodHS256)
	claims := token.Claims.(jwt.MapClaims)
	claims["exp"] = time.Now().Add(time.Hour * 24).Unix()
	claims["iat"] = time.Now().Unix()
	tokenString, err := token.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", err
	}
	return tokenString, nil
}