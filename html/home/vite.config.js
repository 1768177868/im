import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  build: {
    outDir: './dist',
    emptyOutDir: true,
    rollupOptions: {
      input: {
        chat: resolve(__dirname, 'chat.html'),
        button: resolve(__dirname, 'button.html')
      }
    }
  },
  server: {
    port: 3006
  }
})

