<script setup>
import { ref, computed, onMounted, nextTick } from 'vue'
import { useTasksStore } from '@/stores/tasks.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import MarkdownView from '@/components/common/MarkdownView.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

const props = defineProps({
  taskId: { type: Number, required: true },
})

const tasks = useTasksStore()
const auth = useAuthStore()
const notify = useNotificationsStore()
const { isAtLeast } = usePermission()

const loading = ref(false)
const draft = ref('')
const sending = ref(false)
const editingId = ref(null)
const editText = ref('')
const listEl = ref(null)
const deletingId = ref(null)

const list = computed(() => tasks.commentsByTask[props.taskId] || [])

async function load() {
  loading.value = true
  try {
    await tasks.loadComments(props.taskId)
    await nextTick()
    scrollBottom()
  } catch (e) {
    notify.error(e?.message || 'Не удалось загрузить комментарии')
  } finally {
    loading.value = false
  }
}

function scrollBottom() {
  if (listEl.value) listEl.value.scrollTop = listEl.value.scrollHeight
}

async function send() {
  const txt = draft.value.trim()
  if (!txt || sending.value) return
  sending.value = true
  try {
    await tasks.addComment(props.taskId, txt)
    draft.value = ''
    await nextTick()
    scrollBottom()
  } catch (e) {
    notify.error(e?.message || 'Не удалось отправить комментарий')
  } finally {
    sending.value = false
  }
}

function canEdit(c) {
  if (c.author_id === auth.user?.id) return true
  return isAtLeast(ROLES.MANAGER)
}

function startEdit(c) {
  editingId.value = c.id
  editText.value = c.text
}

function cancelEdit() {
  editingId.value = null
  editText.value = ''
}

async function saveEdit() {
  const txt = editText.value.trim()
  if (!txt) return
  try {
    await tasks.editComment(props.taskId, editingId.value, txt)
    cancelEdit()
  } catch (e) {
    notify.error(e?.message || 'Не удалось сохранить')
  }
}

function remove(c) {
  deletingId.value = c.id
}

async function confirmDelete() {
  const id = deletingId.value
  if (id == null) return
  deletingId.value = null
  try {
    await tasks.deleteComment(props.taskId, id)
  } catch (e) {
    notify.error(e?.message || 'Не удалось удалить')
  }
}

function avatarOf(a) {
  if (!a) return ''
  return a.avatar_path ? `/uploads/${a.avatar_path}` : `/api/users/${a.id}/identicon`
}

function fmtTime(d) {
  if (!d) return ''
  const dt = new Date(d)
  return dt.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit' })
}

function onKeydown(e) {
  // Cmd/Ctrl+Enter — отправить.
  if ((e.metaKey || e.ctrlKey) && e.key === 'Enter') {
    e.preventDefault()
    send()
  }
}

onMounted(load)
</script>

<template>
  <div class="comments">
    <div ref="listEl" class="comments-list">
      <div v-if="loading" class="comments-empty">
        <span class="material-symbols-outlined spinning">progress_activity</span>
        Загрузка…
      </div>
      <div v-else-if="!list.length" class="comments-empty">
        <span class="material-symbols-outlined">forum</span>
        Комментариев пока нет
      </div>
      <div v-for="c in list" :key="c.id" class="comment-item">
        <img :src="avatarOf(c.author)" class="comment-ava" :alt="c.author?.fio || ''" />
        <div class="comment-body">
          <div class="comment-head">
            <span class="comment-author">{{ c.author?.fio || 'Сотрудник' }}</span>
            <span class="comment-time" :title="new Date(c.created_at).toLocaleString('ru-RU')">
              {{ fmtTime(c.created_at) }}
              <span v-if="c.updated_at" class="comment-edited" title="отредактировано">·&nbsp;ред.</span>
            </span>
            <div class="comment-actions" v-if="canEdit(c) && editingId !== c.id">
              <button class="ca-btn" @click="startEdit(c)" title="Редактировать">
                <span class="material-symbols-outlined">edit</span>
              </button>
              <button class="ca-btn danger" @click="remove(c)" title="Удалить">
                <span class="material-symbols-outlined">delete</span>
              </button>
            </div>
          </div>
          <div v-if="editingId === c.id" class="comment-edit">
            <textarea v-model="editText" class="comment-textarea" rows="3" />
            <div class="edit-actions">
              <button class="btn-text" @click="cancelEdit">Отмена</button>
              <button class="btn-primary" @click="saveEdit">Сохранить</button>
            </div>
          </div>
          <MarkdownView v-else :source="c.text" class="comment-text" />
        </div>
      </div>
    </div>

    <ConfirmDialog
      :visible="deletingId != null"
      header="Удалить комментарий"
      message="Это действие нельзя отменить. Удалить комментарий?"
      confirm-label="Удалить"
      danger-confirm
      @confirm="confirmDelete"
      @cancel="deletingId = null"
    />

    <div class="comment-input">
      <textarea
        v-model="draft"
        class="comment-textarea"
        rows="2"
        placeholder="Написать комментарий… (Markdown поддерживается, Ctrl+Enter — отправить)"
        @keydown="onKeydown"
      />
      <button class="send-btn" :disabled="sending || !draft.trim()" @click="send">
        <span class="material-symbols-outlined">send</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.comments {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 0;
  flex: 1;
}

.comments-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding-right: 4px;
}

.comments-empty {
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: center;
  justify-content: center;
  padding: 28px 0;
  color: var(--color-on-surface-variant);
  font-size: 13px;
}
.comments-empty .material-symbols-outlined { font-size: 32px; opacity: 0.55; }

.comment-item {
  display: flex;
  gap: 10px;
  align-items: flex-start;
}
.comment-ava { width: 36px; height: 36px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.comment-body { flex: 1; min-width: 0; }
.comment-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 2px;
  flex-wrap: wrap;
}
.comment-author { font-weight: 650; color: var(--color-on-surface); font-size: 13px; }
.comment-time { font-size: 11px; color: var(--color-on-surface-variant); }
.comment-edited { font-style: italic; }

.comment-actions { margin-left: auto; display: inline-flex; gap: 2px; opacity: 0; transition: opacity 0.15s; }
.comment-item:hover .comment-actions { opacity: 1; }
.ca-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--color-on-surface-variant);
}
.ca-btn:hover { background: var(--color-surface-high); color: var(--color-on-surface); }
.ca-btn.danger:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.ca-btn .material-symbols-outlined { font-size: 16px; }

.comment-text {
  background: var(--color-surface-high);
  padding: 8px 12px;
  border-radius: var(--radius-md, 12px);
  font-size: 13px;
}

.comment-edit { display: flex; flex-direction: column; gap: 6px; }
.edit-actions { display: flex; gap: 6px; justify-content: flex-end; }

.comment-textarea {
  width: 100%;
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-variant);
  border-radius: var(--radius-md, 12px);
  padding: 8px 12px;
  font: inherit;
  font-size: 13px;
  color: var(--color-on-surface);
  resize: vertical;
  outline: none;
}
.comment-textarea:focus {
  border-color: var(--color-primary);
}

.comment-input {
  display: flex;
  gap: 8px;
  align-items: flex-end;
  flex-shrink: 0;
}
.send-btn {
  width: 40px;
  height: 40px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}
.send-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.send-btn .material-symbols-outlined { font-size: 20px; }

.btn-text {
  background: transparent;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  padding: 6px 12px;
  border-radius: var(--radius-full, 999px);
  font-weight: 600;
}
.btn-text:hover { background: color-mix(in oklab, var(--color-primary) 10%, transparent); }
.btn-primary {
  background: var(--color-primary);
  color: var(--color-on-primary);
  border: none;
  cursor: pointer;
  padding: 6px 14px;
  border-radius: var(--radius-full, 999px);
  font-weight: 600;
}

.spinning { animation: cspin 1s linear infinite; }
@keyframes cspin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
</style>
