<template>
  <div class="cv split-view">
    <!-- ЛЕВАЯ ПАНЕЛЬ: список календарей -->
    <aside class="split-side">
      <div class="split-side-head">
        <span class="split-side-tile"><span class="material-symbols-outlined">calendar_month</span></span>
        <span class="split-side-title">Календари</span>
      </div>
      <div class="split-side-list">
        <div v-if="store.loadingList" class="split-side-note">Загрузка…</div>
        <div v-else-if="!store.calendars.length" class="split-side-note">Календари отсутствуют</div>
        <button
          v-for="c in store.calendars"
          :key="c.id"
          class="split-side-item"
          :class="{ active: c.id === store.selectedId }"
          @click="store.select(c.id)"
        >
          <span class="split-item-tile"><span class="material-symbols-outlined">event_note</span></span>
          <span class="split-side-name">{{ c.name }}</span>
        </button>
      </div>
    </aside>

    <!-- ПРАВАЯ ПАНЕЛЬ -->
    <section class="split-main">
      <!-- Мобайл: выбор календаря лентой чипов -->
      <div v-if="isMobile && store.calendars.length" class="cv-regstrip">
        <button
          v-for="c in store.calendars"
          :key="c.id"
          class="cv-regchip"
          :class="{ active: c.id === store.selectedId }"
          @click="store.select(c.id)"
        >{{ c.name }}</button>
      </div>

      <template v-if="store.selected">
        <!-- Тулбар -->
        <header class="cv-toolbar">
          <div class="cv-nav">
            <button class="cv-icon-btn" title="Назад" @click="store.step(-1)">
              <span class="material-symbols-outlined">chevron_left</span>
            </button>
            <button class="cv-today" @click="store.today()">Сегодня</button>
            <button class="cv-icon-btn" title="Вперёд" @click="store.step(1)">
              <span class="material-symbols-outlined">chevron_right</span>
            </button>
            <h2 class="cv-period">{{ periodLabel }}</h2>
          </div>

          <SearchField
            v-model="searchInput"
            placeholder="Поиск по записям…"
            hotkey
            @update:model-value="onSearch"
            @clear="clearSearch"
          />

          <div class="cv-viewseg">
            <button v-for="v in viewModes" :key="v.value" :class="{ active: store.view === v.value }" @click="store.setView(v.value)">
              {{ v.label }}
            </button>
          </div>

          <div class="cv-actions">
            <button class="cv-icon-btn" title="Внешние ссылки" @click="openShares">
              <span class="material-symbols-outlined">link</span>
            </button>
            <button class="cv-icon-btn" title="Экспорт в XLSX" @click="openExport">
              <span class="material-symbols-outlined">download</span>
            </button>
            <button class="btn-grad" @click="openCreate()">
              <span class="material-symbols-outlined">add</span>
              <span class="cv-btn-label">Запись</span>
            </button>
          </div>
        </header>

        <!-- Тело: месяц / неделя / день -->
        <div class="cv-body">
          <!-- Десктоп: месяц / неделя — сетка плиток дней -->
          <div v-if="!isMobile && store.view !== 'day'" ref="weekGridRef" class="cv-grid" :class="store.view">
            <!-- Шапка дней недели — только в месяце; в неделе день уже подписан в плитке. -->
            <template v-if="store.view === 'month'">
              <div v-for="(wd, i) in weekdays" :key="'h' + i" class="cv-wd">{{ wd }}</div>
            </template>
            <div
              v-for="day in gridDays"
              :key="dayKey(day)"
              class="cv-day"
              :class="{ dim: store.view === 'month' && !inCurrentMonth(day), today: isToday(day) }"
              @click="openDay(day)"
            >
              <div class="cv-day-head">
                <span class="cv-day-num">{{ day.getDate() }}</span>
                <span v-if="store.view === 'week'" class="cv-day-wd">{{ weekdayShort(day) }}</span>
                <span v-if="dayEntries(day).length" class="cv-day-count">{{ dayEntries(day).length }}</span>
              </div>
              <div class="cv-day-events">
                <div v-for="e in dayPreview(day)" :key="e.id" class="cv-event">
                  <span class="cv-event-time">{{ hhmm(e.event_at) }}</span>
                  <span class="cv-event-title">{{ entryTitle(store.selected, e) }}</span>
                </div>
                <div v-if="dayEntries(day).length > dayPreview(day).length" class="cv-event-more">
                  +{{ dayEntries(day).length - dayPreview(day).length }}
                </div>
              </div>
            </div>
          </div>

          <!-- Мобайл: месяц / неделя — список по датам с количеством записей -->
          <div v-else-if="isMobile && store.view !== 'day'" class="cv-agenda">
            <button
              v-for="day in agendaDays"
              :key="dayKey(day)"
              class="cv-agenda-row"
              :class="{ today: isToday(day) }"
              @click="openDay(day)"
            >
              <div class="cv-agenda-date">
                <span class="cv-agenda-dnum">{{ day.getDate() }}</span>
                <span class="cv-agenda-dwd">{{ weekdayShort(day) }}</span>
              </div>
              <div class="cv-agenda-body">
                <span class="cv-agenda-month">{{ agendaMonth(day) }}</span>
                <span v-if="dayEntries(day).length" class="cv-agenda-prev">{{ agendaPreview(day) }}</span>
                <span v-else class="cv-agenda-empty">Нет записей</span>
              </div>
              <span v-if="dayEntries(day).length" class="cv-day-count">{{ dayEntries(day).length }}</span>
              <span class="material-symbols-outlined cv-agenda-chev">chevron_right</span>
            </button>
          </div>

          <!-- День — хронологический список записей -->
          <div v-else class="cv-daylist">
            <div v-if="!dayEntries(store.cursor).length" class="cv-empty">
              <span class="material-symbols-outlined">event_busy</span>
              <p>На этот день записей нет</p>
              <button class="cv-btn-tonal" @click="openCreate(store.cursor)">
                <span class="material-symbols-outlined">add</span> Добавить запись
              </button>
            </div>
            <button
              v-for="e in dayEntries(store.cursor)"
              :key="e.id"
              class="cv-dayrow"
              @click="openEntry(e)"
            >
              <span class="cv-dayrow-time">{{ hhmm(e.event_at) }}</span>
              <span class="cv-dayrow-body">
                <span class="cv-dayrow-title">{{ entryTitle(store.selected, e) }}</span>
                <span v-for="cf in cardFields(store.selected, e)" :key="cf.field.id" class="cv-dayrow-sub">
                  <span class="cv-dayrow-flabel">{{ cf.field.label }}:</span> {{ cf.value }}
                </span>
              </span>
              <span class="material-symbols-outlined cv-dayrow-chev">chevron_right</span>
            </button>
          </div>

          <div v-if="store.loadingEntries" class="cv-overlay">
            <span class="material-symbols-outlined spin">progress_activity</span>
          </div>
        </div>
      </template>

      <!-- Календарь не выбран -->
      <EmptyState
        v-else
        class="split-empty"
        icon="calendar_month"
        tone="soft"
        :title="isMobile ? 'Выберите календарь сверху' : 'Выберите календарь слева'"
        subtitle="Выберите календарь в списке, чтобы просмотреть его записи"
      />
    </section>

    <AppFab
      :visible="isMobile && !!store.selected && fabVisible"
      icon="add"
      aria-label="Добавить запись"
      @click="openCreate()"
    />

    <CalendarDayDialog
      v-model="dayDialogOpen"
      :calendar="store.selected"
      :date="dayDialogDate"
      :entries="dayDialogEntries"
      @open-entry="openEntry"
      @add="openCreate(dayDialogDate)"
    />

    <CalendarEntryDialog
      v-model="dialogOpen"
      :calendar="store.selected"
      :entry="activeEntry"
      :default-date="defaultDate"
    />

    <!-- Внешние ссылки -->
    <AppDialog
      v-model="sharesOpen"
      title="Внешние ссылки" icon="link" size="md"
      :actions="[{ kind: 'cancel', label: 'Закрыть' }]"
      @cancel="sharesOpen = false"
    >
      <div class="cv-shares">
        <p class="cv-shares-note">
          По внешней ссылке любой человек (без входа в систему) сможет просматривать
          этот календарь, открывать карточки записей и выгружать данные — но не редактировать.
          Ссылку можно отозвать в любой момент.
        </p>
        <button class="btn-grad" :disabled="sharesBusy" @click="createShareLink">
          <span class="material-symbols-outlined">add_link</span> Создать ссылку
        </button>
        <div v-if="sharesLoading" class="cv-shares-empty">Загрузка…</div>
        <div v-else-if="!shares.length" class="cv-shares-empty">Ссылок пока нет</div>
        <ul v-else class="cv-shares-list">
          <li v-for="s in shares" :key="s.id" class="cv-share">
            <input class="cv-share-url" :value="shareUrl(s.code)" readonly @focus="$event.target.select()" />
            <button class="cv-icon-btn sm" title="Копировать" @click="copyShare(s.code)">
              <span class="material-symbols-outlined">content_copy</span>
            </button>
            <a class="cv-icon-btn sm" :href="shareUrl(s.code)" target="_blank" rel="noopener" title="Открыть">
              <span class="material-symbols-outlined">open_in_new</span>
            </a>
            <button class="cv-icon-btn sm danger" title="Отозвать" @click="revokeShareLink(s.id)">
              <span class="material-symbols-outlined">delete</span>
            </button>
          </li>
        </ul>
      </div>
    </AppDialog>

    <!-- Экспорт в XLSX -->
    <AppDialog
      v-model="exportOpen"
      title="Экспорт в XLSX" icon="download" size="md" :busy="exporting"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Экспортировать', icon: 'download' }]"
      @cancel="exportOpen = false" @confirm="doExport"
    >
      <div class="cv-export">
        <p class="cv-export-period">
          Будут выгружены записи за период: <b>{{ periodLabel }}</b>.
          Колонка «Дата и время» включается всегда.
        </p>
        <div class="cv-export-head">
          <span class="cv-export-title">Дополнительные поля</span>
          <div class="cv-export-bulk">
            <button class="cv-btn-text" @click="selectAllExport">Выбрать всё</button>
            <button class="cv-btn-text" @click="clearAllExport">Снять всё</button>
          </div>
        </div>
        <div class="cv-export-fields">
          <label v-for="f in exportableFields" :key="f.id" class="cv-export-row">
            <Checkbox :model-value="exportFields.has(f.id)" binary @update:model-value="toggleExportField(f.id)" />
            <span class="material-symbols-outlined">{{ fieldIcon(f.type) }}</span>
            <span class="cv-export-name">{{ f.label }}</span>
          </label>
          <p v-if="!exportableFields.length" class="cv-export-empty">
            В этом календаре нет дополнительных полей для экспорта (картинки и файлы не выгружаются).
          </p>
        </div>
      </div>
    </AppDialog>
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import Checkbox from 'primevue/checkbox'
import CalendarEntryDialog from '@/components/calendar/CalendarEntryDialog.vue'
import CalendarDayDialog from '@/components/calendar/CalendarDayDialog.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import AppFab from '@/components/common/AppFab.vue'
import { useFabOnScroll } from '@/composables/useFabOnScroll.js'
import { useCalendarsStore, dayKey } from '@/stores/calendars.js'
import { exportEntries, getShares, createShare, revokeShare } from '@/api/calendars.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { fieldIcon, isExportable, entryTitle, hhmm, cardFields } from '@/utils/calendarFields.js'

const store = useCalendarsStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()

const viewModes = [
  { value: 'month', label: 'Месяц' },
  { value: 'week', label: 'Неделя' },
  { value: 'day', label: 'День' },
]
const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

function addDays(d, n) { const x = new Date(d); x.setHours(0, 0, 0, 0); x.setDate(x.getDate() + n); return x }

// Дни видимой сетки (месяц — 42, неделя — 7).
const gridDays = computed(() => {
  const { from, to } = store.range
  const n = Math.round((to.getTime() - from.getTime()) / 86400000)
  const start = new Date(from); start.setHours(0, 0, 0, 0)
  return Array.from({ length: n }, (_, i) => addDays(start, i))
})

function dayEntries(day) { return store.entriesByDay[dayKey(day)] || [] }
function inCurrentMonth(day) { return day.getMonth() === store.cursor.getMonth() }
function weekdayShort(day) { return weekdays[(day.getDay() + 6) % 7] }

// Превью записей в плитке. Месяц — тесный (2). Неделя — столько, сколько влезает
// в высоту столбца; при переполнении оставляем строку под «+N».
const EVENT_H = 22   // .cv-grid.week .cv-event height (var --cv-event-h)
const EVENT_GAP = 3  // .cv-day-events gap
const weekGridRef = ref(null)
const weekColEventsH = ref(0)
let weekRO = null

function measureWeekColumn() {
  const el = weekGridRef.value
  if (!el || store.view !== 'week') return
  // Неделя — единственный ряд (grid-template-rows: 1fr), высота сетки = высота столбца.
  // Вычитаем паддинги плитки (6×2), gap между шапкой и событиями (4) и высоту шапки (24).
  weekColEventsH.value = Math.max(0, el.clientHeight - 12 - 4 - 24)
}

function weekMaxVisible() {
  const h = weekColEventsH.value
  if (h <= 0) return 4 // фолбэк до первого замера
  return Math.max(1, Math.floor((h + EVENT_GAP) / (EVENT_H + EVENT_GAP)))
}

function dayPreview(day) {
  const entries = dayEntries(day)
  if (store.view !== 'week') return entries.slice(0, 2)
  const max = weekMaxVisible()
  if (entries.length <= max) return entries
  return entries.slice(0, Math.max(0, max - 1)) // одна строка уйдёт под «+N»
}

// Мобильная агенда: месяц — все дни месяца курсора (1..N), неделя — её 7 дней.
const agendaDays = computed(() => {
  if (store.view === 'week') return gridDays.value
  const c = store.cursor
  const days = new Date(c.getFullYear(), c.getMonth() + 1, 0).getDate()
  return Array.from({ length: days }, (_, i) => new Date(c.getFullYear(), c.getMonth(), i + 1))
})
function agendaMonth(day) { return day.toLocaleDateString('ru-RU', { month: 'short' }) }
function agendaPreview(day) {
  return dayEntries(day).slice(0, 2)
    .map((e) => `${hhmm(e.event_at)} ${entryTitle(store.selected, e)}`.trim())
    .join(' · ')
}

const todayKey = dayKey(new Date())
function isToday(day) { return dayKey(day) === todayKey }

const periodLabel = computed(() => {
  const c = store.cursor
  if (store.view === 'day') {
    return c.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  }
  if (store.view === 'week') {
    const { from } = store.range
    const start = new Date(from)
    const end = addDays(start, 6)
    const opts = { day: 'numeric', month: 'short' }
    return `${start.toLocaleDateString('ru-RU', opts)} – ${end.toLocaleDateString('ru-RU', opts)} ${end.getFullYear()}`
  }
  return c.toLocaleDateString('ru-RU', { month: 'long', year: 'numeric' })
})

// ── Поиск ──
const searchInput = ref('')

// Мобильный FAB «Добавить запись»: прячется/появляется по прокрутке.
const { fabVisible } = useFabOnScroll()
let searchTimer = null
function onSearch() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(() => store.setSearch(searchInput.value.trim()), 300)
}
function clearSearch() { clearTimeout(searchTimer); searchInput.value = ''; store.setSearch('') }
watch(() => store.selectedId, () => { searchInput.value = '' })

// ── Модалка дня (список записей дня → создать/открыть/удалить) ──
const dayDialogOpen = ref(false)
const dayDialogDate = ref(null)
const dayDialogEntries = computed(() => (dayDialogDate.value ? dayEntries(dayDialogDate.value) : []))
function openDay(day) {
  dayDialogDate.value = new Date(day)
  dayDialogOpen.value = true
}

// ── Диалог записи (поверх модалки дня — стэк PrimeVue) ──
const dialogOpen = ref(false)
const activeEntry = ref(null)
const defaultDate = ref(null)
function openEntry(e) { activeEntry.value = e; defaultDate.value = null; dialogOpen.value = true }
function openCreate(day) {
  activeEntry.value = null
  const base = day ? new Date(day) : new Date(store.cursor)
  // Для новой записи на конкретный день — этот день, время 09:00 по умолчанию.
  if (day) base.setHours(9, 0, 0, 0)
  defaultDate.value = base
  dialogOpen.value = true
}

// ── Внешние ссылки ──
const sharesOpen = ref(false)
const shares = ref([])
const sharesLoading = ref(false)
const sharesBusy = ref(false)
function shareUrl(code) { return `${location.origin}/calendar/${code}` }
async function openShares() {
  sharesOpen.value = true
  sharesLoading.value = true
  try {
    const d = await getShares(store.selectedId)
    shares.value = d.shares ?? []
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить ссылки')
  } finally {
    sharesLoading.value = false
  }
}
async function createShareLink() {
  sharesBusy.value = true
  try {
    const s = await createShare(store.selectedId)
    shares.value.unshift(s)
  } catch (e) {
    notif.error(e?.message || 'Не удалось создать ссылку')
  } finally {
    sharesBusy.value = false
  }
}
async function revokeShareLink(id) {
  try {
    await revokeShare(store.selectedId, id)
    shares.value = shares.value.filter((s) => s.id !== id)
  } catch (e) {
    notif.error(e?.message || 'Не удалось отозвать ссылку')
  }
}
async function copyShare(code) {
  try {
    await navigator.clipboard.writeText(shareUrl(code))
    notif.success('Ссылка скопирована')
  } catch { /* ignore */ }
}

// ── Экспорт ──
const exportOpen = ref(false)
const exporting = ref(false)
const exportFields = ref(new Set())
const exportableFields = computed(() => (store.selected?.fields || []).filter((f) => isExportable(f.type)))
function openExport() {
  exportFields.value = new Set(exportableFields.value.map((f) => f.id))
  exportOpen.value = true
}
function toggleExportField(id) {
  const s = new Set(exportFields.value)
  s.has(id) ? s.delete(id) : s.add(id)
  exportFields.value = s
}
function selectAllExport() { exportFields.value = new Set(exportableFields.value.map((f) => f.id)) }
function clearAllExport() { exportFields.value = new Set() }
async function doExport() {
  exporting.value = true
  try {
    const { from, to } = store.range
    const params = {
      fields: [...exportFields.value],
      from: from.toISOString(),
      to: to.toISOString(),
      search: store.search,
    }
    const resp = await exportEntries(store.selectedId, params)
    if (!resp.ok) {
      let msg = 'Не удалось выгрузить'
      try { msg = (await resp.json()).message || msg } catch { /* ignore */ }
      throw new Error(msg)
    }
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${store.selected?.name || 'calendar'}.xlsx`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    exportOpen.value = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось выгрузить')
  } finally {
    exporting.value = false
  }
}

onMounted(() => {
  store.fetchCalendars()
  weekRO = new ResizeObserver(() => measureWeekColumn())
  if (weekGridRef.value) weekRO.observe(weekGridRef.value)
})
onBeforeUnmount(() => { weekRO?.disconnect(); weekRO = null })

// Грид появляется/исчезает при смене вида и устройства — переподключаем observer
// и пересчитываем после рендера.
watch([() => store.view, isMobile, () => store.selectedId], () => {
  nextTick(() => {
    if (weekRO && weekGridRef.value) { weekRO.disconnect(); weekRO.observe(weekGridRef.value) }
    measureWeekColumn()
  })
})
// После загрузки записей высота шапки могла измениться — пересчитываем.
watch(() => store.loadingEntries, () => nextTick(measureWeekColumn))
</script>

<style scoped>
/* Каркас (стеклянные панели, раскладка, мобильное скрытие левой панели) —
   глобальный паттерн .split-* (main.css). Здесь — только внутренности
   правой панели. */
.cv-toolbar {
  flex-shrink: 0; display: flex; align-items: center; gap: 12px; flex-wrap: wrap;
  padding: 12px 16px; border-bottom: 1px solid var(--color-outline-dim);
}
.cv-nav { display: flex; align-items: center; gap: 8px; }
.cv-period { margin: 0 0 0 6px; font-size: 17px; font-weight: 700; color: var(--color-text); text-transform: capitalize; white-space: nowrap; }
.cv-today {
  height: 36px; padding: 0 14px; border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full); background: var(--acrylic-card-bg);
  color: var(--color-text); font-weight: 600; font-size: 13px; cursor: pointer;
}
.cv-today:hover { background: var(--color-surface-high); }

/* Сегмент вида — как режимы периода в статистике (StatsPeriodControl):
   мягкий контейнер-пилюля, активный пункт — стеклянная пилюля с primary. */
.cv-viewseg {
  display: inline-flex; gap: 2px; padding: 4px;
  background: var(--color-surface-high); border-radius: var(--radius-full);
}
.cv-viewseg button {
  min-height: 36px; padding: 8px 14px; border: none; background: transparent;
  border-radius: var(--radius-full); color: var(--color-text-dim); cursor: pointer;
  font-weight: 600; font-size: 13px;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.cv-viewseg button:hover:not(.active) { color: var(--color-text); }
.cv-viewseg button.active {
  background: var(--acrylic-card-bg); color: var(--color-primary);
  font-weight: 700; box-shadow: var(--shadow-sm);
}

.cv-actions { display: flex; align-items: center; gap: 8px; }
.cv-icon-btn {
  width: 38px; height: 38px; display: grid; place-items: center;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full);
  background: var(--acrylic-card-bg); color: var(--color-text-dim); cursor: pointer;
}
.cv-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.cv-icon-btn.sm { width: 34px; height: 34px; flex-shrink: 0; }
.cv-icon-btn.sm .material-symbols-outlined { font-size: 18px; }
.cv-icon-btn.danger { color: var(--color-error); }

/* ── Тело ── */
.cv-body { position: relative; flex: 1; min-height: 0; overflow: auto; }

/* Сетка месяца/недели */
.cv-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px; background: var(--color-outline-dim); min-height: 100%; }
/* Месяц: шапка дней недели по высоте контента, 6 недель делят остаток поровну.
   Неделя: один ряд плиток на всю высоту (шапка не дублируется — день подписан в плитке). */
.cv-grid.month { grid-template-rows: auto repeat(6, 1fr); }
.cv-grid.week { grid-template-rows: 1fr; }
.cv-wd {
  /* Sticky-шапка: записи прокручиваются под ней — полный акрил */
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  padding: 8px 10px; text-align: center;
  font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase;
  position: sticky; top: 0; z-index: 1;
}
.cv-day {
  background: var(--acrylic-card-bg); min-height: 104px; padding: 6px;
  display: flex; flex-direction: column; gap: 4px; cursor: pointer; overflow: hidden;
}
.cv-grid.week .cv-day { min-height: 0; }
.cv-day:hover { background: var(--color-surface-high); }
.cv-day.dim { background: var(--color-surface-low); }
.cv-day.dim .cv-day-num { color: var(--color-text-dim); opacity: 0.6; }
.cv-day-head { display: flex; align-items: center; justify-content: space-between; }
.cv-day-num { font-size: 13px; font-weight: 700; color: var(--color-text); width: 24px; height: 24px; display: grid; place-items: center; }
.cv-day.today .cv-day-num { background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-full); }
.cv-day-wd { font-size: 11px; color: var(--color-text-dim); text-transform: uppercase; }
.cv-day-count {
  flex-shrink: 0; min-width: 18px; height: 18px; padding: 0 5px;
  display: inline-flex; align-items: center; justify-content: center;
  border-radius: var(--radius-full); background: var(--color-primary);
  color: var(--color-on-primary); font-size: 11px; font-weight: 700;
}
.cv-day-events { display: flex; flex-direction: column; gap: 3px; min-height: 0; }
.cv-event {
  display: flex; align-items: baseline; gap: 6px; width: 100%; text-align: left;
  padding: 3px 6px; border: none; border-radius: var(--radius-sm);
  background: var(--color-primary-container); color: var(--color-on-primary-container);
  font-size: 12px; overflow: hidden;
}
/* В неделе высота строки фиксирована — по ней считаем, сколько событий влезает в столбец. */
.cv-grid.week .cv-event { height: var(--cv-event-h, 22px); box-sizing: border-box; }
.cv-event-time { flex-shrink: 0; font-weight: 700; font-variant-numeric: tabular-nums; }
.cv-event-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cv-event-more { font-size: 11px; font-weight: 600; color: var(--color-text-dim); padding-left: 6px; }

/* ── Мобильная агенда (список по датам) ── */
.cv-agenda { display: flex; flex-direction: column; }
.cv-agenda-row {
  display: flex; align-items: center; gap: 14px; width: 100%; text-align: left;
  padding: 12px 16px; border: none; background: none; cursor: pointer;
  border-bottom: 1px solid var(--color-outline-dim);
}
.cv-agenda-row:hover { background: var(--color-surface-high); }
.cv-agenda-date {
  flex-shrink: 0; width: 44px; display: flex; flex-direction: column; align-items: center;
}
.cv-agenda-dnum { font-size: 18px; font-weight: 700; color: var(--color-text); }
.cv-agenda-row.today .cv-agenda-dnum {
  width: 30px; height: 30px; display: grid; place-items: center;
  background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-full);
}
.cv-agenda-dwd { font-size: 11px; color: var(--color-text-dim); text-transform: uppercase; }
.cv-agenda-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.cv-agenda-month { font-size: 12px; color: var(--color-text-dim); }
.cv-agenda-prev { font-size: 14px; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cv-agenda-empty { font-size: 13px; color: var(--color-text-dim); }
.cv-agenda-chev { flex-shrink: 0; color: var(--color-text-dim); }

/* Режим «День» */
.cv-daylist { display: flex; flex-direction: column; gap: 8px; padding: 16px; }
.cv-dayrow {
  display: flex; align-items: center; gap: 14px; width: 100%; text-align: left;
  padding: 12px 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg); cursor: pointer;
}
.cv-dayrow:hover { background: var(--color-surface-high); border-color: var(--color-outline); }
.cv-dayrow-time {
  flex-shrink: 0; min-width: 56px; font-size: 16px; font-weight: 700; color: var(--color-primary);
  font-variant-numeric: tabular-nums;
}
.cv-dayrow-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.cv-dayrow-title { font-size: 15px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cv-dayrow-sub { font-size: 13px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cv-dayrow-flabel { font-weight: 600; color: var(--color-text); }
.cv-dayrow-chev { flex-shrink: 0; color: var(--color-text-dim); }

.cv-empty {
  display: flex; flex-direction: column; align-items: center; justify-content: center;
  gap: 10px; padding: 48px 16px; color: var(--color-text-dim);
}
.cv-empty .material-symbols-outlined { font-size: 44px; }
.cv-empty p { margin: 0; }

.cv-overlay { position: absolute; inset: 0; display: grid; place-items: center; background: color-mix(in oklch, var(--color-surface) 50%, transparent); }

/* ── Кнопки ── */
.cv-btn-tonal {
  display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px;
  border: 1px solid var(--acrylic-border); border-radius: var(--radius-full);
  /* Матовое стекло единого стиля вместо сплошного primary-container. */
  background: var(--acrylic-card-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  color: var(--color-primary);
  font-weight: 600; font-size: 14px; cursor: pointer;
  transition: background 0.15s;
}
.cv-btn-tonal:hover { background: var(--color-surface-high); }
.cv-btn-text { border: none; background: none; cursor: pointer; color: var(--color-primary); font-weight: 600; font-size: 14px; }
.spin { animation: cvspin 1s linear infinite; font-size: 32px; color: var(--color-primary); }
@keyframes cvspin { to { transform: rotate(360deg); } }

/* ── Внешние ссылки / экспорт ── */
.cv-shares { display: flex; flex-direction: column; gap: 14px; }
.cv-shares-note { margin: 0; font-size: 13px; color: var(--color-text-dim); line-height: 1.5; }
.cv-shares-empty { padding: 16px; text-align: center; color: var(--color-text-dim); font-size: 14px; }
.cv-shares-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.cv-share { display: flex; align-items: center; gap: 6px; }
.cv-share-url {
  flex: 1; min-width: 0; height: 38px; padding: 0 12px;
  border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md);
  background: var(--color-surface-low); color: var(--color-text); font-size: 13px;
}
.cv-export { display: flex; flex-direction: column; gap: 16px; }
.cv-export-period { margin: 0; font-size: 14px; color: var(--color-text); }
.cv-export-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.cv-export-title { font-size: 13px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; }
.cv-export-bulk { display: flex; gap: 12px; }
.cv-export-fields { display: flex; flex-direction: column; gap: 2px; max-height: 320px; overflow-y: auto; }
.cv-export-row { display: flex; align-items: center; gap: 10px; padding: 9px 8px; border-radius: var(--radius-md); cursor: pointer; font-size: 14px; color: var(--color-text); }
.cv-export-row:hover { background: var(--color-surface-high); }
.cv-export-row .material-symbols-outlined { font-size: 20px; color: var(--color-text-dim); }
.cv-export-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.cv-export-empty { margin: 0; color: var(--color-text-dim); font-size: 14px; }

/* ── Мобайл ── */
.cv-regstrip {
  flex: none; display: flex; gap: 8px; padding: 10px 12px;
  min-width: 0; max-width: 100%; overflow-x: auto;
  border-bottom: 1px solid var(--color-outline-dim); -webkit-overflow-scrolling: touch;
  touch-action: pan-x;
  scrollbar-width: none;
}
.cv-regstrip::-webkit-scrollbar { display: none; }
.cv-regchip {
  flex: 0 0 auto; padding: 8px 14px; border-radius: var(--radius-full);
  border: 1px solid var(--color-outline-dim); background: var(--acrylic-card-bg);
  color: var(--color-text-dim); font-size: 14px; font-weight: 600; cursor: pointer; white-space: nowrap;
  max-width: 60vw; overflow: hidden; text-overflow: ellipsis;
}
.cv-regchip.active { background: var(--color-primary); color: var(--color-on-primary); border-color: transparent; }

@media (max-width: 768px) {
  /* Скрытие левой панели и разворот правой — в глобальном .split-* */
  /* Резерв под нижнюю навигацию (64px) + 12px воздуха: список по датам
     скроллится под стекло, последние записи не прячутся за навигацией. */
  .cv-body { padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px)); }
  /* Компактный верх в ДВЕ строки: навигация + действия («Запись» — иконка),
     ниже — переключатель вида и поиск в одной строке. */
  .cv-toolbar { flex-wrap: wrap; gap: 8px; padding: 8px 12px; }
  /* flex-basis auto (не 0!): при переносе строк nav не схлопывается в ноль,
     иначе его кнопки вылезают поверх соседних. Кнопки не сжимаются. */
  .cv-nav { order: 1; flex: 1 1 auto; min-width: 0; gap: 6px; }
  .cv-nav .cv-icon-btn,
  .cv-today { flex-shrink: 0; }
  .cv-period { flex: 1; min-width: 0; font-size: 15px; margin-left: 2px; overflow: hidden; text-overflow: ellipsis; }
  .cv-today { height: 34px; padding: 0 10px; }
  .cv-icon-btn { width: 36px; height: 36px; }
  .cv-actions { order: 2; flex-shrink: 0; gap: 6px; }
  /* Создание записи на мобильном — плавающий FAB, кнопка тулбара не нужна. */
  .cv-actions .btn-grad { display: none; }
  .cv-viewseg { order: 3; flex-shrink: 0; }
  .cv-viewseg button { min-height: 32px; padding: 6px 12px; }
  .cv-toolbar :deep(.search-field) { order: 4; flex: 1; min-width: 130px; }
  .cv-regstrip { padding: 8px 12px 6px; gap: 6px; }
  .cv-regchip { padding: 6px 12px; font-size: 13px; }
  /* На мобайле месяц/неделя — список по датам (cv-agenda), сетка не используется. */
}
</style>
