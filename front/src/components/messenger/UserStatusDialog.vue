<template>
  <AppDialog
    :model-value="modelValue"
    tone="primary"
    icon="mood"
    size="md"
    title="Мой статус"
    dialog-class="status-dialog"
    :actions="actions"
    :busy="saving"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="save"
  >
    <div class="status-emojis">
      <button
        v-for="e in EMOJIS"
        :key="e"
        type="button"
        class="status-emoji"
        :class="{ active: emoji === e }"
        @click="emoji = emoji === e ? '' : e"
      >{{ e }}</button>
    </div>
    <div class="status-field">
      <InputText
        v-model="text"
        class="w-full"
        placeholder="Чем занимаетесь? Например: в отпуске до 20-го"
        maxlength="80"
        @keydown.enter="save"
      />
      <span class="status-counter">{{ text.length }}/80</span>
    </div>
    <p class="status-hint">Статус видят коллеги в мессенджере — в шапке чата и вашем профиле.</p>
  </AppDialog>
</template>

<script setup>
import { ref, watch, computed } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import InputText from 'primevue/inputtext'
import { updateMe } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const EMOJIS = ['😊', '💼', '🎯', '☕', '📞', '🏠', '🏖️', '🤒', '🚀', '🌙']

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const auth = useAuthStore()
const notif = useNotificationsStore()

const emoji = ref('')
const text = ref('')
const saving = ref(false)

watch(() => props.modelValue, (v) => {
  if (!v) return
  emoji.value = auth.user?.status_emoji || ''
  text.value = auth.user?.status_text || ''
})

const hasCurrent = computed(() => !!(auth.user?.status_emoji || auth.user?.status_text))

const actions = computed(() => {
  const out = [{ kind: 'cancel', label: 'Отмена' }]
  if (hasCurrent.value) {
    out.push({ kind: 'neutral', label: 'Убрать', onClick: clearStatus })
  }
  out.push({ kind: 'confirm', label: 'Сохранить', icon: 'check' })
  return out
})

function clearStatus() {
  emoji.value = ''
  text.value = ''
  save()
}

async function save() {
  if (saving.value) return
  saving.value = true
  try {
    await updateMe({ status_emoji: emoji.value, status_text: text.value.trim() })
    await auth.loadMe()
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить статус')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.status-emojis {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 14px;
}

.status-emoji {
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  font-size: 22px;
  line-height: 1;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, transform 0.12s;
}

.status-emoji:hover { transform: scale(1.08); }

.status-emoji.active {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
}

.status-field { position: relative; }
.status-field :deep(.p-inputtext) { width: 100%; }

.status-counter {
  position: absolute;
  right: 10px;
  bottom: -18px;
  font-size: 11px;
  color: var(--color-text-dim);
}

.status-hint {
  margin: 26px 0 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
}

</style>
