<template>
  <div
    ref="root"
    class="app-tabs"
    :class="[`v-${variant}`, `align-${align}`, { full: fullWidth, dense }]"
    role="tablist"
  >
    <!-- Выбранная клавиша — отдельный элемент под вкладками: при смене выбора
         она скользит к новому пункту и подстраивается под его ширину, а не
         перескакивает. Пока не измерена — не показывается. -->
    <span
      v-if="ind"
      class="app-tabs-ind"
      :class="{ moving }"
      :style="{ width: `${ind.w}px`, height: `${ind.h}px`, transform: `translate3d(${ind.x}px, ${ind.y}px, 0)` }"
      aria-hidden="true"
    />
    <button
      v-for="t in items"
      :key="t.value"
      :ref="(el) => setTabRef(t.value, el)"
      class="app-tab"
      :class="{ active: t.value === modelValue }"
      type="button"
      role="tab"
      :aria-selected="t.value === modelValue"
      :disabled="t.disabled"
      @click="select(t)"
    >
      <span v-if="t.icon" class="material-symbols-outlined">{{ t.icon }}</span>
      <span v-if="t.label" class="app-tab-label">{{ t.label }}</span>
      <span v-if="t.badge" class="app-tab-badge">{{ t.badge }}</span>
    </button>
  </div>
</template>

<script setup>
/* Единственный переключатель вкладок платформы: свёл `PillTabs` и
   `SegmentedTabs`, которые различались только оформлением дорожки.

   variant: solid — главный переключатель режима раздела (активная вкладка на
   фирменном градиенте); tint — второстепенный, внутри карточки или тулбара
   (активная вкладка тинтованной пилюлей). */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  modelValue: { type: [String, Number], default: '' },
  /** [{ value, label, icon?, badge?, disabled? }] — `key` принимается как синоним value. */
  tabs: { type: Array, required: true },
  variant: { type: String, default: 'solid', validator: (v) => ['solid', 'tint'].includes(v) },
  align: { type: String, default: 'start', validator: (v) => ['start', 'center'].includes(v) },
  fullWidth: { type: Boolean, default: false },
  dense: { type: Boolean, default: false },
})

const emit = defineEmits(['update:modelValue', 'change'])

// `key` — наследие PillTabs: вкладки половины разделов описаны через него.
const items = computed(() => props.tabs.map((t) => ({ ...t, value: t.value ?? t.key })))

/* ── Скользящая клавиша ── */
const root = ref(null)
const tabEls = new Map()
const ind = ref(null)
// Первое измерение ставит клавишу на место без анимации — иначе она
// «въезжала» бы из угла при каждом показе вкладок.
const moving = ref(false)

function setTabRef(value, el) {
  if (el) tabEls.set(value, el)
  else tabEls.delete(value)
}

function measure() {
  const el = tabEls.get(props.modelValue)
  if (!el) { ind.value = null; return }
  ind.value = { x: el.offsetLeft, y: el.offsetTop, w: el.offsetWidth, h: el.offsetHeight }
}

let ro = null
onMounted(() => {
  measure()
  requestAnimationFrame(() => { moving.value = true })
  if (typeof ResizeObserver === 'function') {
    ro = new ResizeObserver(() => measure())
    ro.observe(root.value)
    tabEls.forEach((el) => ro.observe(el))
  }
})
onBeforeUnmount(() => ro?.disconnect())

watch(() => [props.modelValue, items.value.length], async () => {
  await nextTick()
  measure()
  if (ro) tabEls.forEach((el) => ro.observe(el))
})

function select(t) {
  if (t.disabled || t.value === props.modelValue) return
  emit('update:modelValue', t.value)
  emit('change', t.value)
}
</script>

<style scoped>
.app-tabs {
  position: relative;
  display: inline-flex;
  gap: 3px;
  align-self: flex-start;
  max-width: 100%;
  padding: 4px;
  /* Дорожка — углубление, активная вкладка — приподнятая клавиша в нём. */
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-full);
  background: var(--sk-well-bg);
  box-shadow: var(--sk-well-shadow);
  /* Вкладок может быть больше, чем влезает в ряд, — горизонтальный скролл
     вместо переноса или обрезки. */
  overflow-x: auto;
  scrollbar-width: none;
  -ms-overflow-style: none;
}

.app-tabs::-webkit-scrollbar { display: none; }

.app-tabs.full { display: flex; align-self: stretch; }
.app-tabs.align-center { justify-content: center; }
.app-tabs.full .app-tab { flex: 1; }

.app-tab {
  position: relative;
  z-index: 1;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 36px;
  padding: 8px 18px;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: color 0.2s ease;
}

.app-tabs.dense .app-tab { min-height: 32px; padding: 6px 12px; font-size: 13px; }

.app-tab .material-symbols-outlined {
  font-size: 18px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 20;
}

.app-tab:hover:not(.active):not(:disabled) { color: var(--color-text); }
.app-tab:disabled { opacity: 0.45; cursor: default; }

/* Сама клавиша. Переезд — transform, ширина подстраивается следом; кривая
   с лёгким перелётом даёт ощущение физической детали, а не слайда. */
.app-tabs-ind {
  position: absolute;
  top: 0;
  left: 0;
  border-radius: var(--radius-full);
  pointer-events: none;
}

.app-tabs-ind.moving {
  transition: transform 0.32s cubic-bezier(0.34, 1.3, 0.64, 1),
    width 0.32s cubic-bezier(0.34, 1.3, 0.64, 1);
}

.v-solid .app-tab.active { color: var(--color-on-primary); }
.v-solid .app-tabs-ind { background: var(--grad-primary); }
@supports (color: color-mix(in oklch, red, blue)) {
  .v-solid .app-tabs-ind {
    background:
      var(--grain),
      linear-gradient(180deg,
        color-mix(in oklch, var(--color-primary) 82%, white),
        var(--color-primary) 55%,
        color-mix(in oklch, var(--color-primary) 88%, black));
    box-shadow:
      inset 0 1px 0 color-mix(in oklch, white 40%, transparent),
      inset 0 -1px 0 color-mix(in oklch, black 18%, transparent),
      0 1px 3px color-mix(in oklch, black 22%, transparent);
  }
}

.v-tint .app-tab.active { color: var(--color-primary); }
.v-tint .app-tabs-ind {
  background: var(--sk-raised-bg);
  box-shadow: var(--sk-raised-shadow);
}

@media (prefers-reduced-motion: reduce) {
  .app-tabs-ind.moving { transition: none; }
}

.app-tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: var(--color-on-error);
  font-size: 11px;
  font-weight: 700;
}

.v-solid .app-tab.active .app-tab-badge { background: var(--color-surface); color: var(--color-primary); }
.v-tint .app-tab.active .app-tab-badge { background: var(--color-primary); color: var(--color-on-primary); }

/* Подписи на телефоне НЕ прячем (иконки без текста читаются плохо) — ужимаем
   типографику и отступы. */
@media (max-width: 768px) {
  .app-tab { min-height: 44px; padding: 10px 12px; font-size: 13px; }
}

@media (max-width: 480px) {
  .app-tab { padding: 10px 8px; gap: 5px; font-size: 12px; }
}
</style>
