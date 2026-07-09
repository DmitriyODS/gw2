<template>
  <div class="registries split-view">
    <!-- ЛЕВАЯ ПАНЕЛЬ: список реестров -->
    <aside class="split-side">
      <div class="split-side-head">
        <span class="split-side-tile"><span class="material-symbols-outlined">table_view</span></span>
        <span class="split-side-title">Реестры</span>
      </div>
      <div class="split-side-list">
        <div v-if="store.loadingList" class="split-side-note">Загрузка…</div>
        <div v-else-if="!store.registries.length" class="split-side-note">Реестры отсутствуют</div>
        <button
          v-for="r in store.registries"
          :key="r.id"
          class="split-side-item"
          :class="{ active: r.id === store.selectedId }"
          @click="store.select(r.id)"
        >
          <span class="split-item-tile"><span class="material-symbols-outlined">list_alt</span></span>
          <span class="split-side-name">{{ r.name }}</span>
        </button>
      </div>
    </aside>

    <!-- ПРАВАЯ ПАНЕЛЬ: содержимое выбранного реестра -->
    <section class="split-main">
      <!-- Мобайл: выбор реестра горизонтальной лентой чипов (вместо боковой панели) -->
      <div v-if="isMobile && store.registries.length" class="rg-regstrip">
        <button
          v-for="r in store.registries"
          :key="r.id"
          class="rg-regchip"
          :class="{ active: r.id === store.selectedId }"
          @click="store.select(r.id)"
        >{{ r.name }}</button>
      </div>

      <template v-if="store.selected">
        <!-- Тулбар -->
        <header class="rg-toolbar">
          <h2 class="rg-name" :title="store.selected.name">{{ store.selected.name }}</h2>

          <SearchField
            v-model="searchInput"
            placeholder="Поиск по записям…"
            hotkey
            @update:model-value="onSearch"
            @clear="clearSearch"
          />

          <div class="rg-actions">
            <div v-if="!isMobile && store.selected.fields.length" class="rg-cols">
              <button ref="colsBtn" class="rg-icon-btn" title="Колонки" @click="toggleCols">
                <span class="material-symbols-outlined">view_column</span>
              </button>
              <teleport to="body">
                <template v-if="colsOpen">
                  <div class="rg-cols-backdrop" @click="colsOpen = false" />
                  <div class="rg-cols-pop" :style="colsPopStyle">
                    <div class="rg-cols-title">Колонки таблицы</div>
                    <label v-for="f in store.selected.fields" :key="f.id" class="rg-cols-row">
                      <Checkbox :model-value="visibleCols.includes(f.id)" binary @update:model-value="toggleCol(f.id)" />
                      <span>{{ f.label }}</span>
                    </label>
                  </div>
                </template>
              </teleport>
            </div>
            <button class="rg-icon-btn" title="Внешние ссылки" @click="openShares">
              <span class="material-symbols-outlined">link</span>
            </button>
            <button v-if="store.selected.fields.length" class="rg-icon-btn" title="Экспорт в XLSX" @click="openExport">
              <span class="material-symbols-outlined">download</span>
            </button>
            <button class="btn-grad" @click="openCreate">
              <span class="material-symbols-outlined">add</span>
              <span class="rg-btn-label">Добавить</span>
            </button>
          </div>
        </header>

        <!-- Мобайл: сортировка контролом (на десктопе — клик по заголовку колонки) -->
        <div v-if="isMobile && shownFields.length" class="rg-msort">
          <span class="material-symbols-outlined">sort</span>
          <Select
            class="rg-msort-select"
            :model-value="store.filters.sort"
            :options="sortOptions" option-label="label" option-value="value"
            @update:model-value="mobileSetSort"
          />
          <button class="rg-msort-dir" :title="store.filters.order === 'asc' ? 'По возрастанию' : 'По убыванию'" @click="toggleOrder">
            <span class="material-symbols-outlined">{{ store.filters.order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}</span>
          </button>
        </div>

        <!-- Панель массового выбора -->
        <div v-if="selectedIds.size" class="rg-selbar">
          <span>Выбрано: {{ selectedIds.size }}</span>
          <button class="rg-btn-danger" @click="confirmBulk = true">
            <span class="material-symbols-outlined">delete</span> Удалить выбранные
          </button>
          <button class="rg-btn-text" @click="clearSelection">Сбросить</button>
        </div>

        <!-- Записи: таблица (десктоп) / карточки (мобайл) -->
        <div class="rg-tablebox">
          <div v-if="!isMobile" class="rg-scroll">
            <table class="rg-table">
              <thead>
                <tr>
                  <th class="rg-th-check">
                    <Checkbox :model-value="allSelected" binary :disabled="!store.records.length" @update:model-value="toggleAll" />
                  </th>
                  <th
                    v-for="f in shownFields"
                    :key="f.id"
                    :class="{ sortable: isSortable(f.type) }"
                    @click="isSortable(f.type) && store.setSort(String(f.id))"
                  >
                    <span class="rg-th-inner">
                      {{ f.label }}
                      <span v-if="store.filters.sort === String(f.id)" class="material-symbols-outlined rg-sort">
                        {{ store.filters.order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                      </span>
                    </span>
                  </th>
                  <th class="rg-th-date sortable" @click="store.setSort('created_at')">
                    <span class="rg-th-inner">
                      Создано
                      <span v-if="store.filters.sort === 'created_at'" class="material-symbols-outlined rg-sort">
                        {{ store.filters.order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                      </span>
                    </span>
                  </th>
                </tr>
              </thead>
              <tbody>
                <tr
                  v-for="rec in store.records"
                  :key="rec.id"
                  class="rg-row"
                  :class="{ selected: selectedIds.has(rec.id) }"
                  @click="openRecord(rec)"
                >
                  <td class="rg-td-check" @click.stop>
                    <Checkbox :model-value="selectedIds.has(rec.id)" binary @update:model-value="toggleRow(rec.id)" />
                  </td>
                  <td v-for="f in shownFields" :key="f.id">
                    <span class="rg-cell">{{ textValue(f, rec.data?.[String(f.id)]) }}</span>
                  </td>
                  <td class="rg-td-date">{{ shortDate(rec.created_at) }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Мобайл: карточки записей -->
          <div v-else class="rg-cards">
            <label v-if="store.records.length" class="rg-cards-selall">
              <Checkbox :model-value="allSelected" binary @update:model-value="toggleAll" />
              <span>Выбрать все на странице</span>
            </label>
            <div
              v-for="rec in store.records"
              :key="rec.id"
              class="rg-card"
              :class="{ selected: selectedIds.has(rec.id) }"
              @click="openRecord(rec)"
            >
              <div class="rg-card-head">
                <span class="rg-card-check" @click.stop>
                  <Checkbox :model-value="selectedIds.has(rec.id)" binary @update:model-value="toggleRow(rec.id)" />
                </span>
                <span class="rg-card-title">{{ cardTitle(rec) }}</span>
                <span class="material-symbols-outlined rg-card-chev">chevron_right</span>
              </div>
              <div v-if="cardBodyFields.length" class="rg-card-body">
                <div v-for="f in cardBodyFields" :key="f.id" class="rg-card-row">
                  <span class="rg-card-label">{{ f.label }}</span>
                  <span class="rg-card-val">{{ textValue(f, rec.data?.[String(f.id)]) || '—' }}</span>
                </div>
              </div>
            </div>
          </div>

          <div v-if="store.loadingRecords" class="rg-overlay">
            <span class="material-symbols-outlined spin">progress_activity</span>
          </div>
          <div v-else-if="!store.records.length" class="rg-empty">
            <span class="material-symbols-outlined">inbox</span>
            <p>{{ searchInput ? 'Ничего не найдено' : 'Записей пока нет' }}</p>
          </div>
        </div>

        <!-- Футер: счётчик + пагинация -->
        <footer class="rg-foot">
          <span class="rg-total">Всего записей: {{ store.total }}</span>
          <div v-if="totalPages > 1" class="rg-pager">
            <button class="rg-page-btn" :disabled="store.filters.page <= 1" @click="store.setPage(store.filters.page - 1)">
              <span class="material-symbols-outlined">chevron_left</span>
            </button>
            <span class="rg-page-info">{{ store.filters.page }} / {{ totalPages }}</span>
            <button class="rg-page-btn" :disabled="store.filters.page >= totalPages" @click="store.setPage(store.filters.page + 1)">
              <span class="material-symbols-outlined">chevron_right</span>
            </button>
          </div>
        </footer>
      </template>

      <!-- Реестр не выбран -->
      <EmptyState
        v-else
        class="split-empty"
        icon="table_view"
        tone="soft"
        :title="isMobile ? 'Выберите реестр сверху' : 'Выберите реестр слева'"
        subtitle="Выберите реестр в списке, чтобы просмотреть его данные"
      />
    </section>

    <RegistryRecordDialog v-model="dialogOpen" :registry="store.selected" :record="activeRecord" />
    <ConfirmDialog
      :visible="confirmBulk"
      header="Удалить выбранные записи?"
      :message="`Будет удалено записей: ${selectedIds.size}. Действие необратимо.`"
      confirm-label="Удалить" danger-confirm
      @confirm="doBulkDelete" @cancel="confirmBulk = false"
    />

    <!-- Внешние ссылки -->
    <AppDialog
      v-model="sharesOpen"
      title="Внешние ссылки" icon="link" size="md"
      :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
      @cancel="sharesOpen = false"
    >
      <div class="rg-shares">
        <p class="rg-shares-note">
          По внешней ссылке любой человек (без входа в систему) сможет просматривать таблицу
          этого реестра, открывать карточки и выгружать данные — но не редактировать.
          Ссылку можно отозвать в любой момент.
        </p>
        <button class="btn-grad" :disabled="sharesBusy" @click="createShareLink">
          <span class="material-symbols-outlined">add_link</span> Создать ссылку
        </button>

        <div v-if="sharesLoading" class="rg-shares-empty">Загрузка…</div>
        <div v-else-if="!shares.length" class="rg-shares-empty">Ссылок пока нет</div>
        <ul v-else class="rg-shares-list">
          <li v-for="s in shares" :key="s.id" class="rg-share">
            <input class="rg-share-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
            <button class="rg-icon-btn sm" title="Копировать" @click="copyShare(s.code)">
              <span class="material-symbols-outlined">content_copy</span>
            </button>
            <a class="rg-icon-btn sm" :href="shareUrl(s.code)" target="_blank" rel="noopener" title="Открыть">
              <span class="material-symbols-outlined">open_in_new</span>
            </a>
            <button class="rg-icon-btn sm danger" title="Отозвать" @click="revokeShareLink(s.id)">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </li>
        </ul>
      </div>
    </AppDialog>

    <!-- Экспорт в XLSX -->
    <AppDialog
      v-model="exportOpen"
      title="Экспорт в XLSX" icon="download" size="md" :busy="exporting"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Экспортировать', icon: 'download' }]"
      @cancel="exportOpen = false" @confirm="doExport"
    >
      <div class="rg-export">
        <div v-if="selectedIds.size" class="rg-export-scope">
          <label class="rg-radio">
            <input type="radio" value="all" v-model="exportScope" />
            <span>Все записи<template v-if="store.filters.search"> (по фильтру поиска)</template></span>
          </label>
          <label class="rg-radio">
            <input type="radio" value="selected" v-model="exportScope" />
            <span>Только выбранные ({{ selectedIds.size }})</span>
          </label>
        </div>

        <div class="rg-export-head">
          <span class="rg-export-title">Поля для выгрузки</span>
          <div class="rg-export-bulk">
            <button class="rg-btn-text" @click="selectAllExport">Выбрать всё</button>
            <button class="rg-btn-text" @click="clearAllExport">Снять всё</button>
          </div>
        </div>

        <div class="rg-export-fields">
          <label v-for="f in exportableFields" :key="f.id" class="rg-export-row">
            <Checkbox :model-value="exportFields.has(f.id)" binary @update:model-value="toggleExportField(f.id)" />
            <span class="material-symbols-outlined">{{ fieldIcon(f.type) }}</span>
            <span class="rg-export-name">{{ f.label }}</span>
          </label>
          <p v-if="!exportableFields.length" class="rg-export-empty">
            В этом реестре нет полей, доступных для экспорта (картинки и файлы не выгружаются).
          </p>
        </div>
      </div>
    </AppDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import Checkbox from 'primevue/checkbox'
import Select from 'primevue/select'
import RegistryRecordDialog from '@/components/registry/RegistryRecordDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import { useRegistriesStore } from '@/stores/registries.js'
import { exportRecords, getShares, createShare, revokeShare } from '@/api/registries.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { fieldIcon, isExportable, isSortable, textValue } from '@/utils/registryFields.js'

const store = useRegistriesStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()

// ── Мобильная сортировка и карточки ──
const sortOptions = computed(() => {
  const opts = [{ value: 'created_at', label: 'Дате создания' }]
  for (const f of shownFields.value) {
    if (isSortable(f.type)) opts.push({ value: String(f.id), label: f.label })
  }
  return opts
})
function mobileSetSort(value) {
  store.filters.sort = value
  store.filters.page = 1
  store.fetchRecords()
}
function toggleOrder() {
  store.filters.order = store.filters.order === 'asc' ? 'desc' : 'asc'
  store.fetchRecords()
}
// На карточке первое видимое поле — заголовок, остальные — тело.
function cardTitle(rec) {
  const f = shownFields.value[0]
  const v = f ? textValue(f, rec.data?.[String(f.id)]) : ''
  return v || `Запись #${rec.id}`
}
const cardBodyFields = computed(() => shownFields.value.slice(1))

const searchInput = ref('')
const colsOpen = ref(false)
const colsBtn = ref(null)
const colsPopStyle = ref({})

// Поповер «Колонки» вынесен в body (Teleport) и позиционируется под кнопкой —
// так его не обрезает overflow:hidden и не перекрывает таблица.
function toggleCols() {
  if (colsOpen.value) { colsOpen.value = false; return }
  const r = colsBtn.value?.getBoundingClientRect?.()
  if (r) {
    colsPopStyle.value = {
      top: `${r.bottom + 6}px`,
      right: `${Math.max(8, window.innerWidth - r.right)}px`,
    }
  }
  colsOpen.value = true
}
const dialogOpen = ref(false)
const activeRecord = ref(null)
const confirmBulk = ref(false)
const selectedIds = ref(new Set())

const totalPages = computed(() => Math.max(1, Math.ceil(store.total / store.filters.per_page)))

// ── Видимые колонки (per-реестр, localStorage; дефолт — поля с show_in_table) ──
const visibleCols = ref([])
const colsKey = (id) => `gw_registry_cols_${id}`
function loadCols(reg) {
  if (!reg) { visibleCols.value = []; return }
  const fields = reg.fields || []
  try {
    const raw = localStorage.getItem(colsKey(reg.id))
    if (raw) {
      visibleCols.value = JSON.parse(raw).filter((id) => fields.some((f) => f.id === id))
      return
    }
  } catch { /* ignore */ }
  visibleCols.value = fields.filter((f) => f.show_in_table).map((f) => f.id)
}
function saveCols() {
  if (store.selected) {
    try { localStorage.setItem(colsKey(store.selected.id), JSON.stringify(visibleCols.value)) } catch { /* ignore */ }
  }
}
function toggleCol(id) {
  const i = visibleCols.value.indexOf(id)
  if (i === -1) visibleCols.value.push(id)
  else visibleCols.value.splice(i, 1)
  saveCols()
}
const shownFields = computed(() => (store.selected?.fields || []).filter((f) => visibleCols.value.includes(f.id)))

watch(() => store.selectedId, () => {
  searchInput.value = ''
  clearSelection()
  colsOpen.value = false
  loadCols(store.selected)
})
watch(() => store.selected?.fields, () => loadCols(store.selected))

// ── Выбор записей ──
const allSelected = computed(() =>
  store.records.length > 0 && store.records.every((r) => selectedIds.value.has(r.id)),
)
function toggleRow(id) {
  const s = new Set(selectedIds.value)
  s.has(id) ? s.delete(id) : s.add(id)
  selectedIds.value = s
}
function toggleAll() {
  selectedIds.value = allSelected.value ? new Set() : new Set(store.records.map((r) => r.id))
}
function clearSelection() { selectedIds.value = new Set() }

// Выбор не должен пережить смену страницы/обновление списка.
watch(() => store.records, () => {
  const ids = new Set(store.records.map((r) => r.id))
  const next = new Set([...selectedIds.value].filter((id) => ids.has(id)))
  if (next.size !== selectedIds.value.size) selectedIds.value = next
})

// ── Поиск / пагинация ──
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(searchInput.value.trim()), 300)
}
function clearSearch() { clearTimeout(searchTimer); searchInput.value = ''; store.setSearch('') }

// ── Диалог записи ──
function openRecord(rec) { activeRecord.value = rec; dialogOpen.value = true }
function openCreate() { activeRecord.value = null; dialogOpen.value = true }

async function doBulkDelete() {
  confirmBulk.value = false
  const ids = [...selectedIds.value]
  clearSelection()
  await store.bulkDelete(ids)
}

// ── Внешние ссылки ──
const sharesOpen = ref(false)
const shares = ref([])
const sharesLoading = ref(false)
const sharesBusy = ref(false)

function shareUrl(code) { return `${location.origin}/registry/${code}` }

async function openShares() {
  sharesOpen.value = true
  sharesLoading.value = true
  try {
    const d = await getShares(store.selectedId)
    shares.value = d.shares ?? []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить ссылки')
  } finally {
    sharesLoading.value = false
  }
}
async function createShareLink() {
  sharesBusy.value = true
  try {
    const s = await createShare(store.selectedId)
    shares.value.unshift(s)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать ссылку')
  } finally {
    sharesBusy.value = false
  }
}
async function revokeShareLink(id) {
  try {
    await revokeShare(store.selectedId, id)
    shares.value = shares.value.filter((s) => s.id !== id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось отозвать ссылку')
  }
}
async function copyShare(code) {
  try {
    await navigator.clipboard.writeText(shareUrl(code))
    notif.success('Ссылка скопирована')
  } catch { /* ignore */ }
}

// ── Экспорт в XLSX ──
const exportOpen = ref(false)
const exporting = ref(false)
const exportScope = ref('all') // 'all' (по фильтру) | 'selected'
const exportFields = ref(new Set())
const exportableFields = computed(() => (store.selected?.fields || []).filter((f) => isExportable(f.type)))

function openExport() {
  exportScope.value = selectedIds.value.size ? 'selected' : 'all'
  exportFields.value = new Set(exportableFields.value.map((f) => f.id))
  exportOpen.value = true
}
function toggleExportField(id) {
  const s = new Set(exportFields.value)
  s.has(id) ? s.delete(id) : s.add(id)
  exportFields.value = s
}
function selectAllExport() { exportFields.value = new Set(exportableFields.value.map((f) => f.id)) }
function clearAllExport() { exportFields.value = new Set() }

async function doExport() {
  if (!exportFields.value.size) { notif.error('Выберите хотя бы одно поле'); return }
  exporting.value = true
  try {
    const params = { fields: [...exportFields.value] }
    if (exportScope.value === 'selected' && selectedIds.value.size) params.ids = [...selectedIds.value]
    else params.search = store.filters.search
    const resp = await exportRecords(store.selectedId, params)
    if (!resp.ok) {
      let msg = 'Не удалось выгрузить'
      try { msg = (await resp.json()).message || msg } catch { /* ignore */ }
      throw new Error(msg)
    }
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${store.selected?.name || 'registry'}.xlsx`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    exportOpen.value = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось выгрузить')
  } finally {
    exporting.value = false
  }
}

function shortDate(v) {
  if (!v) return ''
  const d = new Date(v)
  return isNaN(d.getTime()) ? '' : d.toLocaleDateString('ru-RU')
}

onMounted(() => store.fetchRegistries())
</script>

<style scoped>
/* Каркас (стеклянные панели, раскладка, мобильное скрытие левой панели) —
   глобальный паттерн .split-* (main.css). Здесь — только внутренности
   правой панели. */

.rg-toolbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--color-outline-dim);
}
.rg-name {
  flex-shrink: 0;
  margin: 0;
  font-size: 18px;
  font-weight: 700;
  color: var(--color-text);
  max-width: 220px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.rg-actions { flex-shrink: 0; display: flex; align-items: center; gap: 8px; }

.rg-cols { position: relative; }
.rg-icon-btn {
  width: 40px; height: 40px;
  display: grid; place-items: center;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  color: var(--color-text-dim);
  cursor: pointer;
}
.rg-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.rg-icon-btn.sm { width: 34px; height: 34px; flex-shrink: 0; }
.rg-icon-btn.sm .material-symbols-outlined { font-size: 18px; }
.rg-icon-btn.danger { color: var(--color-error); }

/* ── Модалка внешних ссылок ── */
.rg-shares { display: flex; flex-direction: column; gap: 14px; }
.rg-shares-note { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }
.rg-shares-empty { padding: 16px; text-align: center; color: var(--color-text-dim); font-size: 14px; }
.rg-shares-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.rg-share { display: flex; align-items: center; gap: 6px; }
.rg-share-url {
  flex: 1; min-width: 0; height: 38px; padding: 0 12px;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--color-surface-low); color: var(--color-text); font-size: 13px;
}

/* ── Модалка экспорта ── */
.rg-export { display: flex; flex-direction: column; gap: 16px; }
.rg-export-scope { display: flex; flex-direction: column; gap: 8px; padding: 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--color-surface-low); }
.rg-radio { display: flex; align-items: center; gap: 10px; font-size: 14px; color: var(--color-text); cursor: pointer; }
.rg-radio input { width: 18px; height: 18px; accent-color: var(--color-primary); }
.rg-export-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.rg-export-title { font-size: 13px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.rg-export-bulk { display: flex; gap: 12px; }
.rg-export-bulk .rg-btn-text { color: var(--color-primary); }
.rg-export-fields { display: flex; flex-direction: column; gap: 2px; max-height: 320px; overflow-y: auto; }
.rg-export-row { display: flex; align-items: center; gap: 10px; padding: 9px 8px; border-radius: var(--radius-md); cursor: pointer; font-size: 14px; color: var(--color-text); }
.rg-export-row:hover { background: var(--color-surface-high); }
.rg-export-row .material-symbols-outlined { font-size: 20px; color: var(--color-text-dim); }
.rg-export-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rg-export-empty { margin: 0; color: var(--color-text-dim); font-size: 14px; }

/* Поповер вынесен в body (Teleport); позиция задаётся inline по кнопке. */
.rg-cols-backdrop { position: fixed; inset: 0; z-index: 10800; }
.rg-cols-pop {
  position: fixed;
  z-index: 10801;
  min-width: 220px;
  max-height: 60vh;
  overflow-y: auto;
  padding: 8px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
}
.rg-cols-title { padding: 6px 10px; font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.rg-cols-row { display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: var(--radius-md); cursor: pointer; font-size: 14px; }
.rg-cols-row:hover { background: var(--color-surface-high); }

.rg-selbar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 16px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 14px;
  font-weight: 600;
}

/* ── Таблица: собственный скролл, sticky-шапка ── */
.rg-tablebox { position: relative; flex: 1; min-height: 0; display: flex; }
.rg-scroll {
  position: relative;
  flex: 1;
  min-height: 0;
  overflow: auto;
}
.rg-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}
.rg-table thead th {
  /* Sticky-шапка таблицы: строки уезжают под неё — плотное стекло с блюром */
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border-bottom: 1px solid var(--color-outline-dim);
  padding: 12px 14px;
  text-align: left;
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
  user-select: none;
}
.rg-table thead th.sortable { cursor: pointer; }
.rg-table thead th.sortable:hover { color: var(--color-primary); }
.rg-th-inner { display: inline-flex; align-items: center; gap: 4px; }
.rg-sort { font-size: 16px; }
.rg-th-check, .rg-td-check { width: 48px; text-align: center; padding-left: 16px; padding-right: 0; }
.rg-th-date, .rg-td-date { width: 130px; white-space: nowrap; color: var(--color-text-dim); }

.rg-row { cursor: pointer; }
.rg-row:hover { background: var(--color-surface-high); }
.rg-row.selected { background: var(--color-primary-container); }
.rg-table tbody td {
  padding: 11px 14px;
  border-bottom: 1px solid var(--color-outline-dim);
  color: var(--color-text);
}
.rg-td-check { text-align: center; }
.rg-cell {
  display: block;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rg-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
}
.rg-empty {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-dim);
  pointer-events: none;
}
.rg-empty .material-symbols-outlined { font-size: 44px; }
.rg-empty p { margin: 0; }

/* ── Футер ── */
.rg-foot {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border-top: 1px solid var(--color-outline-dim);
}
.rg-total { font-size: 13px; color: var(--color-text-dim); }
.rg-pager { display: flex; align-items: center; gap: 8px; }
.rg-page-btn {
  width: 34px; height: 34px;
  display: grid; place-items: center;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  cursor: pointer;
}
.rg-page-btn:hover:not(:disabled) { background: var(--color-surface-high); }
.rg-page-btn:disabled { opacity: 0.4; cursor: default; }
.rg-page-info { font-size: 13px; color: var(--color-text-dim); min-width: 56px; text-align: center; }

/* ── Кнопки ── */
.rg-btn-danger {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 16px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-error);
  color: var(--color-on-error);
  font-weight: 600;
  font-size: 14px;
  cursor: pointer;
}
.rg-btn-text { border: none; background: none; cursor: pointer; color: inherit; font-weight: 600; font-size: 14px; }

.spin { animation: rgspin 1s linear infinite; font-size: 32px; color: var(--color-primary); }
@keyframes rgspin { to { transform: rotate(360deg); } }

/* ── Мобильная раскладка ── */
/* ── Мобайл: лента реестров, сортировка, карточки ── */
.rg-regstrip {
  flex: none; display: flex; gap: 8px; padding: 10px 12px; overflow-x: auto;
  border-bottom: 1px solid var(--color-outline-dim); -webkit-overflow-scrolling: touch;
}
.rg-regchip {
  flex: none; padding: 8px 14px; border-radius: var(--radius-full);
  border: 1px solid var(--color-outline-dim); background: var(--acrylic-card-bg);
  color: var(--color-text-dim); font-size: 14px; font-weight: 600; cursor: pointer; white-space: nowrap;
}
.rg-regchip.active { background: var(--color-primary); color: var(--color-on-primary); border-color: transparent; }

.rg-msort {
  flex: none; display: flex; align-items: center; gap: 8px;
  padding: 8px 16px; border-bottom: 1px solid var(--color-outline-dim); color: var(--color-text-dim);
}
.rg-msort > .material-symbols-outlined { font-size: 20px; }
.rg-msort-select { flex: 1; min-width: 0; }
.rg-msort-dir {
  width: 38px; height: 38px; flex-shrink: 0; display: grid; place-items: center;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--acrylic-card-bg); color: var(--color-text); cursor: pointer;
}

.rg-cards { flex: 1; min-height: 0; overflow-y: auto; padding: 12px; display: flex; flex-direction: column; gap: 10px; }
.rg-cards-selall { display: flex; align-items: center; gap: 10px; padding: 4px 4px 0; font-size: 13px; color: var(--color-text-dim); }
.rg-card {
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg); overflow: hidden; cursor: pointer;
}
.rg-card.selected { border-color: var(--color-primary); background: var(--color-primary-container); }
.rg-card-head { display: flex; align-items: center; gap: 10px; padding: 12px 14px; }
.rg-card-check { flex: none; display: inline-flex; }
.rg-card-title { flex: 1; min-width: 0; font-size: 15px; font-weight: 700; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rg-card-chev { flex: none; color: var(--color-text-dim); }
.rg-card-body { padding: 0 14px 12px; display: flex; flex-direction: column; gap: 6px; }
.rg-card-row { display: flex; gap: 10px; font-size: 14px; }
.rg-card-label { flex: none; width: 40%; max-width: 160px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.rg-card-val { flex: 1; min-width: 0; color: var(--color-text); word-break: break-word; }

@media (max-width: 768px) {
  /* Скрытие левой панели и разворот правой — в глобальном .split-* */
  .rg-name { display: none; }
  .rg-toolbar { flex-wrap: wrap; gap: 8px; padding: 10px 12px; }
  .rg-toolbar :deep(.search-field) { order: 2; flex-basis: 100%; }
  .rg-actions { order: 1; margin-left: auto; }
  /* Футер (итого/пагинация) всегда под списком — резерв под нижнюю
     навигацию (64px) + 12px воздуха вешаем ему, иначе fixed-навигация
     его накрывает; сам список карточек упирается в футер. */
  .rg-foot {
    padding: 10px 12px;
    padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px));
  }
}
</style>
