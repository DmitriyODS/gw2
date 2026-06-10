<template>
  <div class="rx-bar">
    <button
      v-for="emoji in visible"
      :key="emoji"
      class="rx-btn"
      :class="{ active: mine.includes(emoji) }"
      type="button"
      :aria-label="`Реакция ${emoji}`"
      @click="$emit('toggle', emoji)"
    >
      <span class="rx-emoji">{{ emoji }}</span>
      <span v-if="countOf(emoji)" class="rx-count">{{ countOf(emoji) }}</span>
    </button>
    <button
      class="rx-btn rx-more"
      type="button"
      :aria-label="expanded ? 'Свернуть реакции' : 'Добавить реакцию'"
      @click="expanded = !expanded"
    >
      <span class="material-symbols-outlined">{{ expanded ? 'close' : 'add_reaction' }}</span>
    </button>
  </div>
</template>

<script setup>
import { computed, ref } from 'vue'
import { FEED_REACTIONS } from '@/utils/groove.js'

const props = defineProps({
  reactions: { type: Object, default: () => ({}) },
  myReactions: { type: Array, default: () => [] },
})
defineEmits(['toggle'])

const expanded = ref(false)
const mine = computed(() => props.myReactions || [])

const countOf = (emoji) => props.reactions?.[emoji] || 0

// Свёрнуто — только реакции с откликом; развёрнуто — вся палитра.
const visible = computed(() =>
  expanded.value ? FEED_REACTIONS : FEED_REACTIONS.filter(e => countOf(e) > 0)
)
</script>

<style scoped>
.rx-bar {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.rx-btn {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 9px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: transparent;
  cursor: pointer;
  font-size: 13px;
  line-height: 1;
  color: var(--color-text);
  transition: background 0.15s, border-color 0.15s, transform 0.1s;
}
.rx-btn:hover { background: var(--color-surface-high); }
.rx-btn:active { transform: scale(0.92); }
.rx-btn.active {
  background: var(--color-primary-container);
  border-color: var(--color-primary);
}
.rx-emoji { font-size: 14px; }
.rx-count { font-weight: 600; font-size: 12px; }
.rx-more .material-symbols-outlined {
  font-size: 17px;
  color: var(--color-text-dim);
}
</style>
