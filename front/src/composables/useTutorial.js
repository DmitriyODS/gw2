import { ref } from 'vue'

const STORAGE_KEY = 'gw_tutorial_done'

// Module-level singleton so any component can open/close the tutorial
const isOpen = ref(false)

export function useTutorial() {
  function open() {
    isOpen.value = true
  }

  function close() {
    localStorage.setItem(STORAGE_KEY, '1')
    isOpen.value = false
  }

  function shouldAutoShow() {
    return !localStorage.getItem(STORAGE_KEY)
  }

  return { isOpen, open, close, shouldAutoShow }
}
