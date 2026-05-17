<template>
  <div class="stats-view">
    <header class="stats-header">
      <h1>Статистика</h1>
      <div class="stats-mode-toggle">
        <button data-tutorial="stats-tab-common" :class="{ active: mode === 'common' }" @click="switchMode('common')">Общая</button>
        <button data-tutorial="stats-tab-extended" :class="{ active: mode === 'extended' }" @click="switchMode('extended')">Расширенная</button>
      </div>
    </header>

    <StatsPeriodControl @change="onPeriodChange" />

    <div v-if="loading" class="loading-state">
      <ProgressSpinner />
    </div>

    <!-- Общая статистика -->
    <template v-else-if="mode === 'common' && commonData">
      <div class="stats-grid">
        <StatsWidget title="Задачи за период" :export-fn="canExport ? handleExportCommon : null">
          <div class="task-numbers">
            <div class="task-num debt">
              <span class="num">{{ commonData.tasks?.debt ?? 0 }}</span>
              <span class="label">Долг</span>
            </div>
            <div class="task-num positive">
              <span class="num">+{{ commonData.tasks?.received ?? 0 }}</span>
              <span class="label">Поступило</span>
            </div>
            <div class="task-num negative">
              <span class="num">-{{ commonData.tasks?.closed ?? 0 }}</span>
              <span class="label">Закрыто</span>
            </div>
            <div class="task-num remaining">
              <span class="num">{{ commonData.tasks?.remaining ?? 0 }}</span>
              <span class="label">Осталось</span>
            </div>
          </div>
        </StatsWidget>

        <StatsWidget
          title="Отработка задач по сотрудникам"
          :export-fn="canExportUsers ? handleExportCommon : null"
        >
          <div class="table-scroll">
            <DataTable :value="commonData.tasks_by_employees || []" size="small" :show-gridlines="false">
              <Column field="fio" header="Сотрудник" />
              <Column field="tasks_count" header="Задачи" style="width:100px" />
              <Column header="Время" style="width:100px">
                <template #body="{ data }">
                  {{ roundHours(data.total_hours) }}
                </template>
              </Column>
            </DataTable>
          </div>
        </StatsWidget>

        <StatsWidget title="Задачи по часам">
          <div class="table-scroll">
            <DataTable :value="commonData.tasks_by_hours || []" size="small" :show-gridlines="false">
              <Column field="name" header="Задача" />
              <Column header="Время" style="width:100px">
                <template #body="{ data }">
                  {{ roundHours(data.total_hours) }}
                </template>
              </Column>
            </DataTable>
          </div>
        </StatsWidget>
      </div>
    </template>

    <!-- Расширенная статистика -->
    <template v-else-if="mode === 'extended' && extendedData">
      <div class="stats-grid">
        <StatsWidget title="По типам юнитов">
          <div class="table-scroll">
            <DataTable :value="extendedData.by_unit_types || []" size="small" :show-gridlines="false">
              <Column field="name" header="Тип" />
              <Column header="Время" style="width:100px">
                <template #body="{ data }">{{ roundHours(data.total_hours) }}</template>
              </Column>
              <Column field="tasks_count" header="Задачи" style="width:100px" />
            </DataTable>
          </div>
        </StatsWidget>

        <StatsWidget title="По отделам">
          <div class="table-scroll">
            <DataTable :value="extendedData.by_departments || []" size="small" :show-gridlines="false">
              <Column field="name" header="Отдел" />
              <Column field="tasks_count" header="Задачи" style="width:100px" />
            </DataTable>
          </div>
        </StatsWidget>

        <StatsWidget title="По типам юнитов для пользователей" class="full-width">
          <div class="table-scroll">
            <DataTable :value="flatUserTypes" size="small" :show-gridlines="false">
              <Column field="fio" header="Пользователь" />
              <Column field="type_name" header="Тип" />
              <Column header="Время" style="width:100px">
                <template #body="{ data }">{{ roundHours(data.hours) }}</template>
              </Column>
              <Column field="tasks_count" header="Задачи" style="width:100px" />
            </DataTable>
          </div>
        </StatsWidget>

        <StatsWidget title="Загруженность по дням" class="full-width">
          <CalendarGrid :data="extendedData.calendar || []" />
        </StatsWidget>
      </div>
    </template>

    <!-- Пустое состояние -->
    <div v-else-if="!loading" class="empty-state">
      <span class="material-symbols-outlined">bar_chart</span>
      <p>Нет данных за выбранный период</p>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import {
  getStatsCommon,
  getStatsExtended,
  exportStatsCommon,
  exportStatsExtended
} from '@/api/stats.js'
import { formatHours } from '@/utils/time.js'
import StatsPeriodControl from '@/components/stats/StatsPeriodControl.vue'
import StatsWidget from '@/components/stats/StatsWidget.vue'
import CalendarGrid from '@/components/stats/CalendarGrid.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import ProgressSpinner from 'primevue/progressspinner'

const { isAtLeast } = usePermission()
const notif = useNotificationsStore()

const mode = ref('common')
const loading = ref(false)
const commonData = ref(null)
const extendedData = ref(null)

const currentFrom = ref('')
const currentTo = ref('')

const canExport = computed(() => isAtLeast(ROLES.MANAGER))
const canExportUsers = computed(() => isAtLeast(ROLES.MANAGER))

const flatUserTypes = computed(() => {
  if (!extendedData.value?.by_unit_types_per_user) return []
  const result = []
  for (const user of extendedData.value.by_unit_types_per_user) {
    for (const type of user.unit_types || []) {
      result.push({
        fio: user.fio,
        type_name: type.name,
        hours: type.hours,
        tasks_count: type.tasks_count
      })
    }
  }
  return result
})

function roundHours(val) {
  return formatHours(val)
}

async function loadData() {
  if (!currentFrom.value || !currentTo.value) return
  loading.value = true
  try {
    if (mode.value === 'common') {
      commonData.value = await getStatsCommon(currentFrom.value, currentTo.value)
    } else {
      extendedData.value = await getStatsExtended(currentFrom.value, currentTo.value)
    }
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки статистики')
  } finally {
    loading.value = false
  }
}

function onPeriodChange({ from, to }) {
  currentFrom.value = from
  currentTo.value = to
  loadData()
}

function switchMode(m) {
  mode.value = m
  loadData()
}

async function handleExportCommon() {
  return exportStatsCommon(currentFrom.value, currentTo.value)
}

onMounted(() => {
  // данные загрузятся автоматически через @change от StatsPeriodControl
})
</script>

<style scoped>
.stats-view {
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 24px;
  height: 100%;
  overflow-y: auto;
}

.stats-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}

.stats-header h1 {
  margin: 0;
  font-size: 24px;
  font-weight: 800;
  color: var(--gw-text);
}

.stats-mode-toggle {
  display: flex;
  border: 1px solid var(--gw-border);
  border-radius: 10px;
  overflow: hidden;
}

.stats-mode-toggle button {
  padding: 8px 24px;
  background: transparent;
  border: none;
  cursor: pointer;
  font-size: 14px;
  color: var(--gw-text-secondary);
  transition: background 0.15s, color 0.15s;
}

.stats-mode-toggle button.active {
  background: var(--gw-primary);
  color: var(--color-on-primary);
  font-weight: 600;
}

.stats-mode-toggle button:hover:not(.active) {
  background: var(--gw-bg);
  color: var(--gw-text);
}

.loading-state {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 80px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 80px;
  color: var(--gw-text-secondary);
}

.empty-state .material-symbols-outlined {
  font-size: 56px;
  color: var(--gw-border);
}

.empty-state p {
  margin: 0;
  font-size: 15px;
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(340px, 1fr));
  gap: 16px;
}

.stats-grid .full-width {
  grid-column: 1 / -1;
}

/* Числа задач за период */
.task-numbers {
  display: flex;
  gap: 24px;
  flex-wrap: wrap;
  padding: 8px 0;
}

.task-num {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-width: 80px;
}

.task-num .num {
  font-size: 28px;
  font-weight: 800;
  line-height: 1;
  color: var(--gw-text);
}

.task-num .label {
  font-size: 12px;
  color: var(--gw-text-secondary);
  text-align: center;
}

.task-num.positive .num {
  color: var(--color-success);
}

.task-num.negative .num {
  color: var(--color-error);
}

.task-num.debt .num {
  color: var(--gw-primary);
}

.task-num.remaining .num {
  color: var(--gw-accent);
}

/* Горизонтальный скролл для таблиц */
.table-scroll {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

@media (max-width: 768px) {
  .stats-view {
    padding: 12px;
    gap: 12px;
  }

  .stats-header h1 {
    font-size: 20px;
  }

  .stats-grid {
    grid-template-columns: 1fr;
  }

  .task-numbers {
    gap: 12px;
  }

  .task-num .num {
    font-size: 24px;
  }
}
</style>
