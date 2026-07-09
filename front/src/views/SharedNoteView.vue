<template>
  <div class="sn-page">
    <header class="sn-top">
      <div class="sn-brand"><span class="material-symbols-outlined">note_stack</span> Заметка</div>
      <h1 v-if="note" class="sn-name">{{ note.title || 'Без названия' }}</h1>
      <span v-if="access === 'edit'" class="sn-mode sn-mode-edit">
        <span class="material-symbols-outlined">edit</span>
        Редактирование
        <span class="sn-savestate" :class="saveState">{{ saveLabel }}</span>
      </span>
      <span v-else-if="note" class="sn-mode">Только просмотр</span>
    </header>

    <div v-if="notFound" class="sn-state">
      <span class="material-symbols-outlined">link_off</span>
      <p>Ссылка не найдена или отозвана.</p>
    </div>

    <div v-else-if="note" class="sn-shell">
      <input
        v-if="access === 'edit'"
        v-model="title"
        class="sn-title"
        type="text"
        placeholder="Название заметки"
        maxlength="300"
        @input="markDirty"
        @blur="flush"
      />
      <NoteRichEditor
        class="sn-editor"
        :doc="doc"
        :editable="access === 'edit'"
        @change="onDocChange"
        @blur="flush"
      />
    </div>
  </div>
</template>

<script setup>
// Публичная заметка по коду-capability: view — чтение (editable:false), edit —
// тот же редактор с автосохранением PUT по коду; без «Поделиться»/«Удалить» и
// без загрузки картинок (аплоад только у владельца).
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import NoteRichEditor from '@/components/notes/NoteRichEditor.vue'
import { getSharedNote, updateSharedNote } from '@/api/notes.js'

const route = useRoute()
const code = route.params.code

const note = ref(null)
const access = ref('view')
const notFound = ref(false)
const title = ref('')
const doc = ref(null)

const saveState = ref('saved')
let saveTimer = null
let pendingDoc = null

const saveLabel = computed(() => ({
  saved: '· сохранено', dirty: '· изменено…', saving: '· сохраняю…', error: '· ошибка',
})[saveState.value])

onMounted(async () => {
  try {
    const data = await getSharedNote(code)
    note.value = data.note
    access.value = data.access
    title.value = data.note.title
    doc.value = data.note.doc && Object.keys(data.note.doc).length ? data.note.doc : null
    document.title = `${data.note.title || 'Заметка'} — Groove Work`
  } catch {
    notFound.value = true
  }
  window.addEventListener('beforeunload', flush)
})

onBeforeUnmount(() => {
  window.removeEventListener('beforeunload', flush)
  flush()
})

function markDirty() {
  if (access.value !== 'edit') return
  saveState.value = 'dirty'
  clearTimeout(saveTimer)
  saveTimer = setTimeout(flush, 1500)
}

function onDocChange(json) {
  pendingDoc = json
  markDirty()
}

async function flush() {
  if (access.value !== 'edit') return
  if (saveState.value !== 'dirty' && saveState.value !== 'error') return
  clearTimeout(saveTimer)
  saveState.value = 'saving'
  const body = { title: title.value }
  if (pendingDoc) body.doc = pendingDoc
  try {
    await updateSharedNote(code, body)
    pendingDoc = null
    saveState.value = 'saved'
  } catch {
    saveState.value = 'error'
  }
}
</script>

<style scoped>
.sn-page {
  min-height: 100vh;
  display: flex;
  flex-direction: column;
  padding: 16px;
  gap: 12px;
  max-width: 920px;
  margin: 0 auto;
  width: 100%;
}

.sn-top { display: flex; align-items: center; gap: 14px; flex-wrap: wrap; }
.sn-brand {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 14px;
  font-weight: 700;
  color: var(--color-primary);
}
.sn-brand .material-symbols-outlined { font-size: 20px; }
.sn-name {
  flex: 1;
  min-width: 0;
  margin: 0;
  font-size: 17px;
  font-weight: 750;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sn-mode {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 12px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font-size: 12.5px;
  font-weight: 600;
}
.sn-mode-edit {
  background: color-mix(in oklch, var(--color-primary) 12%, var(--color-surface));
  color: var(--color-primary);
}
.sn-mode .material-symbols-outlined { font-size: 16px; }
.sn-savestate { font-weight: 500; opacity: 0.85; }
.sn-savestate.error { color: var(--color-error); }

.sn-state {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  color: var(--color-text-dim);
}
.sn-state .material-symbols-outlined { font-size: 48px; }

.sn-shell {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  padding: 8px 20px 20px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
}
.sn-title {
  flex-shrink: 0;
  margin: 8px 4px 0;
  padding: 6px 4px;
  border: none;
  background: none;
  outline: none;
  color: var(--color-text);
  font-size: 24px;
  font-weight: 750;
}
.sn-title::placeholder { color: var(--color-text-dim); opacity: 0.55; }
.sn-editor { flex: 1; min-height: 0; }

@media (max-width: 768px) {
  .sn-page { padding: 12px; }
  .sn-shell { padding: 6px 12px 14px; }
}
</style>
