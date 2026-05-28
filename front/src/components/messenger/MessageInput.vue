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

    <div v-if="pending.length" class="pending-attachments">
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
      <textarea
        ref="textarea"
        v-model="text"
        :placeholder="placeholder"
        rows="1"
        class="text-area"
        @keydown.enter.exact.prevent="submit"
        @input="autoresize"
        @paste="onPaste"
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
  </div>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { uploadAttachment } from '@/api/messenger.js'
import ProgressSpinner from 'primevue/progressspinner'

const props = defineProps({
  placeholder: { type: String, default: 'Напишите сообщение…' },
  sending: { type: Boolean, default: false },
  replyTo: { type: Object, default: null },
})

const emit = defineEmits(['send', 'cancel-reply'])

const text = ref('')
const pending = ref([])
const textarea = ref(null)

const canSend = computed(() => {
  if (props.sending) return false
  if (pending.value.some(p => p.uploading)) return false
  return Boolean(text.value.trim()) || pending.value.length > 0
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
  }
  emit('send', payload)
  text.value = ''
  pending.value = []
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

.attach-btn {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  color: var(--color-text-dim);
  border-radius: var(--radius-md);
}

.attach-btn:hover {
  background: var(--color-surface-low);
  color: var(--color-primary);
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
</style>
