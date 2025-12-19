package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250401000002CreateConversationsTable struct{}

func (m *M20250401000002CreateConversationsTable) Signature() string {
	return "20250401000002_create_conversations_table"
}

func (m *M20250401000002CreateConversationsTable) Up() error {
	if facades.Schema().HasTable("conversations") {
		return nil
	}

	return facades.Schema().Create("conversations", func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.UnsignedBigInteger("visitor_id").Comment("访客ID")
		table.UnsignedBigInteger("admin_id").Nullable().Comment("客服ID")
		table.String("title", 200).Nullable().Comment("会话标题")
		table.UnsignedTinyInteger("status").Default(1).Comment("状态 1:进行中 2:已结束 3:已关闭")
		table.UnsignedTinyInteger("priority").Default(1).Comment("优先级 1:普通 2:重要 3:紧急")
		table.UnsignedTinyInteger("rating").Default(0).Comment("评分 0:未评分 1-5:评分")
		table.String("rating_note", 500).Nullable().Comment("评分备注")
		table.Timestamp("started_at").Nullable().Comment("开始时间")
		table.Timestamp("ended_at").Nullable().Comment("结束时间")
		table.Timestamp("last_message_at").Nullable().Comment("最后消息时间")
		table.Timestamps()
		table.Index("visitor_id")
		table.Index("admin_id")
		table.Index("status")
		table.Index("last_message_at")
		table.Comment("会话表")
	})
}

func (m *M20250401000002CreateConversationsTable) Down() error {
	return facades.Schema().DropIfExists("conversations")
}

