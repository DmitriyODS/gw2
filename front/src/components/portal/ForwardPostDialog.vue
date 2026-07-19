<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="forward"
    size="sm"
    title="Переслать пост"
    :busy="sending"
    :actions="dialogActions"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="confirm"
  >
    <div v-if="post" class="fwdp-preview">
      <span class="material-symbols-outlined">campaign</span>
      <span class="fwdp-preview-text">{{ previewText }}</span>
    </div>

    <div class="fwdp-search">
      <span class="material-symbols-outlined">search</span>
      <input
        v-model="q"
        placeholder="Кому переслать — имя или логин"
        class="fwdp-input"
        autofocus
      />
    </div>

    <div v-if="loading && !items.length" class="fwdp-empty">
      <BrandLoader :size="48" />
    </div>
    <div v-else-if="!items.length" class="fwdp-empty">
      <span class="material-symbols-outlined">person_search</span>
      <p>{{ q ? 'Никого не нашли — проверьте логин' : 'Пока нет диалогов. Введите логин, чтобы найти человека.' }}</p>
    </div>
    <ul v-else class="fwdp-list">
      <li
        v-for="u in items"
        :key="u.id"
        class="fwdp-item"
        :class="{ selected: selectedIds.has(u.id) }"
        @click="toggle(u.id)"
      >
        <img class="fwdp-avatar" :src="avatarOf(u)" :alt="u.fio" />
        <div class="fwdp-info">
          <div class="fwdp-name">{{ u.fio }}</div>
          <div class="fwdp-meta">@{{ u.login }} · {{ u.post || u.role?.name }}</div>
        </div>
        <span class="fwdp-check">
          <span class="material-symbols-outlined">
            {{ selectedIds.has(u.id) ? 'check_circle' : 'radio_button_unchecked' }}
          </span>
        </span>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { useContactPicker } from '@/composables/useContactPicker.js'

// Тот же UX выбора адресатов, что и в мессенджере (components/messenger/
// ForwardDialog.vue), но со своим превью (пост, не сообщение) — оригинал
// завязан на форму message-сообщения (kind/call/attachments), поэтому здесь
// отдельная копия под нужды портала.
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  post: { type: Object, default: null },
})

const emit = defineEmits(['update:modelValue', 'confirm'])

const { q, results, loading, reset } = useContactPicker()
const sending = ref(false)
const selectedIds = ref(new Set())

const items = computed(() => results.value)

const previewText = computed(() => {
  const p = props.post
  if (!p) return ''
  const t = p.title || p.body || ''
  return t.length > 80 ? t.slice(0, 80) + '…' : t
})

const dialogActions = computed(() => [
  { kind: 'cancel', label: 'Отмена' },
  {
    kind: 'confirm',
    label: 'Переслать' + (selectedIds.value.size ? ` (${selectedIds.value.size})` : ''),
    icon: 'send',
    disabled: !selectedIds.value.size || sending.value,
  },
])

watch(() => props.modelValue, (v) => {
  if (v) {
    selectedIds.value = new Set()
    sending.value = false
    reset()
  }
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

async function confirm() {
  if (!selectedIds.value.size) return
  sending.value = true
  emit('confirm', { userIds: [...selectedIds.value] })
}

defineExpose({ stopSending: () => { sending.value = false } })
</script>

<style scoped>
.fwdp-preview {
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

.fwdp-preview .material-symbols-outlined {
  font-size: 18px;
  color: var(--color-primary);
}

.fwdp-preview-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.fwdp-search {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}

.fwdp-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.fwdp-input {
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

.fwdp-input:focus { border-color: var(--color-primary); }

.fwdp-list {
  list-style: none;
  padding: 0;
  margin: 0 0 14px;
  max-height: 42dvh;
  overflow-y: auto;
}

.fwdp-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 10px 8px;
  cursor: pointer;
  border-radius: var(--radius-md);
}

.fwdp-item:hover { background: var(--color-surface-low); }

.fwdp-item.selected { background: var(--color-primary-container); }

.fwdp-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.fwdp-info { min-width: 0; flex: 1; }

.fwdp-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.fwdp-item.selected .fwdp-name { color: var(--color-on-primary-container); }

.fwdp-meta {
  font-size: 12px;
  color: var(--color-text-dim);
}

.fwdp-check {
  display: inline-flex;
  align-items: center;
  color: var(--color-text-dim);
  flex-shrink: 0;
}

.fwdp-item.selected .fwdp-check { color: var(--color-primary); }

.fwdp-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  color: var(--color-text-dim);
}

.fwdp-empty .material-symbols-outlined { font-size: 40px; }
</style>
