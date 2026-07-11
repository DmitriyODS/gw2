<template>
  <div class="notes split-view">
    <!-- ЛЕВАЯ ПАНЕЛЬ: группы заметок -->
    <aside class="split-side">
      <!-- Вместо тайла раздела — переключатель хаба Заметки/Ежедневник -->
      <div class="split-side-head">
        <NotesHubTabs full-width />
      </div>

      <div class="split-side-list">
        <button
          class="split-side-item"
          :class="{ active: store.activeGroupId === 0 && !store.showArchived }"
          @click="store.selectGroup(0)"
        >
          <span class="split-item-tile"><span class="material-symbols-outlined">apps</span></span>
          <span class="split-side-name">Все</span>
          <span v-if="allCount != null" class="nt-count">{{ allCount }}</span>
        </button>

        <div v-if="store.loadingGroups && !store.groups.length" class="split-side-note">Загрузка…</div>
        <button
          v-for="g in store.groups"
          :key="g.id"
          class="split-side-item nt-group"
          :class="{ active: g.id === store.activeGroupId }"
          @click="store.selectGroup(g.id)"
        >
          <span class="split-item-tile"><span class="material-symbols-outlined">folder</span></span>
          <template v-if="editGroupId === g.id">
            <input
              ref="editInputEl"
              v-model="editGroupName"
              class="nt-group-input"
              maxlength="100"
              @click.stop
              @keydown.enter.prevent="saveGroupName(g)"
              @keydown.esc.stop="editGroupId = null"
              @blur="saveGroupName(g)"
            />
          </template>
          <template v-else>
            <span class="split-side-name">{{ g.name }}</span>
            <span class="nt-gactions" @click.stop>
              <span class="nt-gaction material-symbols-outlined" title="Переименовать" @click="startRename(g)">edit</span>
              <span class="nt-gaction danger material-symbols-outlined" title="Удалить группу" @click="askDeleteGroup(g)">delete</span>
            </span>
            <span class="nt-count">{{ g.notes_count }}</span>
          </template>
        </button>
      </div>

      <!-- Поделились и Архив — отдельные фильтры вне групп -->
      <div class="split-side-list nt-archive-slot">
        <button
          class="split-side-item"
          :class="{ active: store.showShared }"
          @click="store.selectShared()"
        >
          <span class="split-item-tile"><span class="material-symbols-outlined">group</span></span>
          <span class="split-side-name">Поделились</span>
        </button>
        <button
          class="split-side-item"
          :class="{ active: store.showArchived }"
          @click="store.selectArchive()"
        >
          <span class="split-item-tile"><span class="material-symbols-outlined">archive</span></span>
          <span class="split-side-name">Архив</span>
        </button>
      </div>

      <!-- Добавить группу: ghost-пункт → инлайн-инпут -->
      <form v-if="addingGroup" class="nt-addform" @submit.prevent="submitGroup">
        <input
          ref="addInputEl"
          v-model="newGroupName"
          class="nt-group-input"
          placeholder="Название группы"
          maxlength="100"
          @keydown.esc="addingGroup = false"
          @blur="cancelAddOnBlur"
        />
      </form>
      <button v-else class="split-side-add" @click="startAddGroup">
        <span class="material-symbols-outlined">create_new_folder</span>
        Добавить группу
      </button>
    </aside>

    <!-- ПРАВАЯ ПАНЕЛЬ: плитки-стикеры -->
    <section class="split-main">
      <!-- Мобайл: переключатель хаба + лента групп чипами (боковая панель скрыта) -->
      <div v-if="isMobile" class="nt-mobile-hub">
        <NotesHubTabs full-width />
      </div>
      <header class="nt-toolbar">
        <SearchField v-model="searchInput" placeholder="Поиск по заметкам…" hotkey />
        <div class="nt-actions">
          <!-- Мобайл: группы/фильтры — в шторке, кнопка в один ряд с поиском. -->
          <button
            v-if="isMobile"
            class="btn-glass nt-groups-btn"
            title="Группы и фильтры"
            aria-label="Группы и фильтры"
            @click="groupsSheetOpen = true"
          >
            <span class="material-symbols-outlined">folder_open</span>
            <span v-if="activeFilterName" class="nt-groups-dot" aria-hidden="true" />
          </button>
          <button class="btn-glass" title="Импортировать заметку из .txt" @click="importInput?.click()">
            <span class="material-symbols-outlined">upload_file</span>
            <span class="nt-btn-label">Импорт</span>
          </button>
          <input ref="importInput" type="file" accept=".txt,text/plain" hidden @change="onImportFile" />
          <button class="btn-grad" :disabled="creating" @click="createAndOpen">
            <span class="material-symbols-outlined">add</span>
            <span class="nt-btn-label">Заметка</span>
          </button>
        </div>
      </header>

      <AppDialog
        v-model="groupsSheetOpen"
        tone="primary"
        icon="folder_open"
        size="sm"
        mobile="sheet"
        title="Группы"
      >
        <div class="nt-groupsheet">
          <button
            class="nt-groupitem"
            :class="{ active: store.activeGroupId === 0 && !store.showArchived && !store.showShared }"
            @click="pickGroup(() => store.selectGroup(0))"
          >
            <span class="nt-groupitem-name">Все</span>
            <span v-if="store.activeGroupId === 0 && !store.showArchived && !store.showShared" class="material-symbols-outlined">check</span>
          </button>
          <template v-for="g in store.groups" :key="g.id">
            <form
              v-if="sheetRenamingId === g.id"
              class="nt-groupsheet-add"
              @submit.prevent="submitSheetRename(g)"
            >
              <input
                ref="sheetGroupInput"
                v-model="sheetGroupName"
                class="nt-groupsheet-input"
                placeholder="Название группы"
                maxlength="64"
              />
              <button class="nt-groupsheet-ok" type="submit" :disabled="!sheetGroupName.trim()">
                <span class="material-symbols-outlined">check</span>
              </button>
            </form>
            <button
              v-else
              class="nt-groupitem"
              :class="{ active: g.id === store.activeGroupId }"
              @click="pickGroup(() => store.selectGroup(g.id))"
            >
              <span class="nt-groupitem-name">{{ g.name }}</span>
              <span
                class="material-symbols-outlined nt-groupitem-edit"
                role="button"
                tabindex="0"
                title="Переименовать"
                @click.stop="startSheetRename(g)"
                @keydown.enter.stop="startSheetRename(g)"
              >edit</span>
              <span v-if="g.id === store.activeGroupId" class="material-symbols-outlined">check</span>
            </button>
          </template>
          <button class="nt-groupitem" :class="{ active: store.showShared }" @click="pickGroup(() => store.selectShared())">
            <span class="nt-groupitem-name">Поделились</span>
            <span v-if="store.showShared" class="material-symbols-outlined">check</span>
          </button>
          <button class="nt-groupitem" :class="{ active: store.showArchived }" @click="pickGroup(() => store.selectArchive())">
            <span class="nt-groupitem-name">Архив</span>
            <span v-if="store.showArchived" class="material-symbols-outlined">check</span>
          </button>

          <!-- Новая группа: инлайн-форма прямо в шторке. -->
          <form v-if="sheetAddingGroup" class="nt-groupsheet-add" @submit.prevent="submitSheetGroup">
            <input
              ref="sheetGroupInput"
              v-model="sheetGroupName"
              class="nt-groupsheet-input"
              placeholder="Название группы"
              maxlength="64"
            />
            <button class="nt-groupsheet-ok" type="submit" :disabled="!sheetGroupName.trim()">
              <span class="material-symbols-outlined">check</span>
            </button>
          </form>
          <button v-else class="nt-groupitem nt-groupitem-add" @click="startSheetGroup">
            <span class="material-symbols-outlined">create_new_folder</span>
            <span class="nt-groupitem-name">Добавить группу</span>
          </button>
        </div>
      </AppDialog>

      <div class="nt-body">
        <div v-if="store.loading && !store.notes.length" class="split-side-note">Загрузка…</div>

        <EmptyState
          v-else-if="!store.notes.length && store.search"
          class="split-empty" icon="search_off" tone="soft"
          title="Ничего не найдено"
          subtitle="Попробуйте изменить запрос — ищем по заголовку и тексту заметок."
        />
        <EmptyState
          v-else-if="!store.notes.length && store.showArchived"
          class="split-empty" icon="archive" tone="soft"
          title="Архив пуст"
          subtitle="Сюда попадают заметки, отправленные в архив из контекстного меню плитки."
        />
        <EmptyState
          v-else-if="!store.notes.length && store.activeGroupId !== 0"
          class="split-empty" icon="folder_open" tone="soft"
          title="В группе пусто"
          subtitle="Добавьте заметку в эту группу через её настройки — заметка может быть сразу в нескольких группах."
        />
        <EmptyState
          v-else-if="!store.notes.length"
          class="split-empty" icon="note_stack" tone="soft"
          title="Пока нет ни одной заметки"
          subtitle="Создайте первую — с форматированием, картинками и таблицами. Ей можно поделиться ссылкой или выгрузить в .txt."
        >
          <button class="btn-grad" type="button" @click="createAndOpen">
            <span class="material-symbols-outlined">add</span>
            Создать заметку
          </button>
        </EmptyState>

        <div v-else class="nt-grid">
          <article
            v-for="n in store.notes"
            :key="n.id"
            class="nt-card glass-hover"
            :class="{ colored: n.color }"
            :style="noteColorStyle(n)"
            role="link"
            tabindex="0"
            @click="onTileClick(n)"
            @keydown.enter="openNote(n)"
            @contextmenu.prevent="openMenu($event.clientX, $event.clientY, n)"
            @pointerdown="onTilePointerDown($event, n)"
            @pointermove="onTilePointerMove"
            @pointerup="onTilePointerUp"
            @pointercancel="onTilePointerUp"
          >
            <span v-if="n.pinned_at" class="nt-card-pin material-symbols-outlined" title="Закреплена">keep</span>
            <h3 class="nt-card-title">{{ n.title || 'Без названия' }}</h3>
            <p v-if="n.excerpt" class="nt-card-excerpt">{{ n.excerpt }}</p>
            <p v-else class="nt-card-excerpt dim">Пустая заметка</p>
            <footer class="nt-card-foot">
              <template v-if="store.showShared">
                <img class="nt-card-owner-avatar" :src="ownerAvatar(n)" :alt="n.owner_name || ''" />
                <span class="nt-card-owner">{{ n.owner_name || 'Владелец' }}</span>
                <span class="nt-card-access" :class="n.my_access">
                  {{ n.my_access === 'edit' ? 'редактирование' : 'просмотр' }}
                </span>
              </template>
              <template v-else>
                <span class="material-symbols-outlined">calendar_today</span>
                {{ formatDate(n.created_at) }}
              </template>
            </footer>
          </article>
        </div>
      </div>
    </section>

    <AppFab
      :visible="isMobile && fabVisible"
      icon="add"
      aria-label="Новая заметка"
      @click="createAndOpen"
    />

    <ConfirmDialog
      :visible="!!groupToDelete"
      header="Удалить группу?"
      :message="`Группа «${groupToDelete?.name}» будет удалена. Заметки останутся — они просто выйдут из этой группы.`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDeleteGroup"
      @cancel="groupToDelete = null"
    />

    <!-- Контекстное меню плитки (ПКМ / long-press) и его диалоги -->
    <NoteContextMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :color="menuNote?.color || ''"
      :archived="!!menuNote?.archived"
      :pinned="!!menuNote?.pinned_at"
      :can-post="hasCompany"
      :shared="store.showShared"
      @action="onMenuAction"
      @color="setNoteColor"
      @close="menu.visible = false"
    />
    <!-- Публикация целой заметки на портал: контент уходит статьёй (Markdown) -->
    <PostComposer
      v-if="postPreset"
      v-model="postComposerOpen"
      :preset="postPreset"
      @saved="notif.success('Опубликовано на портале')"
    />
    <!-- Целая заметка в чат: адресат получает доступ на просмотр + ссылку -->
    <NoteSendToChatDialog v-model="sendChatOpen" mode="note" :note="menuNote" />
    <NoteGroupsDialog
      v-model="groupsOpen"
      :note-id="menuNote?.id"
      :group-ids="menuNote?.group_ids || []"
      @saved="onGroupsSaved"
    />
    <NoteShareDialog v-model="shareOpen" :note-id="menuNote?.id" />
    <ConfirmDialog
      :visible="!!noteToDelete"
      header="Удалить заметку?"
      :message="`«${noteToDelete?.title || 'Без названия'}» будет удалена навсегда вместе с картинками. Ссылки на неё перестанут работать.`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDeleteNote"
      @cancel="noteToDelete = null"
    />
  </div>
</template>

<script setup>
import { computed, defineAsyncComponent, nextTick, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import NotesHubTabs from '@/components/notes/NotesHubTabs.vue'
import SearchField from '@/components/common/SearchField.vue'
import AppFab from '@/components/common/AppFab.vue'
import { useFabOnScroll } from '@/composables/useFabOnScroll.js'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import NoteContextMenu from '@/components/notes/NoteContextMenu.vue'
import NoteGroupsDialog from '@/components/notes/NoteGroupsDialog.vue'
import NoteShareDialog from '@/components/notes/NoteShareDialog.vue'
import NoteSendToChatDialog from '@/components/notes/NoteSendToChatDialog.vue'
import * as api from '@/api/notes.js'
import { docToMarkdown } from '@/utils/tiptapMarkdown.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

// Композер портала тяжёлый (стор портала) — грузим по первому использованию.
const PostComposer = defineAsyncComponent(() => import('@/components/portal/PostComposer.vue'))

const router = useRouter()
const store = useNotesStore()
const notif = useNotificationsStore()
const auth = useAuthStore()
const hasCompany = computed(() => !!auth.companyId)

const { isMobile } = useBreakpoint()
// Мобильный FAB «Новая заметка»: прячется/появляется по прокрутке плиток.
const { fabVisible } = useFabOnScroll()

/* Мобильная шторка групп/фильтров (заменяет ряд чипов над тулбаром). */
const groupsSheetOpen = ref(false)

const activeFilterName = computed(() => {
  if (store.showArchived) return 'Архив'
  if (store.showShared) return 'Поделились'
  if (store.activeGroupId) return store.groups.find(g => g.id === store.activeGroupId)?.name || ''
  return ''
})

function pickGroup(fn) {
  fn()
  groupsSheetOpen.value = false
}

/* Создание и переименование групп из мобильной шторки. */
const sheetAddingGroup = ref(false)
const sheetRenamingId = ref(null)
const sheetGroupName = ref('')
const sheetGroupInput = ref(null)

function focusSheetInput() {
  // ref в v-for может быть массивом.
  const el = Array.isArray(sheetGroupInput.value) ? sheetGroupInput.value[0] : sheetGroupInput.value
  el?.focus()
}

function startSheetGroup() {
  sheetRenamingId.value = null
  sheetAddingGroup.value = true
  sheetGroupName.value = ''
  nextTick(focusSheetInput)
}

function startSheetRename(g) {
  sheetAddingGroup.value = false
  sheetRenamingId.value = g.id
  sheetGroupName.value = g.name
  nextTick(focusSheetInput)
}

async function submitSheetGroup() {
  const name = sheetGroupName.value.trim()
  if (!name) return
  sheetAddingGroup.value = false
  try {
    await store.createGroup(name)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать группу')
  }
}

async function submitSheetRename(g) {
  const name = sheetGroupName.value.trim()
  sheetRenamingId.value = null
  if (!name || name === g.name) return
  try {
    await store.renameGroup(g.id, name)
  } catch (e) {
    notif.error(e?.message || 'Не удалось переименовать группу')
  }
}

onMounted(() => {
  store.fetchGroups()
  store.fetchNotes()
})

// Счётчик «Все» — сумма невозможна (заметка может быть в нескольких группах),
// показываем размер выборки только когда открыта вкладка «Все» без поиска.
const allCount = computed(() => store.totalCount)

// ── Поиск (дебаунс 300мс, серверный) ──
const searchInput = ref(store.search)
let searchTimer = null
watch(searchInput, (v) => {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(v.trim()), 300)
})

// ── Создание и открытие ──
const creating = ref(false)
async function createAndOpen() {
  creating.value = true
  try {
    const n = await store.createNote()
    router.push(`/notes/${n.id}`)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать заметку')
  } finally {
    creating.value = false
  }
}

function openNote(n) { router.push(`/notes/${n.id}`) }

// ── Контекстное меню плитки (ПКМ на десктопе, long-press на таче) ──
const menu = ref({ visible: false, x: 0, y: 0 })
const menuNote = ref(null)
const groupsOpen = ref(false)
const shareOpen = ref(false)
const noteToDelete = ref(null)

function openMenu(x, y, n) {
  // У чужих (шаренных) заметок меню ограничено (shared-prop): открыть/экспорт.
  menuNote.value = n
  menu.value = { visible: true, x, y }
}

function ownerAvatar(n) {
  return n.owner_avatar ? `/uploads/${n.owner_avatar}` : `/api/users/${n.owner_id}/identicon`
}

// Long-press 500мс: отменяется движением пальца (скролл); после срабатывания
// подавляем последующий click, чтобы заметка не открылась под меню.
let longPressTimer = null
let longPressFired = false
let pressStartX = 0
let pressStartY = 0

function onTilePointerDown(e, n) {
  if (e.pointerType === 'mouse') return // мышь — только ПКМ (contextmenu)
  longPressFired = false
  pressStartX = e.clientX
  pressStartY = e.clientY
  clearTimeout(longPressTimer)
  longPressTimer = setTimeout(() => {
    longPressFired = true
    if (navigator.vibrate) {
      try { navigator.vibrate(15) } catch { /* iOS Safari */ }
    }
    openMenu(e.clientX, e.clientY, n)
  }, 500)
}

function onTilePointerMove(e) {
  if (Math.abs(e.clientX - pressStartX) > 8 || Math.abs(e.clientY - pressStartY) > 8) {
    clearTimeout(longPressTimer)
  }
}

function onTilePointerUp() { clearTimeout(longPressTimer) }

function onTileClick(n) {
  if (longPressFired) { longPressFired = false; return }
  if (menu.value.visible) return // клик, закрывший меню, не открывает заметку
  openNote(n)
}

function onMenuAction(action) {
  const n = menuNote.value
  if (!n) return
  if (action === 'open') openNote(n)
  else if (action === 'groups') groupsOpen.value = true
  else if (action === 'share') shareOpen.value = true
  else if (action === 'send-chat') sendChatOpen.value = true
  else if (action === 'publish') publishToPortal(n)
  else if (action === 'pin') togglePin(n)
  else if (action === 'export') exportNoteTxt(n)
  else if (action === 'archive') toggleArchive(n)
  else if (action === 'delete') noteToDelete.value = n
}

// ── Публикация целой заметки на портал (статьёй, с форматированием) ──
const postComposerOpen = ref(false)
const postPreset = ref(null)
const sendChatOpen = ref(false)

async function publishToPortal(n) {
  try {
    const full = await api.getNote(n.id) // плитка без doc — тянем целиком
    postPreset.value = { title: full.title || '', body: docToMarkdown(full.doc) }
    postComposerOpen.value = true
  } catch (e) {
    notif.error(e?.message || 'Не удалось открыть заметку')
  }
}

async function togglePin(n) {
  try {
    await store.setPinned(n.id, !n.pinned_at)
  } catch (e) {
    notif.error(e?.message || 'Не удалось изменить закрепление')
  }
}

async function toggleArchive(n) {
  try {
    await store.setArchived(n.id, !n.archived)
  } catch (e) {
    notif.error(e.message || 'Не удалось изменить архив')
  }
}

function onGroupsSaved() {
  store.fetchGroups({ silent: true })
  store.fetchNotes({ silent: true })
}

// ── Цвет плитки (палитра тегов задач) ──
function noteColorStyle(n) {
  if (!n.color) return {}
  // Цвет — стеклянной тонировкой: «иней» поверх полупрозрачного пастельного
  // слоя (как у окрашенных карточек задач), стекло сохраняется.
  return {
    background: `var(--glass-bg), color-mix(in oklch, var(--tag-${n.color}-surface) 55%, transparent)`,
    borderColor: `var(--tag-${n.color}-border)`,
  }
}

async function setNoteColor(color) {
  const n = menuNote.value
  if (!n) return
  const prev = n.color || ''
  if (prev === color) return
  store.upsertNote({ ...n, color }) // оптимистично — палитра закрылась мгновенно
  try {
    const updated = await api.updateNote(n.id, { color })
    store.upsertNote(updated)
  } catch (e) {
    store.upsertNote({ ...n, color: prev })
    notif.error(e?.message || 'Не удалось изменить цвет')
  }
}

async function exportNoteTxt(n) {
  try {
    const resp = await api.exportNote(n.id)
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${(n.title || 'Заметка').slice(0, 100)}.txt`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    notif.error(e?.message || 'Не удалось экспортировать')
  }
}

async function confirmDeleteNote() {
  const n = noteToDelete.value
  noteToDelete.value = null
  if (!n) return
  try {
    await store.removeNote(n.id)
    notif.success('Заметка удалена')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить заметку')
  }
}

// ── Импорт .txt ──
const importInput = ref(null)
async function onImportFile(e) {
  const file = e.target.files?.[0]
  e.target.value = ''
  if (!file) return
  try {
    const n = await store.importNote(file)
    notif.success('Заметка импортирована')
    router.push(`/notes/${n.id}`)
  } catch (err) {
    notif.error(err?.message || 'Не удалось импортировать файл')
  }
}

// ── Группы: добавление ──
const addingGroup = ref(false)
const newGroupName = ref('')
const addInputEl = ref(null)

function startAddGroup() {
  addingGroup.value = true
  newGroupName.value = ''
  nextTick(() => addInputEl.value?.focus())
}

// Enter (submit) и blur инпута могут сработать ОБА на одно добавление (Chrome
// шлёт blur при удалении сфокусированного элемента, мобильная клавиатура — при
// сворачивании) — иначе группа создавалась бы дважды. Поэтому имя забираем и
// форму закрываем СИНХРОННО до запроса: повторный вызов увидит пустой инпут.
let groupSubmitting = false
async function submitGroup() {
  if (groupSubmitting) return
  const name = newGroupName.value.trim()
  newGroupName.value = ''
  addingGroup.value = false
  if (!name) return
  groupSubmitting = true
  try {
    await store.createGroup(name)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать группу')
  } finally {
    groupSubmitting = false
  }
}

function cancelAddOnBlur() {
  // Blur с текстом — сохраняем (как Enter), пустой — просто закрываем.
  if (newGroupName.value.trim()) submitGroup()
  else addingGroup.value = false
}

// ── Группы: переименование ──
const editGroupId = ref(null)
const editGroupName = ref('')
const editInputEl = ref(null)

function startRename(g) {
  editGroupId.value = g.id
  editGroupName.value = g.name
  nextTick(() => {
    const el = Array.isArray(editInputEl.value) ? editInputEl.value[0] : editInputEl.value
    el?.focus()
    el?.select()
  })
}

async function saveGroupName(g) {
  if (editGroupId.value !== g.id) return
  const name = editGroupName.value.trim()
  editGroupId.value = null
  if (!name || name === g.name) return
  try {
    await store.renameGroup(g.id, name)
  } catch (e) {
    notif.error(e?.message || 'Не удалось переименовать группу')
  }
}

// ── Группы: удаление ──
const groupToDelete = ref(null)
function askDeleteGroup(g) { groupToDelete.value = g }
async function confirmDeleteGroup() {
  const g = groupToDelete.value
  groupToDelete.value = null
  if (!g) return
  try {
    await store.removeGroup(g.id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить группу')
  }
}

// ── Формат даты плитки: dd.mm.yyyy HH:mm ──
function formatDate(iso) {
  const d = new Date(iso)
  const pad = (n) => String(n).padStart(2, '0')
  return `${pad(d.getDate())}.${pad(d.getMonth() + 1)}.${d.getFullYear()} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
</script>

<style scoped>
/* Пункт «Архив» — вне скролла групп, прижат к блоку «Добавить группу». */
.nt-archive-slot {
  flex: none;
  overflow: visible;
  border-top: 1px solid var(--color-outline-dim);
  margin-top: 4px;
  padding-top: 6px;
}

/* Счётчик пункта — тинтованная плашка (как .rail-badge). */
.nt-count {
  flex-shrink: 0;
  min-width: 22px;
  padding: 1px 6px;
  border-radius: var(--radius-sm);
  background: color-mix(in oklch, var(--color-primary) 14%, var(--color-surface));
  border: 1px solid color-mix(in oklch, var(--color-primary) 20%, transparent);
  color: var(--color-primary);
  font-size: 11.5px;
  font-weight: 700;
  text-align: center;
}
.split-side-item.active .nt-count {
  background: var(--color-primary);
  border-color: transparent;
  color: var(--color-on-primary);
}

/* Контекст-действия группы — видны по hover. */
.nt-gactions { display: none; align-items: center; gap: 2px; flex-shrink: 0; }
.nt-group:hover .nt-gactions { display: inline-flex; }
.nt-group:hover .nt-count { display: none; }
.nt-gaction {
  font-size: 17px;
  padding: 3px;
  border-radius: var(--radius-sm);
  color: var(--color-text-dim);
}
.nt-gaction:hover { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.nt-gaction.danger:hover { background: color-mix(in oklch, var(--color-error) 12%, transparent); color: var(--color-error); }

.nt-addform { padding: 8px 12px 12px; }
.nt-group-input {
  flex: 1;
  min-width: 0;
  width: 100%;
  height: 32px;
  padding: 0 10px;
  border: 1px solid color-mix(in oklch, var(--color-primary) 30%, transparent);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: 13.5px;
  font-weight: 600;
  outline: none;
}

/* ── Тулбар правой панели ── */
.nt-toolbar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px 10px;
  flex-shrink: 0;
}
/* Как в «Задачах»: поиск тянется на всю ширину, кнопки — сразу справа. */
.nt-toolbar :deep(.search-field) { flex: 1; min-width: 0; }
.nt-actions { display: flex; align-items: center; gap: 10px; }

.nt-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 4px 16px 16px;
  display: flex;
  flex-direction: column;
}

/* ── Сетка плиток-«стикеров» ── */
.nt-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
  gap: 12px;
  align-content: start;
}
.nt-card {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px 16px;
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--acrylic-border);
  border-radius: 18px;
  cursor: pointer;
  /* Long-press открывает контекстное меню — гасим выделение текста и
     системный callout iOS на удержании. */
  user-select: none;
  -webkit-user-select: none;
  -webkit-touch-callout: none;
}
/* Hover — глобальное «запотевание» .glass-hover (main.css). */
.nt-card:focus-visible { outline: 2px solid var(--color-primary); outline-offset: 2px; }

.nt-card { position: relative; }
.nt-card-pin {
  position: absolute;
  top: 10px;
  right: 10px;
  font-size: 17px;
  color: var(--color-tertiary);
  font-variation-settings: 'FILL' 1;
}

.nt-card-owner-avatar {
  width: 18px;
  height: 18px;
  border-radius: 50%;
  object-fit: cover;
}
.nt-card-owner {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nt-card-access {
  margin-left: auto;
  flex-shrink: 0;
  padding: 1px 8px;
  border-radius: var(--radius-full);
  font-size: 11px;
  font-weight: 700;
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-dim);
}
.nt-card-access.edit {
  background: var(--color-primary-container);
  border-color: transparent;
  color: var(--color-on-primary-container);
}

.nt-card-title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.3;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.nt-card-excerpt {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-dim);
  line-height: 1.45;
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
  word-break: break-word;
}
.nt-card-excerpt.dim { font-style: italic; opacity: 0.7; }
.nt-card-foot {
  margin-top: auto;
  padding-top: 6px;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-dim);
}
.nt-card-foot .material-symbols-outlined { font-size: 15px; }

/* ── Мобайл ── */
.nt-mobile-hub { flex-shrink: 0; padding: 10px 12px 0; }

/* Кнопка групп в тулбаре: точка — активный фильтр (не «Все»). */
.nt-groups-btn { position: relative; }
.nt-groups-dot {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
}

/* Шторка групп/фильтров. */
.nt-groupsheet {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.nt-groupitem {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 13px 14px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  font: inherit;
  font-size: 14.5px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.nt-groupitem.active {
  border-color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 10%, var(--acrylic-card-bg));
}
.nt-groupitem-name {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nt-groupitem .material-symbols-outlined { color: var(--color-primary); font-size: 20px; flex-shrink: 0; }

.nt-groupitem-edit {
  color: var(--color-text-dim) !important;
  padding: 6px;
  margin: -6px 0;
  border-radius: 50%;
}
.nt-groupitem-edit:hover { color: var(--color-text) !important; background: var(--color-surface-high); }

.nt-groupitem-add { color: var(--color-primary); border-style: dashed; }
.nt-groupitem-add .nt-groupitem-name { flex: none; }

/* Инлайн-форма создания/переименования группы в шторке. */
.nt-groupsheet-add {
  display: flex;
  align-items: center;
  gap: 8px;
}
.nt-groupsheet-input {
  flex: 1;
  min-width: 0;
  padding: 12px 14px;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 14.5px;
  outline: none;
}
.nt-groupsheet-ok {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-md);
  background: var(--color-primary);
  color: var(--color-on-primary);
  cursor: pointer;
}
.nt-groupsheet-ok:disabled { opacity: 0.5; cursor: not-allowed; }

@media (max-width: 768px) {
  .nt-grid { grid-template-columns: 1fr; }
  .nt-btn-label { display: none; }
  /* Создание заметки на мобильном — плавающий FAB, кнопка тулбара не нужна. */
  .nt-actions .btn-grad { display: none; }
  .nt-toolbar { padding: 12px 12px 8px; }
  /* Резерв под нижнюю навигацию (64px) + 12px воздуха — контент скроллится
     под стекло, последние плитки не прячутся за навигацией. */
  .nt-body {
    padding: 4px 12px;
    padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px));
  }
}
</style>
