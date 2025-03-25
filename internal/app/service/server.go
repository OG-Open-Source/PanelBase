package service

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/OG-Open-Source/PanelBase/internal/app/config"
	"github.com/OG-Open-Source/PanelBase/internal/app/handler"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Server represents the HTTP server
type Server struct {
	config     *config.Config
	router     *gin.Engine
	httpServer *http.Server
}

// NewServer creates a new server instance
func NewServer(cfg *config.Config) *Server {
	// Set Gin mode based on log level
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create router
	router := gin.New()
	router.Use(gin.Recovery())

	// Use custom logger
	router.Use(func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		if statusCode >= 500 {
			logrus.Errorf("[%s] %s %d %s", method, path, statusCode, latency)
		} else if statusCode >= 400 {
			logrus.Warnf("[%s] %s %d %s", method, path, statusCode, latency)
		} else {
			logrus.Infof("[%s] %s %d %s", method, path, statusCode, latency)
		}
	})

	// Create server
	server := &Server{
		config: cfg,
		router: router,
	}

	// Setup routes
	server.setupRoutes()

	return server
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.Server.Host, s.config.Server.Port)

	// Initialize HTTP server
	s.httpServer = &http.Server{
		Addr:    addr,
		Handler: s.router,
	}

	// Channel for capturing OS signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		logrus.Infof("Server starting on %s", addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	<-stop
	logrus.Info("Shutting down server...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := s.httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	logrus.Info("Server stopped gracefully")
	return nil
}

// setupRoutes configures all the API routes
func (s *Server) setupRoutes() {
	// Create handlers
	h := handler.NewHandler(s.config)

	// API routes
	api := s.router.Group("/api")
	{
		// Authentication routes
		auth := api.Group("/auth")
		{
			auth.POST("/login", h.Login)
		}

		// Protected routes (require authentication)
		protected := api.Group("/")
		protected.Use(h.AuthMiddleware())
		{
			// System routes
			system := protected.Group("/system")
			{
				system.GET("/info", h.GetSystemInfo)
			}

			// User management
			users := protected.Group("/users")
			{
				users.GET("/", h.ListUsers)
				users.GET("/:id", h.GetUser)
				users.POST("/", h.CreateUser)
				users.PUT("/:id", h.UpdateUser)
				users.DELETE("/:id", h.DeleteUser)
			}
		}
	}

	// Static files
	s.router.Static("/assets", "./web/themes/default/assets")
	s.router.LoadHTMLGlob("web/themes/default/templates/*")

	// HTML routes
	s.router.GET("/", h.IndexPage)
	s.router.GET("/login", h.LoginPage)

	// Admin entry point with secret path
	s.router.GET("/"+s.config.Server.Entry, h.AdminEntryPage)
}
