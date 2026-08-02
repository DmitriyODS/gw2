<template>
  <!-- Карточка настроек: заголовок с пояснением и содержимое (строки
       `SettingRow`, плитки, свой контент). Тот же стеклянный лист, что у
       карточек разделов, — раньше его CSS был скопирован в каждый компонент
       настроек. -->
  <section class="scard">
    <header v-if="title || slots.head" class="scard-head">
      <div class="scard-head-text">
        <h3 v-if="title" class="scard-title">{{ title }}</h3>
        <p v-if="hint || slots.hint" class="scard-hint"><slot name="hint">{{ hint }}</slot></p>
      </div>
      <slot name="head" />
    </header>

    <slot />
  </section>
</template>

<script setup>
import { useSlots } from 'vue'

defineProps({
  title: { type: String, default: '' },
  hint: { type: String, default: '' },
})

const slots = useSlots()
</script>

<style scoped>
.scard {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
}

.scard-head {
  display: flex;
  align-items: flex-start;
  gap: 12px;
}

.scard-head-text { flex: 1; min-width: 0; }

.scard-title {
  margin: 0 0 4px;
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
}

.scard-hint {
  margin: 0;
  font-size: 0.85rem;
  line-height: 1.45;
  color: var(--color-text-dim);
}

@media (max-width: 560px) {
  .scard { padding: 14px; }
}
</style>
