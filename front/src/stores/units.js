import { defineStore } from 'pinia'
import { ref } from 'vue'
import { getActiveUnit, stopUnit as apiStop } from '@/api/units.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useAuthStore } from '@/stores/auth.js'

export const useUnitsStore = defineStore('units', () => {
  const activeUnit = ref(null)

  async function fetchActiveUnit() {
    try {
      activeUnit.value = await getActiveUnit()
    } catch {
      activeUnit.value = null
    }
  }

  function setActiveUnit(unit) { activeUnit.value = unit }
  function clearActiveUnit() { activeUnit.value = null }

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

  return { activeUnit, fetchActiveUnit, setActiveUnit, clearActiveUnit, startUnit, stop }
})
