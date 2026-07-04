<template>
  <div
    class="tv"
    :data-dark="themeStore.dark"
    @mousemove="bumpControls"
    @touchstart="bumpControls"
  >
    <!-- ═══ HEADER ═══════════════════════════════════════════════════════ -->
    <header class="tv-head">
      <div class="tv-brand">
        <img class="tv-brand-logo" src="/logo.svg" alt="Groove Work" />
        <div class="tv-brand-text">
          <div class="tv-brand-name">Groove Work</div>
          <!-- Честный индикатор: LIVE только пока данные реально свежие -->
          <div v-if="!isStale" class="tv-live">
            <span class="tv-live-dot"></span>
            LIVE
          </div>
          <div v-else class="tv-live tv-live--stale" :class="{ 'is-old': dataAgeMin >= 15 }">
            <span class="material-symbols-outlined">history</span>
            данные {{ dataAgeMin }} мин назад
          </div>
        </div>
      </div>

      <transition name="tv-pill" mode="out-in">
        <div :key="currentSlide.id + '-pill'" class="tv-period-pill">
          <span class="material-symbols-outlined">{{ currentSlide.icon }}</span>
          {{ currentSlide.periodLabel }}
        </div>
      </transition>

      <div class="tv-clock">
        <div class="tv-clock-time">{{ clock }}</div>
        <div class="tv-clock-date">{{ todayLabel }}</div>
      </div>
    </header>

    <!-- ═══ PROGRESS ═════════════════════════════════════════════════════ -->
    <div class="tv-progress">
      <div
        v-for="(s, i) in visibleSlides"
        :key="s.id"
        class="tv-progress-bar"
        :class="{ active: i === activeIdx, done: i < activeIdx }"
      >
        <div
          class="tv-progress-fill"
          :style="i === activeIdx && !paused ? { animationDuration: slideDuration(i) + 'ms' } : {}"
        ></div>
      </div>
    </div>

    <!-- ═══ CANVAS: KPI RAIL | STAGE | ASIDE ════════════════════════════ -->
    <main class="tv-canvas">
      <!-- ─── KPI rail (виден всегда) ──────────────────────────────────── -->
      <aside class="tv-kpi-rail">
        <div class="tv-rail-title">
          <span class="material-symbols-outlined">monitoring</span>
          За {{ railPeriodLabel }}
        </div>

        <TvKpiTile tone="primary" icon="inbox" label="Поступило"
          :value="commonData?.tasks?.received ?? 0" format="int" prefix="+" />
        <TvKpiTile tone="success" icon="task_alt" label="Закрыто"
          :value="commonData?.tasks?.closed ?? 0" format="int" prefix="−" />
        <TvKpiTile tone="tertiary" icon="hourglass_top" label="В работе"
          :value="commonData?.tasks?.remaining ?? 0" format="int" />
        <TvKpiTile tone="secondary" icon="schedule" label="Часы команды"
          :value="totalHours" format="hours" />
      </aside>

      <!-- ─── STAGE (текущий слайд через реестр kind→компонент) ────────── -->
      <transition name="tv-stage" mode="out-in">
        <section :key="currentSlide.id" class="tv-stage">
          <div v-if="slideLoading && !commonData" class="tv-stage-loader">
            <ProgressSpinner />
          </div>
          <component
            v-else
            :is="SLIDE_COMPONENTS[currentSlide.kind]"
            :slide="currentSlide"
            v-bind="stageProps"
          />
        </section>
      </transition>

      <!-- ─── ASIDE rail ────────────────────────────────────────────── -->
      <aside class="tv-aside-rail">
        <transition name="tv-aside" mode="out-in">
          <TvAsideCard :key="currentSlide.id + '-aside'" :slide="currentSlide" :content="asideContent" />
        </transition>
      </aside>
    </main>

    <!-- ═══ TICKER (бегущая строка) ═══════════════════════════════════ -->
    <TvTicker
      :common-by-period="commonByPeriod"
      :extended-by-period="extendedByPeriod"
      :hours-per-day="settings.hoursPerDay"
    />

    <!-- ═══ CONTROLS (auto-hide) ════════════════════════════════════ -->
    <div class="tv-controls" :class="{ visible: controlsVisible }">
      <button class="tv-ctrl" @click="prev" title="Предыдущий слайд">
        <span class="material-symbols-outlined">chevron_left</span>
      </button>
      <button class="tv-ctrl" @click="togglePause" :title="paused ? 'Запустить' : 'Пауза'">
        <span class="material-symbols-outlined">{{ paused ? 'play_arrow' : 'pause' }}</span>
      </button>
      <button class="tv-ctrl" @click="next" title="Следующий слайд">
        <span class="material-symbols-outlined">chevron_right</span>
      </button>
      <button class="tv-ctrl" @click="settingsOpen = true" title="Настройки табло">
        <span class="material-symbols-outlined">settings</span>
      </button>
      <button class="tv-ctrl" @click="toggleFullscreen" :title="isFullscreen ? 'Свернуть' : 'Во весь экран'">
        <span class="material-symbols-outlined">{{ isFullscreen ? 'fullscreen_exit' : 'fullscreen' }}</span>
      </button>
    </div>

    <TvSettingsDialog
      v-model="settingsOpen"
      :slides="slides"
      :settings="settings"
      @update:settings="settings = $event"
    />
  </div>
</template>

<script setup>
import { ref, computed, watch, provide, onMounted, onBeforeUnmount } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useThemeStore } from '@/stores/theme.js'
import { getDirectory } from '@/api/users.js'
import { getTvFact } from '@/api/ai.js'
import { getGrooveTv } from '@/api/groove.js'
import { getStatsResponsibles } from '@/api/stats.js'
import { useTvPeriodData } from '@/composables/useTvPeriodData.js'
import { num, sumHours } from '@/components/tv/tvFormat.js'
import { buildAsideContent } from '@/components/tv/tvAside.js'
import { SLIDES as slides, SLIDE_COMPONENTS } from '@/components/tv/tvSlides.js'
import TvKpiTile from '@/components/tv/TvKpiTile.vue'
import TvAsideCard from '@/components/tv/TvAsideCard.vue'
import TvTicker from '@/components/tv/TvTicker.vue'
import TvSettingsDialog from '@/components/tv/TvSettingsDialog.vue'
import '@/components/tv/tv-shared.css'

const themeStore = useThemeStore()

const REFRESH_MS = 60_000
const CONTROLS_HIDE_MS = 2_500

// ─── Настройки табло (localStorage, применяются без перезагрузки) ────────
const TV_SETTINGS_KEY = 'gw_tv_settings'

function clampNum(v, min, max, def) {
  const n = Number(v)
  if (!Number.isFinite(n)) return def
  return Math.min(max, Math.max(min, Math.round(n)))
}

function loadSettings() {
  const def = { disabled: [], slideSec: 8, brandSec: 30, hoursPerDay: 8 }
  try {
    const raw = JSON.parse(localStorage.getItem(TV_SETTINGS_KEY) || '{}')
    return {
      disabled: Array.isArray(raw.disabled) ? raw.disabled.filter(id => typeof id === 'string') : def.disabled,
      slideSec: clampNum(raw.slideSec, 5, 30, def.slideSec),
      brandSec: clampNum(raw.brandSec, 5, 60, def.brandSec),
      hoursPerDay: clampNum(raw.hoursPerDay, 4, 24, def.hoursPerDay),
    }
  } catch {
    return def
  }
}

const settings = ref(loadSettings())
const settingsOpen = ref(false)

watch(settings, v => {
  try { localStorage.setItem(TV_SETTINGS_KEY, JSON.stringify(v)) } catch { /* табло — молча */ }
}, { deep: true })

// Часы в рабочем дне — всем TvCount в поддереве (формат «N дн M ч»).
provide('tvHoursPerDay', computed(() => settings.value.hoursPerDay))

// ─── Каталог пользователей для аватарок ──────────────────────────────────
const userMap = ref(new Map())
async function loadUsers() {
  try {
    const list = await getDirectory()
    const m = new Map()
    for (const u of list) m.set(u.id, u)
    userMap.value = m
  } catch { /* табло на стене — молча */ }
}

function avatarOf(uid) {
  if (!uid) return '/logo.svg'
  const u = userMap.value.get(uid)
  if (u?.avatar_path) return `/uploads/${u.avatar_path}`
  return `/api/users/${uid}/identicon`
}

// ─── Период → диапазон дат ───────────────────────────────────────────────
function pad(n) { return String(n).padStart(2, '0') }
function fmtDate(d) { return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` }
function makeRange(period) {
  const now = new Date()
  const to = fmtDate(now)
  if (period === 'day') return { from: to, to }
  const d = new Date(now)
  if (period === 'week') d.setDate(d.getDate() - 6)
  if (period === 'month') d.setDate(d.getDate() - 29)
  return { from: fmtDate(d), to }
}

// ─── Данные ───────────────────────────────────────────────────────────────
const {
  commonByPeriod,
  extendedByPeriod,
  loading: slideLoading,
  lastSuccessAt: statsSuccessAt,
  loadPeriod,
} = useTvPeriodData(makeRange)

const grooveData = ref(null)
async function loadGroove() {
  try {
    grooveData.value = await getGrooveTv()
    markFresh()
  } catch {
    grooveData.value = null
  }
}

// Ответственные: кеш последнего удачного ответа (ошибки не сбрасывают его).
const responsiblesData = ref([])
async function loadResponsibles() {
  try {
    const list = await getStatsResponsibles()
    if (Array.isArray(list)) responsiblesData.value = list
    markFresh()
  } catch { /* держим прошлый список */ }
}

const aiFact = ref(null)   // {text, generated_at, kind, slot} | null
async function loadAiFact() {
  try {
    aiFact.value = await getTvFact()   // null если AI выключен / не сгенерён
  } catch { /* фолбэк на brand-цитату */ }
}

// ─── Честный LIVE-индикатор ──────────────────────────────────────────────
const nowTick = ref(Date.now())
const lastDataAt = ref(Date.now())
function markFresh() { lastDataAt.value = Date.now() }
watch(statsSuccessAt, v => { if (v) markFresh() })

const dataAgeMin = computed(() => Math.max(1, Math.round((nowTick.value - lastDataAt.value) / 60_000)))
const isStale = computed(() => nowTick.value - lastDataAt.value > REFRESH_MS * 3)

// ─── Ротация ─────────────────────────────────────────────────────────────
const activeIdx = ref(0)
const paused = ref(false)

// Долг показываем, только когда он есть — по данным периода слайда (неделя).
const debtValue = computed(() => num(commonByPeriod.value['week']?.tasks?.debt))

const visibleSlides = computed(() => {
  const list = slides.filter(s => {
    if (settings.value.disabled.includes(s.id)) return false
    if (s.kind === 'debt' && debtValue.value <= 0) return false
    return true
  })
  // Всё выключили — показываем хотя бы брендовый слайд, табло не гаснет.
  return list.length ? list : slides.filter(s => s.kind === 'brand')
})

const currentSlide = computed(() =>
  visibleSlides.value[activeIdx.value] || visibleSlides.value[0])

// Список видимых слайдов сжался — не выпадаем за его границы.
watch(visibleSlides, list => {
  if (activeIdx.value >= list.length) activeIdx.value = 0
})

const commonData = computed(() => commonByPeriod.value[currentSlide.value.period])
const extendedData = computed(() => extendedByPeriod.value[currentSlide.value.period])
const totalHours = computed(() => sumHours(commonData.value?.tasks_by_employees))

const railPeriodLabel = computed(() => {
  const p = currentSlide.value.period
  if (p === 'day') return 'сегодня'
  if (p === 'week') return 'неделю'
  return 'месяц'
})

// Пропсы текущего слайда (kind → его данные).
const stageProps = computed(() => {
  const s = currentSlide.value
  switch (s.kind) {
    case 'hero-number':
    case 'quad':
      return { common: commonData.value, totalHours: totalHours.value }
    case 'podium':
    case 'ranking':
      return { employees: commonData.value?.tasks_by_employees || [], avatarOf }
    case 'departments':
      return { departments: extendedData.value?.by_departments || [] }
    case 'pulse':
      return { calendar: extendedData.value?.calendar || [] }
    case 'work-types':
      return { unitTypes: extendedData.value?.by_unit_types || [] }
    case 'debt':
      return {
        debt: debtValue.value,
        closed: num(commonData.value?.tasks?.closed),
        remaining: num(commonData.value?.tasks?.remaining),
      }
    case 'responsibles':
      return { responsibles: responsiblesData.value }
    case 'groove':
      return { groove: grooveData.value }
    case 'brand':
      return { aiFact: aiFact.value, dateLabel: longDateLabel.value }
    default:
      return {}
  }
})

const asideContent = computed(() => buildAsideContent(currentSlide.value, {
  common: commonData.value,
  extended: extendedData.value,
  grooveData: grooveData.value,
  commonByPeriod: commonByPeriod.value,
  responsibles: responsiblesData.value,
  totalHours: totalHours.value,
}))

// ─── Таймеры и управление ────────────────────────────────────────────────
const controlsVisible = ref(false)
const isFullscreen = ref(false)
const clock = ref('')
const todayLabel = ref('')
const longDateLabel = ref('')

let slideTimer = null
let refreshTimer = null
let clockTimer = null
let aiFactTimer = null
let controlsTimer = null

function slideDuration(idx) {
  const s = visibleSlides.value[idx]
  const sec = s?.kind === 'brand' ? settings.value.brandSec : settings.value.slideSec
  return (num(sec) || 8) * 1000
}

function scheduleNext() {
  clearTimeout(slideTimer)
  if (paused.value || settingsOpen.value) return
  slideTimer = setTimeout(async () => {
    await goTo((activeIdx.value + 1) % visibleSlides.value.length)
    scheduleNext()
  }, slideDuration(activeIdx.value))
}

async function goTo(idx) {
  activeIdx.value = idx
  const period = visibleSlides.value[idx]?.period
  if (period && !commonByPeriod.value[period]) await loadPeriod(period)
}

async function next() {
  await goTo((activeIdx.value + 1) % visibleSlides.value.length)
  scheduleNext()
}

async function prev() {
  const len = visibleSlides.value.length
  await goTo((activeIdx.value - 1 + len) % len)
  scheduleNext()
}

function togglePause() {
  paused.value = !paused.value
  if (paused.value) clearTimeout(slideTimer)
  else scheduleNext()
}

// Настройки меняют длительности — перезапускаем таймер; на время диалога пауза.
watch(() => [settings.value.slideSec, settings.value.brandSec], () => scheduleNext())
watch(settingsOpen, open => {
  if (open) clearTimeout(slideTimer)
  else scheduleNext()
})

async function toggleFullscreen() {
  if (!document.fullscreenElement) {
    try { await document.documentElement.requestFullscreen() } catch {}
  } else {
    try { await document.exitFullscreen() } catch {}
  }
}

function onFsChange() {
  isFullscreen.value = !!document.fullscreenElement
}

function bumpControls() {
  controlsVisible.value = true
  clearTimeout(controlsTimer)
  controlsTimer = setTimeout(() => { controlsVisible.value = false }, CONTROLS_HIDE_MS)
}

function tickClock() {
  const d = new Date()
  nowTick.value = d.getTime()
  clock.value = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  todayLabel.value = d.toLocaleDateString('ru-RU', { weekday: 'short', day: 'numeric', month: 'short' })
  longDateLabel.value = d.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

onMounted(async () => {
  themeStore.init()
  tickClock()
  clockTimer = setInterval(tickClock, 30_000)
  document.addEventListener('fullscreenchange', onFsChange)

  // Стартуем с первого слайда; параллельно грузим все 3 периода и срезы.
  loadUsers()
  loadGroove()
  loadResponsibles()
  await loadPeriod('day')
  loadPeriod('week', { silent: true })
  loadPeriod('month', { silent: true })

  refreshTimer = setInterval(() => {
    loadPeriod('day', { silent: true })
    loadPeriod('week', { silent: true })
    loadPeriod('month', { silent: true })
    loadGroove()
    loadResponsibles()
  }, REFRESH_MS)

  // AI-факт обновляется на сервере до 6 раз в день — раз в час хватит.
  loadAiFact()
  aiFactTimer = setInterval(loadAiFact, 60 * 60 * 1000)

  scheduleNext()
})

onBeforeUnmount(() => {
  clearTimeout(slideTimer)
  clearInterval(refreshTimer)
  clearInterval(clockTimer)
  clearInterval(aiFactTimer)
  clearTimeout(controlsTimer)
  document.removeEventListener('fullscreenchange', onFsChange)
})
</script>

<style scoped>
/* ════════════════ КАРКАС ════════════════════════════════════════════ */
.tv {
  position: fixed;
  inset: 0;
  overflow: hidden;
  display: grid;
  grid-template-rows: auto auto 1fr auto;
  background:
    radial-gradient(circle at 12% 88%, color-mix(in oklch, var(--color-primary) 10%, transparent), transparent 55%),
    radial-gradient(circle at 88% 12%, color-mix(in oklch, var(--color-tertiary) 10%, transparent), transparent 55%),
    var(--color-bg);
  color: var(--color-text);
  font-family: inherit;
}

/* ════════════════ HEADER ════════════════════════════════════════════ */
.tv-head {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  align-items: center;
  padding: clamp(12px, 1.6vmin, 22px) clamp(20px, 2.6vmin, 36px);
  border-bottom: 1px solid var(--color-outline-dim);
  background: color-mix(in oklch, var(--color-surface) 75%, transparent);
  backdrop-filter: blur(8px);
  gap: clamp(12px, 2vmin, 28px);
}

.tv-brand {
  display: flex;
  align-items: center;
  gap: clamp(8px, 1.2vmin, 16px);
  min-width: 0;
}

.tv-brand-logo {
  width: clamp(36px, 4.4vmin, 56px);
  height: clamp(36px, 4.4vmin, 56px);
  border-radius: 50%;
}

.tv-brand-text { display: flex; flex-direction: column; min-width: 0; }

.tv-brand-name {
  font-size: clamp(14px, 1.8vmin, 22px);
  font-weight: 800;
  letter-spacing: 0.02em;
}

.tv-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: clamp(11px, 1.2vmin, 14px);
  font-weight: 700;
  letter-spacing: 0.18em;
  color: var(--color-error);
  text-transform: uppercase;
}

.tv-live-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-error);
  box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-error) 45%, transparent);
  animation: tv-live-pulse 1.6s ease-out infinite;
}

@keyframes tv-live-pulse {
  0%   { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-error) 65%, transparent); }
  100% { box-shadow: 0 0 0 16px color-mix(in oklch, var(--color-error) 0%, transparent); }
}

/* Данные протухли: без пульса, приглушённый warning; совсем старые — error. */
.tv-live--stale {
  color: var(--color-warning);
  text-transform: none;
  letter-spacing: 0.04em;
  white-space: nowrap;
}

.tv-live--stale.is-old { color: var(--color-error); }

.tv-live--stale .material-symbols-outlined {
  font-size: clamp(13px, 1.5vmin, 17px);
}

.tv-period-pill {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  padding: clamp(8px, 1.2vmin, 14px) clamp(18px, 2.4vmin, 30px);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-radius: 999px;
  font-size: clamp(15px, 2vmin, 22px);
  font-weight: 700;
  justify-self: center;
  white-space: nowrap;
}

.tv-period-pill .material-symbols-outlined {
  font-size: clamp(20px, 2.4vmin, 28px);
}

.tv-clock {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-variant-numeric: tabular-nums;
  min-width: 0;
}

.tv-clock-time {
  font-size: clamp(22px, 3vmin, 36px);
  font-weight: 800;
  line-height: 1;
}

.tv-clock-date {
  font-size: clamp(12px, 1.4vmin, 16px);
  color: var(--color-text-dim);
  font-weight: 600;
  margin-top: 4px;
  text-transform: capitalize;
}

/* ════════════════ PROGRESS BAR ══════════════════════════════════════ */
.tv-progress {
  display: flex;
  gap: 4px;
  padding: 4px clamp(20px, 2.6vmin, 36px);
  background: color-mix(in oklch, var(--color-surface) 75%, transparent);
}

.tv-progress-bar {
  flex: 1;
  height: 3px;
  border-radius: 2px;
  background: var(--color-outline-dim);
  overflow: hidden;
}

.tv-progress-bar.done {
  background: color-mix(in oklch, var(--color-primary) 70%, transparent);
}

.tv-progress-fill {
  height: 100%;
  width: 0;
  background: var(--color-primary);
}

.tv-progress-bar.active .tv-progress-fill {
  animation: tv-progress-fill linear forwards;
}

@keyframes tv-progress-fill {
  from { width: 0; }
  to   { width: 100%; }
}

/* ════════════════ CANVAS ════════════════════════════════════════════ */
.tv-canvas {
  display: grid;
  grid-template-columns: clamp(220px, 22vmin, 320px) 1fr clamp(240px, 24vmin, 340px);
  gap: clamp(14px, 1.8vmin, 24px);
  padding: clamp(14px, 1.8vmin, 24px) clamp(20px, 2.6vmin, 36px);
  min-height: 0;
}

/* ════════════════ KPI RAIL ══════════════════════════════════════════ */
.tv-kpi-rail {
  display: grid;
  grid-template-rows: auto repeat(4, 1fr);
  gap: clamp(10px, 1.2vmin, 16px);
  min-height: 0;
}

.tv-rail-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: clamp(12px, 1.4vmin, 16px);
  font-weight: 700;
  letter-spacing: 0.1em;
  color: var(--color-text-dim);
  text-transform: uppercase;
}

.tv-rail-title .material-symbols-outlined {
  font-size: clamp(16px, 1.8vmin, 22px);
  color: var(--color-primary);
}

/* ════════════════ STAGE ═════════════════════════════════════════════ */
.tv-stage {
  position: relative;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: clamp(18px, 2.4vmin, 28px);
  padding: clamp(18px, 2.4vmin, 32px) clamp(22px, 3vmin, 40px);
  display: flex;
  flex-direction: column;
  gap: clamp(14px, 1.8vmin, 24px);
  min-height: 0;
  overflow: hidden;
}

.tv-stage-loader { flex: 1; display: flex; align-items: center; justify-content: center; }

/* ════════════════ ASIDE RAIL ════════════════════════════════════════ */
.tv-aside-rail {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

/* ════════════════ CONTROLS (auto-hide) ═════════════════════════════ */
.tv-controls {
  position: fixed;
  right: clamp(12px, 1.6vmin, 24px);
  bottom: clamp(56px, 8vmin, 86px);
  display: flex;
  gap: 8px;
  padding: 8px;
  background: color-mix(in oklch, var(--color-surface) 90%, transparent);
  border: 1px solid var(--color-outline-dim);
  border-radius: 999px;
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(8px);
  opacity: 0;
  transform: translateY(8px);
  pointer-events: none;
  transition: opacity 0.2s, transform 0.2s;
  z-index: 50;
}

.tv-controls.visible {
  opacity: 1;
  transform: translateY(0);
  pointer-events: auto;
}

.tv-ctrl {
  width: clamp(36px, 4vmin, 48px);
  height: clamp(36px, 4vmin, 48px);
  border-radius: 50%;
  border: none;
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.12s, color 0.12s;
}

.tv-ctrl:hover {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.tv-ctrl .material-symbols-outlined {
  font-size: clamp(20px, 2.4vmin, 26px);
}

/* ════════════════ TRANSITIONS ══════════════════════════════════════ */
.tv-stage-enter-active,
.tv-stage-leave-active {
  transition: opacity 0.55s cubic-bezier(0.4, 0, 0.2, 1),
              transform 0.55s cubic-bezier(0.4, 0, 0.2, 1),
              filter 0.55s ease;
}

.tv-stage-enter-from {
  opacity: 0;
  transform: translateX(40px) scale(0.98);
  filter: blur(6px);
}

.tv-stage-leave-to {
  opacity: 0;
  transform: translateX(-30px) scale(0.98);
  filter: blur(6px);
}

.tv-aside-enter-active,
.tv-aside-leave-active {
  transition: opacity 0.4s ease, transform 0.4s ease;
}

.tv-aside-enter-from { opacity: 0; transform: translateY(14px); }
.tv-aside-leave-to   { opacity: 0; transform: translateY(-14px); }

.tv-pill-enter-active,
.tv-pill-leave-active {
  transition: opacity 0.3s ease, transform 0.3s ease;
}

.tv-pill-enter-from { opacity: 0; transform: translateY(-8px) scale(0.95); }
.tv-pill-leave-to   { opacity: 0; transform: translateY(8px) scale(0.95); }

/* ════════════════ Compact viewports ════════════════════════════════ */
@media (max-aspect-ratio: 1/1) {
  /* Портретный режим: KPI rail сверху, aside снизу */
  .tv-canvas {
    grid-template-columns: 1fr;
    grid-template-rows: auto 1fr auto;
  }
  .tv-kpi-rail {
    grid-template-rows: auto;
    grid-template-columns: repeat(4, 1fr);
  }
  .tv-rail-title { grid-column: 1 / -1; }
  .tv-aside-rail { min-height: clamp(120px, 14vmin, 180px); }
}

@media (max-width: 900px) {
  .tv-head { grid-template-columns: 1fr auto; }
  .tv-period-pill { display: none; }
}
</style>
