<template>
  <div class="hp-backdrop" :class="{ full }" @pointerdown.self="close">
    <section class="hp" :class="{ full }" :style="style" role="dialog" aria-label="Hola ассистент">
      <header class="hp-bar">
        <HolaIcon :size="full ? 26 : 20" class="hp-mark" />
        <h2 class="hp-title">Hola ассистент</h2>
      </header>

      <HolaPanel ref="panelRef" class="hp-body" @navigate="close" />
    </section>
  </div>
</template>

<script setup>
/**
 * Всплывающая панель Hola поверх каркаса — наследница строки Spotlight, а не
 * окно раздела: ни кнопки в панели задач, ни своего адреса, ни кнопок
 * управления окном у неё нет. Появляется всегда по центру экрана, закрывается
 * клавишей Esc, кликом мимо панели и уходом в найденный раздел.
 *
 * full — мобильный каркас: панель занимает весь экран между панелями каркаса
 * (на телефоне «окно посреди экрана» смысла не имеет), и раскладку целиком
 * держит CSS — отступы под системные вырезы известны только ему.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import { taskbarReserve } from '@/desktop/layout.js'
import HolaPanel from '@/components/hola/HolaPanel.vue'
import HolaIcon from '@/components/common/HolaIcon.vue'

const SIZE = { w: 720, h: 720 }
const MARGIN = 24

const props = defineProps({
  full: { type: Boolean, default: false },
})

const emit = defineEmits(['close'])

const panelRef = ref(null)
const rect = ref(panelRect())

/* Центр экрана с поправкой на панель задач: на невысоком экране панель не
   должна наполовину уезжать под неё. */
function panelRect() {
  const vw = window.innerWidth
  const vh = window.innerHeight - taskbarReserve()
  const w = Math.min(SIZE.w, vw - MARGIN * 2)
  const h = Math.min(SIZE.h, vh - MARGIN * 2)
  return {
    x: Math.max(MARGIN, Math.round((vw - w) / 2)),
    y: Math.max(MARGIN, Math.round((vh - h) / 2)),
    w,
    h,
  }
}

const style = computed(() => (props.full ? {} : {
  transform: `translate3d(${rect.value.x}px, ${rect.value.y}px, 0)`,
  width: `${rect.value.w}px`,
  height: `${rect.value.h}px`,
}))

function close() {
  emit('close')
}

function onKeydown(e) {
  if (e.key === 'Escape') {
    e.stopPropagation()
    close()
  }
}

function onResize() {
  rect.value = panelRect()
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('resize', onResize, { passive: true })
  nextTick(() => panelRef.value?.focus())
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize', onResize)
})
</script>

<style scoped>
.hp-backdrop {
  position: fixed;
  inset: 0;
  z-index: 960;
  background: color-mix(in oklch, var(--color-text) 18%, transparent);
  -webkit-backdrop-filter: blur(2px);
  backdrop-filter: blur(2px);
  transition: opacity 0.18s ease;
}

/* Плавающий слой — единственное место, где размытие реально работает: внутри
   акриловой панели родитель становится backdrop root, и вложенное стекло уже
   ничего не размывает. */
.hp {
  position: absolute;
  top: 0;
  left: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: 0 24px 64px color-mix(in oklch, var(--color-text) 18%, transparent);
  transition: opacity 0.18s ease, translate 0.2s cubic-bezier(0.2, 0, 0, 1),
    scale 0.2s cubic-bezier(0.2, 0, 0, 1);
}

/* Мобильный каркас: панель во весь экран между панелями каркаса — без рамки и
   скруглений, как самостоятельный экран, и с более сильным размытием: под ней
   не однотонное окно, а обои и плитки стартового экрана. */
.hp.full {
  inset: calc(var(--statusbar-height, 0px) + env(safe-area-inset-top, 0px)) 0
    calc(var(--taskbar-height) + env(safe-area-inset-bottom, 0px)) 0;
  border: none;
  border-radius: 0;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur-strong);
  backdrop-filter: var(--acrylic-blur-strong);
  box-shadow: none;
}

/* Панель во весь экран остаётся ПОД панелью задач (900): её кнопка Hola —
   единственный способ закрыть панель на телефоне, и она обязана быть нажимаемой
   (по высоте панель до панели задач и не достаёт). */
.hp-backdrop.full {
  background: none;
  z-index: 890;
}

.hp-bar {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
  height: 46px;
  padding: 0 16px;
  border-bottom: 1px solid color-mix(in oklch, var(--acrylic-border) 70%, transparent);
  user-select: none;
}

/* Отступы — как у центра уведомлений: панель начинается под вырезом, поэтому
   сверху остаётся обычное поле. */
.hp.full .hp-bar {
  height: auto;
  padding: 18px 18px 4px;
  border-bottom: none;
}

.hp.full .hp-body { padding: 14px 18px 18px; }

.hp.full .hp-title { font-size: 1.15rem; }

.hp-mark { color: var(--color-primary); }

.hp-title {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.hp-body { flex: 1; min-height: 0; }

/* Низкий экран — почти всегда открытая клавиатура: шапка отдаёт панели свою
   высоту. Медиазапрос по вьюпорту тут уместен именно потому, что речь о нём:
   высота панели и на телефоне, и на десктопе идёт от высоты экрана, а сжимает
   её системная клавиатура (WebView перерисовывает вьюпорт). Компактную выдачу
   внутри решает уже сама панель — по своему размеру. */
@media (max-height: 560px) {
  .hp-bar { height: 34px; padding: 0 12px; }
  .hp.full .hp-bar { padding: 8px 12px 0; }
  .hp.full .hp-title { font-size: 1rem; }
  .hp.full .hp-body { padding: 0; }
}

/* Появление — как у прежней строки поиска: панель подаётся сверху и тает. */
.hp-enter-from,
.hp-leave-to { opacity: 0; }

.hp-enter-from .hp,
.hp-leave-to .hp {
  opacity: 0;
  translate: 0 -14px;
  scale: 0.97;
}
</style>
