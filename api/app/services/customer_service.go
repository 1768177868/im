package services

import (
	"time"

	"github.com/goravel/framework/facades"
	"github.com/oklog/ulid/v2"

	"goravel/app/models"
	imhub "goravel/app/websocket/im"
)

type CustomerService interface {
	// CreateOrGetVisitor 创建或获取访客
	CreateOrGetVisitor(visitorID string, name, email, phone, ip, userAgent, source, referer, location, device, browser, os string) (*models.Visitor, error)
	// UpdateVisitorStatus 更新访客状态
	UpdateVisitorStatus(visitorID uint, status uint8) error
	// CreateConversation 创建会话
	CreateConversation(visitorID uint, adminID *uint, title string) (*models.Conversation, error)
	// AssignConversation 分配会话给客服
	AssignConversation(conversationID uint, adminID uint) error
	// GetAvailableAdmin 获取可用的客服（最少会话数策略）
	GetAvailableAdmin() (*models.Admin, error)
	// SendMessage 发送消息
	SendMessage(conversationID uint, senderType string, senderID uint, content, msgType, fileURL, fileName string, fileSize int64) (*models.Message, error)
	// MarkMessageAsRead 标记消息为已读
	MarkMessageAsRead(messageID uint) error
	// MarkConversationMessagesAsRead 标记会话所有消息为已读
	MarkConversationMessagesAsRead(conversationID uint, readerID uint) error
	// EndConversation 结束会话
	EndConversation(conversationID uint) error
	// GetConversationMessages 获取会话消息列表
	GetConversationMessages(conversationID uint, page, pageSize int) ([]models.Message, int64, error)
	// GetVisitorAllMessages 获取访客的所有历史消息（跨会话，游标分页）
	// beforeID: 获取比这个ID更早的消息，0表示获取最新消息
	GetVisitorAllMessages(visitorID uint, beforeID uint, limit int) ([]models.Message, bool, error)
	// GetVisitorConversations 获取访客的会话列表
	GetVisitorConversations(visitorID uint) ([]models.Conversation, error)
	// GetAdminConversations 获取客服的会话列表
	GetAdminConversations(adminID uint, status *uint8) ([]models.Conversation, error)
	// GetAdminConversationsPaginated 获取客服的会话列表（分页）
	GetAdminConversationsPaginated(adminID uint, status *uint8, page, pageSize int) ([]models.Conversation, int64, error)
	// GetAllConversations 获取所有会话列表
	GetAllConversations(status *uint8) ([]models.Conversation, error)
	// GetAllConversationsPaginated 获取所有会话列表（分页，支持搜索）
	GetAllConversationsPaginated(status *uint8, conversationID *uint, visitorName string, page, pageSize int) ([]models.Conversation, int64, error)
}

type CustomerServiceImpl struct{}

func NewCustomerService() *CustomerServiceImpl {
	return &CustomerServiceImpl{}
}

// CreateOrGetVisitor 创建或获取访客
func (s *CustomerServiceImpl) CreateOrGetVisitor(visitorID string, name, email, phone, ip, userAgent, source, referer, location, device, browser, os string) (*models.Visitor, error) {
	var visitor models.Visitor

	// 如果 visitorID 为空，生成一个新的
	if visitorID == "" {
		visitorID = ulid.Make().String()
	}

	// 查找或创建访客
	err := facades.Orm().Query().Where("visitor_id", visitorID).FirstOrCreate(&visitor, models.Visitor{
		VisitorID: visitorID,
		Name:      name,
		Email:     email,
		Phone:     phone,
		IP:        ip,
		UserAgent: userAgent,
		Source:    source,
		Referer:   referer,
		Location:  location,
		Device:    device,
		Browser:   browser,
		OS:        os,
		Status:    1,
	})

	if err != nil {
		return nil, err
	}

	// 更新最后活跃时间
	now := time.Now()
	visitor.LastActiveAt = &now
	visitor.Status = 1
	if err := facades.Orm().Query().Save(&visitor); err != nil {
		return nil, err
	}

	return &visitor, nil
}

// UpdateVisitorStatus 更新访客状态
func (s *CustomerServiceImpl) UpdateVisitorStatus(visitorID uint, status uint8) error {
	_, err := facades.Orm().Query().Model(&models.Visitor{}).Where("id", visitorID).Update("status", status)
	return err
}

// CreateConversation 创建会话
func (s *CustomerServiceImpl) CreateConversation(visitorID uint, adminID *uint, title string) (*models.Conversation, error) {
	now := time.Now()
	conversation := models.Conversation{
		VisitorID:     visitorID,
		AdminID:       adminID,
		Title:         title,
		Status:        1, // 进行中
		Priority:      1, // 普通
		StartedAt:     &now,
		LastMessageAt: &now,
	}

	if err := facades.Orm().Query().Create(&conversation); err != nil {
		return nil, err
	}

	// 加载关联数据
	if conversation.VisitorID > 0 {
		facades.Orm().Query().Where("id", conversation.VisitorID).First(&conversation.Visitor)
	}
	if conversation.AdminID != nil && *conversation.AdminID > 0 {
		var admin models.Admin
		if err := facades.Orm().Query().Where("id", *conversation.AdminID).First(&admin); err == nil {
			conversation.Admin = &admin
		}
	}

	return &conversation, nil
}

// AssignConversation 分配会话给客服
func (s *CustomerServiceImpl) AssignConversation(conversationID uint, adminID uint) error {
	_, err := facades.Orm().Query().Model(&models.Conversation{}).Where("id", conversationID).Update("admin_id", adminID)
	return err
}

// GetAvailableAdmin 获取可用的客服（最少会话数策略）
func (s *CustomerServiceImpl) GetAvailableAdmin() (*models.Admin, error) {
	// 先获取客服角色的ID
	var customerServiceRole models.Role
	if err := facades.Orm().Query().Where("slug", "customer-service").Where("status", 1).First(&customerServiceRole); err != nil {
		// 如果没有客服角色，返回 nil
		return nil, nil
	}

	// 获取所有启用的、具有客服角色的管理员
	var adminIDs []uint
	if err := facades.Orm().Query().
		Table("admin_role").
		Select("admin_id").
		Where("role_id", customerServiceRole.ID).
		Pluck("admin_id", &adminIDs); err != nil {
		return nil, err
	}

	if len(adminIDs) == 0 {
		return nil, nil
	}

	// 获取这些管理员的信息
	var admins []models.Admin
	if err := facades.Orm().Query().
		Where("id IN ?", adminIDs).
		Where("status", 1).
		Find(&admins); err != nil {
		return nil, err
	}

	if len(admins) == 0 {
		return nil, nil
	}

	// 获取每个客服的当前会话数
	adminConversationCount := make(map[uint]int64)
	for _, admin := range admins {
		count, err := facades.Orm().Query().Model(&models.Conversation{}).
			Where("admin_id", admin.ID).
			Where("status", 1). // 只统计进行中的会话
			Count()
		if err == nil {
			adminConversationCount[admin.ID] = count
		}
	}

	// 找到会话数最少的在线客服（优先在线且最闲）
	var selectedAdmin *models.Admin
	minCount := int64(999999)

	// 第一轮：优先选择在线且会话数最少的客服
	for i := range admins {
		count := adminConversationCount[admins[i].ID]
		if imhub.Hub().IsAdminOnline(admins[i].ID) {
			if count < minCount {
				minCount = count
				selectedAdmin = &admins[i]
			}
		}
	}

	// 如果没有在线客服，选择会话数最少的（即使不在线）
	if selectedAdmin == nil {
		minCount = int64(999999)
		for i := range admins {
			count := adminConversationCount[admins[i].ID]
			if count < minCount {
				minCount = count
				selectedAdmin = &admins[i]
			}
		}
	}

	return selectedAdmin, nil
}

// SendMessage 发送消息
func (s *CustomerServiceImpl) SendMessage(conversationID uint, senderType string, senderID uint, content, msgType, fileURL, fileName string, fileSize int64) (*models.Message, error) {
	message := models.Message{
		ConversationID: conversationID,
		SenderType:     senderType,
		SenderID:       senderID,
		Content:        content,
		Type:           msgType,
		FileURL:        fileURL,
		FileName:       fileName,
		FileSize:       fileSize,
		IsRead:         false,
	}

	if err := facades.Orm().Query().Create(&message); err != nil {
		return nil, err
	}

	// 更新会话的最后消息时间
	now := time.Now()
	if _, err := facades.Orm().Query().Model(&models.Conversation{}).
		Where("id", conversationID).
		Update("last_message_at", now); err != nil {
		return nil, err
	}

	// 加载关联数据
	if message.ConversationID > 0 {
		facades.Orm().Query().Where("id", message.ConversationID).First(&message.Conversation)
	}

	return &message, nil
}

// MarkMessageAsRead 标记消息为已读
func (s *CustomerServiceImpl) MarkMessageAsRead(messageID uint) error {
	now := time.Now()
	_, err := facades.Orm().Query().Model(&models.Message{}).
		Where("id", messageID).
		Update(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	return err
}

// MarkConversationMessagesAsRead 标记会话所有消息为已读
func (s *CustomerServiceImpl) MarkConversationMessagesAsRead(conversationID uint, readerID uint) error {
	now := time.Now()
	_, err := facades.Orm().Query().Model(&models.Message{}).
		Where("conversation_id", conversationID).
		Where("sender_id != ?", readerID). // 不标记自己发送的消息
		Where("is_read", false).
		Update(map[string]interface{}{
			"is_read": true,
			"read_at": now,
		})
	return err
}

// EndConversation 结束会话
func (s *CustomerServiceImpl) EndConversation(conversationID uint) error {
	now := time.Now()
	_, err := facades.Orm().Query().Model(&models.Conversation{}).
		Where("id", conversationID).
		Update(map[string]interface{}{
			"status":   2, // 已结束
			"ended_at": now,
		})
	return err
}

// GetConversationMessages 获取会话消息列表
func (s *CustomerServiceImpl) GetConversationMessages(conversationID uint, page, pageSize int) ([]models.Message, int64, error) {
	var messages []models.Message
	var total int64

	query := facades.Orm().Query().Model(&models.Message{}).Where("conversation_id", conversationID)

	// 获取总数
	var err error
	total, err = query.Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页查询 - 重新构建查询避免 count 影响
	offset := (page - 1) * pageSize
	if err := facades.Orm().Query().Model(&models.Message{}).Where("conversation_id", conversationID).Order("id DESC").Limit(pageSize).Offset(offset).Find(&messages); err != nil {
		return nil, 0, err
	}

	// 反转顺序，使最早的消息在前
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, total, nil
}

// GetVisitorAllMessages 获取访客的所有历史消息（跨会话，游标分页）
// beforeID: 获取比这个ID更早的消息，0表示获取最新消息
// 返回：消息列表、是否还有更多、错误
func (s *CustomerServiceImpl) GetVisitorAllMessages(visitorID uint, beforeID uint, limit int) ([]models.Message, bool, error) {
	// 先获取访客的所有会话ID
	var conversationIDs []uint
	if err := facades.Orm().Query().Model(&models.Conversation{}).
		Where("visitor_id", visitorID).
		Pluck("id", &conversationIDs); err != nil {
		return nil, false, err
	}

	if len(conversationIDs) == 0 {
		return []models.Message{}, false, nil
	}

	// 转换为 []any 类型
	ids := make([]any, len(conversationIDs))
	for i, id := range conversationIDs {
		ids[i] = id
	}

	// 构建查询
	query := facades.Orm().Query().Model(&models.Message{}).
		WhereIn("conversation_id", ids)

	// 如果有 beforeID，获取比它更早的消息
	if beforeID > 0 {
		query = query.Where("id < ?", beforeID)
	}

	// 多查一条用于判断是否还有更多
	var messages []models.Message
	if err := query.Order("id DESC").Limit(limit + 1).Find(&messages); err != nil {
		return nil, false, err
	}

	// 判断是否还有更多
	hasMore := len(messages) > limit
	if hasMore {
		messages = messages[:limit] // 去掉多查的那一条
	}

	// 反转数组，使最早的消息在前（方便前端显示）
	for i, j := 0, len(messages)-1; i < j; i, j = i+1, j-1 {
		messages[i], messages[j] = messages[j], messages[i]
	}

	return messages, hasMore, nil
}

// GetVisitorConversations 获取访客的会话列表
func (s *CustomerServiceImpl) GetVisitorConversations(visitorID uint) ([]models.Conversation, error) {
	var conversations []models.Conversation
	if err := facades.Orm().Query().
		Where("visitor_id", visitorID).
		Order("last_message_at DESC").
		Find(&conversations); err != nil {
		return nil, err
	}
	// 批量加载关联数据
	for i := range conversations {
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			var admin models.Admin
			if err := facades.Orm().Query().Where("id", *conversations[i].AdminID).First(&admin); err == nil {
				conversations[i].Admin = &admin
			}
		}
	}
	return conversations, nil
}

// GetAdminConversations 获取客服的会话列表
func (s *CustomerServiceImpl) GetAdminConversations(adminID uint, status *uint8) ([]models.Conversation, error) {
	var conversations []models.Conversation
	query := facades.Orm().Query().
		Where("admin_id", adminID)

	if status != nil {
		query = query.Where("status", *status)
	}

	if err := query.Order("last_message_at DESC").Find(&conversations); err != nil {
		return nil, err
	}
	// 批量加载关联数据
	for i := range conversations {
		if conversations[i].VisitorID > 0 {
			facades.Orm().Query().Where("id", conversations[i].VisitorID).First(&conversations[i].Visitor)
		}
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			var admin models.Admin
			if err := facades.Orm().Query().Where("id", *conversations[i].AdminID).First(&admin); err == nil {
				conversations[i].Admin = &admin
			}
		}
	}
	return conversations, nil
}

// GetAllConversations 获取所有会话列表
func (s *CustomerServiceImpl) GetAllConversations(status *uint8) ([]models.Conversation, error) {
	var conversations []models.Conversation
	query := facades.Orm().Query()

	if status != nil {
		query = query.Where("status", *status)
	}

	if err := query.Order("last_message_at DESC").Find(&conversations); err != nil {
		return nil, err
	}

	// 批量加载关联数据
	var visitorIDs []uint
	var adminIDs []uint
	for i := range conversations {
		if conversations[i].VisitorID > 0 {
			visitorIDs = append(visitorIDs, conversations[i].VisitorID)
		}
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			adminIDs = append(adminIDs, *conversations[i].AdminID)
		}
	}

	// 批量查询访客
	visitorMap := make(map[uint]models.Visitor)
	if len(visitorIDs) > 0 {
		var visitors []models.Visitor
		if err := facades.Orm().Query().Where("id IN ?", visitorIDs).Find(&visitors); err == nil {
			for _, visitor := range visitors {
				visitorMap[visitor.ID] = visitor
			}
		}
	}

	// 批量查询管理员
	adminMap := make(map[uint]models.Admin)
	if len(adminIDs) > 0 {
		var admins []models.Admin
		if err := facades.Orm().Query().Where("id IN ?", adminIDs).Find(&admins); err == nil {
			for _, admin := range admins {
				adminMap[admin.ID] = admin
			}
		}
	}

	// 填充关联数据
	for i := range conversations {
		if conversations[i].VisitorID > 0 {
			if visitor, ok := visitorMap[conversations[i].VisitorID]; ok {
				conversations[i].Visitor = visitor
			}
		}
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			if admin, ok := adminMap[*conversations[i].AdminID]; ok {
				conversations[i].Admin = &admin
			}
		}
	}

	return conversations, nil
}

// GetAdminConversationsPaginated 获取客服的会话列表（分页）
func (s *CustomerServiceImpl) GetAdminConversationsPaginated(adminID uint, status *uint8, page, pageSize int) ([]models.Conversation, int64, error) {
	query := facades.Orm().Query().Model(&models.Conversation{}).
		Where("admin_id", adminID)

	if status != nil {
		query = query.Where("status", *status)
	}

	// 获取总数
	var total int64
	var err error
	total, err = query.Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页查询 - 重新构建查询避免 count 影响
	query2 := facades.Orm().Query().Model(&models.Conversation{}).Where("admin_id", adminID)
	if status != nil {
		query2 = query2.Where("status", *status)
	}
	offset := (page - 1) * pageSize
	var conversations []models.Conversation
	if err := query2.Order("last_message_at DESC").Limit(pageSize).Offset(offset).Find(&conversations); err != nil {
		return nil, 0, err
	}

	// 批量加载关联数据
	var visitorIDs []uint
	var adminIDs []uint
	for i := range conversations {
		if conversations[i].VisitorID > 0 {
			visitorIDs = append(visitorIDs, conversations[i].VisitorID)
		}
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			adminIDs = append(adminIDs, *conversations[i].AdminID)
		}
	}

	// 批量查询访客
	visitorMap := make(map[uint]models.Visitor)
	if len(visitorIDs) > 0 {
		var visitors []models.Visitor
		if err := facades.Orm().Query().Where("id IN ?", visitorIDs).Find(&visitors); err == nil {
			for _, visitor := range visitors {
				visitorMap[visitor.ID] = visitor
			}
		}
	}

	// 批量查询管理员
	adminMap := make(map[uint]models.Admin)
	if len(adminIDs) > 0 {
		var admins []models.Admin
		if err := facades.Orm().Query().Where("id IN ?", adminIDs).Find(&admins); err == nil {
			for _, admin := range admins {
				adminMap[admin.ID] = admin
			}
		}
	}

	// 填充关联数据
	for i := range conversations {
		if conversations[i].VisitorID > 0 {
			if visitor, ok := visitorMap[conversations[i].VisitorID]; ok {
				conversations[i].Visitor = visitor
			}
		}
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			if admin, ok := adminMap[*conversations[i].AdminID]; ok {
				conversations[i].Admin = &admin
			}
		}
	}

	return conversations, total, nil
}

// GetAllConversationsPaginated 获取所有会话列表（分页，支持搜索）
func (s *CustomerServiceImpl) GetAllConversationsPaginated(status *uint8, conversationID *uint, visitorName string, page, pageSize int) ([]models.Conversation, int64, error) {
	query := facades.Orm().Query().Model(&models.Conversation{})

	// 搜索条件：会话ID
	if conversationID != nil && *conversationID > 0 {
		query = query.Where("id", *conversationID)
	}

	// 搜索条件：状态
	if status != nil {
		query = query.Where("status", *status)
	}

	// 搜索条件：访客姓名（需要关联查询）
	var visitorIDs []uint
	if visitorName != "" {
		facades.Orm().Query().Model(&models.Visitor{}).
			Where("name LIKE ?", "%"+visitorName+"%").
			OrWhere("visitor_id LIKE ?", "%"+visitorName+"%").
			Pluck("id", &visitorIDs)

		if len(visitorIDs) > 0 {
			// 转换为 []any 类型
			ids := make([]any, len(visitorIDs))
			for i, id := range visitorIDs {
				ids[i] = id
			}
			query = query.WhereIn("visitor_id", ids)
		} else {
			// 如果没有匹配的访客，返回空结果
			return []models.Conversation{}, 0, nil
		}
	}

	// 获取总数
	var total int64
	var err error
	total, err = query.Count()
	if err != nil {
		return nil, 0, err
	}

	// 分页查询 - 重新构建查询避免 count 影响
	query2 := facades.Orm().Query().Model(&models.Conversation{})

	if conversationID != nil && *conversationID > 0 {
		query2 = query2.Where("id", *conversationID)
	}
	if status != nil {
		query2 = query2.Where("status", *status)
	}
	if len(visitorIDs) > 0 {
		ids := make([]any, len(visitorIDs))
		for i, id := range visitorIDs {
			ids[i] = id
		}
		query2 = query2.WhereIn("visitor_id", ids)
	}

	offset := (page - 1) * pageSize
	var conversations []models.Conversation
	if err := query2.Order("last_message_at DESC").Limit(pageSize).Offset(offset).Find(&conversations); err != nil {
		return nil, 0, err
	}

	// 批量加载关联数据
	var conversationVisitorIDs []uint
	var adminIDs []uint
	for i := range conversations {
		if conversations[i].VisitorID > 0 {
			conversationVisitorIDs = append(conversationVisitorIDs, conversations[i].VisitorID)
		}
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			adminIDs = append(adminIDs, *conversations[i].AdminID)
		}
	}

	// 批量查询访客
	visitorMap := make(map[uint]models.Visitor)
	if len(conversationVisitorIDs) > 0 {
		var visitors []models.Visitor
		if err := facades.Orm().Query().Where("id IN ?", conversationVisitorIDs).Find(&visitors); err == nil {
			for _, visitor := range visitors {
				visitorMap[visitor.ID] = visitor
			}
		}
	}

	// 批量查询管理员
	adminMap := make(map[uint]models.Admin)
	if len(adminIDs) > 0 {
		var admins []models.Admin
		if err := facades.Orm().Query().Where("id IN ?", adminIDs).Find(&admins); err == nil {
			for _, admin := range admins {
				adminMap[admin.ID] = admin
			}
		}
	}

	// 填充关联数据
	for i := range conversations {
		if conversations[i].VisitorID > 0 {
			if visitor, ok := visitorMap[conversations[i].VisitorID]; ok {
				conversations[i].Visitor = visitor
			}
		}
		if conversations[i].AdminID != nil && *conversations[i].AdminID > 0 {
			if admin, ok := adminMap[*conversations[i].AdminID]; ok {
				conversations[i].Admin = &admin
			}
		}
	}

	return conversations, total, nil
}
