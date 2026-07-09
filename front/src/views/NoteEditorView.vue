<template>
  <!-- wheel с Ctrl/Cmd (и щипок трекпада — браузер шлёт его как ctrl+wheel)
       перехватываем в зум листа вместо зума страницы. -->
  <div class="np" @wheel="onZoomWheel">
    <div class="np-panel">
      <header class="np-head">
        <button class="np-back" title="К списку заметок" @click="goBack">
          <span class="material-symbols-outlined">arrow_back</span>
          <span class="np-back-label">Заметки</span>
        </button>

        <span v-if="!readOnly" class="np-savestate" :class="saveState">
          <span class="material-symbols-outlined">{{ saveIcon }}</span>
          {{ saveLabel }}
        </span>
        <span v-else class="np-savestate">
          <span class="material-symbols-outlined">visibility</span>
          Только просмотр
        </span>

        <!-- Чужая заметка: владелец -->
        <span v-if="!isOwner && ownerName" class="np-owner" :title="`Владелец: ${ownerName}`">
          <img class="np-owner-avatar" :src="ownerAvatarUrl" :alt="ownerName" />
          {{ ownerName }}
        </span>

        <!-- Кто сейчас в заметке (совместная работа) -->
        <div v-if="collabOthers.length" class="np-presence" title="Сейчас в заметке">
          <span
            v-for="p in collabOthers"
            :key="p.id"
            class="np-presence-dot"
            :style="{ background: `var(--tag-${p.color}-surface)`, borderColor: `var(--tag-${p.color}-accent)` }"
            :title="p.fio"
          >{{ initials(p.fio) }}</span>
        </div>

        <div class="np-actions">
          <!-- Десктоп: масштаб и действия отдельными кнопками -->
          <template v-if="!isMobile">
            <!-- Масштаб листа: −/процент (клик — сброс)/+ -->
            <div class="np-zoom">
              <button class="np-icon np-zoom-btn" title="Уменьшить масштаб" :disabled="zoom <= ZOOM_MIN" @click="stepZoom(-1)">
                <span class="material-symbols-outlined">zoom_out</span>
              </button>
              <button class="np-zoom-value" title="Сбросить масштаб" @click="resetZoom">{{ Math.round(zoom * 100) }}%</button>
              <button class="np-icon np-zoom-btn" title="Увеличить масштаб" :disabled="zoom >= ZOOM_MAX" @click="stepZoom(1)">
                <span class="material-symbols-outlined">zoom_in</span>
              </button>
            </div>
            <template v-if="isOwner">
              <button class="np-icon" title="Группы" @click="groupsOpen = true">
                <span class="material-symbols-outlined">folder</span>
              </button>
              <button class="np-icon" title="Поделиться" @click="shareOpen = true">
                <span class="material-symbols-outlined">share</span>
              </button>
            </template>
            <!-- Экспорт доступен и адресатам шаринга (чтение есть — выгрузка тоже) -->
            <button class="np-icon" title="Экспорт в .txt" @click="exportTxt">
              <span class="material-symbols-outlined">download</span>
            </button>
            <button v-if="isOwner" class="np-icon danger" title="Удалить заметку" @click="deleteOpen = true">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </template>

          <!-- Мобайл: в узкой шапке ряд иконок не помещается — всё в меню «⋮» -->
          <div v-else ref="moreRef" class="np-more">
            <button class="np-icon" title="Ещё" aria-label="Действия с заметкой" @click="moreOpen = !moreOpen">
              <span class="material-symbols-outlined">more_vert</span>
            </button>
            <Transition name="np-more">
              <div v-if="moreOpen" class="np-more-pop">
                <div class="np-more-zoom">
                  <button class="np-icon np-zoom-btn" title="Уменьшить масштаб" :disabled="zoom <= ZOOM_MIN" @click="stepZoom(-1)">
                    <span class="material-symbols-outlined">zoom_out</span>
                  </button>
                  <button class="np-zoom-value" title="Сбросить масштаб" @click="resetZoom">{{ Math.round(zoom * 100) }}%</button>
                  <button class="np-icon np-zoom-btn" title="Увеличить масштаб" :disabled="zoom >= ZOOM_MAX" @click="stepZoom(1)">
                    <span class="material-symbols-outlined">zoom_in</span>
                  </button>
                </div>
                <div class="np-more-divider" />
                <template v-if="isOwner">
                  <button class="np-more-item" @click="pickMore(() => groupsOpen = true)">
                    <span class="material-symbols-outlined">folder</span>
                    Группы
                  </button>
                  <button class="np-more-item" @click="pickMore(() => shareOpen = true)">
                    <span class="material-symbols-outlined">share</span>
                    Поделиться
                  </button>
                </template>
                <button class="np-more-item" @click="pickMore(exportTxt)">
                  <span class="material-symbols-outlined">download</span>
                  Экспорт в .txt
                </button>
                <template v-if="isOwner">
                  <div class="np-more-divider" />
                  <button class="np-more-item danger" @click="pickMore(() => deleteOpen = true)">
                    <span class="material-symbols-outlined">delete</span>
                    Удалить заметку
                  </button>
                </template>
              </div>
            </Transition>
          </div>
        </div>
      </header>

      <div v-if="loading" class="np-loading">Загрузка…</div>
      <EmptyState
        v-else-if="notFound"
        class="np-loading" icon="scan_delete" tone="soft"
        title="Заметка не найдена"
        subtitle="Возможно, она удалена."
      />
      <template v-else>
        <input
          v-model="title"
          class="np-title"
          type="text"
          placeholder="Название заметки"
          maxlength="300"
          :readonly="readOnly"
          @input="markDirty"
          @blur="flush"
          @keydown.enter.prevent="focusEditor"
        />
        <NoteRichEditor
          ref="editorRef"
          class="np-editor"
          :doc="doc"
          :zoom="zoom"
          :editable="!readOnly"
          :upload-image="isOwner ? uploadImageFile : null"
          selection-menu
          @change="onDocChange"
          @blur="flush"
          @selection-menu="onSelectionMenu"
        />
      </template>
    </div>

    <NoteGroupsDialog v-model="groupsOpen" :note-id="noteId" :group-ids="groupIds" @saved="onGroupsSaved" />
    <NoteShareDialog v-model="shareOpen" :note-id="noteId" />

    <!-- ПКМ на выделенном тексте: ИИ-действия + «Создать из выделенного» -->
    <NoteSelectionMenu
      :visible="selMenu.visible"
      :x="selMenu.x"
      :y="selMenu.y"
      :ai-available="hasCompany && !readOnly"
      :can-task="hasCompany"
      @close="selMenu.visible = false"
      @ai="onAiAction"
      @create="onCreateFrom"
      @copy="copySelection"
      @send-chat="sendChatOpen = true"
    />
    <NoteAiDialog
      v-model="ai.open"
      :label="ai.label"
      :loading="ai.loading"
      :error="ai.error"
      :result="ai.result"
      :is-continue="ai.action === 'continue'"
      @apply="applyAi"
      @retry="runAi"
    />
    <NoteToDiaryDialog v-model="diaryOpen" :text="sel.text" />
    <TaskForm
      v-if="taskFormOpen"
      :preset-name="taskPresetName"
      @close="taskFormOpen = false"
      @saved="taskFormOpen = false"
    />
    <!-- Публикация выделенного на портал (Markdown с сохранением форматирования) -->
    <PostComposer
      v-if="postPreset"
      v-model="postComposerOpen"
      :preset="postPreset"
      @saved="notif.success('Опубликовано на портале')"
    />
    <!-- Выделенный фрагмент в чат — уходит текстом с форматированием -->
    <NoteSendToChatDialog v-model="sendChatOpen" mode="text" :text="selectionMarkdown()" />
    <ConfirmDialog
      :visible="deleteOpen"
      header="Удалить заметку?"
      :message="`«${title || 'Без названия'}» будет удалена навсегда вместе с картинками. Ссылки на неё перестанут работать.`"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDelete"
      @cancel="deleteOpen = false"
    />
  </div>
</template>

<script setup>
// Страница заметки: крупный заголовок + rich-редактор. Автосохранение —
// дебаунс 1.5с после правок, немедленно на blur/beforeunload/Cmd+S.
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import NoteRichEditor from '@/components/notes/NoteRichEditor.vue'
import NoteGroupsDialog from '@/components/notes/NoteGroupsDialog.vue'
import NoteShareDialog from '@/components/notes/NoteShareDialog.vue'
import NoteSelectionMenu from '@/components/notes/NoteSelectionMenu.vue'
import NoteAiDialog from '@/components/notes/NoteAiDialog.vue'
import NoteToDiaryDialog from '@/components/notes/NoteToDiaryDialog.vue'
import NoteSendToChatDialog from '@/components/notes/NoteSendToChatDialog.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import { docToMarkdown } from '@/utils/tiptapMarkdown.js'

// Композер портала тяжёлый (стор портала) — грузим по первому использованию.
const PostComposer = defineAsyncComponent(() => import('@/components/portal/PostComposer.vue'))
import * as api from '@/api/notes.js'
import { transformText } from '@/api/ai.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useNoteCollab } from '@/composables/useNoteCollab.js'

const props = defineProps({ id: { type: String, required: true } })

const router = useRouter()
const store = useNotesStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()

// ── Мобильное меню «⋮» в шапке ──
const moreOpen = ref(false)
const moreRef = ref(null)

function pickMore(action) {
  moreOpen.value = false
  action()
}

function onDocPointerDown(e) {
  if (moreOpen.value && !moreRef.value?.contains(e.target)) moreOpen.value = false
}

const noteId = computed(() => Number(props.id))
const loading = ref(true)
const notFound = ref(false)
const title = ref('')
const doc = ref(null)
const groupIds = ref([])
const editorRef = ref(null)

// Доступ: owner | edit | view (заметка может быть чужой — адресный шаринг).
const myAccess = ref('owner')
const ownerName = ref('')
const ownerAvatarUrl = ref('')
const isOwner = computed(() => myAccess.value === 'owner')
const readOnly = computed(() => myAccess.value === 'view')

function initials(fio = '') {
  return fio.split(/\s+/).slice(0, 2).map((w) => w[0] || '').join('').toUpperCase() || '?'
}

// ── Автосохранение ──
const saveState = ref('saved') // saved | dirty | saving | error
let saveTimer = null
let pendingDoc = null // последний JSON из редактора (не кладём в doc — иначе setContent сбросит курсор)

const saveLabel = computed(() => ({
  saved: 'Сохранено', dirty: 'Изменено…', saving: 'Сохраняю…', error: 'Ошибка сохранения',
})[saveState.value])
const saveIcon = computed(() => ({
  saved: 'cloud_done', dirty: 'edit', saving: 'progress_activity', error: 'cloud_off',
})[saveState.value])

onMounted(async () => {
  try {
    const n = await api.getNote(noteId.value)
    title.value = n.title
    doc.value = n.doc && Object.keys(n.doc).length ? n.doc : null
    groupIds.value = n.group_ids ?? []
    myAccess.value = n.my_access || 'owner'
    ownerName.value = n.owner_name || ''
    ownerAvatarUrl.value = n.owner_avatar
      ? `/uploads/${n.owner_avatar}`
      : (n.owner_id ? `/api/users/${n.owner_id}/identicon` : '')
    startCollab(n)
  } catch (e) {
    if (e?.status === 404) notFound.value = true
    else notif.error(e?.message || 'Не удалось загрузить заметку')
  } finally {
    loading.value = false
  }
  window.addEventListener('beforeunload', flush)
  window.addEventListener('keydown', onKeydown)
  document.addEventListener('mousedown', onDocPointerDown, true)
  document.addEventListener('touchstart', onDocPointerDown, { passive: true, capture: true })
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', flush)
  window.removeEventListener('keydown', onKeydown)
  document.removeEventListener('mousedown', onDocPointerDown, true)
  document.removeEventListener('touchstart', onDocPointerDown, true)
  flush()
})

function onKeydown(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    flush()
  }
  if (e.key === 'Escape' && moreOpen.value) moreOpen.value = false
}

function markDirty() {
  saveState.value = 'dirty'
  clearTimeout(saveTimer)
  saveTimer = setTimeout(flush, 1500)
}

function onDocChange(json) {
  pendingDoc = json
  markDirty()
}

async function flush() {
  if (readOnly.value) return
  if (saveState.value !== 'dirty' && saveState.value !== 'error') return
  clearTimeout(saveTimer)
  saveState.value = 'saving'
  const body = { title: title.value }
  if (pendingDoc) body.doc = pendingDoc
  try {
    const n = await api.updateNote(noteId.value, body)
    pendingDoc = null
    saveState.value = 'saved'
    store.upsertNote(n)
  } catch (e) {
    saveState.value = 'error'
    notif.error(e?.message || 'Не удалось сохранить заметку')
  }
}

function focusEditor() { editorRef.value?.editor?.commands.focus('start') }

// ── Масштаб листа (общий для всех заметок, живёт в localStorage) ──
const ZOOM_MIN = 0.6
const ZOOM_MAX = 2
const ZOOM_STEP = 0.1

function loadZoom() {
  const v = Number(localStorage.getItem('gw_note_zoom'))
  return v >= ZOOM_MIN && v <= ZOOM_MAX ? v : 1
}
const zoom = ref(loadZoom())

function setZoom(v) {
  // Сотые — чтобы щипок трекпада масштабировал плавно (кнопки шагают по 0.1).
  zoom.value = Math.round(Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, v)) * 100) / 100
  try { localStorage.setItem('gw_note_zoom', String(zoom.value)) } catch { /* private mode */ }
}
function stepZoom(dir) { setZoom(Math.round((zoom.value + dir * ZOOM_STEP) * 10) / 10) }
function resetZoom() { setZoom(1) }

// Ctrl/Cmd + колесо и pinch-жест трекпада (приходит как wheel с ctrlKey) —
// зумим лист, а не страницу браузера.
function onZoomWheel(e) {
  if (!e.ctrlKey && !e.metaKey) return
  e.preventDefault()
  setZoom(zoom.value * Math.exp(-e.deltaY * 0.0022))
}

async function goBack() {
  await flush()
  router.push('/notes')
}

// ── Картинки редактора ──
async function uploadImageFile(file) {
  try {
    const { path } = await api.uploadImage(noteId.value, file)
    return path
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить изображение')
    return null
  }
}

// ── Группы ──
function onGroupsSaved(n) {
  groupIds.value = n.group_ids ?? []
  store.fetchGroups({ silent: true })
  store.fetchNotes({ silent: true })
}

// ── Экспорт .txt ──
async function exportTxt() {
  await flush()
  try {
    const resp = await api.exportNote(noteId.value)
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${(title.value || 'Заметка').slice(0, 100)}.txt`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    notif.error(e?.message || 'Не удалось экспортировать')
  }
}

// ── Совместное редактирование (присутствие, курсоры, живые правки) ──
const namesMap = ref({}) // user_id → ФИО (владелец + адресаты) для подписей курсоров

const collab = useNoteCollab({
  noteId,
  editorRef,
  canEdit: computed(() => !readOnly.value),
  isLocallyDirty: () => saveState.value !== 'saved',
  fallbackNames: (id) => namesMap.value[id],
})
const collabOthers = collab.others

function startCollab(n) {
  if (n.owner_id && n.owner_name) namesMap.value[n.owner_id] = n.owner_name
  collab.start()
  // Владелец знает адресатов — подписи курсоров без ожидания их join.
  if ((n.my_access || 'owner') === 'owner') {
    api.getMembers(noteId.value)
      .then((data) => {
        for (const m of data.members ?? []) namesMap.value[m.user_id] = m.fio
      })
      .catch(() => { /* подписи — не критично */ })
  }
}

// ── ИИ и «создать из выделенного» (контекстное меню выделения) ──
const auth = useAuthStore()
const hasCompany = computed(() => !!auth.companyId)

const selMenu = ref({ visible: false, x: 0, y: 0 })
const sel = ref({ text: '', from: 0, to: 0 })

function onSelectionMenu({ x, y, text, from, to }) {
  sel.value = { text, from, to }
  selMenu.value = { visible: true, x, y }
}

const ai = ref({ open: false, loading: false, action: '', style: null, label: '', result: '', error: '' })

function onAiAction({ action, style, label }) {
  ai.value = { open: true, loading: true, action, style, label, result: '', error: '' }
  runAi()
}

async function runAi() {
  ai.value.loading = true
  ai.value.error = ''
  try {
    const { text } = await transformText({ action: ai.value.action, style: ai.value.style, text: sel.value.text })
    ai.value.result = text
  } catch (e) {
    ai.value.error = e?.error === 'AI_DISABLED'
      ? 'ИИ не включён в активной компании. Администратор может включить его в настройках компании.'
      : (e?.message || 'Не удалось обработать текст')
  } finally {
    ai.value.loading = false
  }
}

function escapeHtml(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;')
}

// applyAi — вставка результата: replace — на место выделения (переносы строк
// внутри абзаца — <br>), below — новыми абзацами после блока с выделением;
// «продолжить» дописывается сразу за выделением, продолжая предложение.
function applyAi(mode) {
  const ed = editorRef.value?.editor
  if (!ed) return
  const { from, to } = sel.value
  const inline = escapeHtml(ai.value.result).replace(/\n/g, '<br>')
  const chain = ed.chain().focus()
  if (mode === 'replace') {
    chain.insertContentAt({ from, to }, inline).run()
  } else if (ai.value.action === 'continue') {
    chain.insertContentAt(to, ' ' + inline).run()
  } else {
    const paragraphs = ai.value.result
      .split(/\n{2,}/)
      .map((p) => '<p>' + escapeHtml(p).replace(/\n/g, '<br>') + '</p>')
      .join('')
    const end = ed.state.doc.resolve(Math.min(to, ed.state.doc.content.size)).end()
    chain.insertContentAt(end, paragraphs).run()
  }
  ai.value.open = false
}

const diaryOpen = ref(false)
const taskFormOpen = ref(false)
const sendChatOpen = ref(false)
const postComposerOpen = ref(false)
const postPreset = ref(null)
const taskPresetName = computed(() => {
  const firstLine = sel.value.text.split('\n').map((s) => s.trim()).find(Boolean) || ''
  return firstLine.slice(0, 200)
})

// Markdown выделенного фрагмента — с форматированием (жирный, списки, …);
// fallback на плоский текст, если слайс не собрался.
function selectionMarkdown() {
  const ed = editorRef.value?.editor
  if (!ed) return sel.value.text
  try {
    const slice = ed.state.doc.slice(sel.value.from, sel.value.to)
    return docToMarkdown(slice.content.toJSON() || []) || sel.value.text
  } catch {
    return sel.value.text
  }
}

function onCreateFrom(kind) {
  if (kind === 'task') taskFormOpen.value = true
  else if (kind === 'post') {
    postPreset.value = { title: '', body: selectionMarkdown() }
    postComposerOpen.value = true
  } else diaryOpen.value = true
}

async function copySelection() {
  try {
    await navigator.clipboard.writeText(sel.value.text)
    notif.success('Скопировано в буфер обмена')
  } catch {
    notif.error('Не удалось скопировать')
  }
}

// ── Удаление ──
const groupsOpen = ref(false)
const shareOpen = ref(false)
const deleteOpen = ref(false)

async function confirmDelete() {
  deleteOpen.value = false
  try {
    await store.removeNote(noteId.value)
    saveState.value = 'saved' // не пытаться сохранить удалённую при unmount
    pendingDoc = null
    router.push('/notes')
  } catch (e) {
    notif.error(e?.message || 'Не удалось удалить заметку')
  }
}
</script>

<style scoped>
.np {
  height: 100%;
  min-height: 0;
  width: 100%;
  padding: 16px;
  display: flex;
}
.np-panel {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  overflow: hidden;
}

.np-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 16px 8px;
  flex-shrink: 0;
}
.np-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 36px;
  padding: 0 12px;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
}
.np-back:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); color: var(--color-primary); }
.np-back .material-symbols-outlined { font-size: 20px; }

.np-savestate {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 12.5px;
  color: var(--color-text-dim);
}
.np-savestate .material-symbols-outlined { font-size: 16px; }
.np-savestate.saving .material-symbols-outlined { animation: npspin 1s linear infinite; }
.np-savestate.error { color: var(--color-error); }
@keyframes npspin { to { transform: rotate(360deg); } }

/* Владелец чужой заметки */
.np-owner {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px 3px 4px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  font-size: 12.5px;
  font-weight: 600;
  color: var(--color-text-dim);
  max-width: 220px;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.np-owner-avatar { width: 22px; height: 22px; border-radius: 50%; object-fit: cover; }

/* Присутствие: кто сейчас в заметке */
.np-presence { display: inline-flex; align-items: center; }
.np-presence-dot {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  border: 2px solid;
  display: grid;
  place-items: center;
  font-size: 10px;
  font-weight: 800;
  color: var(--color-text);
  margin-left: -6px;
}
.np-presence-dot:first-child { margin-left: 0; }

.np-actions { display: flex; align-items: center; gap: 4px; margin-left: auto; }

.np-zoom {
  display: inline-flex;
  align-items: center;
  gap: 0;
  margin-right: 4px;
}
.np-zoom-btn { width: 32px; height: 32px; }
.np-zoom-btn:disabled { opacity: 0.35; pointer-events: none; }
.np-zoom-value {
  min-width: 44px;
  border: none;
  background: none;
  color: var(--color-text-dim);
  font-size: 12.5px;
  font-weight: 700;
  cursor: pointer;
  border-radius: var(--radius-sm);
  padding: 4px 2px;
}
.np-zoom-value:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); color: var(--color-primary); }
.np-icon {
  width: 38px;
  height: 38px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  cursor: pointer;
}
.np-icon .material-symbols-outlined { font-size: 21px; }
.np-icon:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); color: var(--color-primary); }
.np-icon.danger:hover { background: color-mix(in oklch, var(--color-error) 10%, transparent); color: var(--color-error); }

/* ── Мобильное меню «⋮» (стекло, как поповеры карточек) ── */
.np-more { position: relative; }
.np-more-pop {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 30;
  min-width: 210px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 6px;
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.np-more-zoom {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 2px;
  padding: 2px 6px;
}
/* В меню проценты нужны всегда (глобальный мобильный скрыватель не про нас). */
.np-more-zoom .np-zoom-value { display: block; }
.np-more-item {
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
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.np-more-item:hover { background: var(--color-surface-low); }
.np-more-item .material-symbols-outlined { font-size: 18px; color: var(--color-text-dim); }
.np-more-item.danger,
.np-more-item.danger .material-symbols-outlined { color: var(--color-error); }
.np-more-divider { height: 1px; background: var(--color-outline-dim); margin: 4px 4px; }

.np-more-enter-active, .np-more-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top right;
}
.np-more-enter-from, .np-more-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}

.np-loading {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-dim);
  font-size: 14px;
}

.np-title {
  flex-shrink: 0;
  margin: 4px 24px 0;
  padding: 6px 4px;
  border: none;
  background: none;
  outline: none;
  color: var(--color-text);
  font-size: 27px;
  font-weight: 750;
  line-height: 1.25;
}
.np-title::placeholder { color: var(--color-text-dim); opacity: 0.55; }

.np-editor {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 8px 24px 0;
}

@media (max-width: 768px) {
  /* Во всю страницу: без рамки-«карточки». */
  .np { padding: 0; }
  .np-panel {
    border: none;
    border-radius: 0;
    background: transparent;
  }
  .np-back-label { display: none; }
  .np-title { margin: 2px 14px 0; font-size: 22px; }
  .np-editor { padding: 6px 14px 0; }
  /* Резерв под нижнюю навигацию (64px) + воздух. Именно на .tiptap, а не на
     скроллер .np-editor: длинный документ переполняет flex-бокс .ne-content
     (flex:1 + min-height:0), и padding-bottom контейнера остаётся у края
     бокса — последние строки прятались за акриловой навигацией. */
  .np-editor :deep(.tiptap) {
    padding-bottom: calc(116px + env(safe-area-inset-bottom, 0px));
  }
  .np-owner { display: none; }
}
</style>

<!-- Курсоры соавторов рисуются ProseMirror-декорациями внутри контента
     редактора — стили вне scoped. -->
<style>
.nc-caret {
  position: relative;
  display: inline;
  border-left: 2px solid var(--nc-color);
  margin-left: -1px;
}
.nc-caret-label {
  position: absolute;
  top: -1.35em;
  left: -2px;
  padding: 1px 6px;
  border-radius: var(--radius-xs, 6px) var(--radius-xs, 6px) var(--radius-xs, 6px) 0;
  background: var(--nc-color);
  color: var(--color-surface);
  font-size: 10.5px;
  font-weight: 700;
  line-height: 1.4;
  white-space: nowrap;
  pointer-events: none;
  user-select: none;
}
.nc-selection { border-radius: 2px; }
</style>
