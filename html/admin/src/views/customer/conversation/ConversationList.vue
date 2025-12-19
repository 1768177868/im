<template>
  <div class="conversation-list">
    <el-card>
      <template #header>
        <div class="card-header">
          <span>{{ $t('customer.conversation.title') }}</span>
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

      <!-- 搜索表单 -->
      <SearchForm
        :model="searchForm"
        :fields="searchFields"
        :initial-values="initialSearchValues"
        i18n-prefix="customer.conversation"
        @search="handleSearch"
        @reset="handleReset"
      />

      <!-- 会话列表 -->
      <vxe-table
        ref="tableRef"
        :data="tableData"
        :loading="loading"
        border
        :column-config="{ resizable: true }"
        height="600"
        @sort-change="handleSortChange"
      >
        <vxe-column type="seq" width="60" :title="$t('common.index')" />
        <vxe-column field="id" :title="$t('customer.conversation.id')" width="80" />
        <vxe-column field="visitor.name" :title="$t('customer.conversation.visitor_name')" width="150">
          <template #default="{ row }">
            {{ row.visitor?.name || row.visitor?.visitor_id || `访客${row.visitor_id}` }}
          </template>
        </vxe-column>
        <vxe-column field="admin.nickname" :title="$t('customer.conversation.admin_name')" width="120">
          <template #default="{ row }">
            {{ row.admin?.nickname || '-' }}
          </template>
        </vxe-column>
        <vxe-column field="title" :title="$t('customer.conversation.title')" min-width="200" />
        <vxe-column field="status" :title="$t('customer.conversation.status')" width="100">
          <template #default="{ row }">
            <el-tag :type="getStatusType(row.status)">
              {{ getStatusText(row.status) }}
            </el-tag>
          </template>
        </vxe-column>
        <vxe-column field="last_message_at" :title="$t('customer.conversation.last_message_at')" width="160" sortable>
          <template #default="{ row }">
            {{ formatTime(row.last_message_at) }}
          </template>
        </vxe-column>
        <vxe-column field="created_at" :title="$t('common.created_at')" width="160" sortable>
          <template #default="{ row }">
            {{ formatTime(row.created_at) }}
          </template>
        </vxe-column>
        <vxe-column :title="$t('common.operation')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button 
              type="primary" 
              link
              @click="handleView(row)"
            >
              {{ $t('common.view') }}
            </el-button>
            <el-button 
              v-if="row.status === 1"
              type="warning" 
              link
              :disabled="getButtonState('customer.conversation.end').disabled"
              @click="handleEnd(row)"
            >
              {{ $t('customer.conversation.end') }}
            </el-button>
          </template>
        </vxe-column>
      </vxe-table>

      <Pagination
        v-model="pagination"
        @page-change="handlePageChange"
      />
    </el-card>

    <!-- 会话详情对话框 -->
    <el-dialog
      v-model="detailDialogVisible"
      :title="$t('customer.conversation.detail')"
      width="80%"
      :close-on-click-modal="false"
      class="conversation-detail-dialog"
    >
      <ConversationDetail
        v-if="detailDialogVisible && currentConversationId"
        :conversation-id="currentConversationId"
        @close="detailDialogVisible = false"
      />
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, markRaw } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Refresh } from '@element-plus/icons-vue'
import { getConversations, endConversation } from '@/api/customer'
import { getOptions } from '@/api/option'
import SearchForm from '@/components/SearchForm.vue'
import Pagination from '@/components/Pagination.vue'
import ConversationDetail from './ConversationDetail.vue'
import { useListPage } from '@/composables/useListPage'
import { usePermission } from '@/composables/usePermission'
import ErrorHandler from '@/utils/errorHandler'
import logger from '@/utils/logger'

const RefreshIcon = markRaw(Refresh)
const { t } = useI18n()
const { getButtonState } = usePermission()

const tableRef = ref(null)
const detailDialogVisible = ref(false)
const currentConversationId = ref(null)

// 搜索表单初始值（提取为常量，避免每次渲染创建新对象导致表单被重置）
const initialSearchValues = {
  conversation_id: '',
  visitor_name: '',
  status: '',
  admin_id: ''
}

// 字段名映射
const fieldMapping = {
  'id': 'id',
  'status': 'status',
  'created_at': 'created_at',
  'last_message_at': 'last_message_at'
}

// 使用列表页面 composable
const {
  pagination,
  tableData,
  loading,
  searchForm,
  loadData,
  handleSearch,
  handleReset,
  handlePageChange,
  handleSortChange,
  initDefaultSort
} = useListPage({
  fetchApi: getConversations,
  initialSearchForm: {
    conversation_id: '',
    visitor_name: '',
    status: '',
    admin_id: ''
  },
  sortOptions: {
    tableRef,
    fieldMapping,
    defaultSort: 'last_message_at:desc'
  }
})

// 搜索字段
const searchFields = computed(() => [
  {
    prop: 'conversation_id',
    label: t('customer.conversation.id'),
    type: 'input',
    props: {
      placeholder: t('customer.conversation.id_placeholder')
    }
  },
  {
    prop: 'visitor_name',
    label: t('customer.conversation.visitor_name'),
    type: 'input',
    props: {
      placeholder: t('customer.conversation.visitor_name_placeholder')
    }
  },
  {
    prop: 'status',
    label: t('customer.conversation.status'),
    type: 'select',
    options: [
      { label: t('common.all'), value: '' },
      { label: t('customer.conversation.status_active'), value: '1' },
      { label: t('customer.conversation.status_ended'), value: '2' },
      { label: t('customer.conversation.status_closed'), value: '3' }
    ],
    props: {
      clearable: true
    }
  },
  {
    prop: 'admin_id',
    label: t('customer.conversation.admin'),
    type: 'select',
    options: adminOptions.value,
    props: {
      filterable: true,
      clearable: true
    }
  }
])

const adminOptions = ref([])

// 获取状态文本
const getStatusText = (status) => {
  const statusMap = {
    1: t('customer.conversation.status_active'),
    2: t('customer.conversation.status_ended'),
    3: t('customer.conversation.status_closed')
  }
  return statusMap[status] || '-'
}

// 获取状态类型
const getStatusType = (status) => {
  const typeMap = {
    1: 'success',
    2: 'info',
    3: 'danger'
  }
  return typeMap[status] || ''
}

// 格式化时间
const formatTime = (timeStr) => {
  if (!timeStr) return '-'
  const date = new Date(timeStr)
  return date.toLocaleString('zh-CN')
}

// 查看会话详情
const handleView = (row) => {
  currentConversationId.value = row.id
  detailDialogVisible.value = true
}

// 结束会话
const handleEnd = async (row) => {
  try {
    await ElMessageBox.confirm(
      t('customer.conversation.end_confirm'),
      t('common.confirm'),
      {
        type: 'warning'
      }
    )

    const response = await endConversation({
      conversation_id: row.id
    })

    if (response.code === 200) {
      ElMessage.success(t('customer.conversation.end_success'))
      loadData()
    }
  } catch (error) {
    if (error !== 'cancel') {
      ErrorHandler.handle(error)
    }
  }
}

// 刷新
const handleRefresh = () => {
  loadData()
}

// 加载管理员选项（只加载客服人员）
const loadAdminOptions = async () => {
  try {
    const response = await getOptions('admin', { customer_service_only: true })
    if (response.code === 200) {
      // 后端返回格式: { options: [{ label: "...", value: "..." }] }
      adminOptions.value = response.data?.options || []
    }
  } catch (error) {
    logger.error('加载管理员选项失败:', error)
  }
}

onMounted(() => {
  loadAdminOptions()
  initDefaultSort?.()
  loadData()
})
</script>

<style scoped>
.conversation-list {
  padding: 20px;
}

.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
</style>

<style>
/* 会话详情对话框样式 - 需要全局样式来覆盖 Dialog 的默认样式 */
.conversation-detail-dialog .el-dialog__body {
  padding: 0;
  height: 70vh;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.conversation-detail-dialog .conversation-detail {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}
</style>

