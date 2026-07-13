<script setup>
import { ref } from 'vue'

defineProps({
  label: { type: String, required: true },
})
const emit = defineEmits(['jump'])
const rootEl = ref(null)
</script>

<template>
  <div class="msg-date-divider" ref="rootEl">
    <button type="button" class="msg-date-pill" @click="emit('jump', rootEl)">{{ label }}</button>
  </div>
</template>

<style scoped>
.msg-date-divider {
  position: sticky;
  top: 6px;
  z-index: 2;
  display: flex;
  justify-content: center;
  padding: 4px 0;
  /* Прилипшая плашка не перехватывает клики по пузырям под ней — кликается
     только сама пилюля (pointer-events: auto ниже). */
  pointer-events: none;
}

.msg-date-pill {
  pointer-events: auto;
  cursor: pointer;
  padding: 3px 12px;
  border-radius: var(--radius-full);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--acrylic-border);
  box-shadow: var(--glass-edge);
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-dim);
  user-select: none;
  transition: color 0.15s, transform 0.12s;
}

.msg-date-pill:hover {
  color: var(--color-text);
  transform: translateY(-1px);
}

.msg-date-pill:active {
  transform: translateY(0);
}
</style>
