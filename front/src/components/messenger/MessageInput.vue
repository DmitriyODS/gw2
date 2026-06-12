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

    <div v-if="pending.length || attachedTask || rec.active" class="pending-attachments">
      <div v-if="rec.active" class="pending-att pending-rec">
        <span class="rec-dot" aria-hidden="true"></span>
        <span class="pending-name">Запись экрана · {{ recTime }}</span>
        <button class="remove-att" @click="stopScreencast" title="Остановить запись">
          <span class="material-symbols-outlined">stop</span>
        </button>
      </div>
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
      <div ref="attachWrap" class="attach-wrap">
        <button
          class="attach-btn"
          :class="{ active: attachMenuOpen, recording: rec.active }"
          title="Прикрепить"
          type="button"
          aria-haspopup="menu"
          :aria-expanded="attachMenuOpen"
          @click="attachMenuOpen = !attachMenuOpen"
        >
          <span class="material-symbols-outlined">attach_file</span>
        </button>
        <Transition name="attach-menu">
          <div v-if="attachMenuOpen" class="attach-menu" role="menu">
            <button class="attach-menu-item" type="button" @click="pickFile">
              <span class="material-symbols-outlined attach-menu-ico tone-secondary">upload_file</span>
              <span>Файл</span>
            </button>
            <button v-if="canAttachTask" class="attach-menu-item" type="button" @click="pickTask">
              <span class="material-symbols-outlined attach-menu-ico tone-tertiary">task</span>
              <span>Задачу</span>
            </button>
            <button v-if="canScreencast" class="attach-menu-item" type="button" @click="pickScreencast">
              <span class="material-symbols-outlined attach-menu-ico tone-error">
                {{ rec.active ? 'stop_circle' : 'screen_record' }}
              </span>
              <span>{{ rec.active ? 'Остановить запись' : 'Запись экрана' }}</span>
            </button>
          </div>
        </Transition>
        <input
          ref="fileInput"
          type="file"
          multiple
          style="display:none"
          @change="onFiles"
        />
      </div>
      <textarea
        ref="textarea"
        v-model="text"
        :placeholder="placeholder"
        rows="1"
        class="text-area"
        enterkeyhint="enter"
        @keydown.enter.exact="onEnterKey"
        @input="autoresize"
        @paste="onPaste"
        @contextmenu="onTextContextMenu"
      />
      <button
        class="send-btn"
        :disabled="!canSend"
        @click="submit"
        :title="isTouchDevice ? 'Отправить' : 'Отправить (Enter)'"
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

/* На смартфоне Enter экранной клавиатуры набирает новую строку, отправка —
   только кнопкой: случайные отправки с тач-клавиатуры раздражают сильнее,
   чем лишний тап. На десктопе Enter отправляет, Shift+Enter — новая строка. */
const isTouchDevice = window.matchMedia?.('(hover: none) and (pointer: coarse)').matches ?? false

function onEnterKey(e) {
  if (isTouchDevice) return // default-поведение textarea — перенос строки
  e.preventDefault()
  submit()
}

/* ── Меню «что прикрепить» (файл / задача / запись экрана) ── */
const attachMenuOpen = ref(false)
const attachWrap = ref(null)
const fileInput = ref(null)

function pickFile() {
  attachMenuOpen.value = false
  fileInput.value?.click()
}

function pickTask() {
  attachMenuOpen.value = false
  emit('attach-task')
}

function pickScreencast() {
  attachMenuOpen.value = false
  toggleScreencast()
}

function onDocCloseAttachMenu(e) {
  if (!attachMenuOpen.value) return
  if (attachWrap.value?.contains(e.target)) return
  attachMenuOpen.value = false
}

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
  document.addEventListener('mousedown', onDocCloseAttachMenu, true)
  document.addEventListener('touchstart', onDocCloseAttachMenu, { capture: true, passive: true })
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocClickClose, true)
  document.removeEventListener('mousedown', onDocCloseAttachMenu, true)
  document.removeEventListener('touchstart', onDocCloseAttachMenu, { capture: true })
  stopScreencast()
})

const replyAuthor = computed(() => props.replyTo?.sender_fio || '')
const replyPreview = computed(() => {
  const r = props.replyTo
  if (!r) return ''
  if (r.kind === 'call') return '📞 Звонок'
  if (r.text) return r.text
  if (r.has_attachments || r.attachments?.length) return 'Вложение'
  return 'Сообщение'
})

function autoresize() {
  const el = textarea.value
  if (!el) return
  const max = 180
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, max) + 'px'
  // Скроллбар — только когда текст упёрся в максимальную высоту.
  el.style.overflowY = el.scrollHeight > max ? 'auto' : 'hidden'
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

/* ── Скринкаст: запись экрана → видео-вложение ─────────────────────
   getDisplayMedia + MediaRecorder, готовый файл уходит обычным upload'ом
   (рендерится получателю штатным <video> в AttachmentView). */
const SCREENCAST_MAX_BYTES = 24 * 1024 * 1024 // запас до серверного лимита 25 МБ

const rec = ref({ active: false, seconds: 0 })
let recRecorder = null
let recStream = null
let recChunks = []
let recBytes = 0
let recTimer = null
let recLimitHit = false

const canScreencast = computed(() =>
  !!navigator.mediaDevices?.getDisplayMedia && typeof window.MediaRecorder === 'function')

const recTime = computed(() => {
  const s = rec.value.seconds
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, '0')}`
})

function recMimeType() {
  const candidates = [
    'video/webm;codecs=vp9,opus',
    'video/webm;codecs=vp8,opus',
    'video/webm',
    'video/mp4', // Safari не умеет webm
  ]
  return candidates.find(t => MediaRecorder.isTypeSupported(t)) || ''
}

async function toggleScreencast() {
  if (rec.value.active) {
    stopScreencast()
    return
  }
  let stream
  try {
    stream = await navigator.mediaDevices.getDisplayMedia({
      video: { frameRate: 15 },
      audio: true,
    })
  } catch {
    return // пользователь отменил выбор экрана — не ошибка
  }
  const mime = recMimeType()
  let recorder
  try {
    recorder = new MediaRecorder(stream, {
      ...(mime ? { mimeType: mime } : {}),
      videoBitsPerSecond: 2_000_000,
      audioBitsPerSecond: 128_000,
    })
  } catch (err) {
    stream.getTracks().forEach(t => t.stop())
    console.error('screencast init failed', err)
    window.alert('Запись экрана не поддерживается этим браузером')
    return
  }
  recStream = stream
  recRecorder = recorder
  recChunks = []
  recBytes = 0
  recLimitHit = false
  recorder.ondataavailable = (e) => {
    if (!e.data?.size) return
    recChunks.push(e.data)
    recBytes += e.data.size
    if (recBytes >= SCREENCAST_MAX_BYTES && rec.value.active) {
      recLimitHit = true
      stopScreencast()
    }
  }
  recorder.onstop = onScreencastStop
  // Запись можно остановить и из браузерной панели «Доступ к экрану».
  stream.getVideoTracks()[0]?.addEventListener('ended', stopScreencast)
  recorder.start(1000)
  rec.value = { active: true, seconds: 0 }
  recTimer = setInterval(() => { rec.value.seconds++ }, 1000)
}

function stopScreencast() {
  if (!rec.value.active) return
  rec.value = { ...rec.value, active: false }
  clearInterval(recTimer)
  recTimer = null
  try {
    if (recRecorder && recRecorder.state !== 'inactive') recRecorder.stop()
  } catch {/* recorder уже мёртв */}
  recStream?.getTracks().forEach(t => t.stop())
}

async function onScreencastStop() {
  const chunks = recChunks
  recChunks = []
  recRecorder = null
  recStream = null
  if (!chunks.length) return
  const type = chunks[0].type || 'video/webm'
  const ext = type.includes('mp4') ? 'mp4' : 'webm'
  const stamp = new Date().toISOString().slice(0, 19).replace(/[T:]/g, '-')
  const file = new File(chunks, `screencast-${stamp}.${ext}`, { type })
  if (recLimitHit) {
    window.alert('Запись достигла лимита 25 МБ и была остановлена — ролик прикреплён к сообщению')
  }
  await uploadFiles([file])
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

/* Единственная кнопка-скрепка: остальные варианты вложений — в выпадающем
   меню над ней (файл / задача / запись экрана). */
.attach-wrap {
  position: relative;
  flex-shrink: 0;
}

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

.attach-btn:hover,
.attach-btn.active {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.attach-btn:active { transform: scale(0.94); }

.attach-btn .material-symbols-outlined { font-size: 20px; }

/* Идёт запись экрана — скрепка «горит», чтобы состояние было видно
   и при закрытом меню. */
.attach-btn.recording {
  background: var(--color-error-container);
  color: var(--color-error);
  animation: recPulse 1.4s ease-in-out infinite;
}

.attach-menu {
  position: absolute;
  bottom: calc(100% + 8px);
  left: 0;
  min-width: 210px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  z-index: 90;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.attach-menu-item {
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
  transition: background 0.15s;
}

.attach-menu-item:hover { background: var(--color-surface-low); }

.attach-menu-ico { font-size: 20px; }
.attach-menu-ico.tone-secondary { color: var(--color-secondary); }
.attach-menu-ico.tone-tertiary { color: var(--color-tertiary); }
.attach-menu-ico.tone-error { color: var(--color-error); }

.attach-menu-enter-active,
.attach-menu-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
  transform-origin: bottom left;
}

.attach-menu-enter-from,
.attach-menu-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(4px);
}

@keyframes recPulse {
  0%, 100% { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-error) 40%, transparent); }
  50%      { box-shadow: 0 0 0 6px color-mix(in oklch, var(--color-error) 0%, transparent); }
}

/* Чип идущей записи экрана */
.pending-att.pending-rec {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-color: transparent;
  border-radius: var(--radius-full);
  font-weight: 600;
}

.rec-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-error);
  flex-shrink: 0;
  animation: recPulse 1.4s ease-in-out infinite;
}

.pending-att.pending-rec .remove-att { color: var(--color-on-error-container); }

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
  overflow-y: hidden;
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
