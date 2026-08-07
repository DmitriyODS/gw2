<template>
  <div class="period-control" :class="{ 'is-compact': compact }">
    <!-- Дата + стрелки сдвига — одной группой (стрелки двигают именно даты). -->
    <div class="period-main">
      <div ref="displayRef" class="period-display" @click="openPicker">
        <span class="material-symbols-outlined">calendar_month</span>
        {{ period.displayLabel.value }}
      </div>
      <div class="period-shift">
        <button class="period-btn" @click="period.shift(-1)" :disabled="!canShift" title="Назад">
          <span class="material-symbols-outlined">chevron_left</span>
        </button>
        <button class="period-btn" @click="period.shift(1)" :disabled="!canShift" title="Вперёд">
          <span class="material-symbols-outlined">chevron_right</span>
        </button>
      </div>
    </div>

    <Teleport to="body">
      <div v-if="showPicker" class="period-picker" :style="pickerStyle">
        <DatePicker
          v-model="customRange"
          selection-mode="range"
          date-format="dd.mm.yy"
          inline
          @update:model-value="onCustomRange"
        />
      </div>
    </Teleport>

    <!-- Тесная панель: набор периодов прячется под кнопку — иначе он занимает
         ещё одну-две строки шапки, а раздел про содержимое, а не про фильтры. -->
    <template v-if="compact">
      <button class="period-preset" type="button" @click="openModes">
        <span class="period-preset-label">{{ activeModeLabel }}</span>
        <span class="material-symbols-outlined">expand_more</span>
      </button>
      <ContextMenu
        :visible="modesOpen"
        :x="modesPos.x"
        :y="modesPos.y"
        :items="modeItems"
        @select="pickMode"
        @close="modesOpen = false"
      />
    </template>

    <div v-else class="period-buttons">
      <div class="period-modes">
        <button
          v-for="m in modes"
          :key="m.value"
          class="mode-btn"
          :class="{ active: period.mode.value === m.value }"
          @click="period.selectMode(m.value)"
        >
          {{ m.label }}
        </button>
      </div>

      <button
        class="all-time-btn"
        :class="{ active: period.mode.value === 'all' }"
        @click="period.setAllTime()"
        title="Показать все задачи за весь срок"
      >
        <span class="material-symbols-outlined">all_inclusive</span>
        <span class="all-time-label">Весь срок</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted } from 'vue'
import DatePicker from 'primevue/datepicker'
import ContextMenu from '@/components/common/ContextMenu.vue'
import { useStatsPeriod } from '@/composables/useStatsPeriod.js'

const props = defineProps({
  /** Тесная панель: набор периодов уходит под кнопку. */
  compact: { type: Boolean, default: false },
})

const emit = defineEmits(['change'])

const period = useStatsPeriod()
const showPicker = ref(false)
const customRange = ref(null)
const displayRef = ref(null)
const pickerPos = ref({ top: 0, left: 0 })

function openPicker() {
  const rect = displayRef.value?.getBoundingClientRect()
  if (rect) {
    const pickerWidth = 580
    const left = Math.max(8, Math.min(rect.left, window.innerWidth - pickerWidth - 8))
    pickerPos.value = { top: rect.bottom + 8, left }
  }
  showPicker.value = !showPicker.value
}

function onDocClick(e) {
  if (!showPicker.value) return
  if (displayRef.value?.contains(e.target)) return
  const pickerEl = document.querySelector('.period-picker')
  if (pickerEl && pickerEl.contains(e.target)) return
  showPicker.value = false
}

onMounted(() => document.addEventListener('mousedown', onDocClick, true))
onUnmounted(() => document.removeEventListener('mousedown', onDocClick, true))

const pickerStyle = computed(() => ({
  top: `${pickerPos.value.top}px`,
  left: `${pickerPos.value.left}px`,
}))

const modes = [
  { value: 'day', label: 'День' },
  { value: 'week', label: 'Неделя' },
  { value: 'month', label: 'Месяц' },
  { value: 'year', label: 'Год' },
]

// Сдвиг имеет смысл только для регулярных периодов (не «весь срок»/«произвольный»).
const canShift = computed(() => ['day', 'week', 'month', 'year'].includes(period.mode.value))

/* ── Набор периодов под кнопкой (тесная панель) ── */
const modesOpen = ref(false)
const modesPos = ref({ x: 0, y: 0 })

const allModes = computed(() => [...modes, { value: 'all', label: 'Весь срок' }])
const activeModeLabel = computed(() => (
  allModes.value.find((m) => m.value === period.mode.value)?.label || 'Период'
))
const modeItems = computed(() => allModes.value.map((m) => ({
  label: m.label,
  icon: m.value === period.mode.value ? 'check' : (m.value === 'all' ? 'all_inclusive' : 'calendar_month'),
  action: m.value,
})))

function openModes(e) {
  const r = e.currentTarget.getBoundingClientRect()
  modesPos.value = { x: r.left, y: r.bottom + 6 }
  modesOpen.value = true
}

function pickMode(value) {
  modesOpen.value = false
  if (value === 'all') period.setAllTime()
  else period.selectMode(value)
}

function onCustomRange(val) {
  if (Array.isArray(val) && val[0] && val[1]) {
    period.setCustom(val[0], val[1])
    showPicker.value = false
  }
}

watch(
  [period.fromStr, period.toStr],
  ([from, to]) => {
    emit('change', { from, to })
  },
  { immediate: true }
)
</script>

<style scoped>
.period-control {
  display: flex;
  align-items: center;
  gap: 16px;
  flex-wrap: wrap;
  padding: 4px 0;
  position: relative;
}

.period-main {
  display: flex;
  align-items: center;
  gap: 8px;
}

/* Компактно — всё в одну строку: дата сжимается, кнопки держат размер. */
.period-control.is-compact { gap: 8px; padding: 0; flex-wrap: nowrap; }
.period-control.is-compact .period-main { flex: 1 1 auto; min-width: 0; }
.period-control.is-compact .period-display {
  flex: 1 1 auto;
  min-width: 0;
  overflow: hidden;
  padding: 8px 12px;
  min-height: 40px;
  font-size: 13px;
}

.period-display {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  min-height: 44px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  cursor: pointer;
  background: var(--acrylic-card-bg);
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  transition: border-color 0.15s, background 0.15s;
  user-select: none;
  white-space: nowrap;
}

.period-display:hover {
  border-color: var(--color-primary);
  background: var(--color-surface-high);
}

.period-display .material-symbols-outlined {
  font-size: 18px;
  color: var(--color-primary);
}

/* Кнопка набора периодов (тесная панель) — та же пилюля, что и дата. */
.period-preset {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 12px;
  min-height: 40px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}

.period-preset .material-symbols-outlined { font-size: 18px; color: var(--color-text-dim); }
.period-preset-label { overflow: hidden; text-overflow: ellipsis; }

.period-picker {
  position: fixed;
  z-index: 1001;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-xl, 20px);
  box-shadow: var(--shadow-lg);
  padding: 8px;
}

.period-buttons {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}

.period-modes {
  display: inline-flex;
  background: var(--color-surface-high);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  padding: 4px;
  gap: 2px;
}

.mode-btn {
  padding: 8px 14px;
  min-height: 36px;
  background: transparent;
  border: none;
  border-radius: var(--radius-full);
  color: var(--color-text-dim);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}

.mode-btn:hover:not(.active) {
  color: var(--color-text);
}

.mode-btn.active {
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-weight: 700;
  box-shadow: var(--shadow-sm);
}

.period-shift {
  display: flex;
  align-items: center;
  gap: 4px;
}

.period-btn {
  width: 40px;
  height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
  line-height: 1;
  padding: 0;
}

.period-btn:hover:not(:disabled) {
  background: var(--color-primary);
  border-color: var(--color-primary);
  color: var(--color-on-primary);
}

.period-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.period-btn .material-symbols-outlined {
  font-size: 20px;
}

.all-time-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  min-height: 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

.all-time-btn:hover:not(.active) {
  background: var(--color-surface-high);
  border-color: var(--color-primary);
}

.all-time-btn.active {
  background: var(--color-tertiary-container);
  border-color: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  font-weight: 700;
}

.all-time-btn .material-symbols-outlined {
  font-size: 18px;
}

/* Мобайл: две компактные строки — «дата + стрелки» и «режимы + весь срок». */
@media (max-width: 768px) {
  .period-control {
    gap: 6px;
    padding: 0;
  }

  .period-main {
    width: 100%;
    gap: 6px;
  }

  .period-display {
    flex: 1;
    min-width: 0;
    min-height: 38px;
    padding: 7px 12px;
    font-size: 13px;
    justify-content: center;
  }

  .period-btn {
    width: 38px;
    height: 38px;
  }

  .period-buttons {
    gap: 6px;
    width: 100%;
  }

  .period-modes {
    flex: 1;
    justify-content: center;
    padding: 3px;
  }

  .mode-btn {
    flex: 1;
    padding: 7px 6px;
    min-height: 32px;
    font-size: 12.5px;
  }

  /* «Весь срок» — компактная иконка-«пилюля» в одной строке с режимами. */
  .all-time-btn {
    min-height: 38px;
    padding: 7px 12px;
  }
  .all-time-label { display: none; }
  .all-time-btn .material-symbols-outlined { margin: 0; }
}
</style>
