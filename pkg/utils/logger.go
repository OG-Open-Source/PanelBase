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

	flags := log.Ldate | log.Ltime
	defaultLogger = &Logger{
		dbugLogger: log.New(file, "[DBUG] ", flags),
		infoLogger: log.New(file, "[INFO] ", flags),
		warnLogger: log.New(file, "[WARN] ", flags),
		erorLogger: log.New(file, "[EROR] ", flags),
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
	timeStr := time.Now().Format("2006/01/02 15:04:05")

	switch level {
	case DBUG:
		defaultLogger.dbugLogger.Output(2, msg)
		fmt.Printf("%s[DBUG]%s %s %s\n", colorPurple, colorReset, timeStr, msg)
	case INFO:
		defaultLogger.infoLogger.Output(2, msg)
		fmt.Printf("%s[INFO]%s %s %s\n", colorBlue, colorReset, timeStr, msg)
	case WARN:
		defaultLogger.warnLogger.Output(2, msg)
		fmt.Printf("%s[WARN]%s %s %s\n", colorYellow, colorReset, timeStr, msg)
	case EROR:
		defaultLogger.erorLogger.Output(2, msg)
		fmt.Printf("%s[EROR]%s %s %s\n", colorRed, colorReset, timeStr, msg)
	}
}