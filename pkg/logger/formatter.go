package logger

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ANSI Color Codes (Text Only)
const (
	reset = "\033[0m"
	// Methods
	blue    = "\033[34m"
	cyan    = "\033[36m"
	green   = "\033[32m"
	red     = "\033[31m"
	white   = "\033[37m"
	yellow  = "\033[33m"
	magenta = "\033[35m"
	// Status Codes
	// Gin's default uses background colors for status, we keep it here
)

// MethodToColor maps HTTP methods to text colors
func methodToColor(method string, useColor bool) string {
	if !useColor {
		return method
	}
	var color string
	switch method {
	case http.MethodGet:
		color = blue
	case http.MethodPost:
		color = cyan
	case http.MethodPut:
		color = yellow
	case http.MethodDelete:
		color = red
	case http.MethodPatch:
		color = green
	case http.MethodHead:
		color = magenta
	case http.MethodOptions:
		color = white
	default:
		color = reset // Or handle unknown methods differently
	}
	return fmt.Sprintf("%s%-7s%s", color, method, reset)
}

// CustomLogFormatter is the custom log format function
func CustomLogFormatter(params gin.LogFormatterParams) string {
	var statusColor, resetColor string
	useColor := params.IsOutputColor()
	if useColor {
		statusColor = params.StatusCodeColor() // Keep Gin's default background for status
		resetColor = params.ResetColor()
	}

	// Determine prefix based on path
	var prefix string
	if strings.HasPrefix(params.Path, "/api/") {
		prefix = "[API]"
	} else {
		prefix = "[WEB]"
	}

	// Format timestamp to RFC3339 UTC
	timestamp := params.TimeStamp.UTC().Format(time.RFC3339)

	// Format status code: <color><space>CODE<space></color>
	// Pad with spaces outside the color for the final structure | <colored_status> |
	statusCodeFormatted := fmt.Sprintf("%s %3d %s", statusColor, params.StatusCode, resetColor)

	// Format method with explicit text color
	methodStr := methodToColor(params.Method, useColor)

	// Format latency
	latency := params.Latency
	if latency > time.Minute {
		latency = latency.Truncate(time.Second)
	}

	// Construct the log string: Ensure single '|' between timestamp and status
	return fmt.Sprintf("%s %s | %s | %13v | %15s | %s %s\n%s",
		prefix,
		timestamp,
		statusCodeFormatted,
		latency,
		params.ClientIP,
		methodStr,
		params.Path,
		params.ErrorMessage,
	)
}
