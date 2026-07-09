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

        <span class="np-savestate" :class="saveState">
          <span class="material-symbols-outlined">{{ saveIcon }}</span>
          {{ saveLabel }}
        </span>

        <div class="np-actions">
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
          <button class="np-icon" title="Группы" @click="groupsOpen = true">
            <span class="material-symbols-outlined">folder</span>
          </button>
          <button class="np-icon" title="Поделиться" @click="shareOpen = true">
            <span class="material-symbols-outlined">share</span>
          </button>
          <button class="np-icon" title="Экспорт в .txt" @click="exportTxt">
            <span class="material-symbols-outlined">download</span>
          </button>
          <button class="np-icon danger" title="Удалить заметку" @click="deleteOpen = true">
            <span class="material-symbols-outlined">delete</span>
          </button>
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
          @input="markDirty"
          @blur="flush"
          @keydown.enter.prevent="focusEditor"
        />
        <NoteRichEditor
          ref="editorRef"
          class="np-editor"
          :doc="doc"
          :zoom="zoom"
          :upload-image="uploadImageFile"
          @change="onDocChange"
          @blur="flush"
        />
      </template>
    </div>

    <NoteGroupsDialog v-model="groupsOpen" :note-id="noteId" :group-ids="groupIds" @saved="onGroupsSaved" />
    <NoteShareDialog v-model="shareOpen" :note-id="noteId" />
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import EmptyState from '@/components/common/EmptyState.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import NoteRichEditor from '@/components/notes/NoteRichEditor.vue'
import NoteGroupsDialog from '@/components/notes/NoteGroupsDialog.vue'
import NoteShareDialog from '@/components/notes/NoteShareDialog.vue'
import * as api from '@/api/notes.js'
import { useNotesStore } from '@/stores/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({ id: { type: String, required: true } })

const router = useRouter()
const store = useNotesStore()
const notif = useNotificationsStore()

const noteId = computed(() => Number(props.id))
const loading = ref(true)
const notFound = ref(false)
const title = ref('')
const doc = ref(null)
const groupIds = ref([])
const editorRef = ref(null)

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
  } catch (e) {
    if (e?.status === 404) notFound.value = true
    else notif.error(e?.message || 'Не удалось загрузить заметку')
  } finally {
    loading.value = false
  }
  window.addEventListener('beforeunload', flush)
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', flush)
  window.removeEventListener('keydown', onKeydown)
  flush()
})

function onKeydown(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 's') {
    e.preventDefault()
    flush()
  }
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
  .np { padding: 12px 12px calc(64px + env(safe-area-inset-bottom, 0px)); }
  .np-back-label { display: none; }
  /* Узкая шапка: проценты убираем, остаются кнопки −/+ (сброс — долгим
     сведением к 100% кнопками). */
  .np-zoom-value { display: none; }
  .np-title { margin: 2px 14px 0; font-size: 22px; }
  .np-editor { padding: 6px 14px 0; }
}
</style>
