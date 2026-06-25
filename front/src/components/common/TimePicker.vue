<script setup>
/* Кастомный выбор времени (24ч) в стиле приложения — замена браузерному
   <input type="time">. v-model — строка 'HH:MM' (или null/'' когда не задано).
   Две прокручиваемые колонки часов/минут, клик-снаружи закрывает. */
import { ref, computed, onMounted, onBeforeUnmount, nextTick, watch } from 'vue'

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
const hoursCol = ref(null)
const minutesCol = ref(null)
const open = ref(false)

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
  if (open.value) {
    await nextTick()
    scrollToSelected()
  }
}

function scrollToSelected() {
  for (const col of [hoursCol.value, minutesCol.value]) {
    const el = col?.querySelector('.tp-opt.active')
    if (el) el.scrollIntoView({ block: 'center' })
  }
}

function onClickOutside(e) {
  if (root.value && !root.value.contains(e.target)) open.value = false
}

watch(() => props.modelValue, () => { if (open.value) nextTick(scrollToSelected) })

onMounted(() => document.addEventListener('mousedown', onClickOutside))
onBeforeUnmount(() => document.removeEventListener('mousedown', onClickOutside))
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

    <transition name="tp-pop">
      <div v-if="open" class="tp-pop">
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
  </div>
</template>

<style scoped>
.tp { position: relative; }

.tp-control {
  display: inline-flex; align-items: center; gap: 8px; width: 100%; min-height: 42px;
  padding: 8px 10px 8px 12px;
  border: 1px solid var(--color-outline-variant); border-radius: var(--radius-md, 14px);
  background: var(--color-surface-high); color: var(--color-text);
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
  position: absolute; z-index: 70; top: calc(100% + 6px); left: 0;
  display: flex; align-items: stretch; gap: 2px; padding: 6px;
  background: var(--color-surface); border: 1px solid var(--color-outline-variant);
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
