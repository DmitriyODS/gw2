<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <!-- Тулбар в стиле «Задач»: поиск на всю ширину, статы-чипы, главное действие. -->
      <div class="admin-toolbar">
        <h1 class="cmp-title">{{ isSuper ? 'Компании' : 'Мои компании' }}</h1>
        <SearchField v-model="search" placeholder="Поиск по названию" hotkey />
        <span class="chip-tint chip-tint--primary cmp-chip">
          <span class="material-symbols-outlined">domain</span>
          <strong>{{ rows.length }}</strong>&nbsp;{{ isSuper ? 'всего' : 'под управлением' }}
        </span>
        <template v-if="isSuper">
          <span class="chip-tint chip-tint--success cmp-chip">
            <strong>{{ activeCount }}</strong>&nbsp;активных
          </span>
          <span v-if="disabledCount" class="chip-tint chip-tint--error cmp-chip">
            <strong>{{ disabledCount }}</strong>&nbsp;отключённых
          </span>
        </template>
        <button class="btn-grad desktop-only" @click="openCreate">
          <span class="material-symbols-outlined">add</span>
          <span>Новая компания</span>
        </button>
      </div>
    </header>

    <div ref="bodyRef" class="admin-body">
      <div v-if="isMobile" class="cmp-cards">
        <div v-if="loading" class="state-block">
          <ProgressSpinner />
        </div>
        <EmptyState
          v-else-if="!visible.length"
          class="cmp-cards-empty"
          :icon="search ? 'search_off' : 'domain'"
          :title="search ? 'Ничего не нашли' : 'Компаний пока нет'"
          :subtitle="search ? 'Попробуйте уточнить запрос.' : 'Создайте компанию — вы станете её администратором.'"
        />
        <template v-else>
          <article
            v-for="c in visible"
            :key="c.id"
            class="cmp-card"
            :class="{ off: !c.is_active }"
            tabindex="0"
            @click="openManage(c)"
            @keydown.enter.prevent="openManage(c)"
          >
            <div class="cmp-card-top">
              <span class="cmp-avatar" :class="['tone-' + toneOf(c)]">{{ initials(c.name) }}</span>
              <div class="cmp-card-text">
                <div class="cmp-card-name">{{ c.name }}</div>
                <div v-if="c.description" class="cmp-card-desc">{{ c.description }}</div>
              </div>
              <label v-if="isSuper" class="toggle" @click.stop>
                <input type="checkbox" :checked="c.is_active" :disabled="togglingId === c.id" @change="onToggle(c)" />
                <span class="toggle-track" />
              </label>
              <span v-else class="role-badge" :class="{ creator: isCreator(c) }">{{ roleBadge(c) }}</span>
            </div>

            <div class="cmp-card-stats">
              <span class="stat">
                <span class="material-symbols-outlined">groups</span>
                <strong>{{ c.employees_count }}</strong>
              </span>
              <span class="stat">
                <span class="material-symbols-outlined">checklist</span>
                <strong>{{ c.tasks_count }}</strong>
              </span>
              <span class="stat date">
                <span class="material-symbols-outlined">event</span>
                {{ fmtDate(c.created_at) }}
              </span>
            </div>

            <div v-if="isSuper" class="cmp-card-actions" @click.stop>
              <button class="card-act danger" title="Удалить" @click="askDelete(c)">
                <span class="material-symbols-outlined">delete</span>
              </button>
            </div>
          </article>
        </template>
      </div>

      <AppDataTable
        v-else
        :value="visible"
        :loading="loading"
        v-model:sort-field="sortField"
        v-model:sort-order="sortOrder"
        :row-class="() => 'row-clickable'"
        empty-message="Компаний не найдено"
        @row-click="onRowClick"
      >
        <Column field="name" header="Компания" sortable :sort-field="(d) => d.name?.toLowerCase()">
          <template #body="{ data }">
            <div class="cell-company">
              <span class="cmp-avatar" :class="['tone-' + toneOf(data)]" :title="data.name">
                {{ initials(data.name) }}
              </span>
              <div class="cmp-name-text">
                <div class="cmp-name-main" :class="{ off: !data.is_active }">{{ data.name }}</div>
                <div v-if="data.description" class="cmp-name-sub">{{ data.description }}</div>
              </div>
            </div>
          </template>
        </Column>

        <Column field="created_at" header="Создана" sortable style="width: 140px">
          <template #body="{ data }">
            <span class="mono">{{ fmtDate(data.created_at) }}</span>
          </template>
        </Column>

        <Column v-if="isSuper" header="Создатель" style="min-width: 160px">
          <template #body="{ data }">
            <span v-if="creatorName(data)" class="creator-chip">
              <span class="material-symbols-outlined">person</span>
              {{ creatorName(data) }}
            </span>
            <span v-else class="muted">—</span>
          </template>
        </Column>
        <Column v-else header="Роль" style="width: 150px">
          <template #body="{ data }">
            <span class="role-badge" :class="{ creator: isCreator(data) }">{{ roleBadge(data) }}</span>
          </template>
        </Column>

        <Column field="employees_count" header="Сотрудников" sortable style="width: 200px">
          <template #body="{ data }">
            <div class="num-cell">
              <span class="num-bar">
                <span class="num-bar-fill" :style="{ width: barWidth(data.employees_count, maxEmployees) + '%' }" />
              </span>
              <span class="num-val">{{ data.employees_count }}</span>
            </div>
          </template>
        </Column>

        <Column field="tasks_count" header="Задач" sortable style="width: 200px">
          <template #body="{ data }">
            <div class="num-cell">
              <span class="num-bar tasks">
                <span class="num-bar-fill" :style="{ width: barWidth(data.tasks_count, maxTasks) + '%' }" />
              </span>
              <span class="num-val">{{ data.tasks_count }}</span>
            </div>
          </template>
        </Column>

        <Column v-if="isSuper" header="Статус" style="width: 170px">
          <template #body="{ data }">
            <label class="toggle" :title="data.is_active ? 'Активна' : 'Отключена'" @click.stop>
              <input type="checkbox" :checked="data.is_active" :disabled="togglingId === data.id" @change="onToggle(data)" />
              <span class="toggle-track" />
              <span :class="['toggle-label', { on: data.is_active }]">
                {{ data.is_active ? 'Активна' : 'Отключена' }}
              </span>
            </label>
          </template>
        </Column>

        <Column v-if="isSuper" header="" style="width: 64px" body-style="text-align: right">
          <template #body="{ data }">
            <div class="row-actions" @click.stop>
              <button class="icon-btn danger" title="Удалить" @click="askDelete(data)">
                <span class="material-symbols-outlined">delete</span>
              </button>
            </div>
          </template>
        </Column>
      </AppDataTable>
    </div>

    <CreateCompanyDialog v-model="createOpen" />

    <AppDialog
      v-model="confirmOpen"
      tone="danger"
      icon="warning"
      size="sm"
      :title="`Удалить компанию «${deleteTarget?.name}»?`"
      :busy="deleting"
      :closable="!deleting"
      :actions="[
        { kind: 'cancel', label: 'Отмена', disabled: deleting },
        { kind: 'confirm', label: 'Удалить', icon: 'delete', disabled: deleting },
      ]"
      @confirm="doDelete"
    >
      <p v-if="deleteTarget?.employees_count || deleteTarget?.tasks_count" class="confirm-warn">
        В компании остаются
        <strong v-if="deleteTarget?.employees_count">{{ deleteTarget.employees_count }} сотрудник(а/ов)</strong>
        <template v-if="deleteTarget?.employees_count && deleteTarget?.tasks_count">, </template>
        <strong v-if="deleteTarget?.tasks_count">{{ deleteTarget.tasks_count }} задач(и)</strong>.
        Все данные будут <strong>удалены каскадно</strong>: задачи, юниты, чаты, звонки.
        Восстановить будет нельзя.
      </p>
      <p v-else>Компания пустая — данных не пострадает.</p>
    </AppDialog>

    <AppFab icon="add" aria-label="Новая компания" @click="openCreate" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import Column from 'primevue/column'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import AppDataTable from '@/components/common/AppDataTable.vue'
import AppFab from '@/components/common/AppFab.vue'
import CreateCompanyDialog from '@/components/common/CreateCompanyDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { usePermission } from '@/composables/usePermission.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { listMyCompanies } from '@/api/companies.js'

const { isMobile } = useBreakpoint()
const bodyRef = ref(null)

const router = useRouter()
const companies = useCompaniesStore()
const notif = useNotificationsStore()
const auth = useAuthStore()
const { isSuperAdmin } = usePermission()
const isSuper = computed(() => isSuperAdmin())

// Источник данных: супер-админ видит ВСЕ компании (платформа, стор), обычный
// пользователь — те, где он администратор/создатель (эндпоинт /companies/mine).
const myItems = ref([])
const myLoading = ref(false)
const rows = computed(() => (isSuper.value ? companies.items : myItems.value))
const loading = computed(() =>
  isSuper.value ? companies.loading && !companies.loaded : myLoading.value)

const search = ref('')
const sortField = ref('created_at')
const sortOrder = ref(-1)

const createOpen = ref(false)
const confirmOpen = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)
const togglingId = ref(null)

onMounted(loadData)

async function loadData() {
  if (isSuper.value) {
    companies.load(true)
    return
  }
  myLoading.value = true
  try {
    const res = await listMyCompanies()
    myItems.value = res.items || []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить компании')
  } finally {
    myLoading.value = false
  }
}

const activeCount = computed(() => rows.value.filter((c) => c.is_active).length)
const disabledCount = computed(() => rows.value.filter((c) => !c.is_active).length)

const visible = computed(() => {
  const q = search.value.trim().toLowerCase()
  return q ? rows.value.filter((c) => c.name.toLowerCase().includes(q)) : rows.value
})

const maxEmployees = computed(() => Math.max(1, ...rows.value.map((c) => c.employees_count || 0)))
const maxTasks = computed(() => Math.max(1, ...rows.value.map((c) => c.tasks_count || 0)))

function barWidth(value, max) {
  if (!max) return 0
  return Math.min(100, Math.round(((value || 0) / max) * 100))
}

function fmtDate(s) {
  if (!s) return '—'
  return new Date(s).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function initials(name) {
  if (!name) return '?'
  const parts = name.trim().split(/\s+/).slice(0, 2)
  return parts.map((p) => p[0]).join('').toUpperCase()
}

function creatorName(c) {
  return c.creator?.fio || c.creator?.name || null
}

const TONES = ['primary', 'secondary', 'tertiary']
function toneOf(c) {
  return TONES[(c.id || 0) % TONES.length]
}

// Создатель ли текущий пользователь этой компании (полные права на участников).
function isCreator(c) {
  return c.created_by != null && c.created_by === auth.userId
}
function roleBadge(c) {
  return isCreator(c) ? 'Создатель' : 'Администратор'
}

function openManage(c) {
  router.push(`/companies/${c.id}`)
}
function onRowClick(e) {
  openManage(e.data)
}
function openCreate() {
  createOpen.value = true
}

async function onToggle(c) {
  togglingId.value = c.id
  try {
    await companies.toggleActive(c.id, !c.is_active)
    notif.success(c.is_active ? 'Компания отключена' : 'Компания включена')
  } catch (e) {
    notif.error(e?.message || 'Не удалось переключить статус')
  } finally {
    togglingId.value = null
  }
}

function askDelete(c) {
  deleteTarget.value = c
  confirmOpen.value = true
}

async function doDelete() {
  if (!deleteTarget.value) return
  deleting.value = true
  try {
    await companies.remove(deleteTarget.value.id)
    notif.success('Компания удалена')
    confirmOpen.value = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить')
  } finally {
    deleting.value = false
  }
}
</script>

<style scoped>
/* Тулбар без подложки — прозрачная «плавающая» шапка как в «Задачах». */
.admin-sticky { background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; }
.admin-sticky::after { display: none; }

.cmp-title {
  margin: 0;
  font-size: 20px;
  font-weight: 800;
  color: var(--color-text);
  white-space: nowrap;
}

@media (max-width: 768px) {
  /* На мобильном заголовок — своя строка над поиском и чипами. */
  .cmp-title { flex-basis: 100%; font-size: 18px; }
}

.cell-company {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}
.cmp-avatar {
  width: 38px;
  height: 38px;
  border-radius: var(--radius-md);
  display: grid;
  place-items: center;
  font-weight: 800;
  font-size: 13px;
  letter-spacing: 0.02em;
  flex-shrink: 0;
}
.cmp-avatar.tone-primary {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.cmp-avatar.tone-secondary {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.cmp-avatar.tone-tertiary {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.cmp-name-text { min-width: 0; }
.cmp-name-main {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cmp-name-main.off { opacity: 0.55; }
.cmp-name-sub {
  font-size: 12px;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 340px;
  margin-top: 1px;
}

.role-badge {
  display: inline-flex;
  align-items: center;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  font-size: 12px;
  font-weight: 600;
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  white-space: nowrap;
}
.role-badge.creator {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.mono {
  font-variant-numeric: tabular-nums;
  color: var(--color-text-dim);
  font-size: 13px;
  font-weight: 500;
}

.num-cell {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: flex-end;
}
.num-bar {
  flex: 1;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-surface-highest);
  overflow: hidden;
  min-width: 32px;
  max-width: 120px;
}
.num-bar-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--color-primary);
  transition: width .3s cubic-bezier(0.4, 0, 0.2, 1);
}
.num-bar.tasks .num-bar-fill { background: var(--color-tertiary); }
.num-val {
  font-variant-numeric: tabular-nums;
  font-weight: 700;
  font-size: 13.5px;
  color: var(--color-text);
  min-width: 28px;
  text-align: right;
}

.muted {
  color: var(--color-text-dim);
  font-style: italic;
  font-size: 13px;
}

.creator-chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 3px 10px 3px 8px;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  font-size: 12.5px;
  color: var(--color-text);
  max-width: 100%;
}
.creator-chip .material-symbols-outlined { font-size: 15px; color: var(--color-text-dim); }

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
}
.toggle input { display: none; }
.toggle-track {
  position: relative;
  width: 44px;
  height: 24px;
  border-radius: var(--radius-full);
  background: var(--color-surface-highest);
  border: 2px solid var(--color-outline);
  box-sizing: border-box;
  transition: background .18s, border-color .18s;
  flex-shrink: 0;
}
.toggle-track::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-outline);
  transform: translateY(-50%);
  transition: transform .2s cubic-bezier(0.4, 0, 0.2, 1),
              background .2s, width .2s, height .2s, left .2s;
}
.toggle input:checked + .toggle-track {
  background: var(--color-success);
  border-color: var(--color-success);
}
.toggle input:checked + .toggle-track::after {
  width: 16px;
  height: 16px;
  left: 24px;
  background: var(--color-on-success);
}
.toggle-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-dim);
}
.toggle-label.on { color: var(--color-success); }

.row-actions {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}
.icon-btn {
  appearance: none;
  border: none;
  background: transparent;
  width: 34px;
  height: 34px; min-height: 0;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background .14s, color .14s;
}
.icon-btn:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.icon-btn.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.icon-btn .material-symbols-outlined { font-size: 18px; }

.confirm-warn { color: var(--color-text); }
.confirm-warn strong { color: var(--color-error); }

.cmp-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.cmp-card {
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  cursor: pointer;
  outline: none;
  transition: background .16s, border-color .16s, box-shadow .16s;
}

.cmp-card:hover, .cmp-card:focus-visible {
  background: var(--glass-hover-bg);
  box-shadow: var(--glass-edge), var(--shadow-sm);
}

.cmp-card.off { opacity: 0.65; }

.cmp-card-top {
  display: flex;
  align-items: center;
  gap: 12px;
}

.cmp-card-top .cmp-avatar { width: 44px; height: 44px; border-radius: var(--radius-md); flex-shrink: 0; }

.cmp-card-text { flex: 1; min-width: 0; }
.cmp-card-name {
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cmp-card-desc {
  font-size: 12.5px;
  color: var(--color-text-dim);
  margin-top: 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cmp-card-stats {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.cmp-card-stats .stat {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  font-size: 12.5px;
  color: var(--color-text);
}
.cmp-card-stats .stat .material-symbols-outlined { font-size: 15px; color: var(--color-text-dim); }
.cmp-card-stats .stat.date { color: var(--color-text-dim); }
.cmp-card-stats .stat strong { font-weight: 700; font-variant-numeric: tabular-nums; }

.cmp-card-actions {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
  border-top: 1px solid var(--color-outline-dim);
  padding-top: 10px;
  margin-top: -2px;
}

.card-act {
  appearance: none;
  border: 1px solid var(--acrylic-border);
  background: var(--color-surface-high);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  width: 40px;
  height: 40px;
  border-radius: var(--radius-full);
  display: grid;
  place-items: center;
  color: var(--color-text);
  cursor: pointer;
  transition: background .14s, color .14s;
}
.card-act:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.card-act.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.card-act .material-symbols-outlined { font-size: 20px; }

.cmp-cards-empty {
  background: var(--acrylic-card-bg);
  border-radius: var(--radius-xl);
}

.state-block { display: grid; place-items: center; padding: 48px; }

@media (max-width: 768px) {
  .hide-narrow { display: none; }
  .desktop-only { display: none; }

  .admin-toolbar :deep(.search-field) { flex: 1 1 100%; max-width: 100%; }
}
</style>
