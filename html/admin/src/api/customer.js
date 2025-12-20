import request from '../utils/request'

/**
 * 获取会话列表
 */
export function getConversations(params) {
  return request({
    url: '/customer/conversations',
    method: 'get',
    params
  })
}

/**
 * 获取会话详情
 */
export function getConversationDetail(id) {
  return request({
    url: `/customer/conversations/${id}`,
    method: 'get'
  })
}

export function getVisitorStatus(conversationId) {
  return request({
    url: '/customer/conversations/visitor-status',
    method: 'get',
    params: { conversation_id: conversationId }
  })
}

/**
 * 获取消息列表
 */
export function getMessages(params) {
  return request({
    url: '/customer/messages',
    method: 'get',
    params
  })
}

/**
 * 发送消息
 */
export function sendMessage(data) {
  return request({
    url: '/customer/messages',
    method: 'post',
    data
  })
}

/**
 * 分配会话
 */
export function assignConversation(data) {
  return request({
    url: '/customer/conversations/assign',
    method: 'post',
    data
  })
}

/**
 * 结束会话
 */
export function endConversation(data) {
  return request({
    url: '/customer/conversations/end',
    method: 'post',
    data
  })
}

/**
 * 标记消息已读
 */
export function markMessagesAsRead(data) {
  return request({
    url: '/customer/messages/read',
    method: 'post',
    data
  })
}

/**
 * 获取在线访客列表
 */
export function getOnlineVisitors() {
  return request({
    url: '/customer/visitors/online',
    method: 'get'
  })
}

/**
 * 获取在线客服列表
 */
export function getOnlineAdmins() {
  return request({
    url: '/customer/admins/online',
    method: 'get'
  })
}

/**
 * 转接会话
 */
export function transferConversation(data) {
  return request({
    url: '/customer/conversations/transfer',
    method: 'post',
    data
  })
}

