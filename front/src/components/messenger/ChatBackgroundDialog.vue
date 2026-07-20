<template>
  <AppDialog
    :model-value="modelValue"
    tone="tertiary"
    icon="palette"
    size="lg"
    title="Оформление чата"
    dialog-class="chatbg-dialog"
    :actions="actions"
    :busy="saving"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="apply"
  >
    <!-- Область применения -->
    <div v-if="conversation" class="cbg-scope" role="tablist">
      <button
        type="button" class="cbg-scope-btn" :class="{ active: scope === 'chat' }"
        role="tab" @click="setScope('chat')"
      >Этот чат</button>
      <button
        type="button" class="cbg-scope-btn" :class="{ active: scope === 'all' }"
        role="tab" @click="setScope('all')"
      >Все чаты</button>
    </div>

    <BackgroundEditor :recipe="recipe" :upload-fn="uploadFn" />

    <p class="cbg-hint">
      Оформление личное и синхронизируется на всех ваших устройствах — собеседник
      видит свой фон.
    </p>
  </AppDialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'
import { DEFAULT_RECIPE, normalizeRecipe, cloneRecipe } from '@/utils/chatBackgrounds.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // Открытый чат (null — правка только общего дефолта, напр. из настроек).
  conversation: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])

const messenger = useMessengerStore()
const notif = useNotificationsStore()

const scope = ref('chat')
const saving = ref(false)
const recipe = reactive(cloneRecipe(DEFAULT_RECIPE))

const uploadFn = (file) => uploadAttachment(file)

// Эффективный рецепт выбранной области (для инициализации рабочей копии).
function storedForScope(s) {
  if (s === 'all') return messenger.chatBgDefault || null
  const cid = props.conversation?.id
  return (cid != null && messenger.chatBgByConv[cid]) || messenger.chatBgDefault || null
}

function loadScope(s) {
  const stored = normalizeRecipe(storedForScope(s))
  Object.assign(recipe, cloneRecipe(stored || DEFAULT_RECIPE))
}

function setScope(s) {
  scope.value = s
  loadScope(s)
}

watch(() => props.modelValue, (open) => {
  if (!open) return
  scope.value = props.conversation ? 'chat' : 'all'
  loadScope(scope.value)
})

const hasStored = computed(() => !!storedForScope(scope.value))

const actions = computed(() => {
  const out = [{ kind: 'cancel', label: 'Отмена' }]
  if (hasStored.value) out.push({ kind: 'neutral', label: 'Сбросить', onClick: resetScope })
  out.push({ kind: 'confirm', label: 'Применить', icon: 'check' })
  return out
})

async function apply() {
  if (saving.value) return
  saving.value = true
  try {
    const convId = scope.value === 'all' ? null : props.conversation?.id
    await messenger.saveChatBackground(convId ?? null, cloneRecipe(recipe))
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить оформление')
  } finally {
    saving.value = false
  }
}

async function resetScope() {
  if (saving.value) return
  saving.value = true
  try {
    const convId = scope.value === 'all' ? null : props.conversation?.id
    await messenger.resetChatBackground(convId ?? null)
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сбросить оформление')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.cbg-scope {
  display: flex;
  gap: 4px;
  padding: 4px;
  margin-bottom: 14px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
}

.cbg-scope-btn {
  flex: 1;
  border: none;
  background: transparent;
  color: var(--color-text-dim);
  font-size: 13.5px;
  font-weight: 600;
  padding: 8px 12px;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.cbg-scope-btn.active {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
}

.cbg-hint {
  margin: 18px 0 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
}
</style>
