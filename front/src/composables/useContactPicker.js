import { ref, watch } from 'vue'
import { getDirectory } from '@/api/users.js'
import { useMessengerStore } from '@/stores/messenger.js'

// useContactPicker — единая логика выбора собеседника для всех модалок работы
// с сообщениями (новый чат, пересылка сообщения/поста, отправка заметки в чат).
//
// Правила (общие для всего мессенджера):
//   • по умолчанию и при поиске по ФИО — только те, с кем УЖЕ есть диалог;
//   • нового человека можно найти только глобальным поиском по ЛОГИНУ.
// Так список не превращается в каталог всех пользователей платформы.
export function useContactPicker() {
  const messenger = useMessengerStore()
  const q = ref('')
  const results = ref([])
  const loading = ref(false)
  let debounceTimer = null
  let seq = 0

  // Собеседники существующих 1:1 диалогов (без dev-чата поддержки и дублей).
  function baseContacts() {
    const seen = new Set()
    const out = []
    for (const c of messenger.conversations) {
      const u = c.other_user
      if (!u?.id || c.is_dev_chat || seen.has(u.id)) continue
      seen.add(u.id)
      out.push(u)
    }
    return out
  }

  function localFilter(list, query) {
    const ql = query.toLowerCase()
    return list.filter(
      (u) =>
        (u.fio || '').toLowerCase().includes(ql) ||
        (u.login || '').toLowerCase().includes(ql),
    )
  }

  async function run() {
    const query = q.value.trim()
    const mySeq = ++seq
    const base = baseContacts()
    if (!query) {
      results.value = base
      loading.value = false
      return
    }
    // По ФИО/логину сразу отфильтровали существующие диалоги.
    const local = localFilter(base, query)
    results.value = local
    // Плюс глобальный поиск строго по логину — новые собеседники.
    loading.value = true
    try {
      const found = await getDirectory(query, true, { global: true, byLogin: true })
      if (mySeq !== seq) return
      const have = new Set(local.map((u) => u.id))
      const extra = (found || []).filter((u) => !have.has(u.id))
      results.value = [...local, ...extra]
    } catch {
      /* сеть моргнула — оставляем локальные совпадения */
    } finally {
      if (mySeq === seq) loading.value = false
    }
  }

  // reset — вызывать при открытии модалки: подгружаем диалоги (если не были) и
  // показываем базовый список.
  async function reset() {
    q.value = ''
    seq++
    loading.value = false
    if (!messenger.conversations.length) {
      try { await messenger.fetchConversations() } catch { /* ignore */ }
    }
    results.value = baseContacts()
  }

  watch(q, () => {
    clearTimeout(debounceTimer)
    debounceTimer = setTimeout(run, 250)
  })

  return { q, results, loading, reset, run }
}
