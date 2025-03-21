package logger

import (
	"os"
	"path/filepath"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *logrus.Logger

func InitLogger() error {
	cfg := config.GetConfig()
	logConfig := cfg.Logging

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logConfig.File), 0755); err != nil {
		return err
	}

	// Configure log rotation
	rotator := &lumberjack.Logger{
		Filename:   logConfig.File,
		MaxSize:    logConfig.MaxSize, // MB
		MaxBackups: logConfig.MaxBackups,
		MaxAge:     logConfig.MaxAge, // days
		Compress:   true,
	}

	// Create logger
	Log = logrus.New()
	Log.SetOutput(rotator)

	// Set log level
	level, err := logrus.ParseLevel(logConfig.Level)
	if err != nil {
		return err
	}
	Log.SetLevel(level)

	// Set format
	Log.SetFormatter(&logrus.JSONFormatter{})

	return nil
}
