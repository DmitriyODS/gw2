<template>
  <!-- Колонка списка чатов — обычная панель ядра (AppPage embedded): заголовок,
       команды и строка управления её же, свой каркас здесь не нужен. -->
  <AppPage
    embedded
    :title="headerTitle"
    show-title
    :commands="commands"
    :menu="!narrow"
    menu-icon="left_panel_close"
    menu-label="Свернуть список"
    :loading="loading"
    :scroll="false"
    flush
    @command="onCommand"
    @menu="$emit('toggle-list')"
  >
    <!-- Поиск — в строку названия: в тесной панели он сворачивается в лупу, и
         список чатов начинается на строку выше. -->
    <template #search="{ narrow: tight }">
      <SearchField
        v-model="filter"
        :collapsed="tight"
        :placeholder="tab === 'support' ? 'Поиск по пользователям' : 'Поиск по чатам'"
      />
    </template>

    <!-- Строка управления нужна, только если есть что показать: пустая забирала
         бы у списка чатов высоту строки. -->
    <template v-if="showSupportTab || (showFolders && isMobile)" #subhead>
      <div class="conv-controls">
        <AppTabs
          v-if="showSupportTab"
          :model-value="tab"
          :tabs="tabItems"
          full-width
          @update:model-value="onTab"
        />
        <ChatFolders v-if="showFolders && isMobile" orientation="horizontal" />
      </div>
    </template>

    <div class="conv-body-row" :class="{ 'has-rail': showFolders && !isMobile }">
      <ChatFolders v-if="showFolders && !isMobile" orientation="vertical" class="conv-rail" />

      <EmptyState
        v-if="!visible.length"
        class="conv-empty--rich"
        :icon="emptyInFolder ? 'folder_open' : (filter ? 'person_search' : (tab === 'support' ? 'support_agent' : 'forum'))"
        :title="emptyInFolder ? 'В этой папке пусто' : (filter ? 'Никого не нашли' : (tab === 'support' ? 'Обращений пока нет' : 'Тут пока тишина'))"
        :subtitle="emptySub"
      >
        <AppButton
          v-if="!filter && tab !== 'support' && !emptyInFolder"
          variant="filled"
          icon="edit_square"
          label="Начать переписку"
          @click="$emit('new-chat')"
        />
      </EmptyState>

      <ul v-else class="conv-items">
        <li
          v-for="c in visible"
          :key="c.id"
          class="conv-item"
          :class="{ active: c.id === activeId, unread: c.unread_count > 0, pinned: c.is_pinned }"
          @click="onItemClick(c)"
          @contextmenu.prevent="openCtxMenu(c, $event.clientX, $event.clientY)"
          @touchstart.passive="onTouchStart(c, $event)"
          @touchend="onTouchEnd"
          @touchmove.passive="onTouchMove"
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
          <div v-else-if="c.is_group" class="conv-avatar-wrap group">
            <img v-if="c.avatar_path" class="conv-avatar" :src="`/uploads/${c.avatar_path}`" :alt="c.title" />
            <span v-else class="material-symbols-outlined">groups</span>
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
                <template v-else-if="c.is_group">
                  <span v-if="c.muted" class="material-symbols-outlined conv-mute-mark" title="Уведомления выключены">notifications_off</span>
                  {{ c.title }}
                </template>
                <template v-else>{{ c.other_user?.fio }}</template>
                <span
                  v-if="!c.is_dev_chat && c.other_user?.status_emoji"
                  class="conv-status-emoji"
                  :title="c.other_user?.status_text || 'Статус'"
                >{{ c.other_user.status_emoji }}</span>
              </span>
              <span v-if="c.last_message_at" class="conv-time">{{ formatTime(c.last_message_at) }}</span>
            </div>
            <div class="conv-bottom">
              <span v-if="!c.is_dev_chat && messenger.isTyping(c.id)" class="conv-preview conv-typing">печатает…</span>
              <span v-else class="conv-preview">
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
        </li>
      </ul>
    </div>

    <UserStatusDialog v-model="statusOpen" />
    <FolderManageDialog v-model="folderManageOpen" />

    <!-- Меню чата: ПКМ (десктоп) / долгое нажатие (мобила). Общий ContextMenu —
         он же умеет подменю, поэтому раскладка по папкам живёт в `children`. -->
    <ContextMenu
      :visible="ctxMenu.visible"
      :x="ctxMenu.x"
      :y="ctxMenu.y"
      :items="ctxItems"
      @select="onCtxSelect"
      @close="ctxMenu.visible = false"
    >
      <template #header>
        <div class="conv-ctx-title">{{ ctxTitle }}</div>
      </template>
    </ContextMenu>
  </AppPage>
</template>

<script setup>
import { ref, computed } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import ChatFolders from './ChatFolders.vue'
import FolderManageDialog from './FolderManageDialog.vue'
import UserStatusDialog from './UserStatusDialog.vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { stripMarkdown } from '@/utils/markdown.js'

const messenger = useMessengerStore()
const auth = useAuthStore()
const { isMobile } = useBreakpoint()

// Папки — только на вкладке «Чаты» (не в support-inbox) и когда они есть.
const showFolders = computed(() => props.tab === 'chats' && messenger.folders.length > 0)

// ── Меню чата (ПКМ на десктопе / long-press на мобиле) ──
const ctxMenu = ref({ visible: false, conv: null, x: 0, y: 0 })

const ctxTitle = computed(() => {
  const c = ctxMenu.value.conv
  if (!c) return ''
  if (c.is_dev_chat) return 'Техподдержка'
  if (c.is_group) return c.title || 'Группа'
  return c.other_user?.fio || 'Чат'
})

function openCtxMenu(conv, x, y) {
  // Для dev-чата доступна только раскладка по папкам — без них меню пустое.
  if (conv.is_dev_chat && !messenger.folders.length) return
  ctxMenu.value = { visible: true, conv, x, y }
}

function inFolder(f) {
  return !!ctxMenu.value.conv && f.conversation_ids.includes(ctxMenu.value.conv.id)
}

/* Пункты меню. Папки идут подменю (`children`) — общий ContextMenu умеет их
   сам; галочка показывает, лежит ли чат в папке. */
const ctxItems = computed(() => {
  const c = ctxMenu.value.conv
  if (!c) return []
  const items = []
  if (!c.is_dev_chat) {
    items.push({
      label: c.is_pinned ? 'Открепить чат' : 'Закрепить чат',
      icon: c.is_pinned ? 'keep_off' : 'keep',
      action: { kind: 'pin' },
    })
  }
  if (messenger.folders.length) {
    items.push({
      label: 'В папку',
      icon: 'folder',
      children: messenger.folders.map((f) => ({
        label: f.title,
        icon: inFolder(f) ? 'check_box' : 'check_box_outline_blank',
        action: { kind: 'folder', id: f.id },
      })),
    })
  }
  if (!c.is_dev_chat) {
    items.push({ divider: true })
    items.push({ label: 'Удалить чат', icon: 'delete', danger: true, action: { kind: 'delete' } })
  }
  return items
})

function onCtxSelect(action) {
  const conv = ctxMenu.value.conv
  if (!conv) return
  if (action.kind === 'pin') emit('toggle-pin', conv.id)
  else if (action.kind === 'delete') emit('delete', conv)
  else if (action.kind === 'folder') {
    const f = messenger.folders.find((x) => x.id === action.id)
    if (!f) return
    if (inFolder(f)) messenger.removeFromFolder(f.id, conv.id)
    else messenger.addToFolder(f.id, conv.id)
  }
}

// ── Long-press на тач-устройствах ──
let pressTimer = null
let pressStart = null
let suppressClick = false

function onTouchStart(conv, e) {
  const t = e.touches?.[0]
  if (!t) return
  pressStart = { x: t.clientX, y: t.clientY }
  suppressClick = false
  clearTimeout(pressTimer)
  pressTimer = setTimeout(() => {
    suppressClick = true
    openCtxMenu(conv, pressStart.x, pressStart.y)
    // Лёгкая тактильная отдача, если поддерживается.
    try { navigator.vibrate?.(10) } catch { /* no-op */ }
  }, 500)
}
function onTouchMove(e) {
  if (!pressStart) return
  const t = e.touches?.[0]
  if (!t) return
  if (Math.abs(t.clientX - pressStart.x) > 10 || Math.abs(t.clientY - pressStart.y) > 10) {
    clearTimeout(pressTimer)
  }
}
function onTouchEnd() {
  clearTimeout(pressTimer)
}

// Клик по карточке открывает чат; но если это был long-press — гасим переход.
function onItemClick(c) {
  if (suppressClick) {
    suppressClick = false
    return
  }
  emit('select', c.id)
}

const statusOpen = ref(false)
const folderManageOpen = ref(false)
const myStatusEmoji = computed(() => auth.user?.status_emoji || '')
const myStatusText = computed(() => auth.user?.status_text || '')

/* Команды колонки. `primary` — «Новый чат» (главное действие, в тесной панели
   уезжает на плавающую кнопку), остальное AppCommandBar сам сложит в «ещё». */
const commands = computed(() => {
  if (props.tab === 'support') return []
  return [
    { key: 'new-chat', label: 'Новый чат', icon: 'edit_square', variant: 'filled', primary: true },
    { key: 'new-group', label: 'Новая группа', icon: 'group_add' },
    { key: 'new-call', label: 'Новый звонок', icon: 'video_call' },
    { key: 'folders', label: 'Папки с чатами', icon: 'folder' },
    { key: 'status', label: myStatusText.value || 'Мой статус', icon: myStatusEmoji.value ? '' : 'add_reaction' },
  ]
})

function onCommand(key) {
  if (key === 'new-chat') emit('new-chat')
  else if (key === 'new-group') emit('new-group')
  else if (key === 'new-call') emit('new-call')
  else if (key === 'folders') folderManageOpen.value = true
  else if (key === 'status') statusOpen.value = true
}

const props = defineProps({
  conversations: { type: Array, required: true },
  activeId: { type: Number, default: null },
  loading: { type: Boolean, default: false },
  // Включает вкладку «Техподдержка» (для Администратора системы).
  showSupportTab: { type: Boolean, default: false },
  // Активная вкладка ('chats' | 'support'). Управляется родителем, потому что
  // от неё зависит, какой список передавать в `conversations`.
  tab: { type: String, default: 'chats' },
  supportUnread: { type: Number, default: 0 },
  // Узкая раскладка (её меряет AppListDetail): там колонка одна, сворачивать нечего.
  narrow: { type: Boolean, default: false },
})

const emit = defineEmits(['select', 'new-chat', 'new-group', 'new-call', 'toggle-pin', 'delete', 'change-tab', 'toggle-list'])

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

// Пустой список из-за активной папки (а не отсутствия чатов вообще).
const emptyInFolder = computed(() =>
  props.tab === 'chats' && messenger.activeFolderId != null && !filter.value
)

const emptySub = computed(() => {
  if (emptyInFolder.value) return 'Добавьте сюда чаты или настройте фильтры папки.'
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
    if (c.is_group) {
      return (c.title || '').toLowerCase().includes(q)
    }
    return (
      c.other_user?.fio?.toLowerCase().includes(q) ||
      c.other_user?.login?.toLowerCase().includes(q)
    )
  })
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
  // Разметка в однострочном превью вычищается (жирный/списки/ссылки → текст).
  if (msg.text) return stripMarkdown(msg.text)
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
/* Каркас, шапка, команды, поиск и меню — из ядра (AppPage/AppCommandBar/
   SearchField/ContextMenu). Здесь только своё: рейл папок, строка чата и её
   начинка. */
.conv-controls { display: flex; flex-direction: column; gap: 8px; width: 100%; min-width: 0; }

/* Рейл папок и список — в ряд; список скроллится сам (тело панели не скроллит). */
.conv-body-row { display: flex; min-height: 0; flex: 1; }
.conv-rail { min-height: 0; flex-shrink: 0; }


/* Ширину колонки задаёт каркас раздела (AppListDetail), панель её заполняет:
   иначе в узкой раскладке список оставался 340-пиксельным, а справа зияла
   пустота. */
.conv-company {
  font-weight: 600;
  color: var(--color-text);
}

.conv-dot {
  margin: 0 4px;
  color: var(--color-text-dim);
}

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

.conv-avatar-wrap.group {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  display: grid;
  place-items: center;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  flex-shrink: 0;
  overflow: hidden;
}
.conv-avatar-wrap.group .material-symbols-outlined {
  font-size: 24px;
  font-variation-settings: 'FILL' 1;
}
.conv-avatar-wrap.group .conv-avatar { width: 100%; height: 100%; }

.conv-mute-mark {
  font-size: 15px;
  vertical-align: -2px;
  color: var(--color-text-dim);
  margin-right: 2px;
}

.conv-empty--rich {
  flex: 1;
}

/* M3 filled tonal button: secondary container fill, pill-shape,
   state layer на hover/active, лёгкий лифт по shadow. */
.conv-items {
  list-style: none;
  padding: 0;
  margin: 0;
  overflow-y: auto;
  flex: 1;
}

/* Чаты — скруглённые стеклянные карточки; активный выделен тонировкой. */
.conv-item {
  display: flex;
  gap: 12px;
  padding: 10px 12px;
  margin: 0 8px 2px;
  border-radius: var(--radius-lg, 16px);
  cursor: pointer;
  align-items: center;
  transition: background 0.15s;
  position: relative;
}

.conv-item:hover { background: var(--glass-bg); }

.conv-item.active {
  background: var(--color-surface-low);
  background: var(--glass-bg), color-mix(in oklch, var(--color-primary-container) 40%, transparent);
  box-shadow: var(--glass-edge);
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

.conv-status-emoji {
  margin-left: 4px;
  font-size: 13px;
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

.conv-preview.conv-typing {
  color: var(--color-primary);
  font-style: italic;
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

/* ── Контекстное меню чата (ПКМ / long-press) ── */
@media (max-width: 768px) {
  .conv-items {
    padding: 0 4px;
    /* Запас снизу — под плавающую кнопку нового чата, а не под панель. */
    padding-bottom: 96px;
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
}
</style>
