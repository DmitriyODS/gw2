import { computed, ref } from 'vue'
import { getScreenLock, unlockScreen } from '@/api/auth.js'

/* Экран блокировки: пока человека нет за устройством, приложение закрыто
   пин-кодом, а СЕССИЯ жива — иначе блокировка теряла бы открытые окна,
   черновики и позицию в разделах, и ей бы просто не пользовались.

   Состояние модульное (одно на приложение): блокировку показывает App.vue,
   а включают её в настройках — оба должны видеть одно и то же.

   «Заперто» держится в sessionStorage, а не в памяти: перезагрузка страницы
   не должна открывать запертое приложение. Само снятие проверяет СЕРВЕР —
   правкой хранилища его не обойти. */

const LOCKED_KEY = 'gw_screen_locked'

const enabled = ref(false)
const afterMin = ref(null)
const locked = ref(readLocked())
const ready = ref(false)

function readLocked() {
  try {
    return sessionStorage.getItem(LOCKED_KEY) === '1'
  } catch {
    return false
  }
}

function persistLocked(value) {
  try {
    if (value) sessionStorage.setItem(LOCKED_KEY, '1')
    else sessionStorage.removeItem(LOCKED_KEY)
  } catch { /* приватный режим — блокировка проживёт до перезагрузки */ }
}

// Бездействие: любое действие человека сдвигает отсчёт.
let idleTimer = null

function stopIdle() {
  clearTimeout(idleTimer)
  idleTimer = null
}

function restartIdle() {
  stopIdle()
  if (!enabled.value || locked.value || !afterMin.value) return
  idleTimer = setTimeout(() => lock(), afterMin.value * 60_000)
}

function lock() {
  if (!enabled.value) return
  locked.value = true
  persistLocked(true)
  stopIdle()
}

export function useScreenLock() {
  /** Состояние с сервера: включена ли блокировка и через сколько запирать. */
  async function load() {
    try {
      const state = await getScreenLock()
      enabled.value = !!state.enabled
      afterMin.value = state.after_min ?? null
      // Блокировку выключили с другого устройства — запертый экран снимаем.
      if (!enabled.value && locked.value) {
        locked.value = false
        persistLocked(false)
      }
      restartIdle()
    } catch { /* не удалось узнать — работаем без блокировки */ } finally {
      ready.value = true
    }
  }

  /** Снять экран: пин-код либо пароль от аккаунта. */
  async function unlock(secret) {
    await unlockScreen(secret)
    locked.value = false
    persistLocked(false)
    restartIdle()
  }

  /** Отметить активность: сдвигает отсчёт бездействия. */
  function touch() {
    if (!enabled.value || locked.value || !afterMin.value) return
    restartIdle()
  }

  /** Состояние изменилось в настройках — перечитываем и перезаводим таймер. */
  function apply(state) {
    enabled.value = !!state?.enabled
    afterMin.value = state?.after_min ?? null
    if (!enabled.value) {
      locked.value = false
      persistLocked(false)
    }
    restartIdle()
  }

  function reset() {
    stopIdle()
    enabled.value = false
    afterMin.value = null
    locked.value = false
    persistLocked(false)
  }

  return {
    enabled: computed(() => enabled.value),
    afterMin: computed(() => afterMin.value),
    locked: computed(() => locked.value),
    ready: computed(() => ready.value),
    load, unlock, lock, touch, apply, reset,
  }
}
