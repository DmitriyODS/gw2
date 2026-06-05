<template>
  <div class="msg-input">
    <div v-if="replyTo" class="reply-banner">
      <span class="material-symbols-outlined reply-ico">reply</span>
      <div class="reply-body">
        <span class="reply-author">{{ replyTo.sender_fio || replyAuthor || 'Ответ' }}</span>
        <span class="reply-text">{{ replyPreview }}</span>
      </div>
      <button class="reply-cancel" @click="$emit('cancel-reply')" title="Отменить">
        <span class="material-symbols-outlined">close</span>
      </button>
    </div>

    <div v-if="pending.length || attachedTask" class="pending-attachments">
      <div v-if="attachedTask" class="pending-att pending-task">
        <span class="material-symbols-outlined att-ico">task</span>
        <span class="pending-name">{{ attachedTask.name }}</span>
        <button class="remove-att" @click="attachedTask = null" title="Убрать">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>
      <div v-for="p in pending" :key="p._key" class="pending-att">
        <span v-if="p.uploading" class="pending-name uploading">
          <ProgressSpinner style="width:16px;height:16px" />
          {{ p.file_name }}
        </span>
        <template v-else>
          <span class="material-symbols-outlined att-ico">{{ iconFor(p.mime_type) }}</span>
          <span class="pending-name">{{ p.file_name }}</span>
        </template>
        <button class="remove-att" @click="removePending(p._key)" title="Убрать">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>
    </div>

    <div class="input-row">
      <label class="attach-btn" title="Прикрепить файл">
        <span class="material-symbols-outlined">attach_file</span>
        <input
          type="file"
          multiple
          @change="onFiles"
          style="display:none"
        />
      </label>
      <button
        v-if="canAttachTask"
        class="attach-btn attach-btn--task"
        title="Прикрепить задачу"
        type="button"
        @click="$emit('attach-task')"
      >
        <span class="material-symbols-outlined">task</span>
      </button>
      <textarea
        ref="textarea"
        v-model="text"
        :placeholder="placeholder"
        rows="1"
        class="text-area"
        @keydown.enter.exact.prevent="submit"
        @input="autoresize"
        @paste="onPaste"
        @contextmenu="onTextContextMenu"
      />
      <button
        class="send-btn"
        :disabled="!canSend"
        @click="submit"
        title="Отправить (Enter)"
      >
        <span class="material-symbols-outlined">send</span>
      </button>
    </div>

    <Teleport to="body">
      <Transition name="md-toolbar">
        <div
          v-if="mdToolbar.visible"
          class="md-toolbar"
          :style="mdToolbar.style"
          @mousedown.prevent
        >
          <button v-for="t in MD_TOOLS" :key="t.key" class="md-tool"
                  :title="t.label"
                  @click="applyMarkdown(t)">
            <span class="material-symbols-outlined">{{ t.icon }}</span>
          </button>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onBeforeUnmount, watch } from 'vue'
import { uploadAttachment } from '@/api/messenger.js'
import ProgressSpinner from 'primevue/progressspinner'

const props = defineProps({
  placeholder: { type: String, default: 'Напишите сообщение…' },
  sending: { type: Boolean, default: false },
  replyTo: { type: Object, default: null },
  attachedTask: { type: Object, default: null },
  canAttachTask: { type: Boolean, default: true },
})

const emit = defineEmits(['send', 'cancel-reply', 'attach-task', 'update:attachedTask'])

const text = ref('')
const pending = ref([])
const textarea = ref(null)

const attachedTask = computed({
  get: () => props.attachedTask,
  set: (v) => emit('update:attachedTask', v),
})

const canSend = computed(() => {
  if (props.sending) return false
  if (pending.value.some(p => p.uploading)) return false
  return Boolean(text.value.trim()) || pending.value.length > 0 || !!attachedTask.value
})

/* ── Контекстное меню Markdown по правому клику на выделении ── */
const MD_TOOLS = [
  { key: 'bold', label: 'Жирный', icon: 'format_bold', wrap: '**' },
  { key: 'italic', label: 'Курсив', icon: 'format_italic', wrap: '*' },
  { key: 'strike', label: 'Зачёркнутый', icon: 'format_strikethrough', wrap: '~~' },
  { key: 'code', label: 'Код', icon: 'code', wrap: '`' },
  { key: 'h1', label: 'Заголовок', icon: 'title', prefix: '# ' },
  { key: 'h2', label: 'Подзаголовок', icon: 'format_h2', prefix: '## ' },
  { key: 'link', label: 'Ссылка', icon: 'link', linkify: true },
]

const mdToolbar = ref({ visible: false, x: 0, y: 0, range: null })
const mdStyle = computed(() => ({
  position: 'fixed',
  left: mdToolbar.value.x + 'px',
  top: mdToolbar.value.y + 'px',
  zIndex: 12000,
}))
mdToolbar.value.style = mdStyle.value

watch([() => mdToolbar.value.x, () => mdToolbar.value.y], () => {
  mdToolbar.value.style = mdStyle.value
})

function onTextContextMenu(e) {
  const el = textarea.value
  if (!el) return
  const start = el.selectionStart
  const end = el.selectionEnd
  if (start === end) return  // нет выделения — не перехватываем стандартное меню
  e.preventDefault()
  mdToolbar.value = {
    visible: true,
    x: e.clientX,
    y: e.clientY,
    range: { start, end },
    style: { position: 'fixed', left: e.clientX + 'px', top: e.clientY + 'px', zIndex: 12000 },
  }
}

function closeMdToolbar() { mdToolbar.value = { ...mdToolbar.value, visible: false } }

function applyMarkdown(tool) {
  const range = mdToolbar.value.range
  if (!range) return closeMdToolbar()
  const before = text.value.slice(0, range.start)
  const sel = text.value.slice(range.start, range.end)
  const after = text.value.slice(range.end)
  let replaced
  if (tool.wrap) {
    replaced = `${tool.wrap}${sel}${tool.wrap}`
  } else if (tool.prefix) {
    replaced = `${tool.prefix}${sel}`
  } else if (tool.linkify) {
    const url = window.prompt('Адрес ссылки (URL)', 'https://')
    if (!url) return closeMdToolbar()
    replaced = `[${sel}](${url})`
  }
  text.value = before + replaced + after
  closeMdToolbar()
  nextTick(() => {
    textarea.value?.focus()
    autoresize()
  })
}

function onDocClickClose(e) {
  if (!mdToolbar.value.visible) return
  // Кнопки в Teleport-tooltip имеют .md-tool — игнорируем клики по ним.
  if (e.target.closest?.('.md-toolbar')) return
  closeMdToolbar()
}

onMounted(() => {
  document.addEventListener('mousedown', onDocClickClose, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocClickClose, true)
})

const replyAuthor = computed(() => props.replyTo?.sender_fio || '')
const replyPreview = computed(() => {
  const r = props.replyTo
  if (!r) return ''
  if (r.text) return r.text
  if (r.has_attachments || r.attachments?.length) return 'Вложение'
  return 'Сообщение'
})

function autoresize() {
  const el = textarea.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 180) + 'px'
}

/* Заметка по реактивности: pending хранит plain-objects, но при push() в ref
   они оборачиваются Proxy. Мутация исходной ссылки `tmp` не доходит до Proxy,
   и спиннер на чипе зависает, а canSend не разрешает отправку. Поэтому после
   загрузки находим элемент по локальному ключу и заменяем его новым объектом. */
async function uploadFiles(files) {
  for (const file of files) {
    if (!file) continue
    const key = `${Date.now()}-${Math.random().toString(36).slice(2)}`
    pending.value.push({
      _key: key,
      file_name: file.name || 'файл',
      mime_type: file.type,
      size_bytes: file.size,
      uploading: true,
    })
    try {
      const att = await uploadAttachment(file)
      const idx = pending.value.findIndex(p => p._key === key)
      if (idx !== -1) {
        pending.value[idx] = { ...pending.value[idx], ...att, uploading: false }
      }
    } catch (err) {
      pending.value = pending.value.filter(p => p._key !== key)
      console.error('upload failed', err)
      window.alert(err?.message || 'Не удалось загрузить файл')
    }
  }
}

async function onFiles(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  await uploadFiles(files)
}

/* Drag-and-drop обрабатывает родитель (вся область чата), а не только это
   поле — чтобы файл можно было бросить куда угодно в переписку. Сюда файлы
   приходят через exposed addFiles(). */
defineExpose({ addFiles: uploadFiles })

/* Вставка из буфера: картинки/файлы скриншотов приходят как items типа file. */
async function onPaste(e) {
  const items = Array.from(e.clipboardData?.items || [])
  const files = items
    .filter(it => it.kind === 'file')
    .map(it => it.getAsFile())
    .filter(Boolean)
  if (files.length) {
    e.preventDefault()
    await uploadFiles(files)
  }
}

function removePending(key) {
  pending.value = pending.value.filter(p => p._key !== key)
}

async function submit() {
  if (!canSend.value) return
  const payload = {
    text: text.value.trim(),
    attachment_ids: pending.value.filter(p => p.id).map(p => p.id),
    reply_to_id: props.replyTo?.id || null,
    task_id: attachedTask.value?.id || null,
  }
  emit('send', payload)
  text.value = ''
  pending.value = []
  emit('update:attachedTask', null)
  await nextTick()
  autoresize()
}

function iconFor(mime) {
  if (!mime) return 'attach_file'
  if (mime.startsWith('image/')) return 'image'
  if (mime.startsWith('video/')) return 'videocam'
  if (mime.startsWith('audio/')) return 'music_note'
  return 'description'
}
</script>

<style scoped>
.msg-input {
  position: relative;
  border-top: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  /* Safe-area под полем добавляет родитель (chat-panel / mini-thread), чтобы
     отступ не складывался дважды на мобильном. Сам ввод держим компактным. */
  padding: 10px 14px 12px;
}

.reply-banner {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  margin-bottom: 8px;
  background: var(--color-surface-low);
  border-left: 3px solid var(--color-primary);
  border-radius: var(--radius-sm);
}

.reply-banner .reply-ico {
  font-size: 20px;
  color: var(--color-primary);
  flex-shrink: 0;
}

.reply-body {
  display: flex;
  flex-direction: column;
  min-width: 0;
  flex: 1;
}

.reply-author {
  font-size: 12px;
  font-weight: 700;
  color: var(--color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.reply-text {
  font-size: 12.5px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.reply-cancel {
  background: none;
  border: none;
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  padding: 4px;
  border-radius: 50%;
  flex-shrink: 0;
}

.reply-cancel:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.reply-cancel .material-symbols-outlined { font-size: 18px; }

.pending-attachments {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 8px;
}

.pending-att {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  font-size: 12px;
  color: var(--color-text);
  max-width: 220px;
}

.pending-name {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 160px;
}

.pending-name.uploading {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--color-text-dim);
}

.att-ico { font-size: 18px; color: var(--color-primary); }

.remove-att {
  background: none;
  border: none;
  color: var(--color-text-dim);
  cursor: pointer;
  padding: 0;
  display: flex;
  align-items: center;
}

.remove-att .material-symbols-outlined { font-size: 16px; }

.input-row {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}

/* M3 Expressive icon-button: tonal hover, круглая форма,
   semantic-токены — без серых «безликих» иконок. Базовый тон — secondary
   (для файлов); модификатор --task — tertiary (задача семантически другой
   тип вложения). */
.attach-btn {
  appearance: none;
  border: none;
  width: 40px;
  height: 40px;
  display: grid;
  place-items: center;
  cursor: pointer;
  border-radius: 50%;
  background: var(--color-surface-high);
  color: var(--color-text);
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s, transform 0.12s;
}

.attach-btn:hover {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.attach-btn:active { transform: scale(0.94); }

.attach-btn .material-symbols-outlined { font-size: 20px; }

.attach-btn--task {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.attach-btn--task:hover {
  background: color-mix(in oklch, var(--color-tertiary) 26%, var(--color-tertiary-container));
  color: var(--color-on-tertiary-container);
}
.attach-btn--task .material-symbols-outlined {
  font-variation-settings: 'FILL' 1, 'wght' 500;
}

.text-area {
  flex: 1;
  resize: none;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  font: inherit;
  font-size: 14px;
  background: var(--color-surface-low);
  color: var(--color-text);
  outline: none;
  max-height: 180px;
  min-height: 40px;
  line-height: 1.4;
}

.text-area:focus { border-color: var(--color-primary); }

.send-btn {
  width: 40px;
  height: 40px;
  border-radius: var(--radius-md);
  border: none;
  background: var(--color-primary);
  color: var(--color-on-primary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, opacity 0.15s;
}

.send-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.send-btn:not(:disabled):hover {
  background: var(--color-primary-hover);
}

/* Прикреплённая задача — pill с tertiary-tone, в одной гамме с
   .attach-btn--task. Чтобы файл и задача визуально отличались, файловый
   чип остаётся surface, а задача — окрашенный pill. */
.pending-att.pending-task {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-color: transparent;
  border-radius: var(--radius-full);
  padding: 6px 10px 6px 8px;
  font-weight: 600;
}

.pending-att.pending-task .att-ico {
  color: var(--color-on-tertiary-container);
  font-variation-settings: 'FILL' 1, 'wght' 500;
}

.pending-att.pending-task .remove-att {
  color: var(--color-on-tertiary-container);
  opacity: 0.85;
}
.pending-att.pending-task .remove-att:hover { opacity: 1; }
</style>

<style>
/* Контекстное Markdown-меню (Teleport в body — без scoped). */
.md-toolbar {
  display: flex;
  gap: 2px;
  padding: 4px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
}

.md-tool {
  width: 32px;
  height: 32px;
  border-radius: var(--radius-sm);
  border: none;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
}
.md-tool:hover { background: var(--color-surface-low); }
.md-tool .material-symbols-outlined { font-size: 18px; }

.md-toolbar-enter-active, .md-toolbar-leave-active {
  transition: opacity 0.12s, transform 0.12s;
}
.md-toolbar-enter-from, .md-toolbar-leave-to {
  opacity: 0; transform: translateY(-4px);
}
</style>
