<template>
  <AppListDetail
    v-model:open="detailOpen"
    :loading="store.loadingList && !store.registries.length"
    @narrow-change="narrow = $event"
  >
    <template #list="{ toggle }">
      <RegistryList
        :registries="store.registries"
        :selected-id="store.selectedId"
        :scope="store.scope"
        :renaming-id="renamingId"
        :narrow="narrow"
        @select="selectRegistry"
        @update:scope="store.setScope"
        @create="openCreate"
        @context="onRegistryContext"
        @rename="applyRename"
        @rename-cancel="renamingId = null"
        @toggle="toggle"
      />
    </template>

    <template #detail="{ collapsed, toggle }">
      <AppPage
        embedded
        :title="store.selected?.name || ''"
        :back="narrow"
        back-label="К реестрам"
        :menu="!narrow && collapsed"
        menu-icon="left_panel_open"
        menu-label="Показать список"
        :commands="commands"
        flush
        :scroll="false"
        @back="detailOpen = false"
        @menu="toggle"
        @command="onCommand"
      >
        <!-- Поиск идёт слотом шапки: в тесной панели он сам сворачивается в
             лупу при названии и не отнимает у таблицы целую строку. -->
        <template v-if="store.selected" #search="{ narrow: tight }">
          <SearchField
            v-model="searchInput"
            placeholder="Поиск по записям…"
            hotkey
            :collapsed="tight"
            @update:model-value="onSearch"
            @clear="clearSearch"
          />
        </template>

        <template v-if="store.selected && store.records.length" #footer>
          <span class="rg-total">Всего записей: {{ store.total }}</span>
          <div v-if="totalPages > 1" class="rg-pager">
            <AppButton
              variant="icon" size="sm" icon="chevron_left"
              aria-label="Предыдущая страница"
              :disabled="store.filters.page <= 1"
              @click="store.setPage(store.filters.page - 1)"
            />
            <span class="rg-page-info">{{ store.filters.page }} / {{ totalPages }}</span>
            <AppButton
              variant="icon" size="sm" icon="chevron_right"
              aria-label="Следующая страница"
              :disabled="store.filters.page >= totalPages"
              @click="store.setPage(store.filters.page + 1)"
            />
          </div>
        </template>

        <RegistryRecords
          v-if="store.selected"
          :registry="store.selected"
          :fields="shownFields"
          :records="store.records"
          :loading="store.loadingRecords"
          :sort="store.filters.sort"
          :order="store.filters.order"
          :is-selected="isSelected"
          :all-selected="allSelected"
          :selection-count="selectionCount"
          :selection-all="selectionMode === 'all'"
          :total="store.total"
          :narrow="narrow"
          :view="viewMode"
          :search="store.filters.search"
          :sections="store.sections"
          :section="store.filters.section"
          :column-filters="store.filters.filters"
          :widths="colWidths"
          :can-edit="store.canEdit"
          empty-hint="Добавьте первую запись — она появится в таблице."
          @update:sort="store.applySort"
          @update:section="store.setSection"
          @update:column-filter="store.setColumnFilter"
          @move-column="moveCol"
          @resize-columns="setColWidths"
          @reset-widths="resetColWidths"
          @open="openRecord"
          @edit="editRecord"
          @manage="openIssue"
          @remove="askDeleteRecord"
          @toggle="toggleRow"
          @toggle-all="toggleAll"
          @select-all-matching="selectAllMatching"
          @clear-selection="clearSelection"
        />

        <EmptyState
          v-else
          icon="table_view"
          tone="soft"
          title="Выберите реестр слева"
          subtitle="Выберите реестр в списке, чтобы просмотреть его данные"
        />
      </AppPage>
    </template>

    <!-- Контекстное меню реестра: переименовать, поделиться (подменю), удалить.
         Подменю ContextMenu само разворачивается в ту сторону, где есть место, —
         у краёв экрана и в узкой панели оно не обрезается. -->
    <ContextMenu
      :visible="menuOpen"
      :x="menuX"
      :y="menuY"
      :items="menuItems"
      @select="onMenuSelect"
      @close="menuOpen = false"
    />

    <RegistryStructureDialog
      v-model="structureOpen"
      :registry="structureTarget"
      :save="saveStructure"
      @error="notif.error($event)"
    />

    <RegistryRecordDialog
      v-model="dialogOpen"
      :registry="store.selected"
      :record="activeRecord"
      :readonly="!store.canEdit"
      :start-editing="startEditing"
      :save="saveRecord"
      :upload="uploadRecordFile"
      @manage="openIssueFromCard"
    />

    <RegistryIssueDialog
      v-model="issueOpen"
      :registry-id="store.selectedId"
      :record="issueRecord"
      :title="issueTitle"
      :issue="store.issueRecord"
      :extend="store.extendIssue"
      :back="store.returnIssue"
      :history="fetchIssues"
      @error="notif.error($event)"
    />

    <RegistryShareLinkDialog
      v-model="shareLinkOpen"
      :registry-id="shareTargetId"
      @error="notif.error($event)"
      @copied="notif.success('Ссылка скопирована')"
    />
    <RegistryShareUsersDialog
      v-model="shareUsersOpen"
      :registry-id="shareTargetId"
      @error="notif.error($event)"
      @changed="store.fetchRegistries()"
    />
    <RegistryShareUsersDialog
      v-model="shareCompanyOpen"
      :registry-id="shareTargetId"
      company
      @error="notif.error($event)"
      @changed="store.fetchRegistries()"
    />

    <RegistryQrFindDialog
      v-model="qrFindOpen"
      :registry="store.selected"
      :fetch-page="fetchRecordsPage"
      @found="openRecord"
    />
    <RegistryQrPrintDialog
      v-model="qrPrintOpen"
      :registry="store.selected"
      :fetch-page="fetchRecordsPage"
      :selected-ids="pickedIds"
      :search="store.filters.search"
      :section="store.filters.section"
      :filters="store.filters.filters"
      :sort="store.filters.sort"
      :order="store.filters.order"
    />
    <RegistryExportDialog
      v-model="exportOpen"
      :fields="exportableFields"
      :records="store.records"
      :accounting="!!store.selected?.accounting"
      :selected-ids="pickedIds"
      :selection="selection"
      :filter="store.exportFilter()"
      :filename="store.selected?.name || 'registry'"
      :request="exportRequest"
      @error="notif.error($event)"
    />
    <RegistryColumnsDialog
      v-model="colsOpen"
      :fields="store.selected?.fields || []"
      :visible="visibleCols"
      @toggle="toggleCol"
    />

    <ConfirmDialog
      :visible="confirmBulk"
      header="Удалить выбранные записи?"
      :message="`Будет удалено записей: ${selectionCount}. Действие необратимо.`"
      confirm-label="Удалить" danger-confirm
      @confirm="doBulkDelete" @cancel="confirmBulk = false"
    />
    <ConfirmDialog
      :visible="!!recordToDelete"
      header="Удалить запись?"
      message="Запись и её файлы будут удалены безвозвратно."
      confirm-label="Удалить" danger-confirm
      @confirm="doDeleteRecord" @cancel="recordToDelete = null"
    />
    <ConfirmDialog
      :visible="!!registryToDelete"
      header="Удалить реестр?"
      :message="`«${registryToDelete?.name || ''}» удалится вместе со всеми записями и файлами. Действие необратимо.`"
      confirm-label="Удалить" danger-confirm
      @confirm="doDeleteRegistry" @cancel="registryToDelete = null"
    />
  </AppListDetail>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import RegistryColumnsDialog from '@/components/registry/RegistryColumnsDialog.vue'
import RegistryExportDialog from '@/components/registry/RegistryExportDialog.vue'
import RegistryIssueDialog from '@/components/registry/RegistryIssueDialog.vue'
import RegistryList from '@/components/registry/RegistryList.vue'
import RegistryQrFindDialog from '@/components/registry/RegistryQrFindDialog.vue'
import RegistryQrPrintDialog from '@/components/registry/RegistryQrPrintDialog.vue'
import RegistryRecordDialog from '@/components/registry/RegistryRecordDialog.vue'
import RegistryRecords from '@/components/registry/RegistryRecords.vue'
import RegistryShareLinkDialog from '@/components/registry/RegistryShareLinkDialog.vue'
import RegistryShareUsersDialog from '@/components/registry/RegistryShareUsersDialog.vue'
import RegistryStructureDialog from '@/components/registry/RegistryStructureDialog.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import { useRegistryColumns } from '@/composables/useRegistryColumns.js'
import { useRowSelection } from '@/composables/useRowSelection.js'
import { useRegistriesStore } from '@/stores/registries.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { exportRecords, getIssues, getRecords, uploadFile } from '@/api/registries.js'
import { hasQr, isExportable, textValue } from '@/utils/registryFields.js'

const store = useRegistriesStore()
const route = useRoute()
const authStore = useAuthStore()
const notif = useNotificationsStore()

/* Узкая раскладка — свойство САМОГО раздела (он живёт окном рабочего стола),
   поэтому её сообщает AppListDetail, а не медиазапрос по ширине экрана. */
const narrow = ref(false)
const detailOpen = ref(false)

/* Смена активной компании меняет не сами реестры (они личные), а компанийные
   шары: в другой компании открыт другой набор. */
watch(() => authStore.companyId, (id, prev) => {
  if (id !== prev) store.reloadForCompany()
})

function selectRegistry(id) {
  store.select(id)
  detailOpen.value = true
}

// ── Команды шапки ──
/* «О разделе» здесь НЕТ намеренно: карточка раздела открывается значком в
   заголовке окна (AppWindow) — она нужна всегда, а не только при выбранном
   реестре, и в командах занимала бы место у рабочих действий. */
const commands = computed(() => {
  if (!store.selected) return []
  const has = store.selected.fields.length
  return [
    /* На телефоне главное действие уезжает на плавающую кнопку (решает
       AppPage — там это приём устройства, а не узкой панели); пока что-то
       выбрано, она уступает место плашке выбора: угол один и тот же. */
    ...(store.canEdit
      ? [{
          key: 'add', label: 'Добавить', icon: 'add', variant: 'filled',
          primary: true, fab: true, hidden: selectionCount.value > 0,
        }]
      : []),
    // Вид выбирает человек; в тесной панели выбирать нечего — там только карточки.
    ...(has && !narrow.value
      ? [{
          key: 'view',
          label: viewMode.value === 'cards' ? 'Карточки' : 'Таблица',
          icon: viewMode.value === 'cards' ? 'grid_view' : 'table_rows',
          children: [
            { key: 'view-table', label: 'Таблица', icon: 'table_rows' },
            { key: 'view-cards', label: 'Карточки', icon: 'grid_view' },
          ],
        }]
      : []),
    ...(has && !narrow.value && viewMode.value === 'table'
      ? [{ key: 'cols', label: 'Колонки', icon: 'view_column' }]
      : []),
    ...(hasQrFields.value
      ? [
          { key: 'qr-find', label: 'Найти по QR-коду', icon: 'qr_code_scanner' },
          { key: 'qr-print', label: 'Печать QR-кодов', icon: 'print' },
        ]
      : []),
    ...(has ? [{ key: 'export', label: 'Выгрузить в Excel', icon: 'download' }] : []),
    ...(store.canManage
      ? [
          { key: 'structure', label: 'Настроить реестр', icon: 'tune' },
          /* Поделиться — те же три пути, что и в контекстном меню списка:
             добираться до них только правым кликом неудобно, а на телефоне
             контекстного меню у списка нет вовсе. */
          {
            key: 'share',
            label: 'Поделиться',
            icon: 'share',
            children: [
              { key: 'share-link', label: 'По ссылке', icon: 'link' },
              { key: 'share-users', label: 'Пользователю', icon: 'person_add' },
              { key: 'share-company', label: 'Компания', icon: 'domain' },
            ],
          },
        ]
      : []),
  ]
})

function onCommand(key) {
  const actions = {
    add: openRecordCreate,
    'view-table': () => setView('table'),
    'view-cards': () => setView('cards'),
    cols: () => { colsOpen.value = true },
    'qr-find': () => { qrFindOpen.value = true },
    'qr-print': () => { qrPrintOpen.value = true },
    export: () => { exportOpen.value = true },
    structure: () => openStructure(store.selected),
    'share-link': () => openShare('share-link'),
    'share-users': () => openShare('share-users'),
    'share-company': () => openShare('share-company'),
  }
  actions[key]?.()
}

const searchInput = ref('')

/* Вид записей — личная настройка устройства (как и состав колонок), поэтому
   живёт в localStorage, а не на сервере: на большом мониторе человеку удобна
   таблица, на ноутбуке — карточки, и это про устройство, а не про реестр. */
const VIEW_KEY = 'gw_registry_view'
const viewMode = ref(localStorage.getItem(VIEW_KEY) === 'cards' ? 'cards' : 'table')
function setView(mode) {
  viewMode.value = mode
  localStorage.setItem(VIEW_KEY, mode)
}

const colsOpen = ref(false)
const dialogOpen = ref(false)
const activeRecord = ref(null)
const startEditing = ref(false)
const qrFindOpen = ref(false)
const qrPrintOpen = ref(false)
const exportOpen = ref(false)
const confirmBulk = ref(false)
const recordToDelete = ref(null)
const registryToDelete = ref(null)

const hasQrFields = computed(() => (store.selected?.fields || []).some(hasQr))
const totalPages = computed(() => Math.max(1, Math.ceil(store.total / store.filters.per_page)))

// ── Контекстное меню списка ──
const menuOpen = ref(false)
const menuX = ref(0)
const menuY = ref(0)
const menuTarget = ref(null)
const renamingId = ref(null)

const menuItems = computed(() => {
  const r = menuTarget.value
  if (!r) return []
  const manage = ['admin', 'owner'].includes(r.my_access)
  /* Пункт опознаётся полем `action` (контракт ContextMenu): у пункта с
     подменю его нет — он лишь раскрывает детей, а выполняются уже они. */
  return [
    { label: 'Настроить', icon: 'tune', action: 'structure', disabled: !manage },
    { label: 'Переименовать', icon: 'edit', action: 'rename', disabled: !manage },
    {
      label: 'Поделиться',
      icon: 'share',
      disabled: !manage,
      children: [
        { label: 'По ссылке', icon: 'link', action: 'share-link' },
        { label: 'Пользователю', icon: 'person_add', action: 'share-users' },
        { label: 'Компания', icon: 'domain', action: 'share-company' },
      ],
    },
    { divider: true },
    { label: 'Удалить', icon: 'delete', danger: true, action: 'delete', disabled: r.my_access !== 'owner' },
  ]
})

function onRegistryContext(r, e) {
  menuTarget.value = r
  menuX.value = e.clientX
  menuY.value = e.clientY
  menuOpen.value = true
}

const shareLinkOpen = ref(false)
const shareUsersOpen = ref(false)
const shareCompanyOpen = ref(false)
const shareTargetId = ref(null)

// openShare — общий вход в диалоги доступа: и из команд шапки, и из
// контекстного меню списка. Реестр берётся тот, о котором спросили.
function openShare(action, registry = store.selected) {
  if (!registry) return
  shareTargetId.value = registry.id
  shareLinkOpen.value = action === 'share-link'
  shareUsersOpen.value = action === 'share-users'
  shareCompanyOpen.value = action === 'share-company'
}

function onMenuSelect(action) {
  const r = menuTarget.value
  if (!r) return
  menuOpen.value = false
  if (action === 'structure') {
    openStructure(r)
    return
  }
  if (action === 'rename') {
    renamingId.value = r.id
    return
  }
  if (action === 'delete') {
    registryToDelete.value = r
    return
  }
  if (action.startsWith('share-')) openShare(action, r)
}

async function applyRename(r, name) {
  renamingId.value = null
  try {
    await store.renameRegistry(r.id, name)
  } catch (e) {
    notif.error(e?.message || 'Не удалось переименовать реестр')
  }
}

async function doDeleteRegistry() {
  const r = registryToDelete.value
  registryToDelete.value = null
  try {
    await store.removeRegistry(r.id)
    notif.success('Реестр удалён')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить реестр')
  }
}

// ── Структура реестра ──
const structureOpen = ref(false)
const structureTarget = ref(null)

function openCreate() {
  structureTarget.value = null
  structureOpen.value = true
}

function openStructure(r) {
  structureTarget.value = r
  structureOpen.value = true
}

/* Создание и правка идут одной формой: у нового реестра сначала появляется сам
   реестр, потом его поля — id полей выдаёт сервер, и до создания их некуда
   привязать. */
async function saveStructure({ name, accounting, section_field_id, fields }) {
  let reg = structureTarget.value
  if (!reg) {
    reg = await store.createRegistry(name, accounting)
    if (fields.length) reg = await store.replaceFields(reg.id, fields)
    selectRegistry(reg.id)
    return
  }
  if (fields.length || reg.fields.length) await store.replaceFields(reg.id, fields)
  await store.updateRegistry(reg.id, { name, accounting, section_field_id })
}

// Видимые колонки — per-реестр, в localStorage (тот же механизм, что и на
// публичной странице внешней ссылки).
const { visible: visibleCols, shown: shownFields, toggle: toggleCol, move: moveCol,
  widths: colWidths, setWidths: setColWidths, resetWidths: resetColWidths } = useRegistryColumns(
  () => store.selected?.fields || [],
  () => (store.selectedId == null ? null : `gw_registry_cols_${store.selectedId}`),
)

/* Выбор живёт между страницами: смена реестра, поиска, подраздела или фильтров —
   уже другая выборка, поэтому она его сбрасывает (иначе «выбрано всё» означало
   бы не то, что человек видел). */
const {
  mode: selectionMode, picked, count: selectionCount, isSelected, allSelected,
  toggle: toggleRow, toggleAll, selectAllMatching, clear: clearSelection, payload: selection,
} = useRowSelection(() => store.records, {
  total: () => store.total,
  scope: () => [
    store.selectedId, store.filters.search, store.filters.section,
    JSON.stringify(store.filters.filters),
  ].join('|'),
})

// Диалогам печати и выгрузки нужны именно отмеченные id (режим «все» они
// передают фильтром, как и удаление).
const pickedIds = computed(() => (selectionMode.value === 'all' ? [] : [...picked.value]))

watch(() => store.selectedId, () => {
  searchInput.value = ''
  clearSelection()
  colsOpen.value = false
})

// ── Поиск ──
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(searchInput.value.trim()), 300)
}
function clearSearch() {
  clearTimeout(searchTimer)
  searchInput.value = ''
  store.setSearch('')
}

// ── Записи ──
function openRecord(rec) {
  activeRecord.value = rec
  startEditing.value = false
  dialogOpen.value = true
}

// «Редактировать» из меню — та же карточка, но сразу в правке.
function editRecord(rec) {
  activeRecord.value = rec
  startEditing.value = true
  dialogOpen.value = true
}

function openRecordCreate() {
  activeRecord.value = null
  startEditing.value = false
  dialogOpen.value = true
}
const saveRecord = (data, record) => (
  record ? store.updateRecord(record.id, data) : store.createRecord(data)
)

// Файл кладём В КОНКРЕТНЫЙ реестр: от него зависит и право на загрузку, и чья
// квота платит за место.
const uploadRecordFile = (file) => uploadFile(store.selectedId, file)

function askDeleteRecord(rec) {
  recordToDelete.value = rec
}

async function doDeleteRecord() {
  const rec = recordToDelete.value
  recordToDelete.value = null
  try {
    await store.deleteRecord(rec.id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить запись')
  }
}

async function doBulkDelete() {
  confirmBulk.value = false
  const sel = selection.value
  clearSelection()
  try {
    await store.bulkDelete(sel)
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить записи')
  }
}

// ── Учётный режим ──
const issueOpen = ref(false)
const issueRecordRef = ref(null)
const issueRecord = computed(() =>
  store.records.find((r) => r.id === issueRecordRef.value?.id) || issueRecordRef.value)

const issueTitle = computed(() => {
  const rec = issueRecord.value
  if (!rec) return ''
  for (const f of store.selected?.fields || []) {
    const v = textValue(f, rec.data?.[String(f.id)])
    if (v) return v
  }
  return `Запись №${rec.id}`
})

function openIssue(rec) {
  issueRecordRef.value = rec
  issueOpen.value = true
}

// Из карточки записи: её закрываем, иначе два диалога встают друг на друга.
function openIssueFromCard(rec) {
  dialogOpen.value = false
  openIssue(rec)
}

// ── Выгрузка ──
const exportableFields = computed(() =>
  (store.selected?.fields || []).filter((f) => isExportable(f.type)))
const exportRequest = (params) => exportRecords(store.selectedId, params)

// Страница записей для QR-диалогов: они догружают реестр сами, страницами.
const fetchRecordsPage = (params) => getRecords(store.selectedId, params)
const fetchIssues = (recordId) => getIssues(store.selectedId, recordId)

/* Переход из строки поиска Hola: открыть нужный реестр и подставить искомый
   текст — найденная запись сразу в выборке. */
function applySearchQuery() {
  const { registry, q } = route.query
  if (registry) selectRegistry(Number(registry))
  if (q) {
    searchInput.value = String(q)
    store.setSearch(String(q))
  }
}

onMounted(() => store.fetchRegistries().then(applySearchQuery))
watch(() => route.query, applySearchQuery)

defineExpose({ confirmBulk })
</script>

<style scoped>
/* Каркас, шапка, команды и футер — общие компоненты (AppListDetail / AppPage),
   список и диалоги — components/registry/*. Здесь остаётся только то, что
   принадлежит самому разделу. */
.rg-total { flex: 1; font-size: 13px; color: var(--color-text-dim); }
.rg-pager { display: flex; align-items: center; gap: 8px; }
.rg-page-info { min-width: 56px; text-align: center; font-size: 13px; color: var(--color-text-dim); }
</style>
