import { computed, ref } from 'vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { usePortalStore } from '@/stores/portal.js'
import { usePetsStore } from '@/stores/pets.js'
import { storageGetJSON, storageSetJSON } from '@/utils/storage.js'

/* Центр уведомлений рабочего стола. Два рода строк:

   • ЖУРНАЛ (`gw_notif_journal`) — события, которые случились один раз и нигде
     больше не хранятся: сработавшее напоминание, входящий перевод кудосов,
     упоминание в задаче, побег грувика. Живут в localStorage, поэтому
     ПЕРЕЖИВАЮТ ПЕРЕЗАГРУЗКУ страницы; убираются насовсем крестиком.
   • ЖИВЫЕ строки — производные от сторов состояния: непрочитанные чаты, новые
     публикации портала, болезнь питомца. Их «убрать» значит скрыть до
     следующего изменения: у каждой есть отпечаток состояния (счётчик), и с
     новым событием она возвращается.

   Живёт композаблом, а не в панели, потому что то же самое считает бейдж на
   кнопке уведомлений — а панель смонтирована, только пока открыта. Состояние
   модульное (одно на всё приложение): иначе бейдж и панель разошлись бы. */

const DISMISS_KEY = 'gw_desktop_notif_dismissed'
const JOURNAL_KEY = 'gw_notif_journal'

const JOURNAL_MAX = 40
const JOURNAL_TTL_MS = 7 * 24 * 60 * 60 * 1000

const dismissed = ref(storageGetJSON(DISMISS_KEY, {}))
const journal = ref(loadJournal())

function loadJournal() {
  const list = storageGetJSON(JOURNAL_KEY, [])
  if (!Array.isArray(list)) return []
  const edge = Date.now() - JOURNAL_TTL_MS
  return list.filter((e) => e?.at > edge).slice(0, JOURNAL_MAX)
}

const saveJournal = () => storageSetJSON(JOURNAL_KEY, journal.value)

/**
 * Записать событие в центр уведомлений. Зовут обработчики сокета и сторы —
 * рядом с тостом, который живёт лишь несколько секунд.
 * key — стабильный идентификатор события: повтор поднимает прежнюю строку
 * вместо дубля (например, то же напоминание после «отложить»).
 */
export function pushNotification({ key, icon = 'notifications', tone = 'primary', title, text, path }) {
  const at = Date.now()
  const entry = { journal: true, key: key || `n-${at}`, icon, tone, title, text, path, at }
  journal.value = [entry, ...journal.value.filter((e) => e.key !== entry.key)].slice(0, JOURNAL_MAX)
  saveJournal()
}

/** Полная очистка (выход из системы: чужие уведомления новому хозяину не нужны). */
export function clearNotificationJournal() {
  journal.value = []
  saveJournal()
}

export function useDesktopNotifications() {
  const messenger = useMessengerStore()
  const portal = usePortalStore()
  const pets = usePetsStore()

  const live = computed(() => {
    const list = []

    for (const c of messenger.conversations) {
      if (!c.unread_count) continue
      const title = c.is_group ? (c.title || 'Группа') : (c.other_user?.fio || 'Чат')
      list.push({
        key: `conv-${c.id}`,
        sig: `${c.unread_count}`,
        icon: 'forum',
        tone: 'primary',
        title,
        text: c.unread_count === 1 ? 'Новое сообщение' : `Новых сообщений: ${c.unread_count}`,
        path: `/messenger/${c.id}`,
      })
    }

    if (portal.unread) {
      list.push({
        key: 'portal',
        sig: `${portal.unread}`,
        icon: 'web_stories',
        tone: 'primary',
        title: 'Портал',
        text: `Новых публикаций: ${portal.unread}`,
        path: '/portal',
      })
    }

    const pet = pets.pet
    if (pet?.sick) {
      list.push({
        key: 'pet-sick',
        sig: `sick-${pet.ailment || 'any'}`,
        icon: 'pets',
        tone: 'alert',
        title: 'Питомец заболел',
        text: pet.runaway_in_days ? `Нужно лечение — сбежит через ${pet.runaway_in_days} дн.` : 'Нужно лечение',
        path: '/pets',
      })
    }

    return list.filter((i) => dismissed.value[i.key] !== i.sig)
  })

  // Журнал сверху (у него есть время события), под ним — текущее состояние.
  const items = computed(() => [...journal.value, ...live.value])

  const count = computed(() => items.value.length)

  function dismiss(item) {
    if (item.journal) {
      journal.value = journal.value.filter((e) => e.key !== item.key)
      saveJournal()
      return
    }
    dismissed.value = { ...dismissed.value, [item.key]: item.sig }
    storageSetJSON(DISMISS_KEY, dismissed.value)
  }

  function clearAll() {
    const next = { ...dismissed.value }
    for (const i of live.value) next[i.key] = i.sig
    dismissed.value = next
    storageSetJSON(DISMISS_KEY, dismissed.value)
    clearNotificationJournal()
  }

  return { items, count, dismiss, clearAll }
}
