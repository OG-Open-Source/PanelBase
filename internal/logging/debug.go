package logging

import (
	"fmt"
	"log"
	"sync/atomic"
	"time"
)

var debugEnabled atomic.Bool

// LogLevel defines different logging levels.
// Using iota for auto-incrementing constants.
type LogLevel int

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelError
	// LevelPlain removed - encourage leveled logging
)

func init() {
	// Set default level
	debugEnabled.Store(false) // Debug is off by default
}

// logInternal is the core logging function.
// It formats the message with timestamp, level, module, and optionally action.
func logInternal(level LogLevel, module string, action string, format string, v ...interface{}) {
	// Skip Debug logs if not in debug mode
	if level == LevelDebug && !debugEnabled.Load() {
		return
	}

	timestamp := time.Now().UTC().Format(time.RFC3339)
	var levelPrefix string
	var message string

	switch level {
	case LevelDebug:
		levelPrefix = "[DEBUG]"
	case LevelInfo:
		levelPrefix = "[INFO] " // Added space for alignment
	case LevelError:
		levelPrefix = "[ERROR]"
	default:
		levelPrefix = "[?????]"
	}

	// Format the core message first
	coreMessage := fmt.Sprintf(format, v...)

	// Construct final log string based on debug mode
	if debugEnabled.Load() {
		// Debug Mode: Timestamp [LEVEL] | MODULE.ACTION | Message
		message = fmt.Sprintf("%s %s | %s.%s | %s",
			timestamp, levelPrefix, module, action, coreMessage)
	} else {
		// Release/Test Mode: Timestamp [LEVEL] | MODULE | Message (Action omitted)
		// Use %-7s for levelPrefix to maintain alignment with debug format potentially
		message = fmt.Sprintf("%s %-7s | %s | %s",
			timestamp, levelPrefix, module, coreMessage)
	}

	log.Println(message) // Use Println as we manually format the whole line
}

// SetDebugMode enables or disables debug logging.
func SetDebugMode(enabled bool) {
	debugEnabled.Store(enabled)
	modeStr := "disabled"
	if enabled {
		modeStr = "enabled"
	}
	logInternal(LevelInfo, "LOGGING", "SET MODE", "Debug logging %s.", modeStr)
}

// DebugPrintf logs a message at Debug level with module and action.
func DebugPrintf(module string, action string, format string, v ...interface{}) {
	logInternal(LevelDebug, module, action, format, v...)
}

// Printf logs a message at Info level with module and action.
func Printf(module string, action string, format string, v ...interface{}) {
	logInternal(LevelInfo, module, action, format, v...)
}

// ErrorPrintf logs a message at Error level with module and action.
func ErrorPrintf(module string, action string, format string, v ...interface{}) {
	logInternal(LevelError, module, action, format, v...)
}

/* // Printf and Println removed to encourage leveled logging
// Printf logs a message without a level prefix.
func Printf(format string, v ...interface{}) {
	logInternal(LevelPlain, format, v...)
}

// Println logs a message without a level prefix, handling multiple args.
func Println(v ...interface{}) {
	// Convert multiple args to a single string format specifier
	format := fmt.Sprint(v...)
	logInternal(LevelPlain, format) // No format specifiers needed now
}
*/
