<template>
  <div class="customer-service-button" :class="{ 'mobile': isMobile }" :data-position="position">
    <!-- 浮动按钮 -->
    <el-button
      :type="'primary'"
      :icon="'ChatLineRound'"
      :circle="!isMobile"
      :size="isMobile ? 'large' : 'default'"
      class="float-button"
      @click="openChat"
    >
      <span v-if="isMobile" class="button-text">客服</span>
    </el-button>
    
    <!-- 聊天窗口（弹窗形式） -->
    <el-dialog
      v-model="chatVisible"
      :title="'在线客服'"
      :width="isMobile ? '100%' : '800px'"
      :fullscreen="isMobile"
      :close-on-click-modal="false"
      :close-on-press-escape="true"
      class="chat-dialog"
      @close="handleClose"
    >
      <ChatWindow
        v-if="chatVisible"
        :api-base-url="apiBaseUrl"
        :visitor-id="visitorId"
        :admin-id="adminId"
        :conversation-id="conversationId"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { ChatLineRound } from '@element-plus/icons-vue'
import ChatWindow from './ChatWindow.vue'

const props = defineProps({
  apiBaseUrl: {
    type: String,
    required: true
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
  position: {
    type: String,
    default: 'bottom-right', // bottom-right, bottom-left, top-right, top-left
    validator: (value) => ['bottom-right', 'bottom-left', 'top-right', 'top-left'].includes(value)
  }
})

const isMobile = ref(window.innerWidth <= 768)
const chatVisible = ref(false)
const visitorId = ref(props.visitorId || getVisitorIdFromStorage() || generateVisitorId())
const adminId = ref(props.adminId || '')
const conversationId = ref(props.conversationId || '')
const position = ref(props.position || 'bottom-right')

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

// 打开聊天窗口
function openChat() {
  chatVisible.value = true
}

// 关闭聊天窗口
function handleClose() {
  chatVisible.value = false
}

// 从URL参数获取配置
onMounted(() => {
  const urlParams = new URLSearchParams(window.location.search)
  
  // 从URL参数获取 admin_id
  if (urlParams.get('admin_id')) {
    adminId.value = urlParams.get('admin_id')
  }
  
  // 从URL参数获取 visitor_id
  if (urlParams.get('visitor_id')) {
    visitorId.value = urlParams.get('visitor_id')
  }
  
  // 从URL参数获取 conversation_id
  if (urlParams.get('conversation_id')) {
    conversationId.value = urlParams.get('conversation_id')
  }
  
  // 从URL参数获取 api_base_url
  if (urlParams.get('api_base_url')) {
    // 可以通过URL参数覆盖
  }
})
</script>

<style scoped>
.customer-service-button {
  position: fixed;
  z-index: 9999;
}

.float-button {
  position: fixed;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  transition: all 0.3s;
}

.float-button:hover {
  transform: scale(1.1);
  box-shadow: 0 6px 16px rgba(0, 0, 0, 0.2);
}

/* 位置样式 */
.customer-service-button:has(.float-button) {
  bottom: 20px;
  right: 20px;
}

.customer-service-button[data-position="bottom-left"] .float-button {
  bottom: 20px;
  left: 20px;
  right: auto;
}

.customer-service-button[data-position="top-right"] .float-button {
  top: 20px;
  right: 20px;
  bottom: auto;
}

.customer-service-button[data-position="top-left"] .float-button {
  top: 20px;
  left: 20px;
  bottom: auto;
  right: auto;
}

.button-text {
  margin-left: 5px;
}

.chat-dialog :deep(.el-dialog__body) {
  padding: 0;
  height: 600px;
}

.chat-dialog.mobile :deep(.el-dialog__body) {
  height: calc(100vh - 60px);
}

/* 移动端适配 */
@media (max-width: 768px) {
  .float-button {
    width: 60px;
    height: 60px;
    font-size: 24px;
  }
  
  .customer-service-button:has(.float-button) {
    bottom: 15px;
    right: 15px;
  }
}
</style>

