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
    <template v-if="registry" #status>
      <AppChip
        :icon="canEdit ? 'edit' : 'visibility'"
        :tone="canEdit ? 'primary' : 'neutral'"
        :label="canEdit ? 'просмотр и правка' : 'только просмотр'"
      />
    </template>

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
      <EmptyState
        v-if="error"
        class="sr-error"
        icon="link_off"
        tone="error"
        title="Ссылка недоступна"
        :subtitle="error"
      />

      <RegistryRecords
        v-else
        :fields="shownFields"
        :records="records"
        :loading="loading"
        :sort="filters.sort"
        :order="filters.order"
        :selected="selected"
        :all-selected="allSelected"
        :narrow="tight"
        :search="filters.search"
        empty-hint="Владелец ссылки пока не добавил ни одной записи."
        @update:sort="applySort"
        @open="openRecord"
        @toggle="toggleRow"
        @toggle-all="toggleAll"
      >
        <template #selection>
          <AppInfoBar
            v-if="selected.size"
            class="sr-selbar"
            tone="info"
            icon="checklist"
            :message="`Выбрано: ${selected.size}`"
            inline
          >
            <template #actions>
              <AppButton size="sm" icon="download" label="Выгрузить" @click="exportOpen = true" />
              <AppButton size="sm" variant="text" label="Сбросить" @click="clearSelection" />
            </template>
          </AppInfoBar>
        </template>
      </RegistryRecords>
    </template>
  </AppPage>

  <RegistryRecordDialog
    v-model="dialogOpen"
    :registry="registry"
    :record="activeRecord"
    :readonly="!canEdit"
    :save="saveRecord"
    :upload="uploadRecordFile"
    @saved="fetchRecords"
  />

  <RegistryColumnsDialog
    v-model="colsOpen"
    :fields="registry?.fields || []"
    :visible="visibleCols"
    @toggle="toggleCol"
  />

  <RegistryExportDialog
    v-model="exportOpen"
    :fields="exportableFields"
    :selected-ids="[...selected]"
    :search="filters.search"
    :filename="registry?.name || 'registry'"
    :request="exportRequest"
  />
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppPage from '@/components/ui/AppPage.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import RegistryColumnsDialog from '@/components/registry/RegistryColumnsDialog.vue'
import RegistryExportDialog from '@/components/registry/RegistryExportDialog.vue'
import RegistryRecordDialog from '@/components/registry/RegistryRecordDialog.vue'
import RegistryRecords from '@/components/registry/RegistryRecords.vue'
import { useRegistryColumns } from '@/composables/useRegistryColumns.js'
import { useRowSelection } from '@/composables/useRowSelection.js'
import {
  createSharedRecord, exportSharedRecords, getSharedRecords, getSharedRegistry,
  updateSharedRecord, uploadSharedFile,
} from '@/api/registries.js'
import { isExportable } from '@/utils/registryFields.js'

const route = useRoute()
const code = route.params.code

const registry = ref(null)
const error = ref(null)
const records = ref([])
const total = ref(0)
const loading = ref(false)
const booting = ref(true)
const narrow = ref(false)

const filters = reactive({ search: '', sort: 'created_at', order: 'desc', page: 1, per_page: 30 })

const totalPages = computed(() => Math.max(1, Math.ceil(total.value / filters.per_page)))
const exportableFields = computed(() => (registry.value?.fields || []).filter((f) => isExportable(f.type)))

/* Колонки — личная настройка устройства, ключ по коду ссылки: аккаунта у гостя
   нет, а разные ссылки ведут в разные реестры. */
const { visible: visibleCols, shown: shownFields, toggle: toggleCol } = useRegistryColumns(
  () => registry.value?.fields || [],
  () => `gw_shared_cols_${code}`,
)

const { selected, allSelected, toggle: toggleRow, toggleAll, clear: clearSelection } =
  useRowSelection(() => records.value)

/* Ссылка бывает двух видов: только просмотр и просмотр с правкой записей.
   Уровень решает сервер — здесь по нему лишь показываем или прячем правку. */
const canEdit = computed(() => registry.value?.access === 'edit')

/* Команды шапки. На ссылке с правкой главное действие — добавить запись (на
   телефоне уезжает на плавающую кнопку), иначе — выгрузка. */
const commands = computed(() => {
  if (!registry.value) return []
  const hasFields = registry.value.fields.length
  return [
    ...(canEdit.value && hasFields
      ? [{ key: 'add', label: 'Добавить', icon: 'add', variant: 'filled', primary: true, fab: true }]
      : []),
    ...(exportableFields.value.length
      ? [{
          key: 'export',
          label: 'Экспорт в XLSX',
          icon: 'download',
          variant: 'glass',
          primary: !canEdit.value,
        }]
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
}

const colsOpen = ref(false)
const exportOpen = ref(false)
const dialogOpen = ref(false)
const activeRecord = ref(null)

function openRecord(rec) {
  activeRecord.value = rec
  dialogOpen.value = true
}
function openCreate() {
  activeRecord.value = null
  dialogOpen.value = true
}

const exportRequest = (params) => exportSharedRecords(code, params)

// Запись создаётся и правится теми же ручками по коду ссылки; ссылке на
// просмотр сервер откажет, даже если сюда как-то дойдут.
const saveRecord = (data, record) => (
  record ? updateSharedRecord(code, record.id, data) : createSharedRecord(code, data)
)
const uploadRecordFile = (file) => uploadSharedFile(code, file)

async function load() {
  try {
    registry.value = await getSharedRegistry(code)
    await fetchRecords()
  } catch (e) {
    error.value = e?.message || 'Ссылка не найдена или была отозвана'
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
.sr-selbar { flex: none; margin: 10px 14px 0; }

.sr-total { flex: 1; font-size: 13px; color: var(--color-text-dim); }
.sr-pager { display: flex; align-items: center; gap: 8px; }
.sr-page-info { min-width: 56px; text-align: center; font-size: 13px; color: var(--color-text-dim); }
</style>
