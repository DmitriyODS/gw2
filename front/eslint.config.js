import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import prettierOff from 'eslint-config-prettier/flat'
import globals from 'globals'

/* Линтер ловит ошибки и опасные приёмы, а не стиль: правила чистого
   форматирования (переносы атрибутов, отступы шаблона) сняты последним слоем
   eslint-config-prettier — раскладка кода остаётся за автором и ревью. */
export default [
  { ignores: ['dist/**', 'coverage/**', 'public/**', '!public/sw.js'] },

  js.configs.recommended,
  ...pluginVue.configs['flat/recommended'],

  {
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      globals: { ...globals.browser },
    },
    rules: {
      // Пустой catch — осознанный приём проекта (fail-open: localStorage,
      // необязательные запросы), поэтому пустые блоки допустимы только в нём.
      'no-empty': ['error', { allowEmptyCatch: true }],
      'no-unused-vars': ['error', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        caughtErrors: 'none',
        // `const { skip, ...rest } = o` — идиома «всё, кроме поля».
        ignoreRestSiblings: true,
      }],
    },
  },

  {
    // Сервисный воркер живёт в своём глобальном окружении.
    files: ['public/sw.js'],
    languageOptions: { globals: { ...globals.serviceworker } },
  },

  {
    // Конфиги сборки и интеграционный стенд исполняются Node.
    files: ['*.config.js', 'tests/**/*.js'],
    languageOptions: { globals: { ...globals.node } },
  },

  {
    // Тесты исполняет vitest в Node (jsdom поверх, но process/global — Node).
    files: ['**/*.spec.js', 'tests/**/*.js'],
    languageOptions: { globals: { ...globals.node, ...globals.vitest } },
  },

  prettierOff,
]
