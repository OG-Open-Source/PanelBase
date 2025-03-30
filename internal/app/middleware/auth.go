package middleware

import (
	"fmt"
	"net/http"
	"strings"

	appmodels "github.com/OG-Open-Source/PanelBase/internal/app/models"
	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	sharedmodels "github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
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
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failure",
				"message": "Authentication information not provided",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 解析Bearer token
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failure",
				"message": "Invalid authentication format",
				"data":    nil,
			})
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 特殊处理：如果请求的是API token创建接口，允许过期但签名有效的JWT
		isAPITokenCreation := c.Request.URL.Path == "/api/v1/auth/token" && c.Request.Method == "POST"

		var claims *appmodels.TokenClaims
		var err error

		if isAPITokenCreation {
			// 对API token创建请求使用忽略过期的验证
			token, parseErr := jwt.ParseWithClaims(tokenString, &appmodels.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
				}
				return []byte(secret), nil
			}, jwt.WithoutClaimsValidation())

			if parseErr == nil && token.Valid {
				if jwtClaims, ok := token.Claims.(*appmodels.JWTClaims); ok {
					claims = &appmodels.TokenClaims{
						UserID:           jwtClaims.UserID,
						Type:             appmodels.TokenTypeJWT,
						RegisteredClaims: jwtClaims.RegisteredClaims,
					}
				}
			} else {
				err = parseErr
			}
		} else {
			// 普通API请求使用标准验证
			claims, err = appmodels.ValidateToken(tokenString, secret)
		}

		if err != nil || claims == nil {
			errMsg := "Invalid token"
			if err != nil {
				errMsg = err.Error()
			}
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failure",
				"message": "Invalid token",
				"data": gin.H{
					"details": errMsg,
				},
			})
			c.Abort()
			return
		}

		// 根据token类型处理
		switch claims.Type {
		case appmodels.TokenTypeJWT:
			// 查找用户信息
			var foundUser *sharedmodels.User
			var userID string

			// 先尝试使用 UserID 查找用户
			for id, user := range configService.UsersConfig.Users {
				if id == claims.UserID {
					foundUser = user
					userID = id
					break
				}
			}

			// 如果找不到用户，返回错误
			if foundUser == nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"status":  "failure",
					"message": "User does not exist",
					"data": gin.H{
						"user_id": claims.UserID,
					},
				})
				c.Abort()
				return
			}

			// 设置用户信息到上下文
			c.Set("user_id", userID)
			c.Set("role", foundUser.Role)
			c.Set("token_type", "jwt")
			c.Set("user", foundUser)

		case appmodels.TokenTypeAPI:
			// 处理API token验证
			// 从claims中获取用户ID
			username := claims.UserID

			// 查找用户
			var foundUser *sharedmodels.User
			var userID string

			// 遍历所有用户，查找匹配的用户名和API token
			for id, user := range configService.UsersConfig.Users {
				if user.Username == username {
					foundUser = user
					userID = id

					// 检查是否有匹配的token
					// 使用token字符串尝试匹配user中的API tokens
					for tokenID, apiToken := range user.API {
						if apiToken.Token == tokenString {
							// 更新API token使用统计
							if err := foundUser.UpdateAPITokenUsage(tokenID); err != nil {
								// 记录错误但不中断请求处理
								fmt.Printf("Failed to update API token usage: %v\n", err)
							}

							// 保存用户配置
							err := configService.UsersConfig.Save(configService.BaseDir)
							if err != nil {
								// 记录错误但不中断请求处理
								fmt.Printf("Failed to save user configuration after token usage update: %v\n", err)
							}
							break
						}
					}
					break
				}
			}

			// 如果找不到用户，返回错误
			if foundUser == nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"status":  "failure",
					"message": "User does not exist",
					"data": gin.H{
						"username": username,
					},
				})
				c.Abort()
				return
			}

			// 设置用户信息到上下文
			c.Set("user_id", userID)
			c.Set("role", foundUser.Role)
			c.Set("token_type", "api")
			c.Set("user", foundUser)

		default:
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failure",
				"message": "Unknown token type",
				"data":    nil,
			})
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
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failure",
				"message": "Not authenticated",
				"data":    nil,
			})
			c.Abort()
			return
		}

		// 根据token类型验证权限
		switch tokenType {
		case "jwt":
			userRole, exists := c.Get("role")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{
					"status":  "failure",
					"message": "Not authenticated",
					"data":    nil,
				})
				c.Abort()
				return
			}

			if userRole != role {
				c.JSON(http.StatusForbidden, gin.H{
					"status":  "failure",
					"message": "Insufficient permissions",
					"data":    nil,
				})
				c.Abort()
				return
			}

		case "api":
			permissions, exists := c.Get("api_permissions")
			if !exists {
				c.JSON(http.StatusUnauthorized, gin.H{
					"status":  "failure",
					"message": "Not authenticated",
					"data":    nil,
				})
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
				c.JSON(http.StatusForbidden, gin.H{
					"status":  "failure",
					"message": "Insufficient permissions",
					"data":    nil,
				})
				c.Abort()
				return
			}

		default:
			c.JSON(http.StatusUnauthorized, gin.H{
				"status":  "failure",
				"message": "Unknown token type",
				"data":    nil,
			})
			c.Abort()
			return
		}

		c.Next()
	}
}
