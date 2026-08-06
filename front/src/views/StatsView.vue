<template>
  <AppPage
    class="stats"
    title="Статистика"
    :commands="commands"
    :loading="hasCompany && loading"
    @command="onCommand"
  >
    <!-- В тесной панели строка управления оставляет только период: режим
         («Общая»/«Расширенная») уходит в меню «ещё» — см. commands. -->
    <template #subhead="{ narrow: tight }">
      <AppTabs
        v-if="!tight"
        class="stats-mode-tabs"
        :model-value="mode"
        :tabs="modeTabs"
        @update:model-value="switchMode($event)"
      />
      <StatsPeriodControl class="stats-period" :compact="tight" @change="onPeriodChange" />
    </template>

    <!-- Нет активной компании (например, супер-админ): статистика — контент компании. -->
    <EmptyState
      v-if="!hasCompany"
      icon="domain_disabled"
      subtitle="Статистика доступна в контексте компании. Выберите или создайте компанию."
    />

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
          <MobileStatList v-if="isMobile" :items="commonData.tasks_by_employees || []">
            <template #default="{ row }">
              <div class="m-row-main">
                <span class="m-row-title">{{ row.fio }}</span>
                <span class="m-row-sub">{{ row.tasks_count }} задач</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
            </template>
          </MobileStatList>
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
          <MobileStatList v-if="isMobile" :items="commonData.tasks_by_hours || []">
            <template #default="{ row }">
              <div class="m-row-main">
                <span class="m-row-title">{{ row.name }}</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
            </template>
          </MobileStatList>
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
          <MobileStatList v-if="isMobile && responsiblesData.length" :items="responsiblesData">
            <template #default="{ row: r }">
              <div class="m-row-main">
                <img class="m-avatar" :src="avatarOf(r)" :alt="r.fio" />
                <div class="m-row-text">
                  <span class="m-row-title">{{ r.fio }}</span>
                  <span v-if="r.post" class="m-row-sub">{{ r.post }}</span>
                </div>
              </div>
              <div class="m-row-tail">
                <span class="m-chip chip-primary" title="Открытые">{{ r.open_count }}</span>
                <span class="m-chip chip-success" title="Закрытые">{{ r.closed_count }}</span>
              </div>
            </template>
          </MobileStatList>
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
            <MobileStatList v-if="isMobile" :items="userTasksData.tasks || []">
              <template #default="{ row }">
                <div class="m-row-main">
                  <span class="m-row-title">{{ row.task_name }}</span>
                </div>
                <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
              </template>
            </MobileStatList>
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
          <MobileStatList v-if="isMobile" :items="extendedData.by_unit_types || []">
            <template #default="{ row }">
              <div class="m-row-main">
                <span class="m-row-title">{{ row.name }}</span>
                <span class="m-row-sub">{{ row.tasks_count }} задач</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.total_hours) }}</span>
            </template>
          </MobileStatList>
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
          <MobileStatList v-if="isMobile" :items="extendedData.by_departments || []">
            <template #default="{ row }">
              <div class="m-row-main">
                <span class="m-row-title">{{ row.name }}</span>
              </div>
              <span class="m-chip chip-primary">{{ row.tasks_count }}</span>
            </template>
          </MobileStatList>
          <div v-else class="table-scroll">
            <DataTable :value="extendedData.by_departments || []" size="small" :show-gridlines="false">
              <Column field="name" header="Отдел" />
              <Column field="tasks_count" header="Задачи" style="width:100px" />
            </DataTable>
          </div>
        </StatsWidget>

        <StatsWidget widget-id="unit-types-per-user" title="По типам юнитов для пользователей">
          <MobileStatList v-if="isMobile" :items="flatUserTypes">
            <template #default="{ row }">
              <div class="m-row-main">
                <span class="m-row-title">{{ row.fio }}</span>
                <span class="m-row-sub">{{ row.type_name }} • {{ row.tasks_count }} задач</span>
              </div>
              <span class="m-chip chip-tertiary">{{ roundHours(row.hours) }}</span>
            </template>
          </MobileStatList>
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
  </AppPage>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
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
import MobileStatList from '@/components/stats/MobileStatList.vue'
import CalendarGrid from '@/components/stats/CalendarGrid.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import DataTable from 'primevue/datatable'
import Column from 'primevue/column'
import BrandLoader from '@/components/common/BrandLoader.vue'
import Select from 'primevue/select'

const { isAtLeast } = usePermission()
const authStore = useAuthStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()
const router = useRouter()
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

/* Действия раздела. Все второстепенные — уезжают в меню «ещё» и не спорят с
   переключателем периода за место в строке; на телефоне туда же уходит выбор
   режима статистики. */
const commands = computed(() => [
  ...(isMobile.value ? [{
    key: 'mode',
    label: `Статистика: ${(modeTabs.find((t) => t.value === mode.value)?.label || '').toLowerCase()}`,
    icon: modeTabs.find((t) => t.value === mode.value)?.icon,
    children: modeTabs.map((t) => ({
      key: `mode:${t.value}`,
      label: t.label,
      icon: t.value === mode.value ? 'check' : t.icon,
    })),
  }] : []),
  { key: 'reset', label: 'Сбросить вид', icon: 'restart_alt' },
  { key: 'tv', label: 'ТВ-режим', icon: 'tv' },
])

function onCommand(key) {
  if (key.startsWith('mode:')) switchMode(key.slice(5))
  else if (key === 'reset') resetLayout()
  else if (key === 'tv') openTv()
}

/* ТВ-режим — табло, которое обычно вешают на отдельный экран, поэтому сперва
   пробуем открыть его вкладкой. Не вышло (блокировщик всплывающих окон, мобильная
   обёртка) — уходим туда навигацией: маршрут полноэкранный, так что табло всё
   равно займёт весь экран, просто в этом же окне. */
function openTv() {
  const opened = window.open('/tv', '_blank', 'noopener')
  if (!opened) router.push('/tv')
}

const modeTabs = [
  { value: 'common', label: 'Общая', icon: 'dashboard' },
  { value: 'extended', label: 'Расширенная', icon: 'analytics' },
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

/* Колонок столько, сколько влезает в ПАНЕЛЬ: раздел живёт окном, и ширина
   экрана про него ничего не знает — в узком окне сетка оставалась четырёхколоночной
   и карточки вылезали за края. Дубль @media — для старого WebView без @container. */
@container (max-width: 1280px) {
  .stats-grid { grid-template-columns: repeat(3, minmax(0, 1fr)); }
}
@container (max-width: 960px) {
  .stats-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
/* Одна колонка — ОБЯЗАТЕЛЬНО minmax(0, 1fr): голый `1fr` не уже своего
   min-content, поэтому широкое содержимое карточки (таблица, длинное имя)
   растягивало колонку, и весь раздел уезжал вбок вместе с карточками. */
@container (max-width: 700px) {
  .stats-grid { grid-template-columns: minmax(0, 1fr); gap: 12px; }
  .task-tiles { grid-template-columns: 1fr 1fr; gap: 8px; }
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
  grid-template-columns: repeat(auto-fit, minmax(min(120px, 100%), 1fr));
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
/* Каркас списка (сама строка, пустое состояние, «показать ещё») живёт в
   MobileStatList; здесь — только содержимое строки, оно приходит слотом. */
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
  /* Вкладки режимов берут строку целиком, период встаёт под ними: рядом они
     сжимали друг друга, а действия ушли в меню «ещё» шапки. */
  .stats-mode-tabs { flex: 1 1 100%; min-width: 0; }
  .stats-period { flex: 1 1 100%; min-width: 0; }

  .stats-grid {
    grid-template-columns: minmax(0, 1fr);
    gap: 12px;
  }

  /* Плитки-цифры на телефоне: две в ряд и мельче — четыре ключевых числа
     должны читаться одним взглядом, не занимая экран целиком. */
  .task-tiles {
    grid-template-columns: 1fr 1fr;
    gap: 8px;
  }

  .task-tile {
    gap: 2px;
    padding: 12px;
    min-height: 0;
  }

  .task-tile .tile-icon { font-size: 18px; }
  .task-tile .tile-num { font-size: 24px; }
  .task-tile .tile-label { font-size: 10.5px; letter-spacing: 0.03em; }

  .m-avatar {
    width: 32px;
    height: 32px;
  }

  .employee-select { width: 100%; }
  .employee-selector { margin-bottom: 8px; }
}
</style>
