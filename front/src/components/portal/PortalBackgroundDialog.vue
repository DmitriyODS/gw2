<template>
  <AppDialog
    :model-value="modelValue"
    tone="tertiary"
    size="xl"
    title="Оформление ленты"
    :actions="actions"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="emit('update:modelValue', false)"
  >
    <BackgroundEditor :recipe="recipe" :upload-fn="uploadFn" @update:recipe="save" />

    <p class="pbg-hint">
      Оформление личное и синхронизируется на всех ваших устройствах — коллеги
      видят свой фон ленты.
    </p>
  </AppDialog>
</template>

<script setup>
import { computed, watch } from 'vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import { usePortalStore } from '@/stores/portal.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'
import { useRecipeUndo } from '@/composables/useRecipeUndo.js'
import { DEFAULT_RECIPE, normalizeRecipe } from '@/utils/chatBackgrounds.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const portal = usePortalStore()
const notif = useNotificationsStore()

// Картинка-фон — личный ассет пользователя; грузим через общий uploads
// мессенджера (отдаётся тем же /uploads/, не требует привязки к посту).
const uploadFn = (file) => uploadAttachment(file)

const recipe = computed(() => normalizeRecipe(portal.background) || DEFAULT_RECIPE)

/* Оформление применяется сразу — как темы и обои стола. */
function save(r) {
  portal.saveBackground(r)
    .catch((e) => notif.error(e?.message || 'Не удалось сохранить оформление'))
}

function reset() {
  portal.resetBackground()
    .catch((e) => notif.error(e?.message || 'Не удалось сбросить оформление'))
}

const undo = useRecipeUndo(() => portal.background, (r) => (r ? save(r) : reset()))

watch(() => props.modelValue, (open) => { if (open) undo.capture() })

const actions = computed(() => [
  { kind: 'neutral', label: 'Вернуть как было', icon: 'undo', disabled: !undo.changed.value, onClick: undo.undo },
  { kind: 'neutral', label: 'Сбросить', icon: 'restart_alt', disabled: !portal.background, onClick: reset },
  { kind: 'confirm', label: 'Готово', icon: 'check' },
])
</script>

<style scoped>
.pbg-hint {
  margin: 18px 0 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
}
</style>
