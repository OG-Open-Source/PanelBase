package main

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/api"
	"github.com/OG-Open-Source/PanelBase/pkg/config"
	"github.com/OG-Open-Source/PanelBase/pkg/logger"
	"github.com/OG-Open-Source/PanelBase/pkg/theme"
	"github.com/rs/zerolog/log"
)

func main() {
	// Load configuration
	// The LoadConfig function will create configs directory and default config if needed
	cfg, err := config.LoadConfig("")
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	// Initialize logger
	logger.InitLogger(cfg)

	// Load theme configuration
	// The LoadTheme function will create default theme if needed
	themeConfig, err := theme.LoadTheme("")
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to load theme configuration")
	}

	// Initialize API server
	server := api.New(cfg, themeConfig)

	// Start the server in a goroutine
	go func() {
		logger.Logger.Info().
			Str("entryURL", cfg.GetEntryURL()).
			Msg("Server is accessible at this URL path")

		if err := server.Start(); err != nil {
			// 检查是否是正常的服务器关闭错误
			if !strings.Contains(err.Error(), "Server closed") {
				logger.Logger.Error().Err(err).Msg("Server error")
			}
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Logger.Info().Msg("Shutting down server...")

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := server.Shutdown(ctx); err != nil {
		logger.Logger.Error().Err(err).Msg("Server forced to shutdown")
	}

	logger.Logger.Info().Msg("Server exiting")
}
