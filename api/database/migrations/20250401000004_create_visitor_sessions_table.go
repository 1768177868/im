package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250401000004CreateVisitorSessionsTable struct{}

func (m *M20250401000004CreateVisitorSessionsTable) Signature() string {
	return "20250401000004_create_visitor_sessions_table"
}

func (m *M20250401000004CreateVisitorSessionsTable) Up() error {
	if facades.Schema().HasTable("visitor_sessions") {
		return nil
	}

	return facades.Schema().Create("visitor_sessions", func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.UnsignedBigInteger("visitor_id").Comment("访客ID")
		table.String("ip", 50).Nullable().Comment("IP地址")
		table.String("user_agent", 500).Nullable().Comment("用户代理")
		table.String("source", 100).Nullable().Comment("来源页面")
		table.String("referer", 500).Nullable().Comment("来源URL")
		table.String("location", 200).Nullable().Comment("地理位置")
		table.String("device", 50).Nullable().Comment("设备类型")
		table.String("browser", 50).Nullable().Comment("浏览器")
		table.String("os", 50).Nullable().Comment("操作系统")
		table.Timestamp("started_at").Nullable().Comment("开始时间")
		table.Timestamp("ended_at").Nullable().Comment("结束时间")
		table.Integer("duration").Default(0).Comment("持续时间(秒)")
		table.Integer("page_views").Default(0).Comment("页面浏览量")
		table.Timestamps()
		table.Index("visitor_id")
		table.Index("started_at")
		table.Comment("访客会话表")
	})
}

func (m *M20250401000004CreateVisitorSessionsTable) Down() error {
	return facades.Schema().DropIfExists("visitor_sessions")
}

