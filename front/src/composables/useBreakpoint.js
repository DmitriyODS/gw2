import { ref, onMounted, onUnmounted } from 'vue'

const MOBILE_BP = 768

export function useBreakpoint() {
  const isMobile = ref(typeof window !== 'undefined' ? window.innerWidth <= MOBILE_BP : false)

  function update() {
    isMobile.value = window.innerWidth <= MOBILE_BP
  }

  onMounted(() => window.addEventListener('resize', update, { passive: true }))
  onUnmounted(() => window.removeEventListener('resize', update))

  return { isMobile }
}
