package services

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/goravel/framework/facades"
	"goravel/app/models"
)

type MessageCacheService interface {
	// 缓存会话最近消息（最近100条）
	CacheConversationMessages(conversationID uint, messages []models.Message) error
	GetCachedConversationMessages(conversationID uint) ([]models.Message, error)
	InvalidateConversationCache(conversationID uint) error

	// 缓存会话列表
	CacheConversations(adminID uint, conversations []models.Conversation) error
	GetCachedConversations(adminID uint) ([]models.Conversation, error)
	InvalidateConversationsCache(adminID uint) error

	// 缓存在线用户列表
	CacheOnlineVisitors(visitors []models.Visitor) error
	GetCachedOnlineVisitors() ([]models.Visitor, error)
	InvalidateOnlineVisitorsCache() error
}

type MessageCacheServiceImpl struct{}

func NewMessageCacheService() MessageCacheService {
	return &MessageCacheServiceImpl{}
}

// CacheConversationMessages 缓存会话最近消息
func (s *MessageCacheServiceImpl) CacheConversationMessages(conversationID uint, messages []models.Message) error {
	if len(messages) == 0 {
		return nil
	}

	key := fmt.Sprintf("conversation:messages:%d", conversationID)
	
	// 序列化消息列表
	data, err := json.Marshal(messages)
	if err != nil {
		return err
	}

	// 缓存30分钟
	return facades.Cache().Put(key, string(data), 30*time.Minute)
}

// GetCachedConversationMessages 获取缓存的会话消息
func (s *MessageCacheServiceImpl) GetCachedConversationMessages(conversationID uint) ([]models.Message, error) {
	key := fmt.Sprintf("conversation:messages:%d", conversationID)
	
	value := facades.Cache().Get(key, "")
	if value == "" {
		return nil, nil // 缓存未命中
	}

	var messages []models.Message
	if err := json.Unmarshal([]byte(value.(string)), &messages); err != nil {
		return nil, err
	}

	return messages, nil
}

// InvalidateConversationCache 清除会话缓存
func (s *MessageCacheServiceImpl) InvalidateConversationCache(conversationID uint) error {
	key := fmt.Sprintf("conversation:messages:%d", conversationID)
	facades.Cache().Forget(key)
	return nil
}

// CacheConversations 缓存会话列表
func (s *MessageCacheServiceImpl) CacheConversations(adminID uint, conversations []models.Conversation) error {
	key := fmt.Sprintf("admin:conversations:%d", adminID)
	
	data, err := json.Marshal(conversations)
	if err != nil {
		return err
	}

	// 缓存5分钟
	return facades.Cache().Put(key, string(data), 5*time.Minute)
}

// GetCachedConversations 获取缓存的会话列表
func (s *MessageCacheServiceImpl) GetCachedConversations(adminID uint) ([]models.Conversation, error) {
	key := fmt.Sprintf("admin:conversations:%d", adminID)
	
	value := facades.Cache().Get(key, "")
	if value == "" {
		return nil, nil
	}

	var conversations []models.Conversation
	if err := json.Unmarshal([]byte(value.(string)), &conversations); err != nil {
		return nil, err
	}

	return conversations, nil
}

// InvalidateConversationsCache 清除会话列表缓存
func (s *MessageCacheServiceImpl) InvalidateConversationsCache(adminID uint) error {
	key := fmt.Sprintf("admin:conversations:%d", adminID)
	facades.Cache().Forget(key)
	return nil
}

// CacheOnlineVisitors 缓存在线访客列表
func (s *MessageCacheServiceImpl) CacheOnlineVisitors(visitors []models.Visitor) error {
	key := "visitors:online"
	
	data, err := json.Marshal(visitors)
	if err != nil {
		return err
	}

	// 缓存1分钟（在线列表变化频繁）
	return facades.Cache().Put(key, string(data), 1*time.Minute)
}

// GetCachedOnlineVisitors 获取缓存的在线访客列表
func (s *MessageCacheServiceImpl) GetCachedOnlineVisitors() ([]models.Visitor, error) {
	key := "visitors:online"
	
	value := facades.Cache().Get(key, "")
	if value == "" {
		return nil, nil
	}

	var visitors []models.Visitor
	if err := json.Unmarshal([]byte(value.(string)), &visitors); err != nil {
		return nil, err
	}

	return visitors, nil
}

// InvalidateOnlineVisitorsCache 清除在线访客缓存
func (s *MessageCacheServiceImpl) InvalidateOnlineVisitorsCache() error {
	key := "visitors:online"
	facades.Cache().Forget(key)
	return nil
}

