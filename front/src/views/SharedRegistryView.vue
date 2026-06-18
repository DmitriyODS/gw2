<template>
  <div class="sr">
    <!-- Ошибка доступа -->
    <div v-if="error" class="sr-error">
      <span class="material-symbols-outlined">link_off</span>
      <h2>Ссылка недоступна</h2>
      <p>{{ error }}</p>
    </div>

    <template v-else>
      <header class="sr-head">
        <div class="sr-title">
          <span class="material-symbols-outlined">table</span>
          <h1>{{ registry?.name || 'Реестр' }}</h1>
          <span class="sr-badge">только просмотр</span>
        </div>
        <div class="sr-head-actions">
          <div class="sr-search">
            <span class="material-symbols-outlined">search</span>
            <input v-model="searchInput" type="text" placeholder="Поиск…" @input="onSearch" />
            <button v-if="searchInput" class="sr-search-clear" @click="clearSearch">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
          <div v-if="!isMobile && registry?.fields?.length" class="sr-cols">
            <button ref="colsBtn" class="sr-icon-btn" title="Колонки" @click="toggleCols">
              <span class="material-symbols-outlined">view_column</span>
            </button>
            <teleport to="body">
              <template v-if="colsOpen">
                <div class="sr-cols-backdrop" @click="colsOpen = false" />
                <div class="sr-cols-pop" :style="colsPopStyle">
                  <div class="sr-cols-title">Колонки таблицы</div>
                  <label v-for="f in (registry?.fields || [])" :key="f.id" class="sr-cols-row">
                    <Checkbox :model-value="visibleCols.includes(f.id)" binary @update:model-value="toggleCol(f.id)" />
                    <span>{{ f.label }}</span>
                  </label>
                </div>
              </template>
            </teleport>
          </div>
          <button v-if="exportableFields.length" class="sr-btn" title="Экспорт в XLSX" @click="openExport">
            <span class="material-symbols-outlined">download</span>
            <span class="sr-btn-label">Экспорт</span>
          </button>
        </div>
      </header>

      <div v-if="isMobile && shownFields.length" class="sr-msort">
        <span class="material-symbols-outlined">sort</span>
        <Select
          class="sr-msort-select"
          :model-value="filters.sort"
          :options="sortOptions" option-label="label" option-value="value"
          @update:model-value="mobileSetSort"
        />
        <button class="sr-msort-dir" @click="toggleOrder">
          <span class="material-symbols-outlined">{{ filters.order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}</span>
        </button>
      </div>

      <div v-if="selectedIds.size" class="sr-selbar">
        <span>Выбрано: {{ selectedIds.size }}</span>
        <button class="sr-btn sr-btn-sm" @click="openExport">
          <span class="material-symbols-outlined">download</span> Выгрузить выбранные
        </button>
        <button class="sr-link-btn" @click="clearSelection">Сбросить</button>
      </div>

      <div class="sr-tablebox">
        <div v-if="!isMobile" class="sr-scroll">
          <table class="sr-table">
            <thead>
              <tr>
                <th class="sr-th-check">
                  <Checkbox :model-value="allSelected" binary :disabled="!records.length" @update:model-value="toggleAll" />
                </th>
                <th
                  v-for="f in shownFields"
                  :key="f.id"
                  :class="{ sortable: isSortable(f.type) }"
                  @click="isSortable(f.type) && setSort(String(f.id))"
                >
                  <span class="sr-th-inner">
                    {{ f.label }}
                    <span v-if="filters.sort === String(f.id)" class="material-symbols-outlined sr-sort">
                      {{ filters.order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                    </span>
                  </span>
                </th>
                <th class="sr-th-date sortable" @click="setSort('created_at')">
                  <span class="sr-th-inner">
                    Создано
                    <span v-if="filters.sort === 'created_at'" class="material-symbols-outlined sr-sort">
                      {{ filters.order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                    </span>
                  </span>
                </th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="rec in records" :key="rec.id" class="sr-row" :class="{ selected: selectedIds.has(rec.id) }" @click="openRecord(rec)">
                <td class="sr-td-check" @click.stop>
                  <Checkbox :model-value="selectedIds.has(rec.id)" binary @update:model-value="toggleRow(rec.id)" />
                </td>
                <td v-for="f in shownFields" :key="f.id">
                  <span class="sr-cell">{{ textValue(f, rec.data?.[String(f.id)]) }}</span>
                </td>
                <td class="sr-td-date">{{ shortDate(rec.created_at) }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Мобайл: карточки -->
        <div v-else class="sr-cards">
          <label v-if="records.length" class="sr-cards-selall">
            <Checkbox :model-value="allSelected" binary @update:model-value="toggleAll" />
            <span>Выбрать все на странице</span>
          </label>
          <div
            v-for="rec in records"
            :key="rec.id"
            class="sr-card"
            :class="{ selected: selectedIds.has(rec.id) }"
            @click="openRecord(rec)"
          >
            <div class="sr-card-head">
              <span class="sr-card-check" @click.stop>
                <Checkbox :model-value="selectedIds.has(rec.id)" binary @update:model-value="toggleRow(rec.id)" />
              </span>
              <span class="sr-card-title">{{ cardTitle(rec) }}</span>
              <span class="material-symbols-outlined sr-card-chev">chevron_right</span>
            </div>
            <div v-if="cardBodyFields.length" class="sr-card-body">
              <div v-for="f in cardBodyFields" :key="f.id" class="sr-card-row">
                <span class="sr-card-label">{{ f.label }}</span>
                <span class="sr-card-val">{{ textValue(f, rec.data?.[String(f.id)]) || '—' }}</span>
              </div>
            </div>
          </div>
        </div>

        <div v-if="loading" class="sr-overlay"><span class="material-symbols-outlined spin">progress_activity</span></div>
        <div v-else-if="!records.length" class="sr-empty">
          <span class="material-symbols-outlined">inbox</span>
          <p>{{ searchInput ? 'Ничего не найдено' : 'Записей пока нет' }}</p>
        </div>
      </div>

      <footer class="sr-foot">
        <span class="sr-total">Всего записей: {{ total }}</span>
        <div v-if="totalPages > 1" class="sr-pager">
          <button class="sr-page-btn" :disabled="filters.page <= 1" @click="setPage(filters.page - 1)">
            <span class="material-symbols-outlined">chevron_left</span>
          </button>
          <span class="sr-page-info">{{ filters.page }} / {{ totalPages }}</span>
          <button class="sr-page-btn" :disabled="filters.page >= totalPages" @click="setPage(filters.page + 1)">
            <span class="material-symbols-outlined">chevron_right</span>
          </button>
        </div>
        <span class="sr-brand">Groove Work</span>
      </footer>
    </template>

    <RegistryRecordDialog v-model="dialogOpen" :registry="registry" :record="activeRecord" readonly />

    <!-- Экспорт -->
    <AppDialog
      v-model="exportOpen"
      title="Экспорт в XLSX" icon="download" size="md" :busy="exporting"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Экспортировать', icon: 'download' }]"
      @cancel="exportOpen = false" @confirm="doExport"
    >
      <div class="sr-export">
        <div v-if="selectedIds.size" class="sr-export-scope">
          <label class="sr-radio"><input type="radio" value="all" v-model="exportScope" /> <span>Все записи<template v-if="filters.search"> (по фильтру)</template></span></label>
          <label class="sr-radio"><input type="radio" value="selected" v-model="exportScope" /> <span>Только выбранные ({{ selectedIds.size }})</span></label>
        </div>
        <div class="sr-export-head">
          <span class="sr-export-title">Поля для выгрузки</span>
          <div class="sr-export-bulk">
            <button class="sr-link-btn" @click="selectAllExport">Выбрать всё</button>
            <button class="sr-link-btn" @click="clearAllExport">Снять всё</button>
          </div>
        </div>
        <div class="sr-export-fields">
          <label v-for="f in exportableFields" :key="f.id" class="sr-export-row">
            <Checkbox :model-value="exportFields.has(f.id)" binary @update:model-value="toggleExportField(f.id)" />
            <span class="material-symbols-outlined">{{ fieldIcon(f.type) }}</span>
            <span class="sr-export-name">{{ f.label }}</span>
          </label>
        </div>
      </div>
    </AppDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import Checkbox from 'primevue/checkbox'
import Select from 'primevue/select'
import AppDialog from '@/components/common/AppDialog.vue'
import RegistryRecordDialog from '@/components/registry/RegistryRecordDialog.vue'
import { getSharedRegistry, getSharedRecords, exportSharedRecords } from '@/api/registries.js'
import { fieldIcon, isExportable, isSortable, textValue } from '@/utils/registryFields.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const route = useRoute()
const code = route.params.code
const { isMobile } = useBreakpoint()

const registry = ref(null)
const error = ref(null)
const records = ref([])
const total = ref(0)
const loading = ref(false)
const filters = reactive({ search: '', sort: 'created_at', order: 'desc', page: 1, per_page: 30 })

const dialogOpen = ref(false)
const activeRecord = ref(null)
const selectedIds = ref(new Set())

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filters.per_page)))
const exportableFields = computed(() => (registry.value?.fields || []).filter((f) => isExportable(f.type)))

// ── Видимые столбцы (переключаются, хранятся по коду ссылки) ──
const colsOpen = ref(false)
const colsBtn = ref(null)
const colsPopStyle = ref({})
const visibleCols = ref([])
const shownFields = computed(() => (registry.value?.fields || []).filter((f) => visibleCols.value.includes(f.id)))

const colsKey = () => `gw_shared_cols_${code}`
function loadCols() {
  const fields = registry.value?.fields || []
  try {
    const raw = localStorage.getItem(colsKey())
    if (raw) {
      visibleCols.value = JSON.parse(raw).filter((id) => fields.some((f) => f.id === id))
      return
    }
  } catch { /* ignore */ }
  visibleCols.value = fields.filter((f) => f.show_in_table).map((f) => f.id)
}
function saveCols() {
  try { localStorage.setItem(colsKey(), JSON.stringify(visibleCols.value)) } catch { /* ignore */ }
}
function toggleCol(id) {
  const i = visibleCols.value.indexOf(id)
  if (i === -1) visibleCols.value.push(id)
  else visibleCols.value.splice(i, 1)
  saveCols()
}
function toggleCols() {
  if (colsOpen.value) { colsOpen.value = false; return }
  const r = colsBtn.value?.getBoundingClientRect?.()
  if (r) colsPopStyle.value = { top: `${r.bottom + 6}px`, right: `${Math.max(8, window.innerWidth - r.right)}px` }
  colsOpen.value = true
}

async function load() {
  try {
    registry.value = await getSharedRegistry(code)
    loadCols()
    await fetchRecords()
  } catch (e) {
    error.value = e?.message || 'Ссылка не найдена или была отозвана'
  }
}

let seq = 0
async function fetchRecords() {
  const s = ++seq
  loading.value = true
  try {
    const data = await getSharedRecords(code, { ...filters })
    if (s !== seq) return
    records.value = data.items ?? []
    total.value = data.total ?? records.value.length
  } catch (e) {
    if (s === seq) error.value = e?.message || 'Не удалось загрузить записи'
  } finally {
    if (s === seq) loading.value = false
  }
}

function setSort(key) {
  if (filters.sort === key) filters.order = filters.order === 'asc' ? 'desc' : 'asc'
  else { filters.sort = key; filters.order = 'asc' }
  filters.page = 1
  fetchRecords()
}
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { filters.search = searchInput.value.trim(); filters.page = 1; fetchRecords() }, 300)
}
const searchInput = ref('')
function clearSearch() { searchInput.value = ''; filters.search = ''; filters.page = 1; fetchRecords() }
function setPage(p) { filters.page = p; fetchRecords() }

// Мобильная сортировка контролом + карточки.
const sortOptions = computed(() => {
  const opts = [{ value: 'created_at', label: 'Дате создания' }]
  for (const f of shownFields.value) {
    if (isSortable(f.type)) opts.push({ value: String(f.id), label: f.label })
  }
  return opts
})
function mobileSetSort(value) { filters.sort = value; filters.page = 1; fetchRecords() }
function toggleOrder() { filters.order = filters.order === 'asc' ? 'desc' : 'asc'; fetchRecords() }
function cardTitle(rec) {
  const f = shownFields.value[0]
  const v = f ? textValue(f, rec.data?.[String(f.id)]) : ''
  return v || `Запись #${rec.id}`
}
const cardBodyFields = computed(() => shownFields.value.slice(1))

function openRecord(rec) { activeRecord.value = rec; dialogOpen.value = true }

// Выбор для экспорта.
const allSelected = computed(() => records.value.length > 0 && records.value.every((r) => selectedIds.value.has(r.id)))
function toggleRow(id) { const s = new Set(selectedIds.value); s.has(id) ? s.delete(id) : s.add(id); selectedIds.value = s }
function toggleAll() { selectedIds.value = allSelected.value ? new Set() : new Set(records.value.map((r) => r.id)) }
function clearSelection() { selectedIds.value = new Set() }

// Экспорт.
const exportOpen = ref(false)
const exporting = ref(false)
const exportScope = ref('all')
const exportFields = ref(new Set())
function openExport() {
  exportScope.value = selectedIds.value.size ? 'selected' : 'all'
  exportFields.value = new Set(exportableFields.value.map((f) => f.id))
  exportOpen.value = true
}
function toggleExportField(id) { const s = new Set(exportFields.value); s.has(id) ? s.delete(id) : s.add(id); exportFields.value = s }
function selectAllExport() { exportFields.value = new Set(exportableFields.value.map((f) => f.id)) }
function clearAllExport() { exportFields.value = new Set() }
async function doExport() {
  if (!exportFields.value.size) return
  exporting.value = true
  try {
    const params = { fields: [...exportFields.value] }
    if (exportScope.value === 'selected' && selectedIds.value.size) params.ids = [...selectedIds.value]
    else params.search = filters.search
    const resp = await exportSharedRecords(code, params)
    if (!resp.ok) throw new Error('Не удалось выгрузить')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${registry.value?.name || 'registry'}.xlsx`
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    URL.revokeObjectURL(url)
    exportOpen.value = false
  } catch {
    /* тихо: публичная страница без тостов */
  } finally {
    exporting.value = false
  }
}

function shortDate(v) {
  if (!v) return ''
  const d = new Date(v)
  return isNaN(d.getTime()) ? '' : d.toLocaleDateString('ru-RU')
}

onMounted(load)
</script>

<style scoped>
.sr {
  height: 100%;
  min-height: 100dvh;
  display: flex;
  flex-direction: column;
  background: var(--color-bg);
}

.sr-error {
  flex: 1; min-height: 100dvh;
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 8px; color: var(--color-text-dim); text-align: center; padding: 24px;
}
.sr-error .material-symbols-outlined { font-size: 56px; }
.sr-error h2 { margin: 8px 0 0; color: var(--color-text); }
.sr-error p { margin: 0; }

.sr-head {
  flex: none;
  display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap;
  padding: 16px 20px; border-bottom: 1px solid var(--color-outline-dim); background: var(--color-surface);
}
.sr-title { display: flex; align-items: center; gap: 10px; min-width: 0; }
.sr-title .material-symbols-outlined { color: var(--color-primary); }
.sr-title h1 { margin: 0; font-size: 20px; font-weight: 700; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sr-badge { padding: 3px 10px; border-radius: var(--radius-full); background: var(--color-surface-high); color: var(--color-text-dim); font-size: 12px; font-weight: 600; }
.sr-head-actions { flex: 1; min-width: 0; display: flex; align-items: center; gap: 10px; }
.sr-cols, .sr-btn { flex-shrink: 0; }

.sr-search {
  flex: 1; min-width: 0; /* растягивается на всю ширину, сужается при нехватке места */
  display: flex; align-items: center; gap: 8px; height: 40px; padding: 0 12px;
  background: var(--color-surface-low); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full);
}
.sr-search > .material-symbols-outlined { color: var(--color-text-dim); font-size: 20px; }
.sr-search input { flex: 1; min-width: 0; border: none; background: none; outline: none; color: var(--color-text); font-size: 14px; }
.sr-search-clear { border: none; background: none; cursor: pointer; color: var(--color-text-dim); display: grid; place-items: center; }

.sr-selbar { flex: none; display: flex; align-items: center; gap: 12px; padding: 10px 20px; background: var(--color-primary-container); color: var(--color-on-primary-container); font-size: 14px; font-weight: 600; }

.sr-tablebox { position: relative; flex: 1; min-height: 0; display: flex; padding: 12px 16px; }
.sr-scroll { position: relative; flex: 1; min-height: 0; overflow: auto; background: var(--color-surface); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); }
.sr-table { width: 100%; border-collapse: collapse; font-size: 14px; }
.sr-table thead th {
  position: sticky; top: 0; z-index: 1; background: var(--color-surface);
  border-bottom: 1px solid var(--color-outline-dim); padding: 12px 14px; text-align: left;
  font-weight: 700; color: var(--color-text); white-space: nowrap; user-select: none;
}
.sr-table thead th.sortable { cursor: pointer; }
.sr-table thead th.sortable:hover { color: var(--color-primary); }
.sr-th-inner { display: inline-flex; align-items: center; gap: 4px; }
.sr-sort { font-size: 16px; }
.sr-th-check, .sr-td-check { width: 48px; text-align: center; }
.sr-th-date, .sr-td-date { width: 130px; white-space: nowrap; color: var(--color-text-dim); }
.sr-row { cursor: pointer; }
.sr-row:hover { background: var(--color-surface-high); }
.sr-row.selected { background: var(--color-primary-container); }
.sr-table tbody td { padding: 11px 14px; border-bottom: 1px solid var(--color-outline-dim); color: var(--color-text); }
.sr-td-check { text-align: center; }
.sr-cell { display: block; max-width: 320px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.sr-overlay { position: absolute; inset: 0; display: grid; place-items: center; background: color-mix(in oklch, var(--color-surface) 60%, transparent); }
.sr-empty { position: absolute; inset: 0; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 8px; color: var(--color-text-dim); pointer-events: none; }
.sr-empty .material-symbols-outlined { font-size: 44px; }
.sr-empty p { margin: 0; }

.sr-foot { flex: none; display: flex; align-items: center; gap: 12px; padding: 10px 20px; border-top: 1px solid var(--color-outline-dim); background: var(--color-surface); }
.sr-total { font-size: 13px; color: var(--color-text-dim); }
.sr-pager { display: flex; align-items: center; gap: 8px; margin: 0 auto; }
.sr-page-btn { width: 34px; height: 34px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--color-surface); color: var(--color-text); cursor: pointer; }
.sr-page-btn:hover:not(:disabled) { background: var(--color-surface-high); }
.sr-page-btn:disabled { opacity: 0.4; cursor: default; }
.sr-page-info { font-size: 13px; color: var(--color-text-dim); min-width: 56px; text-align: center; }
.sr-brand { font-size: 12px; font-weight: 700; color: var(--color-text-dim); }

.sr-cols { position: relative; }
.sr-icon-btn { width: 40px; height: 40px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--color-surface); color: var(--color-text-dim); cursor: pointer; }
.sr-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
/* Поповер столбцов вынесен в body (Teleport), позиция — по кнопке. */
.sr-cols-backdrop { position: fixed; inset: 0; z-index: 10800; }
.sr-cols-pop { position: fixed; z-index: 10801; min-width: 220px; max-height: 60vh; overflow-y: auto; padding: 8px; background: var(--color-surface); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); box-shadow: var(--shadow-lg); }
.sr-cols-title { padding: 6px 10px; font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.sr-cols-row { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: var(--radius-md); cursor: pointer; font-size: 14px; color: var(--color-text); }
.sr-cols-row:hover { background: var(--color-surface-high); }

.sr-btn { display: inline-flex; align-items: center; gap: 6px; height: 40px; padding: 0 16px; border: none; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary); font-weight: 600; font-size: 14px; cursor: pointer; }
.sr-btn-sm { height: 32px; padding: 0 12px; }
.sr-link-btn { border: none; background: none; cursor: pointer; color: inherit; font-weight: 600; font-size: 14px; }

.sr-export { display: flex; flex-direction: column; gap: 16px; }
.sr-export-scope { display: flex; flex-direction: column; gap: 8px; padding: 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); }
.sr-radio { display: flex; align-items: center; gap: 10px; font-size: 14px; color: var(--color-text); cursor: pointer; }
.sr-radio input { width: 18px; height: 18px; accent-color: var(--color-primary); }
.sr-export-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.sr-export-title { font-size: 13px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.sr-export-bulk { display: flex; gap: 12px; }
.sr-export-bulk .sr-link-btn { color: var(--color-primary); }
.sr-export-fields { display: flex; flex-direction: column; gap: 2px; max-height: 320px; overflow-y: auto; }
.sr-export-row { display: flex; align-items: center; gap: 10px; padding: 9px 8px; border-radius: var(--radius-md); cursor: pointer; font-size: 14px; color: var(--color-text); }
.sr-export-row:hover { background: var(--color-surface-high); }
.sr-export-row .material-symbols-outlined { font-size: 20px; color: var(--color-text-dim); }
.sr-export-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.spin { animation: srspin 1s linear infinite; font-size: 32px; color: var(--color-primary); }
@keyframes srspin { to { transform: rotate(360deg); } }

/* ── Мобайл: сортировка-контрол и карточки ── */
.sr-msort {
  flex: none; display: flex; align-items: center; gap: 8px;
  padding: 8px 16px; border-bottom: 1px solid var(--color-outline-dim); color: var(--color-text-dim);
}
.sr-msort > .material-symbols-outlined { font-size: 20px; }
.sr-msort-select { flex: 1; min-width: 0; }
.sr-msort-dir {
  width: 38px; height: 38px; flex-shrink: 0; display: grid; place-items: center;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--color-surface); color: var(--color-text); cursor: pointer;
}

.sr-cards { flex: 1; min-height: 0; overflow-y: auto; display: flex; flex-direction: column; gap: 10px; }
.sr-cards-selall { display: flex; align-items: center; gap: 10px; padding: 2px 2px 0; font-size: 13px; color: var(--color-text-dim); }
.sr-card { border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); background: var(--color-surface); overflow: hidden; cursor: pointer; }
.sr-card.selected { border-color: var(--color-primary); background: var(--color-primary-container); }
.sr-card-head { display: flex; align-items: center; gap: 10px; padding: 12px 14px; }
.sr-card-check { flex: none; display: inline-flex; }
.sr-card-title { flex: 1; min-width: 0; font-size: 15px; font-weight: 700; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sr-card-chev { flex: none; color: var(--color-text-dim); }
.sr-card-body { padding: 0 14px 12px; display: flex; flex-direction: column; gap: 6px; }
.sr-card-row { display: flex; gap: 10px; font-size: 14px; }
.sr-card-label { flex: none; width: 40%; max-width: 160px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sr-card-val { flex: 1; min-width: 0; color: var(--color-text); word-break: break-word; }

@media (max-width: 768px) {
  .sr-head { padding: 12px 14px; }
  .sr-title h1 { font-size: 18px; }
  .sr-tablebox { padding: 10px 12px; }
  .sr-cell { max-width: 180px; }
  .sr-btn-label { display: none; }
  .sr-foot { padding: 10px 14px; }
}
</style>
