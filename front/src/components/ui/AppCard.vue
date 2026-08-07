<template>
  <!-- Кликабельная карточка остаётся <section>: внутри у неё бывают собственные
       кнопки, а кнопка в кнопке — невалидная разметка. -->
  <component
    :is="tag"
    class="card"
    :class="[`v-${variant}`, `tone-${tone}`, { clickable, flush, 'no-gap': noGap }]"
    :style="gap != null ? { gap: typeof gap === 'number' ? `${gap}px` : gap } : undefined"
    :role="clickable ? 'button' : undefined"
    :tabindex="clickable ? 0 : undefined"
    @click="clickable && $emit('click', $event)"
    @keydown.enter.prevent="clickable && $emit('click', $event)"
  >
    <header v-if="title || slots.head || slots.hint" class="card-head">
      <div class="card-head-text">
        <h3 v-if="title" class="card-title">{{ title }}</h3>
        <p v-if="hint || slots.hint" class="card-hint"><slot name="hint">{{ hint }}</slot></p>
      </div>
      <div v-if="slots.head" class="card-head-aside"><slot name="head" /></div>
    </header>

    <slot />

    <footer v-if="slots.footer" class="card-foot"><slot name="footer" /></footer>
  </component>
</template>

<script setup>
/* Стеклянный лист — единственная «карточка» платформы: заменила `.gw-card`,
   `.scard` (настройки), `.gw-group` и три десятка scoped-копий по разделам.

   variant: card — обычный лист; group — подложка ПОД карточками (радиус крупнее,
   вложенность скруглений идёт от внешнего к внутреннему). */
import { useSlots } from 'vue'

defineProps({
  title: { type: String, default: '' },
  hint: { type: String, default: '' },
  variant: { type: String, default: 'card', validator: (v) => ['card', 'group'].includes(v) },
  tone: {
    type: String,
    default: 'neutral',
    validator: (v) => ['neutral', 'primary', 'danger', 'success', 'warning'].includes(v),
  },
  /** Карточка сама по себе действие (плитка витрины, выбор темы). */
  clickable: { type: Boolean, default: false },
  /** Без внутренних полей — содержимое рисует их само (таблица, холст, лента). */
  flush: { type: Boolean, default: false },
  /** Содержимое само управляет промежутками между блоками. */
  noGap: { type: Boolean, default: false },
  /** Промежуток между блоками карточки, если стандартный не подходит. */
  gap: { type: [Number, String], default: null },
  tag: { type: String, default: 'section' },
})

defineEmits(['click'])
const slots = useSlots()
</script>

<style scoped>
/* Собственный blur карточке не нужен: за ней ровный фон панели-каркаса,
   поэтому «иней» имитируется градиентом --glass-bg + блик по кромке. */
.card {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
  padding: 18px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  text-align: left;
}

.card.v-group {
  gap: 12px;
  padding: 14px;
  border-radius: var(--radius-xl);
}

.card.flush { padding: 0; overflow: hidden; }
.card.no-gap { gap: 0; }

.card.clickable {
  cursor: pointer;
  transition: border-color 0.2s ease, background 0.2s ease;
}

.card.clickable:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: var(--glass-hover-bg, var(--glass-bg)), var(--acrylic-card-bg);
}

.tone-primary { border-color: color-mix(in oklch, var(--color-primary) 35%, var(--acrylic-border)); }
.tone-danger { border-color: color-mix(in oklch, var(--color-error) 35%, var(--acrylic-border)); }
.tone-success { border-color: color-mix(in oklch, var(--color-success) 35%, var(--acrylic-border)); }
.tone-warning { border-color: color-mix(in oklch, var(--color-warning) 35%, var(--acrylic-border)); }

/* Команды шапки переносятся ПОД заголовок, когда рядом с ним уже не помещаются
   (узкая панель, окно на телефоне): иначе кнопка держала свою ширину и рвала
   заголовок на строчки по одному слову. Перенос делает сам flex — базис
   заголовка задаёт порог, поэтому контейнерные запросы здесь не нужны. */
.card-head {
  display: flex;
  align-items: flex-start;
  flex-wrap: wrap;
  gap: 12px;
}

.card-head-text { flex: 1 1 220px; min-width: 0; }

.card-title {
  margin: 0;
  font-size: 1rem;
  font-weight: 600;
}

.card-hint {
  margin: 4px 0 0;
  font-size: 0.85rem;
  line-height: 1.45;
  color: var(--color-text-dim);
}

.card-head-aside {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.card-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

@media (max-width: 560px) {
  .card { padding: 14px; }
  .card.flush { padding: 0; }
}
</style>
