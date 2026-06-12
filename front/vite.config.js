import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    proxy: {
      // Go-микросервисы; более специфичные префиксы ДОЛЖНЫ стоять раньше
      // '/api', иначе запросы уйдут во Flask.
      '/api/calls': {
        target: 'http://localhost:8090',
        changeOrigin: true
      },
      // Авторизация и пользователи — authsvc.
      '/api/auth': {
        target: 'http://localhost:8091',
        changeOrigin: true
      },
      '/api/users': {
        target: 'http://localhost:8091',
        changeOrigin: true
      },
      '/api': {
        target: 'http://localhost:5001',
        changeOrigin: true
      },
      '/uploads': {
        target: 'http://localhost:5001',
        changeOrigin: true
      },
      '/socket.io': {
        target: 'http://localhost:5001',
        changeOrigin: true,
        ws: true
      }
    }
  }
})
