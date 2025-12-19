package im

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/goravel/framework/facades"
)

// MessageType 消息类型
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeFile     MessageType = "file"
	MessageTypeLocation MessageType = "location"
	MessageTypeSystem   MessageType = "system"
)

// ClientType 客户端类型
type ClientType string

const (
	ClientTypeVisitor ClientType = "visitor"
	ClientTypeAdmin   ClientType = "admin"
)

// Message 消息结构
type Message struct {
	Type           string `json:"type"`            // 消息类型: text/image/file/location/system
	ConversationID uint   `json:"conversation_id"` // 会话ID
	SenderType     string `json:"sender_type"`     // 发送者类型: visitor/admin
	SenderID       uint   `json:"sender_id"`       // 发送者ID
	Content        string `json:"content"`         // 消息内容
	FileURL        string `json:"file_url,omitempty"`
	FileName       string `json:"file_name,omitempty"`
	FileSize       int64  `json:"file_size,omitempty"`
	Timestamp      int64  `json:"timestamp"`
	MessageID      uint   `json:"message_id,omitempty"`
}

// SystemMessage 系统消息
type SystemMessage struct {
	Type      string      `json:"type"`  // 消息类型: system
	Event     string      `json:"event"` // 事件类型: connected/disconnected/typing/read/assigned
	Data      interface{} `json:"data"`  // 事件数据
	Timestamp int64       `json:"timestamp"`
}

// IMHub IM Hub 管理所有连接
type IMHub struct {
	// 访客连接: visitorID -> []*client
	visitorClients map[uint]map[*Client]bool
	// 客服连接: adminID -> []*client
	adminClients map[uint]map[*Client]bool
	// 会话连接: conversationID -> []*client
	conversationClients map[uint]map[*Client]bool
	// 注册通道
	register chan *Client
	// 注销通道
	unregister chan *Client
	// 广播消息通道
	broadcast chan *BroadcastMessage
	// 互斥锁
	mu sync.RWMutex
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	ConversationID uint
	Message        *Message
	SystemMessage  *SystemMessage
	TargetClients  []*Client // 如果指定，只发送给这些客户端
}

var hubInstance *IMHub

func init() {
	hubInstance = newIMHub()
	go hubInstance.run()
}

// Hub 获取 Hub 实例
func Hub() *IMHub {
	return hubInstance
}

func newIMHub() *IMHub {
	return &IMHub{
		visitorClients:      make(map[uint]map[*Client]bool),
		adminClients:        make(map[uint]map[*Client]bool),
		conversationClients: make(map[uint]map[*Client]bool),
		register:            make(chan *Client),
		unregister:          make(chan *Client),
		broadcast:           make(chan *BroadcastMessage, 100),
	}
}

// run 运行 Hub
func (h *IMHub) run() {
	for {
		select {
		case client := <-h.register:
			h.addClient(client)
		case client := <-h.unregister:
			h.removeClient(client)
		case broadcast := <-h.broadcast:
			h.dispatch(broadcast)
		}
	}
}

// addClient 添加客户端
func (h *IMHub) addClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 根据客户端类型添加到对应映射
	if client.ClientType == ClientTypeVisitor {
		if _, ok := h.visitorClients[client.UserID]; !ok {
			h.visitorClients[client.UserID] = make(map[*Client]bool)
		}
		h.visitorClients[client.UserID][client] = true
	} else if client.ClientType == ClientTypeAdmin {
		if _, ok := h.adminClients[client.UserID]; !ok {
			h.adminClients[client.UserID] = make(map[*Client]bool)
		}
		h.adminClients[client.UserID][client] = true
	}

	// 添加到会话映射
	if client.ConversationID > 0 {
		if _, ok := h.conversationClients[client.ConversationID]; !ok {
			h.conversationClients[client.ConversationID] = make(map[*Client]bool)
		}
		h.conversationClients[client.ConversationID][client] = true
	}

	facades.Log().Infof("Client registered: type=%s, userID=%d, conversationID=%d", client.ClientType, client.UserID, client.ConversationID)
}

// removeClient 移除客户端
func (h *IMHub) removeClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// 从访客映射中移除
	if client.ClientType == ClientTypeVisitor {
		if visitorClients, ok := h.visitorClients[client.UserID]; ok {
			if _, exists := visitorClients[client]; exists {
				delete(visitorClients, client)
				close(client.Send)
			}
			if len(visitorClients) == 0 {
				delete(h.visitorClients, client.UserID)
			}
		}
	} else if client.ClientType == ClientTypeAdmin {
		// 从客服映射中移除
		if adminClients, ok := h.adminClients[client.UserID]; ok {
			if _, exists := adminClients[client]; exists {
				delete(adminClients, client)
				close(client.Send)
			}
			if len(adminClients) == 0 {
				delete(h.adminClients, client.UserID)
			}
		}
	}

	// 从会话映射中移除
	if client.ConversationID > 0 {
		if conversationClients, ok := h.conversationClients[client.ConversationID]; ok {
			delete(conversationClients, client)
			if len(conversationClients) == 0 {
				delete(h.conversationClients, client.ConversationID)
			}
		}
	}

	facades.Log().Infof("Client unregistered: type=%s, userID=%d, conversationID=%d", client.ClientType, client.UserID, client.ConversationID)
}

// dispatch 分发消息
func (h *IMHub) dispatch(broadcast *BroadcastMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	var targetClients map[*Client]bool

	// 如果指定了目标客户端，使用指定的
	if len(broadcast.TargetClients) > 0 {
		targetClients = make(map[*Client]bool)
		for _, client := range broadcast.TargetClients {
			targetClients[client] = true
		}
	} else {
		// 否则发送给会话中的所有客户端
		if clients, ok := h.conversationClients[broadcast.ConversationID]; ok {
			targetClients = clients
		}
	}

	if len(targetClients) == 0 {
		return
	}

	var data []byte
	var err error

	if broadcast.Message != nil {
		data, err = json.Marshal(broadcast.Message)
	} else if broadcast.SystemMessage != nil {
		data, err = json.Marshal(broadcast.SystemMessage)
	}

	if err != nil {
		facades.Log().Errorf("Failed to marshal message: %v", err)
		return
	}

	// 发送给所有目标客户端
	for client := range targetClients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			h.removeClient(client)
		}
	}
}

// BroadcastToConversation 广播消息到会话
func (h *IMHub) BroadcastToConversation(conversationID uint, message *Message) {
	h.broadcast <- &BroadcastMessage{
		ConversationID: conversationID,
		Message:        message,
	}
}

// BroadcastSystemMessage 广播系统消息
func (h *IMHub) BroadcastSystemMessage(conversationID uint, event string, data interface{}) {
	h.broadcast <- &BroadcastMessage{
		ConversationID: conversationID,
		SystemMessage: &SystemMessage{
			Type:      "system",
			Event:     event,
			Data:      data,
			Timestamp: time.Now().Unix(),
		},
	}
}

// GetOnlineAdmins 获取在线客服列表
func (h *IMHub) GetOnlineAdmins() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	adminIDs := make([]uint, 0, len(h.adminClients))
	for adminID := range h.adminClients {
		adminIDs = append(adminIDs, adminID)
	}
	return adminIDs
}

// GetOnlineVisitors 获取在线访客列表
func (h *IMHub) GetOnlineVisitors() []uint {
	h.mu.RLock()
	defer h.mu.RUnlock()

	visitorIDs := make([]uint, 0, len(h.visitorClients))
	for visitorID := range h.visitorClients {
		visitorIDs = append(visitorIDs, visitorID)
	}
	return visitorIDs
}

// IsAdminOnline 检查客服是否在线
func (h *IMHub) IsAdminOnline(adminID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.adminClients[adminID]
	return ok && len(clients) > 0
}

// IsVisitorOnline 检查访客是否在线
func (h *IMHub) IsVisitorOnline(visitorID uint) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients, ok := h.visitorClients[visitorID]
	return ok && len(clients) > 0
}

// GetAdminClients 获取客服的所有连接
func (h *IMHub) GetAdminClients(adminID uint) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0)
	if adminClients, ok := h.adminClients[adminID]; ok {
		for client := range adminClients {
			clients = append(clients, client)
		}
	}
	return clients
}

// GetVisitorClients 获取访客的所有连接
func (h *IMHub) GetVisitorClients(visitorID uint) []*Client {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clients := make([]*Client, 0)
	if visitorClients, ok := h.visitorClients[visitorID]; ok {
		for client := range visitorClients {
			clients = append(clients, client)
		}
	}
	return clients
}
