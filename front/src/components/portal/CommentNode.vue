<template>
  <li class="cn-item">
    <div class="cn-row">
      <!-- Автор кликабелен, только пока он есть в каталоге сотрудников. -->
      <button
        v-if="known"
        class="cn-avatar-btn"
        type="button"
        :aria-label="`Открыть профиль: ${author.fio}`"
        @click="emit('open-profile', comment.author_id)"
      >
        <img class="cn-avatar" :src="author.avatarUrl" :alt="author.fio" />
      </button>
      <img v-else class="cn-avatar" :src="author.avatarUrl" :alt="author.fio" />

      <div class="cn-body">
        <div class="cn-head">
          <button
            v-if="known"
            class="cn-author cn-author-link"
            type="button"
            @click="emit('open-profile', comment.author_id)"
          >{{ author.fio }}</button>
          <span v-else class="cn-author">{{ author.fio }}</span>
          <span class="cn-time">{{ formatTime(comment.created_at) }}</span>
        </div>
        <MarkdownView class="cn-md" :source="comment.text" />

        <div class="cn-actions">
          <button
            class="cn-action"
            :class="{ liked: comment.liked }"
            type="button"
            :aria-pressed="!!comment.liked"
            :aria-label="comment.liked ? 'Убрать «Нравится»' : 'Нравится'"
            @click="emit('like', comment)"
          >
            <span class="material-symbols-outlined">{{ comment.liked ? 'favorite' : 'favorite_border' }}</span>
            <span v-if="comment.like_count">{{ comment.like_count }}</span>
            <span v-else>Нравится</span>
          </button>
          <button class="cn-action" type="button" @click="emit('reply', comment)">
            <span class="material-symbols-outlined">reply</span> Ответить
          </button>
        </div>
      </div>

      <button
        v-if="canDelete"
        class="cn-delete"
        type="button"
        title="Удалить"
        aria-label="Удалить комментарий"
        @click="emit('delete', comment)"
      >
        <span class="material-symbols-outlined">delete</span>
      </button>
    </div>

    <!-- Ветка ответов: рекурсия по дереву. Отступ растёт только до
         MAX_INDENT_DEPTH — иначе глубокий тред уезжает за край на мобильном. -->
    <ul v-if="node.children.length" class="cn-children" :class="{ flat: depth >= MAX_INDENT_DEPTH }">
      <CommentNode
        v-for="child in node.children"
        :key="child.comment.id"
        :node="child"
        :depth="depth + 1"
        @open-profile="emit('open-profile', $event)"
        @reply="emit('reply', $event)"
        @like="emit('like', $event)"
        @delete="emit('delete', $event)"
      />
    </ul>
  </li>
</template>

<script setup>
// Узел дерева обсуждения: сам комментарий + рекурсивно его ответы.
import { computed } from 'vue'
import MarkdownView from '@/components/common/MarkdownView.vue'
import { usePortalStore } from '@/stores/portal.js'
import { useAuthStore } from '@/stores/auth.js'
import { usePermission } from '@/composables/usePermission.js'

defineOptions({ name: 'CommentNode' }) // имя нужно самому себе — рекурсия

const props = defineProps({
  // { comment, children: [node…] } — дерево строит CommentsList.
  node: { type: Object, required: true },
  depth: { type: Number, default: 0 },
})
const emit = defineEmits(['open-profile', 'reply', 'like', 'delete'])

const MAX_INDENT_DEPTH = 3

const portal = usePortalStore()
const auth = useAuthStore()
const { isAdmin } = usePermission()

const comment = computed(() => props.node.comment)
const author = computed(() => portal.resolveAuthor(comment.value.author_id))
const known = computed(() => portal.authorMap.has(comment.value.author_id))
const canDelete = computed(() => comment.value.author_id === auth.userId || isAdmin())

function formatTime(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleString('ru', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
}
</script>

<style scoped>
.cn-item { display: flex; flex-direction: column; gap: 8px; }
.cn-row { display: flex; align-items: flex-start; gap: 8px; }

.cn-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.cn-avatar-btn {
  padding: 0;
  border: none;
  background: transparent;
  border-radius: 50%;
  line-height: 0;
  flex-shrink: 0;
  cursor: pointer;
  transition: box-shadow .12s;
}
.cn-avatar-btn:hover,
.cn-avatar-btn:focus-visible {
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 30%, transparent);
}

.cn-body {
  min-width: 0;
  flex: 1;
  background: var(--color-surface-high);
  border-radius: var(--radius-md);
  padding: 6px 10px;
  font-size: 13.5px;
  color: var(--color-text);
}

.cn-head { display: flex; align-items: baseline; gap: 8px; margin-bottom: 2px; }

/* Кликабельное имя автора (до .cn-author, чтобы font: inherit не перебил
   размер/насыщенность имени). */
.cn-author-link {
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
.cn-author-link:hover,
.cn-author-link:focus-visible { color: var(--color-primary); text-decoration: underline; }

.cn-author { font-weight: 700; font-size: 12.5px; }
.cn-time { font-size: 11px; color: var(--color-text-dim); }

.cn-actions { display: flex; gap: 4px; margin-top: 4px; }
.cn-action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 0;
  padding: 3px 8px;
  border: none;
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  font: inherit;
  font-size: 11.5px;
  font-weight: 700;
  cursor: pointer;
}
.cn-action:hover { background: var(--color-surface); color: var(--color-text); }
.cn-action.liked { color: var(--color-error); }
.cn-action .material-symbols-outlined { font-size: 15px; }

.cn-delete {
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
.cn-delete:hover { background: var(--color-surface-high); color: var(--color-error); }
.cn-delete .material-symbols-outlined { font-size: 17px; }

.cn-children {
  list-style: none;
  margin: 0;
  padding: 0 0 0 18px;
  display: flex;
  flex-direction: column;
  gap: 8px;
  border-left: 2px solid var(--color-outline-dim);
  margin-left: 13px; /* центр аватарки родителя */
}
/* Глубже — ветку продолжаем без нового отступа: дерево не должно уползать. */
.cn-children.flat { padding-left: 0; margin-left: 0; border-left: none; }
</style>
