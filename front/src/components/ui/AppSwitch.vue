<template>
  <!-- Без подписи — голый тумблер (его подписывает AppSwitchRow); с подписью —
       компактная пара «текст + тумблер», по которой можно щёлкать целиком. -->
  <component
    :is="label ? 'label' : 'span'"
    class="switch-wrap"
    :class="{ bare: !label }"
  >
    <span v-if="label" class="switch-label">{{ label }}</span>
    <span
      class="switch"
      :class="{ on: modelValue, disabled }"
      role="switch"
      :aria-checked="String(modelValue)"
      :aria-disabled="disabled ? 'true' : undefined"
      @click="toggle"
    />
  </component>
</template>

<script setup>
/* Тумблер. Отдельно от строки: он нужен и в строке настройки (AppSwitchRow), и
   в тулбаре, и в карточке. */
const props = defineProps({
  modelValue: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  /** Компактная подпись рядом с тумблером (для тулбаров и рядов настроек). */
  label: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue'])

function toggle(e) {
  if (props.disabled) return
  e.stopPropagation()
  emit('update:modelValue', !props.modelValue)
}
</script>

<style scoped>
/* Обёртка ничего не занимает у голого тумблера: он остаётся ровно тем же
   элементом, что и раньше, — иначе поехали бы раскладки всех строк настроек. */
.switch-wrap.bare { display: contents; }

.switch-wrap:not(.bare) {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}

.switch-label { font-size: 13px; color: var(--color-text-dim); }

.switch {
  position: relative;
  box-sizing: border-box;
  width: 44px; min-width: 44px; max-width: 44px;
  height: 24px; min-height: 24px; max-height: 24px;
  border: 2px solid var(--color-outline, var(--color-outline-variant));
  border-radius: var(--radius-full);
  background: var(--color-surface-highest, var(--color-surface-high));
  cursor: pointer;
  transition: background 0.18s, border-color 0.18s;
}

.switch.disabled { opacity: 0.5; cursor: not-allowed; }

.switch::after {
  content: '';
  position: absolute;
  top: 50%;
  left: 4px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: var(--color-outline, var(--color-on-surface-variant));
  transform: translateY(-50%);
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1),
    background 0.2s, width 0.2s, height 0.2s, left 0.2s;
}

.switch.on {
  background: var(--color-primary);
  border-color: var(--color-primary);
}

.switch.on::after {
  width: 16px;
  height: 16px;
  left: 24px;
  background: var(--color-on-primary);
}
</style>
