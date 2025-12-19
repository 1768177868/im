# 多客服系统使用文档

## 概述

本系统是一个类似百度商桥的多客服系统，支持访客端接入和客服端管理。

## 功能特性

- ✅ 访客注册和管理
- ✅ 会话创建和管理
- ✅ 实时消息通信（WebSocket）
- ✅ 客服自动分配（最少会话数策略）
- ✅ 消息已读状态
- ✅ 会话状态管理
- ✅ 在线状态监控

## 数据库迁移

运行数据库迁移以创建必要的表：

```bash
go run . artisan migrate
```

这将创建以下表：
- `visitors` - 访客表
- `conversations` - 会话表
- `messages` - 消息表
- `visitor_sessions` - 访客会话表

## API 接口

### 访客端 API（公开，无需认证）

#### 1. 注册访客
```
POST /api/visitor/register
参数：
- visitor_id: 访客ID（可选，不传则自动生成）
- name: 姓名（可选）
- email: 邮箱（可选）
- phone: 手机号（可选）
- source: 来源页面（可选）
- referer: 来源URL（可选）
- location: 地理位置（可选）
- device: 设备类型（可选）
- browser: 浏览器（可选）
- os: 操作系统（可选）
```

#### 2. 创建会话
```
POST /api/visitor/conversations
参数：
- visitor_id: 访客ID（必填）
- title: 会话标题（可选）
```

#### 3. 获取会话列表
```
GET /api/visitor/conversations?visitor_id={visitor_id}
```

#### 4. 获取消息列表
```
GET /api/visitor/messages?conversation_id={conversation_id}&page=1&page_size=20
```

#### 5. 发送消息
```
POST /api/visitor/messages
参数：
- conversation_id: 会话ID（必填）
- visitor_id: 访客ID（必填）
- content: 消息内容（必填）
- type: 消息类型，默认text（可选：text/image/file/location）
```

#### 6. WebSocket 连接
```
WS /api/visitor/ws?visitor_id={visitor_id}&conversation_id={conversation_id}
```

### 客服端 API（需要管理员认证）

#### 1. 获取会话列表
```
GET /api/admin/customer/conversations?admin_id={admin_id}&status={status}
参数：
- admin_id: 客服ID（可选，默认当前登录管理员）
- status: 会话状态（可选：1-进行中，2-已结束，3-已关闭）
```

#### 2. 获取会话详情
```
GET /api/admin/customer/conversations/{id}
```

#### 3. 获取消息列表
```
GET /api/admin/customer/messages?conversation_id={conversation_id}&page=1&page_size=50
```

#### 4. 发送消息
```
POST /api/admin/customer/messages
参数：
- conversation_id: 会话ID（必填）
- content: 消息内容（必填）
- type: 消息类型，默认text（可选）
```

#### 5. 分配会话
```
POST /api/admin/customer/conversations/assign
参数：
- conversation_id: 会话ID（必填）
- admin_id: 客服ID（必填）
```

#### 6. 结束会话
```
POST /api/admin/customer/conversations/end
参数：
- conversation_id: 会话ID（必填）
```

#### 7. 标记消息已读
```
POST /api/admin/customer/messages/read
参数：
- conversation_id: 会话ID（必填）
```

#### 8. 获取在线访客列表
```
GET /api/admin/customer/visitors/online
```

#### 9. 获取在线客服列表
```
GET /api/admin/customer/admins/online
```

#### 10. WebSocket 连接
```
WS /ws/admin/customer?conversation_id={conversation_id}
需要 Authorization Header: Bearer {token}
```

## 前端接入

### 方式一：使用提供的 SDK

在网站页面中引入 SDK：

```html
<script src="customer-service.js"></script>
<script>
  CustomerService.init({
    apiBaseUrl: 'http://your-api-domain.com',
    visitorId: 'visitor_unique_id', // 可选，不传则自动生成
    onMessage: function(message) {
      console.log('收到消息:', message);
      // 处理收到的消息
    },
    onConnect: function() {
      console.log('连接成功');
    },
    onDisconnect: function() {
      console.log('连接断开');
    },
    onError: function(error) {
      console.error('错误:', error);
    }
  });
  
  // 发送消息
  CustomerService.sendMessage('你好，我需要帮助');
  
  // 获取消息列表
  CustomerService.getMessages(1, 20, function(data) {
    console.log('消息列表:', data);
  });
</script>
```

### 方式二：直接调用 API

```javascript
// 1. 注册访客
fetch('http://your-api-domain.com/api/visitor/register', {
  method: 'POST',
  headers: {'Content-Type': 'application/x-www-form-urlencoded'},
  body: 'visitor_id=visitor123&source=' + encodeURIComponent(window.location.href)
})
.then(res => res.json())
.then(data => {
  console.log('访客注册成功:', data);
});

// 2. 创建会话
fetch('http://your-api-domain.com/api/visitor/conversations', {
  method: 'POST',
  headers: {'Content-Type': 'application/x-www-form-urlencoded'},
  body: 'visitor_id=visitor123'
})
.then(res => res.json())
.then(data => {
  console.log('会话创建成功:', data);
  var conversationId = data.data.id;
  
  // 3. 连接 WebSocket
  var ws = new WebSocket('ws://your-api-domain.com/api/visitor/ws?visitor_id=visitor123&conversation_id=' + conversationId);
  ws.onmessage = function(event) {
    var message = JSON.parse(event.data);
    console.log('收到消息:', message);
  };
  
  // 4. 发送消息
  fetch('http://your-api-domain.com/api/visitor/messages', {
    method: 'POST',
    headers: {'Content-Type': 'application/x-www-form-urlencoded'},
    body: 'conversation_id=' + conversationId + '&visitor_id=visitor123&content=你好'
  });
});
```

## 消息格式

### 普通消息
```json
{
  "type": "text",
  "conversation_id": 1,
  "sender_type": "visitor",
  "sender_id": 1,
  "content": "消息内容",
  "timestamp": 1234567890,
  "message_id": 1
}
```

### 系统消息
```json
{
  "type": "system",
  "event": "connected|disconnected|typing|read|assigned|ended",
  "data": {
    "admin_id": 1,
    "admin_name": "客服名称"
  },
  "timestamp": 1234567890
}
```

## 客服分配策略

系统使用"最少会话数"策略自动分配客服：
1. 优先选择在线且会话数最少的客服
2. 如果没有在线客服，选择会话数最少的客服
3. 如果所有客服会话数相同，随机选择

## 注意事项

1. 访客端 API 是公开的，不需要认证
2. 客服端 API 需要管理员 JWT Token 认证
3. WebSocket 连接会自动重连（最多5次）
4. 会话创建时会自动分配客服
5. 消息通过 WebSocket 实时推送

## 扩展功能建议

- [ ] 文件上传支持
- [ ] 图片消息支持
- [ ] 会话转接
- [ ] 会话评价
- [ ] 客服工作台界面
- [ ] 访客聊天窗口组件
- [ ] 消息历史记录
- [ ] 统计报表

