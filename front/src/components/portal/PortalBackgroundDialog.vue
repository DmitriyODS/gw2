<template>
  <AppDialog
    :model-value="modelValue"
    tone="tertiary"
    icon="palette"
    size="lg"
    title="Оформление ленты"
    :actions="actions"
    :busy="saving"
    @update:model-value="$emit('update:modelValue', $event)"
    @confirm="apply"
  >
    <BackgroundEditor :recipe="recipe" :upload-fn="uploadFn" />

    <p class="pbg-hint">
      Оформление личное и синхронизируется на всех ваших устройствах — коллеги
      видят свой фон ленты.
    </p>
  </AppDialog>
</template>

<script setup>
import { ref, reactive, computed, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import BackgroundEditor from '@/components/common/BackgroundEditor.vue'
import { usePortalStore } from '@/stores/portal.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { uploadAttachment } from '@/api/messenger.js'
import { DEFAULT_RECIPE, normalizeRecipe, cloneRecipe } from '@/utils/chatBackgrounds.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue'])

const portal = usePortalStore()
const notif = useNotificationsStore()

const saving = ref(false)
const recipe = reactive(cloneRecipe(DEFAULT_RECIPE))

// Картинка-фон — личный ассет пользователя; грузим через общий uploads
// мессенджера (отдаётся тем же /uploads/, не требует привязки к посту).
const uploadFn = (file) => uploadAttachment(file)

function load() {
  const stored = normalizeRecipe(portal.background)
  Object.assign(recipe, cloneRecipe(stored || DEFAULT_RECIPE))
}

watch(() => props.modelValue, (open) => { if (open) load() })

const actions = computed(() => {
  const out = [{ kind: 'cancel', label: 'Отмена' }]
  if (portal.background) out.push({ kind: 'neutral', label: 'Сбросить', onClick: resetBg })
  out.push({ kind: 'confirm', label: 'Применить', icon: 'check' })
  return out
})

async function apply() {
  if (saving.value) return
  saving.value = true
  try {
    await portal.saveBackground(cloneRecipe(recipe))
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить оформление')
  } finally {
    saving.value = false
  }
}

async function resetBg() {
  if (saving.value) return
  saving.value = true
  try {
    await portal.resetBackground()
    emit('update:modelValue', false)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сбросить оформление')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.pbg-hint {
  margin: 18px 0 0;
  font-size: 12.5px;
  color: var(--color-text-dim);
}
</style>
