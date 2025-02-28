package utils

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"net/http"
	"sync"
)

// WebSocketManager 管理 WebSocket 連接
type WebSocketManager struct {
	// 升級器用於將 HTTP 連接升級為 WebSocket
	upgrader websocket.Upgrader
	// 存儲所有活動的 WebSocket 連接
	connections map[*websocket.Conn]*sync.Mutex
	// 互斥鎖保護連接映射
	mutex sync.RWMutex
}

// WebSocketMessage WebSocket 消息結構
type WebSocketMessage struct {
	Status  string      `json:"status"`            // 狀態：success 或 error
	Message string      `json:"message,omitempty"` // 消息內容
	Data    interface{} `json:"data,omitempty"`    // 數據
	Command string      `json:"command,omitempty"` // 命令參數
}

// NewWebSocketManager 創建新的 WebSocket 管理器
func NewWebSocketManager() *WebSocketManager {
	return &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允許所有來源
			},
		},
		connections: make(map[*websocket.Conn]*sync.Mutex),
	}
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
