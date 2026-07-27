import { ref, computed } from 'vue'
import { changelogApi } from '@/api/changelog.js'

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
const release = ref(null)
let loadPromise = null

async function load() {
  if (release.value) return release.value
  if (!loadPromise) {
    loadPromise = changelogApi
      .get()
      .then((data) => {
        release.value = data || null
        return release.value
      })
      .catch(() => {
        loadPromise = null // дать повторить попытку позже
        return null
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
