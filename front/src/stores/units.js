import { defineStore } from 'pinia'
import { ref, watch } from 'vue'
import { getActiveUnit, stopUnit as apiStop } from '@/api/units.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useAuthStore } from '@/stores/auth.js'

export const useUnitsStore = defineStore('units', () => {
  const activeUnit = ref(null)
  // Свёрнут ли активный юнит: false — крупная модалка, true — яркий баннер сверху.
  const minimized = ref(false)

  async function fetchActiveUnit() {
    try {
      activeUnit.value = await getActiveUnit()
    } catch {
      activeUnit.value = null
    }
  }

  function minimize() { minimized.value = true }
  function expand() { minimized.value = false }

  // Свёрнутый юнит не даёт о себе забыть: через 30 минут после сворачивания
  // модалка разворачивается сама. Свернёт снова — таймер пойдёт заново,
  // так что напоминание повторяется каждые 30 минут.
  const REMIND_MS = 30 * 60 * 1000
  let remindTimer = null
  watch([activeUnit, minimized], ([unit, min]) => {
    clearTimeout(remindTimer)
    if (unit && min) remindTimer = setTimeout(expand, REMIND_MS)
  })

  // Сбрасываем сворачивание только при СМЕНЕ юнита (новый запуск): обновления
  // того же юнита (unit:updated) приходят тем же путём и не должны разворачивать
  // модалку обратно.
  function setActiveUnit(unit) {
    if (unit && unit.id !== activeUnit.value?.id) minimized.value = false
    activeUnit.value = unit
  }
  function clearActiveUnit() {
    activeUnit.value = null
    minimized.value = false
  }

  // Аватарка активного пользователя для карточки: берём из юнита, иначе из профиля.
  function _activeUserFromUnit(unit) {
    if (unit?.user) {
      return { id: unit.user.id, fio: unit.user.fio, avatar_path: unit.user.avatar_path ?? null }
    }
    const me = useAuthStore().user
    return me ? { id: me.id, fio: me.fio, avatar_path: me.avatar_path ?? null } : null
  }

  // Запуск юнита текущим пользователем — сразу отражаем на карточке задачи,
  // не дожидаясь сокет-события (оно может прийти раньше попадания задачи
  // в список или с задержкой). Дедупликация в tasks-store делает это безопасным
  // даже при последующем дублирующем сокет-событии.
  function startUnit(unit) {
    activeUnit.value = unit
    minimized.value = false
    if (unit?.task_id) {
      const tasks = useTasksStore()
      tasks.patchTask({ id: unit.task_id, has_units: true })
      const u = _activeUserFromUnit(unit)
      if (u) tasks.addActiveUser(unit.task_id, u)
    }
  }

  async function stop() {
    if (!activeUnit.value) return
    const { id, task_id, user_id } = activeUnit.value
    await apiStop(id)
    activeUnit.value = null
    if (task_id && user_id != null) {
      useTasksStore().removeActiveUser(task_id, user_id)
    }
  }

  return { activeUnit, minimized, fetchActiveUnit, setActiveUnit, clearActiveUnit, minimize, expand, startUnit, stop }
})
