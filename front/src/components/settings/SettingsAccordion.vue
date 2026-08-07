<template>
  <section class="sa" :class="{ open }">
    <button class="sa-head" type="button" :aria-expanded="open" @click="open = !open">
      <span class="sa-title">{{ title }}</span>
      <span v-if="badge" class="sa-badge">{{ badge }}</span>
      <span class="material-symbols-outlined sa-chev">expand_more</span>
    </button>
    <!-- grid-template-rows 1fr→0fr: сворачивание без замера высоты содержимого
         (внутри бывают картинки и сетки переменной высоты). -->
    <div class="sa-body">
      <div class="sa-inner">
        <slot />
      </div>
    </div>
  </section>
</template>

<script setup>
import { ref } from 'vue'

const props = defineProps({
  title: { type: String, required: true },
  badge: { type: [String, Number], default: '' },
  defaultOpen: { type: Boolean, default: true },
})

const open = ref(props.defaultOpen)
</script>

<style scoped>
/* Тот же стеклянный лист, что у SettingCard: раскрывающийся блок настроек не
   должен выглядеть чужим рядом с обычными карточками. */
.sa {
  display: flex;
  flex-direction: column;
  padding: 6px 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.sa-head {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 12px 0;
  border: none;
  background: none;
  color: var(--color-text);
  cursor: pointer;
  text-align: left;
}

.sa-title {
  flex: 1;
  min-width: 0;
  font-size: 1rem;
  font-weight: 600;
}

.sa-badge {
  padding: 2px 9px;
  border-radius: 999px;
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  font-size: 0.75rem;
  font-weight: 600;
}

.sa-chev {
  color: var(--color-text-dim);
  transition: transform 0.24s cubic-bezier(0.2, 0, 0, 1);
}

.sa.open .sa-chev { transform: rotate(180deg); }

.sa-body {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.28s cubic-bezier(0.2, 0, 0, 1);
}

.sa.open .sa-body { grid-template-rows: 1fr; }

.sa-inner {
  overflow: hidden;
  min-height: 0;
}

.sa.open .sa-inner { padding: 2px 0 14px; }

@media (max-width: 560px) {
  .sa { padding: 4px 14px; }
}

@media (prefers-reduced-motion: reduce) {
  .sa-body, .sa-chev { transition: none; }
}
</style>
