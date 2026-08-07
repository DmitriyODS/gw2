<template>
  <!-- «Диск»: папки и файлы, корзина, избранное и шаринг. Раскладка от ширины
       ОКНА (container-запросы в стилях) — раздел живёт окном, а не экраном. -->
  <AppPage
    title="Диск"
    :loading="store.loading && store.isEmpty"
    :commands="commands"
    :scroll="false"
    @command="onCommand"
  >
    <template #search>
      <SearchField v-model="query" placeholder="поиск по файлам" @update:model-value="onSearch" />
    </template>

    <template #subhead>
      <AppTabs v-model="tab" :tabs="TABS" @update:model-value="store.setView($event)" />
    </template>

    <div
      class="drive"
      :class="{ 'is-dragover': dragover }"
      @dragover.prevent="dragover = true"
      @dragleave.self="dragover = false"
      @drop.prevent="onDrop"
    >
      <!-- Крошки — общий компонент проводника (тот же, что в заметках и
           досках): один вид пути во всех разделах с папками. -->
      <Breadcrumbs
        v-if="store.view === 'files' && !store.search"
        :items="store.path"
        root-label="Мой диск"
        root-icon="cloud"
        @navigate="onCrumb"
      />

      <!-- Идущие загрузки: у каждой своя полоса, размер и отмена. -->
      <AppCard v-if="store.uploads.length" class="uploads">
        <div v-for="u in store.uploads" :key="u.id" class="upload-row">
          <span class="upload-name">{{ u.name }}</span>
          <span class="upload-size">{{ formatBytes(u.size) }}</span>
          <div class="bar"><span :style="{ width: `${Math.round(u.progress * 100)}%` }" /></div>
          <span class="upload-pct">{{ Math.round(u.progress * 100) }}%</span>
          <AppButton
            variant="icon"
            icon="close"
            aria-label="Отменить загрузку"
            @click="store.cancelUpload(u.id)"
          />
        </div>
      </AppCard>

      <!-- Пустое место — тоже цель: правый клик по нему открывает меню папки,
           обычный — снимает выделение. -->
      <div class="drive-body" @contextmenu.self.prevent="openSpaceMenu" @click.self="selected.clear()">
        <BrandLoader v-if="store.loading && store.isEmpty" block :size="64" />

        <EmptyState
          v-else-if="store.isEmpty"
          :icon="emptyIcon"
          :title="emptyTitle"
          :subtitle="emptySubtitle"
        />

        <div
          v-else
          :class="['grid', `is-${layout}`]"
          @contextmenu.self.prevent="openSpaceMenu"
          @click.self="selected.clear()"
        >
          <DriveItem
            v-for="item in store.folders"
            :key="`f${item.id}`"
            :item="item"
            kind="folder"
            :layout="layout"
            :trash="store.isTrash"
            :selected="selected.has(`folder:${item.id}`)"
            @open="store.openFolder(item.id)"
            @select="onSelect($event)"
            @menu="openMenu($event, item, 'folder')"
          />
          <DriveItem
            v-for="item in store.files"
            :key="`d${item.id}`"
            :item="item"
            kind="file"
            :layout="layout"
            :trash="store.isTrash"
            :selected="selected.has(`file:${item.id}`)"
            @open="preview = item"
            @select="onSelect($event)"
            @menu="openMenu($event, item, 'file')"
          />
        </div>
      </div>

      <!-- Панель выбранного: пока что-то отмечено, действия применяются ко
           всему набору сразу. -->
      <AppCard v-if="selected.size" class="selbar">
        <span class="selbar-count">Выбрано: {{ selected.size }}</span>
        <AppStack row :gap="8">
          <AppButton v-if="!store.isTrash" label="Переместить" icon="drive_file_move" @click="moveSelected" />
          <AppButton v-if="!store.isTrash" label="В корзину" icon="delete" tone="danger" @click="trashSelected" />
          <template v-else>
            <AppButton label="Восстановить" icon="restore" @click="restoreSelected" />
            <AppButton label="Удалить навсегда" icon="delete_forever" tone="danger" @click="askPurgeSelected" />
          </template>
          <AppButton label="Снять выбор" variant="text" @click="selected.clear()" />
        </AppStack>
      </AppCard>

      <div v-if="dragover" class="drop-hint">Отпустите файлы, чтобы загрузить</div>
    </div>

    <!-- Скрытый выбор файлов: кнопка «Загрузить» кликает по нему. -->
    <input ref="fileInput" type="file" multiple hidden @change="onPick">

    <DriveItemMenu
      :visible="menu.open"
      :x="menu.x"
      :y="menu.y"
      :item="menu.item"
      :kind="menu.kind"
      :trash="store.isTrash"
      @close="menu.open = false"
      @action="onAction"
    />

    <!-- Меню пустого места: создать папку, загрузить, сменить вид. -->
    <ContextMenu
      :visible="spaceMenu.open"
      :x="spaceMenu.x"
      :y="spaceMenu.y"
      :items="SPACE_ACTIONS"
      @select="onSpaceAction"
      @close="spaceMenu.open = false"
    />

    <DriveMoveDialog
      v-if="moveTargets.length"
      :items="moveTargets"
      @moved="afterMove"
      @close="moveTargets = []"
    />

    <DrivePreviewDialog v-if="preview" :file="preview" @close="preview = null" />
    <DriveShareDialog v-if="shareTarget" v-bind="shareTarget" @close="shareTarget = null" />

    <AppDialog
      :model-value="renameDlg.open"
      title="Переименовать"
      size="sm"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Сохранить' }]"
      @confirm="applyRename"
      @cancel="renameDlg.open = false"
      @update:model-value="(v) => !v && (renameDlg.open = false)"
    >
      <InputText v-model="renameDlg.name" class="dlg-input" autofocus @keyup.enter="applyRename" />
    </AppDialog>

    <AppDialog
      :model-value="folderDlg.open"
      title="Новая папка"
      size="sm"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Создать' }]"
      @confirm="applyCreateFolder"
      @cancel="folderDlg.open = false"
      @update:model-value="(v) => !v && (folderDlg.open = false)"
    >
      <InputText v-model="folderDlg.name" class="dlg-input" placeholder="Название папки" autofocus @keyup.enter="applyCreateFolder" />
    </AppDialog>

    <ConfirmDialog
      :visible="confirm.open"
      :header="confirm.header"
      :message="confirm.message"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirm.run"
      @cancel="confirm.open = false"
    />
  </AppPage>
</template>

<script setup>
import { computed, defineAsyncComponent, onMounted, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import InputText from 'primevue/inputtext'
import BrandLoader from '@/components/common/BrandLoader.vue'
import Breadcrumbs from '@/components/common/Breadcrumbs.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import DriveItem from '@/components/drive/DriveItem.vue'
import DriveItemMenu from '@/components/drive/DriveItemMenu.vue'
import { useDriveStore } from '@/stores/drive.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatBytes } from '@/utils/money.js'

// Тяжёлые диалоги — по требованию: просмотр тянет вьюеры, шаринг — выбор людей.
const DrivePreviewDialog = defineAsyncComponent(() => import('@/components/drive/DrivePreviewDialog.vue'))
const DriveShareDialog = defineAsyncComponent(() => import('@/components/drive/DriveShareDialog.vue'))
const DriveMoveDialog = defineAsyncComponent(() => import('@/components/drive/DriveMoveDialog.vue'))

const route = useRoute()
const store = useDriveStore()
const notif = useNotificationsStore()

const TABS = [
  { value: 'files', label: 'Мой диск', icon: 'folder' },
  { value: 'recent', label: 'Недавние', icon: 'schedule' },
  { value: 'starred', label: 'Избранное', icon: 'star' },
  { value: 'shared', label: 'Поделились', icon: 'group' },
  { value: 'trash', label: 'Корзина', icon: 'delete' },
]

const tab = ref('files')
const query = ref('')
const layout = ref(localStorage.getItem('gw_drive_layout') || 'grid')
const dragover = ref(false)
const fileInput = ref(null)
const preview = ref(null)
const shareTarget = ref(null)
const menu = reactive({ open: false, x: 0, y: 0, item: null, kind: 'file' })
const spaceMenu = reactive({ open: false, x: 0, y: 0 })
const renameDlg = reactive({ open: false, name: '', item: null, kind: 'file' })
const folderDlg = reactive({ open: false, name: '' })
const confirm = reactive({ open: false, header: '', message: '', run: () => {} })

/* Выбранное — ключи вида «file:12»: в одном наборе живут и файлы, и папки,
   а действия применяются ко всему сразу. */
const selected = reactive(new Set())
const itemKey = (item, kind) => `${kind}:${item.id}`

/* Обычный клик ОТКРЫВАЕТ (папку — внутрь, файл — на просмотр) и лишь снимает
   прежний выбор; отмечает только клик с Ctrl/Cmd или Shift. Иначе одно
   действие делало бы две вещи разом — и открывало, и выделяло. */
function onSelect({ item, kind, additive }) {
  if (!additive) {
    selected.clear()
    return
  }
  const key = itemKey(item, kind)
  if (selected.has(key)) selected.delete(key)
  else selected.add(key)
}

// Что отмечено — в виде списка сущностей.
function selectedItems() {
  const out = []
  for (const key of selected) {
    const [kind, id] = key.split(':')
    const list = kind === 'folder' ? store.folders : store.files
    const item = list.find((f) => String(f.id) === id)
    if (item) out.push({ item, kind })
  }
  return out
}

async function forSelected(run) {
  const items = selectedItems()
  selected.clear()
  try {
    for (const { item, kind } of items) await run(item, kind)
  } catch (e) {
    notif.error(e.message || 'Не удалось выполнить действие')
  }
}

const moveTargets = ref([])

// Переместить — и выбранное пачкой, и один элемент из его меню.
function moveSelected() {
  moveTargets.value = selectedItems()
}

async function afterMove() {
  selected.clear()
  moveTargets.value = []
  await store.load()
}

const trashSelected = () => forSelected((item, kind) =>
  kind === 'folder' ? store.trashFolder(item.id) : store.trashFile(item.id))

const restoreSelected = () => forSelected((item, kind) =>
  kind === 'folder' ? store.restoreFolder(item.id) : store.restoreFile(item.id))

function askPurgeSelected() {
  const count = selected.size
  Object.assign(confirm, {
    open: true,
    header: 'Удалить навсегда?',
    message: `Выбрано: ${count}. Вернуть их будет нельзя, место освободится.`,
    run: () => {
      confirm.open = false
      forSelected((item, kind) => (kind === 'folder' ? store.purgeFolder(item.id) : store.purgeFile(item.id)))
    },
  })
}

// Меню пустого места: то же, что кнопки шапки, но под правой кнопкой мыши —
// привычка из файловых менеджеров.
function openSpaceMenu(event) {
  if (store.isTrash) return
  Object.assign(spaceMenu, { open: true, x: event.clientX, y: event.clientY })
}

const SPACE_ACTIONS = [
  { action: 'folder', label: 'Создать папку', icon: 'create_new_folder' },
  { action: 'upload', label: 'Загрузить файлы', icon: 'upload' },
  { action: 'layout', label: 'Сменить вид', icon: 'grid_view' },
]

function onSpaceAction(action) {
  spaceMenu.open = false
  onCommand(action)
}

const commands = computed(() => {
  if (store.isTrash) {
    return [{ key: 'empty', label: 'Очистить корзину', icon: 'delete_forever', tone: 'danger' }]
  }
  return [
    { key: 'upload', label: 'Загрузить', icon: 'upload', primary: true },
    { key: 'folder', label: 'Папка', icon: 'create_new_folder' },
    {
      key: 'layout',
      label: layout.value === 'grid' ? 'Списком' : 'Плитками',
      icon: layout.value === 'grid' ? 'view_list' : 'grid_view',
    },
  ]
})

function onCommand(key) {
  switch (key) {
    case 'upload': fileInput.value?.click(); break
    case 'folder': folderDlg.name = ''; folderDlg.open = true; break
    case 'layout': toggleLayout(); break
    case 'empty': askEmptyTrash(); break
  }
}

const emptyIcon = computed(() => (store.isTrash ? 'delete' : store.search ? 'search_off' : 'cloud_upload'))
const emptyTitle = computed(() => {
  if (store.search) return 'Ничего не нашли'
  if (store.isTrash) return 'Корзина пуста'
  if (store.view === 'starred') return 'Избранного пока нет'
  if (store.view === 'shared') return 'С вами ничем не поделились'
  return 'Здесь пока пусто'
})
const emptySubtitle = computed(() => {
  if (store.search || store.view !== 'files') return ''
  return 'Перетащите файлы сюда или нажмите «Загрузить».'
})

// Крошки отдают ИНДЕКС в пути (-1 — корень), а не саму папку.
function onCrumb(index) {
  store.openFolder(index < 0 ? null : store.path[index]?.id ?? null)
}

function toggleLayout() {
  layout.value = layout.value === 'grid' ? 'list' : 'grid'
  localStorage.setItem('gw_drive_layout', layout.value)
}

function onSearch(value) {
  store.runSearch(value)
}

function onPick(e) {
  if (e.target.files?.length) uploadFiles(e.target.files)
  e.target.value = '' // тот же файл можно выбрать повторно
}

function onDrop(e) {
  dragover.value = false
  if (e.dataTransfer?.files?.length) uploadFiles(e.dataTransfer.files)
}

async function uploadFiles(list) {
  try {
    await store.upload(list)
  } catch (e) {
    notif.error(e.message || 'Не удалось загрузить файл')
  }
}

function openMenu(event, item, kind) {
  Object.assign(menu, { open: true, x: event.clientX, y: event.clientY, item, kind })
}

async function onAction(action) {
  const { item, kind } = menu
  menu.open = false
  try {
    switch (action) {
      case 'open':
        if (kind === 'folder') store.openFolder(item.id)
        else preview.value = item
        break
      case 'download':
        window.open(`/api/drive/files/${item.id}/download`, '_blank')
        break
      case 'rename':
        Object.assign(renameDlg, { open: true, name: item.name, item, kind })
        break
      case 'star':
        await store.toggleStar(item)
        break
      case 'share':
        shareTarget.value = { kind, id: item.id, name: item.name }
        break
      case 'move':
        moveTargets.value = [{ item, kind }]
        break
      case 'trash':
        await (kind === 'folder' ? store.trashFolder(item.id) : store.trashFile(item.id))
        break
      case 'restore':
        await (kind === 'folder' ? store.restoreFolder(item.id) : store.restoreFile(item.id))
        break
      case 'purge':
        askPurge(item, kind)
        break
    }
  } catch (e) {
    notif.error(e.message || 'Не удалось выполнить действие')
  }
}

async function applyRename() {
  const { item, kind, name } = renameDlg
  renameDlg.open = false
  try {
    await (kind === 'folder' ? store.renameFolder(item.id, name) : store.renameFile(item.id, name))
  } catch (e) {
    notif.error(e.message || 'Не удалось переименовать')
  }
}

async function applyCreateFolder() {
  const name = folderDlg.name.trim()
  folderDlg.open = false
  if (!name) return
  try {
    await store.createFolder(name)
  } catch (e) {
    notif.error(e.message || 'Не удалось создать папку')
  }
}

function askPurge(item, kind) {
  Object.assign(confirm, {
    open: true,
    header: 'Удалить навсегда?',
    message: kind === 'folder'
      ? `Папка «${item.name}» и всё её содержимое исчезнут без возможности вернуть.`
      : `Файл «${item.name}» исчезнет без возможности вернуть.`,
    run: async () => {
      confirm.open = false
      try {
        await (kind === 'folder' ? store.purgeFolder(item.id) : store.purgeFile(item.id))
      } catch (e) {
        notif.error(e.message || 'Не удалось удалить')
      }
    },
  })
}

function askEmptyTrash() {
  Object.assign(confirm, {
    open: true,
    header: 'Очистить корзину?',
    message: 'Всё её содержимое исчезнет без возможности вернуть, место освободится.',
    run: async () => {
      confirm.open = false
      try {
        const res = await store.emptyTrash()
        notif.success(res.deleted ? `Удалено файлов: ${res.deleted}` : 'Корзина была пуста')
      } catch (e) {
        notif.error(e.message || 'Не удалось очистить корзину')
      }
    },
  })
}

/* Открытие сразу на папке: по такой ссылке ведёт поиск Hola. Читаем параметр
   через useRoute — в окне рабочего стола он свой, у каждого окна. */
onMounted(() => {
  const id = Number(route.query.folder)
  store.load(Number.isFinite(id) && id > 0 ? { folderId: id } : {})
})
</script>

<style scoped>
.drive {
  container-type: inline-size;
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

/* Тело раздела скроллится само: у AppPage выключен собственный скролл. */
.drive-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.crumbs {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  font-size: 0.9rem;
}

.crumb {
  padding: 4px 8px;
  border: none;
  border-radius: var(--radius-sm);
  background: none;
  color: var(--color-text);
  cursor: pointer;
}

.crumb:hover { background: var(--color-surface-variant); }
.crumb-sep { color: var(--color-text-dim); }

.grid.is-grid {
  display: grid;
  /* min(...) — иначе в узком окне колонка не сожмётся и появится
     горизонтальная прокрутка. */
  grid-template-columns: repeat(auto-fill, minmax(min(170px, 100%), 1fr));
  gap: 10px;
}

.grid.is-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.uploads { padding: 12px; }

.upload-row {
  display: flex;
  align-items: center;
  gap: 10px;
}

/* В узкой панели полоса ужимается, а имя переносится: строка не должна
   выдавливать раздел по горизонтали. */
@container (max-width: 560px) {
  .bar { flex-basis: 25%; }
  .upload-size { display: none; }
}

.upload-name {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
  font-size: 0.88rem;
}

.upload-size,
.upload-pct {
  flex: none;
  color: var(--color-text-dim);
  font-size: 0.82rem;
  font-variant-numeric: tabular-nums;
}

.upload-pct { min-width: 38px; text-align: right; }

.bar {
  flex: 0 0 40%;
  height: 6px;
  border-radius: 999px;
  background: var(--color-surface-variant);
  overflow: hidden;
}

.bar span {
  display: block;
  height: 100%;
  background: var(--color-primary);
  transition: width 0.2s ease;
}

.drive.is-dragover::after {
  content: '';
  position: absolute;
  inset: 0;
  border: 2px dashed var(--color-primary);
  border-radius: var(--radius-lg);
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
  pointer-events: none;
}

/* Поле в диалоге занимает его ширину: короткий инпут посреди пустой карточки
   выглядит обрезком. */
.dlg-input { width: 100%; }

.selbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 10px 14px;
}

.selbar-count { font-weight: 500; }

.drop-hint {
  position: absolute;
  inset-inline: 0;
  bottom: 16px;
  margin-inline: auto;
  width: fit-content;
  padding: 8px 16px;
  border-radius: 999px;
  background: var(--color-primary);
  color: var(--color-on-primary);
  pointer-events: none;
}
</style>
