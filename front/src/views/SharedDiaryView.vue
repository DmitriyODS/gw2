<template>
  <div class="sv-page">
    <header class="sv-top">
      <div class="sv-brand"><span class="material-symbols-outlined">event_note</span> Ежедневник</div>
      <div v-if="diary" class="sv-titlebox">
        <h1 class="sv-title">{{ diary.name }}</h1>
        <span v-if="diary.owner_name" class="sv-owner">{{ diary.owner_name }}</span>
      </div>
      <span class="sv-readonly">Только просмотр</span>
    </header>

    <div v-if="notFound" class="sv-state">
      <span class="material-symbols-outlined">link_off</span>
      <p>Ссылка не найдена или отозвана.</p>
    </div>

    <div v-else-if="diary" class="sv-shell">
      <div class="sv-toolbar">
        <div class="sv-subtabs">
          <SegmentedTabs :model-value="subtab" :tabs="subtabs" dense @update:model-value="setSubtab" />
        </div>
        <template v-if="subtab === 'active'">
          <div class="sv-nav">
            <button class="sv-icon-btn" @click="step(-1)"><span class="material-symbols-outlined">chevron_left</span></button>
            <button class="sv-today" @click="goToday">Сегодня</button>
            <button class="sv-icon-btn" @click="step(1)"><span class="material-symbols-outlined">chevron_right</span></button>
            <h2 class="sv-period">{{ periodLabel }}</h2>
          </div>
          <div class="sv-spacer" />
          <div class="sv-viewseg">
            <button v-for="v in viewModes" :key="v.value" :class="{ active: view === v.value }" @click="setView(v.value)">{{ v.label }}</button>
          </div>
        </template>
        <div v-else class="sv-spacer" />
      </div>

      <div class="sv-body">
        <div v-if="subtab === 'archive'" class="sv-archive">
          <div v-if="!archive.length" class="sv-empty"><span class="material-symbols-outlined">inventory_2</span><p>Архив пуст</p></div>
          <button v-for="e in archive" :key="e.id" class="sv-arow" @click="openEntry(e)">
            <span class="material-symbols-outlined sv-arow-check">check_circle</span>
            <span class="sv-arow-body"><span class="sv-arow-title">{{ e.title }}</span><span class="sv-arow-meta">{{ archiveMeta(e) }}</span></span>
            <span class="material-symbols-outlined sv-chev">chevron_right</span>
          </button>
        </div>

        <template v-else>
          <div v-if="view !== 'day'" class="sv-grid" :class="view">
            <template v-if="view === 'month'"><div v-for="(wd, i) in weekdays" :key="'h' + i" class="sv-wd">{{ wd }}</div></template>
            <div v-for="day in gridDays" :key="dayKey(day)" class="sv-day" :class="{ dim: view === 'month' && !inMonth(day), today: isToday(day) }" @click="openDay(day)">
              <div class="sv-day-head">
                <span class="sv-day-num">{{ day.getDate() }}</span>
                <span v-if="view === 'week'" class="sv-day-wd">{{ weekdayShort(day) }}</span>
                <span v-if="dayEntries(day).length" class="sv-day-count">{{ dayEntries(day).length }}</span>
              </div>
              <div class="sv-day-events">
                <div v-for="e in dayEntries(day).slice(0, view === 'week' ? 4 : 2)" :key="e.id" class="sv-event">
                  <span v-if="entryTime(e)" class="sv-event-time">{{ entryTime(e) }}</span>
                  <span class="sv-event-title">{{ e.title }}</span>
                </div>
              </div>
            </div>
          </div>
          <div v-else class="sv-daylist">
            <div v-if="!dayEntries(cursor).length" class="sv-empty"><span class="material-symbols-outlined">event_busy</span><p>На этот день записей нет</p></div>
            <button v-for="e in dayEntries(cursor)" :key="e.id" class="sv-dayrow" @click="openEntry(e)">
              <span class="sv-dayrow-time">{{ entryTime(e) || '—' }}</span>
              <span class="sv-dayrow-body"><span class="sv-dayrow-title">{{ e.title }}</span><span v-if="e.description" class="sv-dayrow-sub">{{ e.description }}</span></span>
              <span class="material-symbols-outlined sv-chev">chevron_right</span>
            </button>
          </div>
        </template>
      </div>
    </div>

    <div v-else class="sv-state"><span class="material-symbols-outlined spin">progress_activity</span></div>

    <!-- Просмотр записи -->
    <AppDialog v-model="entryOpen" :title="activeEntry?.title || 'Запись'" icon="event_note" size="md" :actions="[{ kind: 'cancel', label: 'Закрыть' }]" @cancel="entryOpen = false">
      <div v-if="activeEntry" class="sv-detail">
        <div class="sv-drow"><span class="material-symbols-outlined">calendar_today</span><span>{{ detailDate }}<template v-if="entryTime(activeEntry)"> · {{ entryTime(activeEntry) }}</template></span></div>
        <p v-if="activeEntry.description" class="sv-ddesc"><LinkifiedText :text="activeEntry.description" /></p>
        <p v-else class="sv-dnone">Без описания.</p>
      </div>
    </AppDialog>

    <!-- День -->
    <AppDialog v-model="dayOpen" :title="dayTitle" icon="today" size="md" :actions="[{ kind: 'cancel', label: 'Закрыть' }]" @cancel="dayOpen = false">
      <ul v-if="dayDialogEntries.length" class="sv-ddlist">
        <li v-for="e in dayDialogEntries" :key="e.id"><button class="sv-ddmain" @click="openEntry(e)"><span v-if="entryTime(e)" class="sv-ddtime">{{ entryTime(e) }}</span><span class="sv-ddtitle">{{ e.title }}</span><span class="material-symbols-outlined">chevron_right</span></button></li>
      </ul>
      <p v-else class="sv-dnone">На этот день записей нет.</p>
    </AppDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppDialog from '@/components/common/AppDialog.vue'
import LinkifiedText from '@/components/common/LinkifiedText.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import { getSharedDiary, getSharedEntries } from '@/api/diaries.js'

const route = useRoute()
const code = route.params.code

const diary = ref(null)
const notFound = ref(false)
const subtab = ref('active')
const view = ref('week')
const cursor = ref(startOfDay(new Date()))
const entries = ref([])
const archive = ref([])

const subtabs = [
  { value: 'active', label: 'Активные', icon: 'checklist' },
  { value: 'archive', label: 'Архив', icon: 'inventory_2' },
]
const viewModes = [
  { value: 'month', label: 'Месяц' },
  { value: 'week', label: 'Неделя' },
  { value: 'day', label: 'День' },
]
const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

function startOfDay(d) { const x = new Date(d); x.setHours(0, 0, 0, 0); return x }
function addDays(d, n) { const x = new Date(d); x.setHours(0, 0, 0, 0); x.setDate(x.getDate() + n); return x }
function startOfWeek(d) { const x = startOfDay(d); return addDays(x, -((x.getDay() + 6) % 7)) }
function startOfMonth(d) { const x = startOfDay(d); x.setDate(1); return x }
function dayKey(d) { const x = new Date(d); const p = (n) => String(n).padStart(2, '0'); return `${x.getFullYear()}-${p(x.getMonth() + 1)}-${p(x.getDate())}` }
const pad = (n) => String(n).padStart(2, '0')
function entryTime(e) {
  if (e.start_min == null) return ''
  const s = `${pad(Math.floor(e.start_min / 60))}:${pad(e.start_min % 60)}`
  return e.end_min == null ? s : `${s}–${pad(Math.floor(e.end_min / 60))}:${pad(e.end_min % 60)}`
}

const range = computed(() => {
  const base = cursor.value
  if (view.value === 'day') { const from = startOfDay(base); return { from, to: addDays(from, 1) } }
  if (view.value === 'week') { const from = startOfWeek(base); return { from, to: addDays(from, 7) } }
  const from = startOfWeek(startOfMonth(base)); return { from, to: addDays(from, 42) }
})
const gridDays = computed(() => {
  const { from, to } = range.value
  const n = Math.round((to.getTime() - from.getTime()) / 86400000)
  return Array.from({ length: n }, (_, i) => addDays(from, i))
})
const byDay = computed(() => { const m = {}; for (const e of entries.value) (m[e.entry_date] ||= []).push(e); return m })
function dayEntries(day) { return byDay.value[dayKey(day)] || [] }
function inMonth(day) { return day.getMonth() === cursor.value.getMonth() }
function weekdayShort(day) { return weekdays[(day.getDay() + 6) % 7] }
const todayKey = dayKey(new Date())
function isToday(day) { return dayKey(day) === todayKey }
function archiveMeta(e) { const d = new Date(e.entry_date).toLocaleDateString('ru-RU', { day: 'numeric', month: 'short', year: 'numeric' }); const t = entryTime(e); return t ? `${d} · ${t}` : d }

const periodLabel = computed(() => {
  const c = cursor.value
  if (view.value === 'day') return c.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  if (view.value === 'week') { const { from } = range.value; const end = addDays(from, 6); const o = { day: 'numeric', month: 'short' }; return `${from.toLocaleDateString('ru-RU', o)} – ${end.toLocaleDateString('ru-RU', o)} ${end.getFullYear()}` }
  return c.toLocaleDateString('ru-RU', { month: 'long', year: 'numeric' })
})

async function fetchEntries() {
  if (subtab.value === 'archive') {
    const data = await getSharedEntries(code, { archived: 1 })
    archive.value = data.items ?? []
  } else {
    const { from, to } = range.value
    const data = await getSharedEntries(code, { from: dayKey(from), to: dayKey(to) })
    entries.value = data.items ?? []
  }
}
function setSubtab(v) { subtab.value = v; fetchEntries() }
function setView(v) { view.value = v; fetchEntries() }
function step(dir) {
  const base = cursor.value
  if (view.value === 'day') cursor.value = addDays(base, dir)
  else if (view.value === 'week') cursor.value = addDays(base, dir * 7)
  else { const x = startOfMonth(base); x.setMonth(x.getMonth() + dir); cursor.value = x }
  fetchEntries()
}
function goToday() { cursor.value = startOfDay(new Date()); fetchEntries() }

const entryOpen = ref(false)
const activeEntry = ref(null)
function openEntry(e) { activeEntry.value = e; entryOpen.value = true; dayOpen.value = false }
const detailDate = computed(() => {
  if (!activeEntry.value) return ''
  const [y, m, d] = activeEntry.value.entry_date.split('-').map(Number)
  return new Date(y, m - 1, d).toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
})

const dayOpen = ref(false)
const dayDate = ref(null)
const dayDialogEntries = computed(() => (dayDate.value ? dayEntries(dayDate.value) : []))
const dayTitle = computed(() => dayDate.value ? new Date(dayDate.value).toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' }) : 'День')
function openDay(day) { dayDate.value = new Date(day); dayOpen.value = true }

onMounted(async () => {
  try {
    diary.value = await getSharedDiary(code)
    await fetchEntries()
  } catch {
    notFound.value = true
  }
})
</script>

<style scoped>
.sv-page { min-height: 100vh; background: var(--color-surface-low); display: flex; flex-direction: column; }
.sv-top { display: flex; align-items: center; gap: 16px; padding: 14px 20px; background: var(--acrylic-card-bg); border-bottom: 1px solid var(--color-outline-dim); }
.sv-brand { display: inline-flex; align-items: center; gap: 8px; font-weight: 700; color: var(--color-primary); }
.sv-titlebox { display: flex; flex-direction: column; min-width: 0; }
.sv-title { margin: 0; font-size: 18px; font-weight: 700; color: var(--color-text); }
.sv-owner { font-size: 13px; color: var(--color-text-dim); }
.sv-readonly { margin-left: auto; font-size: 12px; font-weight: 600; color: var(--color-text-dim); padding: 4px 10px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); }

.sv-state { flex: 1; display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; color: var(--color-text-dim); }
.sv-state .material-symbols-outlined { font-size: 44px; }

.sv-shell { flex: 1; min-height: 0; display: flex; flex-direction: column; margin: 16px; background: var(--acrylic-card-bg); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-xl); overflow: hidden; }
.sv-toolbar { flex-shrink: 0; display: flex; align-items: center; gap: 12px; flex-wrap: wrap; padding: 12px 16px; border-bottom: 1px solid var(--color-outline-dim); }
.sv-nav { display: flex; align-items: center; gap: 8px; }
.sv-period { margin: 0 0 0 6px; font-size: 16px; font-weight: 700; color: var(--color-text); text-transform: capitalize; }
.sv-today { height: 36px; padding: 0 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--acrylic-card-bg); color: var(--color-text); font-weight: 600; font-size: 13px; cursor: pointer; }
.sv-spacer { flex: 1; }
.sv-icon-btn { width: 36px; height: 36px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--acrylic-card-bg); color: var(--color-text-dim); cursor: pointer; }
/* Сегмент вида — единый стиль с периодами статистики (StatsPeriodControl). */
.sv-viewseg { display: inline-flex; gap: 2px; padding: 4px; background: var(--color-surface-high); background: var(--glass-bg); box-shadow: var(--glass-edge); border: 1px solid var(--acrylic-border); border-radius: var(--radius-full); }
.sv-viewseg button {
  min-height: 36px; padding: 8px 14px; border: none; background: transparent;
  border-radius: var(--radius-full); color: var(--color-text-dim); cursor: pointer;
  font-weight: 600; font-size: 13px; transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.sv-viewseg button:hover:not(.active) { color: var(--color-text); }
.sv-viewseg button.active { background: var(--grad-primary); color: var(--color-on-primary); font-weight: 700; box-shadow: var(--shadow-sm); }

.sv-body { flex: 1; min-height: 0; overflow: auto; }
.sv-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px; background: var(--color-outline-dim); min-height: 100%; }
.sv-grid.month { grid-template-rows: auto repeat(6, 1fr); }
.sv-grid.week { grid-template-rows: 1fr; }
.sv-wd { background: var(--acrylic-card-bg); padding: 8px 10px; text-align: center; font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.sv-day { background: var(--acrylic-card-bg); min-height: 100px; padding: 6px; display: flex; flex-direction: column; gap: 4px; cursor: pointer; overflow: hidden; }
.sv-grid.week .sv-day { min-height: 0; }
.sv-day:hover { background: var(--glass-hover-bg); }
.sv-day.dim { background: var(--color-surface-low); }
.sv-day-head { display: flex; align-items: center; justify-content: space-between; }
.sv-day-num { font-size: 13px; font-weight: 700; color: var(--color-text); width: 24px; height: 24px; display: grid; place-items: center; }
.sv-day.today .sv-day-num { background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-full); }
.sv-day-wd { font-size: 11px; color: var(--color-text-dim); text-transform: uppercase; }
.sv-day-count { min-width: 18px; height: 18px; padding: 0 5px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary); font-size: 11px; font-weight: 700; }
.sv-day-events { display: flex; flex-direction: column; gap: 3px; }
.sv-event { display: flex; align-items: baseline; gap: 6px; padding: 3px 6px; border-radius: var(--radius-sm); background: var(--color-primary-container); color: var(--color-on-primary-container); font-size: 12px; overflow: hidden; }
.sv-event-time { flex-shrink: 0; font-weight: 700; }
.sv-event-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.sv-daylist, .sv-archive { display: flex; flex-direction: column; gap: 8px; padding: 16px; }
.sv-dayrow, .sv-arow { display: flex; align-items: center; gap: 14px; width: 100%; text-align: left; padding: 12px 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); background: var(--acrylic-card-bg); cursor: pointer; }
.sv-dayrow:hover, .sv-arow:hover { background: var(--glass-hover-bg); }
.sv-dayrow-time { flex-shrink: 0; min-width: 56px; font-weight: 700; color: var(--color-primary); }
.sv-dayrow-body, .sv-arow-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.sv-dayrow-title { font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sv-dayrow-sub { font-size: 13px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sv-arow-check { color: var(--color-success); flex-shrink: 0; }
.sv-arow-title { font-weight: 600; color: var(--color-text-dim); text-decoration: line-through; }
.sv-arow-meta { font-size: 12px; color: var(--color-text-dim); }
.sv-chev { flex-shrink: 0; color: var(--color-text-dim); }
.sv-empty { display: flex; flex-direction: column; align-items: center; gap: 8px; padding: 40px; color: var(--color-text-dim); }
.sv-empty .material-symbols-outlined { font-size: 40px; }

.sv-detail { display: flex; flex-direction: column; gap: 10px; }
.sv-drow { display: inline-flex; align-items: center; gap: 8px; color: var(--color-text-dim); text-transform: capitalize; }
.sv-ddesc { margin: 0; white-space: pre-wrap; line-height: 1.5; color: var(--color-text); }
.sv-dnone { margin: 0; color: var(--color-text-dim); }
.sv-ddlist { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.sv-ddmain { display: flex; align-items: center; gap: 12px; width: 100%; padding: 10px 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--acrylic-card-bg); cursor: pointer; text-align: left; }
.sv-ddtime { flex-shrink: 0; font-weight: 700; color: var(--color-primary); }
.sv-ddtitle { flex: 1; min-width: 0; font-weight: 600; color: var(--color-text); }
.spin { animation: svspin 1s linear infinite; }
@keyframes svspin { to { transform: rotate(360deg); } }
</style>
