<template>
  <nav
    ref="listEl"
    class="chat-folders"
    :class="[orientation, { 'can-left': canLeft, 'can-right': canRight }]"
    role="tablist"
    aria-label="Папки чатов"
    @scroll.passive="updateEdges"
  >
    <button
      v-for="tab in tabs"
      :key="tab.token"
      class="cf-item"
      :ref="el => tab.id === messenger.activeFolderId && (activeEl = el)"
      :class="{
        active: tab.id === messenger.activeFolderId,
        dragging: dragToken != null && dragToken === tab.token,
        over: overToken != null && overToken === tab.token && dragToken !== tab.token,
      }"
      role="tab"
      :aria-selected="tab.id === messenger.activeFolderId"
      draggable="true"
      @click="messenger.setActiveFolder(tab.id)"
      @dragstart="onDragStart(tab, $event)"
      @dragover="onDragOver(tab, $event)"
      @drop="onDrop(tab)"
      @dragend="onDragEnd"
    >
      <span class="cf-ico-wrap">
        <EmojiGlyph v-if="tab.emoji" :char="tab.emoji" class="cf-emoji" />
        <span v-else class="material-symbols-outlined cf-ico">{{ tab.icon }}</span>
        <span v-if="tab.unread" class="cf-badge">{{ tab.unread > 99 ? '99+' : tab.unread }}</span>
      </span>
      <span class="cf-label">{{ tab.title }}</span>
    </button>
  </nav>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { useMessengerStore } from '@/stores/messenger.js'

const props = defineProps({
  orientation: { type: String, default: 'vertical' }, // 'vertical' | 'horizontal'
})

const messenger = useMessengerStore()

// Позиция виртуальной вкладки «Все чаты» среди папок — персональная настройка
// устройства (в БД её нет; сами папки синхронизируются порядком через сервер).
const ALL_POS_KEY = 'gw_chat_folders_all_pos'
function readAllPos() {
  try {
    const v = parseInt(localStorage.getItem(ALL_POS_KEY) ?? '0', 10)
    return Number.isFinite(v) && v >= 0 ? v : 0
  } catch { return 0 }
}
const allChatsIndex = ref(readAllPos())
function setAllChatsIndex(i) {
  allChatsIndex.value = i
  try { localStorage.setItem(ALL_POS_KEY, String(i)) } catch { /* no-op */ }
}

// token — стабильный идентификатор вкладки для DnD: 'all' у «Все чаты», id у папки.
const ALL = 'all'
const tabs = computed(() => {
  const list = messenger.folders.map(f => ({
    token: f.id,
    id: f.id,
    title: f.title,
    icon: 'folder',
    emoji: f.emoji || '',
    unread: messenger.folderUnread(f.id),
  }))
  const at = Math.min(allChatsIndex.value, list.length)
  list.splice(at, 0, {
    token: ALL, id: null, title: 'Все чаты', icon: 'forum', emoji: '',
    unread: messenger.folderUnread(null),
  })
  return list
})

/* ── Подсказка прокрутки (горизонтальная лента на телефоне) ──
   Папки не влезали в строку, а признака этого не было: лента обрывалась ровно
   по кромке панели и читалась как полный список. Затенение у края показывает,
   что справа/слева есть ещё, — и оно честное: считается по фактической
   прокрутке, а не рисуется всегда. */
const listEl = ref(null)
const activeEl = ref(null)
const canLeft = ref(false)
const canRight = ref(false)

function updateEdges() {
  const el = listEl.value
  if (!el || props.orientation !== 'horizontal') return
  canLeft.value = el.scrollLeft > 4
  canRight.value = el.scrollWidth - el.clientWidth - el.scrollLeft > 4
}

let ro = null
onMounted(() => {
  updateEdges()
  if (typeof ResizeObserver === 'undefined' || !listEl.value) return
  ro = new ResizeObserver(updateEdges)
  ro.observe(listEl.value)
})
onBeforeUnmount(() => ro?.disconnect())

// Активная папка всегда на виду: выбор из меню или переход по ссылке могли
// оставить её за краем ленты.
watch(() => [messenger.activeFolderId, tabs.value.length], async () => {
  await nextTick()
  activeEl.value?.scrollIntoView?.({ block: 'nearest', inline: 'nearest' })
  updateEdges()
})

// ── Перетаскивание вкладок для смены порядка (вкл. «Все чаты») ──
const dragToken = ref(null)
const overToken = ref(null)

function onDragStart(tab, e) {
  dragToken.value = tab.token
  e.dataTransfer.effectAllowed = 'move'
  try { e.dataTransfer.setData('text/plain', String(tab.token)) } catch { /* Safari */ }
}
function onDragOver(tab, e) {
  if (dragToken.value == null) return
  e.preventDefault()
  overToken.value = tab.token
}
function onDrop(tab) {
  const from = dragToken.value
  const to = tab.token
  if (from == null || from === to) return
  const tokens = tabs.value.map(t => t.token)
  const fromIdx = tokens.indexOf(from)
  const toIdx = tokens.indexOf(to)
  if (fromIdx === -1 || toIdx === -1) return
  tokens.splice(fromIdx, 1)
  tokens.splice(toIdx, 0, from)
  // Запоминаем позицию «Все чаты» и переупорядочиваем реальные папки.
  setAllChatsIndex(tokens.indexOf(ALL))
  const folderIds = tokens.filter(t => t !== ALL)
  if (folderIds.length) messenger.reorderFoldersAction(folderIds).catch(() => {})
}
function onDragEnd() {
  dragToken.value = null
  overToken.value = null
}
</script>

<style scoped>
.chat-folders {
  display: flex;
  gap: 2px;
}

/* ── Вертикальный рейл (десктоп) ── */
.chat-folders.vertical {
  flex-direction: column;
  width: 76px;
  flex-shrink: 0;
  padding: 8px 6px;
  overflow-y: auto;
  border-right: 1px solid var(--acrylic-border);
}

/* ── Горизонтальные табы (мобила / мини-хаб) ── */
.chat-folders.horizontal {
  flex-direction: row;
  overflow-x: auto;
  padding: 2px 6px 4px;
  scrollbar-width: none;
}
.chat-folders.horizontal::-webkit-scrollbar { display: none; }

/* Затенение у края = «лента продолжается». Маска работает по альфе, цвет в ней
   не виден — поэтому здесь чёрный, а не токен темы. */
.chat-folders.horizontal.can-right {
  -webkit-mask-image: linear-gradient(to right, #000 calc(100% - 32px), transparent);
  mask-image: linear-gradient(to right, #000 calc(100% - 32px), transparent);
}
.chat-folders.horizontal.can-left {
  -webkit-mask-image: linear-gradient(to right, transparent, #000 32px);
  mask-image: linear-gradient(to right, transparent, #000 32px);
}
.chat-folders.horizontal.can-left.can-right {
  -webkit-mask-image: linear-gradient(to right, transparent, #000 32px, #000 calc(100% - 32px), transparent);
  mask-image: linear-gradient(to right, transparent, #000 32px, #000 calc(100% - 32px), transparent);
}

.cf-item {
  display: flex;
  align-items: center;
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  border-radius: var(--radius-md);
  transition: background 0.15s, color 0.15s;
}

/* Перетаскивание порядка папок. */
.cf-item.dragging { opacity: 0.4; }
.chat-folders.vertical .cf-item.over {
  box-shadow: inset 0 2px 0 0 var(--color-primary);
}
.chat-folders.horizontal .cf-item.over {
  box-shadow: inset 2px 0 0 0 var(--color-primary);
}

.chat-folders.vertical .cf-item {
  flex-direction: column;
  gap: 3px;
  padding: 8px 4px;
  min-height: 0;
}

.chat-folders.horizontal .cf-item {
  flex-direction: row;
  gap: 6px;
  padding: 7px 14px;
  min-height: 0;
  flex-shrink: 0;
  white-space: nowrap;
}

.cf-item:hover { background: var(--glass-bg); color: var(--color-text); }

.cf-item.active { color: var(--color-primary); }
.chat-folders.vertical .cf-item.active {
  background: color-mix(in oklch, var(--color-primary-container) 45%, transparent);
}
.chat-folders.horizontal .cf-item.active {
  background: color-mix(in oklch, var(--color-primary-container) 55%, transparent);
  color: var(--color-on-primary-container);
}

.cf-ico-wrap { position: relative; display: grid; place-items: center; }
.cf-ico { font-size: 22px; }
.cf-emoji { font-size: 22px; line-height: 1; }
.chat-folders.horizontal .cf-ico,
.chat-folders.horizontal .cf-emoji { font-size: 18px; }

.cf-label {
  font-size: 11px;
  font-weight: 600;
  max-width: 64px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.chat-folders.horizontal .cf-label { font-size: 13.5px; max-width: 160px; }

.cf-item.active .cf-label { color: inherit; }

.cf-badge {
  position: absolute;
  top: -6px;
  right: -10px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 10px;
  font-weight: 700;
  line-height: 16px;
  text-align: center;
}
</style>
