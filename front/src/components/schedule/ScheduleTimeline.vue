<template>
  <div class="st" :class="mode" :style="scaleStyle">
    <!-- Шапка колонок: день недели и его дата. -->
    <div v-if="mode === 'week'" class="st-head">
      <div class="st-rail-head" />
      <div
        v-for="d in days"
        :key="'h' + d"
        class="st-col-head"
        :class="{ today: isToday(d) }"
      >
        <b class="st-day-full">{{ WEEKDAYS[d].full }}</b>
        <b class="st-day-short">{{ WEEKDAYS[d].short }}</b>
        <time>{{ dayNumber(d) }}</time>
        <span v-if="busyOfDay(d)" class="st-load">{{ duration(busyOfDay(d)) }}</span>
      </div>
    </div>

    <div class="st-body">
      <!-- Рейка часов: подписи времени, общие для всех колонок. -->
      <div class="st-rail">
        <div v-for="h in hours" :key="h" class="st-rail-hour" :style="{ top: pct(h * 60) }">
          <span>{{ String(h).padStart(2, '0') }}:00</span>
        </div>
      </div>

      <div class="st-cols">
        <div
          v-for="d in days"
          :key="d"
          class="st-col"
          :class="{ today: isToday(d) }"
        >
          <ScheduleLane
            :items="itemsOf(d)"
            :bounds="bounds"
            :categories="categories"
            :fields="fields"
            :gap-min="gapMin"
            :dense="mode === 'week'"
            :editing="editing"
            :readonly="readonly"
            :now-minute="isToday(d) ? nowMinute : null"
            @open="emit('open', $event)"
            @create="emit('create', { weekday: d, ...$event })"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
import ScheduleLane from '@/components/schedule/ScheduleLane.vue'
import { busyOf, scaleBounds, scalePercent } from '@/utils/scheduleLayout.js'
import { WEEKDAYS, addDays, dateKey, duration, itemsOfDay } from '@/utils/scheduleCycle.js'

const props = defineProps({
  schedule: { type: Object, required: true },
  items: { type: Array, default: () => [] },
  monday: { type: [Date, String], required: true },
  // days — какие дни недели показывать (считаются по занятиям).
  days: { type: Array, default: () => [0, 1, 2, 3, 4] },
  gapMin: { type: Number, default: 20 },
  mode: { type: String, default: 'week' }, // week | day
  editing: { type: Boolean, default: false },
  readonly: { type: Boolean, default: false },
})

const emit = defineEmits(['open', 'create'])

const categories = computed(() => {
  const map = {}
  for (const c of props.schedule?.categories || []) map[c.id] = c
  return map
})
const fields = computed(() => props.schedule?.fields || [])

// Занятия показываемой недели по дням — считаются один раз на перерисовку.
const byDay = computed(() => {
  const map = {}
  props.days.forEach((d) => {
    map[d] = itemsOfDay(props.schedule, props.items, addDays(props.monday, d))
  })
  return map
})

// Границы шкалы — по занятиям ВСЕЙ показываемой недели: иначе соседние колонки
// растягивались бы каждая по-своему и время в них не совпадало.
const bounds = computed(() => scaleBounds(props.days.flatMap((d) => byDay.value[d] || [])))

const hours = computed(() => {
  const out = []
  for (let h = Math.ceil(bounds.value.from / 60); h <= Math.floor(bounds.value.to / 60); h++) out.push(h)
  return out
})

const todayKey = computed(() => dateKey(new Date()))
const nowMinute = computed(() => {
  const now = new Date()
  const minute = now.getHours() * 60 + now.getMinutes()
  return minute >= bounds.value.from && minute <= bounds.value.to ? minute : null
})

function itemsOf(weekday) { return byDay.value[weekday] || [] }
function busyOfDay(weekday) { return busyOf(itemsOf(weekday)) }
function isToday(weekday) { return dateKey(addDays(props.monday, weekday)) === todayKey.value }
function dayNumber(weekday) { return addDays(props.monday, weekday).getUTCDate() }
function pct(minute) { return `${scalePercent(minute, bounds.value).toFixed(3)}%` }

/* Высота шкалы — час примерно в 64px: так час читается одинаково и в неделе, и
   в дне, а прокрутка появляется ровно тогда, когда день длиннее экрана. */
const scaleStyle = computed(() => ({
  '--st-cols': String(props.days.length || 1),
  '--st-height': `${Math.max(360, ((bounds.value.to - bounds.value.from) / 60) * 64)}px`,
}))
</script>

<style scoped>
.st {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}

.st-head {
  display: grid;
  grid-template-columns: var(--st-rail) repeat(var(--st-cols), minmax(0, 1fr));
  gap: 4px;
  padding-bottom: 6px;
}
.st-col-head {
  display: flex;
  align-items: baseline;
  gap: 6px;
  padding: 4px 8px;
  border-radius: var(--radius-full);
  font-size: 12px;
  color: var(--color-text-dim);
  overflow: hidden;
  /* Заголовок меряет СВОЮ ширину: сколько дней показано, столько и колонок, и
     от их числа зависит, влезает ли полное имя дня и часы нагрузки. */
  container-type: inline-size;
  white-space: nowrap;
}
.st-col-head b {
  min-width: 0;
  font-size: 13px;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
/* Дата не сжимается: обрезать надо имя дня, а число дня — это то, ради чего
   шапка и нужна. */
.st-col-head time { flex: none; font-variant-numeric: tabular-nums; }
.st-col-head.today {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.st-col-head.today b { color: inherit; }
/* Нагрузка дня («1 ч 30») рвалась на две строки и распирала шапку: в узкой
   колонке её не показываем вовсе, имя дня сокращаем до «Пн». */
.st-load {
  margin-left: auto;
  font-variant-numeric: tabular-nums;
  opacity: 0.8;
  white-space: nowrap;
}
.st-day-short { display: none; }

@container (max-width: 132px) {
  .st-load { display: none; }
}
@container (max-width: 96px) {
  .st-day-full { display: none; }
  .st-day-short { display: inline; }
}

.st-body {
  position: relative;
  display: grid;
  grid-template-columns: var(--st-rail) minmax(0, 1fr);
  gap: 6px;
  flex: 1;
  min-height: 0;
  /* Шкала прокручивается сама: у раздела scroll=false, и без своей прокрутки
     содержимое молча обрезалось бы. Отступ сверху — подписи часов стоят по
     центру своей линии, и у самой верхней половина уезжала под кромку. */
  padding-top: 9px;
  padding-bottom: 4px;
  overflow-y: auto;
}

.st {
  --st-rail: 52px;
}
.st.day { --st-rail: 56px; }

.st-rail {
  position: relative;
  height: var(--st-height);
}
.st-rail-hour {
  position: absolute;
  right: 8px;
  transform: translateY(-50%);
  font-size: 11px;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  color: var(--color-text-dim);
  opacity: 0.75;
}

.st-cols {
  display: grid;
  grid-template-columns: repeat(var(--st-cols), minmax(0, 1fr));
  gap: 4px;
  height: var(--st-height);
}
.st-col {
  position: relative;
  min-width: 0;
  border-radius: var(--radius-md);
  /* Колонка — подложка, а не карточка: сплошная плашка спорила с блоками
     занятий, поэтому фон едва заметный, а форму держит рамка. */
  background: color-mix(in oklch, var(--color-surface-low) 55%, transparent);
  border: 1px solid var(--color-outline-dim);
}
.st-col.today {
  background: color-mix(in oklch, var(--color-primary) 7%, transparent);
  border-color: color-mix(in oklch, var(--color-primary) 35%, transparent);
}
.st.day .st-cols { grid-template-columns: minmax(0, 1fr); }
</style>
