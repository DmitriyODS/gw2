<template>
  <div class="sl" :class="{ dense, editing }" @click="onLaneClick">
    <!-- Часовые линии: их рисует каждая колонка, подписывает только рейка. -->
    <div
      v-for="h in hours"
      :key="h"
      class="sl-hline"
      :style="{ top: pct(h * 60) }"
    />

    <div v-if="!items.length" class="sl-free">свободно</div>

    <!-- Окна между занятиями: в конструкторе клик по окну заводит занятие
         ровно на него — приём прототипа, самый быстрый способ заполнить день. -->
    <component
      :is="editing ? 'button' : 'div'"
      v-for="(g, i) in gaps"
      :key="'g' + i"
      class="sl-gap"
      :type="editing ? 'button' : undefined"
      :style="spanStyle(g.s, g.e)"
      :title="`Окно ${hhmm(g.s)}–${hhmm(g.e)} · ${duration(g.e - g.s)}`"
      @click.stop="editing && emit('create', { start: g.s, end: g.e })"
    >
      <span v-if="g.e - g.s >= 35" class="sl-gap-label">окно {{ duration(g.e - g.s) }}</span>
    </component>

    <!-- Занятия. Пересекающиеся идут ОДНОЙ карточкой в несколько строк:
         так видно оба занятия целиком, а не две обрезанные половины. -->
    <template v-for="(group, i) in groups" :key="'c' + i">
      <component
        :is="group.length === 1 && !readonly ? 'button' : 'div'"
        class="sl-block"
        :class="[group.length > 1 ? 'clash' : `tone-${colorOf(group[0])}`]"
        :type="group.length === 1 && !readonly ? 'button' : undefined"
        :style="groupStyle(group)"
        @click.stop="group.length === 1 && emit('open', group[0])"
      >
        <template v-if="group.length === 1">
          <span class="sl-time">{{ hhmm(group[0].start_min) }}–{{ hhmm(group[0].end_min) }}</span>
          <span class="sl-title">{{ titleOf(group[0]) }}</span>
          <span v-if="metaOf(group[0])" class="sl-meta">{{ metaOf(group[0]) }}</span>
        </template>
        <template v-else>
          <span class="sl-clash-head">
            накладка · {{ hhmm(groupStart(group)) }}–{{ hhmm(groupEnd(group)) }}
          </span>
          <component
            :is="readonly ? 'div' : 'button'"
            v-for="it in group"
            :key="it.id"
            class="sl-row"
            :class="`tone-${colorOf(it)}`"
            :type="readonly ? undefined : 'button'"
            @click.stop="emit('open', it)"
          >
            <b>{{ titleOf(it) }}</b>
            <span>{{ rowMeta(it, group) }}</span>
          </component>
        </template>
      </component>
    </template>

    <!-- Линия «сейчас» — только у сегодняшнего дня. -->
    <div v-if="nowMinute != null" class="sl-now" :style="{ top: pct(nowMinute) }" />
  </div>
</template>

<script setup>
import { computed } from 'vue'
import { clusters, gapsOf, scalePercent } from '@/utils/scheduleLayout.js'
import { duration, hhmm } from '@/utils/scheduleCycle.js'

const props = defineProps({
  items: { type: Array, default: () => [] },
  bounds: { type: Object, required: true },
  categories: { type: Object, default: () => ({}) },
  fields: { type: Array, default: () => [] },
  gapMin: { type: Number, default: 20 },
  // dense — узкая колонка недели: показываем короткое имя занятия.
  dense: { type: Boolean, default: false },
  editing: { type: Boolean, default: false },
  readonly: { type: Boolean, default: false },
  // nowMinute — минута линии «сейчас» (null — не сегодня).
  nowMinute: { type: Number, default: null },
})

const emit = defineEmits(['open', 'create'])

const hours = computed(() => {
  const out = []
  for (let h = Math.ceil(props.bounds.from / 60); h <= Math.floor(props.bounds.to / 60); h++) out.push(h)
  return out
})

const gaps = computed(() => gapsOf(props.items, props.gapMin))
const groups = computed(() => clusters(props.items))

// Поля, вынесенные на блок шкалы (аудитория, преподаватель): подпись под
// названием — как «лекция · 2-3.10» в прототипе.
const blockFields = computed(() => props.fields.filter((f) => f.show_on_block))

function pct(minute) { return `${scalePercent(minute, props.bounds).toFixed(3)}%` }

function spanStyle(start, end) {
  const top = scalePercent(start, props.bounds)
  const height = scalePercent(end, props.bounds) - top
  return { top: `${top.toFixed(3)}%`, height: `${Math.max(height, 1.2).toFixed(3)}%` }
}

function groupStart(group) { return Math.min(...group.map((i) => i.start_min)) }
function groupEnd(group) { return Math.max(...group.map((i) => i.end_min)) }
function groupStyle(group) { return spanStyle(groupStart(group), groupEnd(group)) }

function colorOf(item) {
  return props.categories[item.category_id]?.color || 'blue'
}

function titleOf(item) {
  return (props.dense && item.short) ? item.short : item.title
}

function metaOf(item) {
  const parts = []
  const category = props.categories[item.category_id]
  if (category) parts.push(category.name)
  blockFields.value.forEach((f) => {
    const value = item.data?.[String(f.id)]
    if (value != null && value !== '') parts.push(String(value))
  })
  return parts.join(' · ')
}

// В накладке время строки нужно, только если занятия идут не ровно в одни часы.
function rowMeta(item, group) {
  const sameTime = group.every((i) => i.start_min === group[0].start_min && i.end_min === group[0].end_min)
  const parts = []
  if (!sameTime) parts.push(`${hhmm(item.start_min)}–${hhmm(item.end_min)}`)
  const meta = metaOf(item)
  if (meta) parts.push(meta)
  return parts.join(' · ')
}

/* Клик по пустому месту заводит занятие на это время (шаг 5 минут) — иначе
   заполнение дня требовало бы каждый раз открывать форму и вбивать часы. */
function onLaneClick(event) {
  if (!props.editing) return
  const rect = event.currentTarget.getBoundingClientRect()
  const span = props.bounds.to - props.bounds.from
  const raw = props.bounds.from + ((event.clientY - rect.top) / rect.height) * span
  const start = Math.max(props.bounds.from, Math.min(props.bounds.to - 60, Math.round(raw / 5) * 5))
  emit('create', { start, end: Math.min(props.bounds.to, start + 95) })
}
</script>

<style scoped>
.sl {
  position: relative;
  height: 100%;
  min-height: 0;
}
.editing { cursor: copy; }

.sl-hline {
  position: absolute;
  left: 0;
  right: 0;
  border-top: 1px solid var(--color-outline-dim);
  opacity: 0.5;
  pointer-events: none;
}

.sl-free {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  font-size: 12px;
  color: var(--color-text-dim);
  opacity: 0.6;
  pointer-events: none;
}

.sl-gap {
  position: absolute;
  left: 2px;
  right: 2px;
  display: grid;
  place-items: center;
  border: 1px dashed var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  font-size: 11px;
  padding: 0;
}
button.sl-gap { cursor: pointer; }
button.sl-gap:hover {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
}
.sl-gap-label { opacity: 0.75; }

.sl-block {
  position: absolute;
  left: 2px;
  right: 2px;
  display: flex;
  flex-direction: column;
  gap: 1px;
  overflow: hidden;
  padding: 4px 6px;
  text-align: left;
  border-radius: var(--radius-sm);
  border: 1px solid var(--sl-border, var(--color-outline-dim));
  background: var(--sl-surface, var(--color-surface-high));
  color: var(--color-text);
  font: inherit;
}
button.sl-block { cursor: pointer; }
button.sl-block:hover { filter: brightness(1.04); }

.sl-time {
  font-size: 11px;
  font-variant-numeric: tabular-nums;
  color: var(--sl-accent, var(--color-text-dim));
}
.sl-title {
  font-size: 13px;
  font-weight: 600;
  overflow-wrap: anywhere;
}
.sl-meta {
  font-size: 11px;
  color: var(--color-text-dim);
  overflow-wrap: anywhere;
}
.dense .sl-title { font-size: 12px; }
.dense .sl-meta { display: none; }

/* Накладка: рамка-предупреждение и строки-занятия внутри. */
.clash {
  border-color: var(--color-error);
  background: var(--color-error-container);
  gap: 2px;
}
.sl-clash-head {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-on-error-container);
}
.sl-row {
  display: flex;
  flex-direction: column;
  gap: 0;
  padding: 2px 4px;
  text-align: left;
  border-radius: var(--radius-xs);
  border: 1px solid var(--sl-border, var(--color-outline-dim));
  background: var(--sl-surface, var(--color-surface));
  color: var(--color-text);
  font: inherit;
  font-size: 11px;
}
button.sl-row { cursor: pointer; }
.sl-row b { font-size: 12px; }
.sl-row span { color: var(--color-text-dim); }

.sl-now {
  position: absolute;
  left: 0;
  right: 0;
  height: 2px;
  background: var(--color-primary);
  pointer-events: none;
}
.sl-now::before {
  content: '';
  position: absolute;
  left: -3px;
  top: -3px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-primary);
}

/* Цвет занятия — из палитры категорий (--tag-*), как у тегов задач. */
.tone-red { --sl-surface: var(--tag-red-surface); --sl-border: var(--tag-red-border); --sl-accent: var(--tag-red-accent); }
.tone-orange { --sl-surface: var(--tag-orange-surface); --sl-border: var(--tag-orange-border); --sl-accent: var(--tag-orange-accent); }
.tone-amber { --sl-surface: var(--tag-amber-surface); --sl-border: var(--tag-amber-border); --sl-accent: var(--tag-amber-accent); }
.tone-green { --sl-surface: var(--tag-green-surface); --sl-border: var(--tag-green-border); --sl-accent: var(--tag-green-accent); }
.tone-teal { --sl-surface: var(--tag-teal-surface); --sl-border: var(--tag-teal-border); --sl-accent: var(--tag-teal-accent); }
.tone-blue { --sl-surface: var(--tag-blue-surface); --sl-border: var(--tag-blue-border); --sl-accent: var(--tag-blue-accent); }
.tone-violet { --sl-surface: var(--tag-violet-surface); --sl-border: var(--tag-violet-border); --sl-accent: var(--tag-violet-accent); }
.tone-pink { --sl-surface: var(--tag-pink-surface); --sl-border: var(--tag-pink-border); --sl-accent: var(--tag-pink-accent); }
</style>
