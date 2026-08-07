<script setup>
/* Обсуждение у булавки комментария: текст автора, ответы веткой и пометка
   «решено». Живёт в самой сцене — отдельного хранилища у комментариев нет,
   поэтому они едут вместе с доской (в том числе соавторам и по ссылке). */
import { computed, nextTick, ref, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import Textarea from 'primevue/textarea'

const props = defineProps({
  comment: { type: Object, default: null },
  // Экранная позиция булавки — попап встаёт рядом.
  anchor: { type: Object, default: () => ({ x: 0, y: 0 }) },
  me: { type: Object, default: () => ({}) },
  readOnly: { type: Boolean, default: false },
})

const emit = defineEmits(['update', 'delete', 'close'])

const draft = ref('')
const reply = ref('')
const input = ref(null)

const isNew = computed(() => !props.comment?.text)
const replies = computed(() => props.comment?.replies || [])

watch(() => props.comment?.id, () => {
  draft.value = props.comment?.text || ''
  reply.value = ''
  if (isNew.value) nextTick(() => (input.value?.$el || input.value)?.focus?.())
}, { immediate: true })

const style = computed(() => ({
  left: `${Math.round(props.anchor.x)}px`,
  top: `${Math.round(props.anchor.y)}px`,
}))

function when(iso) {
  if (!iso) return ''
  const d = new Date(iso)
  return d.toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

function saveText() {
  const text = draft.value.trim()
  if (!text) return
  emit('update', { ...props.comment, text })
}

function addReply() {
  const text = reply.value.trim()
  if (!text) return
  emit('update', {
    ...props.comment,
    replies: [...replies.value, {
      author_id: props.me?.id ?? null,
      author: props.me?.fio || 'Участник',
      text,
      created_at: new Date().toISOString(),
    }],
  })
  reply.value = ''
}

function toggleResolved() {
  emit('update', { ...props.comment, resolved: !props.comment.resolved })
}
</script>

<template>
  <div v-if="comment" class="cp" :style="style">
    <header class="cp-head">
      <span class="material-symbols-outlined">chat_bubble</span>
      <span class="cp-author">{{ comment.author || 'Комментарий' }}</span>
      <span class="cp-time">{{ when(comment.created_at) }}</span>
      <button
        v-if="!readOnly && !isNew"
        type="button"
        class="cp-icon"
        :title="comment.resolved ? 'Вернуть в работу' : 'Пометить решённым'"
        @click="toggleResolved"
      >
        <span class="material-symbols-outlined">{{ comment.resolved ? 'undo' : 'check_circle' }}</span>
      </button>
      <button v-if="!readOnly" type="button" class="cp-icon cp-icon--danger" title="Удалить" @click="emit('delete', comment)">
        <span class="material-symbols-outlined">delete</span>
      </button>
      <button type="button" class="cp-icon" title="Закрыть" @click="emit('close')">
        <span class="material-symbols-outlined">close</span>
      </button>
    </header>

    <div class="cp-body">
      <template v-if="isNew && !readOnly">
        <Textarea
          ref="input"
          v-model="draft"
          class="cp-input"
          rows="2"
          auto-resize
          placeholder="Что обсудим?"
          @keydown.enter.exact.prevent="saveText"
        />
        <AppButton variant="filled" label="Добавить" class="cp-send" @click="saveText" />
      </template>

      <template v-else>
        <p class="cp-text" :class="{ 'is-resolved': comment.resolved }">{{ comment.text }}</p>

        <ul v-if="replies.length" class="cp-replies">
          <li v-for="(r, i) in replies" :key="i" class="cp-reply">
            <span class="cp-reply-author">{{ r.author }}</span>
            <span class="cp-time">{{ when(r.created_at) }}</span>
            <p class="cp-text">{{ r.text }}</p>
          </li>
        </ul>

        <div v-if="!readOnly" class="cp-reply-form">
          <Textarea
            v-model="reply"
            class="cp-input"
            rows="1"
            auto-resize
            placeholder="Ответить…"
            @keydown.enter.exact.prevent="addReply"
          />
          <button type="button" class="cp-icon" title="Отправить" @click="addReply">
            <span class="material-symbols-outlined">send</span>
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.cp {
  position: absolute;
  z-index: 5;
  width: 280px;
  max-height: 60%;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  box-shadow: var(--shadow-2);
}

.cp-head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 8px 4px;
}

.cp-author { flex: 1; min-width: 0; font-size: 13px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; }
.cp-time { font-size: 11px; color: var(--color-text-muted); white-space: nowrap; }

.cp-body { display: flex; flex-direction: column; gap: 8px; padding: 4px 10px 10px; overflow-y: auto; }
.cp-text { margin: 0; font-size: 13px; line-height: 1.45; white-space: pre-wrap; word-break: break-word; }
.cp-text.is-resolved { opacity: 0.6; text-decoration: line-through; }

.cp-replies { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0 0 0 10px; list-style: none; border-left: 2px solid var(--color-outline-variant); }
.cp-reply { display: flex; flex-direction: column; gap: 2px; }
.cp-reply-author { font-size: 12px; font-weight: 600; }

.cp-reply-form { display: flex; align-items: flex-end; gap: 4px; }
.cp-input { flex: 1; min-width: 0; font-size: 13px; }
.cp-send { align-self: flex-end; }

.cp-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  max-width: 28px;
  min-height: 28px;
  max-height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-muted);
  cursor: pointer;
}

.cp-icon:hover { background: var(--color-surface-variant); color: var(--color-text); }
.cp-icon--danger:hover { color: var(--color-error); }
.cp-icon .material-symbols-outlined { font-size: 18px; }
</style>
