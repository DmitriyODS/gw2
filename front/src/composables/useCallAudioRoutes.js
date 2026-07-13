import { ref, onMounted, onBeforeUnmount } from 'vue'
import {
  isNativeApp, audioListDevices, audioGetRoute, audioSetRoute, onAudioDevicesChanged,
} from '@/utils/nativeApp.js'

// Иконка/подпись для каждого аудио-маршрута (значения совпадают с routeOf в
// нативном NativeShellPlugin).
export const AUDIO_ROUTE_META = {
  earpiece:  { icon: 'phone_in_talk',   label: 'Телефон' },
  speaker:   { icon: 'volume_up',       label: 'Динамик' },
  wired:     { icon: 'headphones',      label: 'Наушники' },
  bluetooth: { icon: 'bluetooth_audio', label: 'Bluetooth' },
}

// Аудио-маршрутизация звонка на мобильной обёртке: список доступных выходов,
// текущий и переключение. В браузере/Electron — no-op (supported=false).
export function useCallAudioRoutes() {
  const supported = isNativeApp()
  const routes = ref([])
  const current = ref(null)
  let sub = null

  async function refresh() {
    if (!supported) return
    routes.value = await audioListDevices()
    current.value = await audioGetRoute()
  }

  async function setRoute(route) {
    if (!supported) return
    await audioSetRoute(route)
    await refresh()
  }

  onMounted(async () => {
    if (!supported) return
    await refresh()
    sub = await onAudioDevicesChanged(() => refresh())
  })

  onBeforeUnmount(() => { try { sub?.remove?.() } catch {} })

  return { supported, routes, current, refresh, setRoute }
}
