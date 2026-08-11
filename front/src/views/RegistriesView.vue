<template>
  <AppListDetail
    v-model:open="detailOpen"
    :loading="store.loadingList && !store.registries.length"
    @narrow-change="narrow = $event"
  >
    <!-- Список реестров -->
    <template #list="{ toggle }">
      <!-- Список подписан всегда: заголовок окна относится к разделу целиком, а
           без подписи непонятно, перечень чего в колонке. -->
      <AppPage
        embedded
        title="Реестры"
        show-title
        :menu="!narrow"
        menu-icon="left_panel_close"
        menu-label="Свернуть список"
        @menu="toggle"
      >
        <EmptyState
          v-if="!store.registries.length"
          size="sm"
          icon="list_alt"
          title="Реестров нет"
          subtitle="Их заводит администратор компании."
        />
        <AppStack v-else :gap="6">
          <AppRow
            v-for="r in store.registries"
            :key="r.id"
            :title="r.name"
            icon="list_alt"
            dense
            clickable
            :selected="r.id === store.selectedId"
            @click="selectRegistry(r.id)"
          />
        </AppStack>
      </AppPage>
    </template>

    <!-- Содержимое выбранного реестра -->
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
              variant="icon"
              size="sm"
              icon="chevron_left"
              aria-label="Предыдущая страница"
              :disabled="store.filters.page <= 1"
              @click="store.setPage(store.filters.page - 1)"
            />
            <span class="rg-page-info">{{ store.filters.page }} / {{ totalPages }}</span>
            <AppButton
              variant="icon"
              size="sm"
              icon="chevron_right"
              aria-label="Следующая страница"
              :disabled="store.filters.page >= totalPages"
              @click="store.setPage(store.filters.page + 1)"
            />
          </div>
        </template>

        <!-- Записи: таблица (широко) / карточки (узко) — тот же компонент, что
             и на публичной странице внешней ссылки. -->
        <RegistryRecords
          v-if="store.selected"
          :fields="shownFields"
          :records="store.records"
          :loading="store.loadingRecords"
          :sort="store.filters.sort"
          :order="store.filters.order"
          :selected="selectedIds"
          :all-selected="allSelected"
          :narrow="narrow"
          :search="store.filters.search"
          empty-hint="Добавьте первую запись — она появится в таблице."
          @update:sort="store.applySort"
          @open="openRecord"
          @toggle="toggleRow"
          @toggle-all="toggleAll"
        >
          <template #selection>
            <AppInfoBar
              v-if="selectedIds.size"
              class="rg-selbar"
              tone="info"
              icon="checklist"
              :message="`Выбрано: ${selectedIds.size}`"
              inline
            >
              <template #actions>
                <AppButton
                  size="sm"
                  tone="danger"
                  icon="delete"
                  label="Удалить"
                  @click="confirmBulk = true"
                />
                <AppButton size="sm" variant="text" label="Сбросить" @click="clearSelection" />
              </template>
            </AppInfoBar>
          </template>
        </RegistryRecords>

        <!-- Реестр не выбран (широкая раскладка) -->
        <EmptyState
          v-else
          icon="table_view"
          tone="soft"
          title="Выберите реестр слева"
          subtitle="Выберите реестр в списке, чтобы просмотреть его данные"
        />
      </AppPage>
    </template>

    <RegistryRecordDialog
      v-model="dialogOpen"
      :registry="store.selected"
      :record="activeRecord"
      :save="saveRecord"
    />

    <RegistryQrFindDialog
      v-model="qrFindOpen"
      :registry="store.selected"
      @found="openRecord"
    />
    <RegistryQrPrintDialog
      v-model="qrPrintOpen"
      :registry="store.selected"
      :selected-ids="selectedIds"
      :search="store.filters.search"
      :sort="store.filters.sort"
      :order="store.filters.order"
    />
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
      title="Внешние ссылки" size="md"
      :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
      @cancel="sharesOpen = false"
    >
      <div class="rg-shares">
        <p class="rg-shares-note">
          По внешней ссылке человек без входа в систему открывает реестр в браузере.
          Ссылка на просмотр позволяет смотреть таблицу, открывать карточки и выгружать
          данные; ссылка на правку — ещё и добавлять записи и менять существующие.
          Любую ссылку можно отозвать в любой момент.
        </p>
        <AppStack row :gap="8">
          <AppButton
            variant="filled"
            icon="link"
            label="Ссылка на просмотр"
            :loading="sharesBusy === 'view'"
            :disabled="!!sharesBusy"
            @click="createShareLink('view')"
          />
          <AppButton
            variant="glass"
            icon="edit"
            label="Ссылка на правку"
            :loading="sharesBusy === 'edit'"
            :disabled="!!sharesBusy"
            @click="createShareLink('edit')"
          />
        </AppStack>

        <div v-if="sharesLoading" class="rg-shares-empty">Загрузка…</div>
        <div v-else-if="!shares.length" class="rg-shares-empty">Ссылок пока нет</div>
        <ul v-else class="rg-shares-list">
          <li v-for="s in shares" :key="s.id" class="rg-share">
            <AppChip
              :icon="s.access === 'edit' ? 'edit' : 'visibility'"
              :tone="s.access === 'edit' ? 'primary' : 'neutral'"
              :label="s.access === 'edit' ? 'правка' : 'просмотр'"
            />
            <input class="rg-share-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
            <AppButton
              variant="icon"
              size="sm"
              icon="content_copy"
              title="Копировать"
              aria-label="Копировать"
              @click="copyShare(s.code)"
            />
            <AppButton
              tag="a"
              variant="icon"
              size="sm"
              icon="open_in_new"
              title="Открыть"
              aria-label="Открыть"
              :href="shareUrl(s.code)"
              target="_blank"
              rel="noopener"
            />
            <AppButton
              variant="icon"
              size="sm"
              tone="danger"
              icon="delete"
              title="Отозвать"
              aria-label="Отозвать"
              @click="revokeShareLink(s.id)"
            />
          </li>
        </ul>
      </div>
    </AppDialog>

    <RegistryExportDialog
      v-model="exportOpen"
      :fields="exportableFields"
      :selected-ids="[...selectedIds]"
      :search="store.filters.search"
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
  </AppListDetail>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import RegistryColumnsDialog from '@/components/registry/RegistryColumnsDialog.vue'
import RegistryExportDialog from '@/components/registry/RegistryExportDialog.vue'
import RegistryRecordDialog from '@/components/registry/RegistryRecordDialog.vue'
import RegistryRecords from '@/components/registry/RegistryRecords.vue'
import RegistryQrFindDialog from '@/components/registry/RegistryQrFindDialog.vue'
import RegistryQrPrintDialog from '@/components/registry/RegistryQrPrintDialog.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppInfoBar from '@/components/ui/AppInfoBar.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import { useRegistryColumns } from '@/composables/useRegistryColumns.js'
import { useRowSelection } from '@/composables/useRowSelection.js'
import { useRegistriesStore } from '@/stores/registries.js'
import { useAuthStore } from '@/stores/auth.js'
import { exportRecords, getShares, createShare, revokeShare } from '@/api/registries.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { hasQr, isExportable } from '@/utils/registryFields.js'

const store = useRegistriesStore()
const route = useRoute()
const authStore = useAuthStore()
const notif = useNotificationsStore()

/* Узкая раскладка — свойство САМОГО раздела (он живёт окном рабочего стола),
   поэтому её сообщает AppListDetail, а не медиазапрос по ширине экрана.
   От неё зависят и вид записей (таблица ⇄ карточки), и кнопка «назад». */
const narrow = ref(false)
const detailOpen = ref(false)

// Живая смена активной компании: реестры прежней компании не должны остаться
// на экране — сбрасываем и грузим список новой.
watch(() => authStore.companyId, (id, prev) => {
  if (id === prev) return
  store.reloadForCompany()
})

function selectRegistry(id) {
  store.select(id)
  detailOpen.value = true
}

/* Команды шапки. Панель сама снимет подписи и уведёт хвост в меню «ещё», когда
   места не хватит, — поэтому отдельного набора кнопок для телефона больше нет. */
const commands = computed(() => {
  if (!store.selected) return []
  const has = store.selected.fields.length
  return [
    { key: 'add', label: 'Добавить', icon: 'add', variant: 'filled', primary: true, fab: true },
    ...(has && !narrow.value ? [{ key: 'cols', label: 'Колонки', icon: 'view_column' }] : []),
    ...(hasQrFields.value
      ? [
          { key: 'qr-find', label: 'Найти по QR-коду', icon: 'qr_code_scanner' },
          { key: 'qr-print', label: 'Печать QR-кодов', icon: 'print' },
        ]
      : []),
    { key: 'shares', label: 'Внешние ссылки', icon: 'link' },
    ...(has ? [{ key: 'export', label: 'Экспорт в XLSX', icon: 'download' }] : []),
  ]
})

function onCommand(key) {
  if (key === 'add') openCreate()
  else if (key === 'cols') colsOpen.value = true
  else if (key === 'qr-find') qrFindOpen.value = true
  else if (key === 'qr-print') qrPrintOpen.value = true
  else if (key === 'shares') openShares()
  else if (key === 'export') exportOpen.value = true
}

const searchInput = ref('')

const colsOpen = ref(false)
const dialogOpen = ref(false)
const activeRecord = ref(null)

// Кнопки QR появляются, только если в реестре есть поля с включённым QR.
const qrFindOpen = ref(false)
const qrPrintOpen = ref(false)
const hasQrFields = computed(() => (store.selected?.fields || []).some(hasQr))

const confirmBulk = ref(false)

const totalPages = computed(() => Math.max(1, Math.ceil(store.total / store.filters.per_page)))

// Видимые колонки — per-реестр, в localStorage (тот же механизм, что и на
// публичной странице внешней ссылки).
const { visible: visibleCols, shown: shownFields, toggle: toggleCol } = useRegistryColumns(
  () => store.selected?.fields || [],
  () => (store.selectedId == null ? null : `gw_registry_cols_${store.selectedId}`),
)

const {
  selected: selectedIds, allSelected, toggle: toggleRow, toggleAll, clear: clearSelection,
} = useRowSelection(() => store.records)

watch(() => store.selectedId, () => {
  searchInput.value = ''
  clearSelection()
  colsOpen.value = false
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
const saveRecord = (data, record) => (
  record ? store.updateRecord(record.id, data) : store.createRecord(data)
)

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
const sharesBusy = ref('') // какой вид ссылки сейчас создаётся ('' — ни один)

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
async function createShareLink(access) {
  sharesBusy.value = access
  try {
    const s = await createShare(store.selectedId, access)
    shares.value.unshift(s)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать ссылку')
  } finally {
    sharesBusy.value = ''
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
const exportableFields = computed(() => (store.selected?.fields || []).filter((f) => isExportable(f.type)))
const exportRequest = (params) => exportRecords(store.selectedId, params)

/* Переход из строки глобального поиска: открыть нужный реестр и подставить
   искомый текст — найденная запись сразу в выборке. */
function applySearchQuery() {
  const { registry, q } = route.query
  if (registry) selectRegistry(Number(registry))
  if (q) store.setSearch(String(q))
}

onMounted(() => store.fetchRegistries().then(applySearchQuery))
watch(() => route.query, applySearchQuery)
</script>

<style scoped>
/* Каркас, шапка, команды, футер и стеклянные панели — общие компоненты
   (AppListDetail / AppPage), записи и диалоги — components/registry/*.
   Здесь остаётся только то, что принадлежит самому разделу. */

.rg-selbar { flex: none; margin: 10px 14px 0; }

/* ── Футер: счётчик и пагинация ── */
.rg-total { flex: 1; font-size: 13px; color: var(--color-text-dim); }
.rg-pager { display: flex; align-items: center; gap: 8px; }
.rg-page-info { min-width: 56px; text-align: center; font-size: 13px; color: var(--color-text-dim); }

/* ── Диалог внешних ссылок ── */
.rg-shares { display: flex; flex-direction: column; gap: 14px; }
.rg-shares-note { margin: 0; font-size: 13px; line-height: 1.5; color: var(--color-text-dim); }
.rg-shares-empty { padding: 16px; text-align: center; font-size: 14px; color: var(--color-text-dim); }
.rg-shares-list { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0; list-style: none; }
.rg-share { display: flex; align-items: center; gap: 6px; }

.rg-share-url {
  flex: 1;
  min-width: 0;
  height: 38px;
  padding: 0 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font-size: 13px;
}
</style>
