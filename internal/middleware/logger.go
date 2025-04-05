package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
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
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		var statusBgColor, methodColor, resetColor, blackColor string
		statusBgColor = BgColorForStatus(param.StatusCode)
		methodColor = ColorForMethod(param.Method)
		resetColor = Reset
		blackColor = Black

		// Determine log prefix based on path
		logPrefix := "[WEB]"
		if strings.HasPrefix(param.Path, "/api/") {
			logPrefix = "[API]"
		}

		// Get cached request body and format it
		var requestBodyStr string
		bodyVal, bodyExists := param.Keys[ContextKeyRequestBody]
		isBodyEmpty := true
		if bodyExists {
			if bodyBytes, ok := bodyVal.([]byte); ok {
				if len(bodyBytes) > 0 {
					// Try to compact JSON into a single line
					compactBodyBuffer := new(bytes.Buffer)
					if err := json.Compact(compactBodyBuffer, bodyBytes); err == nil {
						// Compaction successful, use the compact JSON string
						requestBodyStr = compactBodyBuffer.String()
						isBodyEmpty = false // Consider it non-empty if compaction worked
					} else {
						// Compaction failed (not valid JSON?), log the original bytes
						requestBodyStr = "[Non-JSON Body]: " + string(bodyBytes)
						isBodyEmpty = false // Still log the non-JSON content
					}

					// Truncate if necessary AFTER compaction/conversion
					const maxBodyLogLength = 1024 // Adjust as needed
					if len(requestBodyStr) > maxBodyLogLength {
						requestBodyStr = requestBodyStr[:maxBodyLogLength] + "... (truncated)"
					}
				} else {
					// Body exists but is empty
					requestBodyStr = "[Empty Body Data]"
					// isBodyEmpty remains true
				}
			} else {
				requestBodyStr = "[Invalid Body Cache Format]"
				isBodyEmpty = false // Treat format error as non-empty for logging
			}
		} else {
			requestBodyStr = "[No Body Cache]"
			// isBodyEmpty remains true
		}

		// Get User ID (sub) from context if available, default to "-"
		var userIDStr string = "-"
		userIDVal, userExists := param.Keys[ContextKeyUserID]
		if userExists {
			if uid, ok := userIDVal.(string); ok {
				userIDStr = uid
			} else {
				userIDStr = "[Invalid Sub Format]"
			}
		}

		// Main log line format
		logLine := fmt.Sprintf("%s %s | %s%s %3d %s | %13v | %15s |%s %-7s %s %s%s",
			logPrefix,
			param.TimeStamp.UTC().Format(time.RFC3339),
			statusBgColor, blackColor, param.StatusCode, resetColor,
			param.Latency,
			param.ClientIP,
			methodColor, param.Method, resetColor,
			param.Path,
			param.ErrorMessage,
		)

		// Append second line IF body exists, regardless of method or auth status
		if !isBodyEmpty {
			// userIDStr will correctly be "-" if user wasn't authenticated
			logLine += fmt.Sprintf("\n      %s: %s", userIDStr, requestBodyStr)
		}

		return logLine + "\n" // Ensure final newline
	})
}
