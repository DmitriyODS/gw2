<template>
  <component
    :is="interactive ? 'button' : 'span'"
    class="chip"
    :class="[`tone-${tone}`, { interactive, selected, 'chip-sm': size === 'sm' }]"
    :type="interactive ? 'button' : undefined"
    :aria-pressed="interactive ? String(selected) : undefined"
    @click="interactive && $emit('click', $event)"
  >
    <span v-if="icon" class="material-symbols-outlined">{{ icon }}</span>
    <slot><span v-if="label">{{ label }}</span></slot>
    <strong v-if="count != null" class="chip-count">{{ count }}</strong>
    <button
      v-if="removable"
      class="chip-x"
      type="button"
      aria-label="Убрать"
      @click.stop="$emit('remove')"
    >
      <span class="material-symbols-outlined">close</span>
    </button>
  </component>
</template>

<script setup>
/* Тинтованная пилюля: счётчик, статус, метка, фильтр. Свела воедино `.chip-tint`,
   `.meta-stat` и три десятка scoped-копий по разделам.

   Это НЕ кнопка действия: у пилюли-действия («Оформление», «Разделы») тело
   стеклянное — для неё AppButton variant="glass" size="sm". */
defineProps({
  tone: {
    type: String,
    default: 'neutral',
    validator: (v) => ['neutral', 'primary', 'success', 'warning', 'error'].includes(v),
  },
  size: { type: String, default: 'md', validator: (v) => ['sm', 'md'].includes(v) },
  icon: { type: String, default: '' },
  label: { type: String, default: '' },
  /** Число справа — счётчики разделов («12 компаний»). */
  count: { type: [Number, String], default: null },
  /** Пилюля-фильтр: реагирует на клик и умеет быть выбранной. */
  interactive: { type: Boolean, default: false },
  selected: { type: Boolean, default: false },
  removable: { type: Boolean, default: false },
})

defineEmits(['click', 'remove'])
</script>

<style scoped>
/* Размеры — общие с остальным управлением (кнопка size-sm, вкладка): пилюля
   ростом ниже соседей читалась как чужой элемент. */
.chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 32px;
  padding: 6px 14px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.chip-sm { min-height: 26px; padding: 3px 10px; font-size: 12px; gap: 4px; }
.chip .material-symbols-outlined { font-size: 18px; flex-shrink: 0; }
.chip-sm .material-symbols-outlined { font-size: 16px; }
.chip-count { font-weight: 700; }

.tone-primary { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.tone-success { background: var(--color-success-container); color: var(--color-on-success-container); }
.tone-warning { background: var(--color-warning-container); color: var(--color-on-warning-container); }
.tone-error { background: var(--color-error-container); color: var(--color-on-error-container); }

/* Пилюля-фильтр: невыбранная — приглушённое стекло, выбранная — тинт своего
   тона с лёгкой тенью, как у активной вкладки. */
.chip.interactive {
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s, box-shadow 0.15s;
}

.chip.interactive.selected { box-shadow: var(--shadow-sm, none); }

.chip.interactive:not(.selected) {
  border-color: var(--acrylic-border);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
}

.chip.interactive:not(.selected):hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  color: var(--color-text);
}

/* Крестик снятия — круглый, размер задаём всеми тремя свойствами (мобильный
   `button { min-height: 36px }` иначе растянет его в овал). */
.chip-x {
  display: grid;
  place-items: center;
  width: 20px; min-width: 20px; max-width: 20px;
  height: 20px; min-height: 20px; max-height: 20px;
  margin-right: -4px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: color-mix(in oklch, currentColor 16%, transparent);
  color: inherit;
  cursor: pointer;
}

.chip-x .material-symbols-outlined { font-size: 14px; }
.chip-sm .chip-x {
  width: 16px; min-width: 16px; max-width: 16px;
  height: 16px; min-height: 16px; max-height: 16px;
}
.chip-sm .chip-x .material-symbols-outlined { font-size: 12px; }
</style>
