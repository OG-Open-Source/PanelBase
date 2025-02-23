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
	DBUG LogLevel = iota
	INFO
	WARN
	EROR
)

const (
	colorPurple = "\033[35m"
	colorBlue   = "\033[34m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorReset  = "\033[0m"
)

type Logger struct {
	dbugLogger *log.Logger
	infoLogger *log.Logger
	warnLogger *log.Logger
	erorLogger *log.Logger
}

var defaultLogger *Logger

func InitLogger() error {
	logDir := "logs"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", time.Now().Format("2006-01-02")))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	defaultLogger = &Logger{
		dbugLogger: log.New(file, "[DBUG] ", log.Ldate|log.Ltime|log.Lshortfile),
		infoLogger: log.New(file, "[INFO] ", log.Ldate|log.Ltime|log.Lshortfile),
		warnLogger: log.New(file, "[WARN] ", log.Ldate|log.Ltime|log.Lshortfile),
		erorLogger: log.New(file, "[EROR] ", log.Ldate|log.Ltime|log.Lshortfile),
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
	case DBUG:
		defaultLogger.dbugLogger.Output(2, msg)
		fmt.Printf("%s[DBUG]%s %s\n", colorPurple, colorReset, msg)
	case INFO:
		defaultLogger.infoLogger.Output(2, msg)
		fmt.Printf("%s[INFO]%s %s\n", colorBlue, colorReset, msg)
	case WARN:
		defaultLogger.warnLogger.Output(2, msg)
		fmt.Printf("%s[WARN]%s %s\n", colorYellow, colorReset, msg)
	case EROR:
		defaultLogger.erorLogger.Output(2, msg)
		fmt.Printf("%s[EROR]%s %s\n", colorRed, colorReset, msg)
	}
} 