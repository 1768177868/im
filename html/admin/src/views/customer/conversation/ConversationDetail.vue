<template>
  <div class="conversation-detail">
    <div v-if="loading" class="loading-container">
      <el-skeleton :rows="10" animated />
    </div>

    <div v-else class="detail-content">
      <!-- 顶部信息栏 -->
      <div class="chat-toolbar">
        <div class="visitor-info">
          <el-avatar :size="36">
            {{ (conversation?.visitor?.name || conversation?.visitor?.visitor_id || '访')?.charAt(0) }}
          </el-avatar>
          <div class="visitor-meta">
            <span class="visitor-name">
              {{ conversation?.visitor?.name || conversation?.visitor?.visitor_id || `访客${conversation?.visitor_id}` }}
            </span>
            <el-tag v-if="conversation?.status" :type="getStatusType(conversation.status)" size="small">
              {{ getStatusText(conversation.status) }}
            </el-tag>
          </div>
        </div>
        <div class="toolbar-actions">
          <el-tooltip :content="$t('common.refresh')" placement="top">
            <el-button 
              :icon="Refresh" 
              circle 
              size="small"
              @click="loadMessages"
              :loading="messagesLoading"
            />
          </el-tooltip>
          <el-popover placement="bottom" :width="280" trigger="click">
            <template #reference>
              <el-button :icon="InfoFilled" circle size="small" />
            </template>
            <el-descriptions :column="1" size="small">
              <el-descriptions-item :label="$t('customer.conversation.admin_name')">
                {{ conversation?.admin?.nickname || '-' }}
              </el-descriptions-item>
              <el-descriptions-item :label="$t('customer.conversation.title')">
                {{ conversation?.title || '-' }}
              </el-descriptions-item>
              <el-descriptions-item :label="$t('common.created_at')">
                {{ formatTime(conversation?.created_at) }}
              </el-descriptions-item>
              <el-descriptions-item :label="$t('customer.conversation.last_message_at')">
                {{ formatTime(conversation?.last_message_at) }}
              </el-descriptions-item>
            </el-descriptions>
          </el-popover>
          <el-button 
            v-if="conversation?.status === 1"
            type="danger" 
            size="small"
            @click="handleEndConversation"
            :loading="ending"
          >
            {{ $t('customer.conversation.end') }}
          </el-button>
        </div>
      </div>

      <!-- 消息列表 -->
      <div class="messages-section">

        <div class="messages-wrapper">
          <div class="messages-container" ref="messagesContainerRef" @scroll="handleScroll">
            <!-- 加载更多 -->
            <div v-if="hasMoreMessages && messages.length > 0" class="load-more">
            <el-button 
              text 
              :loading="loadingMore"
              @click="loadMoreMessages"
              size="small"
            >
              {{ loadingMore ? '加载中...' : '加载更多历史消息' }}
            </el-button>
          </div>
          
          <div v-if="messages.length === 0 && !loadingMore" class="empty-messages">
            <el-empty :description="$t('customer.conversation.no_messages')" />
          </div>
          <div
            v-for="message in messages"
            :key="message.id"
            class="message-item"
            :class="{ 'is-visitor': message.sender_type === 'visitor', 'is-admin': message.sender_type === 'admin' }"
          >
            <el-avatar 
              v-if="message.sender_type === 'admin'"
              :size="32" 
              :src="conversation?.admin?.avatar"
            >
              {{ conversation?.admin?.nickname?.charAt(0) || '客' }}
            </el-avatar>
            <div class="message-content">
              <div class="message-bubble" :class="{ 'visitor-bubble': message.sender_type === 'visitor', 'admin-bubble': message.sender_type === 'admin' }">
                <div class="message-text">{{ message.content }}</div>
                <div class="message-time">
                  <el-icon v-if="message.is_sending" class="sending-icon"><Loading /></el-icon>
                  {{ formatTime(message.created_at) }}
                </div>
              </div>
            </div>
            <el-avatar 
              v-if="message.sender_type === 'visitor'"
              :size="32" 
            >
              {{ conversation?.visitor?.name?.charAt(0) || '访' }}
            </el-avatar>
          </div>
          </div>
          
          <!-- 滚动到底部按钮 / 未读消息提示 -->
          <div 
            v-if="showScrollToBottom || unreadCount > 0" 
            class="scroll-to-bottom-btn"
            @click="scrollToBottomAndClearUnread"
          >
            <el-badge :value="unreadCount" :hidden="unreadCount === 0" :max="99">
              <el-button 
                type="primary" 
                :icon="ArrowDown"
                circle
                size="small"
              />
            </el-badge>
            <span v-if="unreadCount > 0" class="unread-text">{{ unreadCount }}条新消息</span>
          </div>
        </div>
      </div>

      <!-- 发送消息 -->
      <div class="send-message-section" v-if="conversation?.status === 1">
        <el-input
          v-model="inputMessage"
          type="textarea"
          :rows="3"
          :placeholder="$t('customer.conversation.input_placeholder')"
          @keydown.enter.exact.prevent="handleSendMessage"
          @keydown.enter.shift.exact="handleNewLine"
        />
        <div class="send-actions">
          <el-button 
            type="primary" 
            @click="handleSendMessage"
            :disabled="!inputMessage.trim() || sending"
            :loading="sending"
          >
            {{ $t('common.send') }}
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox, ElIcon } from 'element-plus'
import { Loading, ArrowDown, Refresh, InfoFilled } from '@element-plus/icons-vue'
import { getConversationDetail, getMessages, sendMessage, endConversation } from '@/api/customer'
import ErrorHandler from '@/utils/errorHandler'
import Storage from '@/utils/storage'
import { useUserStore } from '@/store/user'

const props = defineProps({
  conversationId: {
    type: Number,
    required: true
  }
})

const emit = defineEmits(['close', 'ended'])

const { t } = useI18n()
const userStore = useUserStore()

const loading = ref(false)
const messagesLoading = ref(false)
const conversation = ref(null)
const messages = ref([])
const inputMessage = ref('')
const sending = ref(false)
const messagesContainerRef = ref(null)
const hasMoreMessages = ref(true)
const loadingMore = ref(false)
const currentPage = ref(1) // 当前页码
const totalMessages = ref(0) // 消息总数
const showScrollToBottom = ref(false) // 是否显示滚动到底部按钮
const isAtBottom = ref(true) // 用户是否在底部
const unreadCount = ref(0) // 未读新消息数量
const ending = ref(false) // 结束会话中
const ws = ref(null) // WebSocket 连接
const wsConnected = ref(false) // WebSocket 连接状态

// 获取会话详情
const loadConversation = async () => {
  loading.value = true
  try {
    const response = await getConversationDetail(props.conversationId)
    if (response.code === 200) {
      conversation.value = response.data
    }
  } catch (error) {
    ErrorHandler.handle(error)
  } finally {
    loading.value = false
  }
}

// 加载消息（使用conversation_id加载当前会话的消息）
const loadMessages = async (loadMore = false) => {
  if (loadingMore.value) return
  
  if (loadMore) {
    loadingMore.value = true
  } else {
    messagesLoading.value = true
  }
  
  try {
    // 构建请求参数 - 始终使用conversation_id加载当前会话的消息
    const page = loadMore ? currentPage.value + 1 : 1
    const params = {
      conversation_id: props.conversationId,
      page: page,
      page_size: 100
    }
    
    const response = await getMessages(params)
    if (response.code === 200) {
      const data = response.data
      const newMessages = data.messages || []
      totalMessages.value = data.total || 0
      
      // 判断是否还有更多消息（当前已加载的消息数小于总数）
      hasMoreMessages.value = messages.value.length + newMessages.length < totalMessages.value
      
      if (loadMore) {
        // 加载更多：将新消息添加到前面（因为分页是从旧到新）
        if (newMessages.length > 0) {
          // 加载更多时，用户肯定不在底部，明确标记
          isAtBottom.value = false
          showScrollToBottom.value = true
          
          const container = messagesContainerRef.value
          const oldScrollHeight = container?.scrollHeight || 0
          
          messages.value = [...newMessages, ...messages.value]
          currentPage.value = page
          
          // 保持滚动位置
          nextTick(() => {
            if (container) {
              const newScrollHeight = container.scrollHeight
              container.scrollTop = newScrollHeight - oldScrollHeight
              // 更新滚动状态
              handleScroll()
            }
          })
        }
      } else {
        // 首次加载
        messages.value = newMessages
        currentPage.value = 1
        // 首次加载后滚动到底部
        scrollToBottom()
        // 确保状态正确
        nextTick(() => {
          isAtBottom.value = true
          showScrollToBottom.value = false
        })
      }
    }
  } catch (error) {
    ErrorHandler.handle(error)
  } finally {
    messagesLoading.value = false
    loadingMore.value = false
  }
}

// 加载更多历史消息
const loadMoreMessages = async () => {
  if (!hasMoreMessages.value || loadingMore.value) return
  await loadMessages(true)
}

// 监听滚动，滚动到顶部时自动加载更多
const handleScroll = () => {
  if (!messagesContainerRef.value) return
  
  const container = messagesContainerRef.value
  const scrollTop = container.scrollTop
  const scrollHeight = container.scrollHeight
  const clientHeight = container.clientHeight
  
  // 判断是否在底部（距离底部50px以内认为在底部）
  const atBottom = scrollHeight - scrollTop - clientHeight < 50
  isAtBottom.value = atBottom
  // 只有在有消息且不在底部时才显示按钮
  showScrollToBottom.value = !atBottom && messages.value.length > 0
  
  // 滚动到底部时清除未读计数
  if (atBottom) {
    unreadCount.value = 0
  }
  
  // 当滚动到顶部附近时，自动加载更多
  if (scrollTop < 50 && hasMoreMessages.value && !loadingMore.value) {
    loadMoreMessages()
  }
}

// 发送消息
const handleSendMessage = async () => {
  if (!inputMessage.value.trim() || sending.value) return

  const content = inputMessage.value.trim()
  inputMessage.value = ''
  sending.value = true

  // 添加临时消息（带loading状态）
  const tempMessageId = 'temp_' + Date.now()
  const tempMessage = {
    id: tempMessageId,
    sender_type: 'admin',
    content: content,
    type: 'text',
    created_at: new Date().toISOString(),
    is_sending: true // 标记为发送中
  }
  messages.value.push(tempMessage)
  scrollToBottom()

  try {
    const response = await sendMessage({
      conversation_id: props.conversationId,
      content: content,
      type: 'text'
    })

    if (response.code === 200) {
      // 找到临时消息并替换为真实消息
      const tempIndex = messages.value.findIndex(m => m.id === tempMessageId)
      if (tempIndex !== -1) {
        if (response.data) {
          // 用真实消息替换临时消息
          messages.value[tempIndex] = response.data
        } else {
          // 如果没有返回消息，移除临时消息并重新加载
          messages.value.splice(tempIndex, 1)
          await loadMessages()
        }
        scrollToBottom()
      } else {
        // 临时消息不存在，检查是否已添加
        if (response.data) {
          const exists = messages.value.some(m => m.id === response.data.id)
          if (!exists) {
            messages.value.push(response.data)
            scrollToBottom()
          }
        }
      }
    }
  } catch (error) {
    // 发送失败，移除临时消息
    const tempIndex = messages.value.findIndex(m => m.id === tempMessageId)
    if (tempIndex !== -1) {
      messages.value.splice(tempIndex, 1)
    }
    // 恢复输入框内容
    inputMessage.value = content
    ErrorHandler.handle(error)
  } finally {
    sending.value = false
  }
}

// 换行处理
const handleNewLine = () => {
  // Shift+Enter 换行
}

// 滚动到底部
const scrollToBottom = () => {
  nextTick(() => {
    if (messagesContainerRef.value) {
      messagesContainerRef.value.scrollTop = messagesContainerRef.value.scrollHeight
      isAtBottom.value = true
      showScrollToBottom.value = false
    }
  })
}

// 滚动到底部并清除未读
const scrollToBottomAndClearUnread = () => {
  scrollToBottom()
  unreadCount.value = 0
}

// 结束会话
const handleEndConversation = async () => {
  try {
    await ElMessageBox.confirm(
      t('customer.conversation.end_confirm'),
      t('common.confirm'),
      { type: 'warning' }
    )
    
    ending.value = true
    const response = await endConversation({ conversation_id: props.conversationId })
    
    if (response.code === 200) {
      ElMessage.success(t('customer.conversation.end_success'))
      // 更新本地状态
      if (conversation.value) {
        conversation.value.status = 2
      }
      // 通知父组件
      emit('ended')
    }
  } catch (error) {
    if (error !== 'cancel') {
      ErrorHandler.handle(error)
    }
  } finally {
    ending.value = false
  }
}

// 获取状态文本
const getStatusText = (status) => {
  const statusMap = {
    1: t('customer.conversation.status_active'),
    2: t('customer.conversation.status_ended'),
    3: t('customer.conversation.status_closed')
  }
  return statusMap[status] || '-'
}

// 获取状态类型
const getStatusType = (status) => {
  if (status == null || status === undefined) {
    return undefined // 返回 undefined 而不是空字符串
  }
  const typeMap = {
    1: 'success',
    2: 'info',
    3: 'danger'
  }
  return typeMap[status] || undefined // 返回 undefined 而不是空字符串
}

// 格式化时间（数据库存储的是 UTC，转换为客户端本地时间显示）
const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  
  let date
  try {
    if (typeof timeStr === 'string') {
      // 如果时间字符串格式是 "YYYY-MM-DD HH:mm:ss"（MySQL datetime 格式）
      // 数据库存储的是 UTC 时间，需要明确指定为 UTC
      if (timeStr.match(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}(\.\d+)?$/)) {
        // MySQL datetime 格式，数据库存储的是 UTC，添加 'Z' 后缀表示 UTC
        date = new Date(timeStr.replace(' ', 'T') + 'Z')
      } else if (timeStr.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/)) {
        // ISO UTC 格式（以 Z 结尾），直接解析
        date = new Date(timeStr)
      } else if (timeStr.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?$/)) {
        // ISO 格式但没有 Z，假设是 UTC 时间，添加 Z
        date = new Date(timeStr + 'Z')
      } else {
        // 其他格式，尝试直接解析
        date = new Date(timeStr)
      }
    } else {
      date = new Date(timeStr)
    }
    
    if (isNaN(date.getTime())) {
      return '-'
    }
  } catch (e) {
    return '-'
  }
  
  // date 对象已经是本地时间（解析 UTC 后自动转换为本地时间）
  // 直接显示具体时间：YYYY/MM/DD HH:mm:ss
  return date.toLocaleString('zh-CN', { 
    year: 'numeric',
    month: '2-digit', 
    day: '2-digit', 
    hour: '2-digit', 
    minute: '2-digit',
    second: '2-digit'
  })
}

// WebSocket 连接
const connectWebSocket = () => {
  if (ws.value || !props.conversationId) return

  const token = Storage.getItem('token', '')
  if (!token) {
    console.warn('No token found, cannot connect WebSocket')
    return
  }

  // 构建 WebSocket URL
  // 在开发环境中使用相对路径（通过 Vite 代理），在生产环境中使用完整 URL
  const isDev = import.meta.env.DEV
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL || 'http://127.0.0.1:3000'
  const wsBaseURL = import.meta.env.VITE_WS_BASE_URL || apiBaseURL
  
  // JWT 中间件期望的参数名是 _token
  const tokenParam = encodeURIComponent(token.trim())
  const conversationParam = props.conversationId
  
  let wsUrl
  if (isDev) {
    // 开发环境：使用相对路径，通过 Vite 代理
    wsUrl = `/ws/admin/customer?conversation_id=${conversationParam}&_token=${tokenParam}`
  } else if (wsBaseURL.startsWith('wss://') || wsBaseURL.startsWith('ws://')) {
    // 生产环境：使用配置的 WebSocket URL
    wsUrl = `${wsBaseURL}/ws/admin/customer?conversation_id=${conversationParam}&_token=${tokenParam}`
  } else if (wsBaseURL.startsWith('https://')) {
    wsUrl = wsBaseURL.replace('https://', 'wss://') + `/ws/admin/customer?conversation_id=${conversationParam}&_token=${tokenParam}`
  } else if (wsBaseURL.startsWith('http://')) {
    wsUrl = wsBaseURL.replace('http://', 'ws://') + `/ws/admin/customer?conversation_id=${conversationParam}&_token=${tokenParam}`
  } else {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    wsUrl = `${protocol}//${wsBaseURL}/ws/admin/customer?conversation_id=${conversationParam}&_token=${tokenParam}`
  }

  try {
    ws.value = new WebSocket(wsUrl)
    
    ws.value.onopen = () => {
      wsConnected.value = true
      console.log('WebSocket connected for conversation', props.conversationId)
    }
    
    ws.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        handleWebSocketMessage(message)
      } catch (error) {
        console.error('Failed to parse WebSocket message:', error)
      }
    }
    
    ws.value.onclose = () => {
      wsConnected.value = false
      ws.value = null
      console.log('WebSocket disconnected')
      // 尝试重连
      setTimeout(() => {
        if (props.conversationId) {
          connectWebSocket()
        }
      }, 3000)
    }
    
    ws.value.onerror = (error) => {
      console.error('WebSocket error:', error)
      wsConnected.value = false
    }
  } catch (error) {
    console.error('Failed to create WebSocket connection:', error)
  }
}

// 处理 WebSocket 消息
const handleWebSocketMessage = (message) => {
  // 处理系统消息
  if (message.type === 'system') {
    if (message.event === 'ended') {
      if (conversation.value) {
        conversation.value.status = 2
      }
      ElMessage.info('会话已结束')
      emit('ended')
    }
    return
  }
  
  // 处理普通消息
  if (message.conversation_id === props.conversationId) {
    // 如果是自己发送的消息（admin），完全忽略 WebSocket 消息
    // 因为我们已经通过 API 响应添加了消息，避免重复显示
    if (message.sender_type === 'admin' && userStore.adminInfo) {
      const currentAdminId = userStore.adminInfo.id
      // 确保 ID 类型一致（可能是 number 或 string）
      if (Number(message.sender_id) === Number(currentAdminId)) {
        // 自己发送的消息，完全忽略 WebSocket 消息
        return
      }
    }
    
    // 检查消息是否已存在（避免重复）
    const exists = messages.value.some(m => m.id === message.message_id)
    if (!exists) {
      // 将 WebSocket 消息格式转换为标准格式
      const newMessage = {
        id: message.message_id,
        conversation_id: message.conversation_id,
        sender_type: message.sender_type,
        sender_id: message.sender_id,
        content: message.content,
        type: message.type,
        file_url: message.file_url || '',
        file_name: message.file_name || '',
        file_size: message.file_size || 0,
        is_read: false,
        created_at: new Date(message.timestamp * 1000).toISOString()
      }
      
      messages.value.push(newMessage)
      
      // 如果用户不在底部，增加未读计数
      if (!isAtBottom.value) {
        unreadCount.value++
      }
      
      // 如果用户在底部，自动滚动到底部
      if (isAtBottom.value) {
        scrollToBottom()
      }
      
      // 更新消息总数
      totalMessages.value++
    }
  }
}

// 断开 WebSocket 连接
const disconnectWebSocket = () => {
  if (ws.value) {
    ws.value.close()
    ws.value = null
    wsConnected.value = false
  }
}

onMounted(async () => {
  await loadConversation()
  await loadMessages()
  // 连接 WebSocket 以接收实时消息
  connectWebSocket()
})

onUnmounted(() => {
  // 断开 WebSocket 连接
  disconnectWebSocket()
})

// 监听 conversationId 变化，重新连接 WebSocket
watch(() => props.conversationId, (newId, oldId) => {
  if (newId !== oldId) {
    disconnectWebSocket()
    if (newId) {
      connectWebSocket()
    }
  }
})

// 监听消息变化，只有在用户已经在底部时才自动滚动
watch(messages, () => {
  if (isAtBottom.value) {
    scrollToBottom()
  }
}, { deep: true })
</script>

<style scoped>
.conversation-detail {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.loading-container {
  padding: 20px;
}

.detail-content {
  display: flex;
  flex-direction: column;
  height: 100%;
}

/* 顶部工具栏 */
.chat-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}

.visitor-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.visitor-meta {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.visitor-name {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* 消息区域 */
.messages-section {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0; /* 确保 flex 子元素可以收缩 */
}

.messages-wrapper {
  position: relative;
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  min-height: 0; /* 确保 flex 子元素可以收缩 */
}

.messages-container {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 15px;
  background: #f5f7fa;
  min-height: 0; /* 确保 flex 子元素可以收缩 */
  /* 自定义滚动条样式 */
  scrollbar-width: thin;
  scrollbar-color: rgba(0, 0, 0, 0.2) transparent;
}

.messages-container::-webkit-scrollbar {
  width: 6px;
}

.messages-container::-webkit-scrollbar-track {
  background: transparent;
}

.messages-container::-webkit-scrollbar-thumb {
  background-color: rgba(0, 0, 0, 0.2);
  border-radius: 3px;
}

.messages-container::-webkit-scrollbar-thumb:hover {
  background-color: rgba(0, 0, 0, 0.3);
}

.scroll-to-bottom-btn {
  position: absolute;
  bottom: 20px;
  right: 20px;
  z-index: 100;
  cursor: pointer;
  transition: all 0.3s ease;
  display: flex;
  align-items: center;
  gap: 8px;
  background: #fff;
  padding: 4px 8px 4px 4px;
  border-radius: 20px;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
}

.scroll-to-bottom-btn:hover {
  transform: translateY(-2px);
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
}

.unread-text {
  font-size: 12px;
  color: #409eff;
  white-space: nowrap;
  font-weight: 500;
}

.load-more {
  display: flex;
  justify-content: center;
  padding: 10px 0;
  margin-bottom: 10px;
}

.empty-messages {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}

.message-item {
  display: flex;
  margin-bottom: 15px;
  align-items: flex-start;
  gap: 10px;
}

.message-item.is-visitor {
  flex-direction: row-reverse;
}

.message-content {
  flex: 1;
  max-width: 70%;
}

.message-bubble {
  padding: 10px 15px;
  border-radius: 8px;
  word-wrap: break-word;
}

.visitor-bubble {
  background: #409eff;
  color: #fff;
  border-bottom-right-radius: 2px;
}

.admin-bubble {
  background: #fff;
  color: #303133;
  border: 1px solid #e4e7ed;
  border-bottom-left-radius: 2px;
}

.message-text {
  font-size: 14px;
  line-height: 1.5;
  margin-bottom: 4px;
}

.message-time {
  font-size: 12px;
  opacity: 0.7;
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 4px;
}

.sending-icon {
  animation: rotate 1s linear infinite;
}

@keyframes rotate {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.send-message-section {
  padding: 12px 16px;
  background: #fff;
  border-top: 1px solid #e4e7ed;
}

.send-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: 10px;
}

/* 会话已结束提示 */
.conversation-ended-tip {
  padding: 20px;
  text-align: center;
  background: #fafafa;
  color: #909399;
}
</style>

