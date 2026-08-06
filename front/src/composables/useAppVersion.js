import { ref, computed } from 'vue'
import { changelogApi } from '@/api/changelog.js'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'

/**
 * Сведения о выпуске: версия, сборка, дата и краткое «что нового».
 *
 * ЕДИНСТВЕННЫЙ источник истины — data/changelog.json на сервере (его же читают
 * сборки Android и десктопа), а не захардкоженный package.json в бандле.
 * Истории версий платформа больше не ведёт: показываем только текущий выпуск —
 * его печатает раздел «Настройки → О приложении».
 */

// Module-level singleton: выпуск один на всё приложение, любой компонент
// может его прочитать, запрос выполняется однократно.
//
// Марка «Groove Work N» стоит на КАЖДОМ экране входа и в меню «Пуск», поэтому
// без кэша ради одной цифры мажорной версии запрос уходил на каждом заходе.
// Выпуск меняется только с деплоем: первый кадр рисуем из localStorage, сеть
// трогаем лишь когда кэш пуст или устарел (в «О приложении» — принудительно,
// там пользователь как раз проверяет версию).
const CACHE_KEY = 'gw_release'
const CACHE_TTL = 6 * 60 * 60 * 1000

const cached = storageGetJSON(CACHE_KEY, null)
const release = ref(cached?.data || null)
let fetchedAt = Number(cached?.at) || 0
let loadPromise = null

async function load({ force = false } = {}) {
  if (!force && release.value && Date.now() - fetchedAt < CACHE_TTL) return release.value
  if (!loadPromise) {
    loadPromise = changelogApi
      .get()
      .then((data) => {
        release.value = data || null
        fetchedAt = Date.now()
        loadPromise = null
        if (release.value) storageSetJSON(CACHE_KEY, { at: fetchedAt, data: release.value })
        return release.value
      })
      .catch(() => {
        loadPromise = null // дать повторить попытку позже
        return release.value
      })
  }
  return loadPromise
}

const version = computed(() => release.value?.version || null)
const build = computed(() => release.value?.build || null)
/** Мажорная версия для марки продукта: «Groove Work 7». */
const majorVersion = computed(() => version.value?.split('.')[0] || '')

export function useAppVersion() {
  return { release, version, build, majorVersion, load }
}
