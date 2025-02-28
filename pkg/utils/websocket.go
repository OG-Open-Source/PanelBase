package utils

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"net/http"
	"os"
	"strings"
	"sync"
)

// WebSocket 相關類型定義
type WebSocketMessage struct {
	Status  string `json:"status"`  // success, failure, running 等
	Message string `json:"message"` // 消息文本
	Data    string `json:"data"`    // 數據內容
	Command string `json:"command"` // 命令名稱
}

// CommandRequest 命令請求結構
type CommandRequest struct {
	Commands []Command `json:"commands"`
}

// Command 命令結構
type Command struct {
	Name string   `json:"name"`
	Args []string `json:"args"`
}

// WebSocketManager WebSocket 管理器
type WebSocketManager struct {
	// 升級器用於將 HTTP 連接升級為 WebSocket
	upgrader websocket.Upgrader
	// 存儲所有活動的 WebSocket 連接
	connections map[*websocket.Conn]*sync.Mutex
	// 互斥鎖保護連接映射
	mutex sync.RWMutex
	baseURL string
}

// NewWebSocketManager 創建新的 WebSocket 管理器
func NewWebSocketManager() *WebSocketManager {
	// 從環境變量讀取配置
	ip := os.Getenv("IP")
	if ip == "" {
		Error("Environment variable 'IP' is not set")
		return nil
	}
	
	port := os.Getenv("PORT")
	if port == "" {
		Error("Environment variable 'PORT' is not set")
		return nil
	}
	
	entry := os.Getenv("ENTRY")
	if entry == "" {
		Error("Environment variable 'ENTRY' is not set")
		return nil
	}

	// 構建基礎 URL
	baseURL := fmt.Sprintf("http://%s:%s/%s", ip, port, entry)

	// 創建 WebSocket 管理器
	manager := &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		connections: make(map[*websocket.Conn]*sync.Mutex),
		baseURL:     baseURL,
	}

	// 生成 main.js 文件
	if err := manager.generateMainJS(); err != nil {
		Error("Failed to generate main.js: %v", err)
		return nil
	}

	return manager
}

// generateMainJS 生成 main.js 文件
func (m *WebSocketManager) generateMainJS() error {
	// 讀取原始的 main.js
	content, err := os.ReadFile("main.js")
	if err != nil {
		return fmt.Errorf("failed to read main.js: %v", err)
	}

	// 替換 baseUrl
	processed := strings.Replace(
		string(content),
		`this.baseUrl = 'http://IP:PORT/ENTRY';`,
		fmt.Sprintf(`this.baseUrl = '%s';`, m.baseURL),
		1,
	)

	// 寫入回原始文件
	if err := os.WriteFile("main.js", []byte(processed), 0644); err != nil {
		return fmt.Errorf("failed to write main.js: %v", err)
	}

	return nil
}

// HandleConnection 處理新的 WebSocket 連接
func (m *WebSocketManager) HandleConnection(w http.ResponseWriter, r *http.Request) error {
	// 升級 HTTP 連接為 WebSocket
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return err
	}

	// 為新連接創建鎖
	m.mutex.Lock()
	m.connections[conn] = &sync.Mutex{}
	m.mutex.Unlock()

	// 當連接關閉時清理
	defer func() {
		m.mutex.Lock()
		delete(m.connections, conn)
		m.mutex.Unlock()
		conn.Close()
	}()

	// 監聽消息
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}

	return nil
}

// Broadcast 向所有連接的客戶端廣播消息
func (m *WebSocketManager) Broadcast(message WebSocketMessage) {
	data, err := json.Marshal(message)
	if err != nil {
		Error("Failed to marshal WebSocket message: %v", err)
		return
	}

	m.mutex.RLock()
	for conn, connLock := range m.connections {
		// 為每個連接的寫入操作加鎖
		connLock.Lock()
		err := conn.WriteMessage(websocket.TextMessage, data)
		connLock.Unlock()
		
		if err != nil {
			Error("Failed to send WebSocket message: %v", err)
			continue
		}
	}
	m.mutex.RUnlock()
}

// SendTo 向特定連接發送消息
func (m *WebSocketManager) SendTo(conn *websocket.Conn, message WebSocketMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	m.mutex.RLock()
	connLock, ok := m.connections[conn]
	m.mutex.RUnlock()

	if ok {
		connLock.Lock()
		err = conn.WriteMessage(websocket.TextMessage, data)
		connLock.Unlock()
		return err
	}
	return nil
}
