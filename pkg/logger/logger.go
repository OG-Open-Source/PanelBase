package logger

import (
	"io"
	"os"
	"time"

	"github.com/OG-Open-Source/PanelBase/pkg/config"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger is the global logger
var Logger zerolog.Logger

// InitLogger initializes the logger with the specified configuration
func InitLogger(cfg *config.Config) {
	// Configure log level
	logLevel := zerolog.InfoLevel
	switch cfg.Logging.Level {
	case "debug":
		logLevel = zerolog.DebugLevel
	case "info":
		logLevel = zerolog.InfoLevel
	case "warn":
		logLevel = zerolog.WarnLevel
	case "error":
		logLevel = zerolog.ErrorLevel
	case "fatal":
		logLevel = zerolog.FatalLevel
	case "panic":
		logLevel = zerolog.PanicLevel
	}
	zerolog.SetGlobalLevel(logLevel)

	// Setup pretty console writer for development
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}

	var writers []io.Writer
	writers = append(writers, consoleWriter)

	// Setup file logger with rotation when configured
	if cfg.Logging.File != "" {
		// Configure lumberjack for log rotation
		fileWriter := &lumberjack.Logger{
			Filename:   cfg.Logging.File,
			MaxSize:    cfg.Logging.MaxSize,
			MaxBackups: cfg.Logging.MaxBackups,
			MaxAge:     cfg.Logging.MaxAge,
			Compress:   true,
		}
		writers = append(writers, fileWriter)
	}

	// Create multi-writer
	mw := io.MultiWriter(writers...)

	// Create the logger
	Logger = zerolog.New(mw).With().Timestamp().Caller().Logger()

	Logger.Info().Msg("Logger initialized")
}
