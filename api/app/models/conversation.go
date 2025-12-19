package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// Conversation 会话表
type Conversation struct {
	orm.Model
	VisitorID     uint       `gorm:"index;comment:访客ID" json:"visitor_id"`
	Visitor       Visitor    `gorm:"foreignKey:VisitorID" json:"visitor"`
	AdminID       *uint      `gorm:"index;comment:客服ID" json:"admin_id"`
	Admin         *Admin     `gorm:"foreignKey:AdminID" json:"admin"`
	Title         string     `gorm:"size:200;comment:会话标题" json:"title"`
	Status        uint8      `gorm:"default:1;comment:状态 1:进行中 2:已结束 3:已关闭" json:"status"`
	Priority      uint8      `gorm:"default:1;comment:优先级 1:普通 2:重要 3:紧急" json:"priority"`
	Rating        uint8      `gorm:"default:0;comment:评分 0:未评分 1-5:评分" json:"rating"`
	RatingNote    string     `gorm:"size:500;comment:评分备注" json:"rating_note"`
	StartedAt     *time.Time `gorm:"comment:开始时间" json:"started_at"`
	EndedAt       *time.Time `gorm:"comment:结束时间" json:"ended_at"`
	LastMessageAt *time.Time `gorm:"index;comment:最后消息时间" json:"last_message_at"`
	Messages      []Message  `gorm:"foreignKey:ConversationID" json:"messages"`
}
