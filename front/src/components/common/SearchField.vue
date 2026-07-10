<template>
  <div ref="wrapRef" class="search-field" :class="{ 'is-collapsed': collapsedMode && !expanded }">
    <!-- Мобайл: свёрнутый поиск — круглая кнопка-лупа; точка — активный запрос. -->
    <button
      v-if="collapsedMode"
      class="search-field-toggle"
      type="button"
      title="Поиск"
      aria-label="Поиск"
      @click="expand"
    >
      <span class="material-symbols-outlined">search</span>
      <span v-if="modelValue" class="search-field-dot" aria-hidden="true" />
    </button>

    <template v-if="!collapsedMode">
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
    </template>

    <!-- Мобайл: развёрнутый поиск плавает поверх остальных элементов шапки. -->
    <Teleport to="body">
      <Transition name="sf-float">
        <div
          v-if="collapsedMode && expanded"
          ref="floatRef"
          class="search-field-float"
          :style="floatStyle"
        >
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
            class="search-field-clear always"
            type="button"
            :title="modelValue ? 'Очистить' : 'Закрыть'"
            :aria-label="modelValue ? 'Очистить поиск' : 'Закрыть поиск'"
            @click="modelValue ? clear() : collapse()"
          >
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, nextTick, onMounted, onUnmounted } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  placeholder: { type: String, default: 'Поиск…' },
  /* Cmd/Ctrl+K фокусирует поле (слушатель регистрируется внутри). */
  hotkey: { type: Boolean, default: false },
  /* Текст kbd-хинта справа; пусто + hotkey → подпись по платформе. */
  kbd: { type: String, default: '' },
  /* На мобильном поле сворачивается в кнопку-лупу; тап разворачивает
     плавающее поле поверх шапки. Отключается для мест, где поиск
     должен быть виден всегда. */
  collapsible: { type: Boolean, default: true },
})

const emit = defineEmits(['update:modelValue', 'clear'])

const inputRef = ref(null)
const wrapRef = ref(null)
const floatRef = ref(null)

/* ── Мобильный свёрнутый режим ── */
const isMobileView = ref(false)
const expanded = ref(false)
const floatStyle = ref({})
let mq = null

const collapsedMode = computed(() => props.collapsible && isMobileView.value)

function onMqChange(e) {
  isMobileView.value = e.matches
  if (!e.matches) collapse()
}

async function expand() {
  // Плавающее поле встаёт по вертикали на уровне кнопки-лупы.
  const rect = wrapRef.value?.getBoundingClientRect()
  floatStyle.value = { top: `${Math.max(8, Math.round(rect?.top ?? 12))}px` }
  expanded.value = true
  await nextTick()
  inputRef.value?.focus()
  document.addEventListener('pointerdown', onDocPointerDown, true)
}

function collapse() {
  if (!expanded.value) return
  expanded.value = false
  document.removeEventListener('pointerdown', onDocPointerDown, true)
}

function onDocPointerDown(e) {
  if (floatRef.value && !floatRef.value.contains(e.target)) collapse()
}

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
  else if (expanded.value) collapse()
  else inputRef.value?.blur()
}

function onHotkey(e) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault()
    focusSearch()
  }
}

function focusSearch() {
  if (collapsedMode.value && !expanded.value) expand()
  else inputRef.value?.focus()
}

onMounted(() => {
  if (props.hotkey) window.addEventListener('keydown', onHotkey)
  mq = window.matchMedia('(max-width: 768px)')
  isMobileView.value = mq.matches
  mq.addEventListener('change', onMqChange)
})
onUnmounted(() => {
  if (props.hotkey) window.removeEventListener('keydown', onHotkey)
  mq?.removeEventListener('change', onMqChange)
  document.removeEventListener('pointerdown', onDocPointerDown, true)
})

defineExpose({ focus: focusSearch })
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

/* Свёрнутый режим: контейнер сжимается до кнопки-лупы. */
.search-field.is-collapsed {
  flex: 0 0 auto;
}

.search-field-toggle {
  position: relative;
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--acrylic-card-bg);
  color: var(--color-text-dim);
  cursor: pointer;
}

.search-field-toggle .material-symbols-outlined { font-size: 20px; }

.search-field-dot {
  position: absolute;
  top: 6px;
  right: 6px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
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
  height: 26px; min-height: 0;
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

<style>
/* Развёрнутый мобильный поиск (Teleport в body — без scoped): плавает поверх
   шапки раздела на уровне кнопки-лупы. */
.search-field-float {
  position: fixed;
  left: 10px;
  right: 10px;
  z-index: 5000;
  display: flex;
  align-items: center;
}

.search-field-float .search-field-icon {
  position: absolute;
  left: 12px;
  font-size: 20px;
  color: var(--color-text-dim);
  pointer-events: none;
  /* backdrop-filter инпута создаёт stacking context — без z-index он
     перекрывает absolutely-позиционированные иконку и крестик. */
  z-index: 1;
}

.search-field-float .search-field-input {
  width: 100%;
  padding: 11px 44px 11px 40px;
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-full);
  font: inherit;
  font-size: 14px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  color: var(--color-text);
  outline: none;
  box-shadow: var(--shadow-lg);
}

.search-field-float .search-field-clear.always {
  position: absolute;
  right: 8px;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px; min-height: 0;
  border: none;
  border-radius: 50%;
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
}

.search-field-float .search-field-clear .material-symbols-outlined { font-size: 20px; }

.sf-float-enter-active, .sf-float-leave-active { transition: opacity 0.15s, transform 0.15s; }
.sf-float-enter-from, .sf-float-leave-to { opacity: 0; transform: translateY(-6px) scale(0.98); }
</style>
