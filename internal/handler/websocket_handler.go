package handler

import (
	"emergency-msystem-backend/internal/websocket"

	"github.com/gin-gonic/gin"
)

// WebSocketHandler WebSocket 连接处理
type WebSocketHandler struct {
	hub *websocket.Hub
}

// NewWebSocketHandler 创建 WebSocket 处理器
func NewWebSocketHandler(hub *websocket.Hub) *WebSocketHandler {
	return &WebSocketHandler{hub: hub}
}

// CommandWS 指挥中心 WebSocket
func (h *WebSocketHandler) CommandWS(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		userID = "anonymous"
	}
	role, _ := c.Get("role")
	if role != "admin" {
		role = "admin" // 指挥中心默认admin
	}
	h.hub.WSHandler(c.Writer, c.Request, userID.(string), role.(string))
}

// UserWS 用户端 WebSocket
func (h *WebSocketHandler) UserWS(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		userID = "unknown"
	}
	role, _ := c.Get("role")
	h.hub.WSHandler(c.Writer, c.Request, userID.(string), role.(string))
}
