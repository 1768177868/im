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
		// WebSocket 连接
		router.Get("ws", visitorController.WebSocket)
	})
}
