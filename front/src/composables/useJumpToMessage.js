import { nextTick, ref } from 'vue'

const FLASH_MS = 1500
// Страховка от бесконечной догрузки на гигантских историях.
const MAX_PAGES = 40

/**
 * Переход к сообщению в ленте чата (клик по reply-цитате, закреплённым):
 * плавный скролл к пузырю + кратковременная подсветка. Если сообщение ещё
 * не подгружено — догружает историю страницами, пока оно не появится.
 *
 * container — ref на scroll-контейнер с [data-msg-id] внутри;
 * getMessages/hasMore/loadOlder — доступ к стору активного диалога.
 */
export function useJumpToMessage({ container, getMessages, hasMore, loadOlder }) {
  const jumping = ref(false)

  function findRow(id) {
    return container.value?.querySelector(`[data-msg-id="${id}"]`) || null
  }

  function flash(row) {
    row.classList.remove('msg-flash')
    void row.offsetWidth // перезапуск css-анимации при повторном переходе
    row.classList.add('msg-flash')
    setTimeout(() => row.classList.remove('msg-flash'), FLASH_MS)
  }

  async function jumpToMessage(id) {
    if (!id || jumping.value) return false
    jumping.value = true
    try {
      const inStore = () => (getMessages() || []).some((m) => m.id === id)
      for (let page = 0; !inStore() && hasMore() && page < MAX_PAGES; page++) {
        const msgs = getMessages()
        if (!msgs?.length) break
        const added = await loadOlder(msgs[0].id)
        if (!added?.length) break
      }
      if (!inStore()) return false
      await nextTick()
      const row = findRow(id)
      if (!row) return false
      row.scrollIntoView({ behavior: 'smooth', block: 'center' })
      flash(row)
      return true
    } finally {
      jumping.value = false
    }
  }

  return { jumping, jumpToMessage }
}
