# 多客服系统访客端

基于 Vue 3 + Element Plus 的访客端聊天界面，支持PC和移动端响应式设计。

## 功能特性

- ✅ 实时聊天（WebSocket）
- ✅ 自动分配最闲的客服
- ✅ 支持指定客服ID
- ✅ PC和移动端响应式设计
- ✅ 消息历史记录
- ✅ 自动重连
- ✅ 可嵌入其他网站

## 快速开始

### 1. 安装依赖

```bash
cd html/home
npm install
```

### 2. 开发

```bash
npm run dev
```

访问：
- 聊天页面：http://localhost:3006/chat.html
- 按钮组件：http://localhost:3006/button.html
- 接入示例：http://localhost:3006/example.html

### 3. 构建

```bash
npm run build
```

构建后的文件在 `dist` 目录。

## 接入方式

> http://localhost:3006/chat.html?api_base_url=http://localhost:3008

### 方式一：使用客服按钮（推荐）

在您的网站中添加：

```html
<!-- 自动分配最闲的客服 -->
<a href="https://your-domain.com/button.html?api_base_url=https://your-api-domain.com" target="_blank">
  联系客服
</a>

<!-- 指定客服ID -->
<a href="https://your-domain.com/button.html?api_base_url=https://your-api-domain.com&admin_id=1" target="_blank">
  联系客服1
</a>
```

### 方式二：直接打开聊天页面

```html
<!-- 自动分配 -->
<a href="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com" target="_blank">
  在线客服
</a>

<!-- 指定客服 -->
<a href="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com&admin_id=1" target="_blank">
  联系客服1
</a>
```

### 方式三：嵌入到iframe

```html
<iframe 
  src="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com&admin_id=1"
  width="100%"
  height="600px"
  frameborder="0"
></iframe>
```

## URL参数

- `api_base_url`: API服务器地址（必填）
- `admin_id`: 指定客服ID（可选，不传则自动分配最闲的客服）
- `visitor_id`: 访客ID（可选，不传则自动生成）
- `conversation_id`: 会话ID（可选，用于继续之前的会话）
- `position`: 按钮位置（可选，bottom-right/bottom-left/top-right/top-left）

## 技术栈

- Vue 3
- Element Plus
- Axios
- WebSocket

## 浏览器支持

- Chrome（推荐）
- Firefox
- Safari
- Edge
- 移动端浏览器

## 注意事项

1. 确保API服务器已配置CORS，允许跨域请求
2. WebSocket连接需要支持跨域
3. 移动端建议使用HTTPS协议

