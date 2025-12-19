package models

import (
	"time"

	"github.com/goravel/framework/database/orm"
)

// Visitor 访客表
type Visitor struct {
	orm.Model
	VisitorID   string `gorm:"uniqueIndex;size:100;comment:访客唯一标识" json:"visitor_id"`
	Name        string `gorm:"size:100;comment:访客姓名" json:"name"`
	Email       string `gorm:"size:100;comment:邮箱" json:"email"`
	Phone       string `gorm:"size:20;comment:手机号" json:"phone"`
	Avatar      string `gorm:"size:255;comment:头像" json:"avatar"`
	IP          string `gorm:"size:50;comment:IP地址" json:"ip"`
	UserAgent   string `gorm:"size:500;comment:用户代理" json:"user_agent"`
	Source      string `gorm:"size:100;comment:来源页面" json:"source"`
	Referer     string `gorm:"size:500;comment:来源URL" json:"referer"`
	Location    string `gorm:"size:200;comment:地理位置" json:"location"`
	Device      string `gorm:"size:50;comment:设备类型" json:"device"`
	Browser     string `gorm:"size:50;comment:浏览器" json:"browser"`
	OS          string `gorm:"size:50;comment:操作系统" json:"os"`
	Status      uint8  `gorm:"default:1;comment:状态 1:在线 0:离线" json:"status"`
	LastActiveAt *time.Time `gorm:"comment:最后活跃时间" json:"last_active_at"`
	orm.SoftDeletes
}

