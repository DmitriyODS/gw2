import { ref } from 'vue'
import { changelogApi } from '@/api/changelog.js'

const STORAGE_KEY = 'gw2_last_seen_version'

// Module-level singleton: окно лога одно на всё приложение, любой компонент
// может его открыть, а автопоказ управляется из App.vue.
const isOpen = ref(false)
const latestVersion = ref(null)

function open() {
  isOpen.value = true
}

function close() {
  isOpen.value = false
  // Запоминаем просмотренную версию, чтобы лог не всплывал повторно.
  if (latestVersion.value) {
    try { localStorage.setItem(STORAGE_KEY, latestVersion.value) } catch {}
  }
}

// Сравнивает последнюю версию из лога с просмотренной пользователем и при
// расхождении открывает окно «Что нового». Вызывать после входа в систему.
async function checkForNewVersion() {
  try {
    const data = await changelogApi.get()
    const latest = data?.versions?.[0]?.version
    if (!latest) return
    latestVersion.value = latest

    let seen = null
    try { seen = localStorage.getItem(STORAGE_KEY) } catch {}

    if (seen !== latest) {
      isOpen.value = true
    }
  } catch {
    // Лог изменений некритичен — молча пропускаем при ошибке.
  }
}

export function useChangelog() {
  return { isOpen, latestVersion, open, close, checkForNewVersion }
}
