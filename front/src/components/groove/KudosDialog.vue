<template>
  <AppDialog
    :model-value="modelValue"
    title="Поблагодарить коллегу"
    subtitle="Кудос появится в ленте — пусть все видят, кто молодец"
    icon="volunteer_activism"
    tone="success"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="kudos-search">
      <span class="material-symbols-outlined">search</span>
      <input v-model.trim="query" placeholder="Найти коллегу" />
    </div>

    <div class="kudos-list">
      <button
        v-for="u in filtered"
        :key="u.id"
        class="kudos-user"
        :class="{ selected: selected?.id === u.id }"
        type="button"
        @click="selected = u"
      >
        <img class="kudos-avatar" :src="avatarUrl(u)" :alt="u.fio" />
        <span class="kudos-fio">{{ u.fio }}</span>
        <span v-if="selected?.id === u.id" class="material-symbols-outlined kudos-check">check_circle</span>
      </button>
      <p v-if="!filtered.length" class="kudos-empty">Никого не нашлось</p>
    </div>

    <textarea
      v-model.trim="text"
      class="kudos-text"
      rows="3"
      maxlength="500"
      placeholder="За что благодарите? Например: «Спасибо за помощь с релизом!»"
    ></textarea>

    <template #footer>
      <div class="kudos-footer">
        <button class="kudos-cancel" type="button" @click="$emit('update:modelValue', false)">Отмена</button>
        <button
          class="kudos-send"
          type="button"
          :disabled="!selected || !text || sending"
          @click="send"
        >
          <span class="material-symbols-outlined">volunteer_activism</span>
          Отправить кудос
        </button>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { getDirectory } from '@/api/users.js'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { avatarUrl } from '@/utils/groove.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const groove = useGrooveStore()
const notify = useNotificationsStore()

const users = ref([])
const query = ref('')
const selected = ref(null)
const text = ref('')
const sending = ref(false)

watch(() => props.modelValue, async (open) => {
  if (!open) return
  selected.value = null
  text.value = ''
  query.value = ''
  try {
    users.value = await getDirectory('', true)
  } catch {
    users.value = []
  }
})

const filtered = computed(() => {
  const q = query.value.toLowerCase()
  if (!q) return users.value
  return users.value.filter(u => u.fio?.toLowerCase().includes(q))
})

async function send() {
  sending.value = true
  try {
    await groove.sendKudos(selected.value.id, text.value)
    notify.success(`Кудос для ${selected.value.fio} улетел в ленту 💚`)
    emit('update:modelValue', false)
  } catch (e) {
    notify.error(e?.message || 'Не удалось отправить')
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.kudos-search {
  display: flex;
  align-items: center;
  gap: 8px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  padding: 7px 14px;
  margin-bottom: 10px;
}
.kudos-search .material-symbols-outlined {
  font-size: 18px;
  color: var(--color-text-dim);
}
.kudos-search input {
  border: none;
  outline: none;
  background: none;
  flex: 1;
  min-width: 0;
  font-size: 14px;
  color: var(--color-text);
}
.kudos-list {
  max-height: 220px;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 2px;
  margin-bottom: 12px;
}
.kudos-user {
  display: flex;
  align-items: center;
  gap: 10px;
  border: none;
  background: none;
  padding: 7px 10px;
  border-radius: 12px;
  cursor: pointer;
  text-align: left;
}
.kudos-user:hover { background: var(--color-surface-high); }
.kudos-user.selected { background: var(--color-primary-container); }
.kudos-avatar { width: 32px; height: 32px; border-radius: 50%; object-fit: cover; }
.kudos-fio { flex: 1; font-size: 14px; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kudos-check { color: var(--color-primary); font-size: 20px; }
.kudos-empty { font-size: 13px; color: var(--color-text-dim); text-align: center; margin: 12px 0; }
.kudos-text {
  width: 100%;
  box-sizing: border-box;
  border: 1px solid var(--color-outline-dim);
  border-radius: 14px;
  padding: 10px 14px;
  font-size: 14px;
  font-family: inherit;
  resize: vertical;
  background: var(--color-surface);
  color: var(--color-text);
  outline: none;
}
.kudos-text:focus { border-color: var(--color-primary); }
.kudos-footer {
  display: flex;
  justify-content: flex-end;
  gap: 10px;
  width: 100%;
}
.kudos-cancel {
  border: none;
  background: none;
  color: var(--color-primary);
  font-size: 14px;
  font-weight: 600;
  padding: 10px 16px;
  border-radius: var(--radius-full);
  cursor: pointer;
}
.kudos-cancel:hover { background: var(--color-surface-high); }
.kudos-send {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  padding: 10px 20px;
  cursor: pointer;
}
.kudos-send:disabled { opacity: 0.45; cursor: default; }
.kudos-send .material-symbols-outlined { font-size: 18px; }
</style>
