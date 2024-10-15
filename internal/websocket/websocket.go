package websocket

import (
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// WebSocketManager defines the interface for managing WebSocket connections.
type WebSocketManager interface {
	HandleWebSocket(w http.ResponseWriter, r *http.Request)
	BroadcastMessage(message string)
}

// WebSocketManagerImpl implements the WebSocketManager interface.
type WebSocketManagerImpl struct {
	clients map[*websocket.Conn]bool
	mutex   sync.RWMutex
	logger  *logrus.Entry
	upgrader websocket.Upgrader
}

// NewWebSocketManager creates a new WebSocketManager instance.
func NewWebSocketManager(logger *logrus.Entry) *WebSocketManagerImpl {
	return &WebSocketManagerImpl{
		clients: make(map[*websocket.Conn]bool),
		logger:  logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins; consider restricting in production
			},
		},
	}
}

// HandleWebSocket upgrades the HTTP connection to a WebSocket and manages the client.
func (wm *WebSocketManagerImpl) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		wm.logger.Errorf("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()

	wm.addClient(conn)
	defer wm.removeClient(conn)

	wm.logger.Info("New WebSocket client connected")

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			wm.logger.Infof("WebSocket client disconnected: %v", err)
			break
		}
		// Optionally handle incoming messages from clients here
	}
}

// BroadcastMessage sends a message to all connected WebSocket clients.
func (wm *WebSocketManagerImpl) BroadcastMessage(message string) {
	wm.mutex.RLock()
	defer wm.mutex.RUnlock()

	for client := range wm.clients {
		err := client.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			wm.logger.Errorf("Failed to send message to client: %v", err)
			client.Close()
			delete(wm.clients, client)
		}
	}
}

// addClient adds a new WebSocket client to the manager.
func (wm *WebSocketManagerImpl) addClient(conn *websocket.Conn) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	wm.clients[conn] = true
}

// removeClient removes a WebSocket client from the manager.
func (wm *WebSocketManagerImpl) removeClient(conn *websocket.Conn) {
	wm.mutex.Lock()
	defer wm.mutex.Unlock()
	if _, exists := wm.clients[conn]; exists {
		delete(wm.clients, conn)
		wm.logger.Info("WebSocket client removed")
	}
}