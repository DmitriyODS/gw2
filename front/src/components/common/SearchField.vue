<template>
  <div class="search-field">
    <span class="material-symbols-outlined search-field-icon">search</span>
    <input
      ref="inputRef"
      class="search-field-input"
      :value="modelValue"
      :placeholder="placeholder"
      type="text"
      @input="$emit('update:modelValue', $event.target.value)"
      @keydown.esc="onEsc"
    />
    <button
      v-if="modelValue"
      class="search-field-clear"
      type="button"
      title="Очистить"
      aria-label="Очистить поиск"
      @click="clear"
    >
      <span class="material-symbols-outlined">close</span>
    </button>
    <slot v-else name="hint">
      <kbd v-if="kbdLabel" class="search-field-kbd">{{ kbdLabel }}</kbd>
    </slot>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: 'Поиск…' },
  /* Cmd/Ctrl+K фокусирует поле (слушатель регистрируется внутри). */
  hotkey: { type: Boolean, default: false },
  /* Текст kbd-хинта справа; пусто + hotkey → подпись по платформе. */
  kbd: { type: String, default: '' },
})

const emit = defineEmits(['update:modelValue', 'clear'])

const inputRef = ref(null)

const kbdLabel = computed(() => {
  if (props.kbd) return props.kbd
  if (!props.hotkey) return ''
  const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
  return isMac ? '⌘K' : 'Ctrl K'
})

function clear() {
  emit('update:modelValue', '')
  emit('clear')
  inputRef.value?.focus()
}

function onEsc() {
  if (props.modelValue) clear()
  else inputRef.value?.blur()
}

function onHotkey(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    inputRef.value?.focus()
  }
}

onMounted(() => {
  if (props.hotkey) window.addEventListener('keydown', onHotkey)
})
onUnmounted(() => {
  if (props.hotkey) window.removeEventListener('keydown', onHotkey)
})

defineExpose({ focus: () => inputRef.value?.focus() })
</script>

<style scoped>
/* Стеклянное поисковое поле-пилюля (паттерн экрана-списка, см. DESIGN.md). */
.search-field {
  flex: 1;
  min-width: 0;
  position: relative;
  display: flex;
  align-items: center;
}

.search-field-icon {
  position: absolute;
  left: 12px;
  font-size: 20px;
  color: var(--color-text-dim);
  pointer-events: none;
}

.search-field-input {
  width: 100%;
  padding: 10px 44px 10px 40px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  font: inherit;
  font-size: 14px;
  background: var(--acrylic-card-bg);
  color: var(--color-text);
  outline: none;
  transition: border-color 0.15s, background 0.15s, box-shadow 0.15s;
}

.search-field-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px color-mix(in oklch, var(--color-primary) 16%, transparent);
}

.search-field-input::placeholder {
  color: var(--color-text-dim);
}

.search-field-clear {
  position: absolute;
  right: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.search-field-clear:hover {
  background: var(--color-surface-highest);
  color: var(--color-text);
}

.search-field-clear .material-symbols-outlined {
  font-size: 18px;
}

.search-field-kbd {
  position: absolute;
  right: 10px;
  padding: 2px 7px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font-family: inherit;
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.4px;
  pointer-events: none;
}

/* На таче горячей клавиши нет — хинт лишний. */
@media (max-width: 768px) {
  .search-field-kbd { display: none; }

  .search-field-input {
    padding-top: 11px;
    padding-bottom: 11px;
  }
}
</style>
