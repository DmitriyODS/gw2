<template>
  <div class="stats-view">
    <!-- Прозрачный тулбар в стиле «Задач»: вкладки-режимы, период, действия. -->
    <div class="stats-sticky">
      <div class="stats-controls-row">
        <SegmentedTabs
          class="stats-mode-tabs"
          :model-value="mode"
          :tabs="modeTabs"
          :full-width="isMobile"
          @update:model-value="switchMode($event)"
        />
        <StatsPeriodControl class="stats-period" @change="onPeriodChange" />
        <div class="stats-actions">
          <button
            class="btn-glass"
            title="Сбросить раскладку виджетов"
            aria-label="Сбросить раскладку"
            @click="resetLayout"
          >
            <span class="material-symbols-outlined">restart_alt</span>
            <span class="stats-btn-label">Сбросить вид</span>
          </button>
          <a
            href="/tv"
            target="_blank"
            rel="noopener"
            class="btn-glass"
            :title="isMobile ? 'ТВ-режим' : 'Открыть ТВ-режим в новой вкладке'"
            :aria-label="'ТВ-режим'"
          >
            <span class="material-symbols-outlined">tv</span>
            <span class="stats-btn-label">ТВ-режим</span>
          </a>
        </div>
      </div>
    </div>

    <div class="stats-scroll">
    <!-- Нет активной компании (например, супер-админ): статистика — контент компании. -->
    <EmptyState
      v-if="!hasCompany"
      icon="domain_disabled"
      subtitle="Статистика доступна в контексте компании. Выберите или создайте компанию."
    />

    <div v-else-if="loading" class="loading-state">
      <BrandLoader />
    </div>

    <!-- Общая статистика -->
    <template v-else-if="mode === 'common' && commonData">
      <div class="stats-grid">
        <StatsWidget widget-id="tasks-period" title="Задачи за период" :export-fn="canExport ? handleExportCommon : null">
          <div class="task-tiles">
            <div class="task-tile tone-warning">
              <span class="material-symbols-outlined tile-icon">hourglass_top</span>
              <span class="tile-num">{{ commonData.tasks?.debt ?? 0 }}</span>
              <span class="tile-label">Долг</span>
            </div>
            <div class="task-tile tone-success">
              <span class="material-symbols-outlined tile-icon">trending_up</span>
              <span class="tile-num">+{{ commonData.tasks?.received ?? 0 }}</span>
              <span class="tile-label">Поступило</span>
            </div>
            <div class="task-tile tone-error">
              <span class="material-symbols-outlined tile-icon">task_alt</span>
              <span class="tile-num">−{{ commonData.tasks?.closed ?? 0 }}</span>
              <span class="tile-label">Закрыто</span>
            </div>
            <div class="task-tile tone-tertiary">
              <span class="material-symbols-outlined tile-icon">pending_actions</span>
              <span class="tile-num">{{ commonData.tasks?.remaining ?? 0 }}</span>
              <span class="tile-label">Осталось</span>
            </div>
          </div>
        </StatsWidget>

        <StatsWidget
          widget-id="by-employees"
          title="Отработка задач по сотрудникам"
          :export-fn="canExportUsers ? handleExportCommon : null"
        >
          <!-- Мобильный card-list -->
          <ul v-if="isMobile" class="m-list">
            <li
              v-for="(row, i) in commonData.tasks_by_employees || []"
              :key="i"
              class="m-row"
            >
              <div class="m-row-main">
                <span class="m-row-title">{{ row.fio }}</span>
                <span class="m-row-sub">{{ row.tasks_count }} задач</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
            </li>
            <li v-if="!(commonData.tasks_by_employees || []).length" class="m-empty">Нет данных</li>
          </ul>
          <!-- Десктоп таблица -->
          <div v-else class="table-scroll">
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

        <StatsWidget widget-id="by-hours" title="Задачи по часам">
          <ul v-if="isMobile" class="m-list">
            <li
              v-for="(row, i) in commonData.tasks_by_hours || []"
              :key="i"
              class="m-row"
            >
              <div class="m-row-main">
                <span class="m-row-title">{{ row.name }}</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
            </li>
            <li v-if="!(commonData.tasks_by_hours || []).length" class="m-empty">Нет данных</li>
          </ul>
          <div v-else class="table-scroll">
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

        <StatsWidget widget-id="responsibles" title="Ответственные по задачам">
          <ul v-if="isMobile && responsiblesData.length" class="m-list">
            <li v-for="r in responsiblesData" :key="r.user_id || r.id" class="m-row">
              <div class="m-row-main">
                <img class="m-avatar" :src="avatarOf(r)" :alt="r.fio" />
                <div class="m-row-text">
                  <span class="m-row-title">{{ r.fio }}</span>
                  <span v-if="r.post" class="m-row-sub">{{ r.post }}</span>
                </div>
              </div>
              <div class="m-row-tail">
                <span class="m-chip chip-primary" :title="'Открытые'">{{ r.open_count }}</span>
                <span class="m-chip chip-success" :title="'Закрытые'">{{ r.closed_count }}</span>
              </div>
            </li>
          </ul>
          <div v-else-if="responsiblesData.length" class="table-scroll">
            <DataTable :value="responsiblesData" size="small" :show-gridlines="false">
              <Column field="fio" header="Сотрудник">
                <template #body="{ data }">
                  <div class="resp-cell">
                    <img class="resp-ava" :src="avatarOf(data)" :alt="data.fio" />
                    <div class="resp-info">
                      <div class="resp-fio">{{ data.fio }}</div>
                      <div v-if="data.post" class="resp-post">{{ data.post }}</div>
                    </div>
                  </div>
                </template>
              </Column>
              <Column header="Открытые" style="width:120px">
                <template #body="{ data }">
                  <span class="resp-num open">{{ data.open_count }}</span>
                </template>
              </Column>
              <Column header="Закрытые" style="width:120px">
                <template #body="{ data }">
                  <span class="resp-num closed">{{ data.closed_count }}</span>
                </template>
              </Column>
            </DataTable>
          </div>
          <div v-else class="user-tasks-empty">Нет назначенных ответственных</div>
        </StatsWidget>

        <StatsWidget widget-id="user-tasks" title="Задачи с участием сотрудника">
          <div v-if="canSelectEmployee" class="employee-selector">
            <Select
              v-model="selectedEmployeeId"
              :options="employees"
              option-label="fio"
              option-value="id"
              placeholder="Выберите сотрудника"
              class="employee-select"
              filter
              filterPlaceholder="Поиск..."
              :loading="employeesLoading"
              @change="loadUserTasks"
            />
          </div>
          <div v-if="userTasksLoading" class="user-tasks-loading">
            <BrandLoader :size="48" />
          </div>
          <template v-else-if="userTasksData">
            <ul v-if="isMobile" class="m-list">
              <li
                v-for="(row, i) in userTasksData.tasks || []"
                :key="i"
                class="m-row"
              >
                <div class="m-row-main">
                  <span class="m-row-title">{{ row.task_name }}</span>
                </div>
                <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
              </li>
              <li v-if="!(userTasksData.tasks || []).length" class="m-empty">Нет данных</li>
            </ul>
            <div v-else class="table-scroll">
              <DataTable :value="userTasksData.tasks || []" size="small" :show-gridlines="false">
                <Column field="task_name" header="Задача" />
                <Column header="Время" style="width:110px">
                  <template #body="{ data }">{{ roundHours(data.total_hours) }}</template>
                </Column>
              </DataTable>
            </div>
            <div class="user-tasks-total">
              Всего задач: <strong>{{ userTasksData.tasks_count }}</strong>
            </div>
          </template>
          <div v-else class="user-tasks-empty">Нет данных за выбранный период</div>
        </StatsWidget>
      </div>
    </template>

    <!-- Расширенная статистика -->
    <template v-else-if="mode === 'extended' && extendedData">
      <div class="stats-grid">
        <StatsWidget widget-id="unit-types" title="По типам юнитов">
          <ul v-if="isMobile" class="m-list">
            <li
              v-for="(row, i) in extendedData.by_unit_types || []"
              :key="i"
              class="m-row"
            >
              <div class="m-row-main">
                <span class="m-row-title">{{ row.name }}</span>
                <span class="m-row-sub">{{ row.tasks_count }} задач</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
            </li>
            <li v-if="!(extendedData.by_unit_types || []).length" class="m-empty">Нет данных</li>
          </ul>
          <div v-else class="table-scroll">
            <DataTable :value="extendedData.by_unit_types || []" size="small" :show-gridlines="false">
              <Column field="name" header="Тип" />
              <Column header="Время" style="width:100px">
                <template #body="{ data }">{{ roundHours(data.total_hours) }}</template>
              </Column>
              <Column field="tasks_count" header="Задачи" style="width:100px" />
            </DataTable>
          </div>
        </StatsWidget>

        <StatsWidget widget-id="departments" title="По отделам">
          <ul v-if="isMobile" class="m-list">
            <li
              v-for="(row, i) in extendedData.by_departments || []"
              :key="i"
              class="m-row"
            >
              <div class="m-row-main">
                <span class="m-row-title">{{ row.name }}</span>
              </div>
              <span class="m-chip chip-primary">{{ row.tasks_count }}</span>
            </li>
            <li v-if="!(extendedData.by_departments || []).length" class="m-empty">Нет данных</li>
          </ul>
          <div v-else class="table-scroll">
            <DataTable :value="extendedData.by_departments || []" size="small" :show-gridlines="false">
              <Column field="name" header="Отдел" />
              <Column field="tasks_count" header="Задачи" style="width:100px" />
            </DataTable>
          </div>
        </StatsWidget>

        <StatsWidget widget-id="unit-types-per-user" title="По типам юнитов для пользователей">
          <ul v-if="isMobile" class="m-list">
            <li
              v-for="(row, i) in flatUserTypes"
              :key="i"
              class="m-row"
            >
              <div class="m-row-main">
                <span class="m-row-title">{{ row.fio }}</span>
                <span class="m-row-sub">{{ row.type_name }} • {{ row.tasks_count }} задач</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.hours) }}</span>
            </li>
            <li v-if="!flatUserTypes.length" class="m-empty">Нет данных</li>
          </ul>
          <div v-else class="table-scroll">
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

        <StatsWidget widget-id="calendar" title="Загруженность по дням">
          <CalendarGrid :data="extendedData.calendar || []" />
        </StatsWidget>
      </div>
    </template>

    <!-- Пустое состояние -->
    <EmptyState v-else-if="!loading" icon="bar_chart" subtitle="Нет данных за выбранный период" />
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useStatsLayout } from '@/composables/useStatsLayout.js'
import {
  getStatsCommon,
  getStatsExtended,
  exportStatsCommon,
  exportStatsExtended,
  getStatsUserTasks,
  getStatsEmployees,
  getStatsResponsibles,
} from '@/api/stats.js'
import { formatHours } from '@/utils/time.js'
import StatsPeriodControl from '@/components/stats/StatsPeriodControl.vue'
import StatsWidget from '@/components/stats/StatsWidget.vue'
import CalendarGrid from '@/components/stats/CalendarGrid.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import BrandLoader from '@/components/common/BrandLoader.vue'
import Select from 'primevue/select'

const { isAtLeast } = usePermission()
const authStore = useAuthStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()
const { reset: resetLayout } = useStatsLayout()

const mode = ref('common')
const loading = ref(false)
const commonData = ref(null)
const extendedData = ref(null)

const currentFrom = ref('')
const currentTo = ref('')

const canExport = computed(() => isAtLeast(ROLES.MANAGER))
const canExportUsers = computed(() => isAtLeast(ROLES.MANAGER))
const canSelectEmployee = computed(() => isAtLeast(ROLES.MANAGER))

const modeTabs = [
  { value: 'common', label: 'Общая', icon: 'dashboard', tutorial: 'stats-tab-common' },
  { value: 'extended', label: 'Расширенная', icon: 'analytics', tutorial: 'stats-tab-extended' },
]

const userTasksData = ref(null)
const userTasksLoading = ref(false)
const employees = ref([])
const employeesLoading = ref(false)
const selectedEmployeeId = ref(null)
const responsiblesData = ref([])

// Статистика — контент компании. Активная компания берётся из токена на бэке
// (?company_id= больше не используется). Супер-админ без компании контент не видит.
const hasCompany = computed(() => authStore.companyId != null)

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.user_id || u.id}/identicon`
}

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
  if (!hasCompany.value) { commonData.value = null; extendedData.value = null; return }
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
  if (mode.value === 'common') {
    loadUserTasks()
    loadResponsibles()
  }
}

async function loadUserTasks() {
  if (!currentFrom.value || !currentTo.value) return
  userTasksLoading.value = true
  try {
    const uid = selectedEmployeeId.value ?? authStore.user?.id
    userTasksData.value = await getStatsUserTasks(uid, currentFrom.value, currentTo.value)
  } catch (e) {
    notif.error(e.message || 'Ошибка загрузки задач сотрудника')
  } finally {
    userTasksLoading.value = false
  }
}

async function loadResponsibles() {
  try {
    responsiblesData.value = await getStatsResponsibles()
  } catch {
    responsiblesData.value = []
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

async function loadEmployees() {
  if (!canSelectEmployee.value || !hasCompany.value) return
  employeesLoading.value = true
  try {
    employees.value = await getStatsEmployees()
  } catch {
    employees.value = []
  } finally {
    employeesLoading.value = false
  }
}

// Пользователь сменил активную компанию (auth.companyId из токена) — перезагружаем.
watch(() => authStore.companyId, () => {
  loadData()
  loadEmployees()
})

onMounted(() => {
  loadEmployees()
})
</script>

<style scoped>
.stats-view {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.stats-sticky {
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px 24px 10px;
  z-index: 2;
}

.stats-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding: 20px 24px;
}

.stats-controls-row {
  display: flex;
  align-items: center;
  gap: 12px 16px;
  flex-wrap: wrap;
  padding: 2px 0;
}

.stats-actions {
  margin-left: auto;
  display: flex;
  align-items: center;
  gap: 8px;
}


.loading-state {
  display: flex;
  justify-content: center;
  align-items: center;
  padding: 80px;
}

.resp-cell {
  display: flex;
  align-items: center;
  gap: 10px;
}
.resp-ava {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.resp-info { min-width: 0; }
.resp-fio {
  font-size: 13.5px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.resp-post {
  font-size: 11.5px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.resp-num {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 36px;
  height: 26px;
  padding: 0 10px;
  border-radius: 13px;
  font-weight: 700;
  font-size: 13px;
}
.resp-num.open {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.resp-num.closed {
  background: var(--color-success-container, var(--color-surface-high));
  color: var(--color-on-success-container, var(--color-success));
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 16px;
  align-items: start;
}

@media (max-width: 1280px) {
  .stats-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 960px) {
  .stats-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}

/* === Задачи за период — M3 Expressive tiles === */
.task-tiles {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  gap: 12px;
  padding: 4px 0;
}

.task-tile {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  padding: 16px 18px;
  border-radius: var(--radius-xl, 20px);
  background: var(--tone-bg, var(--color-surface-high));
  color: var(--tone-fg, var(--color-text));
  min-height: 96px;
  overflow: hidden;
  transition: transform 0.18s, box-shadow 0.18s;
}

.task-tile:hover {
  box-shadow: var(--shadow-sm);
}

.task-tile .tile-icon {
  font-size: 22px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 24;
  opacity: 0.85;
}

.task-tile .tile-num {
  font-size: 32px;
  font-weight: 800;
  line-height: 1;
  letter-spacing: -0.02em;
}

.task-tile .tile-label {
  font-size: 12.5px;
  font-weight: 600;
  opacity: 0.78;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}

.task-tile.tone-warning {
  --tone-bg: var(--color-warning-container, var(--color-tertiary-container));
  --tone-fg: var(--color-on-warning-container, var(--color-on-tertiary-container));
}
.task-tile.tone-success {
  --tone-bg: var(--color-success-container);
  --tone-fg: var(--color-on-success-container);
}
.task-tile.tone-error {
  --tone-bg: var(--color-error-container);
  --tone-fg: var(--color-on-error-container);
}
.task-tile.tone-tertiary {
  --tone-bg: var(--color-tertiary-container);
  --tone-fg: var(--color-on-tertiary-container);
}

/* === Мобильные list-row карточки === */
.m-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.m-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px 14px;
  min-height: 56px;
  background: var(--color-surface-high);
  border-radius: var(--radius-lg, 16px);
}

.m-row-main {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 10px;
}

.m-row-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.m-row-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  word-break: break-word;
  overflow-wrap: anywhere;
  line-height: 1.25;
}

.m-row-sub {
  font-size: 12px;
  color: var(--color-text-dim);
  line-height: 1.2;
}

.m-row-tail {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

.m-avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.m-chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  height: 30px;
  padding: 0 12px;
  border-radius: var(--radius-full);
  font-weight: 700;
  font-size: 13px;
  white-space: nowrap;
  flex-shrink: 0;
}

.m-chip.chip-primary {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.m-chip.chip-success {
  background: var(--color-success-container);
  color: var(--color-on-success-container);
}
.m-chip.chip-tertiary {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}

.m-empty {
  padding: 24px 12px;
  text-align: center;
  color: var(--color-text-dim);
  font-size: 13px;
}

/* Горизонтальный скролл для таблиц */
.table-scroll {
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}

/* Таблицы внутри виджетов наследуют фон карточки — без своей подложки,
   чтобы не выбиваться из surface (задача #2). Границы/текст — на токенах. */
.table-scroll :deep(.p-datatable),
.table-scroll :deep(.p-datatable-table-container),
.table-scroll :deep(.p-datatable-header),
.table-scroll :deep(.p-datatable-thead),
.table-scroll :deep(.p-datatable-header-cell),
.table-scroll :deep(.p-datatable-thead > tr > th),
.table-scroll :deep(.p-datatable-tbody),
.table-scroll :deep(.p-datatable-tbody > tr),
.table-scroll :deep(.p-datatable-tbody > tr > td) {
  background: transparent;
  background-color: transparent;
}

.table-scroll :deep(.p-datatable-header-cell),
.table-scroll :deep(.p-datatable-thead > tr > th) {
  color: var(--color-text-dim);
  border-color: var(--color-outline-dim);
}

.table-scroll :deep(.p-datatable-tbody > tr) {
  color: var(--color-text);
}

.table-scroll :deep(.p-datatable-tbody > tr > td) {
  border-color: var(--color-outline-dim);
}

.table-scroll :deep(.p-datatable-tbody > tr:hover),
.table-scroll :deep(.p-datatable-tbody > tr.p-datatable-row-hover) {
  background: color-mix(in oklch, var(--color-primary) 7%, transparent);
}

.employee-selector {
  margin-bottom: 12px;
}

.employee-select {
  width: 280px;
  max-width: 100%;
}

.user-tasks-loading {
  display: flex;
  justify-content: center;
  padding: 20px;
}

.user-tasks-total {
  margin-top: 10px;
  font-size: 13px;
  color: var(--color-text-dim);
  text-align: right;
}

.user-tasks-total strong {
  color: var(--color-text);
}

.user-tasks-empty {
  text-align: center;
  padding: 20px;
  color: var(--color-text-dim);
  font-size: 14px;
}

@media (max-width: 768px) {
  .stats-sticky {
    padding: 10px 14px 6px;
    gap: 6px;
  }

  /* Компактная шапка: вкладки + действия в одной строке, период — под ними.
     Иначе управление съедает пол-экрана, а контенту остаётся узкая полоска. */
  .stats-controls-row {
    display: grid;
    grid-template-areas:
      'tabs actions'
      'period period';
    grid-template-columns: 1fr auto;
    align-items: center;
    gap: 6px 8px;
    padding: 0;
  }

  .stats-mode-tabs { grid-area: tabs; min-width: 0; }
  .stats-period { grid-area: period; }

  .stats-actions {
    grid-area: actions;
    margin-left: 0;
    gap: 6px;
  }

  /* Кнопки действий — только иконки, 38px. */
  .stats-btn-label { display: none; }
  .stats-actions .btn-glass {
    width: 38px;
    height: 38px;
    padding: 0;
    justify-content: center;
  }

  .stats-scroll {
    padding: 10px 14px;
    padding-bottom: calc(64px + 12px + env(safe-area-inset-bottom, 0px));
    gap: 12px;
  }

  .stats-grid {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .task-tiles {
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }

  .task-tile {
    padding: 14px;
    min-height: 92px;
  }

  .task-tile .tile-num {
    font-size: 28px;
  }
}

@media (max-width: 480px) {
  .stats-sticky {
    padding: 8px 12px 6px;
  }

  .stats-scroll {
    padding: 10px 12px;
    padding-bottom: calc(64px + 12px + env(safe-area-inset-bottom, 0px));
  }

  .task-tile {
    padding: 12px;
    min-height: 88px;
  }

  .task-tile .tile-num {
    font-size: 26px;
  }

  .task-tile .tile-label {
    font-size: 11px;
  }

  .m-row {
    padding: 10px 12px;
  }

  .m-avatar {
    width: 32px;
    height: 32px;
  }
}

@media (max-width: 360px) {
  .task-tiles {
    grid-template-columns: 1fr;
  }
}
</style>
