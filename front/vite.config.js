import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { createReadStream, existsSync, readFileSync } from 'node:fs'
import { join, normalize } from 'node:path'
import { fileURLToPath, URL } from 'node:url'

// В проде /api/changelog отдаёт nginx статикой из data/changelog.json;
// в dev тот же файл отдаёт этот мини-плагин (мидлварь встаёт раньше прокси).
const changelogPath = fileURLToPath(new URL('../data/changelog.json', import.meta.url))
const serveChangelog = () => ({
  name: 'serve-changelog',
  configureServer(server) {
    server.middlewares.use('/api/changelog', (_req, res) => {
      res.setHeader('Content-Type', 'application/json')
      res.end(readFileSync(changelogPath))
    })
  }
})

// В проде /uploads/ отдаёт nginx из общего volume; в dev файлы (аватарки,
// вложения) пишут Go-сервисы в каталог uploads/ корня репо — отдаём оттуда.
const uploadsDir = fileURLToPath(new URL('../uploads', import.meta.url))
const serveUploads = () => ({
  name: 'serve-uploads',
  configureServer(server) {
    server.middlewares.use('/uploads', (req, res, next) => {
      const rel = normalize(decodeURIComponent((req.url || '').split('?')[0]))
      const file = join(uploadsDir, rel)
      if (!file.startsWith(uploadsDir) || !existsSync(file)) return next()
      createReadStream(file).pipe(res)
    })
  }
})

// В проде /apps/ (APK мобильного приложения + version.json с номером сборки)
// отдаёт nginx из каталога apps/ репозитория; в dev — этот мини-плагин.
const appsDir = fileURLToPath(new URL('../apps', import.meta.url))
const serveApps = () => ({
  name: 'serve-apps',
  configureServer(server) {
    server.middlewares.use('/apps', (req, res, next) => {
      const rel = normalize(decodeURIComponent((req.url || '').split('?')[0]))
      const file = join(appsDir, rel)
      if (!file.startsWith(appsDir) || !existsSync(file)) return next()
      if (file.endsWith('.json')) res.setHeader('Content-Type', 'application/json')
      else if (file.endsWith('.apk')) res.setHeader('Content-Type', 'application/vnd.android.package-archive')
      createReadStream(file).pipe(res)
    })
  }
})

export default defineConfig({
  build: {
    // Дефолт Vite 8 — Chrome 111+ (март 2023): заводской Android System WebView
    // на новых устройствах часто старее, бандл падает с SyntaxError до старта
    // Vue — вечный белый экран в мобильной обёртке. Держим планку Vite 5
    // ('modules'): esbuild дотранспилирует синтаксис, LightningCSS по этому же
    // target'у понижает CSS.
    target: ['es2020', 'chrome87', 'edge88', 'firefox78', 'safari14']
  },
  plugins: [vue(), serveChangelog(), serveUploads(), serveApps()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    }
  },
  server: {
    proxy: {
      // Go-микросервисы; более специфичные префиксы стоят раньше общих.
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
      // Presence — домен realtime-шлюза gatewaysvc, остальной мессенджер — msgsvc.
      '/api/messenger/presence': {
        target: 'http://localhost:8096',
        changeOrigin: true
      },
      '/api/messenger': {
        target: 'http://localhost:8092',
        changeOrigin: true
      },
      // Питомцы-грувики — petsvc.
      '/api/pets': {
        target: 'http://localhost:8094',
        changeOrigin: true
      },
      // Пуш-уведомления — pushsvc (регистрация токенов устройств).
      '/api/push': {
        target: 'http://localhost:8097',
        changeOrigin: true
      },
      // Реестры — registrysvc (таблицы-справочники компаний).
      '/api/registries': {
        target: 'http://localhost:8099',
        changeOrigin: true
      },
      // Календари — calendarsvc (списки записей с датой/временем).
      '/api/calendars': {
        target: 'http://localhost:8100',
        changeOrigin: true
      },
      // Ежедневники — diarysvc (личные заметки-задачи по дням).
      '/api/diaries': {
        target: 'http://localhost:8101',
        changeOrigin: true
      },
      // Заметки — notesvc (личные rich-заметки с группами и шарингом).
      '/api/notes': {
        target: 'http://localhost:8103',
        changeOrigin: true
      },
      // Корпоративный портал — portalsvc (посты, комментарии, реакции, разделы).
      '/api/portal': {
        target: 'http://localhost:8102',
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
      }
      // WS realtime-шлюза в dev фронт открывает напрямую (ws://localhost:8096/ws,
      // см. src/socket/index.js) — прокси не нужен.
    }
  }
})
