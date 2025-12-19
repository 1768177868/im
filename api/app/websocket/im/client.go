package im

import (
	"time"

	"github.com/gorilla/websocket"
)

// Client 客户端连接
type Client struct {
	Hub            *IMHub
	Conn           *websocket.Conn
	Send           chan []byte
	ClientType     ClientType
	UserID         uint
	ConversationID uint
}

const (
	// 允许客户端写入的最大时间
	writeWait = 10 * time.Second
	// 允许客户端读取的最大时间
	pongWait = 60 * time.Second
	// 发送 ping 的间隔
	pingPeriod = (pongWait * 9) / 10
	// 最大消息大小
	maxMessageSize = 512 * 1024 // 512KB
)

// readPump 读取消息
func (c *Client) readPump() {
	defer func() {
		c.Hub.unregister <- c
		_ = c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	c.Conn.SetReadLimit(maxMessageSize)
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// 记录错误日志
			}
			break
		}

		// 处理接收到的消息
		// 这里可以添加消息处理逻辑
		_ = message
	}
}

// writePump 写入消息
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// Hub 关闭了通道
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			_, _ = w.Write(message)

			// 添加队列中的其他消息
			n := len(c.Send)
			for i := 0; i < n; i++ {
				_, _ = w.Write([]byte{'\n'})
				_, _ = w.Write(<-c.Send)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// RegisterConnection 注册连接
func (h *IMHub) RegisterConnection(conn *websocket.Conn, clientType ClientType, userID uint, conversationID uint) *Client {
	client := &Client{
		Hub:            h,
		Conn:           conn,
		Send:           make(chan []byte, 256),
		ClientType:     clientType,
		UserID:         userID,
		ConversationID: conversationID,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()

	return client
}

