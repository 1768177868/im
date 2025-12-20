<template>
  <div class="emoji-picker" ref="pickerRef">
    <el-popover
      placement="top"
      width="320"
      trigger="manual"
      v-model:visible="visible"
      :hide-after="0"
      popper-class="emoji-picker-popover"
    >
      <template #reference>
        <el-button
          text
          circle
          size="small"
          @click.stop="toggle"
          class="emoji-btn"
        >
          😊
        </el-button>
      </template>
      
      <div class="emoji-picker-content" @click.stop @mousedown.stop>
        <div class="emoji-tabs">
          <el-tabs v-model="activeTab" size="small">
            <el-tab-pane label="表情" name="emoji">
              <div class="emoji-grid">
                <div
                  v-for="emoji in emojiList"
                  :key="emoji.code"
                  class="emoji-item"
                  :title="emoji.name"
                  @click="selectEmoji(emoji)"
                >
                  {{ emoji.emoji }}
                </div>
              </div>
            </el-tab-pane>
          </el-tabs>
        </div>
      </div>
    </el-popover>
  </div>
</template>

<script setup>
import { ref, onMounted, onUnmounted } from 'vue'

const emit = defineEmits(['select'])

const visible = ref(false)
const activeTab = ref('emoji')
const pickerRef = ref(null)

// 点击外部关闭 Popover
function handleClickOutside(event) {
  if (!visible.value) return
  
  // 检查是否点击在按钮上
  const button = event.target.closest('.emoji-btn')
  if (button) return
  
  // 检查是否点击在 Popover 内容区域
  const popoverContent = document.querySelector('.emoji-picker-popover')
  if (popoverContent && popoverContent.contains(event.target)) {
    return
  }
  
  // 点击外部，关闭 Popover
  visible.value = false
}

onMounted(() => {
  // 使用 setTimeout 确保在下一个事件循环中添加监听器，避免立即触发
  setTimeout(() => {
    document.addEventListener('click', handleClickOutside, true)
  }, 0)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside, true)
})

// 常用表情列表
const emojiList = [
  { code: '[微笑]', name: '微笑', emoji: '😊' },
  { code: '[大笑]', name: '大笑', emoji: '😃' },
  { code: '[开心]', name: '开心', emoji: '😄' },
  { code: '[笑哭]', name: '笑哭', emoji: '😂' },
  { code: '[眨眼]', name: '眨眼', emoji: '😉' },
  { code: '[色]', name: '色', emoji: '😍' },
  { code: '[亲亲]', name: '亲亲', emoji: '😘' },
  { code: '[害羞]', name: '害羞', emoji: '😳' },
  { code: '[得意]', name: '得意', emoji: '😎' },
  { code: '[酷]', name: '酷', emoji: '🆒' },
  { code: '[大哭]', name: '大哭', emoji: '😭' },
  { code: '[流泪]', name: '流泪', emoji: '😢' },
  { code: '[委屈]', name: '委屈', emoji: '😞' },
  { code: '[难过]', name: '难过', emoji: '😔' },
  { code: '[抓狂]', name: '抓狂', emoji: '😤' },
  { code: '[发怒]', name: '发怒', emoji: '😠' },
  { code: '[生气]', name: '生气', emoji: '😡' },
  { code: '[惊讶]', name: '惊讶', emoji: '😲' },
  { code: '[惊恐]', name: '惊恐', emoji: '😱' },
  { code: '[晕]', name: '晕', emoji: '😵' },
  { code: '[困]', name: '困', emoji: '😴' },
  { code: '[思考]', name: '思考', emoji: '🤔' },
  { code: '[疑问]', name: '疑问', emoji: '❓' },
  { code: '[闭嘴]', name: '闭嘴', emoji: '🤐' },
  { code: '[嘘]', name: '嘘', emoji: '🤫' },
  { code: '[鄙视]', name: '鄙视', emoji: '🙄' },
  { code: '[白眼]', name: '白眼', emoji: '🙄' },
  { code: '[赞]', name: '赞', emoji: '👍' },
  { code: '[弱]', name: '弱', emoji: '👎' },
  { code: '[握手]', name: '握手', emoji: '🤝' },
  { code: '[胜利]', name: '胜利', emoji: '✌️' },
  { code: '[OK]', name: 'OK', emoji: '👌' },
  { code: '[爱心]', name: '爱心', emoji: '❤️' },
  { code: '[玫瑰]', name: '玫瑰', emoji: '🌹' },
  { code: '[礼物]', name: '礼物', emoji: '🎁' },
  { code: '[蛋糕]', name: '蛋糕', emoji: '🎂' },
  { code: '[咖啡]', name: '咖啡', emoji: '☕' },
  { code: '[啤酒]', name: '啤酒', emoji: '🍺' },
  { code: '[干杯]', name: '干杯', emoji: '🥂' },
  { code: '[鼓掌]', name: '鼓掌', emoji: '👏' },
  { code: '[加油]', name: '加油', emoji: '💪' },
  { code: '[抱拳]', name: '抱拳', emoji: '🙏' },
  { code: '[再见]', name: '再见', emoji: '👋' },
  { code: '[好的]', name: '好的', emoji: '✅' },
  { code: '[不行]', name: '不行', emoji: '❌' },
  { code: '[星星]', name: '星星', emoji: '⭐' },
  { code: '[月亮]', name: '月亮', emoji: '🌙' },
  { code: '[太阳]', name: '太阳', emoji: '☀️' },
  { code: '[彩虹]', name: '彩虹', emoji: '🌈' },
  { code: '[烟花]', name: '烟花', emoji: '🎆' },
  { code: '[庆祝]', name: '庆祝', emoji: '🎉' },
  { code: '[红包]', name: '红包', emoji: '🧧' },
  { code: '[钱]', name: '钱', emoji: '💰' },
  { code: '[飞机]', name: '飞机', emoji: '✈️' },
  { code: '[汽车]', name: '汽车', emoji: '🚗' },
  { code: '[火车]', name: '火车', emoji: '🚂' },
  { code: '[轮船]', name: '轮船', emoji: '🚢' },
  { code: '[自行车]', name: '自行车', emoji: '🚲' },
  { code: '[电话]', name: '电话', emoji: '📞' },
  { code: '[邮件]', name: '邮件', emoji: '📧' },
  { code: '[电脑]', name: '电脑', emoji: '💻' },
  { code: '[手机]', name: '手机', emoji: '📱' },
  { code: '[音乐]', name: '音乐', emoji: '🎵' },
  { code: '[电影]', name: '电影', emoji: '🎬' },
  { code: '[游戏]', name: '游戏', emoji: '🎮' },
  { code: '[足球]', name: '足球', emoji: '⚽' },
  { code: '[篮球]', name: '篮球', emoji: '🏀' },
  { code: '[乒乓球]', name: '乒乓球', emoji: '🏓' },
]

function toggle() {
  visible.value = !visible.value
}

function selectEmoji(emoji) {
  emit('select', emoji.code)
  visible.value = false
}
</script>

<style scoped>
.emoji-picker {
  display: inline-block;
}

.emoji-btn {
  font-size: 20px;
  padding: 4px 8px;
  min-width: auto;
  width: auto;
  height: auto;
  line-height: 1;
}

.emoji-btn:hover {
  background-color: #f5f7fa;
}

.emoji-picker-content {
  overflow: visible;
}

.emoji-tabs {
  margin-bottom: 10px;
}

.emoji-grid {
  display: grid;
  grid-template-columns: repeat(8, 1fr);
  gap: 4px;
  padding: 8px;
  overflow: visible;
}

.emoji-item {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  font-size: 20px;
  cursor: pointer;
  border-radius: 4px;
  transition: background-color 0.2s;
  user-select: none;
}

.emoji-item:hover {
  background-color: #f0f0f0;
}

.emoji-item:active {
  background-color: #e0e0e0;
}

</style>

