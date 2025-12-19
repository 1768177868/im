import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import ChatWindow from './components/ChatWindow.vue'
import './style.css'

// 从URL参数获取配置
function getApiBaseUrl() {
  const urlParams = new URLSearchParams(window.location.search)
  return urlParams.get('api_base_url') || 
         window.CUSTOMER_SERVICE_API_URL || 
         import.meta.env.VITE_API_BASE_URL ||
         'http://127.0.0.1:3000'
}

function getVisitorId() {
  const urlParams = new URLSearchParams(window.location.search)
  return urlParams.get('visitor_id') || ''
}

function getAdminId() {
  const urlParams = new URLSearchParams(window.location.search)
  return urlParams.get('admin_id') || ''
}

const app = createApp(ChatWindow, {
  apiBaseUrl: getApiBaseUrl(),
  visitorId: getVisitorId(),
  adminId: getAdminId()
})

// 注册 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus, {
  locale: zhCn
})

app.mount('#app')

