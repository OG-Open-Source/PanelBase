package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	"github.com/OG-Open-Source/PanelBase/internal/shared/models"
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 验证用户名和密码
	if h.configService.UsersConfig == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "配置加载失败"})
		return
	}

	// 获取用户
	user, err := h.configService.UsersConfig.GetUser(req.Username)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户不存在"})
		return
	}

	// 验证密码
	if !user.VerifyPassword(req.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "密码错误"})
		return
	}

	// 使用配置文件中的JWT密钥
	jwtSecret := h.configService.UsersConfig.JWTSecret

	// 获取过期时间
	expiration := req.Duration
	if expiration == "" {
		// 使用配置文件中的默认过期时间
		if h.configService.Config != nil && h.configService.Config.Auth.JWTExpiration > 0 {
			expiration = fmt.Sprintf("%d", h.configService.Config.Auth.JWTExpiration)
		} else {
			expiration = "PT24H" // 默认24小时
		}
	}

	// 生成JWT token
	token, err := user.GenerateToken(jwtSecret, expiration)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成token失败"})
		return
	}

	// 更新最后登录时间
	user.LastLogin = time.Now()

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		// 不返回错误，继续登录流程
		fmt.Printf("保存用户配置失败: %v\n", err)
	}

	// 返回token
	c.JSON(http.StatusOK, gin.H{
		"token":   token,
		"expires": expiration,
		"user": gin.H{
			"id":       user.ID,
			"username": user.Username,
			"role":     user.Role,
			"name":     user.Name,
			"email":    user.Email,
		},
	})
}

// CreateAPITokenRequest API Token创建请求
type CreateAPITokenRequest struct {
	Name        string   `json:"name" binding:"required"`
	Permissions []string `json:"permissions"`
	Duration    string   `json:"duration"` // ISO 8601持续时间格式，可选
	RateLimit   int      `json:"rate_limit"`
}

// CreateAPITokenHandler 创建API Token
func (h *AuthHandler) CreateAPITokenHandler(c *gin.Context) {
	var req CreateAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	// 获取当前用户
	userInterface, exists := c.Get("user")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	user, ok := userInterface.(*models.User)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "用户数据类型错误"})
		return
	}

	// 创建API token
	token, err := user.CreateAPIToken(req.Name, req.Permissions, req.Duration, req.RateLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建API token失败", "details": err.Error()})
		return
	}

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存配置失败", "details": err.Error()})
		return
	}

	// 返回token信息
	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"info": gin.H{
			"name":        req.Name,
			"permissions": req.Permissions,
			"duration":    req.Duration,
			"rate_limit":  req.RateLimit,
		},
	})
}
