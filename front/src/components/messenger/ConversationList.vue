<template>
  <aside class="conv-list" :class="{ 'is-mobile-hidden': hideOnMobile }">
    <div class="conv-list-header">
      <h2>Чаты</h2>
      <button class="new-btn" @click="$emit('new-chat')" title="Новый чат">
        <span class="material-symbols-outlined">edit_square</span>
      </button>
    </div>

    <div class="conv-search">
      <span class="material-symbols-outlined">search</span>
      <input
        v-model="filter"
        placeholder="Поиск по чатам"
        class="conv-search-input"
      />
    </div>

    <div v-if="loading" class="conv-empty">
      <ProgressSpinner />
    </div>
    <div v-else-if="!visible.length" class="conv-empty">
      <span class="material-symbols-outlined">forum</span>
      <p>{{ filter ? 'Никого не нашли' : 'Здесь будут ваши чаты' }}</p>
      <button class="btn-text" @click="$emit('new-chat')">Начать переписку</button>
    </div>
    <ul v-else class="conv-items">
      <li
        v-for="c in visible"
        :key="c.id"
        class="conv-item"
        :class="{ active: c.id === activeId, unread: c.unread_count > 0 }"
        @click="$emit('select', c.id)"
      >
        <img class="conv-avatar" :src="avatarOf(c.other_user)" :alt="c.other_user?.fio" />
        <div class="conv-body">
          <div class="conv-top">
            <span class="conv-name">{{ c.other_user?.fio }}</span>
            <span v-if="c.last_message_at" class="conv-time">{{ formatTime(c.last_message_at) }}</span>
          </div>
          <div class="conv-bottom">
            <span class="conv-preview">{{ preview(c.last_message) }}</span>
            <span v-if="c.unread_count" class="conv-badge">{{ c.unread_count }}</span>
          </div>
        </div>
      </li>
    </ul>
  </aside>
</template>

<script setup>
import { ref, computed } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'

const props = defineProps({
  conversations: { type: Array, required: true },
  activeId: { type: Number, default: null },
  loading: { type: Boolean, default: false },
  hideOnMobile: { type: Boolean, default: false },
})

defineEmits(['select', 'new-chat'])

const filter = ref('')

const visible = computed(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return props.conversations
  return props.conversations.filter(c =>
    c.other_user?.fio?.toLowerCase().includes(q) ||
    c.other_user?.login?.toLowerCase().includes(q)
  )
})

function avatarOf(u) {
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function preview(msg) {
  if (!msg) return 'Нет сообщений'
  if (msg.text) return msg.text
  if (msg.attachments?.length) {
    const a = msg.attachments[0]
    if (a.mime_type?.startsWith('image/')) return '📷 Фото'
    if (a.mime_type?.startsWith('video/')) return '🎬 Видео'
    if (a.mime_type?.startsWith('audio/')) return '🎵 Аудио'
    return '📎 Файл'
  }
  return ''
}

function formatTime(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  const now = new Date()
  const sameDay = d.toDateString() === now.toDateString()
  if (sameDay) {
    return d.toLocaleTimeString('ru', { hour: '2-digit', minute: '2-digit' })
  }
  const diff = (now - d) / 86400000
  if (diff < 7) {
    return d.toLocaleDateString('ru', { weekday: 'short' })
  }
  return d.toLocaleDateString('ru', { day: '2-digit', month: '2-digit' })
}
</script>

<style scoped>
.conv-list {
  width: 320px;
  flex-shrink: 0;
  background: var(--color-surface);
  border-right: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.conv-list-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 16px 8px;
}

.conv-list-header h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text);
}

.new-btn {
  background: transparent;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  padding: 6px;
  border-radius: var(--radius-sm);
}

.new-btn:hover { background: var(--color-surface-low); }

.conv-search {
  padding: 8px 16px 12px;
  position: relative;
}

.conv-search .material-symbols-outlined {
  position: absolute;
  left: 28px;
  top: 50%;
  transform: translateY(-50%);
  color: var(--color-text-dim);
  font-size: 18px;
}

.conv-search-input {
  width: 100%;
  padding: 8px 12px 8px 38px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface-low);
  color: var(--color-text);
  font-size: 14px;
  outline: none;
}

.conv-search-input:focus {
  border-color: var(--color-primary);
}

.conv-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 40px 16px;
  color: var(--color-text-dim);
  text-align: center;
}

.conv-empty .material-symbols-outlined {
  font-size: 48px;
}

.btn-text {
  background: none;
  border: none;
  color: var(--color-primary);
  font-weight: 600;
  cursor: pointer;
}

.conv-items {
  list-style: none;
  padding: 0;
  margin: 0;
  overflow-y: auto;
  flex: 1;
}

.conv-item {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  cursor: pointer;
  align-items: center;
  border-left: 3px solid transparent;
  transition: background 0.12s;
}

.conv-item:hover { background: var(--color-surface-low); }

.conv-item.active {
  background: var(--color-surface-low);
  border-left-color: var(--color-primary);
}

.conv-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.conv-body { flex: 1; min-width: 0; }

.conv-top {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  gap: 8px;
}

.conv-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conv-time {
  font-size: 11px;
  color: var(--color-text-dim);
  white-space: nowrap;
}

.conv-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 8px;
  margin-top: 2px;
}

.conv-preview {
  font-size: 13px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.conv-item.unread .conv-preview {
  color: var(--color-text);
  font-weight: 500;
}

.conv-badge {
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 11px;
  font-weight: 700;
  padding: 2px 7px;
  border-radius: var(--radius-full);
  min-width: 20px;
  text-align: center;
}

@media (max-width: 768px) {
  .conv-list {
    width: 100%;
    border-right: none;
  }
  .conv-list.is-mobile-hidden { display: none; }
}
</style>
