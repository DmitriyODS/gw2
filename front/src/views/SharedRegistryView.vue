<template>
  <!-- Реестр по внешней ссылке: страница для человека без аккаунта, но раздел
       тот же самый — каркас, шапка, поиск, записи и диалоги общие с «Реестрами»
       (см. components/registry/*). Отличий два: данные берутся по коду ссылки,
       а правка есть только у ссылки уровня edit (решает сервер). -->
  <AppPage
    :title="registry?.name || 'Реестр'"
    show-title
    :loading="booting"
    flush
    :scroll="false"
    :commands="commands"
    @command="onCommand"
    @narrow-change="narrow = $event"
  >
    <!-- Поиск слотом шапки: в тесной панели он сам сворачивается в лупу. -->
    <template v-if="registry" #search="{ narrow: tight }">
      <SearchField
        v-model="searchInput"
        placeholder="Поиск по записям…"
        :collapsed="tight"
        @update:model-value="onSearch"
        @clear="clearSearch"
      />
    </template>

    <template v-if="registry" #footer>
      <span class="sr-total">Всего записей: {{ total }}</span>
      <div v-if="totalPages > 1" class="sr-pager">
        <AppButton
          variant="icon"
          size="sm"
          icon="chevron_left"
          aria-label="Предыдущая страница"
          :disabled="filters.page <= 1"
          @click="setPage(filters.page - 1)"
        />
        <span class="sr-page-info">{{ filters.page }} / {{ totalPages }}</span>
        <AppButton
          variant="icon"
          size="sm"
          icon="chevron_right"
          aria-label="Следующая страница"
          :disabled="filters.page >= totalPages"
          @click="setPage(filters.page + 1)"
        />
      </div>
      <BrandWordmark :size="15" />
    </template>

    <template #default="{ narrow: tight }">
      <!-- Ссылка «только для своих»: реестра гость не увидит, пока не войдёт.
           Это не поломка, поэтому и вид другой — не отказ, а приглашение. -->
      <EmptyState
        v-if="needAuth"
        class="sr-error"
        icon="lock"
        tone="soft"
        title="Нужен вход в аккаунт"
        subtitle="Владелец открыл этот реестр только для тех, кто вошёл в Groove Work. Войдите или заведите аккаунт — и ссылка откроется."
      >
        <AppStack row :gap="8">
          <AppButton variant="filled" icon="login" label="Войти" @click="goAuth('/login')" />
          <AppButton variant="glass" icon="person_add" label="Регистрация" @click="goAuth('/register')" />
        </AppStack>
      </EmptyState>

      <EmptyState
        v-else-if="error"
        class="sr-error"
        icon="link_off"
        tone="error"
        title="Ссылка недоступна"
        :subtitle="error"
      />

      <RegistryRecords
        v-else-if="registry"
        :fields="shownFields"
        :records="records"
        :loading="loading"
        :sort="filters.sort"
        :order="filters.order"
        :is-selected="isSelected"
        :all-selected="allSelected"
        :selection-count="selectionCount"
        :selection-all="selectionMode === 'all'"
        :total="total"
        :narrow="tight"
        :search="filters.search"
        :sections="sections"
        :section="filters.section"
        :widths="colWidths"
        empty-hint="Владелец ссылки пока не добавил ни одной записи."
        @update:sort="applySort"
        @update:section="setSection"
        @open="openRecord"
        @edit="editRecord"
        @remove="askDeleteRecord"
        @move-column="moveCol"
        @resize-columns="setColWidths"
        @reset-widths="resetColWidths"
        @toggle="toggleRow"
        @toggle-all="toggleAll"
        @select-all-matching="selectAllMatching"
        @clear-selection="clearSelection"
        :registry="registry"
        :can-edit="canEdit"
        @manage="openIssue"
      />
    </template>
  </AppPage>

  <RegistryRecordDialog
    v-model="dialogOpen"
    :registry="registry"
    :record="activeRecord"
    :readonly="!canEdit"
    :start-editing="startEditing"
    :save="saveRecord"
    :upload="uploadRecordFile"
    @saved="fetchRecords"
    @manage="openIssueFromCard"
  />

  <RegistryColumnsDialog
    v-model="colsOpen"
    :fields="registry?.fields || []"
    :visible="visibleCols"
    @toggle="toggleCol"
  />

  <ConfirmDialog
    :visible="!!recordToDelete"
    header="Удалить запись?"
    message="Запись и её файлы будут удалены безвозвратно."
    confirm-label="Удалить" danger-confirm
    @confirm="doDeleteRecord" @cancel="recordToDelete = null"
  />

  <RegistryIssueDialog
    v-model="issueOpen"
    :record="issueRecord"
    :title="issueTitle"
    :issue="issueSharedRecordFn"
    :extend="extendSharedIssueFn"
    :back="returnSharedIssueFn"
    :history="fetchSharedIssues"
    @error="notif.error($event)"
  />

  <RegistryStructureDialog
    v-model="structureOpen"
    :registry="registry"
    :save="saveStructure"
    @error="notif.error($event)"
  />

  <RegistryQrFindDialog
    v-model="qrFindOpen"
    :registry="registry"
    :fetch-page="fetchRecordsPage"
    @found="openRecord"
  />
  <RegistryQrPrintDialog
    v-model="qrPrintOpen"
    :registry="registry"
    :fetch-page="fetchRecordsPage"
    :selected-ids="pickedIds"
    :search="filters.search"
    :section="filters.section"
    :filters="filters.filters"
    :sort="filters.sort"
    :order="filters.order"
  />

  <RegistryExportDialog
    v-model="exportOpen"
    :fields="exportableFields"
    :selected-ids="pickedIds"
    :filter="{ search: filters.search, section: filters.section, filters: filters.filters }"
    :filename="registry?.name || 'registry'"
    :request="exportRequest"
  />
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppPage from '@/components/ui/AppPage.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import RegistryColumnsDialog from '@/components/registry/RegistryColumnsDialog.vue'
import RegistryExportDialog from '@/components/registry/RegistryExportDialog.vue'
import RegistryRecordDialog from '@/components/registry/RegistryRecordDialog.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import RegistryQrFindDialog from '@/components/registry/RegistryQrFindDialog.vue'
import RegistryIssueDialog from '@/components/registry/RegistryIssueDialog.vue'
import RegistryStructureDialog from '@/components/registry/RegistryStructureDialog.vue'
import RegistryQrPrintDialog from '@/components/registry/RegistryQrPrintDialog.vue'
import RegistryRecords from '@/components/registry/RegistryRecords.vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useRegistryColumns } from '@/composables/useRegistryColumns.js'
import { useRowSelection } from '@/composables/useRowSelection.js'
import {
  createSharedRecord, exportSharedRecords, getSharedRecords, getSharedRegistry,
  updateSharedRecord, uploadSharedFile, deleteSharedRecord,
  updateSharedRegistry, replaceSharedFields,
  getSharedIssues, issueSharedRecord, extendSharedIssue, returnSharedIssue,
} from '@/api/registries.js'
import { hasQr, isExportable, sectionOptions, textValue } from '@/utils/registryFields.js'

const route = useRoute()
const router = useRouter()
const notif = useNotificationsStore()
const code = route.params.code

const registry = ref(null)
const error = ref(null)
const records = ref([])
const total = ref(0)
const loading = ref(false)
const booting = ref(true)
// Ссылка требует входа: отдельное состояние, а не текст ошибки — вид у него свой.
const needAuth = ref(false)
const narrow = ref(false)

const filters = reactive({ search: '', sort: 'created_at', order: 'desc', section: '', filters: [], page: 1, per_page: 30 })

// Подразделы — те же, что в разделе: варианты поля-источника.
const sections = computed(() => sectionOptions(registry.value))
function setSection(value) {
  filters.section = filters.section === value ? '' : (value || '')
  filters.page = 1
  fetchRecords()
}

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filters.per_page)))
const exportableFields = computed(() => (registry.value?.fields || []).filter((f) => isExportable(f.type)))

/* Колонки — личная настройка устройства, ключ по коду ссылки: аккаунта у гостя
   нет, а разные ссылки ведут в разные реестры. */
const { visible: visibleCols, shown: shownFields, toggle: toggleCol, move: moveCol,
  widths: colWidths, setWidths: setColWidths, resetWidths: resetColWidths } = useRegistryColumns(
  () => registry.value?.fields || [],
  () => `gw_shared_cols_${code}`,
)

/* Выбор переживает страницы; смена поиска или тега — уже другая выборка,
   поэтому она его сбрасывает. */
const {
  mode: selectionMode, picked, count: selectionCount, isSelected, allSelected,
  toggle: toggleRow, toggleAll, selectAllMatching, clear: clearSelection,
} = useRowSelection(() => records.value, {
  total: () => total.value,
  scope: () => `${filters.search}|${filters.section}`,
})

// Выгрузке нужны отмеченные id; режим «все» уходит фильтром экрана.
const pickedIds = computed(() => (selectionMode.value === 'all' ? [] : [...picked.value]))

/* Ссылка бывает двух видов: только просмотр и просмотр с правкой записей.
   Уровень решает сервер — здесь по нему лишь показываем или прячем правку. */
const canEdit = computed(() => ['edit', 'admin'].includes(registry.value?.access))
const canManage = computed(() => registry.value?.access === 'admin')

/* Команды шапки. На ссылке с правкой главное действие — добавить запись (на
   телефоне уезжает на плавающую кнопку), иначе — выгрузка. */
const commands = computed(() => {
  if (!registry.value) return []
  const hasFields = registry.value.fields.length
  return [
    ...(canEdit.value && hasFields
      // Пока что-то выбрано, плавающая «Добавить» уступает место плашке выбора.
      ? [{
        key: 'add', label: 'Добавить', icon: 'add', variant: 'filled',
        primary: true, fab: true, hidden: selectionCount.value > 0,
      }]
      : []),
    /* Выгрузка — обычная команда панели: как `primary` она забирала на телефоне
       целую строку под себя, хотя ей место в «ещё», в одном ряду с поиском. */
    ...(exportableFields.value.length
      ? [{ key: 'export', label: 'Выгрузить в Excel', icon: 'download', variant: 'glass' }]
      : []),
    // «Администрирование» тем и отличается от правки, что позволяет менять сам
    // реестр, а не только его записи.
    ...(canManage.value
      ? [{ key: 'structure', label: 'Настроить реестр', icon: 'tune' }]
      : []),
    // Печать и поиск по QR — часть ПРОСМОТРА: ссылка на чтение их тоже даёт.
    ...(hasQrFields.value
      ? [
          { key: 'qr-find', label: 'Найти по QR-коду', icon: 'qr_code_scanner' },
          { key: 'qr-print', label: 'Печать QR-кодов', icon: 'print' },
        ]
      : []),
    ...(hasFields && !narrow.value
      ? [{ key: 'cols', label: 'Колонки', icon: 'view_column' }]
      : []),
  ]
})

function onCommand(key) {
  if (key === 'add') openCreate()
  else if (key === 'export') exportOpen.value = true
  else if (key === 'cols') colsOpen.value = true
  else if (key === 'qr-find') qrFindOpen.value = true
  else if (key === 'qr-print') qrPrintOpen.value = true
  else if (key === 'structure') structureOpen.value = true
}

// Ссылка уровня admin правит структуру теми же формами, что и раздел: сама
// форма про источник ничего не знает, ей отдают функцию сохранения.
const structureOpen = ref(false)

async function saveStructure({ name, accounting, section_field_id, fields }) {
  await replaceSharedFields(code, fields)
  const updated = await updateSharedRegistry(code, { name, accounting, section_field_id })
  registry.value = { ...registry.value, ...updated }
  await fetchRecords()
}

const qrFindOpen = ref(false)
const qrPrintOpen = ref(false)
const hasQrFields = computed(() => (registry.value?.fields || []).some(hasQr))

// Записи по коду ссылки — тем же постраничным контрактом, что и в разделе.
const fetchRecordsPage = (params) => getSharedRecords(code, params)

const colsOpen = ref(false)
const exportOpen = ref(false)
const dialogOpen = ref(false)
const activeRecord = ref(null)
const startEditing = ref(false)
const recordToDelete = ref(null)

function openRecord(rec) {
  activeRecord.value = rec
  startEditing.value = false
  dialogOpen.value = true
}

// «Редактировать» из меню записи — та же карточка, но сразу в правке.
function editRecord(rec) {
  activeRecord.value = rec
  startEditing.value = true
  dialogOpen.value = true
}

function openCreate() {
  activeRecord.value = null
  startEditing.value = false
  dialogOpen.value = true
}

/* Учётный режим по ссылке: те же формы, но ручки по коду. Открытую выдачу
   после действия перечитываем вместе со списком — плашка состояния приезжает
   вместе с записями. */
const issueOpen = ref(false)
const issueRecordRef = ref(null)
const issueRecord = computed(() =>
  records.value.find((r) => r.id === issueRecordRef.value?.id) || issueRecordRef.value)

const issueTitle = computed(() => {
  const rec = issueRecord.value
  if (!rec) return ''
  for (const f of registry.value?.fields || []) {
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

const fetchSharedIssues = (recordId) => getSharedIssues(code, recordId)

async function issueSharedRecordFn(recordId, body) {
  const issue = await issueSharedRecord(code, recordId, body)
  await fetchRecords()
  return issue
}

async function extendSharedIssueFn(recordId, body) {
  const issue = await extendSharedIssue(code, recordId, body)
  await fetchRecords()
  return issue
}

async function returnSharedIssueFn(recordId, comment) {
  await returnSharedIssue(code, recordId, comment)
  await fetchRecords()
}

function askDeleteRecord(rec) {
  recordToDelete.value = rec
}

async function doDeleteRecord() {
  const rec = recordToDelete.value
  recordToDelete.value = null
  try {
    await deleteSharedRecord(code, rec.id)
    await fetchRecords()
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить запись')
  }
}

const exportRequest = (params) => exportSharedRecords(code, params)

// Запись создаётся и правится теми же ручками по коду ссылки; ссылке на
// просмотр сервер откажет, даже если сюда как-то дойдут.
const saveRecord = (data, record) => (
  record ? updateSharedRecord(code, record.id, data) : createSharedRecord(code, data)
)
const uploadRecordFile = (file) => uploadSharedFile(code, file)

/* Вход по ссылке возвращает сюда же: человек нажал «Войти» из-за ЭТОГО
   реестра, и после входа он должен оказаться в нём, а не на пустом столе. */
function goAuth(path) {
  router.push({ path, query: { redirect: route.fullPath } })
}

async function load() {
  try {
    registry.value = await getSharedRegistry(code)
    await fetchRecords()
  } catch (e) {
    // 401 у публичной ссылки означает ровно одно: она требует входа.
    if (e?.status === 401 || e?.error === 'SHARE_AUTH_REQUIRED') needAuth.value = true
    else error.value = e?.message || 'Ссылка не найдена или была отозвана'
  } finally {
    booting.value = false
  }
}

// Гонка ответов: медленный запрос не должен перетереть выдачу свежего.
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

function applySort({ sort, order }) {
  filters.sort = sort
  filters.order = order
  filters.page = 1
  fetchRecords()
}

function setPage(page) {
  filters.page = page
  fetchRecords()
}

const searchInput = ref('')
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    filters.search = searchInput.value.trim()
    filters.page = 1
    fetchRecords()
  }, 300)
}
function clearSearch() {
  clearTimeout(searchTimer)
  searchInput.value = ''
  filters.search = ''
  filters.page = 1
  fetchRecords()
}

onMounted(load)
</script>

<style scoped>
/* Каркас, шапка, записи и диалоги — общие компоненты. Здесь остаётся только
   подвал публичной страницы: счётчик, пагинация и марка. */
.sr-error { flex: 1; }

.sr-total { flex: 1; font-size: 13px; color: var(--color-text-dim); }
.sr-pager { display: flex; align-items: center; gap: 8px; }
.sr-page-info { min-width: 56px; text-align: center; font-size: 13px; color: var(--color-text-dim); }
</style>
