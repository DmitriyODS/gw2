<template>
  <div class="fs">
    <EmptyState
      v-if="!summary || !summary.total"
      icon="insights"
      tone="soft"
      title="Ответов пока нет"
      subtitle="Как только форму заполнят, здесь появится сводка."
    />

    <template v-else>
      <!-- Верхние счётчики: сколько ответов, средний балл теста, темп. -->
      <AppGrid :min="200" :gap="12">
        <AppCard :gap="4">
          <span class="fs-kpi">{{ summary.total }}</span>
          <span class="fs-kpi-label">{{ plural(summary.total, 'ответ', 'ответа', 'ответов') }}</span>
        </AppCard>
        <AppCard v-if="summary.quiz" :gap="4">
          <span class="fs-kpi">{{ average }}</span>
          <span class="fs-kpi-label">средний балл из {{ summary.quiz.max_score }}</span>
        </AppCard>
        <AppCard :gap="4">
          <span class="fs-kpi">{{ todayCount }}</span>
          <span class="fs-kpi-label">за сегодня</span>
        </AppCard>
      </AppGrid>

      <!-- Динамика по дням: столбики без библиотек — их высота и есть данные. -->
      <AppCard title="Ответы по дням" :gap="10">
        <div class="fs-timeline">
          <div
            v-for="day in summary.timeline"
            :key="day.date"
            class="fs-bar"
            :title="`${dayLabel(day.date)}: ${day.count}`"
          >
            <span class="fs-bar-fill" :style="{ height: barHeight(day.count) }" />
          </div>
        </div>
        <div class="fs-timeline-legend">
          <span>{{ dayLabel(summary.timeline[0]?.date) }}</span>
          <span>{{ dayLabel(summary.timeline.at(-1)?.date) }}</span>
        </div>
      </AppCard>

      <!-- По вопросу: распределение вариантов, матрица сетки, примеры текстов. -->
      <AppCard
        v-for="q in summary.questions"
        :key="q.question_id"
        :title="q.title || 'Вопрос без названия'"
        :hint="`${q.answered} ${plural(q.answered, 'ответ', 'ответа', 'ответов')}`"
        :gap="10"
      >
        <div v-if="q.options?.length" class="fs-bars">
          <div v-for="opt in q.options" :key="opt.label" class="fs-line">
            <span class="fs-line-label">
              {{ opt.label }}
              <AppChip v-if="opt.other" size="sm" label="свой" />
            </span>
            <span class="fs-line-track">
              <span class="fs-line-fill" :style="{ width: share(opt.count, q.answered) }" />
            </span>
            <span class="fs-line-value">{{ opt.count }}</span>
          </div>
          <span v-if="q.average" class="fs-note">Среднее: {{ q.average.toFixed(2) }}</span>
        </div>

        <div v-else-if="q.rows?.length" class="fs-grid-wrap">
          <table class="fs-grid">
            <thead>
              <tr>
                <th />
                <th v-for="col in gridCols(q)" :key="col">{{ col }}</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="row in q.rows" :key="row.row">
                <th class="fs-grid-row">{{ row.row }}</th>
                <td v-for="opt in row.options" :key="opt.label">{{ opt.count }}</td>
              </tr>
            </tbody>
          </table>
        </div>

        <div v-else-if="q.texts?.length" class="fs-texts">
          <p v-for="(text, i) in q.texts" :key="i" class="fs-text">{{ text }}</p>
          <span v-if="q.answered > q.texts.length" class="fs-note">
            Показаны первые {{ q.texts.length }} — остальные в таблице ответов
          </span>
        </div>

        <span v-else-if="q.files" class="fs-note">
          Приложено файлов: {{ q.files }} — смотрите в таблице ответов
        </span>

        <EmptyState
          v-else
          size="sm"
          icon="inbox"
          title="На этот вопрос ещё не отвечали"
        />
      </AppCard>
    </template>
  </div>
</template>

<script setup>
/* Сводка ответов: счётчики, динамика по дням и разбор каждого вопроса.

   Диаграммы рисуются токенами и обычными блоками — библиотеку графиков ради
   полосок в бандл не тащим (и она всё равно не следовала бы теме). */
import { computed } from 'vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppGrid from '@/components/ui/AppGrid.vue'
import EmptyState from '@/components/common/EmptyState.vue'

const props = defineProps({
  summary: { type: Object, default: null },
})

const average = computed(() => (props.summary?.quiz?.average_score ?? 0).toFixed(1))

const maxDay = computed(() =>
  Math.max(1, ...(props.summary?.timeline || []).map((d) => d.count)))

const todayCount = computed(() => props.summary?.timeline?.at(-1)?.count ?? 0)

function barHeight(count) {
  // Ненулевой день обязан быть видимым, иначе один ответ на фоне сотни
  // выглядит как пустой день.
  return count ? `${Math.max(8, Math.round((count / maxDay.value) * 100))}%` : '2px'
}

function share(count, total) {
  if (!total) return '0%'
  return `${Math.round((count / total) * 100)}%`
}

function gridCols(q) {
  return (q.rows?.[0]?.options || []).map((o) => o.label)
}

function dayLabel(date) {
  if (!date) return ''
  const [y, m, d] = date.split('-')
  return `${d}.${m}`
}

function plural(n, one, few, many) {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return one
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 10 || mod100 >= 20)) return few
  return many
}
</script>

<style scoped>
.fs { display: flex; flex-direction: column; gap: 14px; min-width: 0; }

.fs-kpi { font-size: 26px; font-weight: 700; }
.fs-kpi-label { font-size: 13px; color: var(--color-text-dim); }

.fs-timeline { display: flex; align-items: flex-end; gap: 3px; height: 90px; }
.fs-bar { display: flex; flex: 1; min-width: 0; align-items: flex-end; height: 100%; }
.fs-bar-fill {
  width: 100%;
  border-radius: 2px 2px 0 0;
  background: var(--color-primary);
}
.fs-timeline-legend {
  display: flex;
  justify-content: space-between;
  font-size: 11px;
  color: var(--color-text-dim);
}

.fs-bars { display: flex; flex-direction: column; gap: 6px; }
.fs-line { display: flex; align-items: center; gap: 8px; }
.fs-line-label {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 40%;
  min-width: 0;
  font-size: 13px;
  overflow-wrap: anywhere;
}
.fs-line-track {
  flex: 1;
  min-width: 0;
  height: 12px;
  border-radius: 6px;
  background: var(--color-surface-low);
  overflow: hidden;
}
.fs-line-fill { display: block; height: 100%; background: var(--color-primary); }
.fs-line-value { min-width: 32px; text-align: right; font-size: 13px; color: var(--color-text-dim); }

.fs-note { font-size: 12px; color: var(--color-text-dim); }

.fs-texts { display: flex; flex-direction: column; gap: 6px; }
.fs-text {
  margin: 0;
  padding: 8px 10px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-low);
  font-size: 13px;
  overflow-wrap: anywhere;
}

/* Матрица сетки прокручивается внутри себя: горизонтальной прокрутки у самого
   раздела быть не должно. */
.fs-grid-wrap { overflow-x: auto; }
.fs-grid { border-collapse: collapse; font-size: 13px; width: 100%; }
.fs-grid th, .fs-grid td { padding: 6px 10px; text-align: center; }
.fs-grid-row { text-align: left; color: var(--color-text-dim); max-width: 220px; overflow-wrap: anywhere; }
.fs-grid tbody tr:nth-child(odd) { background: var(--color-surface-low); }
</style>
