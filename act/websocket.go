package act

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocketManager WebSocket 连接管理器
type WebSocketManager struct {
	upgrader    websocket.Upgrader
	connections map[*websocket.Conn]*Client
	mutex       sync.RWMutex
	keyboard    *Keyboard
	stats       *ConnectionStats
}

// Client 客户端连接信息
type Client struct {
	conn       *websocket.Conn
	send       chan []byte
	lastActive time.Time
	id         string
}

// ConnectionStats 连接统计信息
type ConnectionStats struct {
	ActiveConnections int64     `json:"active_connections"`
	TotalConnections  int64     `json:"total_connections"`
	MessagesReceived  int64     `json:"messages_received"`
	MessagesSent      int64     `json:"messages_sent"`
	LastConnectTime   time.Time `json:"last_connect_time"`
	mutex             sync.RWMutex
}

// WebSocketMessage WebSocket 消息结构
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// TouchpadMoveData 触控板移动数据
type TouchpadMoveData struct {
	DeltaX int     `json:"deltaX"`
	DeltaY int     `json:"deltaY"`
	DPI    float64 `json:"dpi"`
}

// TouchpadClickData 触控板点击数据
type TouchpadClickData struct {
	Button string `json:"button"`
	Type   string `json:"type"`
}

// TouchpadScrollData 触控板滚动数据
type TouchpadScrollData struct {
	DeltaX int `json:"deltaX"`
	DeltaY int `json:"deltaY"`
}

// KeyPressData 按键数据
type KeyPressData struct {
	Key      string `json:"key"`
	Duration int    `json:"duration"`
}

// TypeTextData 文本输入数据
type TypeTextData struct {
	Text string `json:"text"`
}

// NewWebSocketManager 创建新的 WebSocket 管理器
func NewWebSocketManager(keyboard *Keyboard) *WebSocketManager {
	return &WebSocketManager{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// 允许所有来源，生产环境应该限制
				return true
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		connections: make(map[*websocket.Conn]*Client),
		keyboard:    keyboard,
		stats: &ConnectionStats{
			LastConnectTime: time.Now(),
		},
	}
}

// HandleWebSocket 处理 WebSocket 连接
func (wsm *WebSocketManager) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := wsm.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[WEBSOCKET] 升级连接失败: %v", err)
		return
	}

	client := &Client{
		conn:       conn,
		send:       make(chan []byte, 256),
		lastActive: time.Now(),
		id:         generateClientID(),
	}

	wsm.addClient(conn, client)

	log.Printf("[WEBSOCKET] 新客户端连接: %s, 来自: %s", client.id, r.RemoteAddr)

	// 启动读写协程
	go wsm.handleClientWrite(client)
	go wsm.handleClientRead(client)
}

// addClient 添加客户端连接
func (wsm *WebSocketManager) addClient(conn *websocket.Conn, client *Client) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	wsm.connections[conn] = client

	wsm.stats.mutex.Lock()
	wsm.stats.ActiveConnections++
	wsm.stats.TotalConnections++
	wsm.stats.LastConnectTime = time.Now()
	wsm.stats.mutex.Unlock()
}

// removeClient 移除客户端连接
func (wsm *WebSocketManager) removeClient(conn *websocket.Conn) {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	if client, exists := wsm.connections[conn]; exists {
		close(client.send)
		delete(wsm.connections, conn)
		conn.Close()

		wsm.stats.mutex.Lock()
		wsm.stats.ActiveConnections--
		wsm.stats.mutex.Unlock()

		log.Printf("[WEBSOCKET] 客户端断开连接: %s", client.id)
	}
}

// handleClientRead 处理客户端读取
func (wsm *WebSocketManager) handleClientRead(client *Client) {
	defer wsm.removeClient(client.conn)

	client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.conn.SetPongHandler(func(string) error {
		client.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var msg WebSocketMessage
		err := client.conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WEBSOCKET] 读取消息错误: %v", err)
			}
			break
		}

		client.lastActive = time.Now()
		wsm.stats.mutex.Lock()
		wsm.stats.MessagesReceived++
		wsm.stats.mutex.Unlock()

		wsm.handleMessage(client, &msg)
	}
}

// handleClientWrite 处理客户端写入
func (wsm *WebSocketManager) handleClientWrite(client *Client) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.send:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				log.Printf("[WEBSOCKET] 写入消息错误: %v", err)
				return
			}

			wsm.stats.mutex.Lock()
			wsm.stats.MessagesSent++
			wsm.stats.mutex.Unlock()

		case <-ticker.C:
			client.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理接收到的消息
func (wsm *WebSocketManager) handleMessage(client *Client, msg *WebSocketMessage) {
	startTime := time.Now()

	switch msg.Type {
	case "touchpad_move":
		wsm.handleTouchpadMove(client, msg)
	case "touchpad_click":
		wsm.handleTouchpadClick(client, msg)
	case "touchpad_scroll":
		wsm.handleTouchpadScroll(client, msg)
	case "key_press":
		wsm.handleKeyPress(client, msg)
	case "type_text":
		wsm.handleTypeText(client, msg)
	case "ping":
		wsm.sendResponse(client, "pong", nil, msg.RequestID)
	default:
		wsm.sendError(client, "unknown_message_type", msg.RequestID)
	}

	duration := time.Since(startTime)
	log.Printf("[WEBSOCKET] 处理消息 %s 耗时: %v", msg.Type, duration)
}

// handleTouchpadMove 处理触控板移动
func (wsm *WebSocketManager) handleTouchpadMove(client *Client, msg *WebSocketMessage) {
	var data TouchpadMoveData
	if err := wsm.parseMessageData(msg.Data, &data); err != nil {
		wsm.sendError(client, "invalid_touchpad_move_data", msg.RequestID)
		return
	}

	if wsm.keyboard.touchpadDriver == nil {
		wsm.sendError(client, "touchpad_driver_unavailable", msg.RequestID)
		return
	}

	// 应用 DPI 调节
	adjustedDeltaX := int(float64(data.DeltaX) * data.DPI)
	adjustedDeltaY := int(float64(data.DeltaY) * data.DPI)

	err := wsm.keyboard.touchpadDriver.MoveTouchpad(adjustedDeltaX, adjustedDeltaY)
	if err != nil {
		wsm.sendError(client, err.Error(), msg.RequestID)
		return
	}

	// 触控板移动成功，不发送响应以减少网络开销
}

// handleTouchpadClick 处理触控板点击
func (wsm *WebSocketManager) handleTouchpadClick(client *Client, msg *WebSocketMessage) {
	var data TouchpadClickData
	if err := wsm.parseMessageData(msg.Data, &data); err != nil {
		wsm.sendError(client, "invalid_touchpad_click_data", msg.RequestID)
		return
	}

	if wsm.keyboard.touchpadDriver == nil {
		wsm.sendError(client, "touchpad_driver_unavailable", msg.RequestID)
		return
	}

	err := wsm.keyboard.touchpadDriver.ClickTouchpad(data.Button, data.Type)
	if err != nil {
		wsm.sendError(client, err.Error(), msg.RequestID)
		return
	}

	wsm.sendResponse(client, "touchpad_click_success", nil, msg.RequestID)
}

// handleTouchpadScroll 处理触控板滚动
func (wsm *WebSocketManager) handleTouchpadScroll(client *Client, msg *WebSocketMessage) {
	var data TouchpadScrollData
	if err := wsm.parseMessageData(msg.Data, &data); err != nil {
		wsm.sendError(client, "invalid_touchpad_scroll_data", msg.RequestID)
		return
	}

	if wsm.keyboard.touchpadDriver == nil {
		wsm.sendError(client, "touchpad_driver_unavailable", msg.RequestID)
		return
	}

	err := wsm.keyboard.touchpadDriver.ScrollTouchpad(data.DeltaX, data.DeltaY)
	if err != nil {
		wsm.sendError(client, err.Error(), msg.RequestID)
		return
	}

	wsm.sendResponse(client, "touchpad_scroll_success", nil, msg.RequestID)
}

// handleKeyPress 处理按键
func (wsm *WebSocketManager) handleKeyPress(client *Client, msg *WebSocketMessage) {
	var data KeyPressData
	if err := wsm.parseMessageData(msg.Data, &data); err != nil {
		wsm.sendError(client, "invalid_key_press_data", msg.RequestID)
		return
	}

	duration := time.Duration(data.Duration) * time.Millisecond
	err := wsm.keyboard.driver.Press(data.Key, duration)
	if err != nil {
		wsm.sendError(client, err.Error(), msg.RequestID)
		return
	}

	wsm.sendResponse(client, "key_press_success", nil, msg.RequestID)
}

// handleTypeText 处理文本输入
func (wsm *WebSocketManager) handleTypeText(client *Client, msg *WebSocketMessage) {
	var data TypeTextData
	if err := wsm.parseMessageData(msg.Data, &data); err != nil {
		wsm.sendError(client, "invalid_type_text_data", msg.RequestID)
		return
	}

	err := wsm.keyboard.driver.Type(data.Text)
	if err != nil {
		wsm.sendError(client, err.Error(), msg.RequestID)
		return
	}

	wsm.sendResponse(client, "type_text_success", nil, msg.RequestID)
}

// sendResponse 发送响应消息
func (wsm *WebSocketManager) sendResponse(client *Client, msgType string, data interface{}, requestID string) {
	response := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
		RequestID: requestID,
	}

	messageBytes, err := json.Marshal(response)
	if err != nil {
		log.Printf("[WEBSOCKET] 序列化响应失败: %v", err)
		return
	}

	select {
	case client.send <- messageBytes:
	default:
		log.Printf("[WEBSOCKET] 客户端发送缓冲区已满: %s", client.id)
		wsm.removeClient(client.conn)
	}
}

// sendError 发送错误消息
func (wsm *WebSocketManager) sendError(client *Client, errorMsg string, requestID string) {
	wsm.sendResponse(client, "error", map[string]string{"message": errorMsg}, requestID)
}

// parseMessageData 解析消息数据
func (wsm *WebSocketManager) parseMessageData(data interface{}, target interface{}) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(dataBytes, target)
}

// GetStats 获取连接统计信息
func (wsm *WebSocketManager) GetStats() ConnectionStats {
	wsm.stats.mutex.RLock()
	defer wsm.stats.mutex.RUnlock()
	return ConnectionStats{
		ActiveConnections: wsm.stats.ActiveConnections,
		TotalConnections:  wsm.stats.TotalConnections,
		MessagesReceived:  wsm.stats.MessagesReceived,
		MessagesSent:      wsm.stats.MessagesSent,
		LastConnectTime:   wsm.stats.LastConnectTime,
	}
}

// generateClientID 生成客户端ID
func generateClientID() string {
	return time.Now().Format("20060102-150405") + "-" + time.Now().Format("000")
}

// Broadcast 广播消息给所有连接的客户端
func (wsm *WebSocketManager) Broadcast(msgType string, data interface{}) {
	message := WebSocketMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	messageBytes, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WEBSOCKET] 序列化广播消息失败: %v", err)
		return
	}

	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()

	for conn, client := range wsm.connections {
		select {
		case client.send <- messageBytes:
		default:
			log.Printf("[WEBSOCKET] 广播时客户端缓冲区已满: %s", client.id)
			wsm.removeClient(conn)
		}
	}
}

// GetActiveConnections 获取活跃连接数
func (wsm *WebSocketManager) GetActiveConnections() int {
	wsm.mutex.RLock()
	defer wsm.mutex.RUnlock()
	return len(wsm.connections)
}

// Close 关闭所有连接
func (wsm *WebSocketManager) Close() {
	wsm.mutex.Lock()
	defer wsm.mutex.Unlock()

	for conn, client := range wsm.connections {
		close(client.send)
		conn.Close()
	}
	wsm.connections = make(map[*websocket.Conn]*Client)

	wsm.stats.mutex.Lock()
	wsm.stats.ActiveConnections = 0
	wsm.stats.mutex.Unlock()

	log.Printf("[WEBSOCKET] 所有连接已关闭")
}
