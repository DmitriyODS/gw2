<template>
  <div class="swn" :class="{ wide }">
    <AppButton
      variant="icon"
      size="sm"
      icon="chevron_left"
      title="Предыдущая неделя"
      aria-label="Предыдущая неделя"
      @click="$emit('step', -1)"
    />
    <button
      type="button"
      class="swn-range"
      :class="{ away: !current }"
      :title="current ? 'Текущая неделя' : 'Вернуться к текущей неделе'"
      @click="$emit('today')"
    >
      {{ range }}
      <span v-if="cycle" class="swn-cycle">{{ cycle }}</span>
    </button>
    <AppButton
      variant="icon"
      size="sm"
      icon="chevron_right"
      title="Следующая неделя"
      aria-label="Следующая неделя"
      @click="$emit('step', 1)"
    />
  </div>
</template>

<script setup>
/* «Где мы во времени» — один орган управления, а не три соседних кнопки:
   стрелки, диапазон и неделя цикла живут в общей пилюле. Сам диапазон
   работает кнопкой возврата к текущей неделе и подсвечивается, когда
   возвращаться есть куда.

   Общий для раздела и публичной страницы: разметка была одинаковой в обоих. */
import AppButton from '@/components/ui/AppButton.vue'

defineProps({
  /** Подпись недели («31.08 — 05.09»). */
  range: { type: String, default: '' },
  /** Неделя цикла («Неделя 1»); пусто — цикла нет, показывать нечего. */
  cycle: { type: String, default: '' },
  /** Показана текущая неделя. */
  current: { type: Boolean, default: false },
  /** Занять строку целиком (тесная панель и телефон). */
  wide: { type: Boolean, default: false },
})

defineEmits(['step', 'today'])
</script>

<style scoped>
.swn {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  /* Не сжимаемся: внутри даты в одну строку, и «сжатие» вылезло бы за пилюлю.
     Место кончилось — строка переносит нас целиком. */
  flex: none;
  padding: 2px;
  border-radius: var(--radius-full);
  background: var(--color-surface-low);
}
/* В тесной панели пилюля по содержимому оставляла справа пустое место.
   В flex-СТРОКЕ ширину даёт доля, а не align-self: stretch. */
.swn.wide { flex: 1 1 100%; }
.swn.wide .swn-range { flex: 1; }

/* По ЦЕНТРУ, а не по базовой линии: диапазон и неделя цикла разного кегля, и
   baseline прижимал обе строки к верху пилюли — снизу оставалось поля больше,
   чем сверху. */
.swn-range {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-width: 0;
  line-height: 1.2;
  padding: 4px 10px;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text);
  font: inherit;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
  cursor: pointer;
}
.swn-range:hover { background: var(--color-surface-high); }
.swn-range.away { color: var(--color-primary); }

.swn-cycle { font-size: 12px; font-weight: 600; line-height: 1.2; color: var(--color-text-dim); }
</style>
