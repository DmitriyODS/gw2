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
.chip {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 4px 11px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.chip-sm { padding: 2px 8px; font-size: 11px; }
.chip .material-symbols-outlined { font-size: 16px; flex-shrink: 0; }
.chip-count { font-weight: 700; }

.tone-primary { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.tone-success { background: var(--color-success-container); color: var(--color-on-success-container); }
.tone-warning { background: var(--color-warning-container); color: var(--color-on-warning-container); }
.tone-error { background: var(--color-error-container); color: var(--color-on-error-container); }

/* Пилюля-фильтр: невыбранная — приглушённое стекло, выбранная — тинт своего тона. */
.chip.interactive {
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}

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
  width: 18px; min-width: 18px; max-width: 18px;
  height: 18px; min-height: 18px; max-height: 18px;
  margin-right: -4px;
  padding: 0;
  border: none;
  border-radius: 50%;
  background: color-mix(in oklch, currentColor 16%, transparent);
  color: inherit;
  cursor: pointer;
}

.chip-x .material-symbols-outlined { font-size: 13px; }
</style>
