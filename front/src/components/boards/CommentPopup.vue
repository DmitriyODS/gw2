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

// Попап не уходит за край холста: булавка у правой или нижней кромки
// открывала ветку наполовину за пределами окна.
const style = computed(() => ({
  left: `clamp(8px, ${Math.round(props.anchor.x)}px, calc(100% - 288px))`,
  top: `clamp(8px, ${Math.round(props.anchor.y)}px, calc(100% - 160px))`,
}))

/* Пустая булавка — это брошенный черновик: закрытие убирает её с доски,
   иначе на холсте копились бы обсуждения без единого слова. */
function close() {
  if (isNew.value && !props.readOnly) emit('delete', props.comment)
  else emit('close')
}

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
  <div v-if="comment" class="cp" :style="style" @keydown.esc.stop="close">
    <header class="cp-head">
      <span class="material-symbols-outlined">chat_bubble</span>
      <span class="cp-author">{{ comment.author || 'Комментарий' }}</span>
      <span class="cp-time">{{ when(comment.created_at) }}</span>
      <AppButton
        v-if="!readOnly && !isNew"
        variant="icon"
        size="sm"
        :icon="comment.resolved ? 'undo' : 'check_circle'"
        :label="comment.resolved ? 'Вернуть в работу' : 'Пометить решённым'"
        :title="comment.resolved ? 'Вернуть в работу' : 'Пометить решённым'"
        @click="toggleResolved"
      />
      <AppButton
        v-if="!readOnly && !isNew"
        variant="icon"
        size="sm"
        tone="danger"
        icon="delete"
        label="Удалить"
        title="Удалить"
        @click="emit('delete', comment)"
      />
      <AppButton variant="icon" size="sm" icon="close" label="Закрыть" title="Закрыть" @click="close" />
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
          <AppButton
            variant="icon"
            size="sm"
            tone="primary"
            icon="send"
            label="Отправить"
            title="Отправить"
            :disabled="!reply.trim()"
            @click="addReply"
          />
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
  box-shadow: var(--shadow-md);
}

.cp-head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 8px 8px 4px;
}

.cp-author { flex: 1; min-width: 0; font-size: 13px; font-weight: 600; overflow: hidden; text-overflow: ellipsis; }
.cp-time { font-size: 11px; color: var(--color-text-dim); white-space: nowrap; }

.cp-body { display: flex; flex-direction: column; gap: 8px; padding: 4px 10px 10px; overflow-y: auto; }
.cp-text { margin: 0; font-size: 13px; line-height: 1.45; white-space: pre-wrap; word-break: break-word; }
.cp-text.is-resolved { opacity: 0.6; text-decoration: line-through; }

.cp-replies { display: flex; flex-direction: column; gap: 8px; margin: 0; padding: 0 0 0 10px; list-style: none; border-left: 2px solid var(--color-outline-dim); }
.cp-reply { display: flex; flex-direction: column; gap: 2px; }
.cp-reply-author { font-size: 12px; font-weight: 600; }

.cp-reply-form { display: flex; align-items: flex-end; gap: 4px; }
.cp-input { flex: 1; min-width: 0; font-size: 13px; }
.cp-send { align-self: flex-end; }

</style>
