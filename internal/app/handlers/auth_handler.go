package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	sharedmodels "github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	configService *services.ConfigService
}

// NewAuthHandler 创建新的认证处理器
func NewAuthHandler(configService *services.ConfigService) *AuthHandler {
	return &AuthHandler{
		configService: configService,
	}
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Duration string `json:"duration"` // ISO 8601持续时间格式，可选
}

// LoginHandler 处理登录请求
func (h *AuthHandler) LoginHandler(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failure",
			"message": "Invalid request data",
			"data": gin.H{
				"user":     req.Username,
				"password": "****",
			},
		})
		return
	}

	// 验证用户名和密码
	if h.configService.UsersConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failure",
			"message": "Configuration loading failed",
			"data":    nil,
		})
		return
	}

	// 获取用户
	user, err := h.configService.UsersConfig.GetUser(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failure",
			"message": "User not found",
			"data": gin.H{
				"user":     req.Username,
				"password": "****",
			},
		})
		return
	}

	// 验证密码
	if !user.VerifyPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failure",
			"message": "Invalid password",
			"data": gin.H{
				"user":     req.Username,
				"password": "****",
			},
		})
		return
	}

	// 查找用戶ID
	userID := utils.FindUserIDByUsername(h.configService.UsersConfig.Users, user.Username)

	// 使用配置文件中的JWT密钥
	jwtSecret := h.configService.UsersConfig.JWTSecret

	// 获取过期时间
	expiration := req.Duration
	if expiration == "" {
		// 使用配置文件中的默认过期时间
		if h.configService.Config != nil && h.configService.Config.Auth.JWTExpiration > 0 {
			expiration = fmt.Sprintf("%d", h.configService.Config.Auth.JWTExpiration)
		} else {
			expiration = "24" // 默认24小时
		}
	}

	// 生成JWT token
	token, err := user.GenerateToken(jwtSecret, expiration, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failure",
			"message": "Failed to generate token",
			"data":    nil,
		})
		return
	}

	// 更新最后登录时间
	user.LastLogin = sharedmodels.JsonTime(time.Now())

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		// 不返回错误，继续登录流程
		fmt.Printf("Failed to save user configuration: %v\n", err)
	}

	// 返回token
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "Login successful",
		"data": gin.H{
			"token":   token,
			"expires": expiration,
			"user": gin.H{
				"id":         userID,
				"username":   user.Username,
				"role":       user.Role,
				"name":       user.Name,
				"email":      user.Email,
				"last_login": formatTimeOrNull(time.Time(user.LastLogin)),
			},
		},
	})
}

// formatTimeOrNull 將時間格式化為 ISO 8601 格式，如果是零值則返回 null
func formatTimeOrNull(t time.Time) interface{} {
	if t.IsZero() {
		return nil
	}
	return t.Format(time.RFC3339)
}

// 新增一个JsonTime格式化函数
func formatJsonTimeOrNull(t sharedmodels.JsonTime) interface{} {
	tt := time.Time(t)
	if tt.IsZero() {
		return nil
	}
	return tt.Format(time.RFC3339)
}

// CreateAPITokenRequest API Token创建请求
type CreateAPITokenRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
	Duration    string   `json:"duration"` // ISO 8601持续时间格式，可选
	RateLimit   int      `json:"rate_limit"`
}

// CreateAPITokenHandler 创建API token
func (h *AuthHandler) CreateAPITokenHandler(c *gin.Context) {
	// 获取用户信息
	user, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failure",
			"message": "User information not found",
			"data":    nil,
		})
		return
	}

	userObj, ok := user.(*sharedmodels.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failure",
			"message": "User type conversion error",
			"data":    nil,
		})
		return
	}

	// 获取用户ID用于保存
	_, exists = c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  "failure",
			"message": "User ID not found",
			"data":    nil,
		})
		return
	}

	// 解析请求体
	var req struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		Duration    string   `json:"duration"`
		RateLimit   int      `json:"rate_limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  "failure",
			"message": "Invalid request data",
			"data": gin.H{
				"details": err.Error(),
			},
		})
		return
	}

	// 获取JWT密钥
	jwtSecret := h.configService.UsersConfig.JWTSecret

	// 创建API token
	tokenString, err := userObj.CreateAPITokenWithSecret(req.Name, req.Permissions, req.Duration, req.RateLimit, jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failure",
			"message": "Failed to create API token",
			"data": gin.H{
				"details": err.Error(),
			},
		})
		return
	}

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "failure",
			"message": "Failed to save API token",
			"data": gin.H{
				"details": err.Error(),
			},
		})
		return
	}

	// 返回创建的token
	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "API token created successfully",
		"data": gin.H{
			"token": tokenString,
			"name":  req.Name,
		},
	})
}
