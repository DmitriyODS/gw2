<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <div class="page-head">
        <div class="page-head-text">
          <h1 class="page-head-title">Компании</h1>
          <div class="page-head-meta">
            <span class="meta-stat">
              <span class="material-symbols-outlined">domain</span>
              <strong>{{ companies.items.length }}</strong> всего
            </span>
            <span class="meta-dot" aria-hidden="true">·</span>
            <span class="meta-stat online">
              <span class="presence-pulse" />
              <strong>{{ activeCount }}</strong> активных
            </span>
            <template v-if="disabledCount">
              <span class="meta-dot" aria-hidden="true">·</span>
              <span class="meta-stat error">
                <strong>{{ disabledCount }}</strong> отключённых
              </span>
            </template>
          </div>
        </div>
        <button class="btn-filled desktop-only" @click="openCreate">
          <span class="material-symbols-outlined">add</span>
          <span>Новая компания</span>
        </button>
      </div>

      <div class="admin-toolbar">
        <div class="cmp-search">
          <span class="material-symbols-outlined">search</span>
          <input
            v-model.trim="search"
            placeholder="Поиск по названию или руководителю"
          />
          <button
            v-if="search"
            class="cmp-search-clear"
            @click="search = ''"
            aria-label="Очистить"
          >
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
      </div>
    </header>

    <div ref="bodyRef" class="admin-body">
      <!-- Мобильное представление: карточки вместо таблицы.
           AppDataTable горизонтально скроллится на узких экранах — UX плохой,
           и поэтому на ≤768px рендерим компактные карточки с теми же данными
           и действиями. -->
      <div v-if="isMobile" class="cmp-cards">
        <div v-if="loading" class="state-block">
          <ProgressSpinner />
        </div>
        <div v-else-if="!visible.length" class="cmp-cards-empty">
          <div class="empty-icon-circle">
            <span class="material-symbols-outlined">{{ search ? 'search_off' : 'domain' }}</span>
          </div>
          <h3>{{ search ? 'Ничего не нашли' : 'Компаний пока нет' }}</h3>
          <p v-if="search">Попробуйте уточнить запрос.</p>
        </div>
        <template v-else>
          <article
            v-for="c in visible"
            :key="c.id"
            class="cmp-card"
            :class="{ off: !c.is_active }"
            tabindex="0"
            @click="openEdit(c)"
            @keydown.enter.prevent="openEdit(c)"
          >
            <div class="cmp-card-top">
              <span
                class="cmp-avatar"
                :class="['tone-' + toneOf(c)]"
              >{{ initials(c.name) }}</span>
              <div class="cmp-card-text">
                <div class="cmp-card-name">{{ c.name }}</div>
                <div v-if="c.description" class="cmp-card-desc">{{ c.description }}</div>
              </div>
              <label class="toggle" @click.stop>
                <input
                  type="checkbox"
                  :checked="c.is_active"
                  :disabled="togglingId === c.id"
                  @change="onToggle(c)"
                />
                <span class="toggle-track" />
              </label>
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

            <div class="cmp-card-actions" @click.stop>
              <button class="card-act" title="Редактировать" @click="openEdit(c)">
                <span class="material-symbols-outlined">edit</span>
              </button>
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
              <span
                class="cmp-avatar"
                :class="['tone-' + toneOf(data)]"
                :title="data.name"
              >
                {{ initials(data.name) }}
              </span>
              <div class="cmp-name-text">
                <div class="cmp-name-main" :class="{ off: !data.is_active }">
                  {{ data.name }}
                </div>
                <div v-if="data.description" class="cmp-name-sub">
                  {{ data.description }}
                </div>
              </div>
            </div>
          </template>
        </Column>

        <Column field="created_at" header="Создана" sortable style="width: 140px">
          <template #body="{ data }">
            <span class="mono">{{ fmtDate(data.created_at) }}</span>
          </template>
        </Column>

        <Column
          field="employees_count"
          header="Сотрудников"
          sortable
          style="width: 200px"
        >
          <template #body="{ data }">
            <div class="num-cell">
              <span class="num-bar">
                <span
                  class="num-bar-fill"
                  :style="{ width: barWidth(data.employees_count, maxEmployees) + '%' }"
                />
              </span>
              <span class="num-val">{{ data.employees_count }}</span>
            </div>
          </template>
        </Column>

        <Column
          field="tasks_count"
          header="Задач"
          sortable
          style="width: 200px"
        >
          <template #body="{ data }">
            <div class="num-cell">
              <span class="num-bar tasks">
                <span
                  class="num-bar-fill"
                  :style="{ width: barWidth(data.tasks_count, maxTasks) + '%' }"
                />
              </span>
              <span class="num-val">{{ data.tasks_count }}</span>
            </div>
          </template>
        </Column>

        <Column header="Статус" style="width: 170px">
          <template #body="{ data }">
            <label class="toggle" :title="data.is_active ? 'Активна' : 'Отключена'" @click.stop>
              <input
                type="checkbox"
                :checked="data.is_active"
                :disabled="togglingId === data.id"
                @change="onToggle(data)"
              />
              <span class="toggle-track" />
              <span :class="['toggle-label', { on: data.is_active }]">
                {{ data.is_active ? 'Активна' : 'Отключена' }}
              </span>
            </label>
          </template>
        </Column>

        <Column header="" style="width: 96px" body-style="text-align: right">
          <template #body="{ data }">
            <div class="row-actions" @click.stop>
              <button
                class="icon-btn"
                title="Редактировать"
                @click="openEdit(data)"
              >
                <span class="material-symbols-outlined">edit</span>
              </button>
              <button
                class="icon-btn danger"
                title="Удалить"
                @click="askDelete(data)"
              >
                <span class="material-symbols-outlined">delete</span>
              </button>
            </div>
          </template>
        </Column>
      </AppDataTable>
    </div>

    <CompanyFormDialog
      ref="formDlgRef"
      v-model="formOpen"
      :company="editTarget"
      @save="onSave"
    />

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

    <AppFab
      icon="add"
      label="Создать"
      :collapsed="isCompact"
      aria-label="Новая компания"
      @click="openCreate"
    />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import Column from 'primevue/column'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import AppDataTable from '@/components/common/AppDataTable.vue'
import AppFab from '@/components/common/AppFab.vue'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useScrollCollapse } from '@/composables/useScrollCollapse.js'
import CompanyFormDialog from '@/components/companies/CompanyFormDialog.vue'

const { isMobile } = useBreakpoint()
const bodyRef = ref(null)
const { isCompact } = useScrollCollapse(bodyRef)

const companies = useCompaniesStore()
const notif = useNotificationsStore()

const loading = computed(() => companies.loading && !companies.loaded)
const search = ref('')
const sortField = ref('created_at')
const sortOrder = ref(-1)

const formOpen = ref(false)
const editTarget = ref(null)
const formDlgRef = ref(null)

const confirmOpen = ref(false)
const deleteTarget = ref(null)
const deleting = ref(false)
const togglingId = ref(null)

onMounted(() => companies.load(true))

const activeCount = computed(() => companies.items.filter(c => c.is_active).length)
const disabledCount = computed(() => companies.items.filter(c => !c.is_active).length)

const visible = computed(() => {
  const q = search.value.toLowerCase()
  return q
    ? companies.items.filter(c => c.name.toLowerCase().includes(q))
    : companies.items
})

const maxEmployees = computed(() =>
  Math.max(1, ...companies.items.map(c => c.employees_count || 0))
)
const maxTasks = computed(() =>
  Math.max(1, ...companies.items.map(c => c.tasks_count || 0))
)

function barWidth(value, max) {
  if (!max) return 0
  return Math.min(100, Math.round(((value || 0) / max) * 100))
}

function fmtDate(s) {
  if (!s) return '—'
  return new Date(s).toLocaleDateString('ru-RU', {
    day: '2-digit', month: '2-digit', year: 'numeric',
  })
}

function initials(name) {
  if (!name) return '?'
  const parts = name.trim().split(/\s+/).slice(0, 2)
  return parts.map(p => p[0]).join('').toUpperCase()
}

/* Тон аватара компании — детерминированно из её id. */
const TONES = ['primary', 'secondary', 'tertiary']
function toneOf(c) {
  return TONES[(c.id || 0) % TONES.length]
}

function onRowClick(e) {
  openEdit(e.data)
}

function openCreate() {
  editTarget.value = null
  formOpen.value = true
}

function openEdit(c) {
  editTarget.value = c
  formOpen.value = true
}

async function onSave({ payload, isEdit, id }) {
  try {
    if (isEdit) {
      await companies.update(id, payload)
      notif.success('Компания обновлена')
    } else {
      await companies.create(payload)
      notif.success('Компания создана')
    }
    formOpen.value = false
  } catch (e) {
    const msg = e?.message?.name?.[0] || e?.message || 'Не удалось сохранить компанию'
    formDlgRef.value?.showError(typeof msg === 'string' ? msg : 'Ошибка сохранения')
  } finally {
    formDlgRef.value?.finish()
  }
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
.page-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
}
.page-head-text { min-width: 0; }
.page-head-title {
  margin: 0 0 6px;
  font-size: 24px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-text);
}
.page-head-meta {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  font-size: 13px;
  color: var(--color-text-dim);
}
.page-head-meta .meta-stat {
  background: var(--color-surface-high);
  color: var(--color-text);
}
.page-head-meta .meta-stat.online {
  background: color-mix(in oklch, var(--color-success) 18%, transparent);
  color: var(--color-text);
}
.page-head-meta .meta-stat.error {
  background: color-mix(in oklch, var(--color-error) 18%, transparent);
  color: var(--color-text);
}

.cmp-search {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 44px;
  padding: 0 10px 0 14px;
  background: var(--color-surface-high);
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  flex: 1 1 320px;
  max-width: 520px;
  min-width: 0;
  transition: border-color .12s, background .12s, box-shadow .12s;
}
.cmp-search:focus-within {
  background: var(--color-surface);
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 18%, transparent);
}
.cmp-search > .material-symbols-outlined { color: var(--color-text-dim); font-size: 20px; }
.cmp-search input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font: inherit;
  min-width: 0;
}
.cmp-search-clear {
  border: none;
  background: var(--color-surface-highest);
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-text-dim);
  transition: background .12s, color .12s;
}
.cmp-search-clear:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.cmp-search-clear .material-symbols-outlined { font-size: 14px; }

/* ============ Ячейки таблицы ============ */
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

/* M3 Expressive toggle. */
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
  height: 34px;
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

/* Кнопки */
.btn-filled {
  appearance: none;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full);
  padding: 10px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  box-shadow: var(--shadow-sm);
  transition: background .14s, box-shadow .14s;
}
.btn-filled:hover { background: var(--color-primary-hover); }
.btn-filled .material-symbols-outlined { font-size: 18px; }

.confirm-warn { color: var(--color-text); }
.confirm-warn strong { color: var(--color-error); }

/* ===== Мобильные карточки компаний ===== */
.cmp-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.cmp-card {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
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
  background: var(--color-surface-high);
  box-shadow: var(--shadow-sm);
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
  border: none;
  background: var(--color-surface-high);
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
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 56px 20px;
  background: var(--color-surface-high);
  border-radius: var(--radius-xl);
  text-align: center;
}
.empty-icon-circle {
  width: 84px;
  height: 84px;
  border-radius: var(--radius-xl);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}
.empty-icon-circle .material-symbols-outlined { font-size: 40px; }
.cmp-cards-empty h3 { margin: 4px 0 0; color: var(--color-text); font-size: 18px; font-weight: 700; }
.cmp-cards-empty p { margin: 0; color: var(--color-text-dim); font-size: 14px; max-width: 320px; }

.state-block { display: grid; place-items: center; padding: 48px; }

@media (max-width: 768px) {
  .hide-narrow { display: none; }
  .desktop-only { display: none; }

  .page-head-title { font-size: 20px; }
  .page-head-meta { font-size: 12px; }
  .meta-dot { display: none; }

  .cmp-search { flex: 1 1 100%; max-width: 100%; }
}
</style>
