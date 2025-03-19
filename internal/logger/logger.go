package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Level string

const (
	INFO  Level = "INFO"
	ERROR Level = "ERROR"
	DEBUG Level = "DEBUG"
	WARN  Level = "WARN"
	FATAL Level = "FATAL"
)

var logFile *os.File

// 初始化日誌文件
func Init() {
	// 創建 logs 目錄
	if err := os.MkdirAll("logs", 0755); err != nil {
		fmt.Printf("Failed to create logs directory: %v\n", err)
		os.Exit(1)
	}

	// 使用 UTC 日期作為日誌文件名
	currentDate := time.Now().UTC().Format("2006-01-02")
	logPath := filepath.Join("logs", fmt.Sprintf("%s.log", currentDate))

	// 打開日誌文件（追加模式）
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Printf("Failed to open log file: %v\n", err)
		os.Exit(1)
	}

	logFile = file
}

// 格式化日誌消息
func formatLog(level Level, message string) string {
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	return fmt.Sprintf("[%s] %s | %s\n", level, timestamp, message)
}

// 寫入日誌
func log(level Level, message string) {
	// 使用 UTC 時間
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	logEntry := fmt.Sprintf("[%s] %s | %s\n", level, timestamp, message)

	// 輸出到控制台
	if level == FATAL {
		fmt.Fprint(os.Stderr, logEntry)
	} else {
		fmt.Fprint(os.Stdout, logEntry)
	}

	// 輸出到文件
	if logFile != nil {
		if _, err := logFile.WriteString(logEntry); err != nil {
			fmt.Printf("Failed to write to log file: %v\n", err)
		}
	}

	// 如果是致命錯誤，關閉文件並退出
	if level == FATAL {
		if logFile != nil {
			logFile.Close()
		}
		os.Exit(1)
	}
}

// 清理資源
func Cleanup() {
	if logFile != nil {
		logFile.Close()
	}
}

// 導出的日誌函數
func Info(message string) {
	log(INFO, message)
}

func Error(message string) {
	log(ERROR, message)
}

func Debug(message string) {
	log(DEBUG, message)
}

func Warn(message string) {
	log(WARN, message)
}

func Fatal(message string) {
	log(FATAL, message)
}