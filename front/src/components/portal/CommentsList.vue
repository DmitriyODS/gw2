<template>
  <div class="comments-list">
    <div v-if="loading" class="comments-loading">
      <ProgressSpinner style="width:24px;height:24px" />
    </div>
    <ul v-else-if="comments.length" class="comments-items">
      <li v-for="c in comments" :key="c.id" class="comment-item">
        <!-- Автор кликабелен, только пока он есть в каталоге сотрудников. -->
        <button
          v-if="isKnown(c.author_id)"
          class="comment-avatar-btn"
          type="button"
          :aria-label="`Открыть профиль: ${authorOf(c.author_id).fio}`"
          @click="$emit('open-profile', c.author_id)"
        >
          <img class="comment-avatar" :src="authorOf(c.author_id).avatarUrl" :alt="authorOf(c.author_id).fio" />
        </button>
        <img v-else class="comment-avatar" :src="authorOf(c.author_id).avatarUrl" :alt="authorOf(c.author_id).fio" />
        <div class="comment-body">
          <div class="comment-head">
            <button
              v-if="isKnown(c.author_id)"
              class="comment-author comment-author-link"
              type="button"
              @click="$emit('open-profile', c.author_id)"
            >{{ authorOf(c.author_id).fio }}</button>
            <span v-else class="comment-author">{{ authorOf(c.author_id).fio }}</span>
            <span class="comment-time">{{ formatTime(c.created_at) }}</span>
          </div>
          <LinkifiedText :text="c.text" />
        </div>
        <button v-if="canDelete(c)" class="comment-delete" title="Удалить" aria-label="Удалить комментарий" @click="remove(c.id)">
          <span class="material-symbols-outlined">delete</span>
        </button>
      </li>
    </ul>
    <div v-else class="comments-status">Комментариев пока нет</div>

    <form class="comment-form" @submit.prevent="submit">
      <input v-model="text" class="comment-input" placeholder="Написать комментарий…" maxlength="2000" />
      <button type="submit" class="comment-send" :disabled="!text.trim() || sending" title="Отправить">
        <span class="material-symbols-outlined">send</span>
      </button>
    </form>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { usePortalStore } from '@/stores/portal.js'
import { useAuthStore } from '@/stores/auth.js'
import { usePermission } from '@/composables/usePermission.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import LinkifiedText from '@/components/common/LinkifiedText.vue'

const props = defineProps({
  postId: { type: Number, required: true },
})
// Клик по автору — профиль открывает родительский PostCard (один диалог на карточку).
defineEmits(['open-profile'])

const portal = usePortalStore()
const auth = useAuthStore()
const { isAdmin } = usePermission()

const comments = computed(() => portal.commentsByPost[props.postId] || [])
const loading = computed(() => !!portal.loadingComments[props.postId])
const text = ref('')
const sending = ref(false)

function authorOf(id) {
  return portal.resolveAuthor(id)
}

function isKnown(id) {
  return portal.authorMap.has(id)
}

function canDelete(c) {
  return c.author_id === auth.userId || isAdmin()
}

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('ru', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}

async function submit() {
  const t = text.value.trim()
  if (!t) return
  sending.value = true
  try {
    await portal.createComment(props.postId, t)
    text.value = ''
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось отправить комментарий')
  } finally {
    sending.value = false
  }
}

async function remove(id) {
  try {
    await portal.deleteComment(props.postId, id)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось удалить комментарий')
  }
}

onMounted(() => portal.fetchComments(props.postId))
</script>

<style scoped>
.comments-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.comments-status {
  font-size: 13px;
  color: var(--color-text-dim);
  padding: 4px 0;
}

.comments-loading {
  display: flex;
  justify-content: center;
  padding: 8px 0;
}

.comments-items {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.comment-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}

.comment-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}

.comment-avatar-btn {
  padding: 0;
  border: none;
  background: transparent;
  border-radius: 50%;
  line-height: 0;
  flex-shrink: 0;
  cursor: pointer;
  transition: box-shadow .12s;
}
.comment-avatar-btn:hover,
.comment-avatar-btn:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 30%, transparent);
}

.comment-body {
  min-width: 0;
  flex: 1;
  background: var(--color-surface-high);
  border-radius: var(--radius-md);
  padding: 6px 10px;
  font-size: 13.5px;
  color: var(--color-text);
}

.comment-head {
  display: flex;
  align-items: baseline;
  gap: 8px;
  margin-bottom: 2px;
}

/* Кликабельное имя автора (до .comment-author, чтобы font: inherit не перебил
   размер/насыщенность имени). */
.comment-author-link {
  border: none;
  background: transparent;
  padding: 0;
  font: inherit;
  color: inherit;
  text-align: left;
  cursor: pointer;
  border-radius: var(--radius-xs);
  transition: color .12s;
}
.comment-author-link:hover,
.comment-author-link:focus-visible {
  color: var(--color-primary);
  text-decoration: underline;
}

.comment-author { font-weight: 700; font-size: 12.5px; }
.comment-time { font-size: 11px; color: var(--color-text-dim); }

.comment-delete {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.comment-delete:hover { background: var(--color-surface-high); color: var(--color-error); }
.comment-delete .material-symbols-outlined { font-size: 17px; }

.comment-form {
  display: flex;
  gap: 8px;
  align-items: center;
}

.comment-input {
  flex: 1;
  min-width: 0;
  padding: 8px 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  font: inherit;
  font-size: 13.5px;
  outline: none;
}
.comment-input:focus { border-color: var(--color-primary); }

.comment-send {
  width: 36px;
  height: 36px;
  flex-shrink: 0;
  border: none;
  border-radius: 50%;
  background: var(--color-primary);
  color: var(--color-on-primary);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.comment-send:disabled { opacity: 0.5; cursor: not-allowed; }
.comment-send .material-symbols-outlined { font-size: 18px; }
</style>
