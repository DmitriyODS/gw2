<template>
  <AppDialog
    :model-value="modelValue"
    tone="tertiary"
    size="xl"
    title="Оформление чата"
    dialog-class="chatbg-dialog"
    :actions="actions"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="emit('update:modelValue', false)"
  >
    <!-- Область применения -->
    <AppTabs
      v-if="conversation"
      variant="tint"
      full-width
      class="cbg-scope"
      :model-value="scope"
      :tabs="SCOPES"
      @update:model-value="setScope"
    />

    <BackgroundEditor :recipe="recipe" :upload-fn="uploadFn" @update:recipe="save" />

    <p class="cbg-hint">
      Оформление личное и синхронизируется на всех ваших устройствах — собеседник
      видит свой фон.
    </p>
  </AppDialog>
</template>

<script setup>
import { ref, computed, watch } from 'vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'
import { useRecipeUndo } from '@/composables/useRecipeUndo.js'
import { DEFAULT_RECIPE, normalizeRecipe } from '@/utils/chatBackgrounds.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // Открытый чат (null — правка только общего дефолта, напр. из настроек).
  conversation: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])

const messenger = useMessengerStore()
const notif = useNotificationsStore()

const scope = ref('chat')
const SCOPES = [
  { value: 'chat', label: 'Этот чат' },
  { value: 'all', label: 'Все чаты' },
]
const uploadFn = (file) => uploadAttachment(file)

const convId = computed(() => (scope.value === 'all' ? null : props.conversation?.id ?? null))

// Собственная настройка области: у чата — его переопределение, у «всех» — дефолт.
const stored = computed(() => (convId.value == null
  ? messenger.chatBgDefault
  : messenger.chatBgByConv[convId.value]) || null)

// Что видно в области сейчас: своё, иначе общее, иначе заводское.
const recipe = computed(() => normalizeRecipe(stored.value || messenger.chatBgDefault) || DEFAULT_RECIPE)

/* Оформление применяется сразу — как темы и обои стола. */
function save(r) {
  messenger.saveChatBackground(convId.value, r)
    .catch((e) => notif.error(e?.message || 'Не удалось сохранить оформление'))
}

function reset() {
  messenger.resetChatBackground(convId.value)
    .catch((e) => notif.error(e?.message || 'Не удалось сбросить оформление'))
}

const undo = useRecipeUndo(() => stored.value, (r) => (r ? save(r) : reset()))

function setScope(s) {
  scope.value = s
  undo.capture()
}

watch(() => props.modelValue, (open) => {
  if (!open) return
  setScope(props.conversation ? 'chat' : 'all')
})

const actions = computed(() => [
  { kind: 'neutral', label: 'Вернуть как было', icon: 'undo', disabled: !undo.changed.value, onClick: undo.undo },
  { kind: 'neutral', label: 'Сбросить', icon: 'restart_alt', disabled: !stored.value, onClick: reset },
  { kind: 'confirm', label: 'Готово', icon: 'check' },
])
</script>

<style scoped>
.cbg-scope { margin-bottom: 14px; }

.cbg-hint {
  margin: 18px 0 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
}
</style>
