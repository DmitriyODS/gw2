<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="edit"
    size="md"
    :title="isEdit ? 'Редактировать пост' : 'Написать пост'"
    :busy="saving"
    :actions="[
      { kind: 'cancel', label: 'Отмена' },
      { kind: 'confirm', label: isEdit ? 'Сохранить' : 'Опубликовать', icon: 'send', disabled: !body.trim() },
    ]"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="submit"
  >
    <div class="composer-field">
      <label class="composer-label">Раздел</label>
      <Select
        v-model="topicId"
        :options="portal.topics"
        option-label="name"
        option-value="id"
        placeholder="Без раздела"
        class="w-full"
        show-clear
      />
    </div>

    <div class="composer-field">
      <input v-model="title" class="composer-input" placeholder="Заголовок (необязательно)" maxlength="200" />
    </div>

    <div class="composer-field">
      <textarea
        v-model="body"
        class="composer-textarea"
        rows="6"
        maxlength="10000"
        placeholder="О чём хотите рассказать команде?"
      />
    </div>

    <div
      class="composer-drop"
      :class="{ over: dragOver }"
      @dragover.prevent="dragOver = true"
      @dragleave.prevent="dragOver = false"
      @drop.prevent="onDrop"
    >
      <span class="material-symbols-outlined">attach_file</span>
      <span>Перетащите файлы сюда или</span>
      <button type="button" class="composer-browse" @click="fileInput?.click()">выберите</button>
      <input ref="fileInput" type="file" multiple hidden @change="onPick" />
    </div>

    <ul v-if="pendingFiles.length || existingAttachments.length" class="composer-files">
      <li v-for="(f, i) in pendingFiles" :key="'pending-' + i" class="composer-file">
        <span class="material-symbols-outlined">description</span>
        <span class="composer-file-name">{{ f.name }}</span>
        <button type="button" class="composer-file-remove" title="Убрать" @click="pendingFiles.splice(i, 1)">
          <span class="material-symbols-outlined">close</span>
        </button>
      </li>
      <li v-for="a in existingAttachments" :key="a.id" class="composer-file existing">
        <span class="material-symbols-outlined">description</span>
        <span class="composer-file-name">{{ a.name }}</span>
      </li>
    </ul>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import Select from 'primevue/select'
import AppDialog from '@/components/common/AppDialog.vue'
import { usePortalStore } from '@/stores/portal.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  post: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue', 'saved'])

const portal = usePortalStore()
const isEdit = computed(() => !!props.post)

const topicId = ref(null)
const title = ref('')
const body = ref('')
const pendingFiles = ref([])
const existingAttachments = ref([])
const dragOver = ref(false)
const saving = ref(false)
const fileInput = ref(null)

function reset() {
  topicId.value = props.post?.topic_id ?? null
  title.value = props.post?.title ?? ''
  body.value = props.post?.body ?? ''
  pendingFiles.value = []
  existingAttachments.value = props.post?.attachments ?? []
}

watch(() => props.modelValue, (v) => { if (v) reset() })

function onPick(e) {
  pendingFiles.value.push(...Array.from(e.target.files || []))
  e.target.value = ''
}

function onDrop(e) {
  dragOver.value = false
  pendingFiles.value.push(...Array.from(e.dataTransfer?.files || []))
}

async function submit() {
  const b = body.value.trim()
  if (!b) return
  saving.value = true
  try {
    const post = isEdit.value
      ? await portal.updatePost(props.post.id, { topicId: topicId.value, title: title.value, body: b })
      : await portal.createPost({ topicId: topicId.value, title: title.value, body: b })
    for (const f of pendingFiles.value) {
      await portal.uploadAttachment(post.id, f)
    }
    emit('update:modelValue', false)
    emit('saved', post)
  } catch (e) {
    useNotificationsStore().error(e?.message || 'Не удалось сохранить пост')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.composer-field {
  margin-bottom: 12px;
}

.composer-label {
  display: block;
  font-size: 12.5px;
  font-weight: 600;
  color: var(--color-text-dim);
  margin-bottom: 4px;
}

.composer-input,
.composer-textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  outline: none;
  box-sizing: border-box;
}
.composer-input:focus,
.composer-textarea:focus { border-color: var(--color-primary); }

.composer-textarea { resize: vertical; min-height: 100px; }

.composer-drop {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  padding: 14px;
  border: 1.5px dashed var(--color-outline-dim);
  border-radius: var(--radius-md);
  color: var(--color-text-dim);
  font-size: 13px;
  margin-bottom: 8px;
}
.composer-drop.over { border-color: var(--color-primary); background: var(--color-primary-container); }
.composer-drop .material-symbols-outlined { font-size: 20px; }

.composer-browse {
  border: none;
  background: transparent;
  color: var(--color-primary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
  padding: 0;
}

.composer-files {
  list-style: none;
  margin: 0 0 4px;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.composer-file {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 6px 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  font-size: 13px;
  color: var(--color-text);
}
.composer-file .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.composer-file-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.composer-file-remove {
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  display: grid;
  place-items: center;
}
.composer-file-remove:hover { color: var(--color-error); }
.composer-file-remove .material-symbols-outlined { font-size: 16px; }
</style>
