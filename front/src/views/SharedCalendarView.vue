<template>
  <div class="sc">
    <!-- Ошибка доступа -->
    <div v-if="error" class="sc-error">
      <span class="material-symbols-outlined">link_off</span>
      <h2>Ссылка недоступна</h2>
      <p>{{ error }}</p>
    </div>

    <template v-else>
      <header class="sc-head">
        <div class="sc-title">
          <span class="material-symbols-outlined">calendar_month</span>
          <h1>{{ calendar?.name || 'Календарь' }}</h1>
          <span class="sc-badge">только просмотр</span>
        </div>
        <div class="sc-head-actions">
          <div class="sc-search">
            <span class="material-symbols-outlined">search</span>
            <input v-model="searchInput" type="text" placeholder="Поиск…" @input="onSearch" />
            <button v-if="searchInput" class="sc-search-clear" @click="clearSearch">
              <span class="material-symbols-outlined">close</span>
            </button>
          </div>
          <button class="sc-btn" title="Экспорт в XLSX" @click="openExport">
            <span class="material-symbols-outlined">download</span>
            <span class="sc-btn-label">Экспорт</span>
          </button>
        </div>
      </header>

      <div class="sc-toolbar">
        <div class="sc-nav">
          <button class="sc-icon-btn" title="Назад" @click="step(-1)"><span class="material-symbols-outlined">chevron_left</span></button>
          <button class="sc-today" @click="goToday">Сегодня</button>
          <button class="sc-icon-btn" title="Вперёд" @click="step(1)"><span class="material-symbols-outlined">chevron_right</span></button>
          <h2 class="sc-period">{{ periodLabel }}</h2>
        </div>
        <div class="sc-spacer" />
        <div class="sc-viewseg">
          <button v-for="v in viewModes" :key="v.value" :class="{ active: view === v.value }" @click="setView(v.value)">{{ v.label }}</button>
        </div>
      </div>

      <div class="sc-body">
        <!-- Десктоп: месяц/неделя — сетка плиток -->
        <div v-if="!isMobile && view !== 'day'" class="sc-grid" :class="view">
          <template v-if="view === 'month'">
            <div v-for="(wd, i) in weekdays" :key="'h' + i" class="sc-wd">{{ wd }}</div>
          </template>
          <div
            v-for="day in gridDays"
            :key="dayKey(day)"
            class="sc-day"
            :class="{ dim: view === 'month' && !inCurrentMonth(day), today: isToday(day) }"
            @click="openDay(day)"
          >
            <div class="sc-day-head">
              <span class="sc-day-num">{{ day.getDate() }}</span>
              <span v-if="view === 'week'" class="sc-day-wd">{{ weekdayShort(day) }}</span>
              <span v-if="dayEntries(day).length" class="sc-day-count">{{ dayEntries(day).length }}</span>
            </div>
            <div class="sc-day-events">
              <div v-for="e in dayPreview(day)" :key="e.id" class="sc-event">
                <span class="sc-event-time">{{ hhmm(e.event_at) }}</span>
                <span class="sc-event-title">{{ entryTitle(calendar, e) }}</span>
              </div>
              <div v-if="dayEntries(day).length > dayPreview(day).length" class="sc-event-more">
                +{{ dayEntries(day).length - dayPreview(day).length }}
              </div>
            </div>
          </div>
        </div>

        <!-- Мобайл: список по датам с количеством записей -->
        <div v-else-if="isMobile && view !== 'day'" class="sc-agenda">
          <button
            v-for="day in agendaDays"
            :key="dayKey(day)"
            class="sc-agenda-row"
            :class="{ today: isToday(day) }"
            @click="openDay(day)"
          >
            <div class="sc-agenda-date">
              <span class="sc-agenda-dnum">{{ day.getDate() }}</span>
              <span class="sc-agenda-dwd">{{ weekdayShort(day) }}</span>
            </div>
            <div class="sc-agenda-body">
              <span class="sc-agenda-month">{{ agendaMonth(day) }}</span>
              <span v-if="dayEntries(day).length" class="sc-agenda-prev">{{ agendaPreview(day) }}</span>
              <span v-else class="sc-agenda-empty">Нет записей</span>
            </div>
            <span v-if="dayEntries(day).length" class="sc-day-count">{{ dayEntries(day).length }}</span>
            <span class="material-symbols-outlined sc-agenda-chev">chevron_right</span>
          </button>
        </div>

        <div v-else class="sc-daylist">
          <div v-if="!dayEntries(cursor).length" class="sc-empty">
            <span class="material-symbols-outlined">event_busy</span>
            <p>На этот день записей нет</p>
          </div>
          <button v-for="e in dayEntries(cursor)" :key="e.id" class="sc-dayrow" @click="openEntry(e)">
            <span class="sc-dayrow-time">{{ hhmm(e.event_at) }}</span>
            <span class="sc-dayrow-body">
              <span class="sc-dayrow-title">{{ entryTitle(calendar, e) }}</span>
              <span v-for="cf in cardFields(calendar, e)" :key="cf.field.id" class="sc-dayrow-sub">
                <span class="sc-dayrow-flabel">{{ cf.field.label }}:</span> {{ cf.value }}
              </span>
            </span>
            <span class="material-symbols-outlined sc-dayrow-chev">chevron_right</span>
          </button>
        </div>

        <div v-if="loading" class="sc-overlay"><span class="material-symbols-outlined spin">progress_activity</span></div>
      </div>

      <footer class="sc-foot">
        <span class="sc-brand">Groove Work</span>
      </footer>
    </template>

    <CalendarDayDialog
      v-model="dayDialogOpen"
      :calendar="calendar"
      :date="dayDialogDate"
      :entries="dayDialogEntries"
      readonly
      @open-entry="openEntry"
    />

    <CalendarEntryDialog v-model="dialogOpen" :calendar="calendar" :entry="activeEntry" readonly />

    <AppDialog
      v-model="exportOpen"
      title="Экспорт в XLSX" icon="download" size="md" :busy="exporting"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Экспортировать', icon: 'download' }]"
      @cancel="exportOpen = false" @confirm="doExport"
    >
      <div class="sc-export">
        <p class="sc-export-period">Будут выгружены записи за период: <b>{{ periodLabel }}</b>. Колонка «Дата и время» включается всегда.</p>
        <div class="sc-export-head">
          <span class="sc-export-title">Дополнительные поля</span>
          <div class="sc-export-bulk">
            <button class="sc-link-btn" @click="selectAllExport">Выбрать всё</button>
            <button class="sc-link-btn" @click="clearAllExport">Снять всё</button>
          </div>
        </div>
        <div class="sc-export-fields">
          <label v-for="f in exportableFields" :key="f.id" class="sc-export-row">
            <Checkbox :model-value="exportFields.has(f.id)" binary @update:model-value="toggleExportField(f.id)" />
            <span class="material-symbols-outlined">{{ fieldIcon(f.type) }}</span>
            <span class="sc-export-name">{{ f.label }}</span>
          </label>
        </div>
      </div>
    </AppDialog>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import Checkbox from 'primevue/checkbox'
import AppDialog from '@/components/common/AppDialog.vue'
import CalendarEntryDialog from '@/components/calendar/CalendarEntryDialog.vue'
import CalendarDayDialog from '@/components/calendar/CalendarDayDialog.vue'
import { getSharedCalendar, getSharedEntries, exportSharedEntries } from '@/api/calendars.js'
import { fieldIcon, isExportable, entryTitle, hhmm, cardFields } from '@/utils/calendarFields.js'
import { dayKey } from '@/stores/calendars.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const { isMobile } = useBreakpoint()

const route = useRoute()
const code = route.params.code

const calendar = ref(null)
const error = ref(null)
const entries = ref([])
const loading = ref(false)

const viewModes = [
  { value: 'month', label: 'Месяц' },
  { value: 'week', label: 'Неделя' },
  { value: 'day', label: 'День' },
]
const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

const view = ref('month')
function startOfDay(d) { const x = new Date(d); x.setHours(0, 0, 0, 0); return x }
function addDays(d, n) { const x = startOfDay(d); x.setDate(x.getDate() + n); return x }
function startOfWeek(d) { const x = startOfDay(d); return addDays(x, -((x.getDay() + 6) % 7)) }
function startOfMonth(d) { const x = startOfDay(d); x.setDate(1); return x }
const cursor = ref(startOfDay(new Date()))

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
const entriesByDay = computed(() => {
  const map = {}
  for (const e of entries.value) (map[dayKey(e.event_at)] ||= []).push(e)
  return map
})
function dayEntries(day) { return entriesByDay.value[dayKey(day)] || [] }
function inCurrentMonth(day) { return day.getMonth() === cursor.value.getMonth() }
function weekdayShort(day) { return weekdays[(day.getDay() + 6) % 7] }
const todayKey = dayKey(new Date())
function isToday(day) { return dayKey(day) === todayKey }

function dayPreview(day) { return dayEntries(day).slice(0, view.value === 'week' ? 4 : 2) }
const agendaDays = computed(() => {
  if (view.value === 'week') return gridDays.value
  const c = cursor.value
  const days = new Date(c.getFullYear(), c.getMonth() + 1, 0).getDate()
  return Array.from({ length: days }, (_, i) => new Date(c.getFullYear(), c.getMonth(), i + 1))
})
function agendaMonth(day) { return day.toLocaleDateString('ru-RU', { month: 'short' }) }
function agendaPreview(day) {
  return dayEntries(day).slice(0, 2)
    .map((e) => `${hhmm(e.event_at)} ${entryTitle(calendar.value, e)}`.trim())
    .join(' · ')
}

// Модалка дня (read-only): список записей, открытие карточки.
const dayDialogOpen = ref(false)
const dayDialogDate = ref(null)
const dayDialogEntries = computed(() => (dayDialogDate.value ? dayEntries(dayDialogDate.value) : []))
function openDay(day) { dayDialogDate.value = new Date(day); dayDialogOpen.value = true }

const periodLabel = computed(() => {
  const c = cursor.value
  if (view.value === 'day') return c.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  if (view.value === 'week') {
    const start = range.value.from
    const end = addDays(start, 6)
    const opts = { day: 'numeric', month: 'short' }
    return `${start.toLocaleDateString('ru-RU', opts)} – ${end.toLocaleDateString('ru-RU', opts)} ${end.getFullYear()}`
  }
  return c.toLocaleDateString('ru-RU', { month: 'long', year: 'numeric' })
})

function setView(v) { if (view.value !== v) { view.value = v; fetchEntries() } }
function step(dir) {
  const base = cursor.value
  if (view.value === 'day') cursor.value = addDays(base, dir)
  else if (view.value === 'week') cursor.value = addDays(base, dir * 7)
  else { const x = startOfMonth(base); x.setMonth(x.getMonth() + dir); cursor.value = x }
  fetchEntries()
}
function goToday() { cursor.value = startOfDay(new Date()); fetchEntries() }

const searchInput = ref('')
const search = ref('')
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => { search.value = searchInput.value.trim(); fetchEntries() }, 300)
}
function clearSearch() { searchInput.value = ''; search.value = ''; fetchEntries() }

let seq = 0
async function fetchEntries() {
  const s = ++seq
  loading.value = true
  try {
    const { from, to } = range.value
    const data = await getSharedEntries(code, { from: from.toISOString(), to: to.toISOString(), search: search.value })
    if (s !== seq) return
    entries.value = data.items ?? []
  } catch (e) {
    if (s === seq) error.value = e?.message || 'Не удалось загрузить записи'
  } finally {
    if (s === seq) loading.value = false
  }
}

async function load() {
  try {
    calendar.value = await getSharedCalendar(code)
    await fetchEntries()
  } catch (e) {
    error.value = e?.message || 'Ссылка не найдена или была отозвана'
  }
}

const dialogOpen = ref(false)
const activeEntry = ref(null)
function openEntry(e) { activeEntry.value = e; dialogOpen.value = true }

// Экспорт.
const exportOpen = ref(false)
const exporting = ref(false)
const exportFields = ref(new Set())
const exportableFields = computed(() => (calendar.value?.fields || []).filter((f) => isExportable(f.type)))
function openExport() { exportFields.value = new Set(exportableFields.value.map((f) => f.id)); exportOpen.value = true }
function toggleExportField(id) { const s = new Set(exportFields.value); s.has(id) ? s.delete(id) : s.add(id); exportFields.value = s }
function selectAllExport() { exportFields.value = new Set(exportableFields.value.map((f) => f.id)) }
function clearAllExport() { exportFields.value = new Set() }
async function doExport() {
  exporting.value = true
  try {
    const { from, to } = range.value
    const resp = await exportSharedEntries(code, {
      fields: [...exportFields.value], from: from.toISOString(), to: to.toISOString(), search: search.value,
    })
    if (!resp.ok) throw new Error('Не удалось выгрузить')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${calendar.value?.name || 'calendar'}.xlsx`
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    URL.revokeObjectURL(url)
    exportOpen.value = false
  } catch {
    /* тихо: публичная страница без тостов */
  } finally {
    exporting.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.sc { height: 100%; min-height: 100dvh; display: flex; flex-direction: column; background: var(--color-bg); }

.sc-error {
  flex: 1; min-height: 100dvh; display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 8px; color: var(--color-text-dim); text-align: center; padding: 24px;
}
.sc-error .material-symbols-outlined { font-size: 56px; }
.sc-error h2 { margin: 8px 0 0; color: var(--color-text); }
.sc-error p { margin: 0; }

.sc-head {
  flex: none; display: flex; align-items: center; justify-content: space-between; gap: 16px; flex-wrap: wrap;
  padding: 16px 20px; border-bottom: 1px solid var(--color-outline-dim); background: var(--acrylic-card-bg);
}
.sc-title { display: flex; align-items: center; gap: 10px; min-width: 0; }
.sc-title .material-symbols-outlined { color: var(--color-primary); }
.sc-title h1 { margin: 0; font-size: 20px; font-weight: 700; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sc-badge { padding: 3px 10px; border-radius: var(--radius-full); background: var(--color-surface-high); color: var(--color-text-dim); font-size: 12px; font-weight: 600; }
.sc-head-actions { flex: 1; min-width: 0; display: flex; align-items: center; gap: 10px; justify-content: flex-end; }

.sc-search {
  flex: 0 1 280px; display: flex; align-items: center; gap: 8px; height: 40px; padding: 0 12px;
  background: var(--color-surface-low); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full);
}
.sc-search > .material-symbols-outlined { color: var(--color-text-dim); font-size: 20px; }
.sc-search input { flex: 1; min-width: 0; border: none; background: none; outline: none; color: var(--color-text); font-size: 14px; }
.sc-search-clear { border: none; background: none; cursor: pointer; color: var(--color-text-dim); display: grid; place-items: center; }

.sc-toolbar {
  flex: none; display: flex; align-items: center; gap: 12px; flex-wrap: wrap;
  padding: 10px 20px; border-bottom: 1px solid var(--color-outline-dim); background: var(--acrylic-card-bg);
}
.sc-nav { display: flex; align-items: center; gap: 8px; }
.sc-period { margin: 0 0 0 6px; font-size: 16px; font-weight: 700; color: var(--color-text); text-transform: capitalize; white-space: nowrap; }
.sc-today { height: 34px; padding: 0 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--acrylic-card-bg); color: var(--color-text); font-weight: 600; font-size: 13px; cursor: pointer; }
.sc-spacer { flex: 1; }
/* Сегмент вида — единый стиль с периодами статистики (StatsPeriodControl). */
.sc-viewseg { display: inline-flex; gap: 2px; padding: 4px; background: var(--color-surface-high); background: var(--glass-bg); box-shadow: var(--glass-edge); border: 1px solid var(--acrylic-border); border-radius: var(--radius-full); }
.sc-viewseg button {
  min-height: 34px; padding: 7px 14px; border: none; background: transparent;
  border-radius: var(--radius-full); color: var(--color-text-dim); cursor: pointer;
  font-weight: 600; font-size: 13px; transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.sc-viewseg button:hover:not(.active) { color: var(--color-text); }
.sc-viewseg button.active { background: var(--grad-primary); color: var(--color-on-primary); font-weight: 700; box-shadow: var(--shadow-sm); }

.sc-icon-btn { width: 36px; height: 36px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--acrylic-card-bg); color: var(--color-text-dim); cursor: pointer; }
.sc-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }

.sc-body { position: relative; flex: 1; min-height: 0; overflow: auto; padding: 16px; }
.sc-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px; background: var(--color-outline-dim); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); overflow: hidden; }
.sc-grid.month { grid-template-rows: auto repeat(6, 1fr); }
.sc-grid.week { grid-template-rows: 1fr; }
.sc-grid.week .sc-day { min-height: 160px; }
.sc-wd { background: var(--acrylic-card-bg); padding: 8px 10px; text-align: center; font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.sc-day { background: var(--acrylic-card-bg); min-height: 104px; padding: 6px; display: flex; flex-direction: column; gap: 4px; overflow: hidden; }
.sc-day.dim { background: var(--color-surface-low); }
.sc-day.dim .sc-day-num { opacity: 0.55; }
.sc-day-head { display: flex; align-items: center; justify-content: space-between; }
.sc-day-num { font-size: 13px; font-weight: 700; color: var(--color-text); width: 24px; height: 24px; display: grid; place-items: center; }
.sc-day.today .sc-day-num { background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-full); }
.sc-day-wd { font-size: 11px; color: var(--color-text-dim); text-transform: uppercase; }
.sc-day-count {
  flex-shrink: 0; min-width: 18px; height: 18px; padding: 0 5px;
  display: inline-flex; align-items: center; justify-content: center;
  border-radius: var(--radius-full); background: var(--color-primary);
  color: var(--color-on-primary); font-size: 11px; font-weight: 700;
}
.sc-day-events { display: flex; flex-direction: column; gap: 3px; }
.sc-event { display: flex; align-items: baseline; gap: 6px; width: 100%; text-align: left; padding: 3px 6px; border: none; border-radius: var(--radius-sm); background: var(--color-primary-container); color: var(--color-on-primary-container); font-size: 12px; overflow: hidden; }
.sc-event-time { flex-shrink: 0; font-weight: 700; font-variant-numeric: tabular-nums; }
.sc-event-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sc-event-more { font-size: 11px; font-weight: 600; color: var(--color-text-dim); padding-left: 6px; }

/* ── Мобильная агенда (список по датам) ── */
.sc-agenda { display: flex; flex-direction: column; }
.sc-agenda-row {
  display: flex; align-items: center; gap: 14px; width: 100%; text-align: left;
  padding: 12px 16px; border: none; background: none; cursor: pointer;
  border-bottom: 1px solid var(--color-outline-dim);
}
.sc-agenda-row:hover { background: var(--glass-hover-bg); }
.sc-agenda-date { flex-shrink: 0; width: 44px; display: flex; flex-direction: column; align-items: center; }
.sc-agenda-dnum { font-size: 18px; font-weight: 700; color: var(--color-text); }
.sc-agenda-row.today .sc-agenda-dnum {
  width: 30px; height: 30px; display: grid; place-items: center;
  background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-full);
}
.sc-agenda-dwd { font-size: 11px; color: var(--color-text-dim); text-transform: uppercase; }
.sc-agenda-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.sc-agenda-month { font-size: 12px; color: var(--color-text-dim); }
.sc-agenda-prev { font-size: 14px; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sc-agenda-empty { font-size: 13px; color: var(--color-text-dim); }
.sc-agenda-chev { flex-shrink: 0; color: var(--color-text-dim); }

.sc-daylist { display: flex; flex-direction: column; gap: 8px; }
.sc-dayrow { display: flex; align-items: center; gap: 14px; width: 100%; text-align: left; padding: 12px 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); background: var(--acrylic-card-bg); cursor: pointer; }
.sc-dayrow:hover { background: var(--glass-hover-bg); }
.sc-dayrow-time { flex-shrink: 0; min-width: 56px; font-size: 16px; font-weight: 700; color: var(--color-primary); font-variant-numeric: tabular-nums; }
.sc-dayrow-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.sc-dayrow-title { font-size: 15px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sc-dayrow-sub { font-size: 13px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sc-dayrow-flabel { font-weight: 600; color: var(--color-text); }
.sc-dayrow-chev { flex-shrink: 0; color: var(--color-text-dim); }

.sc-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 48px 16px; color: var(--color-text-dim); }
.sc-empty .material-symbols-outlined { font-size: 44px; }
.sc-empty p { margin: 0; }

.sc-overlay { position: absolute; inset: 0; display: grid; place-items: center; background: color-mix(in oklch, var(--color-surface) 50%, transparent); }
.spin { animation: scspin 1s linear infinite; font-size: 32px; color: var(--color-primary); }
@keyframes scspin { to { transform: rotate(360deg); } }

.sc-foot { flex: none; display: flex; align-items: center; justify-content: flex-end; padding: 10px 20px; border-top: 1px solid var(--color-outline-dim); background: var(--acrylic-card-bg); }
.sc-brand { font-size: 12px; font-weight: 700; color: var(--color-text-dim); }

.sc-btn { display: inline-flex; align-items: center; gap: 6px; height: 40px; padding: 0 16px; border: none; border-radius: var(--radius-full); background: var(--grad-primary); color: var(--color-on-primary); font-weight: 600; font-size: 14px; cursor: pointer; }
.sc-link-btn { border: none; background: none; cursor: pointer; color: var(--color-primary); font-weight: 600; font-size: 14px; }
.sc-export { display: flex; flex-direction: column; gap: 16px; }
.sc-export-period { margin: 0; font-size: 14px; color: var(--color-text); }
.sc-export-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.sc-export-title { font-size: 13px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.sc-export-bulk { display: flex; gap: 12px; }
.sc-export-fields { display: flex; flex-direction: column; gap: 2px; max-height: 320px; overflow-y: auto; }
.sc-export-row { display: flex; align-items: center; gap: 10px; padding: 9px 8px; border-radius: var(--radius-md); cursor: pointer; font-size: 14px; color: var(--color-text); }
.sc-export-row:hover { background: var(--glass-hover-bg); }
.sc-export-row .material-symbols-outlined { font-size: 20px; color: var(--color-text-dim); }
.sc-export-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

@media (max-width: 768px) {
  .sc-head { padding: 12px 14px; }
  .sc-title h1 { font-size: 18px; }
  .sc-body { padding: 0; }
}
</style>
