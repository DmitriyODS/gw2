<template>
  <!-- Мобильный overlay backdrop -->
  <div
    v-if="isMobileVisible"
    class="filters-backdrop"
    @click="$emit('close')"
  />

  <aside class="task-filters" :class="{ 'mobile-sheet': true, 'mobile-sheet--open': isMobileVisible }">
    <!-- Сортировки -->
    <section class="filter-section">
      <h4 class="filter-title">Сортировки</h4>
      <button
        v-for="s in sorts"
        :key="s.value"
        class="filter-btn"
        :class="{ active: tasksStore.filters.sort === s.value }"
        @click="tasksStore.setFilter('sort', s.value)"
      >
        {{ s.label }}
      </button>
    </section>

    <!-- Фильтры по юнитам -->
    <section class="filter-section">
      <h4 class="filter-title">Фильтры</h4>
      <button
        v-for="f in unitFilters"
        :key="String(f.value)"
        class="filter-btn"
        :class="{ active: tasksStore.filters.has_units === f.value }"
        @click="tasksStore.setFilter('has_units', f.value)"
      >
        {{ f.label }}
      </button>
    </section>

    <!-- От отдела -->
    <section class="filter-section">
      <h4 class="filter-title">От отдела</h4>
      <Select
        :model-value="tasksStore.filters.dept_id"
        :options="deptOptions"
        option-label="name"
        option-value="id"
        placeholder="Все отделы"
        class="dept-select w-full"
        :filter="departments.length > 5"
        filter-placeholder="Поиск по названию..."
        show-clear
        scroll-height="280px"
        empty-message="Отделы не загружены"
        empty-filter-message="Ничего не найдено"
        @update:model-value="onDeptChange"
      />
    </section>

    <!-- Период поступления -->
    <section class="filter-section">
      <h4 class="filter-title">Период поступления</h4>
      <button
        v-for="p in periods"
        :key="String(p.value)"
        class="filter-btn"
        :class="{ active: tasksStore.filters.period_preset === p.value }"
        @click="selectPeriod(p.value)"
      >
        {{ p.label }}
      </button>

      <div
        v-if="tasksStore.filters.period_preset === 'custom' && (tasksStore.filters.received_from || tasksStore.filters.received_to)"
        class="custom-range-label"
      >
        <span class="material-symbols-outlined">date_range</span>
        {{ formatCustomLabel }}
      </div>
    </section>

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
          <button
            type="button"
            class="btn-secondary"
            @click="closeCustomDialog"
          >Отмена</button>
          <button
            type="button"
            class="btn-primary"
            :disabled="!customRangeValid"
            @click="applyCustomRange"
          >Применить</button>
        </div>
      </template>
    </Dialog>

    <!-- Кнопка сброса -->
    <button
      class="reset-btn"
      :disabled="!hasActiveFilters"
      @click="tasksStore.resetFilters()"
      title="Сбросить сортировку и фильтры"
    >
      <span class="material-symbols-outlined">restart_alt</span>
      Сбросить
    </button>

    <!-- Счётчик -->
    <div class="filter-count">
      Кол-во: <strong>{{ tasksStore.total }}</strong>
    </div>

    <!-- Кнопка закрытия на мобильном -->
    <button class="filters-close-btn" @click="$emit('close')">
      <span class="material-symbols-outlined">close</span>
      Закрыть
    </button>
  </aside>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import Select from 'primevue/select'
import Dialog from 'primevue/dialog'
import DatePicker from 'primevue/datepicker'
import { useTasksStore } from '@/stores/tasks.js'
import { getDepartments } from '@/api/departments.js'

const props = defineProps({
  mobileVisible: {
    type: Boolean,
    default: false
  }
})

defineEmits(['close'])

const tasksStore = useTasksStore()

const isMobileVisible = computed(() => props.mobileVisible)

const departments = ref([])
const deptOptions = computed(() => departments.value)

const hasActiveFilters = computed(() => {
  const f = tasksStore.filters
  return f.sort !== 'last_activity'
    || f.dept_id != null
    || f.has_units != null
    || f.period_preset != null
    || f.received_from
    || f.received_to
})

function onDeptChange(value) {
  tasksStore.setFilter('dept_id', value ?? null)
}

const sorts = [
  { label: 'Последняя активность', value: 'last_activity' },
  { label: 'Дата создания', value: 'created_at' },
  { label: 'Дата поступления', value: 'received_at' },
  { label: 'Срок исполнения', value: 'deadline' },
]

const unitFilters = [
  { label: 'Без фильтров', value: null },
  { label: 'Не приступали', value: 'none' },
  { label: 'Уже работал', value: 'mine' },
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
    // Открываем модалку выбора диапазона — фильтр применится после «Применить».
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
  // Если диапазон не задан и пресет ещё не равен custom — ничего не меняем,
  // пользователь просто закрыл модалку без выбора.
}
</script>

<style scoped>
.task-filters {
  width: 220px;
  min-width: 220px;
  background: var(--gw-surface);
  border-right: 1px solid var(--gw-border);
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 24px;
  height: 100%;
  overflow-y: auto;
}

.filter-section {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.filter-title {
  font-size: 12px;
  font-weight: 700;
  color: var(--gw-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin: 0 0 4px 0;
}

.filter-btn {
  background: transparent;
  border: 1px solid transparent;
  border-radius: 8px;
  padding: 7px 10px;
  font-size: 13px;
  color: var(--gw-text);
  cursor: pointer;
  text-align: left;
  transition: background 0.12s, color 0.12s;
}

.filter-btn:hover {
  background: var(--gw-primary-light);
  color: var(--gw-primary);
}

.filter-btn.active {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  font-weight: 600;
}

/* PrimeVue Select подгоняем под визуал боковой панели фильтров */
:deep(.dept-select.p-select) {
  width: 100%;
  font-size: 13px;
  border-radius: 8px;
}

:deep(.dept-select .p-select-label) {
  padding: 7px 10px;
}

.custom-range-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 6px;
  padding: 6px 10px;
  border-radius: 8px;
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

.btn-secondary,
.btn-primary {
  border-radius: 999px;
  padding: 8px 18px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.12s, opacity 0.12s;
  border: none;
}

.btn-secondary {
  background: transparent;
  border: 1px solid var(--gw-border);
  color: var(--gw-text);
}

.btn-secondary:hover {
  background: var(--gw-bg);
}

.btn-primary {
  background: var(--gw-primary);
  color: var(--color-on-primary);
}

.btn-primary:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-primary:not(:disabled):hover {
  background: var(--gw-primary-hover);
}

.reset-btn {
  margin-top: auto;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 8px 12px;
  border: 1px solid var(--gw-border);
  border-radius: 999px;
  background: var(--color-surface);
  color: var(--gw-text);
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

.filter-count {
  padding-top: 12px;
  border-top: 1px solid var(--gw-border);
  font-size: 14px;
  color: var(--gw-text-secondary);
}

.filter-count strong {
  color: var(--gw-primary);
}

/* ── Кнопка закрытия — скрыта на десктопе ── */
.filters-close-btn {
  display: none;
}

/* ── Мобильный backdrop ── */
.filters-backdrop {
  display: none;
}

@media (max-width: 768px) {
  /* На мобильном filters-backdrop показываем */
  .filters-backdrop {
    display: block;
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.4);
    z-index: 299;
  }

  /* Боковая панель прячется по умолчанию */
  .task-filters {
    position: fixed;
    bottom: calc(60px + env(safe-area-inset-bottom, 0px));
    left: 0;
    right: 0;
    width: 100%;
    min-width: unset;
    max-height: 78dvh;
    overflow-y: auto;
    border-right: none;
    border-top: 1px solid var(--gw-border);
    border-radius: 20px 20px 0 0;
    z-index: 300;
    transform: translateY(110%);
    visibility: hidden;
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1),
                visibility 0s linear 0.3s;
    padding-bottom: 8px;
  }

  /* Сортировки на мобильном — в отдельной шторке */
  .filter-section:first-child {
    display: none;
  }

  /* Анимация появления */
  .task-filters.mobile-sheet--open {
    transform: translateY(0);
    visibility: visible;
    transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1),
                visibility 0s linear 0s;
  }

  /* На мобильном кнопка закрытия видна */
  .filters-close-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    width: 100%;
    padding: 10px;
    margin-top: 8px;
    border: 1px solid var(--gw-border);
    border-radius: 10px;
    background: var(--gw-bg);
    color: var(--gw-text);
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
  }

  .filters-close-btn .material-symbols-outlined {
    font-size: 18px;
  }
}
</style>
