<template>
  <AppField :label="label" :hint="hint">
    <template #default="{ id }">
      <div class="af" :class="{ center }">
        <InputText
          :id="id"
          :model-value="modelValue"
          class="af-input"
          :style="toolsWidth ? { paddingRight: `${toolsWidth + 12}px` } : undefined"
          :type="inputType"
          :placeholder="placeholder"
          :disabled="disabled"
          :invalid="invalid"
          :autocomplete="autocomplete"
          :inputmode="inputmode || undefined"
          :maxlength="maxlength || undefined"
          @update:model-value="$emit('update:modelValue', $event ?? '')"
          @keyup.enter="$emit('enter')"
        />
        <span v-if="$slots.tools || type === 'password'" ref="toolsEl" class="af-tools">
          <slot name="tools" />
          <AppButton
            v-if="type === 'password'"
            variant="text"
            size="sm"
            tabindex="-1"
            :icon="revealed ? 'visibility_off' : 'visibility'"
            :aria-label="revealed ? 'Скрыть пароль' : 'Показать пароль'"
            :title="revealed ? 'Скрыть пароль' : 'Показать пароль'"
            @click="revealed = !revealed"
          />
        </span>
      </div>
    </template>
  </AppField>
</template>

<script setup>
/* Поле форм входа — поле ядра (AppField + InputText) с кнопками внутри справа
   (показать пароль, сгенерировать, скопировать). Кнопки в слот `tools` кладутся
   готовыми AppButton variant="text" size="sm". */
import { computed, onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import AppField from '@/components/ui/AppField.vue'
import AppButton from '@/components/ui/AppButton.vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  label: { type: String, default: '' },
  type: { type: String, default: 'text' },
  placeholder: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  autocomplete: { type: String, default: 'off' },
  inputmode: { type: String, default: '' },
  maxlength: { type: Number, default: 0 },
  hint: { type: String, default: '' },
  invalid: { type: Boolean, default: false },
  // Крупный центрированный ввод — для кода подтверждения.
  center: { type: Boolean, default: false },
})

defineEmits(['update:modelValue', 'enter'])

const revealed = ref(false)
const inputType = computed(() => (props.type === 'password' && revealed.value ? 'text' : props.type))

// Текст не должен уходить под кнопки: правый отступ поля — по их ширине.
// Набор кнопок у поля постоянный, поэтому достаточно одного замера.
const toolsEl = ref(null)
const toolsWidth = ref(0)
onMounted(() => { toolsWidth.value = toolsEl.value?.offsetWidth || 0 })
</script>

<style scoped>
.af {
  position: relative;
  display: flex;
  align-items: center;
}

.af-input {
  width: 100%;
  height: 46px;
  padding: 0 14px;
  font-size: 15px;
}

.center .af-input {
  height: 56px;
  text-align: center;
  letter-spacing: 0.42em;
  text-indent: 0.42em;
  font-size: 22px;
  font-weight: 600;
}

.af-tools {
  position: absolute;
  right: 6px;
  display: flex;
  align-items: center;
  gap: 2px;
}
</style>
