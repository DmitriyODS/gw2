<template>
  <div class="fc">
    <div v-if="loading" class="fc-loading">Загрузка…</div>

    <div v-for="c in comments" :key="c.id" class="fc-item" :class="{ bot: c.is_bot }">
      <span v-if="c.is_bot" class="fc-avatar bot" aria-hidden="true">👾</span>
      <img v-else class="fc-avatar" :src="avatarUrl(c.author)" :alt="c.author?.fio || ''" />
      <div class="fc-body">
        <div class="fc-head">
          <span class="fc-name">{{ c.is_bot ? 'Грувик' : (c.author?.fio || 'Без имени') }}</span>
          <span v-if="c.is_bot" class="fc-bot-tag">ИИ</span>
          <span class="fc-time">{{ timeOf(c.created_at) }}</span>
        </div>
        <div v-if="quoted(c)" class="fc-quote">↳ {{ quoted(c) }}</div>
        <p class="fc-text">{{ c.text }}</p>
        <div class="fc-actions">
          <button class="fc-link" type="button" @click="replyTo = c">Ответить</button>
          <button
            v-if="canDelete(c)"
            class="fc-link danger"
            type="button"
            @click="remove(c)"
          >Удалить</button>
        </div>
      </div>
    </div>

    <div v-if="replyTo" class="fc-replying">
      <span class="material-symbols-outlined">reply</span>
      <span class="fc-replying-text">Ответ: {{ shortText(replyTo) }}</span>
      <button class="fc-link" type="button" @click="replyTo = null" aria-label="Отменить ответ">
        <span class="material-symbols-outlined">close</span>
      </button>
    </div>

    <form class="fc-input" @submit.prevent="send">
      <input
        v-model.trim="text"
        placeholder="Написать комментарий…"
        maxlength="2000"
      />
      <button class="fc-send" type="submit" :disabled="!text || sending" aria-label="Отправить">
        <span class="material-symbols-outlined">send</span>
      </button>
    </form>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import { avatarUrl } from '@/utils/groove.js'

const props = defineProps({
  eventId: { type: Number, required: true },
})

const groove = useGrooveStore()
const notify = useNotificationsStore()
const { myLevel, ROLES } = usePermission()

const loading = ref(false)
const sending = ref(false)
const text = ref('')
const replyTo = ref(null)

const comments = computed(() => groove.commentsByEvent[props.eventId] || [])

onMounted(async () => {
  loading.value = true
  try { await groove.fetchComments(props.eventId) } catch {}
  loading.value = false
})

function timeOf(iso) {
  return new Date(iso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function shortText(c) {
  const t = c.text || ''
  return t.length > 60 ? t.slice(0, 60) + '…' : t
}

function quoted(c) {
  if (!c.reply_to_id) return null
  const parent = comments.value.find(x => x.id === c.reply_to_id)
  if (!parent) return null
  const name = parent.is_bot ? 'Грувик' : (parent.author?.fio || '')
  return `${name}: ${shortText(parent)}`
}

function canDelete(c) {
  if (c.is_bot) return myLevel.value >= ROLES.ADMIN
  return c.author?.id === groove.myId || myLevel.value >= ROLES.ADMIN
}

async function send() {
  if (!text.value || sending.value) return
  sending.value = true
  try {
    await groove.addComment(props.eventId, text.value, replyTo.value?.id || null)
    text.value = ''
    replyTo.value = null
  } catch (e) {
    notify.error(e?.message || 'Не удалось отправить комментарий')
  } finally {
    sending.value = false
  }
}

async function remove(c) {
  try {
    await groove.removeComment(props.eventId, c.id)
  } catch (e) {
    notify.error(e?.message || 'Не удалось удалить комментарий')
  }
}
</script>

<style scoped>
.fc {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding-top: 10px;
  border-top: 1px solid var(--color-outline-dim);
  margin-top: 10px;
}
.fc-loading { font-size: 12px; color: var(--color-text-dim); }
.fc-item { display: flex; gap: 8px; }
.fc-avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
}
.fc-avatar.bot {
  display: grid;
  place-items: center;
  font-size: 15px;
  background: var(--color-tertiary-container);
}
.fc-body { min-width: 0; flex: 1; }
.fc-head { display: flex; align-items: baseline; gap: 6px; flex-wrap: wrap; }
.fc-name { font-size: 13px; font-weight: 600; }
.fc-bot-tag {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}
.fc-time { font-size: 11px; color: var(--color-text-dim); }
.fc-quote {
  font-size: 12px;
  color: var(--color-text-dim);
  border-left: 2px solid var(--color-outline-dim);
  padding-left: 8px;
  margin: 2px 0;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.fc-text { margin: 2px 0 0; font-size: 13.5px; line-height: 1.45; word-break: break-word; }
.fc-item.bot .fc-text { font-style: italic; }
.fc-actions { display: flex; gap: 10px; margin-top: 2px; }
.fc-link {
  background: none;
  border: none;
  padding: 0;
  font-size: 12px;
  color: var(--color-primary);
  cursor: pointer;
  display: inline-flex;
  align-items: center;
}
.fc-link.danger { color: var(--color-error); }
.fc-link .material-symbols-outlined { font-size: 16px; }
.fc-replying {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--color-text-dim);
  background: var(--color-surface-high);
  border-radius: var(--radius-md, 10px);
  padding: 4px 8px;
}
.fc-replying .material-symbols-outlined { font-size: 16px; }
.fc-replying-text {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.fc-input { display: flex; gap: 6px; }
.fc-input input {
  flex: 1;
  min-width: 0;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  padding: 7px 14px;
  font-size: 13.5px;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
}
.fc-input input:focus { border-color: var(--color-primary); }
.fc-send {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  border: none;
  display: grid;
  place-items: center;
  background: var(--color-primary);
  color: var(--color-on-primary);
  cursor: pointer;
  flex-shrink: 0;
}
.fc-send:disabled { opacity: 0.45; cursor: default; }
.fc-send .material-symbols-outlined { font-size: 17px; }
</style>
