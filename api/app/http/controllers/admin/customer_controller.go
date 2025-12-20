package admin

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

type CustomerController struct {
	customerService services.CustomerService
}

func NewCustomerController() *CustomerController {
	return &CustomerController{
		customerService: services.NewCustomerService(),
	}
}

// currentAdmin 从 context 中获取当前管理员
func (r *CustomerController) currentAdmin(ctx apphttp.Context) *models.Admin {
	if adminValue := ctx.Value("admin"); adminValue != nil {
		if admin, ok := adminValue.(models.Admin); ok {
			return &admin
		}
		if adminPtr, ok := adminValue.(*models.Admin); ok {
			return adminPtr
		}
	}
	return nil
}

// GetConversations 获取客服的会话列表（支持搜索）
func (r *CustomerController) GetConversations(ctx apphttp.Context) apphttp.Response {
	adminID := ctx.Request().Query("admin_id", "")
	status := ctx.Request().Query("status", "")
	conversationIDStr := ctx.Request().Query("conversation_id", "")
	visitorName := ctx.Request().Query("visitor_name", "")
	page := cast.ToInt(ctx.Request().Query("page", "1"))
	pageSize := cast.ToInt(ctx.Request().Query("page_size", "10"))

	// 从 JWT 中获取管理员ID（用于权限验证）
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, "not_logged_in")
	}

	var conversations []models.Conversation
	var total int64
	var err error

	// 解析搜索参数
	var statusUint *uint8
	if status != "" {
		s := cast.ToUint8(status)
		statusUint = &s
	}

	var conversationIDUint *uint
	if conversationIDStr != "" {
		id := cast.ToUint(conversationIDStr)
		if id > 0 {
			conversationIDUint = &id
		}
	}

	// 如果指定了 admin_id，查询指定客服的会话；否则查询所有会话
	if adminID != "" {
		adminIDUint := cast.ToUint(adminID)
		conversations, total, err = r.customerService.GetAdminConversationsPaginated(adminIDUint, statusUint, page, pageSize)
	} else {
		// 查询所有会话（支持搜索）
		conversations, total, err = r.customerService.GetAllConversationsPaginated(statusUint, conversationIDUint, visitorName, page, pageSize)
	}

	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "get_conversations_failed")
	}

	// 为每个会话添加访客实时在线状态
	type conversationWithVisitorStatus struct {
		models.Conversation
		VisitorOnlineStatus string `json:"visitor_online_status"`
	}

	result := make([]conversationWithVisitorStatus, len(conversations))
	for i, conv := range conversations {
		status := "offline"
		if conv.VisitorID > 0 && imhub.Hub().IsVisitorOnline(conv.VisitorID) {
			if conv.Visitor.Status == 2 {
				status = "away"
			} else {
				status = "online"
			}
		}
		result[i] = conversationWithVisitorStatus{
			Conversation:        conv,
			VisitorOnlineStatus: status,
		}
	}

	return response.Paginate(ctx, result, total, page, pageSize)
}

// GetConversationDetail 获取会话详情
func (r *CustomerController) GetConversationDetail(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Route("id"))
	if conversationID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "conversation_id_required")
	}

	var conversation models.Conversation
	if err := facades.Orm().Query().
		Where("id", conversationID).
		First(&conversation); err != nil {
		return response.Error(ctx, http.StatusNotFound, "conversation_not_found")
	}

	// 手动加载关联数据
	if conversation.VisitorID > 0 {
		facades.Orm().Query().Where("id", conversation.VisitorID).First(&conversation.Visitor)
	}
	if conversation.AdminID != nil && *conversation.AdminID > 0 {
		var admin models.Admin
		if err := facades.Orm().Query().Where("id", *conversation.AdminID).First(&admin); err == nil {
			conversation.Admin = &admin
		}
	}
	facades.Orm().Query().Where("conversation_id", conversationID).Order("id ASC").Find(&conversation.Messages)

	// 检查访客是否有活跃的 WebSocket 连接（实时在线状态）
	visitorOnlineStatus := "offline"
	if conversation.VisitorID > 0 {
		if imhub.Hub().IsVisitorOnline(conversation.VisitorID) {
			// 有 WebSocket 连接，检查数据库中的状态判断是在线还是离开
			if conversation.Visitor.Status == 2 {
				visitorOnlineStatus = "away"
			} else {
				visitorOnlineStatus = "online"
			}
		}
	}

	return response.Success(ctx, map[string]interface{}{
		"id":                    conversation.ID,
		"visitor_id":            conversation.VisitorID,
		"admin_id":              conversation.AdminID,
		"title":                 conversation.Title,
		"status":                conversation.Status,
		"last_message_at":       conversation.LastMessageAt,
		"created_at":            conversation.CreatedAt,
		"updated_at":            conversation.UpdatedAt,
		"visitor":               conversation.Visitor,
		"admin":                 conversation.Admin,
		"messages":              conversation.Messages,
		"visitor_online_status": visitorOnlineStatus,
	})
}

// GetMessages 获取会话消息
// 支持两种模式：
// 1. 传 conversation_id: 获取单个会话的消息（传统分页）
// 2. 传 visitor_id: 获取访客的所有历史消息（游标分页）
func (r *CustomerController) GetMessages(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Query("conversation_id", ""))
	visitorID := cast.ToUint(ctx.Request().Query("visitor_id", ""))

	if visitorID > 0 {
		// 游标分页：获取访客的所有历史消息
		beforeID := cast.ToUint(ctx.Request().Query("before_id", "0"))
		limit := cast.ToInt(ctx.Request().Query("limit", "30"))
		if limit <= 0 || limit > 100 {
			limit = 30
		}

		messages, hasMore, err := r.customerService.GetVisitorAllMessages(visitorID, beforeID, limit)
		if err != nil {
			return response.Error(ctx, http.StatusInternalServerError, "get_messages_failed")
		}

		return response.Success(ctx, map[string]interface{}{
			"messages": messages,
			"has_more": hasMore,
		})
	} else if conversationID > 0 {
		// 传统分页：获取单个会话的消息
		page := cast.ToInt(ctx.Request().Query("page", "1"))
		pageSize := cast.ToInt(ctx.Request().Query("page_size", "100"))

		messages, total, err := r.customerService.GetConversationMessages(conversationID, page, pageSize)
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

	return response.Error(ctx, http.StatusBadRequest, "conversation_id_or_visitor_id_required")
}

// SendMessage 发送消息
func (r *CustomerController) SendMessage(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Input("conversation_id", ""))
	content := ctx.Request().Input("content", "")
	msgType := ctx.Request().Input("type", "text")
	fileURL := ctx.Request().Input("file_url", "")
	fileName := ctx.Request().Input("file_name", "")
	fileSize := cast.ToInt64(ctx.Request().Input("file_size", "0"))

	// 文本消息必须有内容，文件/图片消息必须有 file_url
	if conversationID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "invalid_params")
	}
	if msgType == "text" && content == "" {
		return response.Error(ctx, http.StatusBadRequest, "content_required")
	}
	if (msgType == "image" || msgType == "file") && fileURL == "" {
		return response.Error(ctx, http.StatusBadRequest, "file_url_required")
	}

	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, "not_logged_in")
	}

	message, err := r.customerService.SendMessage(
		conversationID,
		"admin",
		admin.ID,
		content,
		msgType,
		fileURL,
		fileName,
		fileSize,
	)
	if err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "send_message_failed")
	}

	// 通过 WebSocket 广播消息
	imhub.Hub().BroadcastToConversation(conversationID, &imhub.Message{
		Type:           msgType,
		ConversationID: conversationID,
		SenderType:     "admin",
		SenderID:       admin.ID,
		Content:        content,
		FileURL:        fileURL,
		FileName:       fileName,
		FileSize:       fileSize,
		Timestamp:      time.Now().Unix(),
		MessageID:      message.ID,
	})

	return response.Success(ctx, message)
}

// AssignConversation 分配会话给客服
func (r *CustomerController) AssignConversation(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Input("conversation_id", ""))
	adminID := cast.ToUint(ctx.Request().Input("admin_id", ""))

	if conversationID == 0 || adminID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "invalid_params")
	}

	if err := r.customerService.AssignConversation(conversationID, adminID); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "assign_conversation_failed")
	}

	// 发送系统消息：会话已分配
	var admin models.Admin
	if err := facades.Orm().Query().Where("id", adminID).First(&admin); err == nil {
		imhub.Hub().BroadcastSystemMessage(conversationID, "assigned", map[string]interface{}{
			"admin_id":   adminID,
			"admin_name": admin.Nickname,
		})
	}

	return response.Success(ctx, nil)
}

// EndConversation 结束会话
func (r *CustomerController) EndConversation(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Input("conversation_id", ""))
	if conversationID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "conversation_id_required")
	}

	if err := r.customerService.EndConversation(conversationID); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "end_conversation_failed")
	}

	// 发送系统消息：会话已结束
	imhub.Hub().BroadcastSystemMessage(conversationID, "ended", nil)

	return response.Success(ctx, nil)
}

// TransferConversation 转接会话
func (r *CustomerController) TransferConversation(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Input("conversation_id", ""))
	toAdminID := cast.ToUint(ctx.Request().Input("to_admin_id", ""))

	if conversationID == 0 || toAdminID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "conversation_id_and_to_admin_id_required")
	}

	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, "not_logged_in")
	}

	if err := r.customerService.TransferConversation(conversationID, admin.ID, toAdminID); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "transfer_conversation_failed")
	}

	return response.Success(ctx, nil)
}

// MarkMessagesAsRead 标记消息为已读
func (r *CustomerController) MarkMessagesAsRead(ctx apphttp.Context) apphttp.Response {
	conversationID := cast.ToUint(ctx.Request().Input("conversation_id", ""))
	if conversationID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "conversation_id_required")
	}

	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, "not_logged_in")
	}

	if err := r.customerService.MarkConversationMessagesAsRead(conversationID, admin.ID); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "mark_messages_read_failed")
	}

	// 发送系统消息：消息已读
	imhub.Hub().BroadcastSystemMessage(conversationID, "read", map[string]interface{}{
		"reader_id": admin.ID,
	})

	return response.Success(ctx, nil)
}

// GetOnlineVisitors 获取在线访客列表
func (r *CustomerController) GetOnlineVisitors(ctx apphttp.Context) apphttp.Response {
	visitorIDs := imhub.Hub().GetOnlineVisitors()

	var visitors []models.Visitor
	if len(visitorIDs) > 0 {
		facades.Orm().Query().Where("id IN ?", visitorIDs).Find(&visitors)
	}

	return response.Success(ctx, visitors)
}

// GetOnlineAdmins 获取可转接的客服列表（拥有客服角色的管理员，排除当前用户，标注在线状态）
func (r *CustomerController) GetOnlineAdmins(ctx apphttp.Context) apphttp.Response {
	// 获取当前在线的管理员ID列表
	onlineAdminIDs := imhub.Hub().GetOnlineAdmins()
	onlineMap := make(map[uint]bool)
	for _, id := range onlineAdminIDs {
		onlineMap[id] = true
	}

	// 获取当前登录的管理员
	currentAdmin := r.currentAdmin(ctx)
	var currentAdminID uint
	if currentAdmin != nil {
		currentAdminID = currentAdmin.ID
	}

	// 查找客服角色
	var customerServiceRole models.Role
	if err := facades.Orm().Query().Where("slug", "customer-service").Where("status", 1).First(&customerServiceRole); err != nil {
		// 没有客服角色，返回空列表
		return response.Success(ctx, []interface{}{})
	}

	// 查询拥有客服角色的管理员ID（通过 admin_role 中间表）
	type AdminRoleLink struct {
		AdminID uint `gorm:"column:admin_id"`
	}
	var adminRoleLinks []AdminRoleLink
	facades.Orm().Query().Table("admin_role").
		Where("role_id", customerServiceRole.ID).
		Select("admin_id").
		Find(&adminRoleLinks)

	if len(adminRoleLinks) == 0 {
		return response.Success(ctx, []interface{}{})
	}

	// 获取客服管理员ID列表（排除当前用户）
	customerServiceAdminIDs := make([]uint, 0, len(adminRoleLinks))
	for _, link := range adminRoleLinks {
		if link.AdminID != currentAdminID {
			customerServiceAdminIDs = append(customerServiceAdminIDs, link.AdminID)
		}
	}

	if len(customerServiceAdminIDs) == 0 {
		return response.Success(ctx, []interface{}{})
	}

	// 查询这些管理员的详细信息（启用状态）
	var admins []models.Admin
	facades.Orm().Query().
		Where("id IN ?", customerServiceAdminIDs).
		Where("status", 1).
		Find(&admins)

	// 添加在线状态标记
	type AdminWithOnlineStatus struct {
		models.Admin
		IsOnline bool `json:"is_online"`
	}

	result := make([]AdminWithOnlineStatus, len(admins))
	for i, admin := range admins {
		result[i] = AdminWithOnlineStatus{
			Admin:    admin,
			IsOnline: onlineMap[admin.ID],
		}
	}

	return response.Success(ctx, result)
}

// WebSocket 客服 WebSocket 连接
func (r *CustomerController) WebSocket(ctx apphttp.Context) apphttp.Response {
	admin := r.currentAdmin(ctx)
	if admin == nil {
		return response.Error(ctx, http.StatusUnauthorized, "not_logged_in")
	}

	conversationID := cast.ToUint(ctx.Request().Query("conversation_id", "0"))

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
	imhub.Hub().RegisterConnection(conn, imhub.ClientTypeAdmin, admin.ID, conversationID)

	// 发送连接成功消息
	imhub.Hub().BroadcastSystemMessage(conversationID, "connected", map[string]interface{}{
		"admin_id": admin.ID,
	})

	return nil
}
