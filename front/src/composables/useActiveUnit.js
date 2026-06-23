import { ref } from 'vue'
import { useUnitsStore } from '@/stores/units.js'
import { useNotificationsStore } from '@/stores/notifications.js'

/**
 * Общие действия над активным юнитом для модалки и свёрнутого баннера:
 * завершение (с тостами) и сворачивание/разворачивание.
 */
export function useActiveUnit() {
  const unitsStore = useUnitsStore()
  const notifications = useNotificationsStore()
  const stopping = ref(false)

  async function stop() {
    stopping.value = true
    try {
      await unitsStore.stop()
      notifications.success('Юнит успешно завершён')
    } catch (e) {
      notifications.error(e?.message || 'Не удалось завершить юнит')
    } finally {
      stopping.value = false
    }
  }

  return {
    stopping,
    stop,
    minimize: unitsStore.minimize,
    expand: unitsStore.expand,
  }
}
