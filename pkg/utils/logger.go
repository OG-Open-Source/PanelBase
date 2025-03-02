package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// LogLevel 定義日誌級別
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var (
	currentLevel = DEBUG
	logFile      *os.File
)

// SetLogLevel 設置日誌級別
func SetLogLevel(level LogLevel) {
	currentLevel = level
}

// InitLogger 初始化日誌系統
func InitLogger() error {
	// 創建 logs 目錄
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	// 創建日誌文件，使用當前日期作為文件名
	logFileName := filepath.Join(logsDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	logFile = file
	return nil
}

// CloseLogger 關閉日誌文件
func CloseLogger() {
	if logFile != nil {
		logFile.Close()
	}
}

// Debug 輸出調試信息
func Debug(format string, args ...interface{}) {
	logMessage(DEBUG, format, args...)
}

// Info 輸出一般信息
func Info(format string, args ...interface{}) {
	logMessage(INFO, format, args...)
}

// Warn 輸出警告信息
func Warn(format string, args ...interface{}) {
	logMessage(WARN, format, args...)
}

// Error 輸出錯誤信息
func Error(format string, args ...interface{}) {
	logMessage(ERROR, format, args...)
}

// 生成隨機標識符
func generateTraceID() string {
	b := make([]byte, 3) // 6個十六進制字符
	rand.Read(b)
	return hex.EncodeToString(b)
}

// 命令執行的跟踪ID映射
var commandTraceIDs = make(map[string]string)

// logMessage 實現日誌記錄
func logMessage(level LogLevel, format string, args ...interface{}) {
	if level < currentLevel {
		return
	}

	// 獲取當前時間 (使用 ISO 8601 格式)
	now := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")
	
	// 根據日誌級別設置標籤
	var label string
	switch level {
	case DEBUG:
		label = "DEBUG"
	case INFO:
		label = "INFO"
	case WARN:
		label = "WARN"
	case ERROR:
		label = "ERROR"
	}

	// 格式化消息
	message := fmt.Sprintf(format, args...)
	
	// 生成或獲取跟踪ID
	var traceID string
	
	// 處理命令執行相關信息
	if strings.Contains(message, "Starting execution of") {
		// 為批次生成新的跟踪ID
		traceID = generateTraceID()
		message = fmt.Sprintf("Starting batch execution")
	} else if strings.Contains(message, "Executing command") {
		// 提取命令編號
		cmdNumRe := regexp.MustCompile(`\[(\d+)\]`)
		cmdMatches := cmdNumRe.FindStringSubmatch(message)
		
		if len(cmdMatches) > 1 {
			cmdNum := cmdMatches[1]
			// 為命令生成新的跟踪ID
			traceID = generateTraceID()
			commandTraceIDs[cmdNum] = traceID
			
			// 提取命令信息
			re := regexp.MustCompile(`route \[(.*?)\].*?with args \[(.*?)\]`)
			matches := re.FindStringSubmatch(message)
			if len(matches) == 3 {
				message = fmt.Sprintf("Executing: %s %s", matches[1], matches[2])
			}
		}
	} else if strings.Contains(message, "completed successfully") {
		// 提取命令編號
		cmdNumRe := regexp.MustCompile(`\[(\d+)\]`)
		cmdMatches := cmdNumRe.FindStringSubmatch(message)
		
		if strings.Contains(message, "All commands") {
			message = "Batch execution completed"
			// 使用最後一個命令的跟踪ID
			if len(cmdMatches) > 1 {
				traceID = commandTraceIDs[cmdMatches[1]]
			} else {
				traceID = generateTraceID()
			}
		} else if len(cmdMatches) > 1 {
			cmdNum := cmdMatches[1]
			message = "Command completed"
			// 使用對應命令的跟踪ID
			traceID = commandTraceIDs[cmdNum]
		}
	}
	
	// 如果沒有設置跟踪ID，生成一個新的
	if traceID == "" {
		traceID = generateTraceID()
	}

	// 格式化日誌消息
	logEntry := fmt.Sprintf("[%s] %s | %s | %s\n", 
		label,    // 日誌級別
		now,      // 時間戳
		traceID,  // 跟踪ID
		message,  // 消息內容
	)

	// 輸出到控制台
	fmt.Fprint(os.Stderr, logEntry)

	// 寫入日誌文件
	if logFile != nil {
		logFile.WriteString(logEntry)
	}
}
