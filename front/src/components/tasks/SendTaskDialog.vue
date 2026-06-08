<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="send"
    size="sm"
    title="Отправить задачу"
    :subtitle="task ? `«${task.name}»` : ''"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <!-- Шаг 1: выбор получателя -->
    <template v-if="!picked">
      <div class="sendtask-search">
        <span class="material-symbols-outlined">search</span>
        <input
          v-model="q"
          placeholder="Кому отправить? Логин или фамилия"
          class="sendtask-input"
          autofocus
        />
      </div>
      <div v-if="loading" class="sendtask-empty">
        <ProgressSpinner style="width:32px;height:32px" />
      </div>
      <div v-else-if="!results.length" class="sendtask-empty">
        <span class="material-symbols-outlined">person_search</span>
        <p>{{ q ? 'Никого не нашли' : 'Начните вводить' }}</p>
      </div>
      <ul v-else class="sendtask-results">
        <li
          v-for="u in results"
          :key="u.id"
          class="sendtask-item"
          @click="pickUser(u)"
        >
          <img class="sendtask-avatar" :src="avatarOf(u)" :alt="u.fio" />
          <div class="sendtask-info">
            <div class="sendtask-name">{{ u.fio }}</div>
            <div class="sendtask-meta">@{{ u.login }} · {{ u.post || u.role?.name }}</div>
          </div>
        </li>
      </ul>
    </template>

    <!-- Шаг 2: подпись -->
    <template v-else>
      <div class="picked-row">
        <img class="sendtask-avatar small" :src="avatarOf(picked)" :alt="picked.fio" />
        <div class="sendtask-info">
          <div class="sendtask-name">{{ picked.fio }}</div>
          <div class="sendtask-meta">@{{ picked.login }}</div>
        </div>
        <button class="picked-change" @click="picked = null" :disabled="sending" title="Выбрать другого">
          <span class="material-symbols-outlined">person_remove</span>
        </button>
      </div>

      <textarea
        v-model="caption"
        class="sendtask-caption"
        rows="3"
        placeholder="Подпись (необязательно)"
        :disabled="sending"
        @keydown="onCaptionKey"
      />

      <div class="sendtask-actions">
        <button class="btn-text" :disabled="sending" @click="cancel">Отмена</button>
        <button class="btn-filled" :disabled="sending" @click="confirm">
          <span class="material-symbols-outlined">send</span>
          {{ sending ? 'Отправка…' : 'Отправить' }}
        </button>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
import { ref, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import { getDirectory } from '@/api/users.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  task: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const q = ref('')
const results = ref([])
const loading = ref(false)
const picked = ref(null)
const caption = ref('')
const sending = ref(false)
let debounceTimer = null

async function search() {
  loading.value = true
  try {
    results.value = await getDirectory(q.value.trim(), /* excludeSelf */ true)
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (v) => {
  if (v) {
    q.value = ''
    picked.value = null
    caption.value = ''
    sending.value = false
    search()
  }
})

watch(q, () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(search, 200)
})

function avatarOf(u) {
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function pickUser(u) {
  picked.value = u
}

function cancel() {
  emit('update:modelValue', false)
}

function confirm() {
  if (!picked.value || sending.value) return
  sending.value = true
  emit('confirm', { user: picked.value, text: caption.value.trim() })
}

// Родитель вызывает stopSending() через ref после завершения отправки —
// иначе кнопка останется «крутиться» при ошибке.
function stopSending() { sending.value = false }
defineExpose({ stopSending })

function onCaptionKey(e) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    confirm()
  }
}
</script>

<style scoped>
.sendtask-search {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.sendtask-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.sendtask-input {
  width: 100%;
  padding: 10px 12px 10px 40px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  outline: none;
}

.sendtask-input:focus { border-color: var(--color-primary); }

.sendtask-results {
  list-style: none;
  padding: 0;
  margin: 0;
  max-height: 44vh;
  overflow-y: auto;
}

.sendtask-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 10px 8px;
  cursor: pointer;
  border-radius: var(--radius-md);
}

.sendtask-item:hover { background: var(--color-surface-low); }

.sendtask-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.sendtask-avatar.small { width: 36px; height: 36px; }

.sendtask-info { min-width: 0; flex: 1; }

.sendtask-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.sendtask-meta {
  font-size: 12px;
  color: var(--color-text-dim);
}

.sendtask-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  color: var(--color-text-dim);
}

.sendtask-empty .material-symbols-outlined { font-size: 40px; }

.picked-row {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 10px 12px;
  background: var(--color-surface-low);
  border-radius: var(--radius-md);
  margin-bottom: 12px;
}

.picked-change {
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--color-text-dim);
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.picked-change:hover { background: var(--color-surface-high); color: var(--color-text); }

.sendtask-caption {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  resize: vertical;
  outline: none;
  min-height: 72px;
}
.sendtask-caption:focus { border-color: var(--color-primary); }

.sendtask-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
}

.btn-filled, .btn-text {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 40px;
  padding: 0 18px;
  border-radius: var(--radius-full, 20px);
  border: none;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.btn-filled { background: var(--color-primary); color: var(--color-on-primary); }
.btn-filled:hover:not(:disabled) { background: color-mix(in oklch, var(--color-primary) 90%, black); }
.btn-text { background: transparent; color: var(--color-text); }
.btn-text:hover:not(:disabled) { background: var(--color-surface-high); }
button:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-filled .material-symbols-outlined { font-size: 18px; }
</style>
