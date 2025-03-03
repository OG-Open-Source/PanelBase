package middleware

import (
	"net/http"
	"strings"
	"github.com/golang-jwt/jwt"
	"github.com/OG-Open-Source/PanelBase/internal/config"
)

type AuthMiddleware struct {
	config *config.Config
}

func NewAuthMiddleware(cfg *config.Config) *AuthMiddleware {
	return &AuthMiddleware{
		config: cfg,
	}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 檢查是否為受信任的 IP
		clientIP := getClientIP(r)
		if !m.isIPTrusted(clientIP) {
			http.Error(w, "Unauthorized IP", http.StatusUnauthorized)
			return
		}

		// 檢查 JWT Token
		token := r.Header.Get("Authorization")
		if !strings.HasPrefix(token, "Bearer ") {
			http.Error(w, "Invalid token format", http.StatusUnauthorized)
			return
		}

		tokenString := strings.TrimPrefix(token, "Bearer ")
		if !m.validateToken(tokenString) {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (m *AuthMiddleware) isIPTrusted(ip string) bool {
	for _, trustedIP := range m.config.TrustedIPs {
		if ip == trustedIP {
			return true
		}
	}
	return false
}

func (m *AuthMiddleware) validateToken(tokenString string) bool {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(m.config.JWTSecret), nil
	})

	return err == nil && token.Valid
}

func getClientIP(r *http.Request) string {
	// 檢查 X-Forwarded-For 頭
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		return strings.TrimSpace(ips[0])
	}
	// 如果沒有 X-Forwarded-For，使用 RemoteAddr
	return strings.Split(r.RemoteAddr, ":")[0]
} 