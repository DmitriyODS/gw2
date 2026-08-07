<template>
  <label class="af" :class="{ 'is-invalid': invalid }">
    <span v-if="label" class="af-label">{{ label }}</span>
    <span class="af-box">
      <input
        class="af-input"
        :class="{ center }"
        :type="inputType"
        :value="modelValue"
        :placeholder="placeholder"
        :disabled="disabled"
        :autocomplete="autocomplete"
        :inputmode="inputmode || undefined"
        :maxlength="maxlength || undefined"
        @input="$emit('update:modelValue', $event.target.value)"
        @keyup.enter="$emit('enter')"
      />
      <span class="af-tools">
        <slot name="tools" />
        <button
          v-if="type === 'password'"
          type="button"
          class="af-tool"
          tabindex="-1"
          :title="revealed ? 'Скрыть пароль' : 'Показать пароль'"
          @click="revealed = !revealed"
        >
          <span class="material-symbols-outlined">{{ revealed ? 'visibility_off' : 'visibility' }}</span>
        </button>
      </span>
    </span>
    <span v-if="hint" class="af-hint">{{ hint }}</span>
  </label>
</template>

<script setup>
import { computed, ref } from 'vue'

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
</script>

<style scoped>
.af {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}

.af-label {
  font-size: 12.5px;
  font-weight: 600;
  color: var(--color-text-dim);
}

.af-box {
  position: relative;
  display: flex;
  align-items: center;
}

.af-input {
  width: 100%;
  height: 44px;
  box-sizing: border-box;
  padding: 0 14px;
  border-radius: var(--radius-md);
  border: 1px solid color-mix(in oklch, var(--color-outline) 42%, transparent);
  background: color-mix(in oklch, var(--color-surface) 72%, transparent);
  color: var(--color-text);
  font: inherit;
  font-size: 15px;
  outline: none;
  transition: border-color 0.15s, box-shadow 0.15s, background 0.15s;
}

.af-input::placeholder { color: var(--color-text-dim); opacity: 0.75; }

.af-input:focus {
  border-color: var(--color-primary);
  background: color-mix(in oklch, var(--color-surface) 92%, transparent);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 16%, transparent);
}

.af-input:disabled { opacity: 0.55; cursor: not-allowed; }

.af-input.center {
  text-align: center;
  letter-spacing: 0.42em;
  font-size: 22px;
  font-weight: 600;
  padding-right: 0;
  text-indent: 0.42em;
}

.is-invalid .af-input {
  border-color: color-mix(in oklch, var(--color-error) 55%, transparent);
}

/* Кнопки внутри поля (глаз, сгенерировать, скопировать) */
.af-tools {
  position: absolute;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 2px;
}

.af-tools :deep(.af-tool),
.af-tool {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  min-width: 30px;
  min-height: 30px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: none;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: color 0.15s, background 0.15s;
}

.af-tools :deep(.af-tool):hover,
.af-tool:hover {
  color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 12%, transparent);
}

.af-tools :deep(.af-tool) .material-symbols-outlined,
.af-tool .material-symbols-outlined { font-size: 19px; }

.af-hint {
  font-size: 12px;
  color: var(--color-text-dim);
}
</style>
