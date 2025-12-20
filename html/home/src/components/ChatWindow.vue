<template>
  <div class="chat-container">
    <!-- 聊天窗口 -->
    <div class="chat-window" :class="{ 'mobile': isMobile }">
      <!-- 头部 -->
      <div class="chat-header">
        <div class="header-info">
          <el-avatar :size="40" :src="adminInfo?.avatar" v-if="adminInfo">
            {{ adminInfo.nickname?.charAt(0) || '客' }}
          </el-avatar>
          <div class="header-text">
            <div class="admin-name">{{ adminInfo?.nickname || '客服' }}</div>
            <div class="status-text" :class="{ 'online': isConnected, 'reconnecting': isReconnecting }">
              {{ connectionStatusText }}
            </div>
          </div>
        </div>
        <el-button 
          text 
          :icon="isMobile ? 'Close' : 'Minus'" 
          @click="handleClose"
          class="close-btn"
        />
      </div>

      <!-- 消息列表容器 -->
      <div class="chat-messages-wrapper">
        <div class="chat-messages" ref="messagesRef" @scroll="handleScroll">
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
        
        <div v-if="messages.length === 0 && !loadingMore" class="empty-message">
          <el-empty description="暂无消息" :image-size="100" />
        </div>
        <div
          v-for="message in messages"
          :key="message.id"
          class="message-item"
          :class="{ 'is-visitor': message.sender_type === 'visitor', 'is-admin': message.sender_type === 'admin', 'is-system': message.type === 'system' }"
        >
          <!-- 系统消息 -->
          <div v-if="message.type === 'system'" class="system-message">
            {{ getSystemMessageText(message) }}
          </div>
          
          <!-- 普通消息 -->
          <template v-else>
            <el-avatar 
              v-if="message.sender_type === 'admin' && adminInfo"
              :size="32" 
              :src="adminInfo.avatar"
              class="message-avatar"
            >
              {{ adminInfo.nickname?.charAt(0) || '客' }}
            </el-avatar>
            <div class="message-content">
              <div class="message-bubble" :class="{ 'visitor-bubble': message.sender_type === 'visitor', 'admin-bubble': message.sender_type === 'admin' }">
                <!-- 图片消息 -->
                <div v-if="message.type === 'image' && message.file_url" class="message-image">
                  <el-image
                    :src="getMessageImageUrl(message)"
                    :preview-src-list="[getMessageImageUrl(message)]"
                    fit="cover"
                    style="max-width: 200px; max-height: 200px; border-radius: 4px; cursor: pointer;"
                    @load="handleImageLoad"
                    @error="handleImageError"
                  />
                </div>
                <!-- 文本消息 -->
                <div v-if="message.type === 'text'" class="message-text" v-html="renderMessage(message.content)"></div>
                <div class="message-time">
                  <el-icon v-if="message.is_sending" class="sending-icon"><Loading /></el-icon>
                  {{ formatRelativeTime(message.created_at) }}
                </div>
              </div>
            </div>
            <el-avatar 
              v-if="message.sender_type === 'visitor'"
              :size="32" 
              class="message-avatar visitor-avatar"
            >
              {{ visitorInfo?.name?.charAt(0) || '访' }}
            </el-avatar>
          </template>
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

      <!-- 输入区域 -->
      <div class="chat-input">
        <!-- 会话结束提示 -->
        <div v-if="isConversationEnded" class="conversation-ended">
          <el-alert
            title="会话已结束"
            type="info"
            :closable="false"
            show-icon
          >
            <template #default>
              <el-button type="primary" size="small" @click="startNewConversation">
                发起新会话
              </el-button>
            </template>
          </el-alert>
        </div>
        
        <!-- 正常输入 -->
        <template v-else>
          <div class="input-wrapper">
            <el-input
              v-model="inputMessage"
              type="textarea"
              :rows="isMobile ? 2 : 3"
              placeholder="请输入消息..."
              @keydown.enter.exact.prevent="handleSendMessage"
              @keydown.enter.shift.exact="handleNewLine"
              :disabled="!isConnected || !conversationId"
              class="input-textarea"
            />
            <div class="input-toolbar">
              <el-upload
                :action="uploadAction"
                :before-upload="beforeUpload"
                :on-success="handleUploadSuccess"
                :on-error="handleUploadError"
                :show-file-list="false"
                :multiple="false"
                accept="image/*"
              >
                <template #trigger>
                  <el-button text circle size="small" :icon="Picture" />
                </template>
              </el-upload>
              <EmojiPicker @select="handleEmojiSelect" />
            </div>
          </div>
          <div class="input-actions">
            <el-button 
              type="primary" 
              :icon="'Position'"
              @click="handleSendMessage"
              :disabled="!inputMessage.trim() || !isConnected || !conversationId"
              :loading="sending"
            >
              发送
            </el-button>
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { ElMessage, ElIcon } from 'element-plus'
import { Loading, ArrowDown, Picture } from '@element-plus/icons-vue'
import axios from 'axios'
import EmojiPicker from './EmojiPicker.vue'

// 简单的 i18n 对象
const i18n = {
  'zh-CN': {
    customerService: '客服',
    loading: '加载中...',
    loadMore: '加载更多历史消息',
    noMessages: '暂无消息',
    newMessages: '条新消息',
    adminConnected: (name) => `客服 ${name || ''} 已接入`,
    conversationResumed: (name) => `会话已恢复，客服 ${name || ''} 为您服务`,
    conversationEnded: '会话已结束',
    send: '发送',
    inputPlaceholder: '请输入消息...',
    visitor: '访客',
    visitorChar: '访',
    adminChar: '客'
  },
  'en-US': {
    customerService: 'Customer Service',
    loading: 'Loading...',
    loadMore: 'Load More History',
    noMessages: 'No messages',
    newMessages: 'new messages',
    adminConnected: (name) => `Customer service ${name || ''} has joined`,
    conversationResumed: (name) => `Conversation resumed, ${name || ''} is serving you`,
    conversationEnded: 'Conversation ended',
    send: 'Send',
    inputPlaceholder: 'Please enter a message...',
    visitor: 'Visitor',
    visitorChar: 'V',
    adminChar: 'C',
    onlyImageAllowed: 'Only image files are allowed',
    uploadFailed: 'Upload failed'
  }
}

// 从 URL 参数获取语言
function getLanguage() {
  const urlParams = new URLSearchParams(window.location.search)
  const lang = urlParams.get('lang') || urlParams.get('language') || 'zh-CN'
  return lang.toLowerCase() === 'en' || lang.toLowerCase() === 'en-us' ? 'en-US' : 'zh-CN'
}

const currentLang = getLanguage()
const t = (key, ...args) => {
  const translations = i18n[currentLang] || i18n['zh-CN']
  const value = translations[key]
  return typeof value === 'function' ? value(...args) : value
}

// Props
const props = defineProps({
  apiBaseUrl: {
    type: String,
    default: () => {
      // 优先使用环境变量配置（.env 文件），不再从 URL 参数获取
      return import.meta.env.VITE_API_BASE_URL || 
             window.CUSTOMER_SERVICE_API_URL || 
             'http://127.0.0.1:3000'
    }
  },
  visitorId: {
    type: String,
    default: ''
  },
  adminId: {
    type: String,
    default: ''
  },
  conversationId: {
    type: String,
    default: ''
  },
  language: {
    type: String,
    default: 'zh-CN'
  }
})

// 响应式数据
const isMobile = ref(window.innerWidth <= 768)
const visitorId = ref(props.visitorId || getVisitorIdFromStorage() || generateVisitorId())
const conversationId = ref(props.conversationId || getConversationIdFromStorage() || '')
const adminId = ref(props.adminId || '')
const messages = ref([])
const inputMessage = ref('')
const sending = ref(false)
const isConnected = ref(false)
const ws = ref(null)
const messagesRef = ref(null)
const visitorInfo = ref(null)
const adminInfo = ref(null)
const isConversationEnded = ref(false)
const hasMoreMessages = ref(true)
const loadingMore = ref(false)
const oldestMessageId = ref(0) // 用于游标分页
const processedMessageIds = new Set() // 已处理的消息ID集合，防止重复
const pendingMessages = new Map() // 正在发送的消息 Map<tempMessageId, content>
const showScrollToBottom = ref(false) // 是否显示滚动到底部按钮
const isAtBottom = ref(true) // 用户是否在底部（用于判断是否需要自动滚动）
const unreadCount = ref(0) // 未读新消息数量
const reconnectCount = ref(0) // 重连次数
const isReconnecting = ref(false) // 是否正在重连
const maxReconnectAttempts = 10 // 最大重连次数
const heartbeatInterval = ref(null) // 心跳定时器
const heartbeatIntervalTime = 30000 // 心跳间隔：30秒
let audioContext = null // 音频上下文
const uploadingFile = ref(null) // 正在上传的文件
const imageUrlMap = ref(new Map()) // 图片URL缓存（message_id -> blob URL）

// 计算上传地址（使用访客专用的上传接口，会自动添加水印）
const uploadAction = computed(() => {
  return `${props.apiBaseUrl}/api/visitor/upload/image`
})

// 连接状态文本
const connectionStatusText = computed(() => {
  if (isConnected.value) return '在线'
  if (isReconnecting.value) return `重连中 (${reconnectCount.value}/${maxReconnectAttempts})`
  return '连接中...'
})

// 监听窗口大小变化
window.addEventListener('resize', () => {
  isMobile.value = window.innerWidth <= 768
})

// 生成访客ID
function generateVisitorId() {
  const timestamp = Date.now()
  const random = Math.random().toString(36).substring(2, 15)
  return 'visitor_' + timestamp + '_' + random
}

// 从存储获取访客ID
function getVisitorIdFromStorage() {
  return localStorage.getItem('customer_service_visitor_id')
}

// 保存访客ID到存储
function saveVisitorIdToStorage(id) {
  localStorage.setItem('customer_service_visitor_id', id)
}

// 从存储获取会话ID
function getConversationIdFromStorage() {
  return localStorage.getItem('customer_service_conversation_id')
}

// 保存会话ID到存储
function saveConversationIdToStorage(id) {
  if (id) {
    localStorage.setItem('customer_service_conversation_id', id)
  }
}

// 初始化
onMounted(async () => {
  saveVisitorIdToStorage(visitorId.value)
  
  // 注册访客
  await registerVisitor()
  
  // 统一调用 createConversation
  // 后端会检查是否有进行中的会话，有则返回现有会话，无则创建新会话
  await createConversation()
  
  // 初始化后检查滚动位置
  nextTick(() => {
    if (messagesRef.value) {
      handleScroll()
    }
  })
})

// 结束会话
async function endConversation() {
  if (!conversationId.value || isConversationEnded.value) {
    return
  }
  
  try {
    await axios.post(`${props.apiBaseUrl}/api/visitor/conversations/end`, {
      visitor_id: visitorId.value,
      conversation_id: conversationId.value
    }, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      transformRequest: [(data) => {
        let params = new URLSearchParams()
        for (let key in data) {
          params.append(key, data[key])
        }
        return params.toString()
      }]
    })
    
    isConversationEnded.value = true
    localStorage.removeItem('customer_service_conversation_id')
  } catch (error) {
    // 静默失败，不显示错误提示（因为可能是页面关闭导致的）
    console.log('结束会话失败:', error)
  }
}

onUnmounted(() => {
  // 清理资源
  stopHeartbeat() // 停止心跳
  disconnectWebSocket() // 断开WebSocket
  
  // 清理所有 blob URL，避免内存泄漏
  imageUrlMap.value.forEach((blobUrl) => {
    if (blobUrl && blobUrl.startsWith('blob:')) {
      URL.revokeObjectURL(blobUrl)
    }
  })
  imageUrlMap.value.clear()
  
  // 结束会话
  endConversation()
})

// 监听页面关闭/离开事件
window.addEventListener('beforeunload', () => {
  // 使用 sendBeacon 确保请求能够发送（即使页面正在关闭）
  if (conversationId.value && !isConversationEnded.value) {
    const data = new URLSearchParams({
      visitor_id: visitorId.value,
      conversation_id: conversationId.value
    })
    navigator.sendBeacon(
      `${props.apiBaseUrl}/api/visitor/conversations/end`,
      data.toString()
    )
  }
})

// 当前页面可见性状态（根据实际状态初始化）
const isPageVisible = ref(!document.hidden)

// 监听页面隐藏事件（切换标签页等）
document.addEventListener('visibilitychange', () => {
  isPageVisible.value = !document.hidden
  
  if (conversationId.value && !isConversationEnded.value && visitorId.value) {
    // 页面可见性变化时，发送心跳更新状态
    sendHeartbeat(document.hidden ? 'away' : 'online')
  }
})

// 发送心跳（更新访客状态）
async function sendHeartbeat(status = 'online') {
  if (!visitorId.value) return
  
  try {
    await axios.post(`${props.apiBaseUrl}/api/visitor/heartbeat`, {
      visitor_id: visitorId.value,
      conversation_id: conversationId.value || '',
      status: status
    }, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      transformRequest: [(data) => {
        let params = new URLSearchParams()
        for (let key in data) {
          params.append(key, data[key])
        }
        return params.toString()
      }]
    })
  } catch (error) {
    // 心跳失败静默处理
    console.log('心跳发送失败:', error)
  }
}

// 启动心跳
function startHeartbeat() {
  stopHeartbeat() // 先停止之前的心跳
  
  // 立即发送一次心跳
  sendHeartbeat(isPageVisible.value ? 'online' : 'away')
  
  // 定时发送心跳
  heartbeatInterval.value = setInterval(() => {
    sendHeartbeat(isPageVisible.value ? 'online' : 'away')
  }, heartbeatIntervalTime)
}

// 停止心跳
function stopHeartbeat() {
  if (heartbeatInterval.value) {
    clearInterval(heartbeatInterval.value)
    heartbeatInterval.value = null
  }
}

// 注册访客
async function registerVisitor() {
  try {
    const response = await axios.post(`${props.apiBaseUrl}/api/visitor/register`, {
      visitor_id: visitorId.value,
      source: window.location.href,
      referer: document.referrer
    }, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      transformRequest: [(data) => {
        let params = new URLSearchParams()
        for (let key in data) {
          params.append(key, data[key])
        }
        return params.toString()
      }]
    })
    
    if (response.data.code === 200) {
      visitorInfo.value = response.data.data
    }
  } catch (error) {
    console.error('注册访客失败:', error)
  }
}

// 创建会话（如果有进行中的会话，后端会返回现有会话）
async function createConversation() {
  try {
    const params = {
      visitor_id: visitorId.value,
      title: document.title || '新会话'
    }
    
    if (adminId.value) {
      params.admin_id = adminId.value
    }
    
    const response = await axios.post(`${props.apiBaseUrl}/api/visitor/conversations`, params, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      transformRequest: [(data) => {
        let params = new URLSearchParams()
        for (let key in data) {
          params.append(key, data[key])
        }
        return params.toString()
      }]
    })
    
    if (response.data.code === 200) {
      const conversation = response.data.data
      conversationId.value = conversation.id
      
      // 保存会话ID到本地存储
      saveConversationIdToStorage(conversation.id)
      
      if (conversation.admin) {
        adminInfo.value = conversation.admin
      }
      
      // 连接 WebSocket
      connectWebSocket()
      
      // 加载历史消息
      await loadMessages()
    }
  } catch (error) {
    console.error('创建会话失败:', error)
    ElMessage.error('创建会话失败，请刷新重试')
  }
}

// 加载消息（游标分页，基于消息ID）
async function loadMessages(loadMore = false) {
  if (!visitorId.value) return
  if (loadingMore.value) return
  
  try {
    if (loadMore) {
      loadingMore.value = true
    }
    
    // 游标分页参数
    const params = {
      visitor_id: visitorId.value,
      limit: 30
    }
    
    // 加载更多时，传入最早消息的ID
    if (loadMore && oldestMessageId.value > 0) {
      params.before_id = oldestMessageId.value
    }
    
    const response = await axios.get(`${props.apiBaseUrl}/api/visitor/messages/all`, { params })
    
    if (response.data.code === 200) {
      const data = response.data.data
      const newMessages = data.messages || []
      hasMoreMessages.value = data.has_more
      
      if (loadMore) {
        // 加载更多：将新消息添加到前面
        if (newMessages.length > 0) {
          // 加载更多时，用户肯定不在底部，明确标记
          isAtBottom.value = false
          showScrollToBottom.value = true
          
          // 记录当前滚动位置
          const container = messagesRef.value
          const oldScrollHeight = container?.scrollHeight || 0
          
          // 过滤掉已存在的消息（防止重复）
          const filteredMessages = newMessages.filter(msg => {
            if (msg.id && processedMessageIds.has(msg.id)) {
              return false
            }
            if (msg.id) {
              processedMessageIds.add(msg.id)
            }
            return true
          })
          
          messages.value = [...filteredMessages, ...messages.value]
          
          // 为图片消息加载图片
          filteredMessages.forEach(msg => {
            if (msg.type === 'image' && msg.file_url) {
              loadMessageImage(msg)
            }
          })
          
          // 更新最早消息ID（新加载的消息中第一个是最早的）
          if (filteredMessages.length > 0) {
            oldestMessageId.value = filteredMessages[0].id
          }
          
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
        // 记录所有消息ID，防止重连时重复
        newMessages.forEach(msg => {
          if (msg.id) {
            processedMessageIds.add(msg.id)
          }
        })
        
        messages.value = newMessages
        
        // 为图片消息加载图片（统一字段名）
        newMessages.forEach(msg => {
          // 统一文件相关字段名（兼容大小写）
          if (!msg.file_url && msg.FileURL) {
            msg.file_url = msg.FileURL
          }
          if (!msg.file_name && msg.FileName) {
            msg.file_name = msg.FileName
          }
          if (!msg.file_size && msg.FileSize) {
            msg.file_size = msg.FileSize
          }
          
          if (msg.type === 'image' && msg.file_url) {
            loadMessageImage(msg)
          }
        })
        
        // 记录最早消息的ID（数组第一个元素是最早的）
        if (newMessages.length > 0) {
          oldestMessageId.value = newMessages[0].id
        } else {
          oldestMessageId.value = 0
        }
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
    console.error('加载消息失败:', error)
  } finally {
    loadingMore.value = false
  }
}

// 加载更多历史消息
async function loadMoreMessages() {
  if (!hasMoreMessages.value || loadingMore.value) return
  await loadMessages(true)
}

// 连接 WebSocket
function connectWebSocket() {
  if (!conversationId.value) {
    isReconnecting.value = false
    return
  }
  
  // 如果已经有连接且正在连接中，直接返回
  if (ws.value && (ws.value.readyState === WebSocket.CONNECTING || ws.value.readyState === WebSocket.OPEN)) {
    return
  }
  
  // 注意：不在这里检查 isReconnecting，因为重连时需要重置它
  
  // 关闭旧连接（如果存在）
  if (ws.value) {
    try {
      ws.value.onclose = null // 移除旧的 onclose 处理器，避免触发重连
      ws.value.close()
    } catch (e) {
      console.error('关闭旧连接失败:', e)
    }
    ws.value = null
  }
  
  const wsUrl = props.apiBaseUrl.replace(/^http/, 'ws') + '/api/visitor/ws'
  const url = `${wsUrl}?visitor_id=${encodeURIComponent(visitorId.value)}&conversation_id=${conversationId.value}`
  
  try {
    ws.value = new WebSocket(url)
    
    ws.value.onopen = () => {
      isConnected.value = true
      isReconnecting.value = false
      // 只有在重连成功时才显示提示
      if (reconnectCount.value > 0) {
        ElMessage.success('重新连接成功')
      }
      reconnectCount.value = 0
      // 启动心跳
      startHeartbeat()
    }
    
    ws.value.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data)
        handleWebSocketMessage(message)
      } catch (e) {
        console.error('解析消息失败:', e)
      }
    }
    
    ws.value.onerror = (error) => {
      console.error('WebSocket错误:', error)
      // 连接错误时，标记为断开，但不立即重连（等待 onclose 事件）
      isConnected.value = false
    }
    
    ws.value.onclose = (event) => {
      isConnected.value = false
      ws.value = null // 清除连接引用
      
      // 如果会话已结束，不重连
      if (isConversationEnded.value) {
        isReconnecting.value = false
        return
      }
      
      // 达到最大重连次数，停止重连
      if (reconnectCount.value >= maxReconnectAttempts) {
        isReconnecting.value = false
        ElMessage.error('连接已断开，请刷新页面重试')
        return
      }
      
      // 递增重连次数，使用指数退避
      reconnectCount.value++
      const delay = Math.min(1000 * Math.pow(1.5, reconnectCount.value - 1), 30000)
      
      console.log(`WebSocket连接关闭，${delay}ms后尝试重连 (${reconnectCount.value}/${maxReconnectAttempts})`)
      
      // 设置重连状态
      isReconnecting.value = true
      
      setTimeout(() => {
        // 再次检查条件，避免在延迟期间状态发生变化
        if (conversationId.value && !isConversationEnded.value && !isConnected.value) {
          // 重置重连状态，允许创建新连接
          isReconnecting.value = false
          connectWebSocket()
        } else {
          isReconnecting.value = false
        }
      }, delay)
    }
  } catch (e) {
    console.error('WebSocket连接失败:', e)
    isConnected.value = false
    // 连接失败时也尝试重连
    if (reconnectCount.value < maxReconnectAttempts && !isConversationEnded.value) {
      reconnectCount.value++
      const delay = Math.min(1000 * Math.pow(1.5, reconnectCount.value - 1), 30000)
      isReconnecting.value = true
      
      console.log(`WebSocket连接失败，${delay}ms后尝试重连 (${reconnectCount.value}/${maxReconnectAttempts})`)
      
      setTimeout(() => {
        if (conversationId.value && !isConversationEnded.value && !isConnected.value) {
          // 重置重连状态，允许创建新连接
          isReconnecting.value = false
          connectWebSocket()
        } else {
          isReconnecting.value = false
        }
      }, delay)
    } else {
      isReconnecting.value = false
    }
  }
}

// 处理 WebSocket 消息
function handleWebSocketMessage(message) {
  if (message.type === 'system') {
    // 处理系统消息
    if (message.event === 'assigned' && message.data) {
      if (message.data.admin_id && !adminInfo.value) {
        // 可以在这里加载客服信息
      }
    }
    
    // 会话结束处理
    if (message.event === 'ended') {
      isConversationEnded.value = true
      // 清除本地存储的会话ID，下次打开会创建新会话
      localStorage.removeItem('customer_service_conversation_id')
      ElMessage.info('会话已结束，您可以发起新会话')
    }
    
    // 只添加需要显示的系统消息到消息列表（排除 visitor_status、connected、read 等内部事件）
    const displayableEvents = ['assigned', 'ended', 'reactivated']
    if (displayableEvents.includes(message.event)) {
      messages.value.push({
        id: Date.now(),
        type: 'system',
        event: message.event,
        data: message.data,
        created_at: new Date().toISOString()
      })
    }
    return // 系统消息处理完毕，直接返回
  } else {
    // 普通消息
    // 统一消息格式：将 WebSocket 消息格式转换为标准格式
    // 处理时间：timestamp 是 Unix 时间戳（秒），需要转换为 ISO 字符串
    let createdAt = message.created_at
    if (!createdAt && message.timestamp) {
      // timestamp 是 UTC 时间的 Unix 时间戳（秒），转换为 ISO 字符串
      // 添加 'Z' 后缀明确表示这是 UTC 时间
      const date = new Date(message.timestamp * 1000)
      createdAt = date.toISOString()
    } else if (!createdAt) {
      createdAt = new Date().toISOString()
    }
    
    const normalizedMessage = {
      ...message,
      // 使用 message_id 作为 id（如果存在）
      id: message.id || message.message_id,
      // 统一时间字段：确保是 ISO 格式的 UTC 时间字符串
      created_at: createdAt,
      // 统一文件相关字段名（兼容大小写）
      file_url: message.file_url || message.FileURL || message.fileUrl || '',
      file_name: message.file_name || message.FileName || message.fileName || '',
      file_size: message.file_size || message.FileSize || message.fileSize || 0
    }
    
    // 1. 检查消息ID是否已处理
    if (normalizedMessage.id && processedMessageIds.has(normalizedMessage.id)) {
      return
    }
    
    // 2. 检查消息ID是否已存在
    if (normalizedMessage.id) {
      const existsById = messages.value.some(m => 
        m.id && (String(m.id) === String(normalizedMessage.id) || Number(m.id) === Number(normalizedMessage.id))
      )
      if (existsById) {
        return
      }
    }
    
    // 3. 如果是访客消息，检查是否有匹配的待处理临时消息
    if (normalizedMessage.sender_type === 'visitor' && normalizedMessage.content) {
      // 遍历 pendingMessages 找到匹配的
      for (const [tempId, content] of pendingMessages.entries()) {
        if (content === normalizedMessage.content) {
          // 找到匹配的临时消息
          const tempIndex = messages.value.findIndex(m => m.id === tempId)
          if (tempIndex !== -1) {
            // 替换临时消息，确保保留时间字段
            messages.value[tempIndex] = normalizedMessage
            // 记录已处理
            if (normalizedMessage.id) {
              processedMessageIds.add(normalizedMessage.id)
            }
            // 从 pendingMessages 中移除
            pendingMessages.delete(tempId)
            // 只有在用户已经在底部时才自动滚动
            if (isAtBottom.value) {
              scrollToBottom()
            }
            return
          }
        }
      }
    }
    
    // 4. 没有匹配的临时消息，直接添加（来自其他客户端的消息）
    messages.value.push(normalizedMessage)
    if (normalizedMessage.id) {
      processedMessageIds.add(normalizedMessage.id)
    }
    
    // 如果是图片消息，加载图片
    if (normalizedMessage.type === 'image' && normalizedMessage.file_url) {
      loadMessageImage(normalizedMessage)
    }
    
    // 播放新消息提示音（只对客服消息播放）
    if (normalizedMessage.sender_type === 'admin') {
      playNotificationSound()
    }
    
    // 只有在用户已经在底部时才自动滚动
    if (isAtBottom.value) {
      scrollToBottom()
      unreadCount.value = 0
    } else {
      // 如果用户不在底部，显示滚动到底部按钮并增加未读计数
      showScrollToBottom.value = true
      // 只有来自客服的消息才计入未读（访客自己发的不算）
      if (normalizedMessage.sender_type === 'admin') {
        unreadCount.value++
      }
    }
  }
}

// 发送消息
async function handleSendMessage() {
  if (!inputMessage.value.trim() || !conversationId.value || sending.value) return
  
  const content = inputMessage.value.trim()
  inputMessage.value = ''
  sending.value = true

  // 添加临时消息（带loading状态）
  const tempMessageId = 'temp_' + Date.now()
  const tempMessage = {
    id: tempMessageId,
    sender_type: 'visitor',
    content: content,
    type: 'text',
    created_at: new Date().toISOString(),
    is_sending: true // 标记为发送中
  }
  messages.value.push(tempMessage)
  scrollToBottom()
  
  // 记录到 pendingMessages，供 WebSocket 匹配
  pendingMessages.set(tempMessageId, content)
  
  try {
    const response = await axios.post(`${props.apiBaseUrl}/api/visitor/messages`, {
      conversation_id: conversationId.value,
      visitor_id: visitorId.value,
      content: content,
      type: 'text'
    }, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      transformRequest: [(data) => {
        let params = new URLSearchParams()
        for (let key in data) {
          params.append(key, data[key])
        }
        return params.toString()
      }]
    })
    
    if (response.data.code === 200) {
      // 发送成功，等待 WebSocket 推送真实消息来替换临时消息
      // 设置超时：如果 5 秒后临时消息还在，用 API 返回的消息替换
      if (response.data.data && response.data.data.id) {
        const messageId = response.data.data.id
        setTimeout(() => {
          // 检查临时消息是否还存在
          const tempIndex = messages.value.findIndex(m => m.id === tempMessageId)
          if (tempIndex !== -1) {
            // 临时消息还在，说明 WebSocket 没有推送，用 API 返回的消息替换
            // 确保保留时间字段
            const finalMessage = {
              ...response.data.data,
              // 如果 API 返回的消息没有 created_at，保留临时消息的时间
              created_at: response.data.data.created_at || messages.value[tempIndex].created_at
            }
            processedMessageIds.add(messageId)
            messages.value[tempIndex] = finalMessage
            pendingMessages.delete(tempMessageId)
            // 只有在用户已经在底部时才自动滚动
            if (isAtBottom.value) {
              scrollToBottom()
            }
          }
        }, 5000)
      }
    } else {
      // 发送失败，移除临时消息
      const tempIndex = messages.value.findIndex(m => m.id === tempMessageId)
      if (tempIndex !== -1) {
        messages.value.splice(tempIndex, 1)
      }
      pendingMessages.delete(tempMessageId)
      // 恢复输入框内容
      inputMessage.value = content
      ElMessage.error(response.data.message || '发送失败')
    }
  } catch (error) {
    // 发送失败，移除临时消息
    const tempIndex = messages.value.findIndex(m => m.id === tempMessageId)
    if (tempIndex !== -1) {
      messages.value.splice(tempIndex, 1)
    }
    pendingMessages.delete(tempMessageId)
    // 恢复输入框内容
    inputMessage.value = content
    console.error('发送消息失败:', error)
    ElMessage.error('发送失败，请重试')
  } finally {
    sending.value = false
  }
}

// 播放新消息提示音
function playNotificationSound() {
  try {
    // 使用 Web Audio API 创建简单的提示音
    if (!audioContext) {
      audioContext = new (window.AudioContext || window.webkitAudioContext)()
    }
    
    const oscillator = audioContext.createOscillator()
    const gainNode = audioContext.createGain()
    
    oscillator.connect(gainNode)
    gainNode.connect(audioContext.destination)
    
    oscillator.frequency.value = 800 // 频率
    oscillator.type = 'sine'
    
    gainNode.gain.setValueAtTime(0.3, audioContext.currentTime)
    gainNode.gain.exponentialRampToValueAtTime(0.01, audioContext.currentTime + 0.3)
    
    oscillator.start(audioContext.currentTime)
    oscillator.stop(audioContext.currentTime + 0.3)
  } catch (e) {
    // 忽略音频播放错误
    console.log('无法播放提示音:', e)
  }
}

// 换行处理
function handleNewLine() {
  // Shift+Enter 换行，不做处理
}

// 滚动到底部
function scrollToBottom() {
  nextTick(() => {
    if (messagesRef.value) {
      messagesRef.value.scrollTop = messagesRef.value.scrollHeight
      isAtBottom.value = true
      showScrollToBottom.value = false
    }
  })
}

// 滚动到底部并清除未读
function scrollToBottomAndClearUnread() {
  scrollToBottom()
  unreadCount.value = 0
}

// 监听滚动，滚动到顶部时自动加载更多
function handleScroll() {
  if (!messagesRef.value) return
  
  const container = messagesRef.value
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

// 文件上传前处理
const beforeUpload = (file) => {
  // 验证文件类型，只允许图片
  if (!file.type.startsWith('image/')) {
    ElMessage.error(t('onlyImageAllowed') || '只能上传图片文件')
    return false
  }
  
  uploadingFile.value = file
  
  // 使用 axios 上传文件
  const formData = new FormData()
  formData.append('file', file)
  
  axios.post(uploadAction.value, formData, {
    headers: {
      'Content-Type': 'multipart/form-data'
    }
  }).then(response => {
    if (response.data.code === 200) {
      const attachment = response.data.data || response.data
      // 发送图片消息
      handleSendImageMessage(attachment)
    } else {
      ElMessage.error(response.data.message || t('uploadFailed') || '上传失败')
      uploadingFile.value = null
    }
  }).catch(error => {
    console.error('Upload error:', error)
    ElMessage.error(error.response?.data?.message || error.message || t('uploadFailed') || '上传失败')
    uploadingFile.value = null
  })
  
  return false // 阻止默认上传
}

// 处理上传成功
const handleUploadSuccess = (response) => {
  // Element Plus 的 on-success 回调，但我们已经用 beforeUpload 处理了
  uploadingFile.value = null
}

// 处理上传失败
const handleUploadError = (error) => {
  ElMessage.error(t('uploadFailed') || '上传失败')
  uploadingFile.value = null
}

// 发送图片消息
async function handleSendImageMessage(attachment) {
  if (!attachment) return
  
  // 兼容大小写字段名
  const fileUrl = attachment.file_url || attachment.FileURL || attachment.fileUrl
  const fileName = attachment.filename || attachment.Filename || attachment.fileName || ''
  const fileSize = attachment.size || attachment.Size || 0
  
  if (!fileUrl) {
    ElMessage.error(t('uploadFailed') || '上传失败：无法获取文件URL')
    uploadingFile.value = null
    return
  }
  
  sending.value = true
  
  try {
    const response = await axios.post(`${props.apiBaseUrl}/api/visitor/messages`, {
      conversation_id: conversationId.value,
      visitor_id: visitorId.value,
      content: '',
      type: 'image',
      file_url: fileUrl,
      file_name: fileName,
      file_size: fileSize
    }, {
      headers: {
        'Content-Type': 'application/x-www-form-urlencoded'
      },
      transformRequest: [(data) => {
        let params = new URLSearchParams()
        for (let key in data) {
          params.append(key, data[key])
        }
        return params.toString()
      }]
    })
    
    if (response.data.code === 200) {
      if (response.data.data) {
        const newMessage = response.data.data
        // 统一文件相关字段名（兼容大小写）
        if (!newMessage.file_url && newMessage.FileURL) {
          newMessage.file_url = newMessage.FileURL
        }
        if (!newMessage.file_name && newMessage.FileName) {
          newMessage.file_name = newMessage.FileName
        }
        if (!newMessage.file_size && newMessage.FileSize) {
          newMessage.file_size = newMessage.FileSize
        }
        
        const exists = messages.value.some(m => m.id === newMessage.id)
        if (!exists) {
          messages.value.push(newMessage)
          // 如果是图片消息，加载图片
          if (newMessage.type === 'image' && newMessage.file_url) {
            loadMessageImage(newMessage)
          }
          scrollToBottom()
        }
      } else {
        await loadMessages()
      }
    }
  } catch (error) {
    console.error('发送图片消息失败:', error)
    ElMessage.error('发送失败，请重试')
  } finally {
    sending.value = false
    uploadingFile.value = null
  }
}

// 加载消息中的图片并转换为blob URL
const loadMessageImage = async (message) => {
  if (!message || message.type !== 'image' || !message.file_url) return
  
  const messageId = message.id
  if (!messageId) return
  
  // 如果已经加载过，直接返回
  if (imageUrlMap.value.has(messageId)) {
    return
  }
  
  const fileUrl = message.file_url || message.FileURL
  if (!fileUrl) return
  
  // 如果是外部URL（http/https），直接使用
  if (fileUrl.startsWith('http://') || fileUrl.startsWith('https://')) {
    imageUrlMap.value.set(messageId, fileUrl)
    return
  }
  
  // 将 admin 路径转换为 visitor 路径（访客专用接口，不需要认证）
  let processedUrl = fileUrl
  if (fileUrl.startsWith('/api/admin/attachments/') && fileUrl.includes('/preview')) {
    processedUrl = fileUrl.replace('/api/admin/attachments/', '/api/visitor/attachments/')
  }
  
  // 构建完整的URL
  let fullUrl = processedUrl
  if (props.apiBaseUrl) {
    const base = props.apiBaseUrl.replace(/\/+$/, '')
    fullUrl = `${base}${processedUrl.startsWith('/') ? '' : '/'}${processedUrl}`
  }
  
  // 通过axios获取图片并转换为blob URL
  // 访客接口不需要认证
  try {
    const response = await axios.get(fullUrl, {
      responseType: 'blob'
    })
    const blob = new Blob([response.data])
    const blobUrl = URL.createObjectURL(blob)
    imageUrlMap.value.set(messageId, blobUrl)
  } catch (error) {
    console.error('Failed to load message image:', error)
    // 加载失败时设置为原始URL，让浏览器尝试直接加载
    imageUrlMap.value.set(messageId, fileUrl)
  }
}

// 获取消息图片URL
const getMessageImageUrl = (message) => {
  if (!message || message.type !== 'image' || !message.file_url) return ''
  
  const messageId = message.id
  if (!messageId) return message.file_url
  
  // 从缓存中获取
  const cachedUrl = imageUrlMap.value.get(messageId)
  if (cachedUrl) {
    return cachedUrl
  }
  
  // 如果还没有加载，异步加载
  loadMessageImage(message)
  
  // 返回原始URL作为占位符
  return message.file_url
}

// 图片加载成功
const handleImageLoad = () => {
  // 图片加载成功后，如果用户在底部，自动滚动到底部
  // 因为图片加载会导致容器高度变化，需要重新滚动
  if (isAtBottom.value) {
    nextTick(() => {
      scrollToBottom()
    })
  }
}

// 图片加载失败
const handleImageError = (event) => {
  console.error('Image load error:', event)
}

// 解析时间字符串（数据库存储的是 UTC 时间，需要转换为客户端本地时间）
function parseTime(timeStr) {
  if (!timeStr) return null
  
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
        // 其他格式，尝试直接解析（假设是 UTC）
        date = new Date(timeStr)
      }
    } else {
      date = new Date(timeStr)
    }
    
    if (isNaN(date.getTime())) {
      return null
    }
    // Date 对象会自动转换为本地时间，直接返回即可
    return date
  } catch (e) {
    console.error('时间解析失败:', timeStr, e)
    return null
  }
}

// 格式化时间（数据库存储的是 UTC，转换为客户端本地时间显示）
function formatRelativeTime(timeStr) {
  if (!timeStr) {
    return ''
  }
  
  const date = parseTime(timeStr)
  if (!date) {
    return ''
  }
  
  // date 对象已经是本地时间（parseTime 解析 UTC 后自动转换为本地时间）
  // toLocaleString 默认使用本地时区显示
  try {
    return date.toLocaleString('zh-CN', { 
      year: 'numeric',
      month: '2-digit', 
      day: '2-digit', 
      hour: '2-digit', 
      minute: '2-digit',
      second: '2-digit'
    })
  } catch (e) {
    console.error('时间格式化失败:', timeStr, e)
    return ''
  }
}

// 获取系统消息文本
function getSystemMessageText(message) {
  if (message.event === 'assigned') {
    return `客服 ${message.data?.admin_name || ''} 已接入`
  } else if (message.event === 'reactivated') {
    return `会话已恢复，客服 ${message.data?.admin_name || ''} 为您服务`
  } else if (message.event === 'ended') {
    return '会话已结束'
  } else if (message.event === 'connected') {
    return '连接成功'
  }
  return '系统消息'
}

// 表情映射表
const emojiMap = {
  '[微笑]': '😊',
  '[大笑]': '😃',
  '[开心]': '😄',
  '[笑哭]': '😂',
  '[眨眼]': '😉',
  '[色]': '😍',
  '[亲亲]': '😘',
  '[害羞]': '😳',
  '[得意]': '😎',
  '[酷]': '🆒',
  '[大哭]': '😭',
  '[流泪]': '😢',
  '[委屈]': '😞',
  '[难过]': '😔',
  '[抓狂]': '😤',
  '[发怒]': '😠',
  '[生气]': '😡',
  '[惊讶]': '😲',
  '[惊恐]': '😱',
  '[晕]': '😵',
  '[困]': '😴',
  '[思考]': '🤔',
  '[疑问]': '❓',
  '[闭嘴]': '🤐',
  '[嘘]': '🤫',
  '[鄙视]': '🙄',
  '[白眼]': '🙄',
  '[赞]': '👍',
  '[弱]': '👎',
  '[握手]': '🤝',
  '[胜利]': '✌️',
  '[OK]': '👌',
  '[爱心]': '❤️',
  '[玫瑰]': '🌹',
  '[礼物]': '🎁',
  '[蛋糕]': '🎂',
  '[咖啡]': '☕',
  '[啤酒]': '🍺',
  '[干杯]': '🥂',
  '[鼓掌]': '👏',
  '[加油]': '💪',
  '[抱拳]': '🙏',
  '[再见]': '👋',
  '[好的]': '✅',
  '[不行]': '❌',
  '[星星]': '⭐',
  '[月亮]': '🌙',
  '[太阳]': '☀️',
  '[彩虹]': '🌈',
  '[烟花]': '🎆',
  '[庆祝]': '🎉',
  '[红包]': '🧧',
  '[钱]': '💰',
  '[飞机]': '✈️',
  '[汽车]': '🚗',
  '[火车]': '🚂',
  '[轮船]': '🚢',
  '[自行车]': '🚲',
  '[电话]': '📞',
  '[邮件]': '📧',
  '[电脑]': '💻',
  '[手机]': '📱',
  '[音乐]': '🎵',
  '[电影]': '🎬',
  '[游戏]': '🎮',
  '[足球]': '⚽',
  '[篮球]': '🏀',
  '[乒乓球]': '🏓',
}

// 渲染消息内容，将表情代码转换为 emoji
function renderMessage(content) {
  if (!content) return ''
  
  // 转义 HTML 特殊字符
  let html = content
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;')
  
  // 将表情代码替换为 emoji
  for (const [code, emoji] of Object.entries(emojiMap)) {
    const regex = new RegExp(code.replace(/[\[\]]/g, '\\$&'), 'g')
    html = html.replace(regex, `<span class="emoji">${emoji}</span>`)
  }
  
  // 将换行符转换为 <br>
  html = html.replace(/\n/g, '<br>')
  
  return html
}

// 处理表情选择
function handleEmojiSelect(emojiCode) {
  if (!inputMessage.value) {
    inputMessage.value = emojiCode
  } else {
    inputMessage.value += emojiCode
  }
}

// 关闭窗口
function handleClose() {
  if (isMobile.value) {
    // 移动端：关闭窗口或返回
    if (window.history.length > 1) {
      window.history.back()
    } else {
      window.close()
    }
  } else {
    // PC端：最小化（实际应用中可能需要通知父窗口）
    window.close()
  }
}

// 发起新会话
async function startNewConversation() {
  // 清除旧会话状态（但保留历史消息）
  isConversationEnded.value = false
  conversationId.value = ''
  localStorage.removeItem('customer_service_conversation_id')
  
  // 关闭旧的 WebSocket 连接
  if (ws.value) {
    ws.value.close()
    ws.value = null
  }
  
  // 创建新会话（不清空历史消息，保持聊天记录连续）
  await createConversation()
}

// 监听消息变化，只有在用户已经在底部时才自动滚动
watch(messages, () => {
  // 只有在用户已经在底部时才自动滚动
  if (isAtBottom.value) {
    scrollToBottom()
  }
  // 注意：不在这里更新 showScrollToBottom，由 handleScroll 控制
}, { deep: true })
</script>

<style scoped>
.chat-container {
  width: 100%;
  height: 100vh;
  display: flex;
  justify-content: center;
  align-items: center;
  background: #f5f5f5;
}

.chat-window {
  width: 100%;
  max-width: 800px;
  height: 100vh;
  max-height: 800px;
  display: flex;
  flex-direction: column;
  background: #fff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
}

.chat-window.mobile {
  max-width: 100%;
  max-height: 100vh;
  box-shadow: none;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 15px 20px;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;
}

.header-info {
  display: flex;
  align-items: center;
  gap: 12px;
}

.header-text {
  display: flex;
  flex-direction: column;
}

.admin-name {
  font-size: 16px;
  font-weight: 500;
  color: #303133;
}

.status-text {
  font-size: 12px;
  color: #909399;
}

.status-text.online {
  color: #67c23a;
}

.status-text.reconnecting {
  color: #e6a23c;
  animation: blink 1s infinite;
}

@keyframes blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.close-btn {
  font-size: 20px;
}

.chat-messages-wrapper {
  flex: 1;
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
  background: #f5f7fa;
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

.empty-message {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 100%;
}

.message-item {
  display: flex;
  margin-bottom: 20px;
  align-items: flex-start;
  gap: 10px;
}

.message-item.is-visitor {
  flex-direction: row-reverse;
}

.message-item.is-system {
  justify-content: center;
}

.system-message {
  padding: 8px 16px;
  background: #e4e7ed;
  border-radius: 4px;
  font-size: 12px;
  color: #909399;
}

.message-avatar {
  flex-shrink: 0;
}

.message-content {
  flex: 1;
  max-width: 70%;
}

.message-bubble {
  padding: 12px 16px;
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

.message-text :deep(.emoji) {
  display: inline-block;
  font-size: 1.2em;
  line-height: 1;
  vertical-align: middle;
  margin: 0 2px;
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

.chat-input {
  padding: 15px 20px;
  border-top: 1px solid #e4e7ed;
  background: #fff;
}

.input-wrapper {
  position: relative;
}

.input-textarea {
  margin-bottom: 10px;
}

.input-toolbar {
  position: absolute;
  bottom: 15px;
  left: 5px;
  display: flex;
  align-items: center;
  gap: 8px;
  z-index: 10;
}

.input-actions {
  display: flex;
  justify-content: flex-end;
}

.conversation-ended {
  padding: 10px 0;
}

.conversation-ended .el-alert {
  align-items: center;
}

.conversation-ended .el-button {
  margin-left: 10px;
}

/* 移动端适配 */
@media (max-width: 768px) {
  .chat-window {
    border-radius: 0;
  }
  
  .chat-header {
    padding: 12px 15px;
  }
  
  .chat-messages {
    padding: 15px;
  }
  
  .message-content {
    max-width: 80%;
  }
  
  .chat-input {
    padding: 12px 15px;
  }
}
</style>

