package utils

import (
	"fmt"
	"log"
	"log/syslog"
	"os"
	"path/filepath"
	"time"
)

// Logger 日誌記錄器
type Logger struct {
	stdLogger   *log.Logger    // 標準輸出
	sysLogger   *syslog.Writer // 系統日誌
	fileLogger  *log.Logger    // 文件日誌
	programTag  string
	logFile     *os.File
}

// LogLevel 日誌級別
type LogLevel string

const (
	DEBUG LogLevel = "DEBUG"
	INFO  LogLevel = "INFO"
	WARN  LogLevel = "WARN"
	ERROR LogLevel = "ERROR"
	FATAL LogLevel = "FATAL"
)

// NewLogger 創建新的日誌記錄器
func NewLogger() *Logger {
	// 創建日誌目錄
	logDir := filepath.Join("internal", "logs")
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: '%v'", err)
	}

	// 創建日誌文件（按日期）
	currentTime := time.Now().UTC()
	logFileName := filepath.Join(logDir, fmt.Sprintf("%s.log", currentTime.Format("2006-01-02")))
	logFile, err := os.OpenFile(logFileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: '%v'", err)
	}

	// 創建系統日誌連接
	sysLogger, err := syslog.New(syslog.LOG_INFO|syslog.LOG_LOCAL0, "panelbase")
	if err != nil {
		// 如果無法連接到系統日誌，只使用標準輸出和文件日誌
		return &Logger{
			stdLogger:  log.New(os.Stdout, "", 0),
			fileLogger: log.New(logFile, "", 0),
			logFile:    logFile,
			programTag: "panelbase",
		}
	}

	return &Logger{
		stdLogger:  log.New(os.Stdout, "", 0),
		sysLogger:  sysLogger,
		fileLogger: log.New(logFile, "", 0),
		logFile:    logFile,
		programTag: "panelbase",
	}
}

// formatMessage 格式化日誌訊息
func (l *Logger) formatMessage(level LogLevel, message string) string {
	// 格式化時間戳
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	// 返回格式化的訊息，移除檔案名和行號
	return fmt.Sprintf("[%s] %s | %s",
		level,
		timestamp,
		message,
	)
}

// writeLog 寫入日誌到所有輸出
func (l *Logger) writeLog(level LogLevel, message string) {
	formattedMessage := l.formatMessage(level, message)
	
	// 輸出到標準輸出
	l.stdLogger.Print(formattedMessage)

	// 輸出到文件
	if l.fileLogger != nil {
		l.fileLogger.Print(formattedMessage)
	}

	// 如果系統日誌可用，也輸出到系統日誌
	if l.sysLogger != nil {
		switch level {
		case DEBUG:
			l.sysLogger.Debug(formattedMessage)
		case INFO:
			l.sysLogger.Info(formattedMessage)
		case WARN:
			l.sysLogger.Warning(formattedMessage)
		case ERROR:
			l.sysLogger.Err(formattedMessage)
		case FATAL:
			l.sysLogger.Crit(formattedMessage)
		}
	}
}

// Debug 記錄調試級別日誌
func (l *Logger) Debug(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.writeLog(DEBUG, message)
}

// Info 記錄信息級別日誌
func (l *Logger) Info(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.writeLog(INFO, message)
}

// Warn 記錄警告級別日誌
func (l *Logger) Warn(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.writeLog(WARN, message)
}

// Error 記錄錯誤級別日誌
func (l *Logger) Error(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.writeLog(ERROR, message)
}

// Fatal 記錄致命錯誤日誌並結束程序
func (l *Logger) Fatal(format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	l.writeLog(FATAL, message)
	os.Exit(1)
}

// Close 關閉日誌記錄器
func (l *Logger) Close() {
	if l.sysLogger != nil {
		l.sysLogger.Close()
	}
	if l.logFile != nil {
		l.logFile.Close()
	}
}

// 全局日誌實例
var defaultLogger = NewLogger()

// 全局日誌函數
func Debug(format string, args ...interface{}) {
	defaultLogger.Debug(format, args...)
}

func Info(format string, args ...interface{}) {
	defaultLogger.Info(format, args...)
}

func Warn(format string, args ...interface{}) {
	defaultLogger.Warn(format, args...)
}

func Error(format string, args ...interface{}) {
	defaultLogger.Error(format, args...)
}

func Fatal(format string, args ...interface{}) {
	defaultLogger.Fatal(format, args...)
}

// Close 關閉全局日誌記錄器
func Close() {
	defaultLogger.Close()
}
