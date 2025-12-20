package migrations

import (
	"github.com/goravel/framework/contracts/database/schema"
	"github.com/goravel/framework/facades"
)

type M20250401000004AddMessagesIndexes struct{}

func (m *M20250401000004AddMessagesIndexes) Signature() string {
	return "20250401000004_add_messages_indexes"
}

func (m *M20250401000004AddMessagesIndexes) Up() error {
	if !facades.Schema().HasTable("messages") {
		return nil
	}

	// 检查索引是否已存在，避免重复创建
	indexes, err := facades.Schema().GetIndexes("messages")
	hasConversationCreatedIndex := false
	hasCreatedAtIndex := false
	
	if err == nil {
		for _, idx := range indexes {
			if idx.Name == "idx_messages_conversation_created" {
				hasConversationCreatedIndex = true
			}
			if idx.Name == "idx_messages_created_at" {
				hasCreatedAtIndex = true
			}
		}
	}

	// 添加联合索引优化消息查询性能
	// conversation_id + created_at 联合索引：用于按会话和时间查询消息（最常用）
	// 这个索引可以大幅提升分页查询性能，特别是当消息表数据量很大时
	
	// 使用原生 SQL 创建联合索引（Goravel 的 Index 方法不支持多列）
	if !hasConversationCreatedIndex {
		// 根据数据库类型选择不同的 SQL
		driverName := facades.Orm().Query().Driver()
		var sql string
		if driverName == "mysql" {
			sql = "CREATE INDEX idx_messages_conversation_created ON messages(conversation_id, created_at)"
		} else if driverName == "postgres" {
			sql = "CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at)"
		} else {
			// SQLite 或其他数据库
			sql = "CREATE INDEX IF NOT EXISTS idx_messages_conversation_created ON messages(conversation_id, created_at)"
		}
		// 使用框架的 Exec 方法执行原生 SQL
		_, execErr := facades.Orm().Query().Exec(sql)
		if execErr != nil {
			// 如果索引已存在，忽略错误（MySQL 会报错，PostgreSQL/SQLite 使用 IF NOT EXISTS）
			facades.Log().Debugf("创建联合索引（可能已存在）: %v", execErr)
		}
	}

	// 添加 created_at 单独索引，用于时间范围查询
	if !hasCreatedAtIndex {
		// 使用原生 SQL 创建索引
		driverName := facades.Orm().Query().Driver()
		var sql string
		if driverName == "mysql" {
			sql = "CREATE INDEX idx_messages_created_at ON messages(created_at)"
		} else if driverName == "postgres" {
			sql = "CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)"
		} else {
			// SQLite 或其他数据库
			sql = "CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at)"
		}
		_, execErr := facades.Orm().Query().Exec(sql)
		if execErr != nil {
			// 如果索引已存在，忽略错误
			facades.Log().Debugf("创建 created_at 索引（可能已存在）: %v", execErr)
		}
	}

	return nil
}

func (m *M20250401000004AddMessagesIndexes) Down() error {
	return facades.Schema().Table("messages", func(table schema.Blueprint) {
		table.DropIndex("idx_messages_conversation_created")
		table.DropIndex("idx_messages_created_at")
	})
}

