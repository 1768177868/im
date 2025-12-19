package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// VisitorSession 访客会话表（记录访客访问信息）
type VisitorSession struct {
	orm.Model
	VisitorID   uint      `gorm:"index;comment:访客ID" json:"visitor_id"`
	Visitor     Visitor   `gorm:"foreignKey:VisitorID" json:"visitor"`
	IP          string    `gorm:"size:50;comment:IP地址" json:"ip"`
	UserAgent   string    `gorm:"size:500;comment:用户代理" json:"user_agent"`
	Source      string    `gorm:"size:100;comment:来源页面" json:"source"`
	Referer     string    `gorm:"size:500;comment:来源URL" json:"referer"`
	Location    string    `gorm:"size:200;comment:地理位置" json:"location"`
	Device      string    `gorm:"size:50;comment:设备类型" json:"device"`
	Browser     string    `gorm:"size:50;comment:浏览器" json:"browser"`
	OS          string    `gorm:"size:50;comment:操作系统" json:"os"`
	StartedAt   *time.Time `gorm:"comment:开始时间" json:"started_at"`
	EndedAt     *time.Time `gorm:"comment:结束时间" json:"ended_at"`
	Duration    int       `gorm:"default:0;comment:持续时间(秒)" json:"duration"`
	PageViews   int       `gorm:"default:0;comment:页面浏览量" json:"page_views"`
}

