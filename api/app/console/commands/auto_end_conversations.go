package commands

import (
	"fmt"
	"time"

	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"

	"goravel/app/models"
	"goravel/app/services"
	imhub "goravel/app/websocket/im"
)

type AutoEndConversations struct {
	customerService services.CustomerService
}

func (r *AutoEndConversations) Signature() string {
	return "app:auto-end-conversations"
}

func (r *AutoEndConversations) Description() string {
	return "自动结束长时间没有访客连接的会话"
}

func (r *AutoEndConversations) Extend() command.Extend {
	return command.Extend{
		Category: "app",
	}
}

func (r *AutoEndConversations) Handle(ctx console.Context) error {
	r.customerService = services.NewCustomerService()

	// 获取所有进行中的会话（status = 1）
	var conversations []models.Conversation
	if err := facades.Orm().Query().
		Where("status", 1).
		Where("visitor_id > ?", 0).
		Find(&conversations); err != nil {
		return err
	}

	now := time.Now()
	// 如果访客没有心跳超过 2 分钟（心跳间隔30秒，2分钟足够判断离线），自动结束会话
	// 使用较短的时间，因为心跳机制更准确
	heartbeatTimeout := 2 * time.Minute
	// 如果会话创建后从未有消息（访客从未打开聊天页），1 分钟后自动结束
	noMessageTimeout := 1 * time.Minute
	endedCount := 0

	for _, conv := range conversations {
		// 检查该会话的访客是否有活跃的 WebSocket 连接
		hasActiveConnection := imhub.Hub().HasVisitorConnection(conv.VisitorID, conv.ID)

		if !hasActiveConnection {
			// 没有活跃连接，检查最后消息时间或会话最后活动时间
			// 优先使用 last_message_at 字段，如果没有则查询最后一条消息
			shouldEnd := false
			var lastActivityTime time.Time
			hasMessages := false

			// 优先使用 last_message_at 字段（现在也用于存储心跳时间）
			if conv.LastMessageAt != nil {
				lastActivityTime = *conv.LastMessageAt
				hasMessages = true
				// 有消息/心跳的情况下，使用 2 分钟超时（心跳间隔30秒，2分钟足够判断离线）
				if now.Sub(lastActivityTime) > heartbeatTimeout {
					shouldEnd = true
				}
			} else {
				// 如果没有 last_message_at，查询最后一条消息
				var lastMessage models.Message
				err := facades.Orm().Query().
					Where("conversation_id", conv.ID).
					Order("id DESC").
					First(&lastMessage)

				hasMessages = err == nil && lastMessage.ID > 0

				if hasMessages && lastMessage.CreatedAt != nil {
					// 有消息，使用最后消息时间
					// carbon.DateTime 通过 Format 然后解析为 time.Time
					timeStr := lastMessage.CreatedAt.Format("2006-01-02 15:04:05")
					lastActivityTime, parseErr := time.Parse("2006-01-02 15:04:05", timeStr)
					if parseErr != nil {
						continue
					}
					// 有消息的情况下，使用 2 分钟超时（心跳间隔30秒，2分钟足够判断离线）
					if now.Sub(lastActivityTime) > heartbeatTimeout {
						shouldEnd = true
					}
				} else if conv.CreatedAt != nil {
					// 没有消息，使用会话创建时间
					timeStr := conv.CreatedAt.Format("2006-01-02 15:04:05")
					lastActivityTime, parseErr := time.Parse("2006-01-02 15:04:05", timeStr)
					if parseErr != nil {
						continue
					}
					// 没有消息的情况下，使用 1 分钟超时（访客可能从未打开聊天页）
					if now.Sub(lastActivityTime) > noMessageTimeout {
						shouldEnd = true
					}
				} else {
					// 如果都没有时间，跳过这个会话
					continue
				}
			}

			if shouldEnd {
				// 结束会话
				if err := r.customerService.EndConversation(conv.ID); err != nil {
					facades.Log().Errorf("自动结束会话失败: conversation_id=%d, error=%v", conv.ID, err)
					continue
				}

				// 发送系统消息：会话已结束
				imhub.Hub().BroadcastSystemMessage(conv.ID, "ended", nil)

				endedCount++
				facades.Log().Infof("自动结束会话: conversation_id=%d, visitor_id=%d, has_messages=%v, last_activity=%v", conv.ID, conv.VisitorID, hasMessages, lastActivityTime)
			}
		}
	}

	if endedCount > 0 {
		ctx.Info(fmt.Sprintf("自动结束了 %d 个会话", endedCount))
	} else {
		ctx.Info("没有需要结束的会话")
	}

	return nil
}
