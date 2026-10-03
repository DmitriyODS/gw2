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

/* Физический тумблер: утопленная дорожка (колодец) и выпуклая рукоять,
   которая переезжает вправо. Включённая дорожка — колодец, залитый акцентом. */
.switch {
  position: relative;
  box-sizing: border-box;
  width: 46px; min-width: 46px; max-width: 46px;
  height: 26px; min-height: 26px; max-height: 26px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-full);
  background: var(--sk-well-bg);
  box-shadow: var(--sk-well-shadow);
  cursor: pointer;
  transition: background 0.18s, border-color 0.18s;
}

.switch.disabled { opacity: 0.5; cursor: not-allowed; }

.switch::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--sk-knob-bg);
  box-shadow: var(--sk-knob-shadow);
  transition: transform 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}

.switch.on { background: var(--color-primary); border-color: var(--color-primary); }
@supports (color: color-mix(in oklch, red, blue)) {
  .switch.on {
    border-color: color-mix(in oklch, var(--color-primary) 70%, black);
    background:
      var(--grain),
      linear-gradient(180deg,
        color-mix(in oklch, var(--color-primary) 85%, black),
        var(--color-primary) 70%);
    box-shadow:
      inset 0 2px 4px color-mix(in oklch, black 25%, transparent),
      0 1px 0 color-mix(in oklch, white 40%, transparent);
  }
}

.switch.on::after { transform: translateX(20px); }

.switch:active:not(.disabled)::after { box-shadow: var(--sk-knob-shadow), inset 0 0 0 20px color-mix(in oklch, black 4%, transparent); }
</style>
