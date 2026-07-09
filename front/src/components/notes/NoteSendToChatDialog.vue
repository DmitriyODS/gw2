<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="send"
    size="sm"
    :title="mode === 'note' ? 'Отправить заметку в чат' : 'Отправить в чат'"
    :subtitle="mode === 'note' ? 'Адресат получит доступ на просмотр — заметка появится у него во вкладке «Поделились»' : ''"
    :busy="sending"
    :actions="dialogActions"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="confirm"
  >
    <div v-if="mode === 'text'" class="nsc-preview">
      <span class="material-symbols-outlined">format_quote</span>
      <span class="nsc-preview-text">{{ textPreview }}</span>
    </div>
    <div v-else class="nsc-preview">
      <span class="material-symbols-outlined">sticky_note_2</span>
      <span class="nsc-preview-text">{{ note?.title || 'Без названия' }}</span>
    </div>

    <div class="nsc-search">
      <span class="material-symbols-outlined">search</span>
      <input v-model="q" placeholder="Кому отправить — имя или логин" class="nsc-input" autofocus />
    </div>

    <div v-if="loading" class="nsc-empty">
      <ProgressSpinner style="width:32px;height:32px" />
    </div>
    <div v-else-if="!results.length" class="nsc-empty">
      <span class="material-symbols-outlined">person_search</span>
      <p>{{ q ? 'Никого не нашли' : 'Начните вводить' }}</p>
    </div>
    <ul v-else class="nsc-list">
      <li
        v-for="u in results"
        :key="u.id"
        class="nsc-item"
        :class="{ selected: selectedIds.has(u.id) }"
        @click="toggle(u.id)"
      >
        <img class="nsc-avatar" :src="avatarOf(u)" :alt="u.fio" />
        <div class="nsc-info">
          <div class="nsc-name">{{ u.fio }}</div>
          <div class="nsc-meta">@{{ u.login }}</div>
        </div>
        <span class="nsc-check">
          <span class="material-symbols-outlined">
            {{ selectedIds.has(u.id) ? 'check_circle' : 'radio_button_unchecked' }}
          </span>
        </span>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
// Отправка из заметок в мессенджер (паттерн ForwardDialog): mode='text' —
// выделенный фрагмент уходит сообщением с Markdown-форматированием (пузыри
// рендерят MD); mode='note' — целая заметка: адресату выдаётся доступ на
// просмотр (адресный шаринг) и приходит сообщение со ссылкой на заметку.
import { computed, ref, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import { getDirectory } from '@/api/users.js'
import { openConversation, sendMessage } from '@/api/messenger.js'
import { upsertMember } from '@/api/notes.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  mode: { type: String, default: 'text' }, // 'text' | 'note'
  text: { type: String, default: '' },
  note: { type: Object, default: null },   // {id, title}
})
const emit = defineEmits(['update:modelValue', 'sent'])

const notif = useNotificationsStore()

const q = ref('')
const results = ref([])
const loading = ref(false)
const sending = ref(false)
const selectedIds = ref(new Set())
let debounceTimer = null

const textPreview = computed(() =>
  props.text.length > 90 ? props.text.slice(0, 90) + '…' : props.text)

const dialogActions = computed(() => [
  { kind: 'cancel', label: 'Отмена' },
  {
    kind: 'confirm',
    label: 'Отправить' + (selectedIds.value.size ? ` (${selectedIds.value.size})` : ''),
    icon: 'send',
    disabled: !selectedIds.value.size || sending.value,
  },
])

async function search() {
  loading.value = true
  try {
    results.value = await getDirectory(q.value.trim(), /* excludeSelf */ true, { global: true })
  } finally {
    loading.value = false
  }
}

watch(() => props.modelValue, (v) => {
  if (!v) return
  q.value = ''
  selectedIds.value = new Set()
  sending.value = false
  search()
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

function noteMessage() {
  const title = props.note?.title || 'Без названия'
  return `📝 Заметка «${title}»\n${window.location.origin}/notes/${props.note.id}`
}

async function confirm() {
  if (!selectedIds.value.size) return
  sending.value = true
  try {
    for (const userId of selectedIds.value) {
      const conv = await openConversation(userId)
      if (props.mode === 'note') {
        // Просмотр по умолчанию; право можно поднять в диалоге «Поделиться».
        await upsertMember(props.note.id, userId, false)
        await sendMessage(conv.id, { text: noteMessage(), attachment_ids: [] })
      } else {
        await sendMessage(conv.id, { text: props.text, attachment_ids: [] })
      }
    }
    notif.success(props.mode === 'note' ? 'Заметка отправлена в чат' : 'Отправлено в чат')
    emit('update:modelValue', false)
    emit('sent')
  } catch (e) {
    notif.error(e?.message || 'Не удалось отправить')
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.nsc-preview {
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
.nsc-preview .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.nsc-preview-text {
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.nsc-search {
  position: relative;
  display: flex;
  align-items: center;
  margin-bottom: 12px;
}
.nsc-search .material-symbols-outlined {
  position: absolute;
  left: 12px;
  color: var(--color-text-dim);
  font-size: 20px;
  pointer-events: none;
}

.nsc-input {
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
.nsc-input:focus { border-color: var(--color-primary); }

.nsc-list {
  list-style: none;
  padding: 0;
  margin: 0 0 8px;
  max-height: 42dvh;
  overflow-y: auto;
}

.nsc-item {
  display: flex;
  gap: 12px;
  align-items: center;
  padding: 10px 8px;
  cursor: pointer;
  border-radius: var(--radius-md);
}
.nsc-item:hover { background: var(--color-surface-low); }
.nsc-item.selected { background: var(--color-primary-container); }

.nsc-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.nsc-info { min-width: 0; flex: 1; }
.nsc-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.nsc-item.selected .nsc-name { color: var(--color-on-primary-container); }
.nsc-meta { font-size: 12px; color: var(--color-text-dim); }

.nsc-check {
  display: inline-flex;
  align-items: center;
  color: var(--color-text-dim);
  flex-shrink: 0;
}
.nsc-item.selected .nsc-check { color: var(--color-primary); }

.nsc-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 28px 16px;
  color: var(--color-text-dim);
}
.nsc-empty .material-symbols-outlined { font-size: 40px; }
</style>
