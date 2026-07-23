<template>
  <!-- Тот же каркас admin-page, что у Портала/Сотрудников: полная ширина,
       единая стеклянная панель со статичной шапкой и внутренним скроллом
       (у .admin-body есть мобильный отступ под нижнюю навигацию). -->
  <div class="admin-page ea" :class="{ 'has-panel': !isMobile }">
    <div class="hub-panel">
      <header class="admin-sticky ea-sticky">
        <!-- Шапка -->
        <div class="ea-head">
          <button class="ea-back btn-glass" @click="goBack" aria-label="Назад">
            <span class="material-symbols-outlined">arrow_back</span>
          </button>
          <div class="ea-who">
            <img class="ea-avatar" :src="avatarUrl" :alt="employeeName" />
            <div class="ea-who-text">
              <span class="ea-eyebrow">{{ isSelf ? 'Моя активность' : 'Активность сотрудника' }}</span>
              <h1 class="ea-name">{{ employeeName }}</h1>
            </div>
          </div>
          <button class="ea-export btn-grad" :disabled="exporting || loading" @click="exportDocx">
            <span class="material-symbols-outlined">{{ exporting ? 'hourglass_top' : 'download' }}</span>
            <span class="hide-narrow">Скачать .docx</span>
          </button>
        </div>

        <!-- Период -->
        <div class="ea-periods">
          <button
            v-for="p in periods" :key="p.key"
            class="ea-chip" :class="{ active: activePeriod === p.key }"
            @click="setPeriod(p.key)"
          >{{ p.label }}</button>
          <button
            class="ea-chip" :class="{ active: activePeriod === 'custom' }"
            @click="setPeriod('custom')"
          >Произвольный</button>
          <DateRangePicker
            v-if="activePeriod === 'custom'"
            v-model="customRange"
            class="ea-range"
          />
        </div>

        <!-- Подвкладки -->
        <div class="ea-tabs">
          <button class="ea-tab" :class="{ active: tab === 'overview' }" @click="tab = 'overview'">
            <span class="material-symbols-outlined">insights</span> Обзор
          </button>
          <button class="ea-tab" :class="{ active: tab === 'feed' }" @click="tab = 'feed'">
            <span class="material-symbols-outlined">history</span> Полная активность
          </button>
        </div>
      </header>

      <div class="admin-body">
        <div v-if="loading" class="ea-loading"><BrandLoader :size="64" /></div>

        <!-- ОБЗОР -->
        <template v-else-if="tab === 'overview' && data">
          <section class="ea-kpis">
        <article v-for="k in kpis" :key="k.label" class="ea-kpi">
          <div class="ea-kpi-text">
            <span class="ea-kpi-val">{{ k.value }}</span>
            <span class="ea-kpi-label">{{ k.label }}</span>
          </div>
          <span class="ea-kpi-ico material-symbols-outlined" :data-tone="k.tone">{{ k.icon }}</span>
        </article>
      </section>

      <div class="ea-grid">
        <!-- Типы работ -->
        <section class="ea-card">
          <h2 class="ea-card-title">Часы по типам работ</h2>
          <div v-if="data.by_unit_types.length" class="ea-bars">
            <div v-for="t in data.by_unit_types" :key="t.type_id" class="ea-bar-row">
              <span class="ea-bar-label" :title="t.name">{{ t.name }}</span>
              <div class="ea-bar-track">
                <div class="ea-bar-fill" :style="{ width: pct(t.hours, maxTypeHours) + '%' }" />
              </div>
              <span class="ea-bar-val">{{ t.hours }} ч</span>
            </div>
          </div>
          <p v-else class="ea-empty">Нет данных за период</p>
        </section>

        <!-- Недельная динамика -->
        <section class="ea-card">
          <h2 class="ea-card-title">Недельная динамика</h2>
          <div v-if="data.weekly_trend.length" class="ea-cols">
            <div v-for="w in data.weekly_trend" :key="w.week" class="ea-col" :title="`${w.week}: ${w.hours} ч, закрыто ${w.closed}`">
              <div class="ea-col-track">
                <div class="ea-col-fill" :style="{ height: pct(w.hours, maxWeekHours) + '%' }" />
              </div>
              <span class="ea-col-cap">{{ shortWeek(w.week) }}</span>
            </div>
          </div>
          <p v-else class="ea-empty">Нет данных за период</p>
        </section>

        <!-- По дням недели -->
        <section class="ea-card">
          <h2 class="ea-card-title">По дням недели</h2>
          <div class="ea-cols">
            <div v-for="(h, i) in weekdayHours" :key="i" class="ea-col" :title="`${weekdayNames[i]}: ${h} ч`">
              <div class="ea-col-track">
                <div class="ea-col-fill" :style="{ height: pct(h, maxWeekdayHours) + '%' }" />
              </div>
              <span class="ea-col-cap">{{ weekdayShort[i] }}</span>
            </div>
          </div>
        </section>

        <!-- Активность по часам -->
        <section class="ea-card ea-card-wide">
          <h2 class="ea-card-title">Когда работает</h2>
          <p class="ea-card-sub">В какое время суток шла работа: каждый столбец — час дня (0–23), чем ярче, тем больше часов отработано в этот час за период.</p>
          <div class="ea-hours">
            <div
              v-for="(h, i) in hourHours" :key="i"
              class="ea-hour-cell"
              :style="{ background: heat(h, maxHourHours) }"
              :title="`${String(i).padStart(2, '0')}:00 — ${h} ч`"
            />
          </div>
          <div class="ea-hours-axis">
            <span v-for="t in [0, 6, 12, 18, 23]" :key="t" class="ea-hours-tick" :style="{ gridColumnStart: t + 1 }">{{ String(t).padStart(2, '0') }}:00</span>
          </div>
          <div class="ea-legend">
            <span class="ea-legend-cap">меньше</span>
            <span class="ea-legend-bar" />
            <span class="ea-legend-cap">больше</span>
          </div>
        </section>
      </div>
    </template>

    <!-- ПОЛНАЯ АКТИВНОСТЬ -->
    <section v-else-if="tab === 'feed'" class="ea-feed">
      <div v-if="feedLoading" class="ea-loading"><BrandLoader :size="48" /></div>
      <template v-else>
        <ul v-if="feed.items.length" class="ea-events">
          <li v-for="(e, i) in feed.items" :key="i" class="ea-event">
            <span class="ea-event-ico material-symbols-outlined" :data-type="e.type">{{ eventIcon(e.type) }}</span>
            <div class="ea-event-body">
              <span class="ea-event-title">
                {{ eventLabel(e.type) }}
                <span v-if="e.task_name" class="ea-event-task">«{{ e.task_name }}»</span>
              </span>
              <span v-if="e.detail" class="ea-event-detail">{{ e.detail }}</span>
            </div>
            <time class="ea-event-time">{{ fmtDateTime(e.at) }}</time>
          </li>
        </ul>
        <p v-else class="ea-empty">Событий за период нет</p>

        <div v-if="feedPages > 1" class="ea-pager">
          <button class="btn-glass" :disabled="feedPage <= 1" @click="loadFeed(feedPage - 1)">
            <span class="material-symbols-outlined">chevron_left</span>
          </button>
          <span class="ea-pager-info">{{ feedPage }} / {{ feedPages }}</span>
          <button class="btn-glass" :disabled="feedPage >= feedPages" @click="loadFeed(feedPage + 1)">
            <span class="material-symbols-outlined">chevron_right</span>
          </button>
        </div>
      </template>
        </section>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import * as statsApi from '@/api/stats.js'
import { getStatsEmployees } from '@/api/stats.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import BrandLoader from '@/components/common/BrandLoader.vue'
import DateRangePicker from '@/components/common/DateRangePicker.vue'

const props = defineProps({ userId: { type: [String, Number], required: true } })

const router = useRouter()
const notif = useNotificationsStore()
const authStore = useAuthStore()
const { isMobile } = useBreakpoint()

const uid = computed(() => Number(props.userId))
// Своя активность («Моя активность») vs чужая (доступна руководителю компании).
const isSelf = computed(() => uid.value === Number(authStore.userId))
const employeeName = ref('Сотрудник')
const avatarUrl = computed(() => {
  if (isSelf.value && authStore.user?.avatar_path) return `/uploads/${authStore.user.avatar_path}`
  return `/api/users/${uid.value}/identicon`
})

const tab = ref('overview')
const activePeriod = ref('month')
// Произвольный диапазон: [Date, Date] из DateRangePicker (активен при activePeriod === 'custom').
const customRange = ref(null)
const loading = ref(true)
const exporting = ref(false)
const data = ref(null)

const periods = [
  { key: 'week', label: 'Неделя', days: 7 },
  { key: 'month', label: 'Месяц', days: 30 },
  { key: 'quarter', label: 'Квартал', days: 90 },
  { key: 'year', label: 'Год', days: 365 },
]

const weekdayNames = ['Воскресенье', 'Понедельник', 'Вторник', 'Среда', 'Четверг', 'Пятница', 'Суббота']
const weekdayShort = ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб']

// Локальная календарная дата YYYY-MM-DD (без сдвига через UTC — иначе
// выбранный в пикере день «съезжает» на предыдущий в плюсовых поясах).
function isoLocal(d) {
  const p = (n) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

// Диапазон дат текущего периода {from, to} в формате YYYY-MM-DD.
// null — выбран «Произвольный», но обе даты ещё не заданы (грузить нечего).
function currentRange() {
  if (activePeriod.value === 'custom') {
    const r = customRange.value
    if (!r || !r[0] || !r[1]) return null
    return { from: isoLocal(r[0]), to: isoLocal(r[1]) }
  }
  const days = periods.find((p) => p.key === activePeriod.value)?.days ?? 30
  const to = new Date()
  const from = new Date()
  from.setDate(from.getDate() - (days - 1))
  return { from: isoLocal(from), to: isoLocal(to) }
}

const kpis = computed(() => {
  const s = data.value?.summary
  if (!s) return []
  return [
    { label: 'Отработано часов', value: s.worked_hours, icon: 'schedule', tone: 'primary' },
    { label: 'Закрыто задач', value: s.tasks_closed, icon: 'task_alt', tone: 'success' },
    { label: 'Создано задач', value: s.tasks_created, icon: 'add_task', tone: 'secondary' },
    { label: 'Комментариев', value: s.comments, icon: 'forum', tone: 'tertiary' },
    { label: 'Активных дней', value: s.active_days, icon: 'event_available', tone: 'primary' },
    { label: 'Юнитов', value: s.units_count, icon: 'timer', tone: 'secondary' },
    { label: 'Часов на задачу', value: s.avg_hours_per_closed, icon: 'speed', tone: 'tertiary' },
    { label: 'Ср. время закрытия, ч', value: s.avg_cycle_hours, icon: 'hourglass_bottom', tone: 'success' },
  ]
})

// Разрезы, дополненные до полного набора (7 дней / 24 часа).
const weekdayHours = computed(() => {
  const arr = Array(7).fill(0)
  data.value?.by_weekday?.forEach((w) => { arr[w.weekday] = w.hours })
  return arr
})
const hourHours = computed(() => {
  const arr = Array(24).fill(0)
  data.value?.by_hour?.forEach((h) => { arr[h.hour] = h.hours })
  return arr
})
const maxTypeHours = computed(() => Math.max(1, ...(data.value?.by_unit_types || []).map((t) => t.hours)))
const maxWeekHours = computed(() => Math.max(1, ...(data.value?.weekly_trend || []).map((w) => w.hours)))
const maxWeekdayHours = computed(() => Math.max(1, ...weekdayHours.value))
const maxHourHours = computed(() => Math.max(1, ...hourHours.value))

const pct = (v, max) => Math.round((v / max) * 100)
const heat = (v, max) => v <= 0
  ? 'var(--color-surface-low)'
  : `color-mix(in oklch, var(--color-primary) ${Math.round(15 + (v / max) * 70)}%, transparent)`

function shortWeek(w) { return w.replace(/^\d{4}-/, '') }

async function load() {
  const r = currentRange()
  if (!r) return // ждём выбора обеих дат произвольного периода
  loading.value = true
  try {
    data.value = await statsApi.getEmployeeActivity(uid.value, r.from, r.to)
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить активность')
  } finally {
    loading.value = false
  }
}

function setPeriod(key) {
  if (activePeriod.value === key) return
  activePeriod.value = key
  // «Произвольный» без выбранных дат — только показываем пикер, грузим по выбору.
  if (key === 'custom' && !currentRange()) return
  load()
  if (tab.value === 'feed') loadFeed(1)
}

// Выбор обеих дат произвольного периода — перезагрузка обзора и ленты.
watch(customRange, () => {
  if (activePeriod.value !== 'custom' || !currentRange()) return
  load()
  if (tab.value === 'feed') loadFeed(1)
})

// ── Лента ──
const feed = ref({ items: [], total: 0, per_page: 30 })
const feedPage = ref(1)
const feedLoading = ref(false)
const feedPages = computed(() => Math.max(1, Math.ceil((feed.value.total || 0) / (feed.value.per_page || 30))))

async function loadFeed(page) {
  const r = currentRange()
  if (!r) return
  feedLoading.value = true
  feedPage.value = page
  try {
    feed.value = await statsApi.getEmployeeActivityFeed(uid.value, { from: r.from, to: r.to, page, perPage: 30 })
  } catch (e) {
    notif.error(e?.message || 'Не удалось загрузить ленту')
  } finally {
    feedLoading.value = false
  }
}

watch(tab, (t) => { if (t === 'feed' && !feed.value.items.length) loadFeed(1) })

const eventIcons = {
  unit_started: 'play_circle', unit_stopped: 'stop_circle',
  task_created: 'add_task', task_closed: 'task_alt', comment: 'forum',
}
const eventLabels = {
  unit_started: 'Начал юнит', unit_stopped: 'Завершил юнит',
  task_created: 'Создал задачу', task_closed: 'Закрыл задачу', comment: 'Оставил комментарий',
}
const eventIcon = (t) => eventIcons[t] || 'bolt'
const eventLabel = (t) => eventLabels[t] || t

function fmtDateTime(v) {
  const d = new Date(v)
  return d.toLocaleString('ru-RU', { day: '2-digit', month: '2-digit', year: '2-digit', hour: '2-digit', minute: '2-digit' })
}

async function exportDocx() {
  const r = currentRange()
  if (!r) { notif.error('Выберите обе даты периода'); return }
  exporting.value = true
  try {
    // apiRequest({blob:true}) отдаёт Response — забираем из него сам Blob.
    const resp = await statsApi.exportEmployeeActivity(uid.value, r.from, r.to)
    const blob = resp instanceof Blob ? resp : await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `Активность — ${employeeName.value}.docx`
    a.click()
    URL.revokeObjectURL(url)
  } catch (e) {
    notif.error(e?.message || 'Не удалось выгрузить отчёт')
  } finally {
    exporting.value = false
  }
}

function goBack() {
  if (window.history.length > 1) router.back()
  else router.push(isSelf.value ? '/profile' : '/employees')
}

async function resolveName() {
  // Своё имя — из профиля (список сотрудников доступен только менеджеру+).
  if (isSelf.value) {
    if (authStore.user?.fio) employeeName.value = authStore.user.fio
    return
  }
  try {
    const list = await getStatsEmployees()
    const me = (list || []).find((e) => e.id === uid.value)
    if (me?.fio) employeeName.value = me.fio
  } catch { /* имя не критично */ }
}

onMounted(() => { resolveName(); load() })
</script>

<style scoped>
/* Каркас — admin-page + hub-panel (как Портал/Сотрудники): полная ширина,
   стеклянная панель, статичная шапка, внутренний скролл .admin-body. */
/* Шапка без своей акриловой подложки (на мобиле она давала обрезанную плашку):
   контент раздела прокручивается в .admin-body ниже, фон шапке не нужен. */
.ea-sticky { gap: 14px; background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; }
.ea-sticky::after { display: none; }

/* Шапка — лёгкая строка внутри панели. */
.ea-head { display: flex; align-items: center; gap: 12px; }
.ea-back { width: 38px; height: 38px; padding: 0; display: grid; place-items: center; border-radius: var(--radius-full); flex-shrink: 0; }
.ea-who { display: flex; align-items: center; gap: 12px; flex: 1; min-width: 0; }
.ea-avatar { width: 42px; height: 42px; border-radius: 50%; object-fit: cover; flex-shrink: 0; }
.ea-who-text { display: flex; flex-direction: column; min-width: 0; }
.ea-eyebrow { font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.06em; color: var(--color-text-dim); }
.ea-name { margin: 0; font-size: 19px; font-weight: 800; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ea-export { display: inline-flex; align-items: center; gap: 8px; flex-shrink: 0; }
.ea-export .material-symbols-outlined { font-size: 19px; }

.ea-periods { display: flex; flex-wrap: wrap; gap: 8px; align-items: center; }
.ea-range { margin-left: 4px; }
.ea-chip {
  padding: 8px 16px; border-radius: var(--radius-full); border: 1px solid var(--color-outline-dim);
  background: var(--color-surface); color: var(--color-text-dim); font: inherit; font-weight: 600; cursor: pointer;
  transition: all .12s;
}
.ea-chip.active { background: var(--color-primary); border-color: var(--color-primary); color: var(--color-on-primary); }
/* Активное/выбранное состояние показано цветом — гасим залипающую фокус-рамку
   (её низ обрезал overflow панели). Для клавиатуры — inset-ринг, он не режется. */
.ea-chip:focus, .ea-tab:focus { outline: none; }
.ea-chip:focus-visible, .ea-tab:focus-visible { outline: 2px solid var(--color-primary); outline-offset: -2px; }

.ea-tabs { display: flex; gap: 6px; border-bottom: 1px solid var(--color-outline-dim); }
.ea-tab {
  display: inline-flex; align-items: center; gap: 6px; padding: 10px 14px; border: none; background: transparent;
  color: var(--color-text-dim); font: inherit; font-weight: 700; cursor: pointer; border-bottom: 2px solid transparent; margin-bottom: -1px;
}
.ea-tab.active { color: var(--color-primary); border-bottom-color: var(--color-primary); }
.ea-tab .material-symbols-outlined { font-size: 19px; }

.ea-loading { display: flex; justify-content: center; padding: 48px; }

/* KPI — число слева, иконка справа; лёгкий внутренний тайл. */
.ea-kpis { display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 12px; }
.ea-kpi {
  display: flex; flex-direction: row; align-items: center; justify-content: space-between; gap: 12px; padding: 16px;
  background: var(--color-surface); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg);
}
.ea-kpi-text { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.ea-kpi-ico { font-size: 24px; width: 44px; height: 44px; border-radius: var(--radius-md); display: grid; place-items: center; flex-shrink: 0; }
.ea-kpi-ico[data-tone="primary"] { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.ea-kpi-ico[data-tone="secondary"] { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.ea-kpi-ico[data-tone="tertiary"] { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.ea-kpi-ico[data-tone="success"] { background: color-mix(in oklch, var(--color-success) 20%, transparent); color: var(--color-success); }
.ea-kpi-val { font-size: 26px; font-weight: 800; color: var(--color-text); line-height: 1; }
.ea-kpi-label { font-size: 12px; color: var(--color-text-dim); }

/* Карточки-графики — лёгкие внутренние тайлы единой карточки. */
.ea-grid { display: grid; grid-template-columns: repeat(2, 1fr); gap: 16px; }
.ea-card { padding: 18px; background: var(--color-surface); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); }
.ea-card-wide { grid-column: 1 / -1; }
.ea-card-title { margin: 0 0 14px; font-size: 15px; font-weight: 700; color: var(--color-text); }
.ea-empty { margin: 12px 0; color: var(--color-text-dim); font-size: 13px; text-align: center; }

/* Горизонтальные бары (типы работ) */
.ea-bars { display: flex; flex-direction: column; gap: 10px; }
.ea-bar-row { display: grid; grid-template-columns: minmax(80px, 30%) 1fr auto; align-items: center; gap: 10px; }
.ea-bar-label { font-size: 13px; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ea-bar-track { height: 12px; background: var(--color-surface-low); border-radius: var(--radius-full); overflow: hidden; }
.ea-bar-fill { height: 100%; background: linear-gradient(90deg, var(--color-primary), var(--color-tertiary)); border-radius: var(--radius-full); min-width: 3px; }
.ea-bar-val { font-size: 12px; font-weight: 700; color: var(--color-text-dim); white-space: nowrap; }

/* Вертикальные колонки (недели / дни недели) */
.ea-cols { display: flex; align-items: flex-end; gap: 6px; height: 140px; }
.ea-col { flex: 1; display: flex; flex-direction: column; align-items: center; gap: 6px; height: 100%; min-width: 0; }
.ea-col-track { flex: 1; width: 100%; display: flex; align-items: flex-end; }
.ea-col-fill { width: 100%; background: linear-gradient(180deg, var(--color-primary), color-mix(in oklch, var(--color-primary) 55%, var(--color-tertiary))); border-radius: var(--radius-sm) var(--radius-sm) 0 0; min-height: 2px; transition: height .2s; }
.ea-col-cap { font-size: 10px; color: var(--color-text-dim); white-space: nowrap; }

/* Часовая тепловая карта: 24 столбца = часы суток, насыщенность = часы работы. */
.ea-card-sub { margin: -8px 0 14px; font-size: 12.5px; color: var(--color-text-dim); line-height: 1.4; }
.ea-hours { display: grid; grid-template-columns: repeat(24, 1fr); gap: 3px; }
.ea-hour-cell { height: 34px; border-radius: 4px; border: 1px solid var(--color-outline-dim); }
.ea-hours-axis { display: grid; grid-template-columns: repeat(24, 1fr); margin-top: 6px; }
.ea-hours-tick { font-size: 10px; color: var(--color-text-dim); white-space: nowrap; }
.ea-hours-tick:last-child { justify-self: end; text-align: right; }
.ea-legend { display: flex; align-items: center; gap: 8px; margin-top: 12px; }
.ea-legend-cap { font-size: 11px; color: var(--color-text-dim); }
.ea-legend-bar { flex: 1; max-width: 160px; height: 8px; border-radius: var(--radius-full); border: 1px solid var(--color-outline-dim); background: linear-gradient(90deg, var(--color-surface-low), var(--color-primary)); }

/* Лента */
.ea-feed { padding: 8px; background: var(--color-surface); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); }
.ea-events { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; }
.ea-event { display: flex; align-items: center; gap: 12px; padding: 12px; border-radius: var(--radius-md); }
.ea-event:hover { background: var(--color-surface-low); }
.ea-event + .ea-event { border-top: 1px solid var(--color-outline-dim); }
.ea-event-ico { width: 38px; height: 38px; border-radius: 50%; display: grid; place-items: center; font-size: 20px; background: var(--color-surface-low); color: var(--color-text-dim); flex-shrink: 0; }
.ea-event-ico[data-type="task_closed"] { background: color-mix(in oklch, var(--color-success) 18%, transparent); color: var(--color-success); }
.ea-event-ico[data-type="task_created"] { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.ea-event-ico[data-type="comment"] { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.ea-event-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.ea-event-title { font-size: 14px; color: var(--color-text); }
.ea-event-task { font-weight: 600; }
.ea-event-detail { font-size: 12px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ea-event-time { font-size: 12px; color: var(--color-text-dim); white-space: nowrap; flex-shrink: 0; }

.ea-pager { display: flex; align-items: center; justify-content: center; gap: 14px; padding: 14px; }
.ea-pager .btn-glass { width: 40px; height: 40px; padding: 0; display: grid; place-items: center; border-radius: var(--radius-full); }
.ea-pager-info { font-weight: 700; color: var(--color-text-dim); }

@media (max-width: 720px) {
  .ea-grid { grid-template-columns: 1fr; }
  .ea-kpis { grid-template-columns: 1fr 1fr; }
  .ea-hour-cell { height: 40px; }
  .ea-hours-tick { font-size: 9px; }
  .hide-narrow { display: none; }
  .ea-export { width: 40px; height: 40px; padding: 0; justify-content: center; }
}
</style>
