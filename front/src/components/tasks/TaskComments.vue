<script setup>
import { ref, computed, onMounted, onBeforeUnmount, nextTick } from 'vue'
import { useTasksStore } from '@/stores/tasks.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { getDirectory } from '@/api/users.js'
import MarkdownView from '@/components/common/MarkdownView.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmployeeProfileDialog from '@/components/common/EmployeeProfileDialog.vue'

const props = defineProps({
  taskId: { type: Number, required: true },
})

const tasks = useTasksStore()
const auth = useAuthStore()
const notify = useNotificationsStore()
const { isAtLeast } = usePermission()

const loading = ref(false)
const draft = ref('')
const sending = ref(false)
const editingId = ref(null)
const editText = ref('')
const listEl = ref(null)
const deletingId = ref(null)

// ── @-упоминания: автокомплит из сотрудников компании ──
const mentionUsers = ref([])       // все члены активной компании (грузим 1 раз)
const mentionOpen = ref(false)
const mentionItems = ref([])       // отфильтрованные под текущий запрос
const mentionIndex = ref(0)
const mentionQuery = ref('')
const mentionStart = ref(0)        // индекс символа '@' в draft
const textareaRef = ref(null)

// login → ФИО: в чипах комментариев показываем имя, а не логин.
const mentionNames = computed(() => {
  const map = {}
  for (const u of mentionUsers.value) {
    if (u.login) map[u.login.toLowerCase()] = u.fio || u.login
  }
  return map
})

function avatarUrl(u) {
  return u?.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

// Пересчёт состояния автокомплита по позиции каретки: активен, если перед
// кареткой идёт @токен (в начале строки или после пробела), без пробелов внутри.
function updateMentionState() {
  const el = textareaRef.value
  if (!el) return closeMention()
  const pos = el.selectionStart ?? draft.value.length
  const before = draft.value.slice(0, pos)
  const m = before.match(/(?:^|\s)@([\p{L}\p{N}_.]*)$/u)
  if (!m) return closeMention()
  const query = m[1]
  mentionStart.value = pos - query.length - 1
  if (query !== mentionQuery.value) mentionIndex.value = 0
  mentionQuery.value = query
  const q = query.toLowerCase()
  mentionItems.value = mentionUsers.value
    .filter((u) => !q
      || (u.login || '').toLowerCase().includes(q)
      || (u.fio || '').toLowerCase().includes(q))
    .slice(0, 8)
  if (mentionIndex.value >= mentionItems.value.length) mentionIndex.value = 0
  mentionOpen.value = mentionItems.value.length > 0
}

function closeMention() {
  mentionOpen.value = false
  mentionItems.value = []
  mentionQuery.value = ''
}

function selectMention(u) {
  if (!u) return
  const el = textareaRef.value
  const pos = el ? (el.selectionStart ?? draft.value.length) : draft.value.length
  const start = mentionStart.value
  const insert = '@' + (u.login || '') + ' '
  draft.value = draft.value.slice(0, start) + insert + draft.value.slice(pos)
  closeMention()
  nextTick(() => {
    if (!el) return
    const caret = start + insert.length
    el.focus()
    el.setSelectionRange(caret, caret)
  })
}

// keyup для перемещения каретки (стрелки/клик/Home/End), кроме навигации по
// списку — её обрабатывает onKeydown с preventDefault.
function onCaretKeyup(e) {
  if (mentionOpen.value && ['ArrowDown', 'ArrowUp', 'Enter', 'Tab', 'Escape'].includes(e.key)) return
  updateMentionState()
}

function onInputBlur() {
  // Отложенно — чтобы успел отработать клик по элементу списка.
  setTimeout(closeMention, 150)
}

// Клик по @упоминанию в тексте комментария → карточка пользователя.
const profileOpen = ref(false)
const profileUser = ref(null)

function openMentionProfile(login) {
  if (!login) return
  const u = mentionUsers.value.find((x) => (x.login || '').toLowerCase() === login.toLowerCase())
  if (!u) return
  profileUser.value = u
  profileOpen.value = true
}

const list = computed(() => tasks.commentsByTask[props.taskId] || [])

async function load() {
  loading.value = true
  try {
    await tasks.loadComments(props.taskId)
    await nextTick()
    scrollBottom()
  } catch (e) {
    notify.error(e?.message || 'Не удалось загрузить комментарии')
  } finally {
    loading.value = false
  }
}

function scrollBottom() {
  if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight
}

async function send() {
  const txt = draft.value.trim()
  if (!txt || sending.value) return
  sending.value = true
  try {
    await tasks.addComment(props.taskId, txt)
    draft.value = ''
    await nextTick()
    scrollBottom()
  } catch (e) {
    notify.error(e?.message || 'Не удалось отправить комментарий')
  } finally {
    sending.value = false
  }
}

function canEdit(c) {
  if (c.author_id === auth.user?.id) return true
  return isAtLeast(ROLES.MANAGER)
}

function startEdit(c) {
  editingId.value = c.id
  editText.value = c.text
}

function cancelEdit() {
  editingId.value = null
  editText.value = ''
}

async function saveEdit() {
  const txt = editText.value.trim()
  if (!txt) return
  try {
    await tasks.editComment(props.taskId, editingId.value, txt)
    cancelEdit()
  } catch (e) {
    notify.error(e?.message || 'Не удалось сохранить')
  }
}

function remove(c) {
  deletingId.value = c.id
}

async function confirmDelete() {
  const id = deletingId.value
  if (id == null) return
  deletingId.value = null
  try {
    await tasks.deleteComment(props.taskId, id)
  } catch (e) {
    notify.error(e?.message || 'Не удалось удалить')
  }
}

async function copy(c) {
  if (!c?.text) return
  try {
    await navigator.clipboard.writeText(c.text)
    notify.success('Комментарий скопирован')
  } catch {
    notify.error('Не удалось скопировать')
  }
}

function avatarOf(a) {
  if (!a) return ''
  return a.avatar_path ? `/uploads/${a.avatar_path}` : `/api/users/${a.id}/identicon`
}

function fmtTime(d) {
  if (!d) return ''
  const dt = new Date(d)
  return dt.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function onKeydown(e) {
  // Навигация по списку упоминаний перехватывает стрелки/Enter/Tab/Esc.
  if (mentionOpen.value && mentionItems.value.length) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      mentionIndex.value = (mentionIndex.value + 1) % mentionItems.value.length
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      mentionIndex.value = (mentionIndex.value - 1 + mentionItems.value.length) % mentionItems.value.length
      return
    }
    if (e.key === 'Enter' || e.key === 'Tab') {
      e.preventDefault()
      selectMention(mentionItems.value[mentionIndex.value])
      return
    }
    if (e.key === 'Escape') {
      e.preventDefault()
      closeMention()
      return
    }
  }
  // Cmd/Ctrl+Enter — отправить.
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    send()
  }
}

/* ── Контекстное меню (long-press на тач / ПКМ на десктопе) ───────── */
const ctxMenu = ref({ visible: false, x: 0, y: 0, comment: null })

const ctxStyle = computed(() => ({
  position: 'fixed',
  left: ctxMenu.value.x + 'px',
  top: ctxMenu.value.y + 'px',
  zIndex: 12000,
}))

function openCtxMenu(x, y, comment) {
  // Кламп в вьюпорт по приблизительному размеру меню (220×100), точнее
  // выровняем после рендера в nextTick.
  const pad = 8
  const w = 220
  const h = 108
  if (x + w > window.innerWidth - pad) x = window.innerWidth - w - pad
  if (y + h > window.innerHeight - pad) y = window.innerHeight - h - pad
  if (x < pad) x = pad
  if (y < pad) y = pad
  ctxMenu.value = { visible: true, x, y, comment }
}

function closeCtxMenu() {
  ctxMenu.value = { ...ctxMenu.value, visible: false }
}

function ctxAction(action) {
  const c = ctxMenu.value.comment
  closeCtxMenu()
  if (!c) return
  if (action === 'copy') copy(c)
  else if (action === 'edit') startEdit(c)
  else if (action === 'delete') remove(c)
}

let longPressTimer = null
let longPressFired = false
let pointerStartX = 0
let pointerStartY = 0
let pointerActiveId = null

function onCommentPointerDown(e, c) {
  // ПКМ обрабатывает @contextmenu; long-press — только основное касание/клик.
  if (e.button === 2) return
  if (editingId.value === c.id) return
  pointerActiveId = e.pointerId
  longPressFired = false
  pointerStartX = e.clientX
  pointerStartY = e.clientY
  clearTimeout(longPressTimer)
  longPressTimer = setTimeout(() => {
    longPressFired = true
    if (navigator.vibrate) {
      try { navigator.vibrate(15) } catch {/* iOS Safari */}
    }
    openCtxMenu(pointerStartX, pointerStartY, c)
  }, 500)
}

function onCommentPointerMove(e) {
  if (pointerActiveId == null || e.pointerId !== pointerActiveId) return
  const dx = Math.abs(e.clientX - pointerStartX)
  const dy = Math.abs(e.clientY - pointerStartY)
  // Сдвиг больше 10px → отменяем long-press (это скролл/выделение).
  if (dx > 10 || dy > 10) {
    clearTimeout(longPressTimer)
    pointerActiveId = null
  }
}

function onCommentPointerUp(e) {
  if (pointerActiveId != null && e?.pointerId !== pointerActiveId) return
  clearTimeout(longPressTimer)
  pointerActiveId = null
}

function onCommentContextMenu(e, c) {
  if (editingId.value === c.id) return
  openCtxMenu(e.clientX, e.clientY, c)
}

function onDocPointerDown(e) {
  if (!ctxMenu.value.visible) return
  const root = document.querySelector('.cmt-ctx-menu')
  if (!root || !root.contains(e.target)) closeCtxMenu()
}

function onDocScroll() { if (ctxMenu.value.visible) closeCtxMenu() }
function onDocKey(e) { if (e.key === 'Escape' && ctxMenu.value.visible) closeCtxMenu() }

async function loadMentionUsers() {
  try {
    const data = await getDirectory('', true)
    mentionUsers.value = data.items || data || []
  } catch {
    mentionUsers.value = [] // без каталога автокомплит просто не появится
  }
}

onMounted(() => {
  load()
  loadMentionUsers()
  document.addEventListener('pointerdown', onDocPointerDown, true)
  document.addEventListener('scroll', onDocScroll, true)
  document.addEventListener('keydown', onDocKey)
})

onBeforeUnmount(() => {
  clearTimeout(longPressTimer)
  document.removeEventListener('pointerdown', onDocPointerDown, true)
  document.removeEventListener('scroll', onDocScroll, true)
  document.removeEventListener('keydown', onDocKey)
})
</script>

<template>
  <div class="comments">
    <div ref="listEl" class="comments-list">
      <div v-if="loading" class="comments-empty">
        <span class="material-symbols-outlined spinning">progress_activity</span>
        Загрузка…
      </div>
      <div v-else-if="!list.length" class="comments-empty">
        <span class="material-symbols-outlined">forum</span>
        Комментариев пока нет
      </div>
      <div
        v-for="c in list"
        :key="c.id"
        class="comment-item"
        @pointerdown="onCommentPointerDown($event, c)"
        @pointermove="onCommentPointerMove"
        @pointerup="onCommentPointerUp"
        @pointercancel="onCommentPointerUp"
        @contextmenu.prevent="onCommentContextMenu($event, c)"
      >
        <img :src="avatarOf(c.author)" class="comment-ava" :alt="c.author?.fio || ''" />
        <div class="comment-body">
          <div class="comment-head">
            <span class="comment-author">{{ c.author?.fio || 'Сотрудник' }}</span>
            <span class="comment-time" :title="new Date(c.created_at).toLocaleString('ru-RU')">
              {{ fmtTime(c.created_at) }}
              <span v-if="c.updated_at" class="comment-edited" title="отредактировано">·&nbsp;ред.</span>
            </span>
            <!-- Hover-кнопки только для устройств с курсором (CSS гасит на тач);
                 на мобильном — long-press открывает контекстное меню. -->
            <div class="comment-actions" v-if="editingId !== c.id">
              <button class="ca-btn" @click="copy(c)" title="Скопировать комментарий">
                <span class="material-symbols-outlined">content_copy</span>
              </button>
              <template v-if="canEdit(c)">
                <button class="ca-btn" @click="startEdit(c)" title="Редактировать">
                  <span class="material-symbols-outlined">edit</span>
                </button>
                <button class="ca-btn danger" @click="remove(c)" title="Удалить">
                  <span class="material-symbols-outlined">delete</span>
                </button>
              </template>
            </div>
          </div>
          <div v-if="editingId === c.id" class="comment-edit">
            <textarea v-model="editText" class="comment-textarea ctl" rows="3" />
            <div class="edit-actions">
              <button class="btn-text" @click="cancelEdit">Отмена</button>
              <button class="btn-primary" @click="saveEdit">Сохранить</button>
            </div>
          </div>
          <MarkdownView
            v-else
            :source="c.text"
            class="comment-text"
            mentions
            :mention-names="mentionNames"
            @mention="openMentionProfile"
          />
        </div>
      </div>
    </div>

    <ConfirmDialog
      :visible="deletingId != null"
      header="Удалить комментарий"
      message="Это действие нельзя отменить. Удалить комментарий?"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDelete"
      @cancel="deletingId = null"
    />

    <!-- Карточка упомянутого пользователя (elevated — задача открыта в модалке). -->
    <EmployeeProfileDialog v-model="profileOpen" :user="profileUser" elevated />

    <!-- Контекстное меню комментария (long-press на тач / правая кнопка мыши) -->
    <Teleport to="body">
      <Transition name="cmt-ctx">
        <div
          v-if="ctxMenu.visible"
          class="cmt-ctx-menu"
          :style="ctxStyle"
          role="menu"
          @click.stop
        >
          <button class="cmt-ctx-item" @click="ctxAction('copy')">
            <span class="material-symbols-outlined">content_copy</span>
            <span>Скопировать</span>
          </button>
          <template v-if="canEdit(ctxMenu.comment)">
            <button class="cmt-ctx-item" @click="ctxAction('edit')">
              <span class="material-symbols-outlined">edit</span>
              <span>Редактировать</span>
            </button>
            <div class="cmt-ctx-divider" />
            <button class="cmt-ctx-item danger" @click="ctxAction('delete')">
              <span class="material-symbols-outlined">delete</span>
              <span>Удалить</span>
            </button>
          </template>
        </div>
      </Transition>
    </Teleport>

    <div class="comment-input">
      <div class="comment-input-field">
        <!-- Автокомплит @упоминаний: сотрудники активной компании -->
        <ul v-if="mentionOpen" class="mention-menu">
          <li
            v-for="(u, i) in mentionItems"
            :key="u.id"
            class="mention-item"
            :class="{ active: i === mentionIndex }"
            @mousedown.prevent="selectMention(u)"
            @mouseenter="mentionIndex = i"
          >
            <img :src="avatarUrl(u)" class="mention-ava" :alt="u.fio || ''" />
            <span class="mention-info">
              <span class="mention-fio">{{ u.fio || 'Сотрудник' }}</span>
              <span class="mention-login">@{{ u.login }}</span>
            </span>
          </li>
        </ul>
        <textarea
          ref="textareaRef"
          v-model="draft"
          class="comment-textarea ctl"
          rows="2"
          placeholder="Написать комментарий… (@ — упомянуть, Ctrl+Enter — отправить)"
          @keydown="onKeydown"
          @input="updateMentionState"
          @keyup="onCaretKeyup"
          @click="updateMentionState"
          @blur="onInputBlur"
        />
      </div>
      <button class="send-btn" :disabled="sending || !draft.trim()" @click="send">
        <span class="material-symbols-outlined">send</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.comments {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  flex: 1;
}

.comments-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-right: 4px;
}

.comments-empty {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  justify-content: center;
  padding: 28px 0;
  color: var(--color-on-surface-variant);
  font-size: 13px;
}
.comments-empty .material-symbols-outlined { font-size: 32px; opacity: 0.55; }

.comment-item {
  display: flex;
  gap: 10px;
  align-items: flex-start;
  position: relative;
}
.comment-ava { width: 36px; height: 36px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.comment-body { flex: 1; min-width: 0; }
.comment-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
  flex-wrap: wrap;
}
.comment-author { font-weight: 650; color: var(--color-on-surface); font-size: 13px; }
.comment-time { font-size: 11px; color: var(--color-on-surface-variant); }
.comment-edited { font-style: italic; }

/* Actions вытащены из flex-потока — иначе они занимают место в flex и
   при flex-wrap переносятся на следующую строку, создавая огромный отступ
   между ФИО автора и текстом комментария. */
.comment-actions {
  position: absolute;
  top: 0;
  right: 0;
  display: inline-flex;
  gap: 2px;
  opacity: 0;
  transition: opacity 0.15s;
  background: var(--acrylic-card-bg);
  border-radius: var(--radius-full, 999px);
  padding: 2px;
}
.comment-item:hover .comment-actions,
.comment-item:focus-within .comment-actions { opacity: 1; }

/* На тач-устройствах кнопки скрыты — действия открываются long-press'ом
   через контекстное меню (см. .cmt-ctx-menu). */
@media (hover: none) {
  .comment-actions { display: none; }
}
.ca-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  width: 26px;
  height: 26px; min-height: 0;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-on-surface-variant);
}
.ca-btn:hover { background: var(--color-surface-high); color: var(--color-on-surface); }
.ca-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.ca-btn .material-symbols-outlined { font-size: 16px; }

.comment-text {
  background: var(--color-surface-high);
  padding: 8px 12px;
  border-radius: var(--radius-md, 12px);
  font-size: 13px;
}

.comment-edit { display: flex; flex-direction: column; gap: 6px; }
.edit-actions { display: flex; gap: 6px; justify-content: flex-end; }

.comment-textarea {
  width: 100%;
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-md, 12px);
  padding: 8px 12px;
  font: inherit;
  font-size: 13px;
  color: var(--color-on-surface);
  resize: vertical;
  outline: none;
}
.comment-textarea:focus {
  border-color: var(--color-primary);
}

.comment-input {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  flex-shrink: 0;
}
.comment-input-field {
  position: relative;
  flex: 1;
  min-width: 0;
}

/* Выпадающий список упоминаний — над полем ввода. */
.mention-menu {
  position: absolute;
  left: 0;
  right: 0;
  bottom: calc(100% + 6px);
  margin: 0;
  padding: 4px;
  list-style: none;
  max-height: 240px;
  overflow-y: auto;
  background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md, 12px);
  box-shadow: var(--shadow-lg);
  z-index: 20;
}
.mention-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 8px;
  border-radius: var(--radius-sm, 8px);
  cursor: pointer;
}
.mention-item.active { background: var(--color-surface-low); }
.mention-ava { width: 26px; height: 26px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.mention-info { display: flex; flex-direction: column; min-width: 0; }
.mention-fio {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-on-surface);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mention-login { font-size: 11px; color: var(--color-on-surface-variant); }
.send-btn {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.send-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.send-btn .material-symbols-outlined { font-size: 20px; }

.btn-text {
  background: transparent;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  padding: 6px 12px;
  border-radius: var(--radius-full, 999px);
  font-weight: 600;
}
.btn-text:hover { background: color-mix(in oklab, var(--color-primary) 10%, transparent); }
.btn-primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  cursor: pointer;
  padding: 6px 14px;
  border-radius: var(--radius-full, 999px);
  font-weight: 600;
}

.spinning { animation: cspin 1s linear infinite; }
@keyframes cspin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>

<!-- Контекстное меню телепортируется в <body>, scoped-стили на него не
     попадут. Используем обычный (нескоупированный) стиль с уникальным
     префиксом cmt-ctx-, чтобы не задеть другие компоненты. -->
<style>
.cmt-ctx-menu {
  min-width: 200px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md, 12px);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.cmt-ctx-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  text-align: left;
  border-radius: var(--radius-sm, 8px);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.cmt-ctx-item:hover { background: var(--color-surface-low); }
.cmt-ctx-item.danger { color: var(--color-error); }
.cmt-ctx-item.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.cmt-ctx-item .material-symbols-outlined { font-size: 18px; }
.cmt-ctx-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}
.cmt-ctx-enter-active, .cmt-ctx-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.cmt-ctx-enter-from, .cmt-ctx-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
