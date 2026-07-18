<template>
  <div
    class="na"
    :class="{ 'na-explorer': isExplorer }"
    @dragover="onFileDragOver"
    @dragleave="onFileDragLeave"
    @drop="onFileDrop"
  >
    <!-- ── Верхний тулбар ── -->
    <header class="na-toolbar">
      <div class="na-toolbar-row">
        <SearchField v-model="searchInput" placeholder="Поиск по заметкам…" hotkey />
        <div class="na-actions">
          <div v-if="!isMobile" class="na-modes" role="tablist" aria-label="Режим отображения">
            <button
              class="na-mode" :class="{ active: store.viewMode === 'hierarchy' }"
              role="tab" title="Иерархия" @click="store.setViewMode('hierarchy')"
            >
              <span class="material-symbols-outlined">account_tree</span>
            </button>
            <button
              class="na-mode" :class="{ active: store.viewMode === 'explorer' }"
              role="tab" title="Проводник" @click="store.setViewMode('explorer')"
            >
              <span class="material-symbols-outlined">grid_view</span>
            </button>
          </div>
          <button class="btn-glass" title="Новая папка" @click="newFolder()">
            <span class="material-symbols-outlined">create_new_folder</span>
            <span class="na-lbl">Папка</span>
          </button>
          <button class="btn-glass" title="Импорт .txt / .docx" @click="importInput?.click()">
            <span class="material-symbols-outlined">upload_file</span>
            <span class="na-lbl">Импорт</span>
          </button>
          <input ref="importInput" type="file" accept=".txt,.docx,text/plain" hidden multiple @change="onImportPick" />
          <button class="btn-grad" :disabled="creating" @click="createAndOpen">
            <span class="material-symbols-outlined">add</span>
            <span class="na-lbl">Заметка</span>
          </button>
        </div>
      </div>

      <!-- Тулбар действий над выделением (проводник) -->
      <div v-if="store.hasSelection" class="na-selbar">
        <button class="na-selclose" title="Снять выделение" @click="store.clearSelection()">
          <span class="material-symbols-outlined">close</span>
        </button>
        <span class="na-selcount">Выбрано: {{ store.selectionCount }}</span>
        <div class="na-selactions">
          <template v-if="singleNote">
            <button class="na-selbtn" title="Редактировать" @click="openNote(singleNote)"><span class="material-symbols-outlined">edit</span></button>
            <button class="na-selbtn" title="Поделиться" @click="shareItem('note', singleNote)"><span class="material-symbols-outlined">share</span></button>
            <button class="na-selbtn" title="Скопировать" @click="copyNote(singleNote)"><span class="material-symbols-outlined">content_copy</span></button>
            <button class="na-selbtn" title="Переместить" @click="moveItem('note', singleNote.id)"><span class="material-symbols-outlined">drive_file_move</span></button>
            <button class="na-selbtn" title="Скачать .txt" @click="downloadNote(singleNote, 'txt')"><span class="material-symbols-outlined">download</span></button>
          </template>
          <template v-else-if="singleFolder">
            <button class="na-selbtn" title="Переименовать" @click="renameFolderDlg(singleFolder)"><span class="material-symbols-outlined">drive_file_rename_outline</span></button>
            <button class="na-selbtn" title="Поделиться" @click="shareItem('folder', singleFolder)"><span class="material-symbols-outlined">share</span></button>
            <button class="na-selbtn" title="Скопировать" @click="copyFolder(singleFolder)"><span class="material-symbols-outlined">content_copy</span></button>
            <button class="na-selbtn" title="Переместить" @click="moveItem('folder', singleFolder.id)"><span class="material-symbols-outlined">drive_file_move</span></button>
            <button class="na-selbtn" title="Скачать .zip" @click="downloadFolder(singleFolder)"><span class="material-symbols-outlined">folder_zip</span></button>
          </template>
          <button class="na-selbtn danger" title="Удалить" @click="deleteSelection"><span class="material-symbols-outlined">delete</span></button>
        </div>
      </div>
    </header>

    <div class="na-body">
      <!-- ── Иерархия: сайдбар ── -->
      <aside v-if="!isExplorer && !isMobile" class="na-side">
        <div class="na-side-scroll">
          <button class="na-nav" :class="{ active: isAllActive }" @click="store.selectAll()">
            <span class="material-symbols-outlined">notes</span><span>Все заметки</span>
          </button>
          <button class="na-nav" :class="{ active: store.showShared }" @click="store.selectShared()">
            <span class="material-symbols-outlined">group</span><span>Поделились со мной</span>
          </button>
          <button class="na-nav" :class="{ active: store.showArchived }" @click="store.selectArchive()">
            <span class="material-symbols-outlined">archive</span><span>Архив</span>
          </button>

          <div class="na-side-head">
            <span>Папки</span>
            <button class="na-side-add" title="Новая папка" @click="newFolder()">
              <span class="material-symbols-outlined">add</span>
            </button>
          </div>
          <TreeView
            :nodes="ownTree"
            :selected-id="store.activeFolderId"
            :expanded="expanded"
            @select="onTreeSelect"
            @toggle="onTreeToggle"
            @context="onFolderContext"
            @node-dragstart="onFolderDragStart"
            @node-drop="onTreeDrop"
          />
          <template v-if="store.sharedRoots.length">
            <div class="na-side-head"><span>Расшаренные мне</span></div>
            <TreeView
              :nodes="sharedTree"
              :selected-id="store.activeFolderId"
              :expanded="expanded"
              @select="onTreeSelect"
              @context="onFolderContext"
            />
          </template>
        </div>
      </aside>

      <!-- ── Основная область ── -->
      <section class="na-main">
        <!-- Крошки (проводник) -->
        <Breadcrumbs
          v-if="isExplorer"
          :items="store.path"
          :root-label="crumbRootLabel"
          :root-icon="crumbRootIcon"
          class="na-crumbs"
          @navigate="onCrumb"
          @drop-item="onCrumbDrop"
        />

        <!-- Фильтр по тегам: одна строка с горизонтальной прокруткой чипов;
             подпись слева и кнопки справа зафиксированы (не раздувает экран). -->
        <div class="na-tags">
          <span class="na-tags-label"><span class="material-symbols-outlined">filter_alt</span>Теги:</span>
          <div class="na-tags-scroll">
            <button
              v-for="t in store.tags" :key="t.id"
              class="na-tag" :class="{ active: store.activeTagIds.includes(t.id) }"
              :style="tagStyle(t)"
              @click="store.toggleTag(t.id)"
            >
              <span class="material-symbols-outlined">sell</span>{{ t.name }}
            </button>
            <span v-if="!store.tags.length" class="na-tags-empty">тегов пока нет</span>
          </div>
          <button v-if="store.activeTagIds.length" class="na-tag na-tag-clear" title="Сбросить фильтр" @click="store.clearTags()">
            <span class="material-symbols-outlined">close</span>
          </button>
          <button class="na-tag na-tag-manage" :title="store.tags.length ? 'Управление тегами' : 'Создать тег'" @click="tagManageOpen = true">
            <span class="material-symbols-outlined">{{ store.tags.length ? 'tune' : 'add' }}</span>
            <span v-if="!store.tags.length" class="na-tag-manage-label">Создать тег</span>
          </button>
        </div>

        <div class="na-scroll" @contextmenu.prevent="openEmptyMenu($event)">
          <div v-if="store.loading && !store.notes.length" class="na-note">Загрузка…</div>

          <EmptyState
            v-else-if="isEmpty"
            class="na-empty" :icon="emptyIcon" tone="soft"
            :title="emptyTitle" :subtitle="emptySubtitle"
          >
            <button v-if="!store.showShared && !store.showArchived" class="btn-grad" type="button" @click="createAndOpen">
              <span class="material-symbols-outlined">add</span> Создать заметку
            </button>
          </EmptyState>

          <template v-else>
            <!-- Особые группировки проводника (нельзя удалить): Все / Поделились / Архив -->
            <div v-if="showSpecialCards" class="na-folders na-specials">
              <article
                v-for="g in specialGroups" :key="g.key"
                class="na-fcard na-special"
                @click="onSpecialClick(g.key)"
                @contextmenu.prevent.stop="openSpecialMenu(g.key, $event)"
              >
                <span class="na-fcard-ic material-symbols-outlined">{{ g.icon }}</span>
                <div class="na-fcard-body"><span class="na-fcard-name">{{ g.label }}</span></div>
              </article>
            </div>

            <!-- Папки-плитки (проводник) -->
            <div v-if="isExplorer && folderCards.length" class="na-folders">
              <article
                v-for="f in folderCards" :key="'f' + f.id"
                class="na-fcard" :class="{ selected: store.selectedFolderIds.has(f.id), shared: f.owner_id !== myId, 'na-drop': dropFolderId === f.id }"
                :style="folderCardStyle(f)"
                :draggable="f.owner_id === myId"
                @click="onFolderClick(f, $event)"
                @contextmenu.prevent.stop="onFolderContext({ node: f, event: $event })"
                @dragstart="onFolderDragStart(f, $event)"
                @dragend="dragEnd"
                @dragover.prevent="f.owner_id === myId && (dropFolderId = f.id)"
                @dragleave="dropFolderId === f.id && (dropFolderId = null)"
                @drop.prevent="onFolderCardDrop(f)"
              >
                <span class="na-fcard-ic material-symbols-outlined">{{ f.owner_id !== myId ? 'folder_shared' : 'folder' }}</span>
                <div class="na-fcard-body">
                  <span class="na-fcard-name">{{ f.name }}</span>
                  <span class="na-fcard-sub">{{ f.notes_count || 0 }} зам.</span>
                </div>
                <span v-if="f.shared_by_me" class="na-fcard-badge material-symbols-outlined" title="Вы поделились">share</span>
              </article>
            </div>

            <!-- Плитки заметок -->
            <div class="na-grid">
              <article
                v-for="n in store.notes" :key="n.id"
                class="na-card glass-hover"
                :class="{ colored: n.color, selected: store.selectedNoteIds.has(n.id), shared: isShared(n), 'na-drag': true }"
                :style="noteColorStyle(n)"
                :draggable="!isShared(n)"
                @click="onNoteClick(n, $event)"
                @dblclick="openNote(n)"
                @contextmenu.prevent.stop="openNoteMenu($event.clientX, $event.clientY, n)"
                @dragstart="onNoteDragStart(n, $event)"
                @dragend="dragEnd"
              >
                <span v-if="n.pinned_at" class="na-card-pin material-symbols-outlined" title="Закреплена">keep</span>
                <span v-if="n.shared_by_me" class="na-card-shared material-symbols-outlined" title="Вы поделились">share</span>
                <h3 class="na-card-title">{{ n.title || 'Без названия' }}</h3>
                <p v-if="n.excerpt" class="na-card-excerpt">{{ n.excerpt }}</p>
                <p v-else class="na-card-excerpt dim">Пустая заметка</p>
                <div v-if="noteTags(n).length" class="na-card-tags">
                  <span v-for="t in noteTags(n)" :key="t.id" class="na-card-tag" :style="tagStyle(t)">{{ t.name }}</span>
                </div>
                <footer class="na-card-foot">
                  <template v-if="isShared(n)">
                    <img class="na-card-owner-av" :src="ownerAvatar(n)" :alt="n.owner_name || ''" />
                    <span class="na-card-owner">{{ n.owner_name || 'Владелец' }}</span>
                    <span class="na-card-access" :class="n.my_access">{{ n.my_access === 'edit' ? 'правка' : 'просмотр' }}</span>
                  </template>
                  <template v-else>
                    <span class="material-symbols-outlined">schedule</span>{{ formatDate(n.updated_at) }}
                  </template>
                </footer>
              </article>
            </div>
          </template>
        </div>
      </section>
    </div>

    <AppFab :visible="isMobile && fabVisible" icon="add" aria-label="Новая заметка" @click="createAndOpen" />

    <!-- Оверлей перетаскивания файлов с компьютера -->
    <div v-if="importDragging" class="na-dropzone">
      <div class="na-dropzone-inner">
        <span class="material-symbols-outlined">upload_file</span>
        <p>Отпустите, чтобы импортировать .txt / .docx</p>
      </div>
    </div>

    <!-- Единое контекстное меню (заметка / папка / пустая зона) с подменю -->
    <ContextMenu :visible="menu.visible" :x="menu.x" :y="menu.y" :items="menuItems" @select="onMenuSelect" @close="onMenuClose">
      <template v-if="menuColorTarget" #header>
        <div class="na-menu-colors">
          <ColorSwatchPicker :model-value="menuColorTarget.color || ''" @update:model-value="onPickColor" />
        </div>
      </template>
    </ContextMenu>

    <!-- Диалоги -->
    <FolderEditDialog v-model="folderDlgOpen" :folder="folderDlgTarget" :parent-id="folderDlgParent" @saved="onFolderSaved" />
    <ShareDialog v-model="shareOpen" :subject-type="shareType" :subject-id="shareId" @changed="store.fetchFolders({ silent: true })" />
    <MoveToFolderDialog v-model="moveOpen" :item-type="moveType" :item-id="moveId" @moved="store.fetchFolders({ silent: true })" />
    <NoteTagsDialog v-model="tagsOpen" :note-id="menuNote?.id" :tag-ids="menuNote?.tag_ids || []" @saved="onTagsSaved" />
    <TagManageDialog v-model="tagManageOpen" />
    <PostComposer v-if="postPreset" v-model="postComposerOpen" :preset="postPreset" @saved="notif.success('Опубликовано на портале')" />
    <NoteSendToChatDialog v-model="sendChatOpen" mode="note" :note="menuNote" />

    <ConfirmDialog
      :visible="!!confirmDelete"
      :header="confirmDelete?.header"
      :message="confirmDelete?.message"
      confirm-label="Удалить" danger-confirm
      @confirm="confirmDelete?.run()" @cancel="confirmDelete = null"
    />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useFabOnScroll } from '@/composables/useFabOnScroll.js'
import { useDragItem } from '@/composables/useDragItem.js'
import SearchField from '@/components/common/SearchField.vue'
import AppFab from '@/components/common/AppFab.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import TreeView from '@/components/common/TreeView.vue'
import Breadcrumbs from '@/components/common/Breadcrumbs.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import ColorSwatchPicker from '@/components/common/ColorSwatchPicker.vue'
import ShareDialog from '@/components/notes/ShareDialog.vue'
import MoveToFolderDialog from '@/components/notes/MoveToFolderDialog.vue'
import NoteTagsDialog from '@/components/notes/NoteTagsDialog.vue'
import TagManageDialog from '@/components/notes/TagManageDialog.vue'
import FolderEditDialog from '@/components/notes/FolderEditDialog.vue'
import NoteSendToChatDialog from '@/components/notes/NoteSendToChatDialog.vue'
import * as api from '@/api/notes.js'
import { docToMarkdown } from '@/utils/tiptapMarkdown.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const PostComposer = defineAsyncComponent(() => import('@/components/portal/PostComposer.vue'))

const router = useRouter()
const store = useNotesStore()
const notif = useNotificationsStore()
const auth = useAuthStore()
const { isMobile } = useBreakpoint()
const { fabVisible } = useFabOnScroll()
const { dragItem, start: startDrag, end: endDrag } = useDragItem()

const hasCompany = computed(() => !!auth.companyId)
const myId = computed(() => auth.userId)
const isExplorer = computed(() => store.viewMode === 'explorer' || isMobile.value)

const isAllActive = computed(() =>
  !store.activeFolderId && !store.showShared && !store.showArchived)

// ── Деревья сайдбара ──
const expanded = ref(new Set())
function markTree(nodes, ownMine, shared) {
  return nodes.map((n) => ({ ...n, owner_is_me: ownMine, shared, children: markTree(n.children || [], ownMine, shared) }))
}
const ownTree = computed(() => markTree(store.folderTree, true, false))
const sharedTree = computed(() => store.sharedRoots.map((f) => ({ ...f, owner_is_me: false, shared: true, children: [] })))

// Папки-плитки проводника (при активном поиске/плоском виде не показываем).
const folderCards = computed(() =>
  (isExplorer.value && !store.showArchived && !store.showAllFlat && !store.search ? store.browseChildren : []))

// Особые группировки проводника (Все/Поделились/Архив) — только в «домашнем»
// корне проводника (не внутри папки, не в самой группировке, не при поиске).
const specialGroups = [
  { key: 'all', label: 'Все заметки', icon: 'notes' },
  { key: 'shared', label: 'Поделились со мной', icon: 'group' },
  { key: 'archive', label: 'Архив', icon: 'archive' },
]
const showSpecialCards = computed(() =>
  isExplorer.value && !store.activeFolderId && !store.showAllFlat
  && !store.showArchived && !store.showShared && !store.search)
function onSpecialClick(key) {
  if (key === 'all') store.selectAllFlat()
  else if (key === 'shared') store.selectShared()
  else if (key === 'archive') store.selectArchive()
}

// Крошки-«дом» проводника: показывают текущую группировку либо «Проводник».
const crumbRootLabel = computed(() => {
  if (store.showShared) return 'Поделились со мной'
  if (store.showArchived) return 'Архив'
  if (store.showAllFlat) return 'Все заметки'
  return 'Проводник'
})
const crumbRootIcon = computed(() => {
  if (store.showShared) return 'group'
  if (store.showArchived) return 'archive'
  if (store.showAllFlat) return 'notes'
  return 'home'
})

// ── Пустые состояния ──
const isEmpty = computed(() =>
  !store.loading && !store.notes.length && !folderCards.value.length && !showSpecialCards.value)
const emptyIcon = computed(() => store.showArchived ? 'archive' : store.showShared ? 'group' : store.search ? 'search_off' : 'note_stack')
const emptyTitle = computed(() => {
  if (store.search) return 'Ничего не найдено'
  if (store.showArchived) return 'Архив пуст'
  if (store.showShared) return 'С вами пока не поделились'
  return 'Пока нет заметок'
})
const emptySubtitle = computed(() => {
  if (store.search) return 'Попробуйте изменить запрос.'
  if (store.showShared) return 'Здесь появятся заметки и папки, которыми поделились с вами или вашей компанией.'
  return 'Создайте первую заметку или папку — их можно раскладывать, помечать тегами и делиться.'
})

// ── Поиск ──
const searchInput = ref(store.search)
let searchTimer = null
watch(searchInput, (v) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(v.trim()), 300)
})

onMounted(() => {
  store.fetchFolders()
  store.fetchTags()
  if (isExplorer.value) store.selectAll()
  else store.fetchNotes()
})

// ── Хелперы плиток ──
function isShared(n) { return n.owner_id && n.owner_id !== myId.value }
function noteTags(n) {
  const ids = n.tag_ids || []
  return store.tags.filter((t) => ids.includes(t.id))
}
function tagStyle(t) {
  if (!t.color) return {}
  return { background: `var(--tag-${t.color}-surface)`, borderColor: `var(--tag-${t.color}-border)`, color: `var(--tag-${t.color}-accent)` }
}
function noteColorStyle(n) {
  if (!n.color) return {}
  return {
    background: `var(--glass-bg), color-mix(in oklch, var(--tag-${n.color}-surface) 55%, transparent)`,
    borderColor: `var(--tag-${n.color}-border)`,
  }
}
function folderCardStyle(f) {
  if (!f.color) return {}
  return { borderColor: `var(--tag-${f.color}-border)`, '--fc-accent': `var(--tag-${f.color}-accent)` }
}
function ownerAvatar(n) { return n.owner_avatar ? `/uploads/${n.owner_avatar}` : `/api/users/${n.owner_id}/identicon` }
function formatDate(iso) {
  const d = new Date(iso)
  const p = (x) => String(x).padStart(2, '0')
  return `${p(d.getDate())}.${p(d.getMonth() + 1)}.${d.getFullYear()} ${p(d.getHours())}:${p(d.getMinutes())}`
}

// ── Выделение / открытие ──
// Одиночный клик по заметке — открыть (как у папок); Ctrl/Cmd+клик — выделение.
function onNoteClick(n, e) {
  if (menuJustClosed()) return
  if (e.metaKey || e.ctrlKey) store.toggleNoteSelect(n.id, true)
  else openNote(n)
}
// Одиночный клик по папке — переход внутрь; Ctrl/Cmd+клик — выделение для тулбара.
function onFolderClick(f, e) {
  if (menuJustClosed()) return
  if (e.metaKey || e.ctrlKey) store.toggleFolderSelect(f.id, true)
  else store.openFolder(f)
}
function openNote(n) { router.push(`/notes/${n.id}`) }

const singleNote = computed(() =>
  store.selectedNoteIds.size === 1 && !store.selectedFolderIds.size
    ? store.notes.find((n) => n.id === [...store.selectedNoteIds][0]) : null)
const singleFolder = computed(() =>
  store.selectedFolderIds.size === 1 && !store.selectedNoteIds.size
    ? store.browseChildren.find((f) => f.id === [...store.selectedFolderIds][0]) : null)

// ── Создание ──
const creating = ref(false)
async function createAndOpen() {
  creating.value = true
  try {
    const n = await store.createNote()
    router.push(`/notes/${n.id}`)
  } catch (e) { notif.error(e?.message || 'Не удалось создать заметку') } finally { creating.value = false }
}

// ── Папки: диалоги ──
const folderDlgOpen = ref(false)
const folderDlgTarget = ref(null)
const folderDlgParent = ref(null)
function newFolder(parentId = null) {
  folderDlgTarget.value = null
  folderDlgParent.value = parentId ?? (store.activeFolderId && !store.isSharedContext ? store.activeFolderId : null)
  folderDlgOpen.value = true
}
function renameFolderDlg(f) { folderDlgTarget.value = f; folderDlgParent.value = null; folderDlgOpen.value = true }
function onFolderSaved() { store.clearSelection(); if (isExplorer.value) store.fetchBrowseChildren() }

async function copyFolder(f) {
  try { await store.copyFolder(f.id); notif.success('Папка скопирована') }
  catch (e) { notif.error(e?.message || 'Не удалось скопировать') }
}
async function copyNote(n) {
  try { await store.copyNote(n.id); notif.success('Заметка скопирована') }
  catch (e) { notif.error(e?.message || 'Не удалось скопировать') }
}

// ── Шаринг ──
const shareOpen = ref(false)
const shareType = ref('note')
const shareId = ref(null)
function shareItem(type, item) { shareType.value = type; shareId.value = item.id; shareOpen.value = true }

// ── Перемещение ──
const moveOpen = ref(false)
const moveType = ref('note')
const moveId = ref(null)
function moveItem(type, id) { moveType.value = type; moveId.value = id; moveOpen.value = true }

// ── Скачивание ──
async function download(blobPromise, name, ext) {
  try {
    const resp = await blobPromise
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${(name || 'file').slice(0, 100)}.${ext}`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) { notif.error(e?.message || 'Не удалось скачать') }
}
function downloadNote(n, format) { download(api.exportNote(n.id, format), n.title || 'Заметка', format) }
function downloadFolder(f, format = 'txt') { download(api.exportFolder(f.id, format), f.name || 'Папка', 'zip') }

// ── Удаление ──
const confirmDelete = ref(null)
function deleteSelection() {
  if (singleFolder.value) {
    const f = singleFolder.value
    confirmDelete.value = {
      header: 'Удалить папку?',
      message: `Папка «${f.name}» будет удалена. Вложенные заметки и папки переедут на уровень выше.`,
      run: async () => { confirmDelete.value = null; try { await store.removeFolder(f.id) } catch (e) { notif.error(e?.message) } },
    }
    return
  }
  const ids = [...store.selectedNoteIds]
  confirmDelete.value = {
    header: ids.length > 1 ? 'Удалить заметки?' : 'Удалить заметку?',
    message: `Выбранное будет удалено навсегда вместе с картинками.`,
    run: async () => {
      confirmDelete.value = null
      try { for (const id of ids) await store.removeNote(id); store.clearSelection() }
      catch (e) { notif.error(e?.message) }
    },
  }
}

// ── Контекстное меню (заметка/папка/пустая зона) ──
const menuNote = ref(null)
const tagsOpen = ref(false)
const tagManageOpen = ref(false)
const sendChatOpen = ref(false)
const postComposerOpen = ref(false)
const postPreset = ref(null)

// Единое контекстное меню: kind = note | folder | empty.
const menu = ref({ visible: false, x: 0, y: 0, kind: 'note' })
const menuFolder = ref(null)
const menuSpecial = ref(null) // ключ особой группировки (all|shared|archive)

function openNoteMenu(x, y, n) { menuNote.value = n; menu.value = { visible: true, x, y, kind: 'note' } }
function onFolderContext({ node, event }) {
  menuFolder.value = node
  menu.value = { visible: true, x: event.clientX, y: event.clientY, kind: 'folder' }
}
function openEmptyMenu(e) { menu.value = { visible: true, x: e.clientX, y: e.clientY, kind: 'empty' } }
function openSpecialMenu(key, e) {
  menuSpecial.value = key
  menu.value = { visible: true, x: e.clientX, y: e.clientY, kind: 'special' }
}

// ── Модели пунктов (сгруппированы в подменю, чтобы меню не было перегружено) ──
const menuItems = computed(() => {
  if (menu.value.kind === 'folder') return folderMenuItems(menuFolder.value)
  if (menu.value.kind === 'empty') return emptyMenuItems()
  if (menu.value.kind === 'special') return specialMenuItems()
  return noteMenuItems(menuNote.value)
})

function specialMenuItems() {
  return [
    { label: 'Открыть', icon: 'open_in_new', action: 'open' },
    { label: 'Экспорт (.zip)', icon: 'folder_zip', children: [
      { label: 'Заметки как .txt', icon: 'description', action: 'zip-txt' },
      { label: 'Заметки как .docx', icon: 'article', action: 'zip-docx' },
    ] },
  ]
}

function noteMenuItems(n) {
  if (!n) return []
  const dl = { label: 'Скачать', icon: 'download', children: [
    { label: 'Формат .txt', icon: 'description', action: 'export-txt' },
    { label: 'Формат .docx', icon: 'article', action: 'export-docx' },
  ] }
  if (isShared(n)) return [{ label: 'Открыть', icon: 'edit_note', action: 'open' }, dl]
  return [
    { label: 'Открыть', icon: 'edit_note', action: 'open' },
    { label: 'Организация', icon: 'category', children: [
      { label: 'Теги', icon: 'sell', action: 'tags' },
      { label: 'Переместить', icon: 'drive_file_move', action: 'move' },
      { label: 'Скопировать', icon: 'content_copy', action: 'copy' },
      { label: n.pinned_at ? 'Открепить' : 'Закрепить', icon: n.pinned_at ? 'keep_off' : 'keep', action: 'pin' },
    ] },
    { label: 'Поделиться', icon: 'share', children: [
      { label: 'Настроить доступ', icon: 'group_add', action: 'share' },
      { label: 'Отправить в чат', icon: 'send', action: 'send-chat' },
      ...(hasCompany.value ? [{ label: 'На портал', icon: 'campaign', action: 'publish' }] : []),
    ] },
    dl,
    { label: n.archived ? 'Вернуть из архива' : 'В архив', icon: n.archived ? 'unarchive' : 'archive', action: 'archive' },
    { label: 'Удалить', icon: 'delete', action: 'delete', danger: true, divider: true },
  ]
}

function folderMenuItems(f) {
  if (!f) return []
  if (f.owner_id !== myId.value) return [{ label: 'Открыть', icon: 'folder_open', action: 'open' }]
  return [
    { label: 'Открыть', icon: 'folder_open', action: 'open' },
    { label: 'Создать внутри', icon: 'add', children: [
      { label: 'Заметку', icon: 'note_add', action: 'newnote' },
      { label: 'Подпапку', icon: 'create_new_folder', action: 'newsub' },
    ] },
    { label: 'Организация', icon: 'category', children: [
      { label: 'Переименовать', icon: 'drive_file_rename_outline', action: 'rename' },
      { label: 'Переместить', icon: 'drive_file_move', action: 'move' },
      { label: 'Скопировать', icon: 'content_copy', action: 'copy' },
    ] },
    { label: 'Поделиться', icon: 'share', action: 'share' },
    { label: 'Скачать (.zip)', icon: 'folder_zip', children: [
      { label: 'Заметки как .txt', icon: 'description', action: 'zip-txt' },
      { label: 'Заметки как .docx', icon: 'article', action: 'zip-docx' },
    ] },
    { label: 'Удалить', icon: 'delete', action: 'delete', danger: true, divider: true },
  ]
}

function emptyMenuItems() {
  return [
    { label: 'Новая заметка', icon: 'note_add', action: 'note' },
    { label: 'Новая папка', icon: 'create_new_folder', action: 'folder' },
    { label: 'Импорт файла', icon: 'upload_file', action: 'import' },
  ]
}

function onMenuSelect(action) {
  if (menu.value.kind === 'note') onNoteMenuAction(action)
  else if (menu.value.kind === 'folder') fmAction(action)
  else if (menu.value.kind === 'special') smAction(action)
  else emAction(action)
}

// Действия меню особой группировки (Все/Поделились/Архив).
function smAction(action) {
  const key = menuSpecial.value
  if (action === 'open') onSpecialClick(key)
  else if (action === 'zip-txt') downloadScope(key, 'txt')
  else if (action === 'zip-docx') downloadScope(key, 'docx')
}
function downloadScope(key, format) {
  const label = specialGroups.find((g) => g.key === key)?.label || 'Заметки'
  download(api.exportScope(key, format), label, 'zip')
}

function onNoteMenuAction(action) {
  const n = menuNote.value
  if (!n) return
  if (action === 'open') openNote(n)
  else if (action === 'tags') tagsOpen.value = true
  else if (action === 'move') moveItem('note', n.id)
  else if (action === 'copy') copyNote(n)
  else if (action === 'share') shareItem('note', n)
  else if (action === 'send-chat') sendChatOpen.value = true
  else if (action === 'publish') publishToPortal(n)
  else if (action === 'pin') togglePin(n)
  else if (action === 'export-txt') downloadNote(n, 'txt')
  else if (action === 'export-docx') downloadNote(n, 'docx')
  else if (action === 'archive') toggleArchive(n)
  else if (action === 'delete') {
    confirmDelete.value = {
      header: 'Удалить заметку?', message: `«${n.title || 'Без названия'}» будет удалена навсегда.`,
      run: async () => { confirmDelete.value = null; try { await store.removeNote(n.id) } catch (e) { notif.error(e?.message) } },
    }
  }
}
// Цель перекраски в шапке меню: своя заметка или своя папка.
const menuColorTarget = computed(() => {
  if (menu.value.kind === 'note' && menuNote.value && !isShared(menuNote.value)) return menuNote.value
  if (menu.value.kind === 'folder' && menuFolder.value && menuFolder.value.owner_id === myId.value) return menuFolder.value
  return null
})
function onPickColor(color) {
  const t = menuColorTarget.value
  const kind = menu.value.kind
  menu.value.visible = false
  if (!t) return
  const p = kind === 'folder'
    ? store.renameFolder(t.id, t.name, color)
    : store.setNoteColor(t.id, color)
  p.catch((e) => notif.error(e?.message || 'Не удалось изменить цвет'))
}

// Гард: закрытие меню тапом по свободному месту не должно «проваливаться» в
// клик по заметке/папке под меню (особенно на мобильных).
let menuClosedAt = 0
function onMenuClose() { menu.value.visible = false; menuClosedAt = Date.now() }
function menuJustClosed() { return Date.now() - menuClosedAt < 400 }
function onTagsSaved() { store.fetchTags() }
async function togglePin(n) { try { await store.setPinned(n.id, !n.pinned_at) } catch (e) { notif.error(e?.message) } }
async function toggleArchive(n) { try { await store.setArchived(n.id, !n.archived) } catch (e) { notif.error(e?.message) } }
async function publishToPortal(n) {
  try {
    const full = await api.getNote(n.id)
    postPreset.value = { title: full.title || '', body: docToMarkdown(full.doc) }
    postComposerOpen.value = true
  } catch (e) { notif.error(e?.message || 'Не удалось открыть заметку') }
}

function fmAction(action) {
  const f = menuFolder.value
  if (!f) return
  if (action === 'open') store.openFolder(f)
  else if (action === 'rename') renameFolderDlg(f)
  else if (action === 'share') shareItem('folder', f)
  else if (action === 'copy') copyFolder(f)
  else if (action === 'move') moveItem('folder', f.id)
  else if (action === 'newsub') newFolder(f.id)
  else if (action === 'newnote') createNoteInFolder(f.id)
  else if (action === 'zip-txt') downloadFolder(f, 'txt')
  else if (action === 'zip-docx') downloadFolder(f, 'docx')
  else if (action === 'delete') {
    confirmDelete.value = {
      header: 'Удалить папку?',
      message: `Папка «${f.name}» будет удалена. Вложенное переедет на уровень выше.`,
      run: async () => { confirmDelete.value = null; try { await store.removeFolder(f.id) } catch (e) { notif.error(e?.message) } },
    }
  }
}
function emAction(action) {
  if (action === 'note') createAndOpen()
  else if (action === 'folder') newFolder()
  else if (action === 'import') importInput.value?.click()
}

// Создать заметку прямо в папке, от которой был вызван правый клик (не в
// выбранной), и открыть.
async function createNoteInFolder(folderId) {
  try {
    const n = await store.createNote('', folderId)
    router.push(`/notes/${n.id}`)
  } catch (e) { notif.error(e?.message || 'Не удалось создать заметку') }
}

// ── Дерево: клики/раскрытие ──
// Повторный клик по уже выбранной папке снимает выбор → «Все заметки»
// (иначе новые заметки/папки продолжали бы создаваться внутри неё).
function onTreeSelect(node) {
  if (store.activeFolderId === node.id && !store.showShared && !store.showArchived) store.selectAll()
  else store.selectFolder(node.id)
}
function onTreeToggle(id) {
  const s = new Set(expanded.value)
  if (s.has(id)) s.delete(id); else s.add(id)
  expanded.value = s
}

// ── Крошки ──
function onCrumb(index) { store.navigateTo(index) }
function onCrumbDrop(index) {
  const target = index < 0 ? null : store.path[index]?.id ?? null
  performDrop(target)
}

// ── Drag & drop заметок/папок ──
const dropFolderId = ref(null)
function onNoteDragStart(n, e) {
  if (isShared(n)) { e.preventDefault(); return }
  startDrag('note', n.id, n.title)
  try { e.dataTransfer.effectAllowed = 'move'; e.dataTransfer.setData('text/plain', `note:${n.id}`) } catch { /* Safari */ }
}
function onFolderDragStart(f, e) {
  if (f.owner_id !== myId.value) { if (e?.preventDefault) e.preventDefault(); return }
  startDrag('folder', f.id, f.name)
  try { e.dataTransfer.effectAllowed = 'move'; e.dataTransfer.setData('text/plain', `folder:${f.id}`) } catch { /* Safari */ }
}
function dragEnd() { endDrag(); dropFolderId.value = null }
function onFolderCardDrop(f) { dropFolderId.value = null; performDrop(f.id) }
function onTreeDrop(node) { performDrop(node.id) }
async function performDrop(targetFolderId) {
  const item = dragItem.value
  endDrag()
  if (!item) return
  try {
    if (item.kind === 'note') await store.moveNote(item.id, targetFolderId)
    else if (item.kind === 'folder' && item.id !== targetFolderId) await store.moveFolder(item.id, targetFolderId)
  } catch (e) { notif.error(e?.message || 'Не удалось переместить') }
}

// ── Импорт: кнопка + drag&drop файлов ──
const importInput = ref(null)
async function onImportPick(e) {
  const files = [...(e.target.files || [])]
  e.target.value = ''
  await importFiles(files)
}
async function importFiles(files) {
  let last = null
  for (const file of files) {
    try { last = await store.importNote(file) }
    catch (err) { notif.error(err?.message || `Не удалось импортировать ${file.name}`) }
  }
  if (files.length === 1 && last) router.push(`/notes/${last.id}`)
  else if (last) notif.success(`Импортировано: ${files.length}`)
}

const importDragging = ref(false)
let dragDepth = 0
function hasFiles(e) { return [...(e.dataTransfer?.types || [])].includes('Files') }
function onFileDragOver(e) {
  if (!hasFiles(e)) return
  e.preventDefault()
  importDragging.value = true
}
function onFileDragLeave(e) {
  if (!hasFiles(e)) return
  dragDepth--
  if (dragDepth <= 0) { importDragging.value = false; dragDepth = 0 }
}
function onFileDrop(e) {
  if (!hasFiles(e)) return
  e.preventDefault()
  importDragging.value = false
  dragDepth = 0
  importFiles([...(e.dataTransfer.files || [])].filter((f) => /\.(txt|docx)$/i.test(f.name)))
}
</script>

<style scoped>
.na { display: flex; flex-direction: column; height: 100%; min-height: 0; position: relative; }

/* ── Тулбар ── */
.na-toolbar { flex-shrink: 0; padding: 12px 16px 8px; display: flex; flex-direction: column; gap: 8px; }
.na-hub { padding-bottom: 4px; }
.na-toolbar-row { display: flex; align-items: center; gap: 12px; }
.na-toolbar-row :deep(.search-field) { flex: 1; min-width: 0; }
.na-actions { display: flex; align-items: center; gap: 8px; }
.na-modes { display: inline-flex; background: var(--color-surface-high); border-radius: var(--radius-full); padding: 3px; }
.na-mode { width: 36px; height: 32px; display: grid; place-items: center; border: none; background: transparent; color: var(--color-text-dim); border-radius: var(--radius-full); cursor: pointer; }
.na-mode.active { background: var(--color-surface); color: var(--color-primary); box-shadow: var(--shadow-sm); }
.na-mode .material-symbols-outlined { font-size: 20px; }

/* ── Тулбар выделения ── */
.na-selbar { display: flex; align-items: center; gap: 10px; padding: 8px 12px; background: var(--color-primary-container); border-radius: var(--radius-md); }
.na-selclose { width: 30px; height: 30px; display: grid; place-items: center; border: none; border-radius: 50%; background: transparent; color: var(--color-on-primary-container); cursor: pointer; }
.na-selcount { font-size: 13.5px; font-weight: 700; color: var(--color-on-primary-container); }
.na-selactions { margin-left: auto; display: flex; gap: 4px; }
.na-selbtn { width: 36px; height: 36px; display: grid; place-items: center; border: none; border-radius: var(--radius-md); background: transparent; color: var(--color-on-primary-container); cursor: pointer; }
.na-selbtn:hover { background: color-mix(in oklch, var(--color-on-primary-container) 12%, transparent); }
.na-selbtn.danger { color: var(--color-error); }
.na-selbtn .material-symbols-outlined { font-size: 20px; }

/* ── Тело ── */
.na-body { flex: 1; min-height: 0; display: flex; }
.na-side { width: 268px; flex-shrink: 0; border-right: 1px solid var(--color-outline-dim); display: flex; flex-direction: column; min-height: 0; }
.na-side-scroll { flex: 1; overflow-y: auto; padding: 8px 10px 16px; }
.na-nav { width: 100%; display: flex; align-items: center; gap: 10px; padding: 9px 10px; border: none; border-radius: var(--radius-md); background: transparent; color: var(--color-text); font: inherit; font-size: 14px; font-weight: 600; cursor: pointer; }
.na-nav:hover { background: var(--color-surface-high); }
.na-nav.active { background: color-mix(in oklch, var(--color-primary) 14%, transparent); color: var(--color-primary); }
.na-nav .material-symbols-outlined { font-size: 20px; }
.na-side-head { display: flex; align-items: center; justify-content: space-between; padding: 14px 8px 6px; font-size: 11.5px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em; color: var(--color-text-dim); }
.na-side-add { width: 24px; height: 24px; display: grid; place-items: center; border: none; border-radius: 50%; background: transparent; color: var(--color-text-dim); cursor: pointer; }
.na-side-add:hover { background: var(--color-surface-high); color: var(--color-primary); }

.na-main { flex: 1; min-width: 0; display: flex; flex-direction: column; min-height: 0; }
.na-crumbs { flex-shrink: 0; padding: 10px 16px 4px; }
/* Одна строка: подпись слева и кнопки справа зафиксированы, чипы прокручиваются
   горизонтально между ними (паттерн chip-carousel, как в Google Maps). */
.na-tags { flex-shrink: 0; display: flex; flex-wrap: nowrap; align-items: center; gap: 6px; padding: 8px 16px; min-width: 0; }
.na-tags-label { flex-shrink: 0; display: inline-flex; align-items: center; gap: 4px; font-size: 12.5px; font-weight: 700; color: var(--color-text-dim); }
.na-tags-label .material-symbols-outlined { font-size: 16px; }
.na-tags-scroll { flex: 1; min-width: 0; display: flex; flex-wrap: nowrap; align-items: center; gap: 6px; overflow-x: auto; scrollbar-width: none; -webkit-overflow-scrolling: touch; padding: 3px 6px; scroll-padding-inline: 6px; }
.na-tags-scroll::-webkit-scrollbar { display: none; }
.na-tags-scroll .na-tag { flex-shrink: 0; }
.na-tags-empty { font-size: 12.5px; color: var(--color-text-dim); opacity: 0.8; white-space: nowrap; }
.na-tag-clear { flex-shrink: 0; color: var(--color-error); opacity: 1; }
.na-tag-manage { flex-shrink: 0; }
.na-tag-manage-label { margin-left: 2px; }
.na-tag { display: inline-flex; align-items: center; gap: 4px; height: 30px; padding: 0 10px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--color-surface); color: var(--color-text-dim); font: inherit; font-size: 12.5px; font-weight: 600; cursor: pointer; opacity: 0.7; }
.na-tag.active { opacity: 1; box-shadow: 0 0 0 1.5px currentColor; }
.na-tag .material-symbols-outlined { font-size: 15px; }
.na-tag-manage { color: var(--color-text-dim); opacity: 1; }

.na-scroll { flex: 1; min-height: 0; overflow-y: auto; padding: 6px 16px 16px; }
.na-note { padding: 12px; color: var(--color-text-dim); }
.na-empty { margin-top: 24px; }

/* ── Папки-плитки ── */
.na-folders { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 10px; margin-bottom: 14px; }
.na-fcard { display: flex; align-items: center; gap: 10px; padding: 12px 14px; border: 1px solid var(--acrylic-border); border-radius: 14px; background: var(--acrylic-card-bg); cursor: pointer; user-select: none; position: relative; }
.na-fcard:hover { background: var(--color-surface-high); }
.na-fcard.selected { box-shadow: 0 0 0 2px var(--color-primary); }
.na-fcard.na-drop { box-shadow: inset 0 0 0 2px var(--color-primary); background: color-mix(in oklch, var(--color-primary) 10%, transparent); }
.na-fcard-ic { font-size: 26px; color: var(--fc-accent, var(--color-primary)); flex-shrink: 0; }
.na-fcard.shared .na-fcard-ic { color: var(--color-tertiary); }
.na-fcard-body { display: flex; flex-direction: column; min-width: 0; }
.na-fcard-name { font-size: 14px; font-weight: 700; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.na-fcard-sub { font-size: 12px; color: var(--color-text-dim); }
.na-fcard-badge { position: absolute; top: 8px; right: 8px; font-size: 15px; color: var(--color-tertiary); }

/* Особые группировки (Все/Поделились/Архив) — тонированные, без действий. */
.na-specials { margin-bottom: 10px; }
.na-special { background: color-mix(in oklch, var(--color-primary) 8%, var(--acrylic-card-bg)); border-color: color-mix(in oklch, var(--color-primary) 22%, transparent); }
.na-special:hover { background: color-mix(in oklch, var(--color-primary) 14%, var(--acrylic-card-bg)); }
.na-special .na-fcard-ic { color: var(--color-primary); }

/* ── Плитки заметок ── */
.na-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(230px, 1fr)); gap: 12px; align-content: start; }
.na-card { display: flex; flex-direction: column; gap: 6px; padding: 14px 16px; background: var(--glass-bg); box-shadow: var(--glass-edge); border: 1px solid var(--acrylic-border); border-radius: 18px; cursor: pointer; user-select: none; position: relative; }
.na-card.selected { box-shadow: 0 0 0 2px var(--color-primary); }
.na-card.shared { border-style: dashed; }
.na-card-pin { position: absolute; top: 10px; right: 10px; font-size: 17px; color: var(--color-tertiary); font-variation-settings: 'FILL' 1; }
.na-card-shared { position: absolute; top: 10px; right: 10px; font-size: 15px; color: var(--color-tertiary); }
.na-card-title { margin: 0; font-size: 15px; font-weight: 700; color: var(--color-text); line-height: 1.3; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.na-card-excerpt { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.45; display: -webkit-box; -webkit-line-clamp: 3; -webkit-box-orient: vertical; overflow: hidden; word-break: break-word; }
.na-card-excerpt.dim { font-style: italic; opacity: 0.7; }
.na-card-tags { display: flex; flex-wrap: wrap; gap: 4px; }
.na-card-tag { padding: 1px 8px; border-radius: var(--radius-full); border: 1px solid var(--color-outline-dim); font-size: 11px; font-weight: 700; }
.na-card-foot { margin-top: auto; padding-top: 6px; display: inline-flex; align-items: center; gap: 6px; font-size: 12px; color: var(--color-text-dim); }
.na-card-foot .material-symbols-outlined { font-size: 15px; }
.na-card-owner-av { width: 18px; height: 18px; border-radius: 50%; object-fit: cover; }
.na-card-owner { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.na-card-access { margin-left: auto; flex-shrink: 0; padding: 1px 8px; border-radius: var(--radius-full); font-size: 11px; font-weight: 700; background: var(--color-surface-high); border: 1px solid var(--color-outline-dim); }
.na-card-access.edit { background: var(--color-primary-container); border-color: transparent; color: var(--color-on-primary-container); }

/* ── Меню папки ── */
/* Палитра цветов в шапке контекстного меню заметки/папки. */
.na-menu-colors { padding: 8px 10px 10px; border-bottom: 1px solid var(--color-outline-dim); margin-bottom: 4px; }

/* ── Оверлей импорта ── */
.na-dropzone { position: absolute; inset: 0; z-index: 50; display: grid; place-items: center; background: color-mix(in oklch, var(--color-primary) 12%, transparent); backdrop-filter: blur(2px); pointer-events: none; }
.na-dropzone-inner { display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 32px 48px; border: 2px dashed var(--color-primary); border-radius: 20px; background: var(--color-surface); color: var(--color-primary); }
.na-dropzone-inner .material-symbols-outlined { font-size: 44px; }
.na-dropzone-inner p { margin: 0; font-weight: 700; }

@media (max-width: 768px) {
  .na-toolbar { padding: 10px 12px 6px; }
  .na-lbl { display: none; }
  .na-actions .btn-grad { display: none; }
  /* Мобайл: папки и заметки — единым одноколоночным списком (не ломает вёрстку). */
  .na-grid { grid-template-columns: 1fr; }
  .na-folders { grid-template-columns: 1fr; gap: 12px; margin-bottom: 12px; }
  .na-scroll { padding: 6px 12px; padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px)); }
}
</style>
