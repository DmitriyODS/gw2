<template>
  <div class="phone-input" :class="{ invalid: showError, focused }">
    <span class="phone-prefix">+7</span>
    <input
      ref="inputEl"
      type="tel"
      inputmode="numeric"
      autocomplete="tel-national"
      class="phone-field"
      :placeholder="placeholder"
      :value="display"
      :disabled="disabled"
      maxlength="15"
      @beforeinput="onBeforeInput"
      @input="onInput"
      @keydown="onKeyDown"
      @paste="onPaste"
      @focus="focused = true"
      @blur="onBlur"
    />
    <button
      v-if="display && !disabled"
      type="button"
      class="phone-clear"
      @click="clear"
      aria-label="Очистить"
    >
      <span class="material-symbols-outlined">close</span>
    </button>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: '(___) ___-__-__' },
  disabled: { type: Boolean, default: false },
  invalid: { type: Boolean, default: false },
})
const emit = defineEmits(['update:modelValue', 'blur'])

const inputEl = ref(null)
const touched = ref(false)
const focused = ref(false)

const display = computed(() => format(props.modelValue))
const showError = computed(() => props.invalid && touched.value)

function digitsOnly(v) {
  return String(v ?? '').replace(/\D/g, '')
}

function normalizeRaw(raw) {
  let s = String(raw ?? '')
  // Срезаем канонический префикс «+7» до извлечения цифр, чтобы «7» кода страны
  // не попадала в набор цифр при частично набранном номере.
  if (s.startsWith('+7')) s = s.slice(2)
  let d = digitsOnly(s)
  if (!d) return ''
  // Дополнительно — если вставили «+7 999…» без нашего префикса или «8 999…».
  if (d.length === 11 && (d[0] === '7' || d[0] === '8')) d = d.slice(1)
  return d.slice(0, 10)
}

function toCanonical(raw) {
  const d = normalizeRaw(raw)
  return d ? `+7${d}` : ''
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

/* Запрещаем ввод нецифровых символов до того, как они попадут в input —
   так курсор не прыгает и в поле не мигают «лишние» буквы. Браузер уважает
   preventDefault на beforeinput.insertText / insertFromPaste / insertFromDrop. */
function onBeforeInput(ev) {
  if (!ev.data) return
  if (!/^\d+$/.test(ev.data)) {
    ev.preventDefault()
    return
  }
  // Если уже набрано 10 цифр — дальше не пускаем (можно только редактировать).
  const current = digitsOnly(ev.target.value)
  const selLen = (ev.target.selectionEnd ?? 0) - (ev.target.selectionStart ?? 0)
  if (current.length - selLen + ev.data.length > 10) {
    ev.preventDefault()
  }
}

function onInput(ev) {
  emit('update:modelValue', toCanonical(ev.target.value))
}

function onKeyDown(ev) {
  // Разрешаем служебные клавиши.
  if (
    ev.ctrlKey || ev.metaKey || ev.altKey ||
    ['Backspace', 'Delete', 'ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown',
      'Home', 'End', 'Tab', 'Enter', 'Escape'].includes(ev.key)
  ) return
  if (ev.key.length === 1 && !/^\d$/.test(ev.key)) {
    ev.preventDefault()
  }
}

function onPaste(ev) {
  const text = (ev.clipboardData || window.clipboardData)?.getData('text') || ''
  ev.preventDefault()
  emit('update:modelValue', toCanonical(text))
}

function onBlur() {
  touched.value = true
  focused.value = false
  emit('blur')
}

function clear() {
  emit('update:modelValue', '')
  inputEl.value?.focus()
}
</script>

<style scoped>
.phone-input {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px 0 14px;
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  border-radius: var(--radius-full);
  height: 44px;
  transition: border-color .15s, box-shadow .15s, background .15s;
  width: 100%;
}
.phone-input.focused,
.phone-input:focus-within {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 18%, transparent);
  background: var(--color-surface);
}
.phone-input.invalid {
  border-color: var(--color-error);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-error) 18%, transparent);
}

.phone-prefix {
  font-variant-numeric: tabular-nums;
  color: var(--color-text-dim);
  font-weight: 600;
  user-select: none;
}

.phone-field {
  flex: 1;
  border: none;
  outline: none;
  background: transparent;
  font: inherit;
  color: var(--color-text);
  font-variant-numeric: tabular-nums;
  letter-spacing: 0.01em;
  min-width: 0;
  padding: 0;
}
.phone-field::placeholder { color: var(--color-text-dim); opacity: .6; }

.phone-clear {
  display: inline-grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 50%;
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background .12s, color .12s;
}
.phone-clear:hover {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.phone-clear .material-symbols-outlined { font-size: 16px; }
</style>
