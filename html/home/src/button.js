import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import CustomerServiceButton from './components/CustomerServiceButton.vue'
import './style.css'

// 从URL参数或配置获取API地址
function getApiBaseUrl() {
  const urlParams = new URLSearchParams(window.location.search)
  return urlParams.get('api_base_url') || window.CUSTOMER_SERVICE_API_URL || 'http://127.0.0.1:3000'
}

// 从URL参数获取配置
function getConfig() {
  const urlParams = new URLSearchParams(window.location.search)
  return {
    apiBaseUrl: getApiBaseUrl(),
    visitorId: urlParams.get('visitor_id') || '',
    adminId: urlParams.get('admin_id') || '',
    conversationId: urlParams.get('conversation_id') || '',
    position: urlParams.get('position') || 'bottom-right'
  }
}

const config = getConfig()

const app = createApp(CustomerServiceButton, config)

// 注册 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus, {
  locale: zhCn
})

app.mount('#app')

