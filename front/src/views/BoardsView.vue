<script setup>
/* Раздел «Доски»: слева дерево папок (свои + расшаренные мне), справа — плитки
   досок с эскизами холста. Права и организация те же, что у
   заметок (владелец, адресаты, компании, публичные ссылки) — отличается только
   содержимое: вместо текста рисунок. */
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import InputText from 'primevue/inputtext'
import BrandLoader from '@/components/common/BrandLoader.vue'
import Breadcrumbs from '@/components/common/Breadcrumbs.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import TreeView from '@/components/common/TreeView.vue'
import FolderEditDialog from '@/components/boards/FolderEditDialog.vue'
import MoveToFolderDialog from '@/components/boards/MoveToFolderDialog.vue'
import ShareDialog from '@/components/boards/ShareDialog.vue'
import * as api from '@/api/boards.js'
import { timeAgo } from '@/utils/time.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBoardsStore } from '@/stores/boards.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const store = useBoardsStore()
const auth = useAuthStore()
const notify = useNotificationsStore()
const route = useRoute()
const router = useRouter()

const importInput = ref(null)
const expanded = ref(new Set())
const searchDraft = ref('')
let searchTimer = null

// Меню: kind различает, к чему относятся пункты (доска, папка, пустое место).
const menu = ref({ visible: false, x: 0, y: 0, kind: 'board' })
const menuBoard = ref(null)
const menuFolder = ref(null)

const folderDlgOpen = ref(false)
const folderDlgTarget = ref(null)
const shareOpen = ref(false)
const moveOpen = ref(false)
const shareSubject = ref({ type: 'board', id: null })
const moveSubject = ref({ type: 'board', id: null })
const confirmTarget = ref(null)  // { kind: 'board'|'folder', item }

const myId = computed(() => auth.userId)

// Своя доска/папка или расшаренная мне. Владельческие плитки сервер отдаёт без
// полей владельца и my_access — их он заполняет только для чужих, поэтому они
// надёжнее сравнения id (тип id в клеймах и в JSON может не совпасть).
function isMine(item) {
  if (!item) return false
  if (item.my_access) return item.my_access === 'owner'
  if (item.owner_name) return false
  return item.owner_id == null || String(item.owner_id) === String(myId.value)
}

/** Когда правили в последний раз: «12 мин назад» и точная дата в подсказке. */
function editedAt(item) {
  const at = item?.updated_at || item?.created_at
  if (!at) return ''
  return timeAgo(at)
}

function editedAtFull(item) {
  const at = item?.updated_at || item?.created_at
  if (!at) return ''
  return new Date(at).toLocaleString('ru-RU', {
    day: 'numeric', month: 'long', year: 'numeric', hour: '2-digit', minute: '2-digit',
  })
}

const crumbRootLabel = computed(() => {
  if (store.showShared) return 'Поделились со мной'
  if (store.showArchived) return 'Архив'
  return 'Мои доски'
})

const title = computed(() => {
  if (store.search.trim()) return 'Результаты поиска'
  return store.activeFolder?.name || crumbRootLabel.value
})

// Подпапки текущей папки — плитками рядом с досками.
const childFolders = computed(() => {
  if (store.showShared || store.showArchived || store.search.trim()) return []
  const own = store.folders.filter((f) => (f.parent_id ?? null) === (store.activeFolderId ?? null))
  // В корне показываем ещё и расшаренные мне папки — иначе до них не добраться.
  return store.activeFolderId ? own : [...own, ...store.sharedRoots]
})

const isEmpty = computed(() => !store.boards.length && !childFolders.value.length)

onMounted(async () => {
  await store.fetchFolders()
  // Глубокая ссылка из глобального поиска и ленты действий: ?board=&folder=&q=.
  const { board, folder, q } = route.query
  if (q) {
    searchDraft.value = String(q)
    store.search = String(q)
  }
  if (folder) await store.openFolder(Number(folder))
  else await store.fetchBoards()
  if (board) openBoard({ id: Number(board) })
})

function onSearchInput() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    store.search = searchDraft.value
    store.fetchBoards()
  }, 250)
}

function openBoard(b) {
  router.push(`/boards/${b.id}`)
}

async function createAndOpen() {
  try {
    openBoard(await store.createBoard('Новая доска'))
  } catch {
    notify.error('Не удалось создать доску')
  }
}

function newFolder() {
  folderDlgTarget.value = null
  folderDlgOpen.value = true
}

// ── Дерево и крошки ──────────────────────────────────────────────

function onTreeSelect(node) {
  store.openFolder(node.id)
}

function onTreeToggle(id) {
  const next = new Set(expanded.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  expanded.value = next
}

function onCrumb(index) {
  store.openFolder(index < 0 ? null : store.path[index]?.id ?? null)
}

// ── Контекстное меню ─────────────────────────────────────────────

function openBoardMenu(b, e) {
  menuBoard.value = b
  menu.value = { visible: true, x: e.clientX, y: e.clientY, kind: 'board' }
}

function onFolderContext({ node, event }) {
  menuFolder.value = node
  menu.value = { visible: true, x: event.clientX, y: event.clientY, kind: 'folder' }
}

function openEmptyMenu(e) {
  menu.value = { visible: true, x: e.clientX, y: e.clientY, kind: 'empty' }
}

const menuItems = computed(() => {
  if (menu.value.kind === 'folder') return folderMenuItems(menuFolder.value)
  if (menu.value.kind === 'empty') return emptyMenuItems()
  return boardMenuItems(menuBoard.value)
})

function boardMenuItems(b) {
  if (!b) return []
  const mine = isMine(b)
  return [
    { label: 'Открыть', icon: 'open_in_new', action: 'open' },
    ...(mine ? [
      { label: b.pinned_at ? 'Открепить' : 'Закрепить', icon: 'push_pin', action: 'pin' },
      { label: 'Переместить…', icon: 'drive_file_move', action: 'move' },
      { label: 'Дублировать', icon: 'content_copy', action: 'copy' },
      { label: 'Поделиться…', icon: 'share', action: 'share' },
      { divider: true },
      { label: b.archived ? 'Вернуть из архива' : 'В архив', icon: 'inventory_2', action: 'archive' },
    ] : []),
    {
      label: 'Скачать',
      icon: 'download',
      children: [
        { label: 'Картинкой (.svg)', icon: 'image', action: 'export-svg' },
        { label: 'Сценой (.json)', icon: 'data_object', action: 'export-json' },
      ],
    },
    ...(mine ? [
      { divider: true },
      { label: 'Удалить', icon: 'delete', danger: true, action: 'delete' },
    ] : []),
  ]
}

function folderMenuItems(f) {
  if (!f) return []
  const mine = isMine(f)
  return [
    { label: 'Открыть', icon: 'folder_open', action: 'open' },
    ...(mine ? [
      { label: 'Переименовать…', icon: 'drive_file_rename_outline', action: 'rename' },
      { label: 'Переместить…', icon: 'drive_file_move', action: 'move' },
      { label: 'Поделиться…', icon: 'share', action: 'share' },
    ] : []),
    { label: 'Скачать архивом', icon: 'folder_zip', action: 'export' },
    ...(mine ? [
      { divider: true },
      { label: 'Удалить', icon: 'delete', danger: true, action: 'delete' },
    ] : []),
  ]
}

function emptyMenuItems() {
  return [
    { label: 'Новая доска', icon: 'add', action: 'new-board' },
    { label: 'Новая папка', icon: 'create_new_folder', action: 'new-folder' },
    { label: 'Импорт из файла…', icon: 'upload_file', action: 'import' },
  ]
}

function onMenuSelect(action) {
  if (menu.value.kind === 'board') boardAction(action)
  else if (menu.value.kind === 'folder') folderAction(action)
  else emptyAction(action)
}

function onMenuClose() {
  menu.value = { ...menu.value, visible: false }
}

function boardAction(action) {
  const b = menuBoard.value
  if (!b) return
  switch (action) {
    case 'open': openBoard(b); break
    case 'pin': store.togglePinned(b); break
    case 'archive': store.toggleArchived(b); break
    case 'copy': store.copyBoard(b.id).catch(() => notify.error('Не удалось дублировать доску')); break
    case 'move': moveSubject.value = { type: 'board', id: b.id }; moveOpen.value = true; break
    case 'share': shareSubject.value = { type: 'board', id: b.id }; shareOpen.value = true; break
    case 'export-svg': download(b, 'svg'); break
    case 'export-json': download(b, 'json'); break
    case 'delete': confirmTarget.value = { kind: 'board', item: b }; break
    default: break
  }
}

function folderAction(action) {
  const f = menuFolder.value
  if (!f) return
  switch (action) {
    case 'open': store.openFolder(f.id); break
    case 'rename': folderDlgTarget.value = f; folderDlgOpen.value = true; break
    case 'move': moveSubject.value = { type: 'folder', id: f.id }; moveOpen.value = true; break
    case 'share': shareSubject.value = { type: 'folder', id: f.id }; shareOpen.value = true; break
    case 'export': downloadFolder(f); break
    case 'delete': confirmTarget.value = { kind: 'folder', item: f }; break
    default: break
  }
}

function emptyAction(action) {
  if (action === 'new-board') createAndOpen()
  else if (action === 'new-folder') newFolder()
  else if (action === 'import') importInput.value?.click()
}

async function confirmDelete() {
  const t = confirmTarget.value
  confirmTarget.value = null
  if (!t) return
  try {
    if (t.kind === 'board') {
      await store.removeBoard(t.item.id)
      notify.success('Доска удалена')
    } else {
      await store.removeFolder(t.item.id)
      notify.success('Папка удалена')
    }
  } catch {
    notify.error('Не удалось удалить')
  }
}

// ── Выгрузка и импорт ────────────────────────────────────────────

async function download(b, format) {
  try {
    saveBlob(await api.exportBoard(b.id, format), `${b.title || 'Доска'}.${format}`)
  } catch {
    notify.error('Не удалось выгрузить доску')
  }
}

async function downloadFolder(f) {
  try {
    saveBlob(await api.exportFolder(f.id, 'svg'), `${f.name}.zip`)
  } catch (e) {
    notify.error(e?.message || 'Не удалось выгрузить папку')
  }
}

function saveBlob(blob, name) {
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = name
  a.click()
  URL.revokeObjectURL(url)
}

async function onImportPick(e) {
  const files = [...(e.target.files || [])]
  e.target.value = ''
  let ok = 0
  for (const file of files) {
    try {
      await api.importBoard(file, store.activeFolderId)
      ok += 1
    } catch {
      notify.error(`Не удалось импортировать «${file.name}»`)
    }
  }
  if (ok) {
    await store.fetchBoards({ silent: true })
    notify.success(ok === 1 ? 'Доска загружена' : `Загружено досок: ${ok}`)
  }
}
</script>

<template>
  <div class="bv">
    <aside class="bv-side">
      <div class="bv-side-actions">
        <button type="button" class="btn-grad bv-new" @click="createAndOpen">
          <span class="material-symbols-outlined">add</span> Новая доска
        </button>
        <button type="button" class="btn-glass bv-new" @click="newFolder">
          <span class="material-symbols-outlined">create_new_folder</span> Папка
        </button>
      </div>

      <nav class="bv-scopes">
        <button
          type="button"
          class="bv-scope"
          :class="{ 'is-active': !store.showShared && !store.showArchived && !store.activeFolderId }"
          @click="store.openFolder(null)"
        >
          <span class="material-symbols-outlined">gesture</span> Мои доски
        </button>
        <button
          type="button"
          class="bv-scope"
          :class="{ 'is-active': store.showShared }"
          @click="store.openShared()"
        >
          <span class="material-symbols-outlined">group</span> Поделились со мной
        </button>
        <button
          type="button"
          class="bv-scope"
          :class="{ 'is-active': store.showArchived }"
          @click="store.openArchive()"
        >
          <span class="material-symbols-outlined">inventory_2</span> Архив
        </button>
      </nav>

      <div class="bv-tree">
        <TreeView
          :nodes="store.folderTree"
          :selected-id="store.activeFolderId"
          :expanded="expanded"
          @select="onTreeSelect"
          @toggle="onTreeToggle"
          @context="onFolderContext"
        />
        <template v-if="store.sharedRoots.length">
          <div class="bv-side-title"><span>Расшаренные мне</span></div>
          <TreeView
            :nodes="store.sharedRoots"
            :selected-id="store.activeFolderId"
            :expanded="expanded"
            @select="onTreeSelect"
            @context="onFolderContext"
          />
        </template>
      </div>

    </aside>

    <section class="bv-main" @contextmenu.self.prevent="openEmptyMenu">
      <div class="bv-sticky">
      <header class="bv-head">
        <Breadcrumbs
          :items="store.path"
          :root-label="crumbRootLabel"
          root-icon="gesture"
          class="bv-crumbs"
          @navigate="onCrumb"
        />
        <div class="bv-search">
          <span class="material-symbols-outlined">search</span>
          <InputText
            v-model="searchDraft"
            placeholder="Поиск по названиям и надписям"
            @input="onSearchInput"
          />
        </div>
        <button type="button" class="bv-icon" title="Импорт доски" @click="importInput?.click()">
          <span class="material-symbols-outlined">upload_file</span>
        </button>
        <input ref="importInput" type="file" accept=".json,.txt" hidden multiple @change="onImportPick" />
      </header>

      <h2 class="bv-title">{{ title }}</h2>
      </div>

      <BrandLoader v-if="store.loading" :size="64" class="bv-loader" />

      <EmptyState
        v-else-if="isEmpty"
        class="bv-empty"
        icon="gesture"
        tone="soft"
        title="Здесь пока пусто"
        subtitle="Доска — бесконечный холст: схемы, наброски, стикеры. Рисуйте сами или откройте доступ коллегам."
      >
        <button v-if="!store.showShared && !store.showArchived" type="button" class="btn-grad" @click="createAndOpen">
          <span class="material-symbols-outlined">add</span> Создать доску
        </button>
      </EmptyState>

      <div v-else class="bv-grid" @contextmenu.self.prevent="openEmptyMenu">
        <button
          v-for="f in childFolders"
          :key="`f${f.id}`"
          type="button"
          class="bv-card bv-card--folder"
          :title="f.name"
          @click="store.openFolder(f.id)"
          @contextmenu.prevent.stop="onFolderContext({ node: f, event: $event })"
        >
          <div class="bv-thumb">
            <span
              class="material-symbols-outlined bv-folder-ic"
              :style="f.color ? { color: `var(--tag-${f.color}-accent)` } : null"
            >{{ isMine(f) ? 'folder' : 'folder_shared' }}</span>
          </div>
          <div class="bv-card-body">
            <h3 class="bv-card-title">{{ f.name }}</h3>
            <div class="bv-card-meta">
              <span :title="editedAtFull(f)">{{ editedAt(f) }}</span>
              <span v-if="f.shared_by_me" class="material-symbols-outlined bv-badge" title="Есть доступ у других">group</span>
            </div>
          </div>
        </button>

        <article
          v-for="b in store.boards"
          :key="b.id"
          class="bv-card"
          :class="{ 'is-pinned': b.pinned_at }"
          :style="b.color ? { '--card-tint': `var(--tag-${b.color}-surface)` } : null"
          :title="b.title || 'Без названия'"
          @click="openBoard(b)"
          @contextmenu.prevent.stop="openBoardMenu(b, $event)"
        >
          <div class="bv-thumb">
            <img v-if="b.preview_url" :src="b.preview_url" :alt="b.title || 'Доска'" loading="lazy" />
            <span v-else class="material-symbols-outlined bv-thumb-ph">gesture</span>
          </div>
          <div class="bv-card-body">
            <h3 class="bv-card-title">{{ b.title || 'Без названия' }}</h3>
            <div class="bv-card-meta">
              <span :title="editedAtFull(b)">{{ editedAt(b) }}</span>
              <span v-if="b.owner_name" class="bv-owner">· {{ b.owner_name }}</span>
              <span v-if="b.my_access === 'view'" class="bv-chip">только чтение</span>
              <span v-if="b.shared_by_me" class="material-symbols-outlined bv-badge" title="Есть доступ у других">group</span>
              <span v-if="b.pinned_at" class="material-symbols-outlined bv-badge">push_pin</span>
            </div>
          </div>
        </article>

      </div>
    </section>

    <ContextMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :items="menuItems"
      @select="onMenuSelect"
      @close="onMenuClose"
    />

    <FolderEditDialog v-model="folderDlgOpen" :folder="folderDlgTarget" />
    <ShareDialog
      v-model="shareOpen"
      :subject-type="shareSubject.type"
      :subject-id="shareSubject.id"
      @changed="store.fetchFolders({ silent: true })"
    />
    <MoveToFolderDialog
      v-model="moveOpen"
      :item-type="moveSubject.type"
      :item-id="moveSubject.id"
      @moved="store.refresh()"
    />

    <ConfirmDialog
      :visible="!!confirmTarget"
      header="Удалить?"
      :message="confirmTarget?.kind === 'folder'
        ? `Папка «${confirmTarget?.item?.name}» будет удалена, а её содержимое переедет на уровень выше.`
        : `Доска «${confirmTarget?.item?.title || 'Без названия'}» будет удалена без возможности восстановить.`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDelete"
      @cancel="confirmTarget = null"
    />
  </div>
</template>

<style scoped>
.bv {
  display: grid;
  grid-template-columns: 260px 1fr;
  gap: 12px;
  height: 100%;
  min-height: 0;
  padding: 12px;
}

.bv-side {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  overflow-y: auto;
  padding: 12px;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.bv-side-actions { display: flex; flex-direction: column; gap: 6px; }
.bv-new { justify-content: center; gap: 6px; }
.bv-scopes { display: flex; flex-direction: column; gap: 2px; }

.bv-scope {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  font-size: 14px;
  text-align: left;
  cursor: pointer;
}

.bv-scope:hover { background: var(--color-surface-variant); }
.bv-scope.is-active { background: var(--color-primary-container); color: var(--color-on-primary-container); }

.bv-side-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 6px;
  font-size: 12px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
}

.bv-tree { min-height: 0; }

.bv-main {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-height: 0;
  overflow-y: auto;
}

/* Крошки, поиск и заголовок остаются на месте: прокручиваются только плитки. */
.bv-sticky {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-bottom: 8px;
  /* Полупрозрачная плашка со стеклом: под ней проезжают плитки, а фон
     приложения (градиент) остаётся виден. */
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
}

.bv-head { display: flex; align-items: center; gap: 8px; }
.bv-crumbs { min-width: 0; }

.bv-search {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  padding: 0 10px;
  border: 1px solid var(--glass-edge);
  border-radius: 999px;
  background: var(--glass-bg);
}

.bv-search :deep(input) {
  flex: 1;
  min-width: 0;
  border: none;
  background: transparent;
  box-shadow: none;
}

.bv-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 34px;
  max-width: 34px;
  min-height: 34px;
  max-height: 34px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}

.bv-icon:hover { background: var(--color-surface-variant); }
.bv-title { margin: 0; font-size: 1.1rem; font-weight: 600; }

.bv-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 12px;
  padding-bottom: 24px;
}

.bv-card {
  /* Доски и папки — квадраты одного размера: превью занимает всё место, кроме
     нижней подписи. */
  aspect-ratio: 1 / 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  background: var(--card-tint, var(--acrylic-card-bg));
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition: transform 0.15s ease, box-shadow 0.15s ease, border-color 0.15s ease;
}

.bv-card:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--glass-edge));
  box-shadow: var(--shadow-2);
}

.bv-card.is-pinned { border-color: var(--color-primary); }

.bv-thumb {
  display: flex;
  flex: 1;
  min-height: 0;
  align-items: center;
  justify-content: center;
  background: var(--color-surface-variant);
}

.bv-thumb img { width: 100%; height: 100%; object-fit: cover; }
.bv-thumb-ph { font-size: 40px; color: var(--color-text-muted); }
.bv-folder-ic { font-size: 44px; color: var(--color-primary); }

.bv-card-body { display: flex; flex: 0 0 auto; flex-direction: column; gap: 2px; padding: 7px 10px 9px; }

.bv-card-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bv-card-meta {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  font-size: 11px;
  color: var(--color-text-muted);
  white-space: nowrap;
  overflow: hidden;
}

.bv-owner { overflow: hidden; text-overflow: ellipsis; }
.bv-chip { padding: 1px 8px; border-radius: 999px; background: var(--color-surface-variant); }
.bv-badge { font-size: 14px; color: var(--color-text-muted); }
.bv-loader { margin: auto; }
.bv-empty { margin: auto; }

@media (max-width: 900px) {
  .bv { grid-template-columns: 1fr; padding: 8px; }
  .bv-side { display: none; }
  .bv-grid { grid-template-columns: repeat(auto-fill, minmax(140px, 1fr)); }
}
</style>
