<template>
  <aside class="conv-list" :class="{ 'is-mobile-hidden': hideOnMobile, 'has-rail': showFolders && !isMobile }">
    <ChatFolders v-if="showFolders && !isMobile" orientation="vertical" class="conv-rail" />
    <div class="conv-main">
    <div class="conv-list-header">
      <h2>{{ headerTitle }}</h2>
      <div class="header-actions">
        <button
          v-if="tab !== 'support'"
          class="new-btn new-btn--status"
          :title="myStatusText || 'Мой статус'"
          @click="statusOpen = true"
        >
          <span v-if="myStatusEmoji" class="status-btn-emoji">{{ myStatusEmoji }}</span>
          <span v-else class="material-symbols-outlined">add_reaction</span>
        </button>
        <button
          v-if="tab !== 'support'"
          class="new-btn new-btn--call"
          title="Новый звонок — комната, куда можно позвать коллег"
          @click="$emit('new-call')"
        >
          <span class="material-symbols-outlined">video_call</span>
        </button>
        <button v-if="tab !== 'support'" class="new-btn new-btn--folders" @click="folderManageOpen = true" title="Папки с чатами">
          <span class="material-symbols-outlined">folder</span>
        </button>
        <button v-if="tab !== 'support'" class="new-btn new-btn--group" @click="$emit('new-group')" title="Новая группа">
          <span class="material-symbols-outlined">group_add</span>
        </button>
        <button v-if="tab !== 'support'" class="new-btn" @click="$emit('new-chat')" title="Новый чат">
          <span class="material-symbols-outlined">edit_square</span>
        </button>
      </div>
    </div>

    <UserStatusDialog v-model="statusOpen" />
    <FolderManageDialog v-model="folderManageOpen" />

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

    <ChatFolders v-if="showFolders && isMobile" orientation="horizontal" />

    <div class="conv-search">
      <span class="material-symbols-outlined">search</span>
      <input
        v-model="filter"
        :placeholder="tab === 'support' ? 'Поиск по пользователям' : 'Поиск по чатам'"
        class="conv-search-input"
      />
    </div>

    <div v-if="loading" class="conv-empty">
      <BrandLoader :size="64" />
    </div>
    <EmptyState
      v-else-if="!visible.length"
      class="conv-empty--rich"
      :icon="emptyInFolder ? 'folder_open' : (filter ? 'person_search' : (tab === 'support' ? 'support_agent' : 'forum'))"
      :title="emptyInFolder ? 'В этой папке пусто' : (filter ? 'Никого не нашли' : (tab === 'support' ? 'Обращений пока нет' : 'Тут пока тишина'))"
      :subtitle="emptySub"
    >
      <button v-if="!filter && tab !== 'support' && !emptyInFolder" class="btn-filled-tonal" @click="$emit('new-chat')">
        <span class="material-symbols-outlined">edit_square</span>
        Начать переписку
      </button>
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

    <!-- Контекстное меню чата: ПКМ (десктоп) / долгое нажатие (мобила).
         Действия pin/delete + раскладка по папкам. Размещается так, чтобы не
         обрезаться краем экрана, и ужимается со скроллом, если выше вьюпорта. -->
    <Teleport to="body">
      <div v-if="ctxMenu.visible" class="cm-mask" @click="closeCtxMenu" @contextmenu.prevent="closeCtxMenu">
        <div
          ref="ctxMenuEl"
          class="cm-menu"
          :style="ctxMenu.style"
          @click.stop
        >
          <!-- Основной вид -->
          <template v-if="ctxMenu.view === 'main'">
            <div class="cm-title">{{ ctxTitle }}</div>

            <button v-if="!ctxMenu.conv?.is_dev_chat" class="cm-item" @click="ctxTogglePin">
              <span class="material-symbols-outlined cm-ico" :class="{ 'tone-tertiary': ctxMenu.conv?.is_pinned }">
                {{ ctxMenu.conv?.is_pinned ? 'keep_off' : 'keep' }}
              </span>
              <span>{{ ctxMenu.conv?.is_pinned ? 'Открепить чат' : 'Закрепить чат' }}</span>
            </button>

            <button v-if="messenger.folders.length" class="cm-item" @click="openFoldersView">
              <span class="material-symbols-outlined cm-ico tone-tertiary">folder</span>
              <span>В папку</span>
              <span class="material-symbols-outlined cm-arrow">chevron_right</span>
            </button>

            <template v-if="!ctxMenu.conv?.is_dev_chat">
              <div class="cm-divider" />
              <button class="cm-item danger" @click="ctxDelete">
                <span class="material-symbols-outlined cm-ico tone-error">delete</span>
                <span>Удалить чат</span>
              </button>
            </template>
          </template>

          <!-- Вложенный вид: список папок -->
          <template v-else>
            <button class="cm-item cm-back" @click="backToMain">
              <span class="material-symbols-outlined cm-ico">arrow_back</span>
              <span>В папку</span>
            </button>
            <div class="cm-divider" />
            <button
              v-for="f in messenger.folders"
              :key="f.id"
              class="cm-item"
              @click="toggleFolder(f)"
            >
              <span class="cm-emoji">
                <EmojiGlyph v-if="f.emoji" :char="f.emoji" class="cm-emoji-glyph" />
                <span v-else class="material-symbols-outlined">folder</span>
              </span>
              <span class="cm-name">{{ f.title }}</span>
              <span class="cm-check material-symbols-outlined">
                {{ inFolder(f) ? 'check_box' : 'check_box_outline_blank' }}
              </span>
            </button>
          </template>
        </div>
      </div>
    </Teleport>
  </aside>
</template>

<script setup>
import { ref, computed, nextTick } from 'vue'
import { fitToViewport } from '@/utils/menuPlacement.js'
import BrandLoader from '@/components/common/BrandLoader.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
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

// ── Контекстное меню чата (ПКМ на десктопе / long-press на мобиле) ──
const ctxMenu = ref({ visible: false, conv: null, view: 'main', anchor: { x: 0, y: 0 }, style: {} })
const ctxMenuEl = ref(null)

const ctxTitle = computed(() => {
  const c = ctxMenu.value.conv
  if (!c) return ''
  if (c.is_dev_chat) return 'Техподдержка'
  if (c.is_group) return c.title || 'Группа'
  return c.other_user?.fio || 'Чат'
})

async function openCtxMenu(conv, x, y) {
  // Для dev-чата доступна только раскладка по папкам — без них меню пустое.
  if (conv.is_dev_chat && !messenger.folders.length) return
  ctxMenu.value = {
    visible: true,
    conv,
    view: 'main',
    anchor: { x, y },
    // Стартовая позиция у точки вызова; уточним после измерения.
    style: { position: 'fixed', left: x + 'px', top: y + 'px', visibility: 'hidden' },
  }
  await placeCtxMenu()
}

// Пересчёт позиции меню после рендера/смены вида (высота меняется).
async function placeCtxMenu() {
  await nextTick()
  const el = ctxMenuEl.value
  if (!el) return
  const { x, y } = ctxMenu.value.anchor
  const p = fitToViewport(el, { x, y, pad: 8 })
  ctxMenu.value.style = {
    position: 'fixed',
    left: p.left + 'px',
    top: p.top + 'px',
    width: p.width + 'px',
    maxHeight: p.maxHeight + 'px',
  }
}

function openFoldersView() {
  ctxMenu.value.view = 'folders'
  placeCtxMenu()
}
function backToMain() {
  ctxMenu.value.view = 'main'
  placeCtxMenu()
}

function closeCtxMenu() {
  ctxMenu.value.visible = false
}

function ctxTogglePin() {
  emit('toggle-pin', ctxMenu.value.conv.id)
  closeCtxMenu()
}

function ctxDelete() {
  emit('delete', ctxMenu.value.conv)
  closeCtxMenu()
}

function inFolder(f) {
  return !!ctxMenu.value.conv && f.conversation_ids.includes(ctxMenu.value.conv.id)
}
function toggleFolder(f) {
  const conv = ctxMenu.value.conv
  if (!conv) return
  if (inFolder(f)) messenger.removeFromFolder(f.id, conv.id)
  else messenger.addToFolder(f.id, conv.id)
  // Меню не закрываем — можно разложить чат сразу по нескольким папкам.
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

const emit = defineEmits(['select', 'new-chat', 'new-call', 'toggle-pin', 'delete', 'change-tab'])

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
.conv-list {
  width: 340px;
  flex-shrink: 0;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  overflow: hidden;
  display: flex;
  flex-direction: row;
  min-height: 0;
}

/* С рейлом папок панель шире ровно на его ширину (76px) — колонка списка
   остаётся комфортной, а без папок ширина прежняя. */
.conv-list.has-rail { width: 416px; }

/* Рейл папок — фиксированная колонка слева, ширина в самом компоненте. */
.conv-rail { min-height: 0; }

/* Основная колонка списка (шапка/табы/поиск/лента). */
.conv-main {
  flex: 1;
  min-width: 0;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

@media (max-width: 768px) {
  /* На мобильном список — полноэкранный слой (как раньше), без рамки-стекла. */
  .conv-list { border: none; border-radius: 0; }
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

/* На мобиле создание 1:1-чата делает FAB — дублирующая кнопка в шапке не нужна.
   Кнопки «Новая группа», «Новый звонок» и «Мой статус» остаются: у них FAB-дубля нет. */
@media (max-width: 768px) {
  .new-btn { display: none; }
  .new-btn--call, .new-btn--status, .new-btn--group, .new-btn--folders { display: block; }
}

.status-btn-emoji {
  font-size: 20px;
  line-height: 1;
  display: block;
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
  padding: 9px 14px 9px 38px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--color-surface-low);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
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

/* Пустой список — общий EmptyState, здесь только растягивание на всю высоту. */
.conv-empty--rich {
  flex: 1;
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
.cm-mask {
  position: fixed;
  inset: 0;
  z-index: 10060;
}
.cm-menu {
  position: fixed;
  min-width: 224px;
  overflow-y: auto;
  padding: 6px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
}
.cm-title {
  padding: 6px 10px 8px;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.cm-section {
  padding: 6px 10px 4px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: 0.3px;
  text-transform: uppercase;
  color: var(--color-text-dim);
}
.cm-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}
.cm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px;
  border: none;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  border-radius: var(--radius-sm);
  text-align: left;
  font-size: 14px;
  font-weight: 500;
}
.cm-item:hover { background: var(--color-surface-low); }
.cm-item.danger { color: var(--color-error); }
.cm-item.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.cm-ico { font-size: 20px; color: var(--color-text-dim); flex-shrink: 0; }
.cm-ico.tone-tertiary { color: var(--color-tertiary); }
.cm-ico.tone-error { color: var(--color-error); }
.cm-emoji {
  width: 26px; height: 26px; flex-shrink: 0;
  display: grid; place-items: center;
  border-radius: var(--radius-sm);
  background: var(--color-tertiary-container); color: var(--color-on-tertiary-container);
}
.cm-emoji-glyph { font-size: 15px; line-height: 1; }
.cm-emoji .material-symbols-outlined { font-size: 16px; font-variation-settings: 'FILL' 1; }
.cm-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cm-check { font-size: 20px; color: var(--color-text-dim); flex-shrink: 0; }
.cm-arrow { margin-left: auto; font-size: 20px; color: var(--color-text-dim); flex-shrink: 0; }
.cm-back { font-weight: 700; }

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

  /* На тач-устройствах контекстное меню и его пункты чуть крупнее для пальца. */
  .cm-item { padding: 12px 10px; }
}
</style>
