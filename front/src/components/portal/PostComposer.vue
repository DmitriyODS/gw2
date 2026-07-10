<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    size="lg"
    mobile="full"
    :show-icon="false"
    :title="isEdit ? 'Редактировать публикацию' : 'Создать публикацию'"
    :busy="saving"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: isEdit ? 'Сохранить' : 'Опубликовать', icon: 'send', disabled: !body.trim() },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="submit"
  >
    <div
      class="composer"
      @dragover.prevent="dragOver = true"
      @dragleave.prevent="dragOver = false"
      @drop.prevent="onDrop"
    >
      <!-- Шапка как в композерах соцсетей: автор + выбор раздела -->
      <div class="composer-author">
        <img class="composer-avatar" :src="me.avatarUrl" :alt="me.fio" />
        <div class="composer-author-info">
          <span class="composer-author-name">{{ me.fio }}</span>
          <Select
            v-model="topicId"
            :options="portal.topics"
            option-label="name"
            option-value="id"
            placeholder="Без раздела"
            class="composer-topic"
            show-clear
          />
        </div>
      </div>

      <input v-model="title" class="composer-title" placeholder="Заголовок (необязательно)" maxlength="200" />

      <!-- Панель форматирования Markdown + предпросмотр -->
      <div class="composer-toolbar">
        <button
          v-for="b in TOOLBAR"
          :key="b.icon"
          type="button"
          class="composer-tbtn"
          :title="b.title"
          :disabled="preview"
          @click="b.run()"
        >
          <span class="material-symbols-outlined">{{ b.icon }}</span>
        </button>
        <button
          type="button"
          class="composer-tbtn composer-preview-btn"
          :class="{ active: preview }"
          :title="preview ? 'Продолжить редактирование' : 'Предпросмотр'"
          @click="preview = !preview"
        >
          <span class="material-symbols-outlined">{{ preview ? 'edit' : 'visibility' }}</span>
          <span class="composer-preview-label">{{ preview ? 'Редактор' : 'Предпросмотр' }}</span>
        </button>
      </div>

      <MarkdownView v-if="preview" class="composer-preview" :source="body || '*Пока пусто…*'" />
      <textarea
        v-else
        ref="bodyEl"
        v-model="body"
        class="composer-body"
        rows="5"
        maxlength="10000"
        placeholder="О чём хотите рассказать команде?"
        @input="autogrow"
        @keydown="onBodyKeydown"
        @contextmenu="onBodyContextmenu"
      />
      <div v-if="body.length > 9000" class="composer-counter">{{ body.length }}/10000</div>

      <!-- Медиа-сетка: превью картинок с удалением (как в соцсетях) -->
      <div v-if="mediaItems.length" class="composer-media" :class="`cols-${Math.min(mediaItems.length, 3)}`">
        <div v-for="m in mediaItems" :key="m.key" class="composer-media-item">
          <img :src="m.url" :alt="m.name" />
          <button
            type="button"
            class="composer-media-remove composer-media-edit"
            title="Редактировать"
            @click="m.file ? editImage(m.file) : editExisting(m.attachment)"
          >
            <span class="material-symbols-outlined">crop_rotate</span>
          </button>
          <button type="button" class="composer-media-remove" title="Убрать" @click="removeItem(m)">
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
      </div>

      <!-- Файлы-чипы -->
      <ul v-if="fileItems.length" class="composer-files">
        <li v-for="f in fileItems" :key="f.key" class="composer-file">
          <span class="material-symbols-outlined">description</span>
          <span class="composer-file-name">{{ f.name }}</span>
          <button type="button" class="composer-file-remove" title="Убрать" @click="removeItem(f)">
            <span class="material-symbols-outlined">close</span>
          </button>
        </li>
      </ul>

      <!-- «Добавить к посту» -->
      <div class="composer-add" :class="{ over: dragOver }">
        <span class="composer-add-label">Добавить к публикации</span>
        <div class="composer-add-actions">
          <button type="button" class="composer-add-btn photo" title="Фото" @click="photoInput?.click()">
            <span class="material-symbols-outlined">image</span>
          </button>
          <button type="button" class="composer-add-btn file" title="Файл" @click="fileInput?.click()">
            <span class="material-symbols-outlined">attach_file</span>
          </button>
        </div>
        <input ref="photoInput" type="file" accept="image/*" multiple hidden @change="onPick" />
        <input ref="fileInput" type="file" multiple hidden @change="onPick" />
      </div>
    </div>

    <!-- ПКМ на выделенном тексте — меню форматирования (как в заметках) -->
    <Teleport to="body">
      <Transition name="cfm">
        <div v-if="ctxMenu.visible" ref="ctxEl" class="cfm" :style="ctxStyle" role="menu" @click.stop>
          <div class="cfm-caption">
            <span class="material-symbols-outlined">match_case</span>
            <span>Форматирование</span>
          </div>
          <button
            v-for="b in TOOLBAR"
            :key="'ctx-' + b.icon"
            type="button"
            class="cfm-item"
            @mousedown.prevent
            @click="runCtx(b)"
          >
            <span class="material-symbols-outlined">{{ b.icon }}</span>
            <span>{{ b.title.replace(/ \(.+\)$/, '') }}</span>
          </button>
        </div>
      </Transition>
    </Teleport>

    <ImageEditDialog v-model="imageEditOpen" :file="imageEditTarget" @apply="applyEditedImage" />
  </AppDialog>
</template>

<script setup>
// Композер публикации в стиле соцсетей (Facebook/X): шапка с автором и
// выбором раздела, безрамочные поля, панель Markdown-форматирования с
// предпросмотром, сетка превью медиа с удалением, панель «Добавить к
// публикации». Текст поста — Markdown (лента рендерит его MarkdownView).
import { computed, nextTick, ref, watch } from 'vue'
import Select from 'primevue/select'
import AppDialog from '@/components/common/AppDialog.vue'
import ImageEditDialog from '@/components/common/ImageEditDialog.vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import { usePortalStore } from '@/stores/portal.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  post: { type: Object, default: null },
  // Предзаполнение при создании (публикация из заметки): {title, body}.
  preset: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const portal = usePortalStore()
const auth = useAuthStore()
const isEdit = computed(() => !!props.post)

const me = computed(() => portal.resolveAuthor(auth.userId))

const topicId = ref(null)
const title = ref('')
const body = ref('')
const preview = ref(false)
const pendingFiles = ref([])          // File[] — ещё не загруженные
const existingAttachments = ref([])   // вложения редактируемого поста
const removedIds = ref(new Set())     // существующие, помеченные к удалению
const dragOver = ref(false)
const saving = ref(false)
const bodyEl = ref(null)
const photoInput = ref(null)
const fileInput = ref(null)

let objectUrls = []

function reset() {
  topicId.value = props.post?.topic_id ?? null
  title.value = props.post?.title ?? props.preset?.title ?? ''
  body.value = props.post?.body ?? props.preset?.body ?? ''
  preview.value = false
  pendingFiles.value = []
  existingAttachments.value = props.post?.attachments ?? []
  removedIds.value = new Set()
  objectUrls.forEach((u) => URL.revokeObjectURL(u))
  objectUrls = []
  nextTick(autogrow)
}

// immediate — композер из заметок монтируется лениво (v-if) уже открытым:
// без немедленного вызова reset() preset-текст и разделы не подставились бы.
watch(() => props.modelValue, (v) => {
  if (!v) return
  reset()
  // Каталог авторов и разделы могли ещё не грузиться (открытие из заметок).
  portal.loadAuthors()
  if (!portal.topics.length) portal.fetchTopics()
}, { immediate: true })

// ── Медиа и файлы (pending + существующие, единый список для сетки) ──
const isImage = (mime) => !!mime?.startsWith('image/')

const mediaItems = computed(() => [
  ...existingAttachments.value
    .filter((a) => isImage(a.mime) && !removedIds.value.has(a.id))
    .map((a) => ({ key: 'a' + a.id, url: a.url, name: a.name, attachment: a })),
  ...pendingFiles.value
    .filter((f) => isImage(f.type))
    .map((f) => ({ key: 'p' + pendingFiles.value.indexOf(f), url: previewUrl(f), name: f.name, file: f })),
])

const fileItems = computed(() => [
  ...existingAttachments.value
    .filter((a) => !isImage(a.mime) && !removedIds.value.has(a.id))
    .map((a) => ({ key: 'a' + a.id, name: a.name, attachment: a })),
  ...pendingFiles.value
    .filter((f) => !isImage(f.type))
    .map((f) => ({ key: 'p' + pendingFiles.value.indexOf(f), name: f.name, file: f })),
])

const urlByFile = new Map()
function previewUrl(f) {
  if (!urlByFile.has(f)) {
    const u = URL.createObjectURL(f)
    urlByFile.set(f, u)
    objectUrls.push(u)
  }
  return urlByFile.get(f)
}

function removeItem(item) {
  if (item.file) pendingFiles.value = pendingFiles.value.filter((f) => f !== item.file)
  else removedIds.value = new Set([...removedIds.value, item.attachment.id])
}

function onPick(e) {
  pendingFiles.value.push(...Array.from(e.target.files || []))
  e.target.value = ''
}

function onDrop(e) {
  dragOver.value = false
  pendingFiles.value.push(...Array.from(e.dataTransfer?.files || []))
}

// ── Редактирование картинки (обрезка/поворот) до загрузки ──
const imageEditOpen = ref(false)
const imageEditTarget = ref(null)

function editImage(file) {
  imageEditTarget.value = file
  imageEditOpen.value = true
}

// Существующее вложение: скачиваем, редактируем как новый файл; оригинал
// помечается к удалению — итог применится при сохранении публикации.
async function editExisting(attachment) {
  try {
    const resp = await fetch(attachment.url)
    const blob = await resp.blob()
    const file = new File([blob], attachment.name || 'image', { type: attachment.mime || blob.type })
    removedIds.value = new Set([...removedIds.value, attachment.id])
    pendingFiles.value.push(file)
    editImage(file)
  } catch {
    useNotificationsStore().error('Не удалось открыть изображение для редактирования')
  }
}

function applyEditedImage(edited) {
  const i = pendingFiles.value.indexOf(imageEditTarget.value)
  if (i !== -1) pendingFiles.value.splice(i, 1, edited)
  imageEditTarget.value = null
}

// ── Markdown-хелперы над textarea ──
function applyEdit(next, selStart, selEnd) {
  body.value = next
  nextTick(() => {
    const el = bodyEl.value
    if (!el) return
    el.focus()
    el.setSelectionRange(selStart, selEnd)
    autogrow()
  })
}

// Обернуть выделение маркерами (**…**, `…` и т.п.).
function surround(mark, placeholder = 'текст') {
  const el = bodyEl.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e } = el
  const sel = body.value.slice(s, e) || placeholder
  const next = body.value.slice(0, s) + mark + sel + mark + body.value.slice(e)
  applyEdit(next, s + mark.length, s + mark.length + sel.length)
}

// Префикс каждой строки выделения (списки, цитаты); numbered — «1. 2. …».
// Повторный вызов с тем же префиксом снимает его.
function linePrefix(prefix, { numbered = false } = {}) {
  const el = bodyEl.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e } = el
  const from = body.value.lastIndexOf('\n', s - 1) + 1
  const to = body.value.indexOf('\n', e) === -1 ? body.value.length : body.value.indexOf('\n', e)
  const block = body.value.slice(from, to)
  const lines = block.split('\n')
  const allPrefixed = !numbered && lines.every((l) => l.startsWith(prefix))
  const next = lines
    .map((l, i) => {
      if (allPrefixed) return l.slice(prefix.length)
      const clean = l.replace(/^(\s*)([-*+]\s(\[[ xX]\]\s)?|\d+[.)]\s|>\s|#{1,3}\s)/, '')
      return numbered ? `${i + 1}. ${clean}` : prefix + clean
    })
    .join('\n')
  applyEdit(body.value.slice(0, from) + next + body.value.slice(to), from, from + next.length)
}

// Заголовок по циклу: нет → ## → ### → # → нет.
function cycleHeading() {
  const el = bodyEl.value
  if (!el) return
  const s = el.selectionStart
  const from = body.value.lastIndexOf('\n', s - 1) + 1
  const to = body.value.indexOf('\n', from) === -1 ? body.value.length : body.value.indexOf('\n', from)
  const line = body.value.slice(from, to)
  const m = line.match(/^(#{1,3})\s+(.*)$/)
  const level = m ? m[1].length : 0
  const rest = m ? m[2] : line
  const nextLevel = { 0: 2, 2: 3, 3: 1, 1: 0 }[level]
  const nextLine = nextLevel ? '#'.repeat(nextLevel) + ' ' + rest : rest
  applyEdit(body.value.slice(0, from) + nextLine + body.value.slice(to), from + nextLine.length, from + nextLine.length)
}

function insertLink() {
  const el = bodyEl.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e } = el
  const sel = body.value.slice(s, e) || 'текст'
  const inserted = `[${sel}](https://)`
  const next = body.value.slice(0, s) + inserted + body.value.slice(e)
  const urlStart = s + sel.length + 3
  applyEdit(next, urlStart, urlStart + 8)
}

function codeFence() {
  const el = bodyEl.value
  if (!el) return
  const { selectionStart: s, selectionEnd: e } = el
  const sel = body.value.slice(s, e) || 'код'
  const next = body.value.slice(0, s) + '```\n' + sel + '\n```' + body.value.slice(e)
  applyEdit(next, s + 4, s + 4 + sel.length)
}

const TOOLBAR = [
  { icon: 'format_bold', title: 'Жирный (⌘B)', run: () => surround('**') },
  { icon: 'format_italic', title: 'Курсив (⌘I)', run: () => surround('*') },
  { icon: 'strikethrough_s', title: 'Зачёркнутый', run: () => surround('~~') },
  { icon: 'code', title: 'Код', run: () => surround('`', 'код') },
  { icon: 'format_h2', title: 'Заголовок', run: cycleHeading },
  { icon: 'format_quote', title: 'Цитата', run: () => linePrefix('> ') },
  { icon: 'format_list_bulleted', title: 'Список', run: () => linePrefix('- ') },
  { icon: 'format_list_numbered', title: 'Нумерованный список', run: () => linePrefix('', { numbered: true }) },
  { icon: 'checklist', title: 'Чек-лист', run: () => linePrefix('- [ ] ') },
  { icon: 'link', title: 'Ссылка', run: insertLink },
  { icon: 'code_blocks', title: 'Блок кода', run: codeFence },
]

function onBodyKeydown(e) {
  if (!(e.metaKey || e.ctrlKey)) return
  const k = e.key.toLowerCase()
  if (k === 'b') { e.preventDefault(); surround('**') }
  else if (k === 'i') { e.preventDefault(); surround('*') }
}

// ── ПКМ-меню форматирования на выделении (как NoteSelectionMenu) ──
const ctxMenu = ref({ visible: false, x: 0, y: 0 })
const ctxEl = ref(null)

const ctxStyle = computed(() => ({
  position: 'fixed',
  left: ctxMenu.value.x + 'px',
  top: ctxMenu.value.y + 'px',
  zIndex: 12000,
}))

async function onBodyContextmenu(e) {
  const el = bodyEl.value
  // Без выделения — нативное меню (вставить, орфография и т.п.).
  if (!el || el.selectionStart === el.selectionEnd) return
  e.preventDefault()
  ctxMenu.value = { visible: true, x: e.clientX, y: e.clientY }
  await nextTick()
  const r = ctxEl.value?.getBoundingClientRect()
  if (!r) return
  const pad = 8
  let { x, y } = ctxMenu.value
  if (x + r.width > window.innerWidth - pad) x = window.innerWidth - r.width - pad
  if (y + r.height > window.innerHeight - pad) y = e.clientY - r.height - 6
  ctxMenu.value = { visible: true, x: Math.max(pad, x), y: Math.max(pad, y) }
}

function runCtx(b) {
  ctxMenu.value.visible = false
  b.run()
}

function onDocPointerDown(e) {
  if (ctxMenu.value.visible && !ctxEl.value?.contains(e.target)) ctxMenu.value.visible = false
}
function onDocKey(e) {
  if (e.key === 'Escape') ctxMenu.value.visible = false
}

watch(() => ctxMenu.value.visible, (v) => {
  const fn = v ? 'addEventListener' : 'removeEventListener'
  document[fn]('mousedown', onDocPointerDown, true)
  document[fn]('keydown', onDocKey)
  document[fn]('scroll', onDocKey, true)
})

function autogrow() {
  const el = bodyEl.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, window.innerHeight * 0.4) + 'px'
}

// ── Сохранение ──
async function submit() {
  const b = body.value.trim()
  if (!b) return
  saving.value = true
  try {
    const post = isEdit.value
      ? await portal.updatePost(props.post.id, { topicId: topicId.value, title: title.value, body: b })
      : await portal.createPost({ topicId: topicId.value, title: title.value, body: b })
    for (const id of removedIds.value) {
      await portal.deleteAttachment(post.id, id)
    }
    for (const f of pendingFiles.value) {
      await portal.uploadAttachment(post.id, f)
    }
    emit('update:modelValue', false)
    emit('saved', post)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось сохранить публикацию')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.composer {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.composer-author {
  display: flex;
  align-items: center;
  gap: 10px;
}

.composer-avatar {
  width: 42px;
  height: 42px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.composer-author-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.composer-author-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
}

/* Компактный селектор раздела — как выбор аудитории в соцсетях. */
.composer-topic {
  height: 28px;
  font-size: 12.5px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-dim);
  align-items: center;
  width: fit-content;
  min-width: 130px;
}
.composer-topic :deep(.p-select-label) { padding: 3px 4px 3px 12px; font-size: 12.5px; }
.composer-topic :deep(.p-select-dropdown) { width: 28px; }

.composer-title {
  border: none;
  background: transparent;
  outline: none;
  color: var(--color-text);
  font: inherit;
  font-size: 17px;
  font-weight: 700;
  padding: 2px 0;
}
.composer-title::placeholder { color: var(--color-text-dim); opacity: 0.6; }

.composer-toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-wrap: wrap;
  padding: 4px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
}

.composer-tbtn {
  width: 32px;
  height: 32px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
}
.composer-tbtn:hover:not(:disabled) { background: var(--color-surface-high); color: var(--color-text); }
.composer-tbtn:disabled { opacity: 0.35; cursor: default; }
.composer-tbtn .material-symbols-outlined { font-size: 19px; }

.composer-preview-btn {
  width: auto;
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  font-size: 12.5px;
  font-weight: 600;
}
.composer-preview-btn.active { background: var(--color-primary-container); color: var(--color-on-primary-container); }

.composer-body {
  width: 100%;
  min-height: 200px;
  border: none;
  background: transparent;
  outline: none;
  resize: none;
  color: var(--color-text);
  font: inherit;
  font-size: 15px;
  line-height: 1.5;
  padding: 2px 0;
  box-sizing: border-box;
}
.composer-body::placeholder { color: var(--color-text-dim); opacity: 0.6; }

.composer-preview {
  min-height: 200px;
  font-size: 15px;
  padding: 2px 0;
}

.composer-counter {
  align-self: flex-end;
  font-size: 12px;
  color: var(--color-text-dim);
}

.composer-media {
  display: grid;
  gap: 6px;
  grid-template-columns: repeat(3, 1fr);
}
.composer-media.cols-1 { grid-template-columns: 1fr; }
.composer-media.cols-2 { grid-template-columns: repeat(2, 1fr); }

.composer-media-item {
  position: relative;
  border-radius: var(--radius-md);
  overflow: hidden;
  aspect-ratio: 4 / 3;
  background: var(--color-surface-high);
}
.composer-media-item img { width: 100%; height: 100%; object-fit: cover; display: block; }

.composer-media-remove {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 50%;
  background: var(--color-scrim, var(--color-surface));
  color: var(--color-text);
  display: grid;
  place-items: center;
  cursor: pointer;
  box-shadow: var(--shadow-sm);
}
.composer-media-remove .material-symbols-outlined { font-size: 16px; }
.composer-media-remove:hover { color: var(--color-error); }

.composer-media-edit { right: 38px; }
.composer-media-edit:hover { color: var(--color-primary); }

.composer-files {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.composer-file {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--color-text);
}
.composer-file .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.composer-file-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.composer-file-remove {
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.composer-file-remove:hover { color: var(--color-error); }
.composer-file-remove .material-symbols-outlined { font-size: 16px; }

.composer-add {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  transition: border-color 0.15s, background 0.15s;
}
.composer-add.over {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
}

.composer-add-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text-dim);
}

.composer-add-actions {
  margin-left: auto;
  display: flex;
  gap: 4px;
}

.composer-add-btn {
  width: 36px;
  height: 36px;
  border: none;
  border-radius: 50%;
  background: transparent;
  display: grid;
  place-items: center;
  cursor: pointer;
}
.composer-add-btn:hover { background: var(--color-surface-high); }
.composer-add-btn.photo { color: var(--color-success, var(--color-primary)); }
.composer-add-btn.file { color: var(--color-tertiary); }
.composer-add-btn .material-symbols-outlined { font-size: 21px; }
</style>

<!-- ПКМ-меню телепортируется в body — стили вне scoped. -->
<style>
.cfm {
  min-width: 220px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md, 12px);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  gap: 1px;
  max-height: 60dvh;
  overflow-y: auto;
}

.cfm-caption {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px 4px;
  color: var(--color-primary);
  font-size: 11.5px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.4px;
}
.cfm-caption .material-symbols-outlined { font-size: 16px; }

.cfm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 12px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  font-weight: 500;
  text-align: left;
  border-radius: var(--radius-sm, 8px);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.cfm-item:hover { background: var(--color-surface-low); }
.cfm-item .material-symbols-outlined { font-size: 18px; color: var(--color-text-dim); }
.cfm-item:hover .material-symbols-outlined { color: var(--color-primary); }

.cfm-enter-active, .cfm-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.cfm-enter-from, .cfm-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
