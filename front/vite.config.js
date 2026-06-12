import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { readFileSync } from 'node:fs'
import { fileURLToPath, URL } from 'node:url'

// В проде /api/changelog отдаёт nginx статикой из back/data/changelog.json;
// в dev тот же файл отдаёт этот мини-плагин (мидлварь встаёт раньше прокси).
const changelogPath = fileURLToPath(new URL('../back/data/changelog.json', import.meta.url))
const serveChangelog = () => ({
  name: 'serve-changelog',
  configureServer(server) {
    server.middlewares.use('/api/changelog', (_req, res) => {
      res.setHeader('Content-Type', 'application/json')
      res.end(readFileSync(changelogPath))
    })
  }
})

export default defineConfig({
  plugins: [vue(), serveChangelog()],
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
      // Авторизация, пользователи, роли и бэкап — authsvc.
      '/api/auth': {
        target: 'http://localhost:8091',
        changeOrigin: true
      },
      '/api/users': {
        target: 'http://localhost:8091',
        changeOrigin: true
      },
      '/api/roles': {
        target: 'http://localhost:8091',
        changeOrigin: true
      },
      '/api/backup': {
        target: 'http://localhost:8091',
        changeOrigin: true
      },
      // Presence остаётся во Flask (in-memory), остальной мессенджер — msgsvc.
      '/api/messenger/presence': {
        target: 'http://localhost:5001',
        changeOrigin: true
      },
      '/api/messenger': {
        target: 'http://localhost:8092',
        changeOrigin: true
      },
      // «Мой Groove» — groovesvc.
      '/api/groove': {
        target: 'http://localhost:8094',
        changeOrigin: true
      },
      // Ядро задач — tasksvc (задачи, юниты, типы, этапы, отделы, статистика).
      '/api/tasks': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      '/api/units': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      '/api/unit-types': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      '/api/departments': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      '/api/stages': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      '/api/stats': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      // YouGile-интеграция — тоже tasksvc.
      '/api/yougile': {
        target: 'http://localhost:8095',
        changeOrigin: true
      },
      // ТВ-факт дня — aisvc.
      '/api/ai': {
        target: 'http://localhost:8093',
        changeOrigin: true
      },
      // ИИ-настройки компаний — aisvc; ключ с '^' vite трактует как RegExp.
      // Стоит ДО префикса '/api/companies', иначе ai-settings уйдут в authsvc.
      '^/api/companies/\\d+/ai-settings': {
        target: 'http://localhost:8093',
        changeOrigin: true
      },
      // Остальные компании — authsvc.
      '/api/companies': {
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
