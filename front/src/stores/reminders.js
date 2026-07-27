import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/reminders.js'
import { logActivity } from '@/utils/activityLog.js'

/* Напоминания пользователя: список раздела, ближайшие для живой плитки и
   приём сработавших. Само срабатывание считает сервер (планировщик
   remindersvc) — клиент лишь показывает событие reminder:fire и даёт отложить
   или отметить сделанным.

   Списки обновляются upsert'ом по id: сокет-событие приходит раньше ответа на
   HTTP-запрос, поэтому слепой push плодил бы дубли. */
export const useRemindersStore = defineStore('reminders', () => {
  const items = ref([])          // активные
  const done = ref([])           // журнал сработавших
  const upcoming = ref([])       // ближайшие (плитка/центр уведомлений)
  // Сработавшие прямо сейчас — их показывает баннер, пока не закроют.
  const fired = ref([])

  const loading = ref(false)

  const activeCount = computed(() => items.value.length)
  const nextReminder = computed(() => upcoming.value[0] || items.value[0] || null)

  async function fetchAll() {
    loading.value = true
    try {
      const [act, journal] = await Promise.all([api.getReminders('active'), api.getReminders('done')])
      items.value = act.items ?? []
      done.value = journal.items ?? []
    } finally {
      loading.value = false
    }
  }

  async function fetchUpcoming(limit = 10) {
    const data = await api.getUpcoming(limit)
    upcoming.value = data.items ?? []
    return upcoming.value
  }

  async function create(body) {
    const created = await api.createReminder(body)
    upsert(created)
    logActivity({ kind: 'reminder', id: created.id, title: created.title })
    return created
  }

  async function update(id, body) {
    const updated = await api.updateReminder(id, body)
    upsert(updated)
    return updated
  }

  async function remove(id) {
    await api.deleteReminder(id)
    drop(id)
  }

  async function snooze(id, minutes = 10) {
    const updated = await api.snoozeReminder(id, minutes)
    upsert(updated)
    dismissFired(id)
    return updated
  }

  async function complete(id) {
    const updated = await api.completeReminder(id)
    upsert(updated)
    dismissFired(id)
    return updated
  }

  /** Привязанные к записи ежедневника/календаря — для актуализации снимка. */
  async function forRecord(kind, recordId) {
    const data = await api.getLinked(kind, recordId)
    return data.items ?? []
  }

  /**
   * Запись раздела переехала на другое время — двигаем её напоминания,
   * сохраняя «за сколько минут до» и подхватывая новое название.
   * Зовут ежедневник и календарь после сохранения записи.
   */
  async function syncLinked(kind, recordId, { eventAt, title } = {}) {
    if (!eventAt) return
    const linked = await forRecord(kind, recordId)
    await Promise.all(linked.map((r) => {
      const lead = r.link?.lead_minutes || 0
      const remindAt = new Date(new Date(eventAt).getTime() - lead * 60_000).toISOString()
      const link = { ...r.link, ...(title != null ? { title } : {}) }
      return update(r.id, { remind_at: remindAt, link }).catch(() => { /* не критично */ })
    }))
  }

  // ── Сокет ──
  /** Сработало: показываем баннер и обновляем списки. */
  function applyFire(payload) {
    if (!payload?.id) return
    fired.value = [payload, ...fired.value.filter((f) => f.id !== payload.id)].slice(0, 5)
  }

  function applySocket(kind, payload) {
    if (!payload?.id) return
    if (kind === 'deleted') drop(payload.id)
    else upsert(payload)
  }

  function dismissFired(id) {
    fired.value = fired.value.filter((f) => f.id !== id)
  }

  function upsert(r) {
    // Активные и журнал — две стороны одного списка: где напоминание живёт,
    // решает его active, поэтому при каждом обновлении переносим между ними.
    const target = r.active ? items : done
    const other = r.active ? done : items
    other.value = other.value.filter((x) => x.id !== r.id)
    const idx = target.value.findIndex((x) => x.id === r.id)
    if (idx >= 0) target.value[idx] = { ...target.value[idx], ...r }
    else target.value.push(r)
    if (r.active) {
      target.value.sort((a, b) => new Date(a.remind_at) - new Date(b.remind_at))
    } else {
      target.value.sort((a, b) => new Date(b.remind_at) - new Date(a.remind_at))
    }
    upcoming.value = upcoming.value.filter((x) => x.id !== r.id)
    if (r.active) upcoming.value = [...upcoming.value, r].sort((a, b) => new Date(a.remind_at) - new Date(b.remind_at))
  }

  function drop(id) {
    items.value = items.value.filter((x) => x.id !== id)
    done.value = done.value.filter((x) => x.id !== id)
    upcoming.value = upcoming.value.filter((x) => x.id !== id)
    dismissFired(id)
  }

  function reset() {
    items.value = []
    done.value = []
    upcoming.value = []
    fired.value = []
  }

  return {
    items, done, upcoming, fired, loading,
    activeCount, nextReminder,
    fetchAll, fetchUpcoming, create, update, remove, snooze, complete,
    forRecord, syncLinked,
    applyFire, applySocket, dismissFired, reset,
  }
})
