import { ref } from 'vue'
import { storageGet, storageSet } from '@/utils/storage.js'

const STORAGE_KEY = 'gw_tutorial_done'

// Module-level singleton so any component can open/close the tutorial
const isOpen = ref(false)
// Если задан — тур стартует с шага с таким id (используется для «показать
// в туре» из справки). Очищается при close().
const startAtId = ref(null)

export function useTutorial() {
  function open(opts = {}) {
    startAtId.value = opts.startAt || null
    isOpen.value = true
  }

  function close() {
    storageSet(STORAGE_KEY, '1')
    isOpen.value = false
    startAtId.value = null
  }

  function shouldAutoShow() {
    return !storageGet(STORAGE_KEY, '')
  }

  return { isOpen, startAtId, open, close, shouldAutoShow }
}
