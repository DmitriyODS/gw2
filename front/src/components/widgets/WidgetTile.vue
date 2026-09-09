<template>
  <button
    class="wp-tile"
    :class="{ active, opened, dragging }"
    type="button"
    :title="app.title"
  >
    <LiveTile
      :title="app.title"
      :icon="app.icon"
      :faces="faces"
      wide
      dense
      :order="order"
      :paused="paused"
    />
    <span v-if="badge" class="wp-tile-badge" :class="{ alert: badge === '!' }">{{ badge }}</span>
  </button>
</template>

<script setup>
/**
 * Плитка раздела в панели «Виджетов». Отдельный компонент, потому что рисуется
 * в двух списках сразу — в избранном и в категориях «Всех», — а перетаскивание
 * у них разное: слушатели вешает список, они садятся на корневую кнопку.
 */
import LiveTile from '@/components/desktop/LiveTile.vue'

defineProps({
  /** Раздел из реестра desktop/apps.js. */
  app: { type: Object, required: true },
  /** Грани живой плитки (пусто — обычный значок). */
  faces: { type: Array, default: () => [] },
  /** Счётчик поверх плитки: число либо '!' (тревога). */
  badge: { type: [Number, String], default: 0 },
  /** Раздел на экране прямо сейчас. */
  active: { type: Boolean, default: false },
  /** Раздел открыт, но сейчас не на экране. */
  opened: { type: Boolean, default: false },
  /** Плитку тащат. */
  dragging: { type: Boolean, default: false },
  order: { type: Number, default: 0 },
  paused: { type: Boolean, default: false },
})
</script>

<style scoped>
/* Плитка всегда широкая: панель — одна колонка. */
.wp-tile {
  position: relative;
  display: flex;
  align-items: stretch;
  height: 74px;
  padding: 10px 12px;
  overflow: hidden;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.wp-tile:hover {
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--acrylic-border));
  background: color-mix(in oklch, var(--color-primary) 6%, var(--glass-bg));
}

.wp-tile.dragging { opacity: 0.45; }

/* Открытый раздел помечен полосой у кромки, активный — залит и подсвечен:
   панель заодно служит списком открытого, панели задач в этом каркасе нет. */
.wp-tile.opened::before {
  content: '';
  position: absolute;
  left: 0;
  top: 22%;
  bottom: 22%;
  width: 3px;
  border-radius: 0 var(--radius-full) var(--radius-full) 0;
  background: color-mix(in oklch, var(--color-primary) 55%, transparent);
}

.wp-tile.active {
  border-color: color-mix(in oklch, var(--color-primary) 45%, var(--acrylic-border));
  background: color-mix(in oklch, var(--color-primary) 12%, var(--glass-bg));
}

.wp-tile.active::before { background: var(--color-primary); top: 12%; bottom: 12%; }

.wp-tile-badge {
  position: absolute;
  top: 8px;
  right: 10px;
  min-width: 20px;
  height: 20px;
  padding: 0 6px;
  display: grid;
  place-items: center;
  border-radius: var(--radius-sm);
  background: color-mix(in oklch, var(--color-primary) 16%, var(--color-surface));
  border: 1px solid color-mix(in oklch, var(--color-primary) 24%, transparent);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 700;
}

.wp-tile-badge.alert {
  background: var(--color-error-container);
  border-color: color-mix(in oklch, var(--color-error) 30%, transparent);
  color: var(--color-on-error-container);
}
</style>
