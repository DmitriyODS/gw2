<template>
  <div class="cmp-view">
    <header class="cmp-header">
      <div class="cmp-title-row">
        <h1 class="cmp-title">Компании</h1>
        <button class="btn-filled" @click="openCreate">
          <span class="material-symbols-outlined">add</span>
          Новая компания
        </button>
      </div>
      <p class="cmp-subtitle">
        Управление компаниями платформы. Отключённая компания блокирует вход
        всем своим сотрудникам.
      </p>
      <div class="cmp-search-row">
        <div class="cmp-search">
          <span class="material-symbols-outlined">search</span>
          <input v-model.trim="search" placeholder="Поиск по названию или директору" />
          <button v-if="search" class="cmp-search-clear" @click="search = ''" aria-label="Очистить">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
        <div class="cmp-summary">
          <span class="chip">Всего: <strong>{{ companies.items.length }}</strong></span>
          <span class="chip on">Активных: <strong>{{ activeCount }}</strong></span>
          <span v-if="disabledCount" class="chip off">Отключённых: <strong>{{ disabledCount }}</strong></span>
        </div>
      </div>
    </header>

    <div v-if="loading" class="cmp-loading">
      <ProgressSpinner />
    </div>

    <div v-else-if="!visible.length" class="cmp-empty-state">
      <div class="empty-icon">
        <span class="material-symbols-outlined">domain</span>
      </div>
      <h3>{{ search ? 'Ничего не найдено' : 'Компаний пока нет' }}</h3>
      <p v-if="!search">Создайте первую компанию, чтобы начать работу.</p>
      <button v-if="!search" class="btn-filled" @click="openCreate">
        <span class="material-symbols-outlined">add</span> Создать компанию
      </button>
    </div>

    <div v-else class="cmp-table-wrap">
      <table class="cmp-table">
        <thead>
          <tr>
            <th class="th-sort" @click="setSort('name')">
              Название <SortIcon :col="'name'" :sort="sort" />
            </th>
            <th class="th-sort" @click="setSort('created_at')">
              Дата создания <SortIcon :col="'created_at'" :sort="sort" />
            </th>
            <th class="th-sort" @click="setSort('employees_count')">
              Сотрудников <SortIcon :col="'employees_count'" :sort="sort" />
            </th>
            <th class="th-sort" @click="setSort('tasks_count')">
              Задач <SortIcon :col="'tasks_count'" :sort="sort" />
            </th>
            <th>Руководитель</th>
            <th>Статус</th>
            <th class="th-actions"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="c in visible" :key="c.id" :class="{ off: !c.is_active }">
            <td>
              <div class="cmp-name">
                <span class="material-symbols-outlined cmp-icon" :class="{ off: !c.is_active }">
                  domain
                </span>
                <div>
                  <div class="cmp-name-main">{{ c.name }}</div>
                  <div v-if="c.description" class="cmp-name-sub">{{ c.description }}</div>
                </div>
              </div>
            </td>
            <td class="td-mono">{{ fmtDate(c.created_at) }}</td>
            <td class="td-num">{{ c.employees_count }}</td>
            <td class="td-num">{{ c.tasks_count }}</td>
            <td>
              <div v-if="c.director" class="director-cell">
                <span class="director-avatar">{{ initials(c.director.fio) }}</span>
                <span class="director-name">{{ c.director.fio }}</span>
              </div>
              <span v-else class="muted">не назначен</span>
            </td>
            <td>
              <label class="toggle">
                <input
                  type="checkbox"
                  :checked="c.is_active"
                  :disabled="togglingId === c.id"
                  @change="onToggle(c)"
                />
                <span class="toggle-track"></span>
                <span class="toggle-label">{{ c.is_active ? 'Активна' : 'Отключена' }}</span>
              </label>
            </td>
            <td class="td-actions">
              <button class="icon-btn" title="Редактировать" @click="openEdit(c)">
                <span class="material-symbols-outlined">edit</span>
              </button>
              <button class="icon-btn danger" title="Удалить" @click="askDelete(c)">
                <span class="material-symbols-outlined">delete</span>
              </button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <CompanyFormDialog
      ref="formDlgRef"
      v-model="formOpen"
      :company="editTarget"
      @save="onSave"
    />

    <Dialog
      v-model:visible="confirmOpen"
      modal
      :style="{ width: '440px' }"
      :show-header="false"
    >
      <div class="confirm-body">
        <div class="confirm-icon"><span class="material-symbols-outlined">warning</span></div>
        <h3>Удалить компанию «{{ deleteTarget?.name }}»?</h3>
        <p v-if="deleteTarget?.employees_count || deleteTarget?.tasks_count" class="confirm-warn">
          В компании остаются
          <strong v-if="deleteTarget?.employees_count">{{ deleteTarget.employees_count }} сотрудник(а/ов)</strong>
          <template v-if="deleteTarget?.employees_count && deleteTarget?.tasks_count">, </template>
          <strong v-if="deleteTarget?.tasks_count">{{ deleteTarget.tasks_count }} задач(и)</strong>.
          Все данные будут <strong>удалены каскадно</strong>: задачи, юниты, чаты, звонки.
          Восстановить будет нельзя.
        </p>
        <p v-else>Компания пустая — данных не пострадает.</p>
      </div>
      <template #footer>
        <button class="btn-text" :disabled="deleting" @click="confirmOpen = false">Отмена</button>
        <button class="btn-filled danger" :disabled="deleting" @click="doDelete">
          <span v-if="deleting" class="material-symbols-outlined spin">progress_activity</span>
          Удалить
        </button>
      </template>
    </Dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, h } from 'vue'
import Dialog from 'primevue/dialog'
import ProgressSpinner from 'primevue/progressspinner'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import CompanyFormDialog from '@/components/companies/CompanyFormDialog.vue'

const companies = useCompaniesStore()
const notif = useNotificationsStore()

const loading = computed(() => companies.loading && !companies.loaded)
const search = ref('')
const sort = ref({ col: 'created_at', dir: 'desc' })

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
  const filtered = q
    ? companies.items.filter(c =>
        c.name.toLowerCase().includes(q) ||
        (c.director?.fio || '').toLowerCase().includes(q),
      )
    : [...companies.items]
  const { col, dir } = sort.value
  const sign = dir === 'asc' ? 1 : -1
  filtered.sort((a, b) => {
    const va = _val(a, col)
    const vb = _val(b, col)
    if (va == null && vb == null) return 0
    if (va == null) return 1
    if (vb == null) return -1
    if (typeof va === 'string') return sign * va.localeCompare(vb, 'ru')
    if (va instanceof Date && vb instanceof Date) return sign * (va - vb)
    return sign * (va - vb)
  })
  return filtered
})

function _val(c, col) {
  if (col === 'created_at') return c.created_at ? new Date(c.created_at) : null
  if (col === 'name') return c.name || ''
  return c[col] ?? 0
}

function setSort(col) {
  if (sort.value.col === col) {
    sort.value.dir = sort.value.dir === 'asc' ? 'desc' : 'asc'
  } else {
    sort.value = { col, dir: col === 'name' ? 'asc' : 'desc' }
  }
}

function fmtDate(s) {
  if (!s) return '—'
  return new Date(s).toLocaleDateString('ru-RU', { day: '2-digit', month: '2-digit', year: 'numeric' })
}

function initials(fio) {
  if (!fio) return '?'
  const parts = fio.trim().split(/\s+/).slice(0, 2)
  return parts.map(p => p[0]).join('').toUpperCase()
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

// Inline-компонент стрелки сортировки.
const SortIcon = {
  props: ['col', 'sort'],
  setup(p) {
    return () => {
      const active = p.sort.col === p.col
      const ic = active ? (p.sort.dir === 'asc' ? 'arrow_upward' : 'arrow_downward') : 'unfold_more'
      return h('span', { class: ['sort-ic', { active }] }, [
        h('span', { class: 'material-symbols-outlined' }, ic),
      ])
    }
  },
}
</script>

<style scoped>
.cmp-view {
  padding: 24px;
  max-width: 1280px;
  margin: 0 auto;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.cmp-header { display: flex; flex-direction: column; gap: 14px; }

.cmp-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}
.cmp-title { font-size: 28px; font-weight: 700; margin: 0; color: var(--color-on-surface); }
.cmp-subtitle { font-size: 14px; color: var(--color-on-surface-variant); margin: 0; }

.cmp-search-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
}

.cmp-search {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 40px;
  padding: 0 10px 0 12px;
  background: var(--color-surface-container);
  border: 1px solid transparent;
  border-radius: var(--radius-full, 999px);
  flex: 1 1 280px;
  max-width: 420px;
  transition: border-color .12s, background .12s;
}
.cmp-search:focus-within {
  background: var(--color-surface);
  border-color: var(--color-primary);
}
.cmp-search .material-symbols-outlined { color: var(--color-on-surface-variant); font-size: 20px; }
.cmp-search input {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-on-surface);
  font: inherit;
  min-width: 0;
}
.cmp-search-clear {
  border: none;
  background: var(--color-surface-high);
  width: 24px;
  height: 24px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-on-surface-variant);
}
.cmp-search-clear .material-symbols-outlined { font-size: 14px; }

.cmp-summary { display: inline-flex; gap: 6px; }
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: var(--color-surface-container);
  color: var(--color-on-surface-variant);
  border-radius: var(--radius-full, 999px);
  font-size: 12px;
}
.chip strong { color: var(--color-on-surface); }
.chip.on { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.chip.off { background: var(--color-error-container); color: var(--color-on-error-container); }

.cmp-loading { display: grid; place-items: center; min-height: 240px; }

.cmp-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 20px;
  background: var(--color-surface-container);
  border-radius: var(--radius-lg, 16px);
  text-align: center;
}
.empty-icon {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
}
.empty-icon .material-symbols-outlined { font-size: 32px; }
.cmp-empty-state h3 { margin: 0; font-size: 18px; color: var(--color-on-surface); }
.cmp-empty-state p { margin: 0; color: var(--color-on-surface-variant); font-size: 14px; }

.cmp-table-wrap {
  background: var(--color-surface);
  border-radius: var(--radius-lg, 16px);
  border: 1px solid var(--color-outline-variant);
  overflow: hidden;
}

.cmp-table { width: 100%; border-collapse: collapse; }
.cmp-table thead {
  background: var(--color-surface-container);
}
.cmp-table th, .cmp-table td {
  padding: 12px 14px;
  text-align: left;
  font-size: 14px;
  border-bottom: 1px solid var(--color-outline-variant);
}
.cmp-table th {
  font-weight: 600;
  color: var(--color-on-surface-variant);
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  user-select: none;
}
.th-sort { cursor: pointer; }
.th-sort:hover { color: var(--color-on-surface); }
.th-actions { width: 96px; }
.cmp-table tbody tr:last-child td { border-bottom: none; }
.cmp-table tbody tr:hover { background: var(--color-surface-container); }
.cmp-table tr.off td:not(.td-actions) { opacity: .58; }

.cmp-name { display: flex; align-items: center; gap: 10px; min-width: 0; }
.cmp-icon {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: var(--radius-md, 12px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 20px;
  flex: none;
}
.cmp-icon.off { background: var(--color-surface-high); color: var(--color-on-surface-variant); }
.cmp-name-main { font-weight: 600; color: var(--color-on-surface); }
.cmp-name-sub {
  font-size: 12px;
  color: var(--color-on-surface-variant);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 340px;
}

.td-mono { font-variant-numeric: tabular-nums; color: var(--color-on-surface-variant); }
.td-num { font-variant-numeric: tabular-nums; font-weight: 600; }

.director-cell { display: inline-flex; align-items: center; gap: 8px; }
.director-avatar {
  display: grid;
  place-items: center;
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  font-size: 12px;
  font-weight: 700;
}
.director-name { font-size: 13px; }
.muted { color: var(--color-on-surface-variant); font-style: italic; font-size: 13px; }

/* M3 Expressive switch:
   off  → track surface-container-highest + видимая outline-рамка, thumb
          уменьшен и в тон outline (всегда виден на любом фоне);
   on   → track primary, thumb on-primary, увеличивается до полного размера. */
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
  border-radius: 999px;
  background: var(--color-surface-highest, var(--color-surface-high));
  border: 2px solid var(--color-outline, var(--color-outline-variant));
  box-sizing: border-box;
  transition: background .18s, border-color .18s;
}
.toggle-track::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-outline, var(--color-on-surface-variant));
  transform: translateY(-50%);
  transition: transform .2s cubic-bezier(0.4, 0, 0.2, 1),
              background .2s, width .2s, height .2s, left .2s;
}
.toggle input:checked + .toggle-track {
  background: var(--color-primary);
  border-color: var(--color-primary);
}
.toggle input:checked + .toggle-track::after {
  /* при включении: thumb растёт и сдвигается вправо */
  width: 16px;
  height: 16px;
  left: 24px;
  background: var(--color-on-primary);
}
.toggle-label { font-size: 12px; color: var(--color-on-surface-variant); }
.toggle input:checked ~ .toggle-label { color: var(--color-primary); font-weight: 600; }

.td-actions { display: flex; gap: 4px; justify-content: flex-end; }
.icon-btn {
  border: none;
  background: transparent;
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border-radius: 50%;
  color: var(--color-on-surface-variant);
  cursor: pointer;
  transition: background .12s, color .12s;
}
.icon-btn:hover { background: var(--color-surface-container); color: var(--color-on-surface); }
.icon-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.icon-btn .material-symbols-outlined { font-size: 20px; }

.sort-ic { display: inline-flex; vertical-align: middle; opacity: .4; margin-left: 2px; }
.sort-ic.active { opacity: 1; color: var(--color-primary); }
.sort-ic .material-symbols-outlined { font-size: 14px; }

.btn-filled, .btn-text {
  appearance: none;
  border: none;
  cursor: pointer;
  border-radius: var(--radius-full, 999px);
  padding: 10px 18px;
  font: inherit;
  font-weight: 600;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover { filter: brightness(.94); }
.btn-filled.danger { background: var(--color-error); color: var(--color-on-error); }
.btn-text { background: transparent; color: var(--color-on-surface-variant); }
.btn-text:hover { background: var(--color-surface-container); color: var(--color-on-surface); }
.btn-filled:disabled, .btn-text:disabled { opacity: .55; cursor: not-allowed; }

.confirm-body {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 24px 16px 8px;
  text-align: center;
}
.confirm-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  display: grid;
  place-items: center;
}
.confirm-icon .material-symbols-outlined { font-size: 28px; }
.confirm-body h3 { margin: 0; font-size: 18px; color: var(--color-on-surface); }
.confirm-body p { margin: 0; color: var(--color-on-surface-variant); font-size: 14px; line-height: 1.5; }
.confirm-warn { color: var(--color-on-surface); }
.confirm-warn strong { color: var(--color-error); }

.spin { animation: spin 1s linear infinite; font-size: 18px; }
@keyframes spin { to { transform: rotate(360deg); } }

:deep(.p-dialog-footer) { display: flex; justify-content: flex-end; gap: 8px; padding-top: 14px; }

@media (max-width: 900px) {
  .cmp-view { padding: 16px; }
  .cmp-table thead { display: none; }
  .cmp-table, .cmp-table tbody, .cmp-table tr, .cmp-table td { display: block; }
  .cmp-table tr {
    background: var(--color-surface);
    border: 1px solid var(--color-outline-variant);
    border-radius: var(--radius-lg, 16px);
    padding: 4px;
    margin-bottom: 10px;
  }
  .cmp-table tbody tr:hover { background: var(--color-surface); }
  .cmp-table td {
    border: none;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    padding: 6px 12px;
  }
  .cmp-table td::before {
    content: attr(data-label);
    font-size: 12px;
    color: var(--color-on-surface-variant);
  }
  .td-actions { justify-content: flex-end; }
}
</style>
