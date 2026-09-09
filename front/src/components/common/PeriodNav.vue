<template>
  <div class="period-nav" :class="{ tight }">
    <!-- Три кнопки — один орган управления, поэтому они собраны в общую
         пилюлю-дорожку, как переключатель вида справа: две отдельно парящие
         круглые кнопки со «стеклянной» пилюлей между ними читались набором
         случайных элементов. -->
    <div class="pn-group">
      <button
        class="pn-btn"
        type="button"
        title="Предыдущий период"
        aria-label="Предыдущий период"
        @click="$emit('step', -1)"
      >
        <span class="material-symbols-outlined">chevron_left</span>
      </button>
      <button class="pn-btn pn-today" type="button" title="Сегодня" @click="$emit('today')">
        <span v-if="tight" class="material-symbols-outlined">today</span>
        <span v-else>Сегодня</span>
      </button>
      <button
        class="pn-btn"
        type="button"
        title="Следующий период"
        aria-label="Следующий период"
        @click="$emit('step', 1)"
      >
        <span class="material-symbols-outlined">chevron_right</span>
      </button>
    </div>

    <h2 class="pn-label">{{ label }}</h2>

    <!-- Тесная панель переключает вид пунктом меню «ещё» (см. periodViews.js):
         строка вкладок съедала бы место, которого и так нет. -->
    <AppTabs
      v-if="!tight && views"
      class="pn-views"
      :model-value="view"
      :tabs="PERIOD_VIEWS"
      variant="tint"
      dense
      @update:model-value="$emit('update:view', $event)"
    />
  </div>
</template>

<script setup>
/* Навигация по периоду с переключателем вида — общая для ежедневников и
   календарей (раньше в обоих разделах лежала одна и та же разметка).

   Раскладка строки постоянная: слева — шаг по периоду и «Сегодня», рядом
   подпись периода, справа — вид. Так устроены календари, к которым люди
   привыкли, и ничто не «плавает» по центру при изменении ширины окна. */
import AppTabs from '@/components/ui/AppTabs.vue'
import { PERIOD_VIEWS } from '@/utils/periodViews.js'

defineProps({
  /** Подпись текущего периода — считает раздел (у них разный формат). */
  label: { type: String, default: '' },
  /** Текущий вид: month | week | day. */
  view: { type: String, default: 'week' },
  /** Тесная панель: «Сегодня» значком, вкладки видов не показываем. */
  tight: { type: Boolean, default: false },
  /** Показывать вкладки видов. Панели средней ширины гасят их отдельно от
      `tight`: место под «Сегодня» и период ещё есть, а под вкладки уже нет. */
  views: { type: Boolean, default: true },
})

defineEmits(['step', 'today', 'update:view'])
</script>

<style scoped>
.period-nav {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  flex: 1 1 auto;
}

.pn-group {
  display: inline-flex;
  align-items: center;
  flex: none;
  gap: 2px;
  padding: 3px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

.pn-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  min-height: 32px;
  padding: 0 10px;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.18s, color 0.18s;
}

.pn-btn:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); color: var(--color-text); }
.pn-btn .material-symbols-outlined { font-size: 20px; }
.pn-today { color: var(--color-text); }

/* Подпись не растягивает строку и не сжимает соседей: место она забирает
   ровно по тексту, а переключатель вида остаётся прижатым к правому краю. */
.pn-label {
  flex: 0 1 auto;
  min-width: 0;
  margin: 0;
  font-size: 1.02rem;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Только ПЕРВАЯ буква прописная: `capitalize` поднимал каждое слово, и
   «7 сент. – 13 сент. 2026» превращалось в «7 Сент. – 13 Сент. 2026». */
.pn-label::first-letter { text-transform: uppercase; }

.pn-views { flex: none; margin-left: auto; }

.period-nav.tight { gap: 8px; }
.period-nav.tight .pn-label { font-size: 0.95rem; }
.period-nav.tight .pn-btn { min-width: 30px; min-height: 30px; padding: 0 6px; }

@media (max-width: 768px) {
  .pn-btn { min-height: 36px; }
}
</style>
