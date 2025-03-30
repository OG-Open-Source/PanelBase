package models

import (
	"github.com/golang-jwt/jwt/v5"
)

// JWTTokenClaims 定义JWT token的claims
type JWTTokenClaims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.RegisteredClaims
}
