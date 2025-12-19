package visitor

import (
	"net/http"
	"time"

	apphttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"

	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	imhub "goravel/app/websocket/im"
)

type VisitorController struct {
	customerService services.CustomerService
}

func NewVisitorController() *VisitorController {
	return &VisitorController{
		customerService: services.NewCustomerService(),
	}
}

// Register 访客注册/获取访客信息
func (r *VisitorController) Register(ctx apphttp.Context) apphttp.Response {
	visitorID := ctx.Request().Input("visitor_id", "")
	name := ctx.Request().Input("name", "")
	email := ctx.Request().Input("email", "")
	phone := ctx.Request().Input("phone", "")

	// 获取访客信息
	ip := ctx.Request().Ip()
	userAgent := ctx.Request().Header("User-Agent", "")
	source := ctx.Request().Input("source", "")
	referer := ctx.Request().Header("Referer", "")
	location := ctx.Request().Input("location", "")
	device := ctx.Request().Input("device", "")
	browser := ctx.Request().Input("browser", "")
	os := ctx.Request().Input("os", "")

	visitor, err := r.customerService.CreateOrGetVisitor(
		visitorID, name, email, phone,
		ip, userAgent, source, referer, location, device, browser, os,
	)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "create_visitor_failed")
	}

	return response.Success(ctx, visitor)
}

// CreateConversation 创建会话（如果有进行中的会话则返回现有会话）
func (r *VisitorController) CreateConversation(ctx apphttp.Context) apphttp.Response {
	visitorID := ctx.Request().Input("visitor_id", "")
	if visitorID == "" {
		return response.Error(ctx, http.StatusBadRequest, "visitor_id_required")
	}

	// 查找访客
	var visitor models.Visitor
	if err := facades.Orm().Query().Where("visitor_id", visitorID).First(&visitor); err != nil {
		return response.Error(ctx, http.StatusNotFound, "visitor_not_found")
	}

	// 检查是否有进行中的会话（status=1 表示进行中）
	var existingConversation models.Conversation
	facades.Orm().Query().
		Where("visitor_id", visitor.ID).
		Where("status", 1). // 进行中
		Order("last_message_at DESC").
		First(&existingConversation)

	// 如果存在进行中的会话，直接返回
	if existingConversation.ID > 0 {
		// 加载关联数据
		if existingConversation.VisitorID > 0 {
			facades.Orm().Query().Where("id", existingConversation.VisitorID).First(&existingConversation.Visitor)
		}
		if existingConversation.AdminID != nil && *existingConversation.AdminID > 0 {
			var admin models.Admin
			if err := facades.Orm().Query().Where("id", *existingConversation.AdminID).First(&admin); err == nil {
				existingConversation.Admin = &admin
			}
		}
		return response.Success(ctx, existingConversation)
	}

	// 检查是否有已结束的会话，如果有则重新激活
	var endedConversation models.Conversation
	facades.Orm().Query().
		Where("visitor_id", visitor.ID).
		Where("status", 2). // 已结束
		Order("last_message_at DESC").
		First(&endedConversation)

	title := ctx.Request().Input("title", "新会话")
	adminIDStr := ctx.Request().Input("admin_id", "") // 支持指定客服ID

	var adminID *uint
	var admin *models.Admin

	// 如果指定了客服ID，验证该客服是否存在、启用且具有客服角色
	if adminIDStr != "" {
		adminIDUint := cast.ToUint(adminIDStr)
		if adminIDUint > 0 {
			var specifiedAdmin models.Admin
			if err := facades.Orm().Query().
				Where("id", adminIDUint).
				Where("status", 1).
				First(&specifiedAdmin); err == nil {
				// 检查是否具有客服角色
				var customerServiceRole models.Role
				if err := facades.Orm().Query().
					Where("slug", "customer-service").
					Where("status", 1).
					First(&customerServiceRole); err == nil {
					// 检查管理员是否有这个角色
					count, err := facades.Orm().Query().
						Table("admin_role").
						Where("admin_id", adminIDUint).
						Where("role_id", customerServiceRole.ID).
						Count()
					if err == nil && count > 0 {
						adminID = &adminIDUint
						admin = &specifiedAdmin
					}
				}
			}
		}
	}

	// 如果没有指定客服或指定的客服不存在，自动分配最闲的客服
	if adminID == nil {
		availableAdmin, err := r.customerService.GetAvailableAdmin()
		if err != nil {
			return response.Error(ctx, http.StatusInternalServerError, "get_admin_failed")
		}
		if availableAdmin != nil {
			adminID = &availableAdmin.ID
			admin = availableAdmin
		}
	}

	var conversation *models.Conversation

	// 如果有已结束的会话，重新激活它
	if endedConversation.ID > 0 {
		now := time.Now()
		updateData := map[string]interface{}{
			"status":          1, // 重新激活为进行中
			"last_message_at": now,
		}
		// 如果有新分配的客服，更新客服
		if adminID != nil {
			updateData["admin_id"] = *adminID
		}
		facades.Orm().Query().Model(&models.Conversation{}).Where("id", endedConversation.ID).Update(updateData)

		// 重新查询获取更新后的会话
		facades.Orm().Query().Where("id", endedConversation.ID).First(&endedConversation)
		conversation = &endedConversation

		// 加载关联数据
		if conversation.VisitorID > 0 {
			facades.Orm().Query().Where("id", conversation.VisitorID).First(&conversation.Visitor)
		}
		if conversation.AdminID != nil && *conversation.AdminID > 0 {
			var adminData models.Admin
			if err := facades.Orm().Query().Where("id", *conversation.AdminID).First(&adminData); err == nil {
				conversation.Admin = &adminData
				admin = &adminData
			}
		}

		// 发送系统消息：会话已重新激活
		if admin != nil {
			imhub.Hub().BroadcastSystemMessage(conversation.ID, "reactivated", map[string]interface{}{
				"admin_id":   *conversation.AdminID,
				"admin_name": admin.Nickname,
			})
		}
	} else {
		// 创建新会话
		var err error
		conversation, err = r.customerService.CreateConversation(visitor.ID, adminID, title)
		if err != nil {
			return response.Error(ctx, http.StatusInternalServerError, "create_conversation_failed")
		}

		// 发送系统消息：会话已创建
		if adminID != nil && admin != nil {
			imhub.Hub().BroadcastSystemMessage(conversation.ID, "assigned", map[string]interface{}{
				"admin_id":   *adminID,
				"admin_name": admin.Nickname,
			})
		}
	}

	return response.Success(ctx, conversation)
}

// GetConversations 获取访客的会话列表
func (r *VisitorController) GetConversations(ctx apphttp.Context) apphttp.Response {
	visitorID := ctx.Request().Query("visitor_id", "")
	if visitorID == "" {
		return response.Error(ctx, http.StatusBadRequest, "visitor_id_required")
	}

	// 查找访客
	var visitor models.Visitor
	if err := facades.Orm().Query().Where("visitor_id", visitorID).First(&visitor); err != nil {
		return response.Error(ctx, http.StatusNotFound, "visitor_not_found")
	}

	conversations, err := r.customerService.GetVisitorConversations(visitor.ID)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "get_conversations_failed")
	}

	return response.Success(ctx, conversations)
}

// GetMessages 获取会话消息
func (r *VisitorController) GetMessages(ctx apphttp.Context) apphttp.Response {
	conversationID := ctx.Request().Query("conversation_id", "")
	if conversationID == "" {
		return response.Error(ctx, http.StatusBadRequest, "conversation_id_required")
	}

	convID := cast.ToUint(conversationID)
	page := cast.ToInt(ctx.Request().Query("page", "1"))
	pageSize := cast.ToInt(ctx.Request().Query("page_size", "20"))

	messages, total, err := r.customerService.GetConversationMessages(convID, page, pageSize)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "get_messages_failed")
	}

	return response.Success(ctx, map[string]interface{}{
		"messages":  messages,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// GetAllMessages 获取访客的所有历史消息（跨会话，游标分页）
func (r *VisitorController) GetAllMessages(ctx apphttp.Context) apphttp.Response {
	visitorID := ctx.Request().Query("visitor_id", "")
	if visitorID == "" {
		return response.Error(ctx, http.StatusBadRequest, "visitor_id_required")
	}

	// 查找访客
	var visitor models.Visitor
	if err := facades.Orm().Query().Where("visitor_id", visitorID).First(&visitor); err != nil {
		return response.Error(ctx, http.StatusNotFound, "visitor_not_found")
	}

	// 游标分页参数
	beforeID := cast.ToUint(ctx.Request().Query("before_id", "0")) // 获取比这个ID更早的消息
	limit := cast.ToInt(ctx.Request().Query("limit", "30"))
	if limit <= 0 || limit > 100 {
		limit = 30
	}

	messages, hasMore, err := r.customerService.GetVisitorAllMessages(visitor.ID, beforeID, limit)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "get_messages_failed")
	}

	return response.Success(ctx, map[string]interface{}{
		"messages": messages,
		"has_more": hasMore,
	})
}

// SendMessage 发送消息
func (r *VisitorController) SendMessage(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Input("conversation_id", ""))
	visitorID := ctx.Request().Input("visitor_id", "")
	content := ctx.Request().Input("content", "")
	msgType := ctx.Request().Input("type", "text")

	if conversationID == 0 || visitorID == "" || content == "" {
		return response.Error(ctx, http.StatusBadRequest, "invalid_params")
	}

	// 查找访客
	var visitor models.Visitor
	if err := facades.Orm().Query().Where("visitor_id", visitorID).First(&visitor); err != nil {
		return response.Error(ctx, http.StatusNotFound, "visitor_not_found")
	}

	// 验证会话是否属于该访客
	var conversation models.Conversation
	if err := facades.Orm().Query().Where("id", conversationID).Where("visitor_id", visitor.ID).First(&conversation); err != nil {
		return response.Error(ctx, http.StatusNotFound, "conversation_not_found")
	}

	message, err := r.customerService.SendMessage(
		conversationID,
		"visitor",
		visitor.ID,
		content,
		msgType,
		"",
		"",
		0,
	)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "send_message_failed")
	}

	// 通过 WebSocket 广播消息
	imhub.Hub().BroadcastToConversation(conversationID, &imhub.Message{
		Type:           string(msgType),
		ConversationID: conversationID,
		SenderType:     "visitor",
		SenderID:       visitor.ID,
		Content:        content,
		Timestamp:      time.Now().Unix(),
		MessageID:      message.ID,
	})

	return response.Success(ctx, message)
}

// WebSocket 访客 WebSocket 连接
func (r *VisitorController) WebSocket(ctx apphttp.Context) apphttp.Response {
	visitorID := ctx.Request().Query("visitor_id", "")
	conversationID := cast.ToUint(ctx.Request().Query("conversation_id", "0"))

	if visitorID == "" {
		return response.Error(ctx, http.StatusBadRequest, "visitor_id_required")
	}

	// 查找访客
	var visitor models.Visitor
	if err := facades.Orm().Query().Where("visitor_id", visitorID).First(&visitor); err != nil {
		return response.Error(ctx, http.StatusNotFound, "visitor_not_found")
	}

	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(ctx.Response().Writer(), ctx.Request().Origin(), nil)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "websocket_upgrade_failed")
	}

	// 注册 WebSocket 连接
	imhub.Hub().RegisterConnection(conn, imhub.ClientTypeVisitor, visitor.ID, conversationID)

	// 发送连接成功消息
	imhub.Hub().BroadcastSystemMessage(conversationID, "connected", map[string]interface{}{
		"visitor_id": visitor.ID,
	})

	return nil
}
