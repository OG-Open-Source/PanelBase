package server

import (
	"fmt"
	"net/http"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/OG-Open-Source/PanelBase/internal/config"
	"github.com/OG-Open-Source/PanelBase/internal/middleware"
	"github.com/OG-Open-Source/PanelBase/internal/logger"
	"github.com/OG-Open-Source/PanelBase/internal/executor"
)

type Server struct {
	config *config.Config
	router *mux.Router
	upgrader websocket.Upgrader
	auth *middleware.AuthMiddleware
	executor *executor.Executor
}

func New(cfg *config.Config) *Server {
	s := &Server{
		config: cfg,
		router: mux.NewRouter(),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		auth: middleware.NewAuthMiddleware(cfg),
		executor: executor.NewExecutor(),
	}
	
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// 設置入口點
	entryRouter := s.router.PathPrefix("/" + s.config.EntryPoint).Subrouter()
	
	// API 路由 (需要認證)
	apiRouter := entryRouter.PathPrefix("/api").Subrouter()
	apiRouter.Use(s.auth.Authenticate)
	apiRouter.HandleFunc("/ws", s.handleWebSocket)
	
	// 代理路由
	entryRouter.PathPrefix("/").HandlerFunc(s.handleProxy)
	
	// 添加任務相關的 API 路由
	taskHandler := NewTaskHandler(s.executor, s.config)
	taskHandler.RegisterRoutes(apiRouter)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("WebSocket upgrade failed: %v", err))
		http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
		return
	}
	defer conn.Close()
	
	logger.Info(fmt.Sprintf("New WebSocket connection from %s", r.RemoteAddr))
	// WebSocket 處理邏輯
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	logger.Info(fmt.Sprintf("Proxy request: %s %s", r.Method, r.URL.Path))
	// 代理邏輯實現
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("%s:%d", s.config.IP, s.config.Port)
	logger.Info(fmt.Sprintf("Server listening on %s", addr))
	return http.ListenAndServe(addr, s.router)
} 