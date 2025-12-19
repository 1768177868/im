package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250401000001CreateVisitorsTable struct{}

func (m *M20250401000001CreateVisitorsTable) Signature() string {
	return "20250401000001_create_visitors_table"
}

func (m *M20250401000001CreateVisitorsTable) Up() error {
	if facades.Schema().HasTable("visitors") {
		return nil
	}

	return facades.Schema().Create("visitors", func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.String("visitor_id", 100).Comment("访客唯一标识")
		table.String("name", 100).Nullable().Comment("访客姓名")
		table.String("email", 100).Nullable().Comment("邮箱")
		table.String("phone", 20).Nullable().Comment("手机号")
		table.String("avatar", 255).Nullable().Comment("头像")
		table.String("ip", 50).Nullable().Comment("IP地址")
		table.String("user_agent", 500).Nullable().Comment("用户代理")
		table.String("source", 100).Nullable().Comment("来源页面")
		table.String("referer", 500).Nullable().Comment("来源URL")
		table.String("location", 200).Nullable().Comment("地理位置")
		table.String("device", 50).Nullable().Comment("设备类型")
		table.String("browser", 50).Nullable().Comment("浏览器")
		table.String("os", 50).Nullable().Comment("操作系统")
		table.UnsignedTinyInteger("status").Default(1).Comment("状态 1:在线 0:离线")
		table.Timestamp("last_active_at").Nullable().Comment("最后活跃时间")
		table.Timestamps()
		table.SoftDeletes()
		table.Unique("visitor_id", "deleted_at")
		table.Index("status")
		table.Comment("访客表")
	})
}

func (m *M20250401000001CreateVisitorsTable) Down() error {
	return facades.Schema().DropIfExists("visitors")
}

