/**
 * 多客服系统访客端 SDK
 * 类似百度商桥的接入方式
 * 
 * 使用方法：
 * <script src="customer-service.js"></script>
 * <script>
 *   CustomerService.init({
 *     apiBaseUrl: 'http://your-api-domain.com',
 *     visitorId: 'visitor_unique_id', // 可选，不传则自动生成
 *     onMessage: function(message) {
 *       console.log('收到消息:', message);
 *     },
 *     onConnect: function() {
 *       console.log('连接成功');
 *     }
 *   });
 * </script>
 */

(function(window) {
  'use strict';

  var CustomerService = {
    config: {
      apiBaseUrl: '',
      visitorId: '',
      conversationId: null,
      ws: null,
      reconnectAttempts: 0,
      maxReconnectAttempts: 5,
      reconnectDelay: 3000,
      onMessage: null,
      onConnect: null,
      onDisconnect: null,
      onError: null
    },

    /**
     * 初始化客服系统
     */
    init: function(options) {
      var self = this;
      
      // 合并配置
      Object.assign(this.config, options || {});
      
      // 如果没有 visitorId，生成一个
      if (!this.config.visitorId) {
        this.config.visitorId = this.generateVisitorId();
      }
      
      // 注册访客
      this.registerVisitor(function(visitor) {
        // 创建会话
        self.createConversation(function(conversation) {
          self.config.conversationId = conversation.id;
          // 连接 WebSocket
          self.connectWebSocket();
        });
      });
    },

    /**
     * 生成访客ID
     */
    generateVisitorId: function() {
      var timestamp = Date.now();
      var random = Math.random().toString(36).substring(2, 15);
      return 'visitor_' + timestamp + '_' + random;
    },

    /**
     * 注册访客
     */
    registerVisitor: function(callback) {
      var self = this;
      var xhr = new XMLHttpRequest();
      xhr.open('POST', this.config.apiBaseUrl + '/api/visitor/register', true);
      xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
      
      xhr.onreadystatechange = function() {
        if (xhr.readyState === 4) {
          if (xhr.status === 200) {
            var response = JSON.parse(xhr.responseText);
            if (response.code === 200 && response.data) {
              callback(response.data);
            } else {
              self.handleError('注册访客失败: ' + (response.message || '未知错误'));
            }
          } else {
            self.handleError('注册访客失败，HTTP状态码: ' + xhr.status);
          }
        }
      };
      
      var params = 'visitor_id=' + encodeURIComponent(this.config.visitorId);
      params += '&source=' + encodeURIComponent(window.location.href);
      params += '&referer=' + encodeURIComponent(document.referrer);
      
      // 检测设备信息
      var ua = navigator.userAgent;
      var device = this.detectDevice(ua);
      params += '&device=' + encodeURIComponent(device.device);
      params += '&browser=' + encodeURIComponent(device.browser);
      params += '&os=' + encodeURIComponent(device.os);
      
      xhr.send(params);
    },

    /**
     * 创建会话
     */
    createConversation: function(callback) {
      var self = this;
      var xhr = new XMLHttpRequest();
      xhr.open('POST', this.config.apiBaseUrl + '/api/visitor/conversations', true);
      xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
      
      xhr.onreadystatechange = function() {
        if (xhr.readyState === 4) {
          if (xhr.status === 200) {
            var response = JSON.parse(xhr.responseText);
            if (response.code === 200 && response.data) {
              callback(response.data);
            } else {
              self.handleError('创建会话失败: ' + (response.message || '未知错误'));
            }
          } else {
            self.handleError('创建会话失败，HTTP状态码: ' + xhr.status);
          }
        }
      };
      
      var params = 'visitor_id=' + encodeURIComponent(this.config.visitorId);
      params += '&title=' + encodeURIComponent(document.title || '新会话');
      
      xhr.send(params);
    },

    /**
     * 连接 WebSocket
     */
    connectWebSocket: function() {
      var self = this;
      var wsUrl = this.config.apiBaseUrl.replace(/^http/, 'ws') + '/api/visitor/ws';
      wsUrl += '?visitor_id=' + encodeURIComponent(this.config.visitorId);
      if (this.config.conversationId) {
        wsUrl += '&conversation_id=' + this.config.conversationId;
      }
      
      try {
        this.config.ws = new WebSocket(wsUrl);
        
        this.config.ws.onopen = function() {
          self.config.reconnectAttempts = 0;
          if (self.config.onConnect) {
            self.config.onConnect();
          }
        };
        
        this.config.ws.onmessage = function(event) {
          try {
            var message = JSON.parse(event.data);
            if (self.config.onMessage) {
              self.config.onMessage(message);
            }
          } catch (e) {
            console.error('解析消息失败:', e);
          }
        };
        
        this.config.ws.onerror = function(error) {
          self.handleError('WebSocket错误: ' + error);
        };
        
        this.config.ws.onclose = function() {
          if (self.config.onDisconnect) {
            self.config.onDisconnect();
          }
          // 尝试重连
          self.reconnect();
        };
      } catch (e) {
        this.handleError('WebSocket连接失败: ' + e.message);
      }
    },

    /**
     * 重连 WebSocket
     */
    reconnect: function() {
      var self = this;
      if (this.config.reconnectAttempts < this.config.maxReconnectAttempts) {
        this.config.reconnectAttempts++;
        setTimeout(function() {
          self.connectWebSocket();
        }, this.config.reconnectDelay);
      } else {
        this.handleError('WebSocket重连失败，已达到最大重试次数');
      }
    },

    /**
     * 发送消息
     */
    sendMessage: function(content, type) {
      type = type || 'text';
      
      if (!this.config.conversationId) {
        this.handleError('会话ID不存在，请先创建会话');
        return;
      }
      
      if (!content || content.trim() === '') {
        this.handleError('消息内容不能为空');
        return;
      }
      
      var self = this;
      var xhr = new XMLHttpRequest();
      xhr.open('POST', this.config.apiBaseUrl + '/api/visitor/messages', true);
      xhr.setRequestHeader('Content-Type', 'application/x-www-form-urlencoded');
      
      xhr.onreadystatechange = function() {
        if (xhr.readyState === 4) {
          if (xhr.status === 200) {
            var response = JSON.parse(xhr.responseText);
            if (response.code === 200) {
              // 消息发送成功
            } else {
              self.handleError('发送消息失败: ' + (response.message || '未知错误'));
            }
          } else {
            self.handleError('发送消息失败，HTTP状态码: ' + xhr.status);
          }
        }
      };
      
      var params = 'conversation_id=' + this.config.conversationId;
      params += '&visitor_id=' + encodeURIComponent(this.config.visitorId);
      params += '&content=' + encodeURIComponent(content);
      params += '&type=' + encodeURIComponent(type);
      
      xhr.send(params);
    },

    /**
     * 获取消息列表
     */
    getMessages: function(page, pageSize, callback) {
      page = page || 1;
      pageSize = pageSize || 20;
      
      if (!this.config.conversationId) {
        this.handleError('会话ID不存在');
        return;
      }
      
      var self = this;
      var xhr = new XMLHttpRequest();
      var url = this.config.apiBaseUrl + '/api/visitor/messages';
      url += '?conversation_id=' + this.config.conversationId;
      url += '&page=' + page;
      url += '&page_size=' + pageSize;
      
      xhr.open('GET', url, true);
      xhr.onreadystatechange = function() {
        if (xhr.readyState === 4) {
          if (xhr.status === 200) {
            var response = JSON.parse(xhr.responseText);
            if (response.code === 200 && callback) {
              callback(response.data);
            }
          }
        }
      };
      xhr.send();
    },

    /**
     * 检测设备信息
     */
    detectDevice: function(ua) {
      var device = {
        device: 'desktop',
        browser: 'unknown',
        os: 'unknown'
      };
      
      // 检测设备类型
      if (/mobile|android|iphone|ipad/i.test(ua)) {
        device.device = 'mobile';
      } else if (/tablet|ipad/i.test(ua)) {
        device.device = 'tablet';
      }
      
      // 检测浏览器
      if (/chrome/i.test(ua) && !/edge/i.test(ua)) {
        device.browser = 'Chrome';
      } else if (/firefox/i.test(ua)) {
        device.browser = 'Firefox';
      } else if (/safari/i.test(ua) && !/chrome/i.test(ua)) {
        device.browser = 'Safari';
      } else if (/edge/i.test(ua)) {
        device.browser = 'Edge';
      } else if (/msie|trident/i.test(ua)) {
        device.browser = 'IE';
      }
      
      // 检测操作系统
      if (/windows/i.test(ua)) {
        device.os = 'Windows';
      } else if (/mac/i.test(ua)) {
        device.os = 'macOS';
      } else if (/linux/i.test(ua)) {
        device.os = 'Linux';
      } else if (/android/i.test(ua)) {
        device.os = 'Android';
      } else if (/ios|iphone|ipad/i.test(ua)) {
        device.os = 'iOS';
      }
      
      return device;
    },

    /**
     * 错误处理
     */
    handleError: function(message) {
      console.error('[CustomerService]', message);
      if (this.config.onError) {
        this.config.onError(message);
      }
    },

    /**
     * 断开连接
     */
    disconnect: function() {
      if (this.config.ws) {
        this.config.ws.close();
        this.config.ws = null;
      }
    }
  };

  // 导出到全局
  window.CustomerService = CustomerService;

})(window);

