<template>
  <div class="phone-input" :class="{ invalid: showError }">
    <span class="phone-prefix">+7</span>
    <input
      ref="inputEl"
      type="tel"
      inputmode="numeric"
      autocomplete="tel"
      class="phone-field"
      :placeholder="placeholder"
      :value="display"
      :disabled="disabled"
      @input="onInput"
      @blur="onBlur"
    />
    <button v-if="display && !disabled" type="button"
            class="phone-clear" @click="clear" aria-label="Очистить">
      <span class="material-symbols-outlined">close</span>
    </button>
  </div>
</template>

<script setup>
import { ref, computed, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '(___) ___-__-__' },
  disabled: { type: Boolean, default: false },
  invalid: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'blur'])

const inputEl = ref(null)
const touched = ref(false)

const display = computed(() => format(props.modelValue))
const showError = computed(() => props.invalid && touched.value)

function digitsOnly(v) {
  return String(v || '').replace(/\D/g, '')
}

// Принимает любой ввод и нормализует в 10 цифр (без ведущей 7/8).
function normalizeRaw(raw) {
  let d = digitsOnly(raw)
  if (!d) return ''
  if (d.length === 11 && (d[0] === '7' || d[0] === '8')) d = d.slice(1)
  return d.slice(0, 10)
}

// Возвращает E.164: +7XXXXXXXXXX (или '' если нет 10 цифр).
function toCanonical(raw) {
  const d = normalizeRaw(raw)
  return d.length === 10 ? `+7${d}` : d ? `+7${d}` : ''
}

function format(canonical) {
  const d = normalizeRaw(canonical)
  if (!d) return ''
  const a = d.slice(0, 3)
  const b = d.slice(3, 6)
  const c = d.slice(6, 8)
  const e = d.slice(8, 10)
  let out = `(${a}`
  if (d.length >= 3) out += ')'
  if (b) out += ` ${b}`
  if (c) out += `-${c}`
  if (e) out += `-${e}`
  return out
}

function onInput(ev) {
  const canonical = toCanonical(ev.target.value)
  emit('update:modelValue', canonical)
}

function onBlur() {
  touched.value = true
  emit('blur')
}

function clear() {
  emit('update:modelValue', '')
  inputEl.value?.focus()
}

watch(() => props.modelValue, () => {})
</script>

<style scoped>
.phone-input {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px 0 12px;
  border: 1px solid var(--color-outline-variant);
  background: var(--color-surface);
  border-radius: var(--radius-md, 12px);
  height: 44px;
  transition: border-color .15s, box-shadow .15s;
  width: 100%;
}
.phone-input:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklab, var(--color-primary) 18%, transparent);
}
.phone-input.invalid { border-color: var(--color-error); }

.phone-prefix {
  font-variant-numeric: tabular-nums;
  color: var(--color-on-surface-variant);
  font-weight: 500;
}

.phone-field {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font: inherit;
  color: var(--color-on-surface);
  font-variant-numeric: tabular-nums;
  min-width: 0;
}
.phone-field::placeholder { color: var(--color-on-surface-variant); opacity: .7; }

.phone-clear {
  display: inline-grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 50%;
  background: var(--color-surface-container);
  color: var(--color-on-surface-variant);
  cursor: pointer;
}
.phone-clear:hover { background: var(--color-surface-high); }
.phone-clear .material-symbols-outlined { font-size: 16px; }
</style>
