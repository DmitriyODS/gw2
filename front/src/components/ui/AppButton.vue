<template>
  <component
    :is="tag"
    class="btn"
    :class="[`v-${variant}`, `tone-${tone}`, `size-${size}`, { block, 'icon-only': iconOnly, busy: loading }]"
    :type="tag === 'button' ? type : undefined"
    :disabled="tag === 'button' && (disabled || loading) ? true : undefined"
    :aria-busy="loading ? 'true' : undefined"
    :aria-label="iconOnly ? (ariaLabel || label) : ariaLabel || undefined"
    @click="onClick"
  >
    <span v-if="loading" class="btn-spin" aria-hidden="true" />
    <span v-else-if="icon" class="material-symbols-outlined">{{ icon }}</span>
    <span v-if="!iconOnly" class="btn-label"><slot>{{ label }}</slot></span>
    <span v-if="trailingIcon && !iconOnly" class="material-symbols-outlined btn-trail">{{ trailingIcon }}</span>
  </component>
</template>

<script setup>
/* Единственная кнопка платформы. Заменила разошедшиеся `.btn-grad`,
   `.btn-glass`, `.gw-chip`, `.icon-btn` и их scoped-копии по разделам —
   те приводились к общему виду слоем `!important` в main.css.

   Варианты: filled — главное действие экрана (градиент), glass — второстепенное
   (стекло), text — третьестепенное (без тела), icon — круглая иконочная. */
import { computed, useSlots } from 'vue'

const props = defineProps({
  variant: {
    type: String,
    default: 'glass',
    validator: (v) => ['filled', 'glass', 'text', 'icon'].includes(v),
  },
  tone: {
    type: String,
    default: 'primary',
    validator: (v) => ['primary', 'danger', 'success', 'neutral'].includes(v),
  },
  size: { type: String, default: 'md', validator: (v) => ['sm', 'md', 'lg'].includes(v) },
  icon: { type: String, default: '' },
  trailingIcon: { type: String, default: '' },
  label: { type: String, default: '' },
  loading: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  /** Растянуть на всю ширину — узкие строки настроек, подвалы модалок. */
  block: { type: Boolean, default: false },
  type: { type: String, default: 'button' },
  /** Отрисовать как <a>/<router-link>-подобный тег (для ссылок-кнопок). */
  tag: { type: String, default: 'button' },
  ariaLabel: { type: String, default: '' },
})

const emit = defineEmits(['click'])
const slots = useSlots()

const iconOnly = computed(() => props.variant === 'icon' || (!props.label && !slots.default))

function onClick(e) {
  if (props.disabled || props.loading) return
  emit('click', e)
}
</script>

<style scoped>
.btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  font: inherit;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, filter 0.15s, box-shadow 0.15s;
}

.btn:disabled { cursor: not-allowed; opacity: 0.55; }
.btn.busy { cursor: progress; }
.btn.block { width: 100%; }
.btn .material-symbols-outlined { font-size: 20px; }
.btn-label { min-width: 0; overflow: hidden; text-overflow: ellipsis; }
.btn-trail { margin-left: 2px; color: inherit; opacity: 0.8; }

/* ── Размеры ── */
.size-sm { padding: 7px 14px; font-size: 12.5px; }
.size-sm .material-symbols-outlined { font-size: 18px; }
.size-md { padding: 10px 18px; font-size: 14px; }
.size-lg { padding: 12px 24px; font-size: 15px; }

/* Круглая иконочная: ширину и высоту задаём жёстко всеми тремя свойствами —
   глобальный мобильный `button { min-height: 36px }` иначе растянет её в овал. */
.v-icon,
.icon-only {
  gap: 0;
  padding: 0;
  width: 40px; min-width: 40px; max-width: 40px;
  height: 40px; min-height: 40px; max-height: 40px;
}

.size-sm.v-icon, .size-sm.icon-only {
  width: 32px; min-width: 32px; max-width: 32px;
  height: 32px; min-height: 32px; max-height: 32px;
}

.size-lg.v-icon, .size-lg.icon-only {
  width: 46px; min-width: 46px; max-width: 46px;
  height: 46px; min-height: 46px; max-height: 46px;
}

/* ── filled: главное действие, фирменный градиент ── */
.v-filled {
  background: var(--grad-primary);
  color: var(--color-on-primary);
  box-shadow: var(--shadow-sm);
}
.v-filled:hover:not(:disabled) { filter: brightness(1.06); box-shadow: var(--shadow-md); }
.v-filled.tone-danger { background: var(--color-error); color: var(--color-on-error); }
.v-filled.tone-success { background: var(--color-success); color: var(--color-on-success); }
.v-filled.tone-neutral { background: var(--color-surface-high); color: var(--color-text); }

/* ── glass / icon: стеклянное тело с бликом по кромке ── */
.v-glass,
.v-icon {
  border-color: var(--acrylic-border);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
}

.v-glass:hover:not(:disabled),
.v-icon:hover:not(:disabled) {
  background: var(--glass-bg), color-mix(in oklch, var(--color-primary) 12%, transparent);
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
}

.v-glass.tone-danger,
.v-icon.tone-danger {
  color: var(--color-error);
  border-color: color-mix(in oklch, var(--color-error) 30%, var(--color-outline-dim));
}

.v-glass.tone-danger:hover:not(:disabled),
.v-icon.tone-danger:hover:not(:disabled) {
  background: var(--glass-bg), color-mix(in oklch, var(--color-error) 12%, transparent);
  border-color: color-mix(in oklch, var(--color-error) 45%, var(--acrylic-border));
}

.v-glass.tone-success {
  background: var(--color-success-container);
  border-color: transparent;
  color: var(--color-on-success-container);
}

.v-glass.tone-success:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-success) 26%, var(--color-success-container));
}

/* ── text: без тела, только подпись ── */
.v-text {
  background: none;
  color: var(--color-text-dim);
}
.v-text:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-primary) 10%, transparent);
  color: var(--color-text);
}
.v-text.tone-danger { color: var(--color-error); }
.v-text.tone-danger:hover:not(:disabled) {
  background: color-mix(in oklch, var(--color-error) 10%, transparent);
}

/* ── Индикатор загрузки: кольцо в размер иконки ── */
.btn-spin {
  width: 18px;
  height: 18px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  opacity: 0.85;
  animation: btn-spin 0.7s linear infinite;
}

@keyframes btn-spin { to { transform: rotate(360deg); } }

@media (prefers-reduced-motion: reduce) {
  .btn-spin { animation-duration: 2s; }
}
</style>
