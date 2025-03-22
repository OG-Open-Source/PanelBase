package api

import (
	"context"

	"github.com/OG-Open-Source/PanelBase/internal/handlers"
	"github.com/OG-Open-Source/PanelBase/pkg/config"
	"github.com/OG-Open-Source/PanelBase/pkg/logger"
	"github.com/OG-Open-Source/PanelBase/pkg/theme"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

// Server represents the API server
type Server struct {
	echo *echo.Echo
	cfg  *config.Config
}

// New creates a new API server
func New(cfg *config.Config, themeConfig *theme.ThemeConfig) *Server {
	e := echo.New()

	// Add middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Create handlers
	themeHandler := handlers.NewThemeHandler(themeConfig)

	// Register routes
	themeHandler.RegisterRoutes(e, cfg.Server.Entry)

	// Add health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(200, map[string]string{
			"status": "ok",
		})
	})

	return &Server{
		echo: e,
		cfg:  cfg,
	}
}

// Start starts the API server
func (s *Server) Start() error {
	logger.Logger.Info().
		Str("address", s.cfg.GetServerAddress()).
		Str("entryURL", s.cfg.GetEntryURL()).
		Msg("Starting API server")

	return s.echo.Start(s.cfg.GetServerAddress())
}

// Shutdown gracefully shuts down the API server
func (s *Server) Shutdown(ctx context.Context) error {
	logger.Logger.Info().Msg("Shutting down API server")
	return s.echo.Shutdown(ctx)
}
