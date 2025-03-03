package server

import (
    "fmt"
    "net/http"
    "path"
    "github.com/gorilla/mux"
    "github.com/gorilla/websocket"
    "github.com/OG-Open-Source/PanelBase/internal/config"
    "github.com/OG-Open-Source/PanelBase/internal/middleware"
)

type Server struct {
    config *config.Config
    router *mux.Router
    upgrader websocket.Upgrader
    auth *middleware.AuthMiddleware
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
    }
    
    s.setupRoutes()
    return s
}

func (s *Server) setupRoutes() {
    // 設置入口點
    entryRouter := s.router.PathPrefix("/" + s.config.EntryPoint).Subrouter()
    
    // API 路由 (需要認證)
    api := entryRouter.PathPrefix("/api").Subrouter()
    api.Use(s.auth.Authenticate)
    api.HandleFunc("/ws", s.handleWebSocket)
    
    // 代理路由
    entryRouter.PathPrefix("/").HandlerFunc(s.handleProxy)
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
    conn, err := s.upgrader.Upgrade(w, r, nil)
    if err != nil {
        http.Error(w, "Could not upgrade connection", http.StatusInternalServerError)
        return
    }
    defer conn.Close()
    
    // WebSocket 處理邏輯
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
    // 代理邏輯實現
}

func (s *Server) Start() error {
    addr := fmt.Sprintf(":%d", s.config.Port)
    return http.ListenAndServe(addr, s.router)
} 