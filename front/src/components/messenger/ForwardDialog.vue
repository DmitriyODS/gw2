<template>
  <Dialog
    :visible="modelValue"
    @update:visible="$emit('update:modelValue', $event)"
    modal
    header="Переслать сообщение"
    :draggable="false"
    :style="{ width: '440px', maxWidth: '92vw' }"
  >
    <div class="fwd">
      <div v-if="preview" class="fwd-preview">
        <span class="material-symbols-outlined">forward</span>
        <span class="fwd-preview-text">{{ preview }}</span>
      </div>

      <div class="fwd-search">
        <span class="material-symbols-outlined">search</span>
        <input
          v-model="q"
          placeholder="Кому переслать — имя или логин"
          class="fwd-input"
          autofocus
        />
      </div>

      <div v-if="loading" class="fwd-empty">
        <ProgressSpinner style="width:32px;height:32px" />
      </div>
      <div v-else-if="!items.length" class="fwd-empty">
        <span class="material-symbols-outlined">person_search</span>
        <p>{{ q ? 'Никого не нашли' : 'Начните вводить' }}</p>
      </div>
      <ul v-else class="fwd-list">
        <li
          v-for="u in items"
          :key="u.id"
          class="fwd-item"
          :class="{ selected: selectedIds.has(u.id) }"
          @click="toggle(u.id)"
        >
          <img class="fwd-avatar" :src="avatarOf(u)" :alt="u.fio" />
          <div class="fwd-info">
            <div class="fwd-name">{{ u.fio }}</div>
            <div class="fwd-meta">@{{ u.login }} · {{ u.post || u.role?.name }}</div>
          </div>
          <span class="fwd-check">
            <span class="material-symbols-outlined">
              {{ selectedIds.has(u.id) ? 'check_circle' : 'radio_button_unchecked' }}
            </span>
          </span>
        </li>
      </ul>

      <div class="fwd-actions">
        <button class="btn-text" @click="cancel">Отмена</button>
        <button class="btn-filled" :disabled="!selectedIds.size || sending" @click="confirm">
          <span class="material-symbols-outlined">send</span>
          Переслать{{ selectedIds.size ? ` (${selectedIds.size})` : '' }}
        </button>
      </div>
    </div>
  </Dialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import Dialog from 'primevue/dialog'
import ProgressSpinner from 'primevue/progressspinner'
import { getDirectory } from '@/api/users.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  message: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const q = ref('')
const results = ref([])
const loading = ref(false)
const sending = ref(false)
const selectedIds = ref(new Set())
let debounceTimer = null

const items = computed(() => results.value)

const preview = computed(() => {
  const m = props.message
  if (!m) return ''
  if (m.text) return m.text.length > 80 ? m.text.slice(0, 80) + '…' : m.text
  if (m.attachments?.length) return 'Вложение'
  return 'Сообщение'
})

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
    selectedIds.value = new Set()
    sending.value = false
    search()
  }
})

watch(q, () => {
  clearTimeout(debounceTimer)
  debounceTimer = setTimeout(search, 200)
})

function toggle(id) {
  const next = new Set(selectedIds.value)
  if (next.has(id)) next.delete(id)
  else next.add(id)
  selectedIds.value = next
}

function avatarOf(u) {
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function cancel() {
  emit('update:modelValue', false)
}

async function confirm() {
  if (!selectedIds.value.size) return
  sending.value = true
  emit('confirm', { userIds: [...selectedIds.value] })
}

defineExpose({ stopSending: () => { sending.value = false } })
</script>

<style scoped>
.fwd-preview {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  margin-bottom: 12px;
  background: var(--color-surface-low);
  border-left: 3px solid var(--color-primary);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--color-text-dim);
}

.fwd-preview .material-symbols-outlined {
  font-size: 18px;
  color: var(--color-primary);
}

.fwd-preview-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.fwd-search {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.fwd-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.fwd-input {
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

.fwd-input:focus { border-color: var(--color-primary); }

.fwd-list {
  list-style: none;
  padding: 0;
  margin: 0 0 14px;
  max-height: 42vh;
  overflow-y: auto;
}

.fwd-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 10px 8px;
  cursor: pointer;
  border-radius: var(--radius-md);
}

.fwd-item:hover { background: var(--color-surface-low); }

.fwd-item.selected { background: var(--color-primary-container); }

.fwd-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.fwd-info { min-width: 0; flex: 1; }

.fwd-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.fwd-item.selected .fwd-name { color: var(--color-on-primary-container); }

.fwd-meta {
  font-size: 12px;
  color: var(--color-text-dim);
}

.fwd-check {
  display: inline-flex;
  align-items: center;
  color: var(--color-text-dim);
  flex-shrink: 0;
}

.fwd-item.selected .fwd-check { color: var(--color-primary); }

.fwd-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  color: var(--color-text-dim);
}

.fwd-empty .material-symbols-outlined { font-size: 40px; }

.fwd-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
}

.btn-text {
  background: none;
  border: none;
  color: var(--color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  padding: 10px 16px;
  border-radius: var(--radius-full);
  cursor: pointer;
}

.btn-text:hover { background: var(--color-surface-low); }

.btn-filled {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  background: var(--color-primary);
  color: var(--color-on-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  padding: 10px 18px 10px 14px;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: background 0.15s, opacity 0.15s;
}

.btn-filled:disabled { opacity: 0.5; cursor: not-allowed; }
.btn-filled:not(:disabled):hover { background: var(--color-primary-hover); }
.btn-filled .material-symbols-outlined { font-size: 18px; }
</style>
