<template>
  <div class="ss-page">
    <header class="ss-top">
      <div class="ss-brand">
        <span class="material-symbols-outlined">calendar_view_week</span> Расписание
      </div>
      <div v-if="schedule" class="ss-titlebox">
        <h1 class="ss-title">{{ schedule.name }}</h1>
        <span v-if="schedule.owner_name" class="ss-owner">{{ schedule.owner_name }}</span>
      </div>
      <span class="ss-readonly">Только просмотр</span>
    </header>

    <div v-if="notFound" class="ss-state">
      <span class="material-symbols-outlined">link_off</span>
      <p>Ссылка не найдена или отозвана.</p>
    </div>

    <BrandLoader v-else-if="loading" size="88" class="ss-loader" />

    <div v-else-if="schedule" class="ss-shell">
      <div class="ss-toolbar">
        <AppButton variant="icon" icon="chevron_left" label="Предыдущая неделя" @click="step(-1)" />
        <button type="button" class="ss-week" @click="goToday">
          <b>{{ weekRange }}</b>
          <span v-if="schedule.cycle_weeks > 1" class="ss-cycle">{{ cycleLabel }}</span>
        </button>
        <AppButton variant="icon" icon="chevron_right" label="Следующая неделя" @click="step(1)" />
      </div>

      <div class="ss-body">
        <ScheduleTimeline
          :schedule="schedule"
          :items="items"
          :monday="monday"
          :days="narrow ? [weekday] : days"
          :gap-min="schedule.gap_min || 20"
          :mode="narrow ? 'day' : 'week'"
          readonly
        />
        <ScheduleSummary :items="dayItems" :gap-min="schedule.gap_min || 20" :label="dayLabel" />
      </div>

      <!-- Узкий экран: полоса дней вместо колонок недели. -->
      <div v-if="narrow" class="ss-days">
        <button
          v-for="d in days"
          :key="d"
          type="button"
          class="ss-day"
          :class="{ active: d === weekday }"
          @click="weekday = d"
        >
          <b>{{ WEEKDAYS[d].short }}</b>
        </button>
      </div>
    </div>
  </div>
</template>

<script setup>
/* Публичная страница расписания по ссылке: код в адресе и есть доступ, вход не
   нужен. Рисуется теми же компонентами, что и раздел, — расхождения между
   «своим» и «открытым по ссылке» видом быть не должно. */
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import ScheduleSummary from '@/components/schedule/ScheduleSummary.vue'
import ScheduleTimeline from '@/components/schedule/ScheduleTimeline.vue'
import { getSharedSchedule } from '@/api/schedules.js'
import {
  WEEKDAYS, addDays, itemsOfDay, mondayOf, weekIndex, weekLabel,
} from '@/utils/scheduleCycle.js'
import { visibleDays } from '@/utils/scheduleLayout.js'

const route = useRoute()

const schedule = ref(null)
const items = ref([])
const loading = ref(true)
const notFound = ref(false)
const monday = ref(mondayOf(new Date()))
const weekday = ref((new Date().getDay() + 6) % 7)
const narrow = ref(window.innerWidth <= 768)

const days = computed(() => visibleDays(items.value, (new Date().getDay() + 6) % 7))
const dayItems = computed(() => (schedule.value
  ? itemsOfDay(schedule.value, items.value, addDays(monday.value, weekday.value))
  : []))
const dayLabel = computed(() => WEEKDAYS[weekday.value]?.full || '')

const cycleLabel = computed(() => (schedule.value
  ? weekLabel(schedule.value, weekIndex(schedule.value.cycle_anchor, monday.value, schedule.value.cycle_weeks))
  : ''))

const weekRange = computed(() => {
  const to = addDays(monday.value, days.value[days.value.length - 1] ?? 6)
  return `${dayMonth(monday.value)} — ${dayMonth(to)}`
})

function dayMonth(date) {
  const d = addDays(date, 0)
  return `${String(d.getUTCDate()).padStart(2, '0')}.${String(d.getUTCMonth() + 1).padStart(2, '0')}`
}

function step(delta) { monday.value = addDays(monday.value, delta * 7) }
function goToday() {
  monday.value = mondayOf(new Date())
  weekday.value = (new Date().getDay() + 6) % 7
}

function onResize() { narrow.value = window.innerWidth <= 768 }

onMounted(async () => {
  window.addEventListener('resize', onResize)
  try {
    const data = await getSharedSchedule(route.params.code)
    schedule.value = data.schedule
    items.value = data.items || []
  } catch {
    notFound.value = true
  } finally {
    loading.value = false
  }
})

onBeforeUnmount(() => window.removeEventListener('resize', onResize))
</script>

<style scoped>
.ss-page {
  display: flex;
  flex-direction: column;
  height: 100dvh;
  padding: 12px;
  gap: 10px;
  background: var(--color-surface);
  color: var(--color-on-surface);
}

.ss-top {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.ss-brand {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
  color: var(--color-primary);
}
.ss-titlebox { display: flex; align-items: baseline; gap: 8px; min-width: 0; }
.ss-title { margin: 0; font-size: 18px; overflow-wrap: anywhere; }
.ss-owner { font-size: 13px; color: var(--color-on-surface-variant); }
.ss-readonly {
  margin-left: auto;
  padding: 2px 10px;
  border-radius: var(--radius-pill);
  background: var(--color-surface-container);
  font-size: 12px;
  color: var(--color-on-surface-variant);
}

.ss-state {
  display: grid;
  place-items: center;
  gap: 8px;
  flex: 1;
  color: var(--color-on-surface-variant);
}
.ss-loader { margin: auto; }

.ss-shell { display: flex; flex-direction: column; gap: 8px; flex: 1; min-height: 0; }
.ss-toolbar { display: flex; align-items: center; gap: 4px; }
.ss-week {
  display: flex;
  align-items: baseline;
  gap: 8px;
  padding: 4px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}
.ss-week b { font-variant-numeric: tabular-nums; }
.ss-cycle {
  padding: 1px 8px;
  border-radius: var(--radius-pill);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 12px;
}

.ss-body { display: flex; flex-direction: column; gap: 8px; flex: 1; min-height: 0; }

.ss-days { display: flex; gap: 4px; overflow-x: auto; }
.ss-day {
  min-width: 44px;
  padding: 6px 4px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-surface-container-lowest);
  color: var(--color-on-surface-variant);
  font: inherit;
  cursor: pointer;
}
.ss-day.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
</style>
