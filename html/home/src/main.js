import { createApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import zhCn from 'element-plus/dist/locale/zh-cn.mjs'
import en from 'element-plus/dist/locale/en.mjs'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import ChatWindow from './components/ChatWindow.vue'
import './style.css'

// 从环境变量获取配置（优先使用 .env 配置，不再从 URL 参数获取）
function getApiBaseUrl() {
  return import.meta.env.VITE_API_BASE_URL || 
         window.CUSTOMER_SERVICE_API_URL || 
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

// 从 URL 参数获取语言设置
function getLanguage() {
  const urlParams = new URLSearchParams(window.location.search)
  const lang = urlParams.get('lang') || urlParams.get('language') || 'zh-CN'
  return lang.toLowerCase()
}

// 根据语言代码获取 Element Plus locale
function getElementPlusLocale(lang) {
  const langMap = {
    'zh-cn': zhCn,
    'zh': zhCn,
    'en-us': en,
    'en': en
  }
  return langMap[lang] || zhCn
}

const language = getLanguage()
const locale = getElementPlusLocale(language)

const app = createApp(ChatWindow, {
  apiBaseUrl: getApiBaseUrl(),
  visitorId: getVisitorId(),
  adminId: getAdminId(),
  language: language
})

// 注册 Element Plus 图标
for (const [key, component] of Object.entries(ElementPlusIconsVue)) {
  app.component(key, component)
}

app.use(ElementPlus, {
  locale: locale
})

app.mount('#app')

