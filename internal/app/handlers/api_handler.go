package handlers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/services"
	sharedmodels "github.com/OG-Open-Source/PanelBase/internal/shared/models"
	"github.com/OG-Open-Source/PanelBase/pkg/utils"
	"github.com/gin-gonic/gin"
)

// APIHandler API处理程序
type APIHandler struct {
	configService *services.ConfigService
}

// NewAPIHandler 创建新的API处理程序
func NewAPIHandler() *APIHandler {
	return &APIHandler{}
}

// SetConfigService 设置配置服务
func (h *APIHandler) SetConfigService(configService *services.ConfigService) {
	h.configService = configService
}

// HandleLogin 处理登录请求
func (h *APIHandler) HandleLogin(c *gin.Context) {
	var loginData struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的登录数据"})
		return
	}

	log.Printf("登录尝试: 用户=%s\n", loginData.Username)

	// 查找用户
	user, err := h.configService.UsersConfig.GetUser(loginData.Username)
	if err != nil {
		log.Printf("用户查找失败: %v\n", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 查找用戶ID
	userID := utils.FindUserIDByUsername(h.configService.UsersConfig.Users, user.Username)

	log.Printf("找到用户: %s (ID: %s, 角色: %s)\n", user.Username, userID, user.Role)

	if !user.VerifyPassword(loginData.Password) {
		log.Printf("密码验证失败，用户: %s\n", loginData.Username)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	// 更新最后登录时间
	user.LastLogin = sharedmodels.JsonTime(time.Now())

	// 使用字符串格式的过期时间（ISO 8601）
	expirationStr := formatDuration(24 * time.Hour)
	token, err := user.GenerateToken(h.configService.UsersConfig.JWTSecret, expirationStr, userID)
	if err != nil {
		log.Printf("生成令牌失败: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成令牌失败"})
		return
	}

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		log.Printf("保存用户配置失败: %v\n", err)
		// 不返回错误，继续登录流程
	}

	log.Printf("用户 %s 登录成功\n", loginData.Username)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
		"user": gin.H{
			"username": user.Username,
			"role":     user.Role,
			"name":     user.Name,
			"email":    user.Email,
		},
	})
}

// GetDashboardData 获取仪表板数据
func (h *APIHandler) GetDashboardData(c *gin.Context) {
	// TODO: Implement dashboard data retrieval
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"system_info": gin.H{
				"cpu_usage":    0,
				"memory_usage": 0,
				"disk_usage":   0,
			},
			"recent_events": []gin.H{},
		},
	})
}

// GetEvents 获取事件列表
func (h *APIHandler) GetEvents(c *gin.Context) {
	// TODO: Implement event retrieval
	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   []gin.H{},
	})
}

// CreateAPIToken 创建API token
func (h *APIHandler) CreateAPIToken(c *gin.Context) {
	var tokenData struct {
		Name        string   `json:"name" binding:"required"`
		Permissions []string `json:"permissions" binding:"required"`
		ExpiresIn   int      `json:"expires_in"` // 过期时间（小时）
		RateLimit   int      `json:"rate_limit"` // 速率限制（每分钟请求数）
	}

	if err := c.ShouldBindJSON(&tokenData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	user := c.MustGet("user").(*sharedmodels.User)
	// 将过期时间转换为字符串
	expirationStr := formatDuration(time.Duration(tokenData.ExpiresIn) * time.Hour)

	// 获取JWT密钥
	jwtSecret := h.configService.UsersConfig.JWTSecret

	// 创建API token时传入JWT密钥
	token, err := user.CreateAPIToken(tokenData.Name, tokenData.Permissions, expirationStr, tokenData.RateLimit, jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存用户配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data": gin.H{
			"token": token,
		},
	})
}

// ListAPITokens 列出用户的API tokens
func (h *APIHandler) ListAPITokens(c *gin.Context) {
	user := c.MustGet("user").(*sharedmodels.User)
	tokens := user.GetAPITokens()

	c.JSON(http.StatusOK, gin.H{
		"status": "success",
		"data":   tokens,
	})
}

// UpdateAPIToken 更新API token
func (h *APIHandler) UpdateAPIToken(c *gin.Context) {
	tokenID := c.Param("id")
	var tokenData struct {
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		ExpiresIn   int      `json:"expires_in"`
		RateLimit   int      `json:"rate_limit"`
	}

	if err := c.ShouldBindJSON(&tokenData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求数据"})
		return
	}

	user := c.MustGet("user").(*sharedmodels.User)
	// 将过期时间转换为字符串
	expirationStr := formatDuration(time.Duration(tokenData.ExpiresIn) * time.Hour)
	err := user.UpdateAPIToken(tokenID, tokenData.Name, tokenData.Permissions, expirationStr, tokenData.RateLimit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存用户配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "API令牌已更新",
	})
}

// DeleteAPIToken 删除API token
func (h *APIHandler) DeleteAPIToken(c *gin.Context) {
	tokenID := c.Param("id")
	user := c.MustGet("user").(*sharedmodels.User)

	err := user.DeleteAPIToken(tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存用户配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "API令牌已删除",
	})
}

// ResetAPITokenUsage 重置API token使用统计
func (h *APIHandler) ResetAPITokenUsage(c *gin.Context) {
	tokenID := c.Param("id")
	user := c.MustGet("user").(*sharedmodels.User)

	err := user.ResetAPITokenUsage(tokenID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 保存用户配置
	if err := h.configService.UsersConfig.Save(h.configService.BaseDir); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "保存用户配置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": "API令牌使用统计已重置",
	})
}

// 将time.Duration转换为ISO 8601持续时间格式
func formatDuration(d time.Duration) string {
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	seconds := int(d.Seconds()) % 60

	result := "PT"
	if hours > 0 {
		result += fmt.Sprintf("%dH", hours)
	}
	if minutes > 0 {
		result += fmt.Sprintf("%dM", minutes)
	}
	if seconds > 0 || (hours == 0 && minutes == 0) {
		result += fmt.Sprintf("%dS", seconds)
	}
	return result
}
