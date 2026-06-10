<template>
  <aside class="conv-list" :class="{ 'is-mobile-hidden': hideOnMobile }">
    <div class="conv-list-header">
      <h2>{{ headerTitle }}</h2>
      <div class="header-actions">
        <button v-if="tab !== 'support'" class="new-btn" @click="$emit('new-chat')" title="Новый чат">
          <span class="material-symbols-outlined">edit_square</span>
        </button>
      </div>
    </div>

    <!-- Сегментированные табы «Чаты / Техподдержка». Для рут-админа во вкладке
         «Техподдержка» — inbox обращений; для всех остальных — личный dev-чат
         с командой разработки. -->
    <div v-if="showSupportTab" class="conv-tabs-wrap">
      <SegmentedTabs
        :model-value="tab"
        :tabs="tabItems"
        full-width
        @update:model-value="onTab"
      />
    </div>

    <div class="conv-search">
      <span class="material-symbols-outlined">search</span>
      <input
        v-model="filter"
        :placeholder="tab === 'support' ? 'Поиск по пользователям' : 'Поиск по чатам'"
        class="conv-search-input"
      />
    </div>

    <div v-if="loading" class="conv-empty">
      <ProgressSpinner />
    </div>
    <div v-else-if="!visible.length" class="conv-empty conv-empty--rich">
      <div class="empty-icon">
        <span class="material-symbols-outlined">
          {{ filter ? 'person_search' : (tab === 'support' ? 'support_agent' : 'forum') }}
        </span>
      </div>
      <h3 class="empty-title">
        {{ filter
          ? 'Никого не нашли'
          : (tab === 'support' ? 'Обращений пока нет' : 'Тут пока тишина') }}
      </h3>
      <p class="empty-sub">
        {{ emptySub }}
      </p>
      <button v-if="!filter && tab !== 'support'" class="btn-filled-tonal" @click="$emit('new-chat')">
        <span class="material-symbols-outlined">edit_square</span>
        Начать переписку
      </button>
    </div>
    <ul v-else class="conv-items">
      <li
        v-for="c in visible"
        :key="c.id"
        class="conv-item"
        :class="{ active: c.id === activeId, unread: c.unread_count > 0, pinned: c.is_pinned }"
        @click="$emit('select', c.id)"
      >
        <!-- Аватар. В support-inbox у админа аватар = фото владельца, не
             support_agent (иконка дублировала бы вкладку). У владельца —
             всегда иконка техподдержки. -->
        <div v-if="tab === 'support' && c.owner_user" class="conv-avatar-wrap">
          <img class="conv-avatar" :src="avatarOf(c.owner_user)" :alt="c.owner_user?.fio" />
          <span v-if="messenger.isOnline(c.owner_user?.id)" class="online-dot" title="В сети"></span>
        </div>
        <div v-else-if="c.is_dev_chat" class="conv-avatar-wrap dev">
          <span class="material-symbols-outlined">support_agent</span>
        </div>
        <div v-else-if="c.is_pet_chat" class="conv-avatar-wrap pet">
          <span class="conv-pet-emoji">👾</span>
        </div>
        <div v-else class="conv-avatar-wrap">
          <img class="conv-avatar" :src="avatarOf(c.other_user)" :alt="c.other_user?.fio" />
          <span v-if="messenger.isOnline(c.other_user?.id)" class="online-dot" title="В сети"></span>
        </div>
        <div class="conv-body">
          <div class="conv-top">
            <span class="conv-name">
              <template v-if="tab === 'support' && c.owner_user">
                {{ c.owner_user.fio }}
              </template>
              <template v-else-if="c.is_dev_chat">Техподдержка</template>
              <template v-else-if="c.is_pet_chat">{{ petChatName || c.pet_name || 'Грувик' }}</template>
              <template v-else>{{ c.other_user?.fio }}</template>
            </span>
            <span v-if="c.last_message_at" class="conv-time">{{ formatTime(c.last_message_at) }}</span>
          </div>
          <div class="conv-bottom">
            <span class="conv-preview">
              <template v-if="tab === 'support' && c.company_name">
                <span class="conv-company">{{ c.company_name }}</span>
                <span class="conv-dot">·</span>
              </template>
              {{ preview(c.last_message) }}
            </span>
            <span v-if="c.unread_count" class="conv-badge">{{ c.unread_count }}</span>
            <span v-else-if="c.is_pinned" class="conv-pin-mark" title="Закреплён">
              <span class="material-symbols-outlined">keep</span>
            </span>
          </div>
        </div>
        <!-- Действия pin/delete — только на обычных диалогах. Чат техподдержки
             (и у владельца, и у админа в инбоксе) нельзя ни закрепить, ни
             удалить. -->
        <div v-if="!c.is_dev_chat && !c.is_pet_chat" class="conv-actions" @click.stop>
          <button
            class="conv-action"
            :class="{ active: c.is_pinned }"
            :title="c.is_pinned ? 'Открепить' : 'Закрепить'"
            @click="$emit('toggle-pin', c.id); $event.currentTarget.blur()"
          >
            <span class="material-symbols-outlined">{{ c.is_pinned ? 'keep_off' : 'keep' }}</span>
          </button>
          <button
            class="conv-action danger"
            title="Удалить чат"
            @click="$emit('delete', c); $event.currentTarget.blur()"
          >
            <span class="material-symbols-outlined">delete</span>
          </button>
        </div>
      </li>
    </ul>
  </aside>
</template>

<script setup>
import { ref, computed } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useMessengerStore } from '@/stores/messenger.js'

const messenger = useMessengerStore()

const props = defineProps({
  conversations: { type: Array, required: true },
  activeId: { type: Number, default: null },
  loading: { type: Boolean, default: false },
  hideOnMobile: { type: Boolean, default: false },
  // Включает вкладку «Техподдержка» (для Администратора системы).
  showSupportTab: { type: Boolean, default: false },
  // Активная вкладка ('chats' | 'support'). Управляется родителем, потому что
  // от неё зависит, какой список передавать в `conversations`.
  tab: { type: String, default: 'chats' },
  supportUnread: { type: Number, default: 0 },
})

const emit = defineEmits(['select', 'new-chat', 'toggle-pin', 'delete', 'change-tab'])

const filter = ref('')

function onTab(t) {
  if (t !== props.tab) {
    filter.value = ''
    emit('change-tab', t)
  }
}

const tabItems = computed(() => ([
  { value: 'chats', label: 'Чаты', icon: 'chat' },
  {
    value: 'support',
    label: 'Техподдержка',
    icon: 'support_agent',
    badge: props.supportUnread || null,
  },
]))

const headerTitle = computed(() =>
  props.tab === 'support' ? 'Техподдержка' : 'Чаты'
)

const emptySub = computed(() => {
  if (filter.value) return 'Попробуйте другое имя или логин.'
  if (props.tab === 'support') {
    return 'Здесь появятся обращения пользователей в техподдержку. Все ответы отправятся от имени «Техподдержки» — ФИО админа скрыто.'
  }
  return 'Напишите коллеге — обсудите задачу, поделитесь файлом или просто поздоровайтесь.'
})

const visible = computed(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return props.conversations
  return props.conversations.filter(c => {
    if (props.tab === 'support') {
      const owner = c.owner_user
      return (
        owner?.fio?.toLowerCase().includes(q) ||
        owner?.login?.toLowerCase().includes(q) ||
        (c.company_name || '').toLowerCase().includes(q)
      )
    }
    if (c.is_dev_chat) {
      return 'техподдержка'.includes(q)
    }
    if (c.is_pet_chat) {
      const name = (petChatName.value || c.pet_name || 'Грувик').toLowerCase()
      return 'грувик'.includes(q) || name.includes(q)
    }
    return (
      c.other_user?.fio?.toLowerCase().includes(q) ||
      c.other_user?.login?.toLowerCase().includes(q)
    )
  })
})

// Живое имя своего Грувика (обновляется сокетом pet:update); если groove-store
// ещё не загружен — null, тогда берём снапшот pet_name из ответа API.
const petChatName = computed(() => {
  try { return useGrooveStore().pet?.name || null } catch { return null }
})

function avatarOf(u) {
  if (!u) return ''
  return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
}

function preview(msg) {
  if (!msg) return 'Нет сообщений'
  // Системная плашка звонка: показываем тип звонка вместо пустой строки.
  if (msg.kind === 'call') {
    return msg.call?.media === 'audio' ? '📞 Аудиозвонок' : '📹 Видеозвонок'
  }
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

.header-actions { display: flex; align-items: center; gap: 4px; }

/* На мобиле создание чата делает FAB — дублирующая кнопка в шапке не нужна. */
@media (max-width: 768px) {
  .new-btn { display: none; }
}

/* Обёртка для SegmentedTabs «Чаты / Техподдержка». */
.conv-tabs-wrap {
  padding: 0 16px 8px;
}

.conv-company {
  font-weight: 600;
  color: var(--color-text);
}

.conv-dot {
  margin: 0 4px;
  color: var(--color-text-dim);
}

.conv-avatar-wrap.pet {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--color-tertiary-container);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.conv-pet-emoji { font-size: 24px; }

.conv-avatar-wrap.dev {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  flex-shrink: 0;
}
.conv-avatar-wrap.dev .material-symbols-outlined {
  font-size: 22px;
  font-variation-settings: 'FILL' 1;
}

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

/* M3 Expressive empty state: иконка в круглом tinted container,
   крупный заголовок, мягкая подпись и filled-tonal pill-кнопка с иконкой.
   Все цвета — через семантические токены, чтобы тёмная тема и кастомные
   палитры подхватывались автоматически. */
.conv-empty--rich {
  gap: 14px;
  padding: 48px 24px;
  flex: 1;
  justify-content: center;
}

.empty-icon {
  width: 80px;
  height: 80px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: flex;
  align-items: center;
  justify-content: center;
  margin-bottom: 4px;
  box-shadow: var(--shadow-sm);
  transition: transform 0.25s ease, box-shadow 0.25s ease;
}

.empty-icon:hover {
  transform: translateY(-1px);
  box-shadow: var(--shadow-md);
}

.empty-icon .material-symbols-outlined {
  font-size: 40px;
  font-variation-settings: 'FILL' 1, 'wght' 400, 'GRAD' 0, 'opsz' 40;
}

.empty-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.1px;
  color: var(--color-text);
}

.empty-sub {
  margin: 0 0 4px;
  font-size: 13.5px;
  line-height: 1.45;
  color: var(--color-text-dim);
  max-width: 260px;
}

/* M3 filled tonal button: secondary container fill, pill-shape,
   state layer на hover/active, лёгкий лифт по shadow. */
.btn-filled-tonal {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 22px 0 18px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font: inherit;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: 0.1px;
  cursor: pointer;
  position: relative;
  overflow: hidden;
  isolation: isolate;
  transition: box-shadow 0.2s ease, transform 0.15s ease;
}

.btn-filled-tonal::before {
  content: '';
  position: absolute;
  inset: 0;
  background: currentColor;
  opacity: 0;
  transition: opacity 0.18s ease;
  z-index: -1;
}

.btn-filled-tonal:hover {
  box-shadow: var(--shadow-sm);
}

.btn-filled-tonal:hover::before { opacity: 0.08; }
.btn-filled-tonal:focus-visible::before { opacity: 0.12; }
.btn-filled-tonal:active::before { opacity: 0.16; }
.btn-filled-tonal:active { transform: scale(0.98); }

.btn-filled-tonal:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.btn-filled-tonal .material-symbols-outlined {
  font-size: 20px;
  font-variation-settings: 'FILL' 0, 'wght' 500, 'GRAD' 0, 'opsz' 24;
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
  position: relative;
}

.conv-item:hover { background: var(--color-surface-low); }

.conv-item.active {
  background: var(--color-surface-low);
  border-left-color: var(--color-primary);
}

/* Закреплённый — мягкий tertiary-accent на левой границе */
.conv-item.pinned:not(.active) {
  border-left-color: var(--color-tertiary);
}

.conv-avatar-wrap {
  position: relative;
  flex-shrink: 0;
}

.conv-avatar {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  object-fit: cover;
  display: block;
}

.online-dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-success);
  border: 2px solid var(--color-surface);
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

.conv-pin-mark {
  display: inline-flex;
  align-items: center;
  color: var(--color-tertiary);
}

.conv-pin-mark .material-symbols-outlined {
  font-size: 16px;
  font-variation-settings: 'FILL' 1, 'wght' 500, 'GRAD' 0, 'opsz' 20;
}

/* Действия на карточке диалога — показываются на hover, на тач-устройствах
   видны всегда (по media (hover: none)). */
/* Абсолютно позиционируем, чтобы вне ховера действия не «съедали» ширину
   карточки (раньше прозрачные кнопки оставляли пустоту справа). На ховере
   выезжают плавающим чипом поверх правого края. */
.conv-actions {
  position: absolute;
  right: 10px;
  top: 50%;
  transform: translateY(-50%);
  display: flex;
  gap: 2px;
  padding: 2px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  box-shadow: var(--shadow-sm);
  opacity: 0;
  pointer-events: none;
  transition: opacity 0.15s;
}

.conv-item:hover .conv-actions,
.conv-item:focus-within .conv-actions {
  opacity: 1;
  pointer-events: auto;
}

.conv-action {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s, color 0.15s;
}

.conv-action:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.conv-action.active {
  color: var(--color-tertiary);
}

.conv-action.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.conv-action .material-symbols-outlined {
  font-size: 18px;
}

@media (hover: none) {
  /* На тач-устройствах кнопки видны всегда и встроены в поток (без наложения
     на время/бейдж). */
  .conv-actions {
    position: static;
    transform: none;
    background: transparent;
    box-shadow: none;
    opacity: 1;
    pointer-events: auto;
    flex-shrink: 0;
  }
}

@media (max-width: 768px) {
  .conv-list {
    width: 100%;
    height: 100%;
    border-right: none;
  }
  .conv-list.is-mobile-hidden { display: none; }

  .conv-list-header {
    padding: 14px 12px 6px;
    padding-top: calc(14px + env(safe-area-inset-top, 0px));
  }
  .conv-list-header h2 { font-size: 20px; font-weight: 800; }

  .conv-tabs-wrap { padding: 0 12px 8px; }

  .conv-search { padding: 6px 12px 10px; }
  .conv-search .material-symbols-outlined { left: 24px; }
  .conv-search-input {
    height: 44px;
    border-radius: var(--radius-full);
    padding-left: 40px;
    background: var(--color-surface-high);
    border-color: transparent;
    font-size: 14.5px;
  }

  .conv-items {
    padding: 0 4px;
    padding-bottom: calc(60px + 96px + env(safe-area-inset-bottom, 0px));
  }
  .conv-item {
    border-radius: var(--radius-lg);
    padding: 12px;
    margin: 2px 4px;
  }
  .conv-avatar-wrap, .conv-avatar { width: 48px !important; height: 48px !important; }
  .conv-name { font-size: 15px; font-weight: 700; }
  .conv-time { font-size: 11.5px; }
  .conv-preview { font-size: 13px; }

  /* На тач-экранах кнопки pin/delete должны быть всегда видны:
     рендерим их фиксированной полосой действий при свайпе нет, но через
     toolbar при тапе — упрощение: всегда видны как иконки справа. */
  .conv-actions {
    opacity: 1 !important;
    pointer-events: auto !important;
  }
}
</style>
