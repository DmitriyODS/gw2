<template>
  <div
    class="tile"
    :class="{ clickable, selected }"
    :role="clickable ? 'button' : undefined"
    :tabindex="clickable ? 0 : undefined"
    :aria-pressed="selected ? 'true' : undefined"
    @click="clickable && $emit('click', $event)"
    @keydown.enter.prevent="clickable && $emit('click', $event)"
  >
    <slot name="media">
      <span v-if="icon" class="material-symbols-outlined tile-icon">{{ icon }}</span>
    </slot>
    <span class="tile-label"><slot>{{ label }}</slot></span>
    <small v-if="hint" class="tile-hint">{{ hint }}</small>
  </div>
</template>

<script setup>
/* Плитка: значок или превью сверху, подпись снизу. Категории, ярлыки, витрины,
   выбор оформления. Заменила `.gw-tile` и его scoped-копии. */
defineProps({
  icon: { type: String, default: '' },
  label: { type: String, default: '' },
  hint: { type: String, default: '' },
  clickable: { type: Boolean, default: true },
  selected: { type: Boolean, default: false },
})

defineEmits(['click'])
</script>

<style scoped>
.tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  width: 100%;
  min-height: 112px;
  padding: 18px 12px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-lg);
  background: var(--sk-raised-bg);
  box-shadow: var(--sk-raised-shadow);
  color: var(--color-text);
  font: inherit;
  font-size: 0.88rem;
  font-weight: 600;
  text-align: center;
  transition: transform 0.15s ease;
}

.tile.clickable { cursor: pointer; }

.tile.clickable:hover:not(.selected) {
  background: var(--sk-raised-hover-bg);
  transform: translateY(-1px);
}

.tile.clickable:active { transform: none; box-shadow: var(--sk-pressed-shadow); }

/* Выбранная плитка — нажатая и зафиксированная: вдавлена и залита тоном. */
.tile.selected {
  border-color: color-mix(in oklch, var(--color-primary) 40%, var(--sk-edge));
  background: var(--grain), var(--color-primary-container);
  box-shadow: var(--sk-pressed-shadow);
  color: var(--color-on-primary-container);
}

.tile-icon { font-size: 27px; color: var(--color-text-dim); }
.tile.selected .tile-icon { color: inherit; }

.tile-label { min-width: 0; overflow: hidden; text-overflow: ellipsis; }

.tile-hint {
  font-size: 0.76rem;
  font-weight: 500;
  line-height: 1.3;
  color: var(--color-text-dim);
}

.tile.selected .tile-hint { color: inherit; opacity: 0.85; }
</style>
