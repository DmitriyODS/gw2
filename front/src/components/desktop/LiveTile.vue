<template>
  <span class="lt" :class="{ wide }">
    <!-- Стопка граней: видна одна, смена — вертикальным «переворотом», как у
         живых плиток Metro. -->
    <span class="lt-stack">
      <Transition name="lt-flip">
        <span v-if="!current" key="face-icon" class="lt-face lt-face-icon">
          <span class="material-symbols-outlined lt-icon">{{ icon }}</span>
        </span>
        <span v-else :key="`face-${current.key}`" class="lt-face lt-face-data" :class="current.tone">
          <span class="lt-value">{{ current.value }}</span>
          <span v-if="current.label" class="lt-label">{{ current.label }}</span>
        </span>
      </Transition>
    </span>

    <span class="lt-title">{{ title }}</span>
  </span>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = defineProps({
  title: { type: String, required: true },
  icon: { type: String, required: true },
  // Грани из desktop/liveTiles.js: [{ key, value, label, tone }].
  faces: { type: Array, default: () => [] },
  wide: { type: Boolean, default: false },
  // Порядковый номер плитки — задаёт задержку старта: плитки переворачиваются
  // вразнобой, а не строем.
  order: { type: Number, default: 0 },
  // Пауза: плитку тащат или на неё навели — читать мешать не надо.
  paused: { type: Boolean, default: false },
})

const PERIOD = 5000
// Небольшой разбег, чтобы плитки переворачивались вразнобой, а не строем.
const STAGGER = 400

// Кадр 0 — обычная плитка (иконка), дальше грани с данными.
const step = ref(0)
const frames = computed(() => props.faces.length + 1)
const current = computed(() => (step.value === 0 ? null : props.faces[step.value - 1] || null))

const reduced = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(prefers-reduced-motion: reduce)').matches
  : false

let timer = null

function stop() {
  clearTimeout(timer)
  timer = null
}

function schedule(delay) {
  stop()
  timer = setTimeout(() => {
    step.value = (step.value + 1) % frames.value
    schedule(PERIOD)
  }, delay)
}

/* Крутим, только когда есть что показывать и никто не мешает.
   Данные показываем СРАЗУ: меню «Пуск» открывается заново при каждом клике, и
   ждать первого переворота пользователю не приходится. При «уменьшить
   движение» первая грань просто остаётся статичной. */
watch(
  () => [frames.value, props.paused],
  () => {
    stop()
    if (frames.value < 2) {
      step.value = 0
      return
    }
    if (step.value === 0) step.value = 1
    if (reduced) return
    if (!props.paused) schedule(PERIOD + props.order * STAGGER)
  },
  { immediate: true },
)

onBeforeUnmount(stop)
</script>

<style scoped>
.lt {
  display: flex;
  flex-direction: column;
  align-items: stretch;
  justify-content: space-between;
  width: 100%;
  height: 100%;
  min-width: 0;
  gap: 4px;
}

.lt-stack {
  position: relative;
  flex: 1;
  min-height: 0;
  display: block;
  overflow: hidden;
}

.lt-face {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
}

.lt-face-icon { align-items: center; }

.lt-icon {
  font-size: 30px;
  color: var(--color-text);
}

/* Справа сверху у плитки может висеть бейдж — оставляем ему место, чтобы
   длинное значение не уезжало под него. */
.lt-face-data {
  gap: 2px;
  padding-right: 24px;
  text-align: left;
}

.lt-value {
  font-size: 17px;
  font-weight: 700;
  line-height: 1.15;
  color: var(--color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.lt-face-data.alert .lt-value { color: var(--color-error); }

.lt-label {
  font-size: 10.5px;
  line-height: 1.3;
  color: var(--color-text-dim);
  /* Мелкий кегль позволяет показать три строки: названия задач и событий
     длинные, а обрезать их жалко. */
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.lt.wide .lt-value { font-size: 19px; }
.lt.wide .lt-label { font-size: 11.5px; -webkit-line-clamp: 2; }

/* Подпись — слева снизу (кнопка-плитка центрирует текст, перебиваем явно),
   значок при этом центрирован по горизонтали. */
.lt-title {
  text-align: left;
  font-size: 12.5px;
  font-weight: 500;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Переворот: новая грань приезжает снизу, прежняя уходит вверх. */
.lt-flip-enter-active,
.lt-flip-leave-active {
  transition: opacity 0.28s ease, translate 0.36s cubic-bezier(0.2, 0, 0, 1);
}

.lt-flip-enter-from {
  opacity: 0;
  translate: 0 100%;
}

.lt-flip-leave-to {
  opacity: 0;
  translate: 0 -100%;
}

@media (prefers-reduced-motion: reduce) {
  .lt-flip-enter-active,
  .lt-flip-leave-active { transition: none; }
}
</style>
