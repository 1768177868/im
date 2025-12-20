<template>
  <div class="watermark-config">
    <el-form
      ref="formRef"
      :model="formData"
      :rules="formRules"
      label-width="150px"
      label-position="left"
    >
      <el-form-item :label="$t('config.watermark_image')" prop="watermark_image_path">
        <div class="watermark-image-upload">
          <el-upload
            :auto-upload="false"
            :on-change="handleFileChange"
            :on-remove="handleRemove"
            :file-list="fileList"
            :limit="1"
            accept="image/*"
            list-type="picture"
            :show-file-list="true"
          >
            <el-button type="primary" :icon="Picture">{{ $t('config.watermark_upload') }}</el-button>
            <template #tip>
              <div class="el-upload__tip">{{ $t('config.watermark_image_tip') }}</div>
            </template>
          </el-upload>
          <div v-if="watermarkImageUrl" class="watermark-preview">
            <el-image
              :src="watermarkImageUrl"
              fit="contain"
              style="max-width: 200px; max-height: 100px; margin-top: 10px;"
            />
          </div>
        </div>
      </el-form-item>

      <el-form-item :label="$t('config.watermark_position')" prop="watermark_position">
        <el-select 
          v-model="formData.watermark_position" 
          :placeholder="$t('config.watermark_position_placeholder')"
          style="width: 100%"
        >
          <el-option :label="$t('config.watermark_position_top_left')" value="top-left" />
          <el-option :label="$t('config.watermark_position_top_right')" value="top-right" />
          <el-option :label="$t('config.watermark_position_bottom_left')" value="bottom-left" />
          <el-option :label="$t('config.watermark_position_bottom_right')" value="bottom-right" />
          <el-option :label="$t('config.watermark_position_center')" value="center" />
        </el-select>
      </el-form-item>

      <el-row :gutter="20">
        <el-col :span="12">
          <el-form-item :label="$t('config.watermark_opacity')" prop="watermark_opacity">
            <el-slider
              v-model="formData.watermark_opacity"
              :min="0"
              :max="255"
              :step="1"
              show-input
              :show-input-controls="false"
            />
            <div class="form-item-tip">{{ $t('config.watermark_opacity_tip') }}</div>
          </el-form-item>
        </el-col>
        <el-col :span="12">
          <el-form-item :label="$t('config.watermark_scale')" prop="watermark_scale">
            <el-slider
              v-model="formData.watermark_scale"
              :min="0.1"
              :max="1.0"
              :step="0.1"
              show-input
              :show-input-controls="false"
            />
            <div class="form-item-tip">{{ $t('config.watermark_scale_tip') }}</div>
          </el-form-item>
        </el-col>
      </el-row>

      <el-form-item>
        <el-button type="primary" @click="handleSubmit" :loading="submitting">
          {{ $t('common.save') }}
        </el-button>
      </el-form-item>
    </el-form>

    <!-- 图片裁剪对话框 -->
    <el-dialog
      v-model="showCropDialog"
      :title="$t('config.watermark_crop_title')"
      width="80%"
      :close-on-click-modal="false"
      @close="handleCropDialogClose"
    >
      <div class="crop-container">
        <img
          ref="cropImageRef"
          :src="cropImageSrc"
          style="display: block; max-width: 100%;"
        />
      </div>
      <template #footer>
        <el-button @click="handleCropDialogClose">{{ $t('common.cancel') }}</el-button>
        <el-button type="primary" @click="handleCropConfirm" :loading="cropping">
          {{ $t('common.confirm') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted, computed, onUnmounted, nextTick } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Picture } from '@element-plus/icons-vue'
import { getConfigByGroup, saveConfig } from '../../../api/config'
import Storage from '../../../utils/storage'
import axios from 'axios'
import Cropper from 'cropperjs'
import 'cropperjs/dist/cropper.css'

const { t } = useI18n()
const formRef = ref(null)
const submitting = ref(false)
const fileList = ref([])
const watermarkImageUrl = ref('') // 水印图片的blob URL
const showCropDialog = ref(false) // 是否显示裁剪对话框
const cropImageRef = ref(null) // 裁剪图片的DOM引用
const cropImageSrc = ref('') // 裁剪图片的源URL
const cropperInstance = ref(null) // Cropper实例
const cropping = ref(false) // 是否正在裁剪
const pendingFile = ref(null) // 待裁剪的文件

const formData = reactive({
  watermark_image_path: '',
  watermark_position: 'bottom-right',
  watermark_opacity: 128,
  watermark_scale: 0.3
})

const uploadAction = computed(() => {
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL || ''
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  return `${apiBaseURL}${apiPrefix}/attachments/upload`
})

const uploadHeaders = computed(() => {
  const token = Storage.getItem('token', '') || ''
  return {
    'Authorization': `Bearer ${token}`
  }
})

const formRules = {
  watermark_opacity: [
    { type: 'number', min: 0, max: 255, message: t('config.watermark_opacity_range'), trigger: 'blur' }
  ],
  watermark_scale: [
    { type: 'number', min: 0.1, max: 1.0, message: t('config.watermark_scale_range'), trigger: 'blur' }
  ]
}

// 处理文件选择（不直接上传，先打开裁剪对话框）
const handleFileChange = (file) => {
  const isImage = file.raw && file.raw.type.startsWith('image/')
  if (!isImage) {
    ElMessage.error(t('config.watermark_only_image_allowed'))
    fileList.value = []
    return
  }
  
  // 保存待裁剪的文件
  pendingFile.value = file.raw
  
  // 读取文件并显示裁剪对话框
  const reader = new FileReader()
  reader.onload = (e) => {
    cropImageSrc.value = e.target.result
    showCropDialog.value = true
    
    // 等待DOM更新后初始化Cropper
    nextTick(() => {
      initCropper()
    })
  }
  reader.readAsDataURL(file.raw)
}

// 初始化Cropper
const initCropper = () => {
  if (!cropImageRef.value) return
  
  // 销毁旧实例
  if (cropperInstance.value) {
    cropperInstance.value.destroy()
    cropperInstance.value = null
  }
  
  // 创建新实例
  cropperInstance.value = new Cropper(cropImageRef.value, {
    aspectRatio: NaN, // 不限制宽高比，允许自由裁剪
    viewMode: 1, // 限制裁剪框不能超出图片
    dragMode: 'move', // 默认拖拽模式为移动
    autoCropArea: 0.8, // 自动裁剪区域占80%
    restore: false,
    guides: true,
    center: true,
    highlight: false,
    cropBoxMovable: true,
    cropBoxResizable: true,
    toggleDragModeOnDblclick: false,
    zoomable: true,
    zoomOnTouch: true,
    zoomOnWheel: true,
    minCanvasWidth: 0,
    minCanvasHeight: 0,
    minCropBoxWidth: 10,
    minCropBoxHeight: 10,
    ready: function() {
      // Cropper初始化完成
    }
  })
}

// 确认裁剪并上传
const handleCropConfirm = async () => {
  if (!cropperInstance.value || !pendingFile.value) {
    return
  }
  
  cropping.value = true
  
  try {
    // 获取裁剪后的canvas
    const canvas = cropperInstance.value.getCroppedCanvas({
      imageSmoothingEnabled: true,
      imageSmoothingQuality: 'high'
    })
    
    // 转换为blob
    canvas.toBlob(async (blob) => {
      if (!blob) {
        ElMessage.error(t('config.watermark_crop_failed'))
        cropping.value = false
        return
      }
      
      // 创建FormData并上传
      const uploadFormData = new FormData()
      const filename = pendingFile.value.name || 'watermark.png'
      uploadFormData.append('file', blob, filename)
      
      try {
        const token = Storage.getItem('token', '') || ''
        const tokenStr = typeof token === 'string' ? token.trim() : ''
        
        const response = await axios.post(uploadAction.value, uploadFormData, {
          headers: {
            'Authorization': `Bearer ${tokenStr}`,
            'Content-Type': 'multipart/form-data'
          }
        })
        
        // 处理上传成功
        await handleUploadSuccess(response.data || response)
        
        // 关闭裁剪对话框
        handleCropDialogClose()
      } catch (error) {
        console.error('Upload error:', error)
        ElMessage.error(error.response?.data?.message || t('config.watermark_upload_failed'))
      } finally {
        cropping.value = false
      }
    }, 'image/png', 0.95) // 使用PNG格式，质量95%
  } catch (error) {
    console.error('Crop error:', error)
    ElMessage.error(t('config.watermark_crop_failed'))
    cropping.value = false
  }
}

const handleUploadSuccess = async (response) => {
  console.log('Upload response:', response) // 调试日志
  
  // Element Plus 的 on-success 回调可能直接传递 response.data，也可能传递整个 response
  // 需要兼容多种响应格式
  let data = response
  let attachment = null
  
  // 如果 response 有 data 属性，说明是完整的响应对象
  if (response && response.data) {
    data = response.data
  }
  
  // 检查响应格式
  if (data && data.code === 200) {
    // 标准格式：{ code: 200, data: { id: 27, ... } }
    attachment = data.data || data
  } else if (data && (data.id || data.ID)) {
    // 直接是附件对象：{ id: 27, filename: '...', ... }
    attachment = data
  } else if (response && (response.id || response.ID)) {
    // response 本身就是附件对象
    attachment = response
  }
  
  if (attachment) {
    const attachmentId = attachment.id || attachment.ID
    
    if (attachmentId) {
      // 保存附件ID，后端通过ID查找路径
      formData.watermark_image_path = `attachment:${attachmentId}`
      
      // 获取预览URL用于显示
      const fileUrl = attachment.file_url || attachment.FileURL || attachment.fileUrl || ''
      const previewUrl = fileUrl || `/api/admin/attachments/${attachmentId}/preview`
      
      // 先加载图片并转换为blob URL（因为预览接口需要JWT认证）
      await loadWatermarkImage(previewUrl)
      
      // 更新文件列表显示，使用blob URL作为缩略图
      fileList.value = [{
        name: attachment.filename || attachment.Filename || 'watermark.png',
        url: watermarkImageUrl.value || previewUrl, // 使用blob URL，如果没有则使用原始URL
        status: 'success'
      }]
      
      ElMessage.success(t('config.watermark_upload_success'))
      return
    }
  }
  
  // 如果到这里说明解析失败
  console.error('Failed to parse upload response:', response)
  ElMessage.error(data?.message || t('config.watermark_upload_failed'))
}

const handleUploadError = (error) => {
  ElMessage.error(t('config.watermark_upload_failed'))
  console.error('Upload error:', error)
}

// 关闭裁剪对话框
const handleCropDialogClose = () => {
  showCropDialog.value = false
  
  // 清理Cropper实例
  if (cropperInstance.value) {
    cropperInstance.value.destroy()
    cropperInstance.value = null
  }
  
  // 清理裁剪图片的blob URL
  if (cropImageSrc.value && cropImageSrc.value.startsWith('blob:')) {
    URL.revokeObjectURL(cropImageSrc.value)
  }
  cropImageSrc.value = ''
  pendingFile.value = null
  
  // 清空文件列表（因为还没有上传）
  fileList.value = []
}

const handleRemove = () => {
  // 释放blob URL
  if (watermarkImageUrl.value && watermarkImageUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(watermarkImageUrl.value)
  }
  watermarkImageUrl.value = ''
  formData.watermark_image_path = ''
  fileList.value = []
  pendingFile.value = null
  
  // 如果裁剪对话框打开，关闭它
  if (showCropDialog.value) {
    handleCropDialogClose()
  }
}

// 加载水印图片并转换为blob URL
const loadWatermarkImage = async (fileUrl) => {
  if (!fileUrl) {
    watermarkImageUrl.value = ''
    return
  }
  
  // 如果是外部URL（http/https），直接使用
  if (fileUrl.startsWith('http://') || fileUrl.startsWith('https://')) {
    watermarkImageUrl.value = fileUrl
    return
  }
  
  // 构建完整的URL
  const apiBaseURL = import.meta.env.VITE_API_BASE_URL || ''
  const apiPrefix = import.meta.env.VITE_API_PREFIX || '/api/admin'
  let fullUrl = fileUrl
  if (apiBaseURL) {
    const base = apiBaseURL.replace(/\/+$/, '')
    fullUrl = `${base}${fileUrl.startsWith('/') ? '' : '/'}${fileUrl}`
  } else {
    fullUrl = `${apiPrefix}${fileUrl.startsWith('/') ? '' : '/'}${fileUrl}`
  }
  
  // 通过axios获取图片并转换为blob URL
  // 因为预览接口需要JWT认证，直接使用src无法携带认证头
  try {
    const token = Storage.getItem('token', '') || ''
    const tokenStr = typeof token === 'string' ? token.trim() : ''
    const response = await axios.get(fullUrl, {
      responseType: 'blob',
      headers: {
        'Authorization': `Bearer ${tokenStr}`
      }
    })
    const blob = new Blob([response.data])
    const blobUrl = URL.createObjectURL(blob)
    
    // 如果之前有blob URL，先释放
    if (watermarkImageUrl.value && watermarkImageUrl.value.startsWith('blob:')) {
      URL.revokeObjectURL(watermarkImageUrl.value)
    }
    
    watermarkImageUrl.value = blobUrl
  } catch (error) {
    console.error('Failed to load watermark image:', error)
    watermarkImageUrl.value = ''
  }
}

// 获取水印图片URL（用于显示）
const getWatermarkImageUrl = () => {
  return watermarkImageUrl.value
}

const loadData = async () => {
  try {
    const res = await getConfigByGroup('watermark')
    if (res.data && res.data.configs) {
      const configs = res.data.configs
      configs.forEach(config => {
        const key = config.Key || config.key
        const value = config.Value || config.value || ''
        if (formData.hasOwnProperty(key)) {
          if (key === 'watermark_opacity') {
            formData[key] = parseInt(value) || 128
          } else if (key === 'watermark_scale') {
            formData[key] = parseFloat(value) || 0.3
          } else {
            formData[key] = value
          }
        }
      })
      
      // 如果有水印图片路径，加载图片并显示
      if (formData.watermark_image_path) {
        // 如果是 attachment:ID 格式，构建预览URL
        let previewUrl = ''
        if (formData.watermark_image_path.startsWith('attachment:')) {
          const attachmentId = formData.watermark_image_path.replace('attachment:', '')
          previewUrl = `/api/admin/attachments/${attachmentId}/preview`
        } else {
          previewUrl = formData.watermark_image_path
        }
        
        // 加载图片
        await loadWatermarkImage(previewUrl)
        
        // 更新文件列表显示，使用blob URL作为缩略图
        fileList.value = [{
          name: 'watermark.png',
          url: watermarkImageUrl.value || previewUrl, // 使用blob URL，如果没有则使用原始URL
          status: 'success'
        }]
      }
    }
  } catch (error) {
    console.error('Load watermark config error:', error)
  }
}

const handleSubmit = async () => {
  if (!formRef.value) return

  await formRef.value.validate(async (valid) => {
    if (!valid) return

    submitting.value = true
    try {
      const configs = {
        watermark_image_path: formData.watermark_image_path,
        watermark_position: formData.watermark_position,
        watermark_opacity: formData.watermark_opacity.toString(),
        watermark_scale: formData.watermark_scale.toString()
      }

      await saveConfig('watermark', configs)
      ElMessage.success(t('common.save_success'))
    } catch (error) {
      console.error('Save watermark config error:', error)
      ElMessage.error(t('common.save_failed'))
    } finally {
      submitting.value = false
    }
  })
}

onMounted(() => {
  loadData()
})

onUnmounted(() => {
  // 清理blob URL，避免内存泄漏
  if (watermarkImageUrl.value && watermarkImageUrl.value.startsWith('blob:')) {
    URL.revokeObjectURL(watermarkImageUrl.value)
  }
  
  // 清理Cropper实例
  if (cropperInstance.value) {
    cropperInstance.value.destroy()
    cropperInstance.value = null
  }
  
  // 清理裁剪图片的blob URL
  if (cropImageSrc.value && cropImageSrc.value.startsWith('blob:')) {
    URL.revokeObjectURL(cropImageSrc.value)
  }
})

defineExpose({
  loadData
})
</script>

<style scoped>
.watermark-config {
  padding: 20px;
}

.form-item-tip {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.5;
}

.watermark-image-upload {
  width: 100%;
}

.watermark-preview {
  margin-top: 10px;
}

.crop-container {
  width: 100%;
  min-height: 400px;
  max-height: 70vh;
  background: #f5f7fa;
  border-radius: 4px;
}

.crop-container img {
  max-width: 100%;
}
</style>

