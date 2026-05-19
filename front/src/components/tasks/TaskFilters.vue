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
      <select
        class="dept-select"
        :value="tasksStore.filters.dept_id"
        @change="tasksStore.setFilter('dept_id', $event.target.value || null)"
      >
        <option value="">Все отделы</option>
        <option
          v-for="dept in departments"
          :key="dept.id"
          :value="dept.id"
        >
          {{ dept.name }}
        </option>
      </select>
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

      <template v-if="tasksStore.filters.period_preset === 'custom'">
        <div class="date-range">
          <label class="date-label">С</label>
          <input
            type="date"
            class="date-input"
            :value="tasksStore.filters.received_from"
            @change="tasksStore.setFilter('received_from', $event.target.value || null)"
          />
          <label class="date-label">По</label>
          <input
            type="date"
            class="date-input"
            :value="tasksStore.filters.received_to"
            @change="tasksStore.setFilter('received_to', $event.target.value || null)"
          />
        </div>
      </template>
    </section>

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

function selectPeriod(value) {
  tasksStore.filters.period_preset = value

  if (value === null) {
    tasksStore.setFilter('received_from', null)
    tasksStore.setFilter('received_to', null)
    return
  }

  if (value === 'custom') {
    return
  }

  const now = new Date()
  const toStr = (d) => d.toISOString().split('T')[0]

  let from = null
  const to = toStr(now)

  if (value === 'today') {
    from = toStr(now)
  } else if (value === 'week') {
    const d = new Date(now)
    d.setDate(d.getDate() - 7)
    from = toStr(d)
  } else if (value === 'month') {
    const d = new Date(now)
    d.setMonth(d.getMonth() - 1)
    from = toStr(d)
  }

  tasksStore.setFilter('received_from', from)
  tasksStore.setFilter('received_to', to)
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

.dept-select {
  width: 100%;
  padding: 7px 10px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  background: var(--gw-bg);
  color: var(--gw-text);
  font-size: 13px;
  cursor: pointer;
  outline: none;
  transition: border-color 0.15s;
}

.dept-select:focus {
  border-color: var(--gw-primary);
}

.date-range {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: 4px;
}

.date-label {
  font-size: 11px;
  color: var(--gw-text-secondary);
  font-weight: 600;
}

.date-input {
  width: 100%;
  padding: 6px 8px;
  border: 1px solid var(--gw-border);
  border-radius: 8px;
  background: var(--gw-bg);
  color: var(--gw-text);
  font-size: 13px;
  outline: none;
  transition: border-color 0.15s;
  box-sizing: border-box;
}

.date-input:focus {
  border-color: var(--gw-primary);
}

.filter-count {
  margin-top: auto;
  padding-top: 16px;
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
