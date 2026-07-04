import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// Отдельный конфиг тестов (vitest предпочитает его vite.config.js и НЕ мержит
// с ним) — поэтому здесь заново объявляем vue-плагин и alias '@' → src.
// Проекты разделены: `unit` — компонентные и логические юнит-тесты рядом с
// кодом (`src/**/*.spec.js`); место под `integration` (tests/integration/**)
// оставлено второму тест-агенту — он допишет проект, переиспользуя эту оснастку.
const alias = { '@': fileURLToPath(new URL('./src', import.meta.url)) }

export default defineConfig({
  plugins: [vue()],
  resolve: { alias },
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./tests/setup.js'],
    projects: [
      {
        extends: true,
        test: {
          name: 'unit',
          include: ['src/**/*.spec.js'],
        },
      },
      {
        extends: true,
        test: {
          name: 'integration',
          include: ['tests/integration/**/*.spec.js'],
          // Реальный бэкенд: поднять сервисы (globalSetup), поставить fetch-шим
          // и jsdom-заглушки (harness). Сценарии ходят на общую БД —
          // последовательно (не форкать воркеры параллельно).
          globalSetup: ['./tests/integration/setup/backend.js'],
          setupFiles: ['./tests/setup.js', './tests/integration/setup/harness.js'],
          fileParallelism: false,
          testTimeout: 30000,
          hookTimeout: 300000,
        },
      },
    ],
  },
})
