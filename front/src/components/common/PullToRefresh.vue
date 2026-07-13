<template>
  <Teleport to="body">
    <div
      v-if="active && (pull > 0 || refreshing)"
      class="ptr"
      :style="{ transform: `translate(-50%, ${indicatorY}px)`, opacity }"
    >
      <span
        class="material-symbols-outlined ptr-ico"
        :class="{ spin: refreshing }"
        :style="refreshing ? {} : { transform: `rotate(${arrowDeg}deg)` }"
      >{{ refreshing ? 'progress_activity' : 'arrow_downward' }}</span>
    </div>
  </Teleport>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'

/* Pull-to-refresh для мобильной версии: оттяжка вниз у верха активного
   скролл-контейнера обновляет страницу (location.reload). Работает на всех
   экранах — цепляется на touch-события документа и сам находит контейнер
   прокрутки под пальцем. Гейт `active` (обычно !fullscreen-роут) отключает
   жест в звонке/лайтбоксе, где вертикальные жесты заняты. */
const props = defineProps({
  active: { type: Boolean, default: true },
})

const THRESHOLD = 72        // px оттяжки для срабатывания
const MAX_PULL = 120        // потолок визуальной оттяжки
const RESISTANCE = 0.5      // «резина»: палец проходит вдвое больше индикатора

// Жест доступен только на тач-устройствах (мобильная версия/обёртка).
const isTouch = typeof window !== 'undefined'
  && window.matchMedia?.('(hover: none) and (pointer: coarse)').matches

const pull = ref(0)
const refreshing = ref(false)

let startX = 0
let startY = 0
let tracking = false      // палец на экране, начали у верха
let engaged = false       // оттяжка признана вертикальной — перехватываем скролл
let scroller = null

const indicatorY = computed(() => Math.min(pull.value, MAX_PULL) - 8)
const opacity = computed(() => Math.min(1, pull.value / THRESHOLD))
const arrowDeg = computed(() => (pull.value >= THRESHOLD ? 180 : 0))

// Ближайший вертикально-прокручиваемый предок точки касания.
function scrollableAt(el) {
  let node = el
  while (node && node !== document.body && node !== document.documentElement) {
    if (node.nodeType === 1) {
      const style = getComputedStyle(node)
      const oy = style.overflowY
      if ((oy === 'auto' || oy === 'scroll') && node.scrollHeight > node.clientHeight + 1) {
        return node
      }
    }
    node = node.parentNode
  }
  return document.scrollingElement || document.documentElement
}

function atTop(el) {
  if (!el) return true
  return (el.scrollTop || 0) <= 0
}

function onTouchStart(e) {
  if (!props.active || refreshing.value || e.touches.length !== 1) return
  // Игнорируем зоны, где вертикальные жесты заняты (звонок, лайтбокс, карта и т.п.).
  if (e.target.closest?.('[data-no-ptr]')) return
  scroller = scrollableAt(e.target)
  if (!atTop(scroller)) return
  startX = e.touches[0].clientX
  startY = e.touches[0].clientY
  tracking = true
  engaged = false
}

function onTouchMove(e) {
  if (!tracking || refreshing.value) return
  const dx = e.touches[0].clientX - startX
  const dy = e.touches[0].clientY - startY
  if (!engaged) {
    // Ещё не решили: горизонталь/вверх — отдаём жест штатному поведению.
    if (dy <= 0 || Math.abs(dx) > Math.abs(dy)) { tracking = false; return }
    if (dy < 6) return
    if (!atTop(scroller)) { tracking = false; return }
    engaged = true
  }
  if (dy <= 0) { pull.value = 0; return }
  // Перехватываем: гасим нативный overscroll/скролл, тянем индикатор с резиной.
  if (e.cancelable) e.preventDefault()
  pull.value = Math.min(dy * RESISTANCE, MAX_PULL)
}

function onTouchEnd() {
  if (!tracking) return
  tracking = false
  if (pull.value >= THRESHOLD && !refreshing.value) {
    refreshing.value = true
    pull.value = THRESHOLD
    // Даём индикатору отрисоваться, затем перезагружаем страницу.
    setTimeout(() => { try { window.location.reload() } catch { /* no-op */ } }, 150)
    return
  }
  pull.value = 0
  engaged = false
}

onMounted(() => {
  if (!isTouch) return
  document.addEventListener('touchstart', onTouchStart, { passive: true })
  // Non-passive: внутри preventDefault'им оттяжку, чтобы не сработал нативный overscroll.
  document.addEventListener('touchmove', onTouchMove, { passive: false })
  document.addEventListener('touchend', onTouchEnd, { passive: true })
  document.addEventListener('touchcancel', onTouchEnd, { passive: true })
})

onBeforeUnmount(() => {
  document.removeEventListener('touchstart', onTouchStart)
  document.removeEventListener('touchmove', onTouchMove)
  document.removeEventListener('touchend', onTouchEnd)
  document.removeEventListener('touchcancel', onTouchEnd)
})
</script>

<style scoped>
.ptr {
  position: fixed;
  top: max(8px, env(safe-area-inset-top, 0px));
  left: 50%;
  z-index: 10060; /* выше плавающих хабов/питомца */
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  background: var(--acrylic-bg-strong, var(--color-surface-high));
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--shadow-md);
  pointer-events: none;
  transition: opacity 0.15s;
}

.ptr-ico {
  font-size: 22px;
  color: var(--color-primary);
  transition: transform 0.15s;
}

.ptr-ico.spin {
  animation: ptr-spin 0.8s linear infinite;
}

@keyframes ptr-spin {
  to { transform: rotate(360deg); }
}
</style>
