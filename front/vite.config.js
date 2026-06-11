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
      // Звонки — Go-микросервис callsvc; более специфичный префикс ДОЛЖЕН
      // стоять раньше '/api', иначе запросы уйдут во Flask.
      '/api/calls': {
        target: 'http://localhost:8090',
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
