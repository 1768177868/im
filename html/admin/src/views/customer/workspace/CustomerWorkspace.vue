<template>
  <div class="customer-workspace">
    <!-- 左侧会话列表 -->
    <div class="conversation-sidebar">
      <div class="sidebar-header">
        <h3>{{ $t('customer.workspace.conversations') }}</h3>
        <el-button 
          type="primary" 
          :icon="Refresh" 
          circle 
          size="small"
          @click="loadConversations"
          :loading="loading"
        />
      </div>
      
      <!-- 筛选标签 -->
      <div class="filter-tabs">
        <el-radio-group v-model="statusFilter" size="small" @change="handleFilterChange">
          <el-radio-button label="">{{ $t('common.all') }}</el-radio-button>
          <el-radio-button label="1">
            {{ $t('customer.conversation.status_active') }}
            <el-badge v-if="activeCount > 0" :value="activeCount" class="tab-badge" />
          </el-radio-button>
          <el-radio-button label="2">{{ $t('customer.conversation.status_ended') }}</el-radio-button>
        </el-radio-group>
      </div>
      
      <!-- 搜索框 -->
      <div class="search-box">
        <el-input
          v-model="searchKeyword"
          :placeholder="$t('customer.workspace.search_placeholder')"
          clearable
          size="small"
          @input="handleSearchDebounced"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
      </div>
      
      <!-- 会话列表 -->
      <div class="conversation-list-container" v-loading="loading">
        <div v-if="conversations.length === 0" class="empty-list">
          <el-empty :description="$t('customer.workspace.no_conversations')" :image-size="80" />
        </div>
        <div
          v-for="conv in conversations"
          :key="conv.id"
          class="conversation-item"
          :class="{ 'active': selectedConversationId === conv.id }"
          @click="selectConversation(conv)"
        >
          <div class="conv-avatar">
            <el-avatar :size="40">
              {{ (conv.visitor?.name || conv.visitor?.visitor_id || '访')?.charAt(0) }}
            </el-avatar>
            <span v-if="conv.status === 1" class="status-dot online"></span>
          </div>
          <div class="conv-info">
            <div class="conv-header">
              <span class="visitor-name">
                {{ conv.visitor?.name || conv.visitor?.visitor_id || `访客${conv.visitor_id}` }}
              </span>
              <span class="conv-time">{{ formatRelativeTime(conv.last_message_at) }}</span>
            </div>
            <div class="conv-preview">
              <span class="last-message">{{ conv.last_message || conv.title || '暂无消息' }}</span>
              <el-tag v-if="conv.status === 1" type="success" size="small">进行中</el-tag>
              <el-tag v-else-if="conv.status === 2" type="info" size="small">已结束</el-tag>
            </div>
          </div>
        </div>
        
        <!-- 加载更多 -->
        <div v-if="hasMore" class="load-more-btn">
          <el-button text :loading="loadingMore" @click="loadMoreConversations">
            {{ loadingMore ? '加载中...' : '加载更多' }}
          </el-button>
        </div>
      </div>
    </div>
    
    <!-- 右侧聊天区域 -->
    <div class="chat-main">
      <template v-if="selectedConversationId">
        <ConversationDetail 
          :key="selectedConversationId"
          :conversation-id="selectedConversationId" 
          @close="handleCloseDetail"
          @ended="handleConversationEnded"
        />
      </template>
      <template v-else>
        <div class="no-selection">
          <el-empty :description="$t('customer.workspace.select_conversation')" :image-size="120" />
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh, Search } from '@element-plus/icons-vue'
import { getConversations } from '@/api/customer'
import ConversationDetail from '../conversation/ConversationDetail.vue'
import { debounce } from 'lodash-es'

const { t } = useI18n()

// 状态
const loading = ref(false)
const loadingMore = ref(false)
const conversations = ref([])
const selectedConversationId = ref(null)
const statusFilter = ref('1') // 默认显示进行中的会话
const searchKeyword = ref('')
const currentPage = ref(1)
const pageSize = ref(20)
const total = ref(0)
const hasMore = computed(() => conversations.value.length < total.value)
const activeCount = ref(0)

// 自动刷新定时器
let refreshTimer = null
// 防止重复请求
let isLoading = false

// 加载会话列表
const loadConversations = async (append = false) => {
  // 防止重复请求
  if (isLoading && !append) {
    return
  }
  
  if (append) {
    if (loadingMore.value) return // 如果正在加载更多，直接返回
    loadingMore.value = true
  } else {
    if (loading.value) return // 如果正在加载，直接返回
    isLoading = true
    loading.value = true
    currentPage.value = 1
  }
  
  try {
    const params = {
      page: currentPage.value,
      page_size: pageSize.value
    }
    
    if (statusFilter.value) {
      params.status = statusFilter.value
    }
    
    if (searchKeyword.value) {
      params.visitor_name = searchKeyword.value
    }
    
    const response = await getConversations(params)
    if (response.code === 200) {
      const data = response.data
      if (append) {
        // 加载更多：追加数据，但要去重
        const existingIds = new Set(conversations.value.map(c => c.id))
        const newItems = (data.list || []).filter(item => !existingIds.has(item.id))
        conversations.value = [...conversations.value, ...newItems]
      } else {
        // 刷新：直接替换
        conversations.value = data.list || []
      }
      total.value = data.total || 0
      
      // 如果筛选的是全部，单独获取进行中的数量
      if (!statusFilter.value) {
        loadActiveCount()
      } else if (statusFilter.value === '1') {
        activeCount.value = total.value
      }
    }
  } catch (error) {
    console.error('加载会话列表失败:', error)
  } finally {
    loading.value = false
    loadingMore.value = false
    isLoading = false
  }
}

// 加载进行中会话数量
const loadActiveCount = async () => {
  try {
    const response = await getConversations({ status: 1, page: 1, page_size: 1 })
    if (response.code === 200) {
      activeCount.value = response.data?.total || 0
    }
  } catch (error) {
    console.error('获取进行中会话数量失败:', error)
  }
}

// 加载更多
const loadMoreConversations = () => {
  currentPage.value++
  loadConversations(true)
}

// 筛选变化
const handleFilterChange = () => {
  loadConversations()
}

// 搜索（防抖）
const handleSearchDebounced = debounce(() => {
  loadConversations()
}, 300)

// 选择会话
const selectConversation = (conv) => {
  selectedConversationId.value = conv.id
}

// 关闭详情
const handleCloseDetail = () => {
  selectedConversationId.value = null
}

// 会话结束回调
const handleConversationEnded = () => {
  // 刷新列表
  loadConversations()
}

// 格式化相对时间
const formatRelativeTime = (timeStr) => {
  if (!timeStr) return ''
  
  let date
  try {
    if (typeof timeStr === 'string') {
      if (timeStr.match(/^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$/)) {
        date = new Date(timeStr.replace(' ', 'T') + 'Z')
      } else if (timeStr.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(\.\d+)?Z$/)) {
        date = new Date(timeStr)
      } else if (timeStr.match(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}/)) {
        date = new Date(timeStr + 'Z')
      } else {
        date = new Date(timeStr)
      }
    } else {
      date = new Date(timeStr)
    }
    
    if (isNaN(date.getTime())) return ''
    
    const now = new Date()
    const diff = now - date
    const seconds = Math.floor(diff / 1000)
    const minutes = Math.floor(seconds / 60)
    const hours = Math.floor(minutes / 60)
    const days = Math.floor(hours / 24)
    
    if (seconds < 60) return '刚刚'
    if (minutes < 60) return `${minutes}分钟前`
    if (hours < 24) return `${hours}小时前`
    if (days < 7) return `${days}天前`
    
    return date.toLocaleDateString('zh-CN')
  } catch (e) {
    return ''
  }
}

// 启动自动刷新
const startAutoRefresh = () => {
  refreshTimer = setInterval(() => {
    loadConversations()
  }, 30000) // 每30秒刷新一次
}

// 停止自动刷新
const stopAutoRefresh = () => {
  if (refreshTimer) {
    clearInterval(refreshTimer)
    refreshTimer = null
  }
}

onMounted(() => {
  loadConversations()
  loadActiveCount()
  startAutoRefresh()
})

onUnmounted(() => {
  stopAutoRefresh()
})
</script>

<style scoped>
.customer-workspace {
  display: flex;
  height: calc(100vh - 120px);
  min-height: 600px;
  background: #f5f7fa;
  border-radius: 4px;
  overflow: hidden;
}

/* 左侧会话列表 */
.conversation-sidebar {
  width: 320px;
  min-width: 280px;
  background: #fff;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e4e7ed;
}

.sidebar-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.filter-tabs {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
}

.filter-tabs :deep(.el-radio-button__inner) {
  padding: 6px 12px;
}

.tab-badge {
  margin-left: 4px;
}

.tab-badge :deep(.el-badge__content) {
  top: -2px;
}

.search-box {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
}

.conversation-list-container {
  flex: 1;
  overflow-y: auto;
}

.empty-list {
  display: flex;
  justify-content: center;
  align-items: center;
  height: 200px;
}

.conversation-item {
  display: flex;
  padding: 12px 16px;
  cursor: pointer;
  transition: background 0.2s;
  border-bottom: 1px solid #f0f0f0;
}

.conversation-item:hover {
  background: #f5f7fa;
}

.conversation-item.active {
  background: #ecf5ff;
  border-left: 3px solid #409eff;
}

.conv-avatar {
  position: relative;
  margin-right: 12px;
}

.status-dot {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 2px solid #fff;
}

.status-dot.online {
  background: #67c23a;
}

.conv-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
}

.conv-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 4px;
}

.visitor-name {
  font-size: 14px;
  font-weight: 500;
  color: #303133;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 140px;
}

.conv-time {
  font-size: 12px;
  color: #909399;
  flex-shrink: 0;
}

.conv-preview {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
}

.last-message {
  font-size: 13px;
  color: #909399;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.load-more-btn {
  display: flex;
  justify-content: center;
  padding: 12px;
}

/* 右侧聊天区域 */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: #fff;
  overflow: hidden;
}

.no-selection {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  background: #fafafa;
}

/* 覆盖 ConversationDetail 的样式 */
.chat-main :deep(.conversation-detail) {
  height: 100%;
  display: flex;
  flex-direction: column;
  padding: 0;
}

.chat-main :deep(.messages-section) {
  flex: 1;
  display: flex;
  flex-direction: column;
  margin-top: 0;
  overflow: hidden;
}

.chat-main :deep(.messages-wrapper) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.chat-main :deep(.messages-container) {
  flex: 1;
  max-height: none;
}

.chat-main :deep(.el-descriptions) {
  margin: 16px;
}

.chat-main :deep(.messages-header) {
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  margin-bottom: 0;
}

.chat-main :deep(.send-message-section) {
  margin: 0;
  padding: 16px;
  border-top: 1px solid #e4e7ed;
}

/* 响应式 */
@media (max-width: 768px) {
  .customer-workspace {
    flex-direction: column;
  }
  
  .conversation-sidebar {
    width: 100%;
    height: 300px;
    min-height: 200px;
  }
  
  .chat-main {
    flex: 1;
  }
}
</style>

