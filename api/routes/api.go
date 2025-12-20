package routes

import (
	"github.com/goravel/framework/contracts/route"
	"github.com/goravel/framework/facades"

	"goravel/app/http/controllers/visitor"
)

func Api() {
	// 访客端路由（公开 API，不需要认证）
	visitorController := visitor.NewVisitorController()
	facades.Route().Prefix("api/visitor").Group(func(router route.Router) {
		// 访客注册
		router.Post("register", visitorController.Register)
		// 创建会话
		router.Post("conversations", visitorController.CreateConversation)
		// 获取会话列表
		router.Get("conversations", visitorController.GetConversations)
		// 获取消息列表
		router.Get("messages", visitorController.GetMessages)
		// 获取所有历史消息（跨会话）
		router.Get("messages/all", visitorController.GetAllMessages)
		// 发送消息
		router.Post("messages", visitorController.SendMessage)
		// 结束会话
		router.Post("conversations/end", visitorController.EndConversation)
		// 评价会话
		router.Post("conversations/rate", visitorController.RateConversation)
		// 上传图片（公开接口，添加水印）
		router.Post("upload/image", visitorController.UploadImage)
		// 预览附件（访客专用，公开接口）
		router.Get("attachments/{id}/preview", visitorController.PreviewAttachment)
		// 心跳接口（用于保持会话活跃状态）
		router.Post("heartbeat", visitorController.Heartbeat)
		// WebSocket 连接
		router.Get("ws", visitorController.WebSocket)
	})
}
