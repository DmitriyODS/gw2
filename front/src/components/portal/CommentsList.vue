<template>
  <div class="comments-list">
    <div v-if="loading" class="comments-loading">
      <BrandLoader :size="40" />
    </div>
    <ul v-else-if="tree.length" class="comments-items">
      <CommentNode
        v-for="node in tree"
        :key="node.comment.id"
        :node="node"
        @open-profile="$emit('open-profile', $event)"
        @reply="startReply"
        @like="toggleLike"
        @delete="remove"
      />
    </ul>
    <div v-else class="comments-status">Комментариев пока нет</div>

    <!-- Кому отвечаем: баннер над полем (как reply в мессенджере). -->
    <div v-if="replyTo" class="comment-reply-banner">
      <span class="material-symbols-outlined">reply</span>
      <span class="comment-reply-text">Ответ: {{ replyAuthorName }}</span>
      <button class="comment-reply-cancel" type="button" aria-label="Отменить ответ" @click="replyTo = null">
        <span class="material-symbols-outlined">close</span>
      </button>
    </div>

    <form class="comment-form" @submit.prevent="submit">
      <InputText
        ref="inputEl"
        v-model="text"
        class="comment-input"
        :placeholder="replyTo ? 'Ваш ответ…' : 'Написать комментарий…'"
        maxlength="2000"
      />
      <button type="submit" class="comment-send" :disabled="!text.trim() || sending" title="Отправить">
        <span class="material-symbols-outlined">send</span>
      </button>
    </form>
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import InputText from 'primevue/inputtext'
import { usePortalStore } from '@/stores/portal.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import CommentNode from '@/components/portal/CommentNode.vue'

const props = defineProps({
  postId: { type: Number, required: true },
})
// Клик по автору — профиль открывает родительский PostCard (один диалог на карточку).
defineEmits(['open-profile'])

const portal = usePortalStore()

const comments = computed(() => portal.commentsByPost[props.postId] || [])
const loading = computed(() => !!portal.loadingComments[props.postId])
const text = ref('')
const sending = ref(false)
const replyTo = ref(null)
const inputEl = ref(null)

// Дерево обсуждения из плоского списка: сервер отдаёт хронологию с
// reply_to_id, вложенность собираем здесь. Ответ на удалённого родителя
// невозможен (каскад уносит ветку), но осиротевший узел на всякий случай
// показываем корневым — потерять комментарий хуже, чем показать не там.
const tree = computed(() => {
  const nodes = new Map()
  for (const c of comments.value) nodes.set(c.id, { comment: c, children: [] })
  const roots = []
  for (const node of nodes.values()) {
    const parent = node.comment.reply_to_id ? nodes.get(node.comment.reply_to_id) : null
    if (parent) parent.children.push(node)
    else roots.push(node)
  }
  return roots
})

const replyAuthorName = computed(() => (replyTo.value
  ? portal.resolveAuthor(replyTo.value.author_id).fio
  : ''))

function startReply(comment) {
  replyTo.value = comment
  nextTick(() => inputEl.value?.$el?.focus())
}

async function submit() {
  const t = text.value.trim()
  if (!t) return
  sending.value = true
  try {
    await portal.createComment(props.postId, t, replyTo.value?.id ?? null)
    text.value = ''
    replyTo.value = null
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось отправить комментарий')
  } finally {
    sending.value = false
  }
}

async function toggleLike(comment) {
  try {
    await portal.likeComment(props.postId, comment.id)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось поставить «Нравится»')
  }
}

async function remove(comment) {
  try {
    await portal.deleteComment(props.postId, comment.id)
    if (replyTo.value?.id === comment.id) replyTo.value = null
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

.comment-reply-banner {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 10px;
  border-radius: var(--radius-md);
  background: var(--color-surface-high);
  border-left: 3px solid var(--color-primary);
  font-size: 12px;
}
.comment-reply-banner .material-symbols-outlined { font-size: 15px; color: var(--color-primary); }
.comment-reply-text { flex: 1; min-width: 0; font-weight: 600; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.comment-reply-cancel {
  width: 24px; height: 24px; min-height: 0; flex-shrink: 0;
  border: none; border-radius: 50%; background: transparent;
  color: var(--color-text-dim); cursor: pointer; display: grid; place-items: center;
}
.comment-reply-cancel:hover { background: var(--color-surface); color: var(--color-text); }
.comment-reply-cancel .material-symbols-outlined { font-size: 14px; color: inherit; }

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
