<template>
  <!-- Мобильный overlay backdrop -->
  <div
    v-if="isMobileVisible"
    class="filters-backdrop"
    @click="$emit('close')"
  />

  <aside class="task-filters" :class="{ 'mobile-sheet--open': isMobileVisible }">
    <div class="filters-handle" />

    <div class="filters-head">
      <h3 class="filters-head-title">
        <span class="material-symbols-outlined">tune</span>
        Фильтры
      </h3>
      <span class="filters-count">{{ tasksStore.total }}</span>
    </div>

    <div class="filters-scroll">
      <!-- Сортировки (на мобильном — в отдельной шторке) -->
      <section class="filter-section sort-section">
        <h4 class="filter-title">Сортировка</h4>
        <div class="chip-group">
          <button
            v-for="s in sorts"
            :key="s.value"
            class="chip"
            :class="{ active: tasksStore.filters.sort === s.value }"
            @click="tasksStore.setFilter('sort', s.value)"
          >
            <span class="material-symbols-outlined">{{ s.icon }}</span>
            {{ s.label }}
          </button>
        </div>
      </section>

      <!-- Фильтры по юнитам -->
      <section class="filter-section">
        <h4 class="filter-title">Участие</h4>
        <div class="chip-group">
          <button
            v-for="f in unitFilters"
            :key="String(f.value)"
            class="chip"
            :class="{ active: tasksStore.filters.has_units === f.value }"
            @click="tasksStore.setFilter('has_units', f.value)"
          >
            <span class="material-symbols-outlined">{{ f.icon }}</span>
            {{ f.label }}
          </button>
        </div>
      </section>

      <!-- От отдела -->
      <section class="filter-section">
        <h4 class="filter-title">Заказчик</h4>
        <Select
          :model-value="tasksStore.filters.dept_id"
          :options="deptOptions"
          option-label="name"
          option-value="id"
          placeholder="Все отделы"
          class="dept-select w-full"
          :filter="departments.length > 5"
          filter-placeholder="Поиск по названию…"
          show-clear
          scroll-height="280px"
          empty-message="Отделы не загружены"
          empty-filter-message="Ничего не найдено"
          @update:model-value="onDeptChange"
        />
      </section>

      <!-- Этапы (условно) -->
      <section v-if="usesStages && stages.length" class="filter-section">
        <h4 class="filter-title">Этап</h4>
        <div class="chip-group">
          <button
            class="chip"
            :class="{ active: tasksStore.filters.stage_id == null }"
            @click="tasksStore.setFilter('stage_id', null)"
          >
            Все
          </button>
          <button
            v-for="s in stages"
            :key="s.id"
            class="chip stage-chip-filter"
            :class="{ active: tasksStore.filters.stage_id === s.id }"
            :style="stageChipStyle(s)"
            @click="tasksStore.setFilter('stage_id', s.id)"
          >
            <span class="stage-chip-dot" :style="{ background: `var(--tag-${s.color}-accent)` }" />
            {{ s.name }}
          </button>
        </div>
      </section>

      <!-- Период поступления -->
      <section class="filter-section">
        <h4 class="filter-title">Период поступления</h4>
        <div class="chip-group">
          <button
            v-for="p in periods"
            :key="String(p.value)"
            class="chip"
            :class="{ active: tasksStore.filters.period_preset === p.value }"
            @click="selectPeriod(p.value)"
          >
            {{ p.label }}
          </button>
        </div>

        <div
          v-if="tasksStore.filters.period_preset === 'custom' && (tasksStore.filters.received_from || tasksStore.filters.received_to)"
          class="custom-range-label"
        >
          <span class="material-symbols-outlined">date_range</span>
          {{ formatCustomLabel }}
        </div>
      </section>
    </div>

    <!-- Модалка выбора произвольного периода -->
    <Dialog
      v-if="showCustomDialog"
      :visible="showCustomDialog"
      @update:visible="closeCustomDialog"
      modal
      header="Свой период"
      :style="{ width: '380px', maxWidth: '95vw' }"
    >
      <div class="custom-range-picker">
        <DatePicker
          v-model="customRange"
          selection-mode="range"
          date-format="dd.mm.yy"
          inline
          :manual-input="false"
        />
      </div>
      <template #footer>
        <div class="custom-range-footer">
          <button type="button" class="btn-outlined" @click="closeCustomDialog">Отмена</button>
          <button type="button" class="btn-filled" :disabled="!customRangeValid" @click="applyCustomRange">Применить</button>
        </div>
      </template>
    </Dialog>

    <div class="filters-foot">
      <button
        class="reset-btn"
        :disabled="!hasActiveFilters"
        @click="tasksStore.resetFilters()"
        title="Сбросить сортировку и фильтры"
      >
        <span class="material-symbols-outlined">restart_alt</span>
        Сбросить всё
      </button>
      <button class="filters-close-btn" @click="$emit('close')">
        Показать результаты
      </button>
    </div>
  </aside>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import Select from 'primevue/select'
import Dialog from 'primevue/dialog'
import DatePicker from 'primevue/datepicker'
import { useTasksStore } from '@/stores/tasks.js'
import { getDepartments } from '@/api/departments.js'
import { getStages } from '@/api/stages.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'

const props = defineProps({
  mobileVisible: {
    type: Boolean,
    default: false
  }
})

defineEmits(['close'])

const tasksStore = useTasksStore()
const { usesStages } = useCompanySettings()

const isMobileVisible = computed(() => props.mobileVisible)

const departments = ref([])
const deptOptions = computed(() => departments.value)
const stages = ref([])

function stageChipStyle(s) {
  if (!s?.color) return {}
  return {
    background: `var(--tag-${s.color}-surface)`,
    color: `var(--tag-${s.color}-accent)`,
  }
}

const hasActiveFilters = computed(() => {
  const f = tasksStore.filters
  return f.sort !== 'last_activity'
    || f.dept_id != null
    || f.stage_id != null
    || f.has_units != null
    || f.period_preset != null
    || f.received_from
    || f.received_to
})

function onDeptChange(value) {
  tasksStore.setFilter('dept_id', value ?? null)
}

const sorts = [
  { label: 'Последняя активность', value: 'last_activity', icon: 'history' },
  { label: 'Дата создания', value: 'created_at', icon: 'calendar_today' },
  { label: 'Дата поступления', value: 'received_at', icon: 'inbox' },
  { label: 'Срок исполнения', value: 'deadline', icon: 'event' },
]

const unitFilters = [
  { label: 'Все', value: null, icon: 'apps' },
  { label: 'Не приступали', value: 'none', icon: 'radio_button_unchecked' },
  { label: 'Уже работал', value: 'mine', icon: 'task_alt' },
]

const periods = [
  { label: 'Весь период', value: null },
  { label: 'Сегодня', value: 'today' },
  { label: 'Неделя', value: 'week' },
  { label: 'Месяц', value: 'month' },
  { label: 'Задать свой', value: 'custom' },
]

onMounted(async () => {
  try {
    const data = await getDepartments()
    departments.value = Array.isArray(data) ? data : (data.departments ?? data.items ?? [])
  } catch {
    departments.value = []
  }
  if (usesStages.value) {
    try {
      const data = await getStages()
      stages.value = Array.isArray(data) ? data : (data.items ?? [])
    } catch {
      stages.value = []
    }
  }
})

const showCustomDialog = ref(false)
const customRange = ref(null)

const customRangeValid = computed(() => {
  const r = customRange.value
  return Array.isArray(r) && r[0] instanceof Date && r[1] instanceof Date
})

const formatCustomLabel = computed(() => {
  const from = tasksStore.filters.received_from
  const to = tasksStore.filters.received_to
  if (!from && !to) return ''
  const fmt = (s) => {
    if (!s) return '—'
    const d = new Date(s)
    return d.toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
  }
  return `${fmt(from)} — ${fmt(to)}`
})

function dateToStr(d) {
  if (!(d instanceof Date)) return null
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function selectPeriod(value) {
  if (value === 'custom') {
    const from = tasksStore.filters.received_from
    const to = tasksStore.filters.received_to
    customRange.value = from && to ? [new Date(from), new Date(to)] : null
    showCustomDialog.value = true
    return
  }

  tasksStore.filters.period_preset = value

  if (value === null) {
    tasksStore.setFilter('received_from', null)
    tasksStore.setFilter('received_to', null)
    return
  }

  const now = new Date()
  let from = null
  const to = dateToStr(now)

  if (value === 'today') {
    from = dateToStr(now)
  } else if (value === 'week') {
    const d = new Date(now)
    d.setDate(d.getDate() - 7)
    from = dateToStr(d)
  } else if (value === 'month') {
    const d = new Date(now)
    d.setMonth(d.getMonth() - 1)
    from = dateToStr(d)
  }

  tasksStore.setFilter('received_from', from)
  tasksStore.setFilter('received_to', to)
}

function applyCustomRange() {
  if (!customRangeValid.value) return
  const [from, to] = customRange.value
  tasksStore.filters.period_preset = 'custom'
  tasksStore.setFilter('received_from', dateToStr(from))
  tasksStore.setFilter('received_to', dateToStr(to))
  showCustomDialog.value = false
}

function closeCustomDialog() {
  showCustomDialog.value = false
}
</script>

<style scoped>
.task-filters {
  width: 256px;
  min-width: 256px;
  background: var(--color-surface);
  border-right: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.filters-handle {
  display: none;
}

.filters-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 18px 18px 12px;
  flex-shrink: 0;
}

.filters-head-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
}

.filters-head-title .material-symbols-outlined {
  font-size: 20px;
  color: var(--color-primary);
}

.filters-count {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  height: 24px;
  padding: 0 8px;
  border-radius: var(--radius-full);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 13px;
  font-weight: 700;
}

.filters-scroll {
  flex: 1;
  overflow-y: auto;
  padding: 0 18px 12px;
  display: flex;
  flex-direction: column;
  gap: 22px;
}

.filter-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.filter-title {
  font-size: 11px;
  font-weight: 700;
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.6px;
  margin: 0;
}

.chip-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  background: var(--color-surface-high);
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  padding: 7px 13px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.14s, color 0.14s, border-color 0.14s;
}

.chip .material-symbols-outlined {
  font-size: 16px;
}

.chip:hover {
  background: var(--color-surface-highest);
}

.chip.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-color: color-mix(in oklch, var(--color-primary) 35%, transparent);
}

.stage-chip-filter.active {
  outline: 2px solid currentColor;
  outline-offset: -2px;
}

.stage-chip-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}

/* PrimeVue Select под визуал панели */
:deep(.dept-select.p-select) {
  width: 100%;
  font-size: 13px;
  border-radius: var(--radius-md);
  background: var(--color-surface-high);
  border-color: transparent;
}

:deep(.dept-select .p-select-label) {
  padding: 9px 12px;
}

.custom-range-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  border-radius: var(--radius-md);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 12px;
  font-weight: 600;
}

.custom-range-label .material-symbols-outlined {
  font-size: 16px;
}

.custom-range-picker {
  display: flex;
  justify-content: center;
  padding: 4px 0;
}

.custom-range-footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.btn-outlined,
.btn-filled {
  border-radius: var(--radius-full);
  padding: 9px 18px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, opacity 0.12s;
  border: none;
}

.btn-outlined {
  background: transparent;
  border: 1px solid var(--color-outline-dim);
  color: var(--color-text);
}

.btn-outlined:hover {
  background: var(--color-surface-high);
}

.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.btn-filled:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-filled:not(:disabled):hover {
  background: var(--color-primary-hover);
}

/* Подвал */
.filters-foot {
  flex-shrink: 0;
  padding: 12px 18px;
  border-top: 1px solid var(--color-outline-dim);
}

.reset-btn {
  width: 100%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}

.reset-btn:hover:not(:disabled) {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-color: color-mix(in oklch, var(--color-error) 40%, var(--color-outline-dim));
}

.reset-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.reset-btn .material-symbols-outlined {
  font-size: 18px;
}

.filters-close-btn {
  display: none;
}

/* ── Мобильный backdrop ── */
.filters-backdrop {
  display: none;
}

@media (max-width: 768px) {
  .filters-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    background: var(--color-scrim);
    z-index: 299;
  }

  .task-filters {
    position: fixed;
    /* Прижимаем к нижней кромке экрана — нижняя навигация перекрывается
       шторкой так же, как в SortSheet. Если оставлять зазор (bottom: 60px),
       шторка «висит в воздухе» и выглядит оторванной. */
    bottom: 0;
    left: 0;
    right: 0;
    width: 100%;
    min-width: unset;
    /* Переопределяем десктопный height: 100% — иначе шторка тянется на весь
       экран даже если фильтров всего пара. Высота — по содержимому, ограничена
       максимумом, чтобы при большом количестве фильтров можно было скроллить. */
    height: auto;
    max-height: 80dvh;
    /* safe-area-inset-bottom — это нижний «вырез» (iPhone home indicator).
       Добавляем как padding, а не offset — шторка по-прежнему касается низа. */
    padding-bottom: calc(16px + env(safe-area-inset-bottom, 0px));
    border-right: none;
    border-top: 1px solid var(--color-outline-dim);
    border-radius: var(--radius-xl) var(--radius-xl) 0 0;
    z-index: 300;
    transform: translateY(110%);
    visibility: hidden;
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1),
                visibility 0s linear 0.3s;
  }

  .filters-handle {
    display: block;
    width: 36px;
    height: 4px;
    border-radius: 2px;
    background: var(--color-outline-dim);
    margin: 10px auto 2px;
    flex-shrink: 0;
  }

  /* Сортировки на мобильном — в отдельной шторке SortSheet */
  .sort-section {
    display: none;
  }

  .task-filters.mobile-sheet--open {
    transform: translateY(0);
    visibility: visible;
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1),
                visibility 0s linear 0s;
  }

  .filters-close-btn {
    display: block;
    width: 100%;
    margin-top: 8px;
    padding: 12px;
    border: none;
    border-radius: var(--radius-full);
    background: var(--color-primary);
    color: var(--color-on-primary);
    font-size: 14px;
    font-weight: 650;
    cursor: pointer;
  }
}
</style>
