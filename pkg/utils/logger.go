package utils

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
}

var defaultLogger *Logger

func InitLogger() error {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("panelbase_%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	defaultLogger = &Logger{
		debugLogger: log.New(file, "[DEBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
		infoLogger:  log.New(file, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLogger:  log.New(file, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile),
		errorLogger: log.New(file, "[ERROR] ", log.Ldate|log.Ltime|log.Lshortfile),
	}

	return nil
}

func Log(level LogLevel, format string, v ...interface{}) {
	if defaultLogger == nil {
		if err := InitLogger(); err != nil {
			log.Printf("Failed to initialize logger: %v", err)
			return
		}
	}

	msg := fmt.Sprintf(format, v...)
	switch level {
	case DEBUG:
		defaultLogger.debugLogger.Output(2, msg)
	case INFO:
		defaultLogger.infoLogger.Output(2, msg)
	case WARN:
		defaultLogger.warnLogger.Output(2, msg)
	case ERROR:
		defaultLogger.errorLogger.Output(2, msg)
	}
} 