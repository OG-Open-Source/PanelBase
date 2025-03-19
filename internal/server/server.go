package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/executor"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/user"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Server represents the HTTP server
type Server struct {
	config      *config.Config
	router      *mux.Router
	upgrader    websocket.Upgrader
	auth        *middleware.AuthMiddleware
	executor    *executor.Executor
	userManager user.UserManager
	staticDir   string
}

// New creates a new server instance
func New(cfg *config.Config) (*Server, error) {
	// Create user store
	dataDir := filepath.Join("data")
	userStore, err := user.NewFileStore(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize user store: %w", err)
	}

	// Create auth middleware
	authMiddleware := middleware.NewAuthMiddleware(cfg, userStore)

	// Create task executor
	taskExecutor := executor.NewExecutor()

	// Ensure web directory exists
	staticDir := filepath.Join("web", "static")
	if err := os.MkdirAll(staticDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create static directory: %w", err)
	}

	s := &Server{
		config: cfg,
		router: mux.NewRouter(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		auth:        authMiddleware,
		executor:    taskExecutor,
		userManager: userStore,
		staticDir:   staticDir,
	}

	if err := s.setupRoutes(); err != nil {
		return nil, err
	}

	return s, nil
}

// setupRoutes sets up all the routes for the server
func (s *Server) setupRoutes() error {
	// Set up entry point
	entryRouter := s.router.PathPrefix("/" + s.config.EntryPoint).Subrouter()

	// API routes (require authentication)
	apiRouter := entryRouter.PathPrefix("/api").Subrouter()

	// WebSocket route - requires authentication
	wsRouter := apiRouter.PathPrefix("/ws").Subrouter()
	wsRouter.Use(s.auth.Authenticate)
	wsRouter.HandleFunc("", s.handleWebSocket).Methods("GET")

	// Register task routes
	taskHandler := NewTaskHandler(s.executor, s.config, s.auth)
	taskHandler.RegisterRoutes(apiRouter.PathPrefix("/tasks").Subrouter())

	// Register user routes
	userHandler := NewUserHandler(s.userManager, s.auth)
	userHandler.RegisterRoutes(apiRouter)

	// Static files (CSS, JS, images)
	staticFileServer := http.FileServer(http.Dir(s.staticDir))
	entryRouter.PathPrefix("/static/").Handler(http.StripPrefix("/"+s.config.EntryPoint+"/static/", staticFileServer))

	// Frontend routes - serve index.html for all non-API, non-static routes
	entryRouter.PathPrefix("/").HandlerFunc(s.handleFrontend)

	return nil
}

// handleWebSocket handles WebSocket connections
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// The authentication middleware already validated the user
	currentUser := middleware.GetUserFromContext(r)
	if currentUser == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("WebSocket upgrade failed: %v", err))
		http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	logger.Info(fmt.Sprintf("New WebSocket connection from %s (User: %s)", r.RemoteAddr, currentUser.Username))

	// Simple WebSocket communication loop
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			logger.Warn(fmt.Sprintf("WebSocket error: %v", err))
			break
		}

		// Echo the message back for now (in a real implementation, this would be more sophisticated)
		if err := conn.WriteMessage(messageType, p); err != nil {
			logger.Warn(fmt.Sprintf("WebSocket write error: %v", err))
			break
		}
	}
}

// handleFrontend serves the frontend application
func (s *Server) handleFrontend(w http.ResponseWriter, r *http.Request) {
	logger.Info(fmt.Sprintf("Frontend request: %s %s", r.Method, r.URL.Path))

	// Serve the index.html file from the web directory
	indexPath := filepath.Join("web", "index.html")

	// Check if the file exists
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8">
	<meta name="viewport" content="width=device-width, initial-scale=1.0">
	<title>PanelBase - Setup Required</title>
	<style>
		body {
			font-family: Arial, sans-serif;
			line-height: 1.6;
			margin: 0;
			padding: 20px;
			color: #333;
			max-width: 800px;
			margin: 0 auto;
		}
		.alert {
			background-color: #f8d7da;
			color: #721c24;
			padding: 15px;
			border-radius: 5px;
			margin-bottom: 20px;
		}
		.info {
			background-color: #e2f0fb;
			color: #0c5460;
			padding: 15px;
			border-radius: 5px;
			margin-bottom: 20px;
		}
		code {
			background-color: #f4f4f4;
			padding: 2px 5px;
			border-radius: 3px;
			font-family: monospace;
		}
		pre {
			background-color: #f4f4f4;
			padding: 15px;
			border-radius: 5px;
			overflow-x: auto;
		}
		h1, h2 {
			color: #2c3e50;
		}
	</style>
</head>
<body>
	<h1>PanelBase is running</h1>
	<div class="alert">
		<strong>Frontend not installed.</strong> The web interface is not yet installed.
	</div>
	<div class="info">
		<p>API is accessible at: <code>/%s/api</code></p>
		<p>Default admin credentials:</p>
		<ul>
			<li>Username: <code>admin</code></li>
			<li>Password: <code>admin</code></li>
		</ul>
	</div>
	<h2>API Documentation</h2>
	<p>To login and get started, use the following API endpoint:</p>
	<pre>POST /%s/api/auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin"
}</pre>

	<h2>Installation</h2>
	<p>For full installation instructions, including setting up the frontend, please refer to the README.md file.</p>
</body>
</html>
`, s.config.EntryPoint, s.config.EntryPoint)
		return
	}

	// Serve the existing index.html file
	http.ServeFile(w, r, indexPath)
}

// Start starts the server
func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.IP, s.config.Port)
	logger.Info(fmt.Sprintf("Server listening on %s", addr))
	logger.Info(fmt.Sprintf("Web interface available at http://%s:%d/%s", s.config.IP, s.config.Port, s.config.EntryPoint))
	return http.ListenAndServe(addr, s.router)
}
