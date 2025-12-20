package visitor

import (
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	apphttp "github.com/goravel/framework/contracts/http"
	"github.com/goravel/framework/facades"
	"github.com/gorilla/websocket"
	"github.com/spf13/cast"

	"goravel/app/http/response"
	"goravel/app/models"
	"goravel/app/services"
	"goravel/app/utils"
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
	fileURL := ctx.Request().Input("file_url", "")
	fileName := ctx.Request().Input("file_name", "")
	fileSize := cast.ToInt64(ctx.Request().Input("file_size", "0"))

	// 文本消息必须有内容，文件/图片消息必须有 file_url
	if conversationID == 0 || visitorID == "" {
		return response.Error(ctx, http.StatusBadRequest, "invalid_params")
	}
	if msgType == "text" && content == "" {
		return response.Error(ctx, http.StatusBadRequest, "content_required")
	}
	if (msgType == "image" || msgType == "file") && fileURL == "" {
		return response.Error(ctx, http.StatusBadRequest, "file_url_required")
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
		SenderType:     "visitor",
		SenderID:       visitor.ID,
		Content:        content,
		FileURL:        fileURL,
		FileName:       fileName,
		FileSize:       fileSize,
		Timestamp:      time.Now().Unix(),
		MessageID:      message.ID,
	})

	return response.Success(ctx, message)
}

// EndConversation 访客结束会话
func (r *VisitorController) EndConversation(ctx apphttp.Context) apphttp.Response {
	visitorID := ctx.Request().Input("visitor_id", "")
	conversationID := cast.ToUint(ctx.Request().Input("conversation_id", "0"))

	if visitorID == "" {
		return response.Error(ctx, http.StatusBadRequest, "visitor_id_required")
	}

	if conversationID == 0 {
		return response.Error(ctx, http.StatusBadRequest, "conversation_id_required")
	}

	// 查找访客
	var visitor models.Visitor
	if err := facades.Orm().Query().Where("visitor_id", visitorID).First(&visitor); err != nil {
		return response.Error(ctx, http.StatusNotFound, "visitor_not_found")
	}

	// 验证会话属于该访客
	var conversation models.Conversation
	if err := facades.Orm().Query().Where("id", conversationID).Where("visitor_id", visitor.ID).First(&conversation); err != nil {
		return response.Error(ctx, http.StatusNotFound, "conversation_not_found")
	}

	// 只有进行中的会话才能结束
	if conversation.Status != 1 {
		return response.Error(ctx, http.StatusBadRequest, "conversation_already_ended")
	}

	// 结束会话
	if err := r.customerService.EndConversation(conversationID); err != nil {
		return response.Error(ctx, http.StatusInternalServerError, "end_conversation_failed")
	}

	// 发送系统消息：会话已结束
	imhub.Hub().BroadcastSystemMessage(conversationID, "ended", nil)

	return response.Success(ctx, nil)
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

// UploadImage 访客上传图片（公开接口，添加水印）
func (r *VisitorController) UploadImage(ctx apphttp.Context) apphttp.Response {
	file, err := ctx.Request().File("file")
	if err != nil {
		return response.Error(ctx, http.StatusBadRequest, "file_required")
	}

	filename := file.GetClientOriginalName()
	if filename == "" {
		filename = "uploaded_image.jpg"
	}

	// 验证文件类型，只允许图片
	ext := filepath.Ext(filename)
	mimeType := mime.TypeByExtension(ext)
	if !strings.HasPrefix(mimeType, "image/") {
		return response.Error(ctx, http.StatusBadRequest, "only_image_allowed")
	}

	// 读取文件内容
	storage := facades.Storage().Disk("local")
	savedPath, err := storage.PutFile("", file)
	if err != nil {
		return response.ErrorWithLog(ctx, "visitor_upload", err, map[string]any{
			"filename": filename,
		})
	}

	// 读取文件内容
	fileDataStr, err := storage.Get(savedPath)
	if err != nil {
		_ = storage.Delete(savedPath)
		return response.ErrorWithLog(ctx, "visitor_upload", err, map[string]any{
			"filename": filename,
		})
	}

	// 清理临时文件
	_ = storage.Delete(savedPath)

	// 转换为字节数组
	fileData := []byte(fileDataStr)

	// 从配置中获取水印设置
	watermarkImagePath := utils.GetConfigValue("watermark", "watermark_image_path", "")
	watermarkPosition := utils.GetConfigValue("watermark", "watermark_position", "bottom-right")
	watermarkOpacity := utils.GetConfigValueInt("watermark", "watermark_opacity", 128)
	watermarkScale := utils.GetConfigValueFloat("watermark", "watermark_scale", 0.3)

	// 如果启用了水印，添加水印
	if watermarkImagePath != "" {
		// 如果路径是 attachment:ID 格式，通过ID查找附件路径
		var actualPath string
		if strings.HasPrefix(watermarkImagePath, "attachment:") {
			attachmentIDStr := strings.TrimPrefix(watermarkImagePath, "attachment:")
			attachmentID := cast.ToUint(attachmentIDStr)
			if attachmentID > 0 {
				var attachment models.Attachment
				if err := facades.Orm().Query().Where("id", attachmentID).First(&attachment); err == nil {
					actualPath = attachment.Path
				} else {
					facades.Log().Errorf("查找水印附件失败: %v, id: %d", err, attachmentID)
				}
			}
		} else {
			actualPath = watermarkImagePath
		}

		if actualPath != "" {
			watermarkedData, err := utils.AddWatermark(
				fileData,
				actualPath,
				watermarkPosition,
				watermarkOpacity,
				watermarkScale,
			)
			if err != nil {
				facades.Log().Errorf("添加水印失败: %v", err)
				// 水印添加失败，使用原图
			} else {
				fileData = watermarkedData
			}
		}
	}

	// 使用附件服务保存文件
	attachmentService := services.NewAttachmentService(ctx)
	attachment, err := attachmentService.UploadFile(fileData, filename, mimeType)
	if err != nil {
		return response.ErrorWithLog(ctx, "visitor_upload", err, map[string]any{
			"filename": filename,
		})
	}

	// 返回访客专用的文件URL（不需要认证）
	visitorFileURL := fmt.Sprintf("/api/visitor/attachments/%d/preview", attachment.ID)

	return response.Success(ctx, "upload_success", apphttp.Json{
		"id":        attachment.ID,
		"filename":  attachment.Filename,
		"size":      attachment.Size,
		"mime_type": attachment.MimeType,
		"file_type": attachment.FileType,
		"file_url":  visitorFileURL,
	})
}

// PreviewAttachment 预览附件（访客专用，公开接口）
func (r *VisitorController) PreviewAttachment(ctx apphttp.Context) apphttp.Response {
	id := cast.ToUint(ctx.Request().Route("id"))
	if id == 0 {
		return response.Error(ctx, http.StatusBadRequest, "id_required")
	}

	var attachment models.Attachment
	if err := facades.Orm().Query().Where("id", id).First(&attachment); err != nil {
		return response.Error(ctx, http.StatusNotFound, "record_not_found")
	}

	if attachment.Path == "" || attachment.Disk == "" {
		return response.Error(ctx, http.StatusBadRequest, "file_path_required")
	}

	// 对于云存储，尝试生成临时URL并重定向
	if attachment.Disk != "local" && attachment.Disk != "public" {
		storage := facades.Storage().Disk(attachment.Disk)
		if url, err := storage.TemporaryUrl(attachment.Path, time.Now().Add(24*time.Hour)); err == nil {
			return ctx.Response().Redirect(http.StatusFound, url)
		}

		// 如果生成临时URL失败，尝试从配置获取基础URL
		attachmentService := services.NewAttachmentService(ctx)
		directURL := attachmentService.GetFileURL(&attachment)
		if directURL != "" && !strings.Contains(directURL, "/api/admin/attachments/") {
			return ctx.Response().Redirect(http.StatusFound, directURL)
		}
	}

	// 对于本地存储，使用服务器中转
	storage := facades.Storage().Disk(attachment.Disk)
	content, err := storage.Get(attachment.Path)
	if err != nil {
		return response.ErrorWithLog(ctx, "visitor_attachment", err, map[string]any{
			"disk": attachment.Disk,
			"path": attachment.Path,
		})
	}

	// 设置响应头
	mimeType := attachment.MimeType
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// 设置响应头
	response := ctx.Response().
		Header("Content-Type", mimeType).
		Header("Content-Length", fmt.Sprintf("%d", len(content))).
		Header("Cache-Control", "public, max-age=3600")

	// 对于图片和视频，支持范围请求
	if attachment.FileType == "image" || attachment.FileType == "video" {
		response = response.Header("Accept-Ranges", "bytes")
	}

	return response.String(http.StatusOK, content)
}

// Heartbeat 心跳接口，用于更新会话的最后活跃时间和访客状态
func (r *VisitorController) Heartbeat(ctx apphttp.Context) apphttp.Response {
	visitorID := ctx.Request().Input("visitor_id", "")
	conversationID := ctx.Request().Input("conversation_id", "")
	// status: online（在线/活跃）, away（离开/页面不可见）
	status := ctx.Request().Input("status", "online")

	if visitorID == "" {
		return response.Error(ctx, http.StatusBadRequest, "visitor_id_required")
	}

	// 查找访客
	var visitor models.Visitor
	if err := facades.Orm().Query().Where("visitor_id", visitorID).First(&visitor); err != nil {
		return response.Error(ctx, http.StatusNotFound, "visitor_not_found")
	}

	now := time.Now()

	// 更新访客的最后活跃时间和状态
	visitorStatus := uint8(1) // 1: 在线
	if status == "away" {
		visitorStatus = 2 // 2: 离开（页面不可见）
	}
	facades.Orm().Query().
		Model(&models.Visitor{}).
		Where("id", visitor.ID).
		Update(map[string]interface{}{
			"status":         visitorStatus,
			"last_active_at": now,
		})

	// 如果提供了会话ID，更新会话的最后活跃时间
	if conversationID != "" {
		conversationIDUint := cast.ToUint(conversationID)
		if conversationIDUint > 0 {
			facades.Orm().Query().
				Model(&models.Conversation{}).
				Where("id", conversationIDUint).
				Where("status", 1). // 只更新进行中的会话
				Update("last_message_at", now)
		}
	}

	// 广播访客状态变更给相关客服
	if conversationID != "" {
		conversationIDUint := cast.ToUint(conversationID)
		if conversationIDUint > 0 {
			imhub.Hub().BroadcastSystemMessage(conversationIDUint, "visitor_status", map[string]interface{}{
				"visitor_id": visitor.ID,
				"status":     status,
			})
		}
	}

	return response.Success(ctx, "heartbeat_success", apphttp.Json{
		"timestamp": now.Unix(),
		"status":    status,
	})
}
