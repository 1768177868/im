/**
 * 图片压缩工具
 * 使用 Canvas API 压缩图片，减少文件大小
 */

/**
 * 压缩图片
 * @param {File} file - 原始图片文件
 * @param {Object} options - 压缩选项
 * @param {number} options.maxWidth - 最大宽度，默认 1920
 * @param {number} options.maxHeight - 最大高度，默认 1920
 * @param {number} options.quality - 压缩质量 (0-1)，默认 0.8
 * @param {number} options.maxSize - 最大文件大小（字节），超过此大小才压缩，默认 500KB
 * @returns {Promise<File>} 压缩后的图片文件
 */
export function compressImage(file, options = {}) {
  return new Promise((resolve, reject) => {
    // 默认配置
    const config = {
      maxWidth: 1920,
      maxHeight: 1920,
      quality: 0.8,
      maxSize: 500 * 1024, // 500KB
      ...options
    }

    // 如果文件大小小于 maxSize，不压缩
    if (file.size <= config.maxSize) {
      resolve(file)
      return
    }

    // 检查是否为图片文件
    if (!file.type.startsWith('image/')) {
      reject(new Error('文件不是图片类型'))
      return
    }

    // 创建 FileReader 读取文件
    const reader = new FileReader()
    
    reader.onload = (e) => {
      const img = new Image()
      
      img.onload = () => {
        try {
          // 计算压缩后的尺寸
          let width = img.width
          let height = img.height
          
          // 如果图片尺寸超过最大尺寸，按比例缩放
          if (width > config.maxWidth || height > config.maxHeight) {
            const ratio = Math.min(
              config.maxWidth / width,
              config.maxHeight / height
            )
            width = width * ratio
            height = height * ratio
          }

          // 创建 Canvas
          const canvas = document.createElement('canvas')
          canvas.width = width
          canvas.height = height
          
          // 绘制图片到 Canvas
          const ctx = canvas.getContext('2d')
          ctx.drawImage(img, 0, 0, width, height)

          // 转换为 Blob
          canvas.toBlob(
            (blob) => {
              if (!blob) {
                reject(new Error('图片压缩失败'))
                return
              }

              // 创建新的 File 对象
              const compressedFile = new File(
                [blob],
                file.name,
                {
                  type: file.type,
                  lastModified: Date.now()
                }
              )

              // 如果压缩后文件反而更大，使用原文件
              if (compressedFile.size >= file.size) {
                resolve(file)
              } else {
                resolve(compressedFile)
              }
            },
            file.type,
            config.quality
          )
        } catch (error) {
          console.error('图片压缩错误:', error)
          // 压缩失败时返回原文件
          resolve(file)
        }
      }

      img.onerror = () => {
        reject(new Error('图片加载失败'))
      }

      // 加载图片
      img.src = e.target.result
    }

    reader.onerror = () => {
      reject(new Error('文件读取失败'))
    }

    // 读取文件
    reader.readAsDataURL(file)
  })
}

