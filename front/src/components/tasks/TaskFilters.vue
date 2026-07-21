<template>
  <!-- Мобильный overlay backdrop -->
  <div
    v-if="isMobileVisible"
    class="filters-backdrop"
    @click="$emit('close')"
  />

  <aside class="task-filters rail" :class="{ 'mobile-sheet--open': isMobileVisible }">
    <div class="filters-handle" />

    <!-- Шапка только для мобильной шторки -->
    <div class="filters-head">
      <h3 class="filters-head-title">
        <span class="material-symbols-outlined">tune</span>
        Фильтры
      </h3>
      <span class="filters-count">{{ tasksStore.total }}</span>
    </div>

    <div class="rail-scroll filters-scroll">
      <!-- Вкладки (на мобильном — SegmentedTabs в шапке экрана) -->
      <section class="rail-section tabs-section">
        <button
          v-for="t in tabs"
          :key="t.value"
          class="rail-item"
          :class="{ active: tasksStore.filters.tab === t.value }"
          :data-tutorial="t.tutorial"
          @click="tasksStore.setTab(t.value)"
        >
          <span class="material-symbols-outlined">{{ t.icon }}</span>
          {{ t.label }}
          <span v-if="tasksStore.filters.tab === t.value" class="rail-badge">
            {{ tasksStore.total }}
          </span>
        </button>
      </section>

      <!-- Сортировки (на мобильном — в отдельной шторке SortSheet) -->
      <section class="rail-section sort-section">
        <h4 class="rail-label">Сортировка</h4>
        <button
          v-for="s in sorts"
          :key="s.value"
          class="rail-item"
          :class="{ active: tasksStore.filters.sort === s.value }"
          @click="tasksStore.setFilter('sort', s.value)"
        >
          <span class="material-symbols-outlined">{{ s.icon }}</span>
          {{ s.label }}
        </button>
      </section>

      <!-- Фильтры по юнитам -->
      <section class="rail-section">
        <h4 class="rail-label">Участие</h4>
        <button
          v-for="f in unitFilters"
          :key="String(f.value)"
          class="rail-item"
          :class="{ active: tasksStore.filters.has_units === f.value }"
          @click="tasksStore.setFilter('has_units', f.value)"
        >
          <span class="material-symbols-outlined">{{ f.icon }}</span>
          {{ f.label }}
        </button>
      </section>

      <!-- Автор -->
      <section class="rail-section">
        <h4 class="rail-label">Авторство</h4>
        <button
          class="rail-item"
          :class="{ active: tasksStore.filters.created_by_me }"
          @click="tasksStore.setFilter('created_by_me', !tasksStore.filters.created_by_me)"
        >
          <span class="material-symbols-outlined">edit_note</span>
          Создано мной
        </button>
      </section>

      <!-- От отдела -->
      <section class="rail-section">
        <h4 class="rail-label">Заказчик</h4>
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
      <section v-if="usesStages && stages.length" class="rail-section">
        <h4 class="rail-label">Этап</h4>
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

      <!-- Теги (мультивыбор: «хотя бы один из отмеченных») -->
      <section v-if="tags.length || canManageTags" class="rail-section">
        <h4 class="rail-label tags-label">
          Теги
          <button
            v-if="canManageTags"
            class="tags-manage-btn"
            type="button"
            title="Управлять тегами"
            aria-label="Управлять тегами"
            @click="tagsManageOpen = true"
          >
            <span class="material-symbols-outlined">settings</span>
          </button>
        </h4>
        <div class="chip-group">
          <button
            v-for="t in tags"
            :key="t.id"
            class="chip stage-chip-filter"
            :class="{ active: tasksStore.filters.tag_ids?.includes(t.id) }"
            :style="stageChipStyle(t)"
            @click="tasksStore.toggleTagFilter(t.id)"
          >
            <span class="stage-chip-dot" :style="{ background: `var(--tag-${t.color}-accent)` }" />
            {{ t.name }}
          </button>
          <p v-if="!tags.length" class="tags-empty">Тегов пока нет — создайте первый</p>
        </div>
      </section>

      <!-- Мой цвет карточки (личный, мультивыбор) -->
      <section class="rail-section">
        <h4 class="rail-label">Мой цвет</h4>
        <div class="color-filter-group">
          <button
            v-for="c in TASK_COLORS"
            :key="c.id"
            class="color-filter-swatch"
            :class="{ active: tasksStore.filters.colors?.includes(c.id) }"
            :style="{ background: `var(--tag-${c.id}-surface)`, borderColor: `var(--tag-${c.id}-border)` }"
            :title="c.label"
            :aria-label="c.label"
            type="button"
            @click="tasksStore.toggleColorFilter(c.id)"
          >
            <span
              v-if="tasksStore.filters.colors?.includes(c.id)"
              class="material-symbols-outlined color-filter-check"
              :style="{ color: `var(--tag-${c.id}-accent)` }"
            >check</span>
          </button>
        </div>
      </section>

      <!-- Период поступления -->
      <section class="rail-section">
        <h4 class="rail-label">Период поступления</h4>
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

    <TagManageDialog v-model="tagsManageOpen" :tags="tags" @changed="reloadTags" />

    <div class="filters-foot">
      <button
        class="rail-reset"
        :disabled="!hasActiveFilters"
        @click="tasksStore.resetFilters()"
        title="Сбросить сортировку и фильтры"
        aria-label="Сбросить сортировку и фильтры"
      >
        <span class="material-symbols-outlined">restart_alt</span>
        <span class="reset-btn-label">Сбросить всё</span>
      </button>
      <button class="filters-close-btn" @click="$emit('close')">
        Показать результаты
      </button>
    </div>
  </aside>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import Select from 'primevue/select'
import Dialog from 'primevue/dialog'
import DatePicker from 'primevue/datepicker'
import { useTasksStore } from '@/stores/tasks.js'
import { useAuthStore } from '@/stores/auth.js'
import { getDepartments } from '@/api/departments.js'
import { getStages } from '@/api/stages.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import TagManageDialog from '@/components/tasks/TagManageDialog.vue'
import { TASK_COLORS } from '@/utils/taskColors.js'
import { TASK_SORTS } from '@/components/tasks/taskSorts.js'

const props = defineProps({
  mobileVisible: {
    type: Boolean,
    default: false
  }
})

defineEmits(['close'])

const tasksStore = useTasksStore()
const authStore = useAuthStore()
const { usesStages } = useCompanySettings()

const isMobileVisible = computed(() => props.mobileVisible)

const departments = ref([])
const deptOptions = computed(() => departments.value)
const stages = ref([])
const tags = computed(() => tasksStore.tags)
const tagsManageOpen = ref(false)
const { isAtLeast } = usePermission()
const canManageTags = computed(() => isAtLeast(ROLES.MANAGER))

const reloadTags = () => tasksStore.fetchTags({ force: true })

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
    || f.created_by_me
    || (f.tag_ids?.length > 0)
    || (f.colors?.length > 0)
})

function onDeptChange(value) {
  tasksStore.setFilter('dept_id', value ?? null)
}

const tabs = [
  { value: 'active', label: 'Активные', icon: 'checklist', tutorial: 'tab-active' },
  { value: 'favorites', label: 'Избранное', icon: 'star', tutorial: 'tab-favorites' },
  { value: 'archive', label: 'Архив', icon: 'inventory_2', tutorial: 'tab-archive' },
]

const sorts = TASK_SORTS

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

// Справочники панели фильтров — company-scoped; перезагружаем при монтировании
// и при живой смене активной компании, иначе в фильтрах видны отделы/этапы/теги
// покинутой компании.
async function loadCatalogs() {
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
  } else {
    stages.value = []
  }
  await tasksStore.fetchTags({ force: true })
}

onMounted(loadCatalogs)

watch(() => authStore.companyId, (id, prev) => {
  if (prev == null || id === prev) return
  loadCatalogs()
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
/* Каркас рейки — глобальные классы .rail/.rail-* (main.css).
   Здесь — только привязка к странице задач и мобильная шторка. */
.task-filters {
  margin: 22px 0 22px 24px;
  max-height: calc(100% - 44px);
}

.filters-handle,
.filters-head {
  display: none;
}

.filters-scroll {
  padding: 4px 2px;
}

/* ── Чипы (этапы, период) ── */
.chip-group {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  padding: 0 6px;
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

/* ── Теги ── */
.tags-label {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.tags-manage-btn {
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: inline-flex;
  padding: 2px;
  border-radius: var(--radius-full);
}

.tags-manage-btn:hover { color: var(--color-text); }
.tags-manage-btn .material-symbols-outlined { font-size: 16px; }

.tags-empty {
  margin: 0;
  font-size: 12px;
  color: var(--color-text-dim);
}

/* ── Фильтр по личному цвету ── */
.color-filter-group {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  padding: 0 6px;
}

.color-filter-swatch {
  width: 26px;
  height: 26px;
  border-radius: var(--radius-sm);
  border: 1px solid;
  cursor: pointer;
  padding: 0;
  display: grid;
  place-items: center;
  transition: transform 0.12s, outline-color 0.12s;
}

.color-filter-swatch.active {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
  transform: scale(1.06);
}

.color-filter-check { font-size: 15px; font-weight: 700; }

/* PrimeVue Select под визуал панели */
.dept-select {
  margin: 0 6px;
}

:deep(.dept-select.p-select) {
  font-size: 13px;
  border-radius: var(--radius-md);
}

:deep(.dept-select .p-select-label) {
  padding: 9px 12px;
}

.custom-range-label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 6px 6px 0;
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
  transition: background 0.12s, opacity 0.12s, filter 0.12s;
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
  background: var(--grad-primary);
  color: var(--color-on-primary);
}

.btn-filled:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-filled:not(:disabled):hover {
  filter: brightness(1.06);
}

/* Подвал (на десктопе — только кнопка сброса) */
.filters-foot {
  flex-shrink: 0;
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
    opacity: 0;
    transition: opacity 0.25s ease;
  }

  .task-filters.mobile-sheet--open ~ .filters-backdrop,
  .filters-backdrop {
    opacity: 1;
  }

  /* Шторка перекрывает глобальный .rail: фиксированная снизу, стекло с blur. */
  .task-filters {
    position: fixed;
    /* Прижимаем к нижней кромке экрана — нижняя навигация перекрывается
       шторкой так же, как в SortSheet. Если оставлять зазор (bottom: 60px),
       шторка «висит в воздухе» и выглядит оторванной. */
    top: auto;
    bottom: 0;
    left: 0;
    right: 0;
    width: 100%;
    min-width: unset;
    margin: 0;
    padding: 0;
    /* Максимум 85dvh, минимум — чтобы поместился sticky-header и хоть одна
       секция. */
    height: auto;
    max-height: 85dvh;
    background: var(--acrylic-bg);
    -webkit-backdrop-filter: var(--acrylic-blur);
    backdrop-filter: var(--acrylic-blur);
    border: none;
    border-top: 1px solid var(--color-outline-dim);
    border-radius: 24px 24px 0 0;
    z-index: 300;
    transform: translateY(110%);
    visibility: hidden;
    transition: transform 0.32s cubic-bezier(0.4, 0, 0.2, 1),
                visibility 0s linear 0.32s;
    box-shadow: 0 -8px 24px color-mix(in oklch, var(--color-scrim) 60%, transparent);
  }

  .task-filters.mobile-sheet--open {
    transform: translateY(0);
    visibility: visible;
    transition: transform 0.32s cubic-bezier(0.4, 0, 0.2, 1),
                visibility 0s linear 0s;
  }

  /* На мобильном — handle, sticky header и sticky footer. */
  .filters-handle {
    display: block;
    width: 36px;
    height: 4px;
    border-radius: 2px;
    background: var(--color-outline-dim);
    margin: 10px auto 2px;
    flex-shrink: 0;
  }

  .filters-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 6px 18px 12px;
    border-bottom: 1px solid var(--color-outline-dim);
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
    padding: 16px 16px 8px;
    gap: 18px;
    /* Включаем плавный momentum-скролл в шторке на iOS. */
    -webkit-overflow-scrolling: touch;
    overscroll-behavior: contain;
  }

  /* Пункты — тач-зона ≥44px. */
  .task-filters .rail-item {
    min-height: 44px;
    font-size: 14px;
  }

  /* Чипы крупнее — тач-зона ≥40px. */
  .chip {
    padding: 9px 14px;
    font-size: 13.5px;
    min-height: 40px;
  }

  .chip .material-symbols-outlined {
    font-size: 17px;
  }

  /* PrimeVue Select тоже подрастает. */
  :deep(.dept-select .p-select-label) {
    padding: 12px 12px;
    font-size: 14px;
  }

  /* Вкладки — в SegmentedTabs шапки экрана, сортировки — в шторке SortSheet. */
  .tabs-section,
  .sort-section {
    display: none;
  }

  .filters-foot {
    display: flex;
    gap: 10px;
    padding: 12px 16px calc(12px + env(safe-area-inset-bottom, 0px));
    border-top: 1px solid var(--color-outline-dim);
  }

  .filters-foot .rail-reset {
    flex: 0 0 auto;
    width: auto;
    margin-top: 0;
    padding: 11px 16px;
    border: 1px solid var(--color-outline-dim);
    border-radius: var(--radius-full);
    color: var(--color-text);
  }

  .filters-foot .rail-reset .material-symbols-outlined {
    font-size: 20px;
  }

  .filters-close-btn {
    display: block;
    flex: 1;
    padding: 12px 16px;
    border: none;
    border-radius: var(--radius-full);
    background: var(--grad-primary);
    color: var(--color-on-primary);
    font-size: 14.5px;
    font-weight: 650;
    cursor: pointer;
    min-height: 44px;
    transition: filter 0.15s;
  }

  .filters-close-btn:active {
    filter: brightness(1.06);
  }
}

@media (max-width: 380px) {
  .filters-foot .rail-reset {
    padding: 11px;
  }
  .filters-foot .reset-btn-label {
    display: none;
  }
}
</style>
