package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// Message 消息表
type Message struct {
	orm.Model
	ConversationID uint         `gorm:"index;comment:会话ID" json:"conversation_id"`
	Conversation   Conversation  `gorm:"foreignKey:ConversationID" json:"conversation"`
	SenderType     string       `gorm:"size:20;comment:发送者类型 visitor/admin" json:"sender_type"`
	SenderID       uint         `gorm:"index;comment:发送者ID" json:"sender_id"`
	Content        string       `gorm:"type:text;comment:消息内容" json:"content"`
	Type           string       `gorm:"size:20;default:text;comment:消息类型 text/image/file/location/system" json:"type"`
	FileURL        string       `gorm:"size:500;comment:文件URL" json:"file_url"`
	FileName       string       `gorm:"size:255;comment:文件名" json:"file_name"`
	FileSize       int64        `gorm:"comment:文件大小(字节)" json:"file_size"`
	IsRead         bool         `gorm:"default:0;comment:是否已读" json:"is_read"`
	ReadAt         *time.Time    `gorm:"comment:已读时间" json:"read_at"`
}

