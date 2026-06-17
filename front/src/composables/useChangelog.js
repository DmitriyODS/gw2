import { ref } from 'vue'
import { changelogApi } from '@/api/changelog.js'
import { storageGet, storageSet } from '@/utils/storage.js'

const STORAGE_KEY = 'gw2_last_seen_version'

// Module-level singleton: окно лога одно на всё приложение, любой компонент
// может его открыть, а автопоказ управляется из App.vue.
const isOpen = ref(false)
// Текущая версия продукта — ЕДИНСТВЕННЫЙ источник истины это сервер (первая
// запись data/changelog.json), а не захардкоженный package.json в бандле.
const latestVersion = ref(null)

// Однократная загрузка версии: дедупим параллельные вызовы из разных
// компонентов (футер настроек, экран «О приложении»).
let latestPromise = null

function open() {
  isOpen.value = true
}

// Подтягивает текущую версию с сервера (не открывая окно лога). Идемпотентна:
// результат кэшируется на время жизни приложения.
async function loadLatest() {
  if (latestVersion.value) return latestVersion.value
  if (!latestPromise) {
    latestPromise = changelogApi
      .get()
      .then((data) => {
        latestVersion.value = data?.versions?.[0]?.version ?? null
        return latestVersion.value
      })
      .catch(() => {
        latestPromise = null // дать повторить попытку позже
        return null
      })
  }
  return latestPromise
}

function close() {
  isOpen.value = false
  // Запоминаем просмотренную версию, чтобы лог не всплывал повторно.
  if (latestVersion.value) {
    storageSet(STORAGE_KEY, latestVersion.value)
  }
}

// Сравнивает последнюю версию из лога с просмотренной пользователем и при
// расхождении открывает окно «Что нового». Вызывать после входа в систему.
async function checkForNewVersion() {
  const latest = await loadLatest()
  if (!latest) return

  const seen = storageGet(STORAGE_KEY, null)
  if (seen !== latest) {
    isOpen.value = true
  }
}

export function useChangelog() {
  return { isOpen, latestVersion, open, close, loadLatest, checkForNewVersion }
}
