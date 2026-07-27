<script setup>
/* Кастомный выбор времени (24ч) в стиле приложения — замена браузерному
   <input type="time">. v-model — строка 'HH:MM' (или null/'' когда не задано).
   Две прокручиваемые колонки часов/минут, клик-снаружи закрывает.

   Список часов/минут телепортируется в body и позиционируется fixed по рамке
   поля: внутри диалогов (у них overflow: hidden) абсолютный поповер обрезался
   бы и распирал макет. */
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'
import { placeByAnchor } from '@/utils/menuPlacement.js'

const props = defineProps({
  modelValue: { type: [String, null], default: null },
  placeholder: { type: String, default: 'Время' },
  clearable: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  minuteStep: { type: Number, default: 5 },
  icon: { type: String, default: 'schedule' },
})
const emit = defineEmits(['update:modelValue'])

const root = ref(null)
const popEl = ref(null)
const hoursCol = ref(null)
const minutesCol = ref(null)
const open = ref(false)
const popStyle = ref({})

const valid = computed(() => /^([01]\d|2[0-3]):[0-5]\d$/.test(props.modelValue || ''))
const cur = computed(() => {
  if (!valid.value) return { h: null, m: null }
  const [h, m] = props.modelValue.split(':').map(Number)
  return { h, m }
})

const hours = Array.from({ length: 24 }, (_, i) => i)
const minutes = computed(() => {
  const step = Math.min(Math.max(props.minuteStep, 1), 30)
  return Array.from({ length: Math.ceil(60 / step) }, (_, i) => i * step)
})

const pad = (n) => String(n).padStart(2, '0')

function commit(h, m) {
  emit('update:modelValue', `${pad(h)}:${pad(m)}`)
}
function pickHour(h) { commit(h, cur.value.m ?? 0) }
function pickMinute(m) { commit(cur.value.h ?? 9, m) }

function clear() {
  emit('update:modelValue', null)
  open.value = false
}

async function toggle() {
  if (props.disabled) return
  open.value = !open.value
  if (!open.value) return
  // Позицию задаём сразу, ещё до появления списка: у fixed-элемента без
  // координат нет «своего» места — он уехал бы в конец страницы.
  const anchor = root.value?.getBoundingClientRect()
  if (anchor) popStyle.value = { left: `${anchor.left}px`, top: `${anchor.bottom + 6}px` }
  await nextTick()
  place()          // уточняем по реальным размерам (может флипнуться вверх)
  scrollToSelected()
}

// Позиция поповера считается от рамки поля: он живёт в body, поэтому
// собственного «родителя» для absolute у него нет.
function place() {
  const anchor = root.value?.getBoundingClientRect()
  if (!anchor || !popEl.value) return
  const { left, top, maxHeight } = placeByAnchor(popEl.value, anchor, { gap: 6, align: 'left' })
  popStyle.value = { left: `${left}px`, top: `${top}px`, maxHeight: `${maxHeight}px` }
}

// Прокрутка колонок к выбранному значению — своими руками, БЕЗ scrollIntoView:
// он листает и внешние контейнеры (диалог, страницу), а его событие scroll
// сразу же прилетало в обработчик ниже и захлопывало только что открытый список.
function scrollToSelected() {
  for (const col of [hoursCol.value, minutesCol.value]) {
    const el = col?.querySelector('.tp-opt.active')
    if (el) col.scrollTop = el.offsetTop - col.clientHeight / 2 + el.offsetHeight / 2
  }
}

function onClickOutside(e) {
  if (!open.value) return
  const inField = root.value?.contains(e.target)
  const inPopup = popEl.value?.contains(e.target)
  if (!inField && !inPopup) open.value = false
}

// Прокрутка страницы и ресайз уводят fixed-поповер от поля — пересчитываем
// позицию. Прокрутку внутри самого списка часов/минут пропускаем.
function onViewportChange(e) {
  if (!open.value) return
  if (e?.target && popEl.value?.contains(e.target)) return
  place()
}

watch(() => props.modelValue, () => { if (open.value) nextTick(scrollToSelected) })

onMounted(() => {
  document.addEventListener('mousedown', onClickOutside)
  window.addEventListener('scroll', onViewportChange, true)
  window.addEventListener('resize', onViewportChange)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onClickOutside)
  window.removeEventListener('scroll', onViewportChange, true)
  window.removeEventListener('resize', onViewportChange)
})
</script>

<template>
  <div class="tp" ref="root">
    <button type="button" class="tp-control" :class="{ open, empty: !valid }" :disabled="disabled" @click="toggle">
      <span class="material-symbols-outlined tp-ico">{{ icon }}</span>
      <span class="tp-value">{{ valid ? modelValue : placeholder }}</span>
      <button
        v-if="clearable && valid && !disabled"
        type="button" class="tp-clear" title="Очистить"
        @click.stop="clear"
      >
        <span class="material-symbols-outlined">close</span>
      </button>
      <span v-else class="material-symbols-outlined tp-chevron">expand_more</span>
    </button>

    <Teleport to="body">
      <transition name="tp-pop">
        <div v-if="open" ref="popEl" class="tp-pop" :style="popStyle">
          <div class="tp-col" ref="hoursCol">
            <button
              v-for="h in hours" :key="'h' + h" type="button"
              class="tp-opt" :class="{ active: h === cur.h }"
              @click="pickHour(h)"
            >{{ pad(h) }}</button>
          </div>
          <div class="tp-colon">:</div>
          <div class="tp-col" ref="minutesCol">
            <button
              v-for="m in minutes" :key="'m' + m" type="button"
              class="tp-opt" :class="{ active: m === cur.m }"
              @click="pickMinute(m)"
            >{{ pad(m) }}</button>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<style scoped>
.tp { position: relative; }

.tp-control {
  display: inline-flex; align-items: center; gap: 8px; width: 100%; min-height: 42px;
  padding: 8px 10px 8px 12px;
  border: 1px solid var(--acrylic-border); border-radius: var(--radius-md, 14px);
  background: var(--color-surface-high);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit; font-weight: 600; cursor: pointer; text-align: left;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.tp-control:hover:not(:disabled) { border-color: var(--color-primary); }
.tp-control.open { border-color: var(--color-primary); box-shadow: 0 0 0 2px color-mix(in oklch, var(--color-primary) 22%, transparent); }
.tp-control:disabled { opacity: 0.55; cursor: not-allowed; }
.tp-control.empty .tp-value { color: var(--color-text-dim); font-weight: 500; }

.tp-ico { font-size: 20px; color: var(--color-text-dim); flex-shrink: 0; }
.tp-value { flex: 1; min-width: 0; font-variant-numeric: tabular-nums; overflow: hidden; text-overflow: ellipsis; }
.tp-chevron { font-size: 20px; color: var(--color-text-dim); flex-shrink: 0; }
.tp-clear {
  flex-shrink: 0; width: 24px; height: 24px; display: grid; place-items: center;
  border: none; background: none; cursor: pointer; color: var(--color-text-dim); border-radius: var(--radius-full);
}
.tp-clear:hover { background: var(--color-surface); color: var(--color-error); }
.tp-clear .material-symbols-outlined { font-size: 18px; }

.tp-pop {
  position: fixed; z-index: 11000;
  display: flex; align-items: stretch; gap: 2px; padding: 6px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 16px); box-shadow: var(--shadow-lg);
}
.tp-col { display: flex; flex-direction: column; gap: 2px; max-height: 220px; overflow-y: auto; padding: 0 2px; scrollbar-width: thin; }
.tp-colon { display: grid; place-items: center; font-weight: 800; color: var(--color-text-dim); padding: 0 2px; }
.tp-opt {
  min-width: 52px; padding: 8px 12px; border: none; border-radius: var(--radius-md, 12px);
  background: none; color: var(--color-text); font: inherit; font-weight: 600;
  font-variant-numeric: tabular-nums; cursor: pointer; text-align: center;
}
.tp-opt:hover { background: var(--color-surface-high); }
.tp-opt.active { background: var(--color-primary); color: var(--color-on-primary); }

.tp-pop-enter-active, .tp-pop-leave-active { transition: opacity 0.16s, transform 0.16s; transform-origin: top center; }
.tp-pop-enter-from, .tp-pop-leave-to { opacity: 0; transform: scale(0.96) translateY(-4px); }
</style>
