<template>
  <!-- Панель поиска по открытой заметке. Совпадения считает и подсвечивает
       composables/useNoteFind.js — здесь только ввод, счёт и переходы
       (Enter — вперёд, Shift+Enter — назад). -->
  <div class="nf-bar">
    <span class="material-symbols-outlined nf-bar-ic">search</span>
    <input
      ref="inputEl"
      :value="query"
      class="nf-bar-input"
      type="text"
      placeholder="Найти в заметке"
      autocomplete="off"
      spellcheck="false"
      @input="$emit('update:query', $event.target.value)"
      @keydown.enter.prevent="$emit('step', $event.shiftKey ? -1 : 1)"
    />
    <span class="nf-bar-count">{{ countLabel }}</span>
    <button class="nf-bar-btn" title="Предыдущее (Shift+Enter)" :disabled="!total" @click="$emit('step', -1)">
      <span class="material-symbols-outlined">keyboard_arrow_up</span>
    </button>
    <button class="nf-bar-btn" title="Следующее (Enter)" :disabled="!total" @click="$emit('step', 1)">
      <span class="material-symbols-outlined">keyboard_arrow_down</span>
    </button>
    <button class="nf-bar-btn" title="Закрыть (Esc)" @click="$emit('close')">
      <span class="material-symbols-outlined">close</span>
    </button>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'

const props = defineProps({
  query: { type: String, default: '' },
  total: { type: Number, default: 0 },
  current: { type: Number, default: 0 },
})

defineEmits(['update:query', 'step', 'close'])

const inputEl = ref(null)

const countLabel = computed(() => {
  if (props.total) return `${props.current}/${props.total}`
  return props.query.trim() ? 'нет' : ''
})

defineExpose({ focus: () => inputEl.value?.select() })
</script>

<style scoped>
.nf-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px 6px 12px;
  flex-shrink: 0;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
}

.nf-bar-ic { font-size: 19px; color: var(--color-text-dim); flex-shrink: 0; }

.nf-bar-input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
}

.nf-bar-count {
  flex-shrink: 0;
  min-width: 44px;
  text-align: right;
  font-size: 12.5px;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-dim);
}

.nf-bar-btn {
  width: 30px;
  min-width: 30px;
  max-width: 30px;
  height: 30px;
  min-height: 30px;
  max-height: 30px;
  display: grid;
  place-items: center;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  cursor: pointer;
}

.nf-bar-btn .material-symbols-outlined { font-size: 19px; }
.nf-bar-btn:hover:not(:disabled) { background: color-mix(in oklch, var(--color-primary) 12%, transparent); color: var(--color-primary); }
.nf-bar-btn:disabled { opacity: 0.4; cursor: default; }
</style>

<!-- Совпадения рисуются ProseMirror-декорациями внутри контента редактора —
     их стили вне scoped (как курсоры соавторов). -->
<style>
.nf-hit {
  border-radius: 2px;
  background: color-mix(in oklch, var(--color-primary) 22%, transparent);
}

.nf-hit-current {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
</style>
