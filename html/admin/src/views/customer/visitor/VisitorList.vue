<template>
  <div class="visitor-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ $t('customer.visitor.title') }}</span>
          <el-button 
            type="primary" 
            :icon="Refresh"
            @click="handleRefresh"
            :loading="loading"
          >
            {{ $t('common.refresh') }}
          </el-button>
        </div>
      </template>

      <!-- 在线访客列表 -->
      <vxe-table
        ref="tableRef"
        :data="tableData"
        :loading="loading"
        border
        :column-config="{ resizable: true }"
        height="600"
      >
        <vxe-column type="seq" width="60" :title="$t('common.index')" />
        <vxe-column field="visitor_id" :title="$t('customer.visitor.visitor_id')" width="200" />
        <vxe-column field="name" :title="$t('customer.visitor.name')" width="120" />
        <vxe-column field="email" :title="$t('customer.visitor.email')" width="180" />
        <vxe-column field="phone" :title="$t('customer.visitor.phone')" width="120" />
        <vxe-column field="ip" :title="$t('customer.visitor.ip')" width="150" />
        <vxe-column field="location" :title="$t('customer.visitor.location')" width="150" />
        <vxe-column field="device" :title="$t('customer.visitor.device')" width="100" />
        <vxe-column field="browser" :title="$t('customer.visitor.browser')" width="120" />
        <vxe-column field="os" :title="$t('customer.visitor.os')" width="120" />
        <vxe-column field="status" :title="$t('customer.visitor.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="row.status === 1 ? 'success' : 'info'">
              {{ row.status === 1 ? $t('customer.visitor.status_online') : $t('customer.visitor.status_offline') }}
            </el-tag>
          </template>
        </vxe-column>
        <vxe-column field="last_active_at" :title="$t('customer.visitor.last_active_at')" width="160" sortable>
          <template #default="{ row }">
            {{ formatTime(row.last_active_at) }}
          </template>
        </vxe-column>
        <vxe-column field="created_at" :title="$t('common.created_at')" width="160" sortable>
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </vxe-column>
      </vxe-table>
    </el-card>
  </div>
</template>

<script setup>
import { ref, onMounted, markRaw } from 'vue'
import { useI18n } from 'vue-i18n'
import { Refresh } from '@element-plus/icons-vue'
import { getOnlineVisitors } from '@/api/customer'
import ErrorHandler from '@/utils/errorHandler'

const RefreshIcon = markRaw(Refresh)
const { t } = useI18n()

const tableRef = ref(null)
const loading = ref(false)
const tableData = ref([])

// 加载数据
const loadData = async () => {
  loading.value = true
  try {
    const response = await getOnlineVisitors()
    if (response.code === 200) {
      tableData.value = response.data || []
    }
  } catch (error) {
    ErrorHandler.handle(error)
  } finally {
    loading.value = false
  }
}

// 刷新
const handleRefresh = () => {
  loadData()
}

// 格式化时间
const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

onMounted(() => {
  loadData()
  // 每30秒自动刷新
  setInterval(() => {
    loadData()
  }, 30000)
})
</script>

<style scoped>
.visitor-list {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

