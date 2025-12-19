package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250401000003CreateMessagesTable struct{}

func (m *M20250401000003CreateMessagesTable) Signature() string {
	return "20250401000003_create_messages_table"
}

func (m *M20250401000003CreateMessagesTable) Up() error {
	if facades.Schema().HasTable("messages") {
		return nil
	}

	return facades.Schema().Create("messages", func(table schema.Blueprint) {
		table.BigIncrements("id")
		table.UnsignedBigInteger("conversation_id").Comment("会话ID")
		table.String("sender_type", 20).Comment("发送者类型 visitor/admin")
		table.UnsignedBigInteger("sender_id").Comment("发送者ID")
		table.Text("content").Comment("消息内容")
		table.String("type", 20).Default("text").Comment("消息类型 text/image/file/location/system")
		table.String("file_url", 500).Nullable().Comment("文件URL")
		table.String("file_name", 255).Nullable().Comment("文件名")
		table.BigInteger("file_size").Default(0).Nullable().Comment("文件大小(字节)")
		table.Boolean("is_read").Default(false).Comment("是否已读")
		table.Timestamp("read_at").Nullable().Comment("已读时间")
		table.Timestamps()
		table.Index("conversation_id")
		table.Index("sender_type")
		table.Index("sender_id")
		table.Index("is_read")
		table.Comment("消息表")
	})
}

func (m *M20250401000003CreateMessagesTable) Down() error {
	return facades.Schema().DropIfExists("messages")
}

