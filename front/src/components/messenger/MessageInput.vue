<template>
  <div class="msg-input">
    <div v-if="pending.length" class="pending-attachments">
      <div v-for="(p, i) in pending" :key="p.id ?? ('tmp-' + i)" class="pending-att">
        <span v-if="p.uploading" class="pending-name uploading">
          <ProgressSpinner style="width:16px;height:16px" />
          {{ p.file_name }}
        </span>
        <template v-else>
          <span class="material-symbols-outlined att-ico">{{ iconFor(p.mime_type) }}</span>
          <span class="pending-name">{{ p.file_name }}</span>
        </template>
        <button class="remove-att" @click="removePending(i)" title="Убрать">
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
})

const emit = defineEmits(['send'])

const text = ref('')
const pending = ref([])
const textarea = ref(null)

const canSend = computed(() => {
  if (props.sending) return false
  if (pending.value.some(p => p.uploading)) return false
  return Boolean(text.value.trim()) || pending.value.length > 0
})

function autoresize() {
  const el = textarea.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 180) + 'px'
}

async function onFiles(e) {
  const files = Array.from(e.target.files || [])
  e.target.value = ''
  for (const file of files) {
    const tmp = {
      file_name: file.name,
      mime_type: file.type,
      size_bytes: file.size,
      uploading: true,
    }
    pending.value.push(tmp)
    try {
      const att = await uploadAttachment(file)
      Object.assign(tmp, att, { uploading: false })
    } catch (err) {
      pending.value = pending.value.filter(p => p !== tmp)
      console.error('upload failed', err)
      window.alert(err?.message || 'Не удалось загрузить файл')
    }
  }
}

function removePending(i) {
  pending.value.splice(i, 1)
}

async function submit() {
  if (!canSend.value) return
  const payload = {
    text: text.value.trim(),
    attachment_ids: pending.value.filter(p => p.id).map(p => p.id),
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
  border-top: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  padding: 10px 14px env(safe-area-inset-bottom, 10px);
}

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
