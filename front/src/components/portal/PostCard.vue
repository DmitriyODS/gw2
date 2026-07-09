<template>
  <article class="post-card" :class="{ pinned: !!post.pinned_at }">
    <header class="post-head">
      <!-- Автор кликабелен, только пока он состоит в компании (есть в
           каталоге сотрудников) — иначе профиль показать нечем. -->
      <button
        v-if="authorClickable"
        class="post-avatar-btn"
        type="button"
        :aria-label="`Открыть профиль: ${author.fio}`"
        @click="openAuthorProfile(post.author_id)"
      >
        <img class="post-avatar" :src="author.avatarUrl" :alt="author.fio" />
      </button>
      <img v-else class="post-avatar" :src="author.avatarUrl" :alt="author.fio" />
      <div class="post-head-info">
        <div class="post-head-top">
          <button
            v-if="authorClickable"
            class="post-author post-author-link"
            type="button"
            @click="openAuthorProfile(post.author_id)"
          >{{ author.fio }}</button>
          <span v-else class="post-author">{{ author.fio }}</span>
          <span v-if="topic" class="post-topic-chip" :style="topicChipStyle">{{ topic.name }}</span>
        </div>
        <div class="post-meta">
          <span>{{ formattedDate }}</span>
          <span v-if="post.updated_at && post.updated_at !== post.created_at"> · изменено</span>
          <span v-if="post.pinned_at && post.pinned_until" class="post-pin-until"> · закреплено до {{ pinnedUntilText }}</span>
        </div>
      </div>
      <div v-if="post.pinned_at" class="post-pin-badge" title="Закреплено">
        <span class="material-symbols-outlined">keep</span>
      </div>
      <div v-if="canManage" ref="menuRef" class="post-menu">
        <button class="post-icon-btn" title="Ещё" aria-label="Действия с постом" @click="toggleMenu">
          <span class="material-symbols-outlined">more_vert</span>
        </button>
        <div v-if="menuOpen" class="post-menu-pop">
          <button v-if="post.pinned_at" class="post-menu-item" @click="onUnpin">
            <span class="material-symbols-outlined">keep_off</span>
            Открепить
          </button>
          <template v-else>
            <!-- Инлайн-подменю выбора срока закрепления -->
            <button class="post-menu-item" @click="pinChoicesOpen = !pinChoicesOpen">
              <span class="material-symbols-outlined">keep</span>
              Закрепить
              <span class="material-symbols-outlined post-menu-caret">{{ pinChoicesOpen ? 'expand_less' : 'expand_more' }}</span>
            </button>
            <template v-if="pinChoicesOpen">
              <button
                v-for="opt in PIN_OPTIONS"
                :key="opt.label"
                class="post-menu-item post-menu-sub"
                @click="onPin(opt.days)"
              >{{ opt.label }}</button>
            </template>
          </template>
          <button class="post-menu-item" @click="onEdit">
            <span class="material-symbols-outlined">edit</span>
            Редактировать
          </button>
          <button class="post-menu-item danger" @click="onDelete">
            <span class="material-symbols-outlined">delete</span>
            Удалить
          </button>
        </div>
      </div>
    </header>

    <h3 v-if="post.title" class="post-title">{{ post.title }}</h3>
    <div class="post-body">
      <LinkifiedText :text="displayBody" />
      <button v-if="isTruncated" class="post-more-btn" @click="expanded = !expanded">
        {{ expanded ? 'Свернуть' : 'Показать полностью' }}
      </button>
    </div>

    <div v-if="images.length" class="post-images" :class="`cols-${Math.min(images.length, 3)}`">
      <a v-for="a in images" :key="a.id" class="post-image" :href="a.url" target="_blank" rel="noopener noreferrer">
        <img :src="a.url" :alt="a.name" loading="lazy" />
      </a>
    </div>

    <div v-if="files.length" class="post-files">
      <a v-for="a in files" :key="a.id" class="post-file" :href="a.url" target="_blank" rel="noopener noreferrer">
        <span class="material-symbols-outlined">description</span>
        <span class="post-file-name">{{ a.name }}</span>
        <span class="post-file-size">{{ formatSize(a.size) }}</span>
      </a>
    </div>

    <footer class="post-footer">
      <div class="post-reactions">
        <button
          v-for="emoji in REACTIONS"
          :key="emoji"
          class="post-reaction"
          :class="{ active: myReactions.has(emoji) }"
          type="button"
          :aria-label="`Реакция ${emoji}`"
          :aria-pressed="myReactions.has(emoji)"
          @click="toggleReaction(emoji)"
        >
          <span aria-hidden="true">{{ emoji }}</span>
          <span v-if="reactionCounts[emoji]" class="post-reaction-count">{{ reactionCounts[emoji] }}</span>
        </button>
      </div>
      <button class="post-action" type="button" @click="commentsOpen = !commentsOpen">
        <span class="material-symbols-outlined">chat_bubble</span>
        {{ post.comment_count || 0 }}
      </button>
      <button class="post-action" type="button" @click="$emit('forward', post)">
        <span class="material-symbols-outlined">forward</span>
        Переслать
      </button>
    </footer>

    <CommentsList
      v-if="commentsOpen"
      :post-id="post.id"
      class="post-comments"
      @open-profile="openAuthorProfile"
    />

    <!-- Профиль автора: монтируем лениво после первого клика — на ленте
         десятки карточек, пустые диалоги в каждой не нужны. -->
    <EmployeeProfileDialog v-if="profileUser" v-model="profileOpen" :user="profileUser" />
  </article>
</template>

<script setup>
import { computed, defineAsyncComponent, onBeforeUnmount, onMounted, ref } from 'vue'
import { usePortalStore } from '@/stores/portal.js'
import { useAuthStore } from '@/stores/auth.js'
import { usePermission } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import LinkifiedText from '@/components/common/LinkifiedText.vue'
import CommentsList from './CommentsList.vue'

// Async: диалог профиля тянет тяжёлые сторы (звонки) — грузим по первому клику.
const EmployeeProfileDialog = defineAsyncComponent(() =>
  import('@/components/common/EmployeeProfileDialog.vue'))

const props = defineProps({
  post: { type: Object, required: true },
})
const emit = defineEmits(['edit', 'delete', 'forward'])

const portal = usePortalStore()
const auth = useAuthStore()
const { isAdmin } = usePermission()

const REACTIONS = ['👍', '❤️', '🎉', '😂', '👏']
const BODY_LIMIT = 320

const author = computed(() => portal.resolveAuthor(props.post.author_id))

// Профиль автора: данные — из уже загруженного каталога сотрудников
// (portal.loadAuthors → getDirectory, тот же источник, что у EmployeesView).
const authorClickable = computed(() => portal.authorMap.has(props.post.author_id))
const profileOpen = ref(false)
const profileUser = ref(null)

function openAuthorProfile(id) {
  const u = portal.authorMap.get(id)
  if (!u) return // автор уже не сотрудник компании — профиля нет
  profileUser.value = u
  profileOpen.value = true
}
const topic = computed(() => portal.topics.find((t) => t.id === props.post.topic_id) || null)
const topicChipStyle = computed(() => (topic.value?.color
  ? { background: `var(--tag-${topic.value.color}-surface)`, borderColor: `var(--tag-${topic.value.color}-border)`, color: 'var(--color-text)' }
  : {}))

const canManage = computed(() => props.post.author_id === auth.userId || isAdmin())

const reactionCounts = computed(() => props.post.reaction_counts || {})
const myReactions = computed(() => new Set(props.post.my_reactions || []))

const expanded = ref(false)
const isTruncated = computed(() => (props.post.body || '').length > BODY_LIMIT)
const displayBody = computed(() => {
  if (!isTruncated.value || expanded.value) return props.post.body || ''
  return props.post.body.slice(0, BODY_LIMIT) + '…'
})

const images = computed(() => (props.post.attachments || []).filter((a) => a.mime?.startsWith('image/')))
const files = computed(() => (props.post.attachments || []).filter((a) => !a.mime?.startsWith('image/')))

function formatSize(bytes) {
  if (!bytes) return ''
  if (bytes < 1024) return `${bytes} Б`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} КБ`
  return `${(bytes / 1024 / 1024).toFixed(1)} МБ`
}

const formattedDate = computed(() => {
  if (!props.post.created_at) return ''
  return new Date(props.post.created_at).toLocaleString('ru', {
    day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit',
  })
})

// Срок закрепления: «закреплено до DD.MM».
const pinnedUntilText = computed(() => {
  if (!props.post.pinned_until) return ''
  return new Date(props.post.pinned_until).toLocaleDateString('ru', { day: '2-digit', month: '2-digit' })
})

const menuOpen = ref(false)
const menuRef = ref(null)
const commentsOpen = ref(false)

// Подменю срока сворачивается при каждом открытии меню заново.
function toggleMenu() {
  menuOpen.value = !menuOpen.value
  pinChoicesOpen.value = false
}

// Закрытие меню «⋮» кликом мимо и по Escape (паттерн MessageContextMenu);
// клики внутри .post-menu не считаются «мимо», иначе mousedown закроет
// меню, а следующий click по той же кнопке тут же откроет его снова.
function onDocPointerDown(e) {
  if (menuOpen.value && !menuRef.value?.contains(e.target)) menuOpen.value = false
}
function onDocKeydown(e) {
  if (e.key === 'Escape' && menuOpen.value) menuOpen.value = false
}

onMounted(() => {
  document.addEventListener('mousedown', onDocPointerDown, true)
  document.addEventListener('touchstart', onDocPointerDown, { passive: true, capture: true })
  document.addEventListener('keydown', onDocKeydown)
})

onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocPointerDown, true)
  document.removeEventListener('touchstart', onDocPointerDown, true)
  document.removeEventListener('keydown', onDocKeydown)
})

async function toggleReaction(emoji) {
  const notif = useNotificationsStore()
  try {
    if (myReactions.value.has(emoji)) await portal.removeReaction(props.post.id, emoji)
    else await portal.addReaction(props.post.id, emoji)
  } catch (e) {
    notif.error(e?.message || 'Не удалось поставить реакцию')
  }
}

// Выбор срока закрепления: 1/7/30 дней или null — бессрочно.
const PIN_OPTIONS = [
  { days: 1, label: '1 день' },
  { days: 7, label: '7 дней' },
  { days: 30, label: '30 дней' },
  { days: null, label: 'Бессрочно' },
]
const pinChoicesOpen = ref(false)

async function onPin(days) {
  menuOpen.value = false
  pinChoicesOpen.value = false
  try {
    await portal.pinPost(props.post.id, days)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось закрепить пост')
  }
}

async function onUnpin() {
  menuOpen.value = false
  try {
    await portal.unpinPost(props.post.id)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось открепить пост')
  }
}

function onEdit() {
  menuOpen.value = false
  emit('edit', props.post)
}

function onDelete() {
  menuOpen.value = false
  emit('delete', props.post)
}
</script>

<style scoped>
.post-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  border-radius: var(--radius-lg);
  /* Карточка в потоке ленты: полупрозрачная подложка без blur (см. tokens.css). */
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--shadow-sm);
}

.post-card.pinned {
  box-shadow: var(--shadow-sm), inset 3px 0 0 0 var(--color-tertiary);
}

.post-head {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.post-avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.post-avatar-btn {
  padding: 0;
  border: none;
  background: transparent;
  border-radius: 50%;
  line-height: 0;
  flex-shrink: 0;
  cursor: pointer;
  transition: box-shadow .12s;
}
.post-avatar-btn:hover,
.post-avatar-btn:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 30%, transparent);
}

.post-head-info { min-width: 0; flex: 1; }

.post-head-top {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* Кликабельное имя автора (до .post-author, чтобы font: inherit не перебил
   размер/насыщенность имени). */
.post-author-link {
  border: none;
  background: transparent;
  padding: 0;
  font: inherit;
  text-align: left;
  cursor: pointer;
  border-radius: var(--radius-xs);
  transition: color .12s;
}
.post-author-link:hover,
.post-author-link:focus-visible {
  color: var(--color-primary);
  text-decoration: underline;
}

.post-author {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
}

.post-topic-chip {
  font-size: 11.5px;
  font-weight: 600;
  padding: 2px 9px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
}

.post-meta {
  font-size: 12px;
  color: var(--color-text-dim);
  margin-top: 2px;
}

.post-pin-badge {
  color: var(--color-tertiary);
  flex-shrink: 0;
  display: flex;
  align-items: center;
}

.post-pin-badge .material-symbols-outlined {
  font-size: 20px;
  font-variation-settings: 'FILL' 1;
}

.post-menu { position: relative; flex-shrink: 0; }

.post-icon-btn {
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.post-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }

.post-menu-pop {
  position: absolute;
  top: calc(100% + 4px);
  right: 0;
  z-index: 20;
  min-width: 190px;
  /* Плавающий поповер — стекло (Expressive Glass). */
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 6px;
  display: flex;
  flex-direction: column;
}

.post-menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 10px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  text-align: left;
  cursor: pointer;
}
.post-menu-item:hover { background: var(--color-surface-high); }
.post-menu-item.danger { color: var(--color-error); }
.post-menu-item .material-symbols-outlined { font-size: 18px; }

/* Инлайн-подменю выбора срока закрепления */
.post-menu-caret { margin-left: auto; color: var(--color-text-dim); }
.post-menu-sub { padding-left: 38px; font-size: 13px; }

.post-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  color: var(--color-text);
}

.post-body {
  font-size: 14px;
  line-height: 1.5;
  color: var(--color-text);
  white-space: pre-wrap;
  word-break: break-word;
}

.post-more-btn {
  display: block;
  margin-top: 4px;
  border: none;
  background: transparent;
  color: var(--color-primary);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  padding: 0;
}

.post-images {
  display: grid;
  gap: 6px;
  grid-template-columns: repeat(2, 1fr);
}
.post-images.cols-1 { grid-template-columns: 1fr; }

.post-image {
  display: block;
  border-radius: var(--radius-md);
  overflow: hidden;
  aspect-ratio: 4 / 3;
  background: var(--color-surface-high);
}
.post-image img { width: 100%; height: 100%; object-fit: cover; display: block; }

.post-files {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.post-file {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-outline-dim);
  color: var(--color-text);
  text-decoration: none;
  font-size: 13px;
}
.post-file:hover { background: var(--color-surface-high); }
.post-file .material-symbols-outlined { color: var(--color-primary); font-size: 20px; }
.post-file-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.post-file-size { color: var(--color-text-dim); font-size: 12px; flex-shrink: 0; }

.post-footer {
  display: flex;
  align-items: center;
  gap: 6px;
  padding-top: 6px;
  border-top: 1px solid var(--color-outline-dim);
  flex-wrap: wrap;
}

.post-reactions {
  display: flex;
  gap: 4px;
  flex-wrap: wrap;
}

.post-reaction {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 36px;
  padding: 4px 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: transparent;
  font-size: 14px;
  cursor: pointer;
  line-height: 1;
}
@media (max-width: 768px) {
  .post-reaction { min-height: 44px; padding: 4px 12px; }
}
.post-reaction:hover { background: var(--color-surface-high); }
.post-reaction.active {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
}
.post-reaction-count { font-size: 11.5px; font-weight: 700; color: var(--color-text-dim); }
.post-reaction.active .post-reaction-count { color: var(--color-on-primary-container); }

.post-action {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  margin-left: auto;
  padding: 6px 10px;
  border: none;
  background: transparent;
  border-radius: var(--radius-full);
  color: var(--color-text-dim);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.post-action:hover { background: var(--color-surface-high); color: var(--color-text); }
.post-action .material-symbols-outlined { font-size: 18px; }
.post-action + .post-action { margin-left: 0; }

.post-comments {
  margin-top: 4px;
  padding-top: 10px;
  border-top: 1px solid var(--color-outline-dim);
}
</style>
