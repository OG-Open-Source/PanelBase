package middleware

import (
	"net/http"
	"strings"

	appmodels "github.com/OG-Open-Source/PanelBase/internal/app/models"
	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	sharedmodels "github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/gin-gonic/gin"
)

// AuthMiddleware 认证中间件
func AuthMiddleware(configService *services.ConfigService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 将ConfigService加入上下文
		c.Set("configService", configService)

		// 获取JWT密钥
		secret := configService.UsersConfig.JWTSecret

		// 获取Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未提供认证信息"})
			c.Abort()
			return
		}

		// 解析Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的认证格式"})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 验证token
		claims, err := appmodels.ValidateToken(tokenString, secret)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的token", "details": err.Error()})
			c.Abort()
			return
		}

		// 根据token类型处理
		switch claims.Type {
		case appmodels.TokenTypeJWT:
			// 验证用户信息是否发生变化
			user, err := configService.UsersConfig.GetUser(claims.UserID)
			if err != nil || user == nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}

			// 验证用户信息是否与token中的一致
			jwtClaims, err := appmodels.ValidateJWTToken(tokenString, secret)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的JWT token"})
				c.Abort()
				return
			}

			if jwtClaims.Role != user.Role ||
				jwtClaims.Name != user.Name ||
				jwtClaims.Email != user.Email ||
				jwtClaims.Password != user.Password {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户信息已更改，请重新登录"})
				c.Abort()
				return
			}

			// 设置用户信息到上下文
			c.Set("user_id", claims.UserID)
			c.Set("role", jwtClaims.Role)
			c.Set("token_type", "jwt")
			c.Set("user", user)

		case appmodels.TokenTypeAPI:
			// 验证API token
			user, err := configService.UsersConfig.GetUser(claims.UserID)
			if err != nil || user == nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
				c.Abort()
				return
			}

			// 获取API token信息
			apiClaims, err := appmodels.ValidateAPIToken(tokenString, secret)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的API token"})
				c.Abort()
				return
			}

			// 验证API token是否存在且有效
			var apiToken *sharedmodels.APIToken
			for i := range user.API {
				if user.API[i].ID == apiClaims.APIID {
					apiToken = &user.API[i]
					break
				}
			}

			if apiToken == nil {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "API token已失效"})
				c.Abort()
				return
			}

			// 设置用户信息和API token信息到上下文
			c.Set("user_id", claims.UserID)
			c.Set("api_id", apiClaims.APIID)
			c.Set("api_permissions", apiToken.Permissions)
			c.Set("token_type", "api")
			c.Set("user", user)

		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未知的token类型"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RoleMiddleware 角色验证中间件
func RoleMiddleware(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取token类型
		tokenType, exists := c.Get("token_type")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
			c.Abort()
			return
		}

		// 根据token类型验证权限
		switch tokenType {
		case "jwt":
			userRole, exists := c.Get("role")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
				c.Abort()
				return
			}

			if userRole != role {
				c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
				c.Abort()
				return
			}

		case "api":
			permissions, exists := c.Get("api_permissions")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
				c.Abort()
				return
			}

			// 检查API token是否具有所需权限
			hasPermission := false
			for _, p := range permissions.([]string) {
				if p == role {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
				c.Abort()
				return
			}

		default:
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未知的token类型"})
			c.Abort()
			return
		}

		c.Next()
	}
}
