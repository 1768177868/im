# 快速接入指南

## 最简单的接入方式

### 1. 在您的网站中添加客服链接

**自动分配最闲的客服：**
```html
<a href="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com" target="_blank">
  联系客服
</a>
```

**指定某个客服：**
```html
<a href="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com&admin_id=1" target="_blank">
  联系客服1
</a>
```

### 2. 使用客服按钮（浮动按钮）

**自动分配：**
```html
<a href="https://your-domain.com/button.html?api_base_url=https://your-api-domain.com" target="_blank">
  <button style="position: fixed; bottom: 20px; right: 20px; padding: 15px 25px; background: #409eff; color: white; border: none; border-radius: 25px; cursor: pointer; box-shadow: 0 4px 12px rgba(0,0,0,0.15);">
    联系客服
  </button>
</a>
```

**指定客服：**
```html
<a href="https://your-domain.com/button.html?api_base_url=https://your-api-domain.com&admin_id=1" target="_blank">
  <button style="position: fixed; bottom: 20px; right: 20px; padding: 15px 25px; background: #409eff; color: white; border: none; border-radius: 25px; cursor: pointer; box-shadow: 0 4px 12px rgba(0,0,0,0.15);">
    联系客服1
  </button>
</a>
```

### 3. 嵌入到iframe

```html
<iframe 
  src="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com&admin_id=1"
  width="100%"
  height="600px"
  frameborder="0"
  style="border: 1px solid #ddd; border-radius: 8px;"
></iframe>
```

## 参数说明

| 参数 | 说明 | 必填 | 示例 |
|------|------|------|------|
| `api_base_url` | API服务器地址 | 是 | `https://api.example.com` |
| `admin_id` | 指定客服ID | 否 | `1`（不传则自动分配最闲的客服） |
| `visitor_id` | 访客ID | 否 | 不传则自动生成 |
| `conversation_id` | 会话ID | 否 | 用于继续之前的会话 |

## 示例

### 示例1：自动分配客服
```html
<a href="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com">
  在线客服
</a>
```

### 示例2：指定客服ID为1
```html
<a href="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com&admin_id=1">
  联系客服1
</a>
```

### 示例3：指定客服ID为2
```html
<a href="https://your-domain.com/chat.html?api_base_url=https://your-api-domain.com&admin_id=2">
  联系客服2
</a>
```

## 响应式设计

- **PC端**：固定宽度800px，居中显示
- **移动端**：全屏显示，优化触摸操作
- 自动检测设备类型，无需额外配置

## 注意事项

1. 确保 `api_base_url` 指向正确的API服务器地址
2. 如果指定 `admin_id`，请确保该客服存在且已启用
3. 如果不指定 `admin_id`，系统会自动分配当前最闲的在线客服
4. 移动端建议使用HTTPS协议

