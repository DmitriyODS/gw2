<template>
  <div ref="root" class="inline-edit">
    <InputText
      ref="input"
      v-model="draft"
      class="inline-edit-field"
      :maxlength="maxlength"
      :placeholder="placeholder"
      @keydown.enter.stop.prevent="save"
      @keydown.esc.stop.prevent="cancel"
      @click.stop
    />

    <!-- Кнопки парят ПОД полем и привязаны к нему хвостиком: так видно, к какому
         именно пункту списка относится правка, и они не расталкивают строки. -->
    <div class="inline-edit-tools" :class="{ up: openUp }">
      <AppButton
        variant="icon"
        size="sm"
        tone="neutral"
        icon="close"
        title="Отменить"
        aria-label="Отменить"
        @click.stop="cancel"
      />
      <AppButton
        variant="icon"
        size="sm"
        tone="success"
        icon="check"
        title="Сохранить"
        aria-label="Сохранить"
        :disabled="!canSave"
        @click.stop="save"
      />
    </div>
  </div>
</template>

<script setup>
/* Правка названия прямо в списке: пункт превращается в поле ввода, а под ним
   парят подтверждение и отмена.

   Общий компонент ядра: так переименовывают реестр, и то же нужно папкам,
   заметкам и доскам. Своих запросов не делает — отдаёт новое значение наверх.

   Кнопки уходят ВВЕРХ, если снизу не хватает места: у нижних пунктов длинного
   списка они иначе оказались бы за краем панели. */
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import AppButton from './AppButton.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '' },
  maxlength: { type: Number, default: 120 },
  /** Пустое значение недопустимо (название реестра нельзя стереть). */
  required: { type: Boolean, default: true },
})
const emit = defineEmits(['save', 'cancel'])

const root = ref(null)
const input = ref(null)
const draft = ref(props.modelValue)
const openUp = ref(false)

const canSave = computed(() => !props.required || draft.value.trim().length > 0)

function save() {
  const value = draft.value.trim()
  if (!canSave.value) return
  if (value === props.modelValue) return cancel()
  emit('save', value)
}

function cancel() {
  emit('cancel')
}

// Клик мимо — это отказ от правки, а не молчаливое сохранение: иначе случайный
// промах менял бы название на недописанное.
function onDocPointerDown(e) {
  if (root.value && !root.value.contains(e.target)) cancel()
}

onMounted(async () => {
  await nextTick()
  const el = input.value?.$el || input.value
  el?.focus?.()
  el?.select?.()
  const rect = root.value?.getBoundingClientRect()
  // 92px — высота парящей панели с запасом на отступ.
  if (rect) openUp.value = window.innerHeight - rect.bottom < 92
  document.addEventListener('pointerdown', onDocPointerDown, true)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onDocPointerDown, true)
})
</script>

<style scoped>
.inline-edit {
  position: relative;
  width: 100%;
}

.inline-edit-field {
  width: 100%;
  font-size: 14px;
}

.inline-edit-tools {
  position: absolute;
  right: 0;
  z-index: 5;
  top: calc(100% + 6px);
  display: flex;
  gap: 6px;
  padding: 5px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), var(--acrylic-card-bg);
  -webkit-backdrop-filter: blur(12px);
  backdrop-filter: blur(12px);
  box-shadow: var(--shadow-md);
}

.inline-edit-tools.up { top: auto; bottom: calc(100% + 6px); }

/* Хвостик к полю: связь панели с правящимся пунктом должна читаться сразу. */
.inline-edit-tools::before {
  content: '';
  position: absolute;
  right: 14px;
  width: 8px;
  height: 8px;
  border: 1px solid var(--acrylic-border);
  background: var(--color-surface);
  transform: rotate(45deg);
}

.inline-edit-tools:not(.up)::before {
  top: -5px;
  border-width: 1px 0 0 1px;
}

.inline-edit-tools.up::before {
  bottom: -5px;
  border-width: 0 1px 1px 0;
}
</style>
