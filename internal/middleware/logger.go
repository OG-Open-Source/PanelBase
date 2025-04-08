package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
	// "github.com/rs/zerolog/log" // No longer needed for audience check
	// "github.com/golang-jwt/jwt/v5" // No longer needed for audience check
)

// ANSI color codes (Text)
const (
	Reset   = "\033[0m"
	Red     = "\033[31m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
	Black   = "\033[30m"
)

// ANSI background color codes
const (
	BgRed     = "\033[41m"
	BgGreen   = "\033[42m"
	BgYellow  = "\033[43m"
	BgBlue    = "\033[44m"
	BgMagenta = "\033[45m"
	BgCyan    = "\033[46m"
	BgWhite   = "\033[47m"
)

// BgColorForStatus returns a background color based on the status code.
func BgColorForStatus(code int) string {
	switch {
	case code >= 200 && code < 300:
		return BgGreen
	case code >= 300 && code < 400:
		return BgCyan // Redirects
	case code >= 400 && code < 500:
		return BgYellow // Client errors
	default: // >= 500
		return BgRed // Server errors
	}
}

// ColorForMethod returns a text color based on the HTTP method.
// Keep this for method coloring
func ColorForMethod(method string) string {
	switch method {
	case "GET":
		return Blue
	case "POST":
		return Cyan
	case "PUT":
		return Yellow
	case "DELETE":
		return Red
	case "PATCH":
		return Green
	case "HEAD":
		return Magenta
	case "OPTIONS":
		return White
	default:
		return Reset
	}
}

// CustomLogger returns a gin.HandlerFunc (middleware) that logs requests.
func CustomLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Stop timer
		duration := time.Since(start)

		// Get status code
		statusCode := c.Writer.Status()

		// Get client IP
		clientIP := c.ClientIP()

		// Get request method and path
		method := c.Request.Method
		path := c.Request.URL.Path

		// Get user agent
		userAgent := c.Request.UserAgent()

		// Get request ID from context
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = "unknown"
		}

		// Get user ID from context
		userID := c.GetString("user_id")
		if userID == "" {
			userID = "anonymous"
		}

		// Log the request details
		log.Printf("%s %s %s %s %s %s %d %v %s",
			time.Now().UTC().Format(time.RFC3339),
			requestID,
			clientIP,
			method,
			path,
			userID,
			statusCode,
			duration,
			userAgent,
		)
	}
}

// getStatusBgColor 根据状态码返回背景颜色
func getStatusBgColor(status int) string {
	switch {
	case status >= 500:
		return "\033[41m" // 红色背景
	case status >= 400:
		return "\033[43m" // 黄色背景
	case status >= 300:
		return "\033[46m" // 青色背景
	case status >= 200:
		return "\033[42m" // 绿色背景
	default:
		return "\033[47m" // 白色背景
	}
}

// getMethodColor 根据请求方法返回颜色
func getMethodColor(method string) string {
	switch method {
	case "GET":
		return "\033[36m" // 青色
	case "POST":
		return "\033[32m" // 绿色
	case "PUT":
		return "\033[33m" // 黄色
	case "DELETE":
		return "\033[31m" // 红色
	case "PATCH":
		return "\033[35m" // 紫色
	default:
		return "\033[37m" // 白色
	}
}
