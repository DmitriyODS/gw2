<template>
  <div class="nb" :class="{ compact }">
    <div v-for="need in NEEDS" :key="need.key" class="nb-row" :title="need.hint">
      <span class="nb-emoji"><EmojiGlyph :char="need.emoji" /></span>
      <span v-if="!compact" class="nb-title">{{ need.title }}</span>
      <span class="nb-bar">
        <span class="nb-fill" :class="level(value(need.key))" :style="{ width: value(need.key) + '%' }"></span>
      </span>
      <span v-if="!compact" class="nb-value">{{ value(need.key) }}</span>
    </div>
  </div>
</template>

<script setup>
// Шкалы потребностей грувика: полный вид — в модалке питомца, compact —
// на карточках коллег (там важно лишь «всё ли у него хорошо»).
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { NEEDS } from '@/utils/pets.js'

const props = defineProps({
  needs: { type: Object, default: null },
  compact: { type: Boolean, default: false },
})

function value(key) {
  return Math.max(0, Math.min(100, props.needs?.[key] ?? 100))
}

// Цвет шкалы — светофор: пустая шкала ведёт в болезнь, и это должно быть
// видно до того, как питомец слёг.
function level(v) {
  if (v <= 20) return 'critical'
  if (v <= 50) return 'low'
  return 'ok'
}
</script>

<style scoped>
.nb { display: flex; flex-direction: column; gap: 7px; }
.nb.compact { gap: 4px; }

.nb-row { display: flex; align-items: center; gap: 8px; }
.nb-emoji { font-size: 14px; line-height: 1; flex-shrink: 0; }
.nb.compact .nb-emoji { font-size: 11px; }

.nb-title { font-size: 12.5px; color: var(--color-text-dim); width: 62px; flex-shrink: 0; }

.nb-bar {
  flex: 1;
  min-width: 0;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.nb.compact .nb-bar { height: 4px; }

.nb-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  transition: width 0.3s;
  background: var(--color-success);
}
.nb-fill.low { background: var(--color-warning); }
.nb-fill.critical { background: var(--color-error); }

.nb-value {
  font-size: 11.5px;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-dim);
  width: 24px;
  text-align: right;
  flex-shrink: 0;
}
</style>
