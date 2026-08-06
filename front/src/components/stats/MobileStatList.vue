<template>
  <ul class="m-list">
    <li v-for="(row, i) in shown" :key="i" class="m-row">
      <slot :row="row" :index="i" />
    </li>
    <li v-if="!items.length" class="m-empty">{{ empty }}</li>
  </ul>

  <button v-if="rest > 0" class="m-more" type="button" @click="expanded = true">
    Показать ещё {{ rest }}
    <span class="material-symbols-outlined">expand_more</span>
  </button>
  <button v-else-if="expanded" class="m-more" type="button" @click="expanded = false">
    Свернуть
    <span class="material-symbols-outlined">expand_less</span>
  </button>
</template>

<script setup>
/* Список статистики на телефоне: несколько первых строк, остальное — по кнопке.
   Раньше карточка показывала весь список в собственной прокрутке — на таче это
   худший вариант: жест попадает во вложенный скролл вместо страницы, а карточка
   на пол-экрана вытесняет соседние. Строку рисует вызывающий (слот), поэтому его
   scoped-стили продолжают работать. */
import { computed, ref, watch } from 'vue'

const props = defineProps({
  items: { type: Array, default: () => [] },
  /** Сколько строк видно до раскрытия. */
  limit: { type: Number, default: 6 },
  empty: { type: String, default: 'Нет данных' },
})

const expanded = ref(false)

// Сменился период или сотрудник — список другой, раскрытие сбрасываем.
watch(() => props.items, () => { expanded.value = false })

const shown = computed(() => (expanded.value ? props.items : props.items.slice(0, props.limit)))
const rest = computed(() => Math.max(0, props.items.length - shown.value.length))
</script>

<style scoped>
.m-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.m-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  min-width: 0;
  background: var(--color-surface-high);
  border-radius: var(--radius-lg, 16px);
}

.m-empty {
  padding: 20px 12px;
  text-align: center;
  color: var(--color-text-dim);
  font-size: 13px;
}

.m-more {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  width: 100%;
  margin-top: 8px;
  padding: 9px 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-primary);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.m-more .material-symbols-outlined { font-size: 18px; }
</style>
