package websocket

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// 消息类型
const (
	MsgTypeNewHelpRequest  = "new_help_request"
	MsgTypeMatchUpdate     = "match_update"
	MsgTypeGPSUpdate       = "gps_update"
	MsgTypeAlert           = "alert"
	MsgTypeEscalation      = "escalation"
	MsgTypeSystemBroadcast = "system_broadcast"
	MsgTypePing            = "ping"
	MsgTypePong            = "pong"
)

// WSMessage WebSocket 消息
type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
}

// Client WebSocket 客户端
type Client struct {
	ID     string
	UserID string
	Role   string           // user / volunteer / admin
	Hub    *Hub
	Conn   *websocket.Conn
	Send   chan []byte
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true // 生产环境应限制 Origin
	},
}

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 65536
)

// ReadPump 读取消息
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, msgBytes, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WS] Read error: %v", err)
			}
			break
		}

		// 处理消息（目前只处理 ping）
		var msg WSMessage
		if err := json.Unmarshal(msgBytes, &msg); err == nil && msg.Type == MsgTypePing {
			// 客户端心跳
		}
	}
}

// WritePump 写入消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)
			w.Close()

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// --- Hub: WebSocket 连接管理中心 ---

// Hub 管理所有 WebSocket 连接
type Hub struct {
	mu         sync.RWMutex
	// 用户连接分组
	userClients    map[string]*Client  // user_id -> client (Mobile端)
	commandClients map[string]*Client  // client_id -> client (PC指挥中心)
	// 频道
	channels map[string]chan []byte
	// 注册/注销
	Register   chan *Client
	Unregister chan *Client
	// 全局广播
	Broadcast chan []byte
}

// NewHub 创建 Hub
func NewHub() *Hub {
	return &Hub{
		userClients:    make(map[string]*Client),
		commandClients: make(map[string]*Client),
		channels:       make(map[string]chan []byte),
		Register:       make(chan *Client, 256),
		Unregister:     make(chan *Client, 256),
		Broadcast:      make(chan []byte, 512),
	}
}

// Run 启动 Hub
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if client.Role == "admin" {
				h.commandClients[client.ID] = client
				log.Printf("[WS] Command client connected: %s", client.ID[:8])
			} else {
				h.userClients[client.UserID] = client
				log.Printf("[WS] User client connected: %s (user=%s)", client.ID[:8], client.UserID[:8])
			}
			h.mu.Unlock()

		case client := <-h.Unregister:
			h.mu.Lock()
			if client.Role == "admin" {
				if _, ok := h.commandClients[client.ID]; ok {
					delete(h.commandClients, client.ID)
					close(client.Send)
				}
			} else {
				if c, ok := h.userClients[client.UserID]; ok && c.ID == client.ID {
					delete(h.userClients, client.UserID)
					close(client.Send)
				}
			}
			h.mu.Unlock()
			log.Printf("[WS] Client disconnected: %s", client.ID[:8])

		case message := <-h.Broadcast:
			h.mu.RLock()
			for _, client := range h.commandClients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.commandClients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// SendToCommand 发送消息给所有指挥中心客户端
func (h *Hub) SendToCommand(msgType string, data interface{}) {
	h.sendMessage(h.commandClients, msgType, data)
}

// SendToUser 发送消息给指定用户
func (h *Hub) SendToUser(userID, msgType string, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	client, exists := h.userClients[userID]
	if !exists {
		return
	}

	msg := WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	select {
	case client.Send <- msgBytes:
	default:
	}
}

// SendToAll 广播给所有用户
func (h *Hub) SendToAll(msgType string, data interface{}) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	msg := WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	msgBytes, _ := json.Marshal(msg)

	// 发送给所有用户
	for _, client := range h.userClients {
		select {
		case client.Send <- msgBytes:
		default:
		}
	}

	// 发送给指挥中心
	for _, client := range h.commandClients {
		select {
		case client.Send <- msgBytes:
		default:
		}
	}
}

// sendMessage 内部发送消息
func (h *Hub) sendMessage(clients map[string]*Client, msgType string, data interface{}) {
	msg := WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now().UnixMilli(),
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return
	}

	for _, client := range clients {
		select {
		case client.Send <- msgBytes:
		default:
		}
	}
}

// WSHandler WebSocket 升级处理
func (h *Hub) WSHandler(w http.ResponseWriter, r *http.Request, userID, role string) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WS] Upgrade error: %v", err)
		return
	}

	client := &Client{
		ID:     userID + "-" + randomString(8),
		UserID: userID,
		Role:   role,
		Hub:    h,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.Register <- client

	go client.WritePump()
	go client.ReadPump()
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
		time.Sleep(1) // 简单去重
	}
	return string(b)
}
