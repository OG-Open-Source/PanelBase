package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

// SetupLogWriter initializes logging to both console and a timestamped file.
// It ensures the log directory exists and returns an io.Writer that multiplexes
// output to os.Stdout and the created log file.
func SetupLogWriter(logDir string) (io.Writer, *os.File, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, nil, fmt.Errorf("failed to create log directory '%s': %w", logDir, err)
	}

	// Generate log file path
	logFileNameFormat := "2006-01-02T15_04_05Z"
	logFileName := time.Now().UTC().Format(logFileNameFormat) + ".log"
	logFilePath := filepath.Join(logDir, logFileName)

	// Open/Create log file for writing
	logFile, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open log file '%s': %w", logFilePath, err)
	}

	// Create a multi-writer to write to both stdout and the log file
	multiWriter := io.MultiWriter(os.Stdout, logFile)

	log.Printf("Logging to console and file: %s", logFilePath) // Log initialization info

	return multiWriter, logFile, nil
}
