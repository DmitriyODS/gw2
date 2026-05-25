<template>
  <div class="tv-view" :data-dark="themeStore.dark">
    <header class="tv-header">
      <div class="tv-logo">
        <img src="/logo.svg" alt="Grove Work" />
        <span class="tv-title">Grove Work</span>
      </div>
      <div class="tv-slide-name">{{ currentSlide.title }}</div>
      <div class="tv-clock">{{ clock }}</div>
    </header>

    <div class="tv-progress">
      <div
        v-for="(s, i) in slides"
        :key="s.id"
        class="tv-progress-bar"
        :class="{ active: i === activeIdx, done: i < activeIdx }"
      >
        <div
          class="tv-progress-fill"
          :style="i === activeIdx ? { animationDuration: SLIDE_MS + 'ms' } : {}"
        ></div>
      </div>
    </div>

    <main class="tv-main">
      <transition name="tv-fade" mode="out-in">
        <section :key="currentSlide.id" class="tv-slide">
          <div class="tv-period-label">
            <span class="material-symbols-outlined">{{ currentSlide.icon }}</span>
            {{ currentSlide.periodLabel }}
          </div>

          <div v-if="slideLoading" class="tv-loader">
            <ProgressSpinner />
          </div>

          <template v-else-if="commonData && extendedData">
            <!-- Большие цифры по задачам -->
            <div class="tv-numbers-row">
              <div class="tv-stat tv-stat--debt">
                <div class="tv-stat-label">Долг</div>
                <div class="tv-stat-value">{{ commonData.tasks?.debt ?? 0 }}</div>
              </div>
              <div class="tv-stat tv-stat--received">
                <div class="tv-stat-label">Поступило</div>
                <div class="tv-stat-value">+{{ commonData.tasks?.received ?? 0 }}</div>
              </div>
              <div class="tv-stat tv-stat--closed">
                <div class="tv-stat-label">Закрыто</div>
                <div class="tv-stat-value">−{{ commonData.tasks?.closed ?? 0 }}</div>
              </div>
              <div class="tv-stat tv-stat--remaining">
                <div class="tv-stat-label">Осталось</div>
                <div class="tv-stat-value">{{ commonData.tasks?.remaining ?? 0 }}</div>
              </div>
              <div class="tv-stat tv-stat--hours">
                <div class="tv-stat-label">Всего отработано</div>
                <div class="tv-stat-value">{{ formatHoursShort(totalHours) }}</div>
              </div>
            </div>

            <!-- Топы -->
            <div class="tv-grid">
              <div class="tv-card">
                <div class="tv-card-title">
                  <span class="material-symbols-outlined">workspace_premium</span>
                  Топ сотрудников
                </div>
                <div v-if="topEmployees.length === 0" class="tv-empty">Никто ещё не работал</div>
                <ol v-else class="tv-list">
                  <li
                    v-for="(e, i) in topEmployees"
                    :key="e.user_id || e.fio + i"
                    class="tv-list-item"
                  >
                    <span class="tv-rank">{{ i + 1 }}</span>
                    <span class="tv-name">{{ e.fio }}</span>
                    <div class="tv-bar-track">
                      <div
                        class="tv-bar-fill"
                        :style="{ width: percent(e.total_hours, employeesMax) + '%' }"
                      ></div>
                    </div>
                    <span class="tv-value">{{ formatHoursShort(e.total_hours) }}</span>
                  </li>
                </ol>
              </div>

              <div class="tv-card">
                <div class="tv-card-title">
                  <span class="material-symbols-outlined">trending_up</span>
                  Топ задач по времени
                </div>
                <div v-if="topTasks.length === 0" class="tv-empty">За период работа не велась</div>
                <ol v-else class="tv-list">
                  <li
                    v-for="(t, i) in topTasks"
                    :key="t.task_id || t.name + i"
                    class="tv-list-item"
                  >
                    <span class="tv-rank">{{ i + 1 }}</span>
                    <span class="tv-name">{{ t.name }}</span>
                    <div class="tv-bar-track">
                      <div
                        class="tv-bar-fill tv-bar-fill--alt"
                        :style="{ width: percent(t.total_hours, tasksMax) + '%' }"
                      ></div>
                    </div>
                    <span class="tv-value">{{ formatHoursShort(t.total_hours) }}</span>
                  </li>
                </ol>
              </div>

              <div class="tv-card tv-card--wide">
                <div class="tv-card-title">
                  <span class="material-symbols-outlined">apartment</span>
                  Активность по отделам
                </div>
                <div v-if="deptList.length === 0" class="tv-empty">Нет данных</div>
                <div v-else class="tv-dept-grid">
                  <div
                    v-for="d in deptList"
                    :key="d.dept_id || d.name"
                    class="tv-dept-card"
                  >
                    <div class="tv-dept-name">{{ d.name }}</div>
                    <div class="tv-dept-num">{{ d.tasks_count }}</div>
                    <div class="tv-dept-sub">задач</div>
                  </div>
                </div>
              </div>
            </div>
          </template>

          <div v-else class="tv-empty tv-empty--big">
            <span class="material-symbols-outlined">hourglass_empty</span>
            Загружаем данные…
          </div>
        </section>
      </transition>
    </main>

    <footer class="tv-footer">
      <button class="tv-ctrl-btn" @click="prev" title="Предыдущий слайд">
        <span class="material-symbols-outlined">chevron_left</span>
      </button>
      <button class="tv-ctrl-btn" @click="togglePause" :title="paused ? 'Запустить' : 'Пауза'">
        <span class="material-symbols-outlined">{{ paused ? 'play_arrow' : 'pause' }}</span>
      </button>
      <button class="tv-ctrl-btn" @click="next" title="Следующий слайд">
        <span class="material-symbols-outlined">chevron_right</span>
      </button>
      <div class="tv-footer-spacer"></div>
      <button class="tv-ctrl-btn" @click="toggleFullscreen" title="Во весь экран">
        <span class="material-symbols-outlined">{{ isFullscreen ? 'fullscreen_exit' : 'fullscreen' }}</span>
      </button>
    </footer>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useThemeStore } from '@/stores/theme.js'
import { getStatsCommon, getStatsExtended } from '@/api/stats.js'

const themeStore = useThemeStore()

const SLIDE_MS = 10_000
const REFRESH_MS = 60_000

const slides = [
  { id: 'day', title: 'За день', icon: 'today', periodLabel: 'Сегодня', range: () => makeRange('day') },
  { id: 'week', title: 'За неделю', icon: 'date_range', periodLabel: 'Последние 7 дней', range: () => makeRange('week') },
  { id: 'month', title: 'За месяц', icon: 'calendar_month', periodLabel: 'Последние 30 дней', range: () => makeRange('month') },
]

const activeIdx = ref(0)
const paused = ref(false)
const isFullscreen = ref(false)
const clock = ref('')

const commonBySlide = ref({})
const extendedBySlide = ref({})
const slideLoading = ref(false)

const currentSlide = computed(() => slides[activeIdx.value])
const commonData = computed(() => commonBySlide.value[currentSlide.value.id])
const extendedData = computed(() => extendedBySlide.value[currentSlide.value.id])

function fmt(d) {
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
}

function makeRange(kind) {
  const now = new Date()
  const to = fmt(now)
  const d = new Date(now)
  if (kind === 'day') {
    return { from: to, to }
  }
  if (kind === 'week') d.setDate(d.getDate() - 6)
  if (kind === 'month') d.setDate(d.getDate() - 29)
  return { from: fmt(d), to }
}

async function loadSlide(slide, { silent = false } = {}) {
  const { from, to } = slide.range()
  if (!silent) slideLoading.value = true
  try {
    const [common, extended] = await Promise.all([
      getStatsCommon(from, to),
      getStatsExtended(from, to),
    ])
    commonBySlide.value = { ...commonBySlide.value, [slide.id]: common }
    extendedBySlide.value = { ...extendedBySlide.value, [slide.id]: extended }
  } catch {
    // оставляем старые данные, ошибки не показываем — это табло на стене
  } finally {
    if (!silent) slideLoading.value = false
  }
}

// Бэк отдаёт total_hours как Decimal → строка в JSON. Везде приводим к Number.
function num(v) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

const topEmployees = computed(() => {
  const list = commonData.value?.tasks_by_employees || []
  return [...list].sort((a, b) => num(b.total_hours) - num(a.total_hours)).slice(0, 7)
})

const topTasks = computed(() => {
  const list = commonData.value?.tasks_by_hours || []
  return [...list].sort((a, b) => num(b.total_hours) - num(a.total_hours)).slice(0, 7)
})

const deptList = computed(() => {
  const list = extendedData.value?.by_departments || []
  return [...list].sort((a, b) => num(b.tasks_count) - num(a.tasks_count)).slice(0, 8)
})

const employeesMax = computed(() => Math.max(1, ...topEmployees.value.map(e => num(e.total_hours))))
const tasksMax = computed(() => Math.max(1, ...topTasks.value.map(t => num(t.total_hours))))

const totalHours = computed(() => {
  const list = commonData.value?.tasks_by_employees || []
  return list.reduce((acc, e) => acc + num(e.total_hours), 0)
})

function formatHoursShort(val) {
  const hours = num(val)
  if (hours <= 0) return '0 ч'
  const totalMinutes = Math.round(hours * 60)
  const h = Math.floor(totalMinutes / 60)
  const m = totalMinutes % 60
  if (h === 0) return `${m} мин`
  if (m === 0) return `${h} ч`
  return `${h} ч ${m} мин`
}

function percent(val, max) {
  const m = num(max)
  if (!m) return 0
  return Math.max(4, Math.round((num(val) / m) * 100))
}

let slideTimer = null
let refreshTimer = null
let clockTimer = null

function scheduleNext() {
  clearTimeout(slideTimer)
  if (paused.value) return
  slideTimer = setTimeout(async () => {
    await goTo((activeIdx.value + 1) % slides.length)
    scheduleNext()
  }, SLIDE_MS)
}

async function goTo(idx) {
  activeIdx.value = idx
  await loadSlide(slides[idx])
}

async function next() {
  await goTo((activeIdx.value + 1) % slides.length)
  scheduleNext()
}

async function prev() {
  await goTo((activeIdx.value - 1 + slides.length) % slides.length)
  scheduleNext()
}

function togglePause() {
  paused.value = !paused.value
  if (paused.value) clearTimeout(slideTimer)
  else scheduleNext()
}

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

function tickClock() {
  const d = new Date()
  clock.value = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

onMounted(async () => {
  themeStore.init()
  tickClock()
  clockTimer = setInterval(tickClock, 30_000)
  document.addEventListener('fullscreenchange', onFsChange)

  await loadSlide(slides[0])
  // Параллельно прогреваем остальные слайды, чтобы переключение было мгновенным.
  slides.slice(1).forEach(s => loadSlide(s, { silent: true }))

  refreshTimer = setInterval(() => {
    slides.forEach(s => loadSlide(s, { silent: true }))
  }, REFRESH_MS)

  scheduleNext()
})

onBeforeUnmount(() => {
  clearTimeout(slideTimer)
  clearInterval(refreshTimer)
  clearInterval(clockTimer)
  document.removeEventListener('fullscreenchange', onFsChange)
})
</script>

<style scoped>
.tv-view {
  position: fixed;
  inset: 0;
  background: var(--color-bg);
  color: var(--color-text);
  display: flex;
  flex-direction: column;
  font-family: inherit;
  overflow: hidden;
}

.tv-header {
  display: flex;
  align-items: center;
  gap: 24px;
  padding: 20px 32px;
  border-bottom: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
}

.tv-logo {
  display: flex;
  align-items: center;
  gap: 12px;
}

.tv-logo img {
  width: 40px;
  height: 40px;
  border-radius: 50%;
}

.tv-title {
  font-size: 22px;
  font-weight: 800;
  letter-spacing: 0.02em;
}

.tv-slide-name {
  flex: 1;
  text-align: center;
  font-size: 28px;
  font-weight: 700;
  color: var(--color-primary);
}

.tv-clock {
  font-size: 28px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
}

.tv-progress {
  display: flex;
  gap: 6px;
  padding: 6px 32px;
  background: var(--color-surface);
}

.tv-progress-bar {
  flex: 1;
  height: 4px;
  border-radius: 2px;
  background: var(--color-outline-dim);
  overflow: hidden;
}

.tv-progress-bar.done {
  background: var(--color-primary);
}

.tv-progress-fill {
  height: 100%;
  width: 0;
  background: var(--color-primary);
}

.tv-progress-bar.active .tv-progress-fill {
  animation: tv-fill linear forwards;
}

@keyframes tv-fill {
  from { width: 0; }
  to { width: 100%; }
}

.tv-main {
  flex: 1;
  display: flex;
  padding: 28px 32px;
  min-height: 0;
}

.tv-slide {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 24px;
  min-height: 0;
}

.tv-period-label {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  align-self: flex-start;
  padding: 8px 20px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-radius: 999px;
  font-size: 18px;
  font-weight: 700;
}

.tv-period-label .material-symbols-outlined {
  font-size: 22px;
}

.tv-loader {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.tv-numbers-row {
  display: grid;
  grid-template-columns: repeat(5, 1fr);
  gap: 16px;
}

.tv-stat {
  padding: 22px 18px;
  border-radius: 22px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  gap: 6px;
  align-items: flex-start;
}

.tv-stat-label {
  font-size: 15px;
  color: var(--color-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 600;
}

.tv-stat-value {
  font-size: 56px;
  font-weight: 800;
  line-height: 1;
  font-variant-numeric: tabular-nums;
}

.tv-stat--debt { border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim)); }
.tv-stat--debt .tv-stat-value { color: var(--color-primary); }
.tv-stat--received { border-color: color-mix(in oklch, var(--color-success) 35%, var(--color-outline-dim)); }
.tv-stat--received .tv-stat-value { color: var(--color-success); }
.tv-stat--closed { border-color: color-mix(in oklch, var(--color-error) 35%, var(--color-outline-dim)); }
.tv-stat--closed .tv-stat-value { color: var(--color-error); }
.tv-stat--remaining { border-color: color-mix(in oklch, var(--color-tertiary) 35%, var(--color-outline-dim)); }
.tv-stat--remaining .tv-stat-value { color: var(--color-tertiary); }
.tv-stat--hours { border-color: color-mix(in oklch, var(--color-secondary) 35%, var(--color-outline-dim)); }
.tv-stat--hours .tv-stat-value { color: var(--color-secondary); font-size: 44px; }

.tv-grid {
  flex: 1;
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr auto;
  gap: 16px;
  min-height: 0;
}

.tv-card {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 22px;
  padding: 20px 24px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-height: 0;
  overflow: hidden;
}

.tv-card--wide {
  grid-column: 1 / -1;
}

.tv-card-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 20px;
  font-weight: 700;
  color: var(--color-text);
}

.tv-card-title .material-symbols-outlined {
  font-size: 24px;
  color: var(--color-primary);
}

.tv-list {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  overflow: hidden;
}

.tv-list-item {
  display: grid;
  grid-template-columns: 32px minmax(0, 1.5fr) minmax(0, 2fr) auto;
  gap: 14px;
  align-items: center;
  font-size: 17px;
}

.tv-rank {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 800;
  font-size: 15px;
}

.tv-name {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-bar-track {
  height: 12px;
  background: var(--color-outline-dim);
  border-radius: 999px;
  overflow: hidden;
}

.tv-bar-fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: 999px;
  transition: width 0.5s ease;
}

.tv-bar-fill--alt {
  background: var(--color-tertiary);
}

.tv-value {
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
  white-space: nowrap;
}

.tv-dept-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 14px;
}

.tv-dept-card {
  padding: 16px 18px;
  border-radius: 18px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: flex-start;
}

.tv-dept-name {
  font-size: 14px;
  font-weight: 600;
  opacity: 0.85;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}

.tv-dept-num {
  font-size: 38px;
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

.tv-dept-sub {
  font-size: 13px;
  opacity: 0.7;
}

.tv-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 0;
  color: var(--color-text-secondary);
  font-size: 16px;
}

.tv-empty--big {
  flex: 1;
  flex-direction: column;
  gap: 16px;
  font-size: 22px;
}

.tv-empty--big .material-symbols-outlined {
  font-size: 56px;
  opacity: 0.5;
}

.tv-footer {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 24px;
  border-top: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
}

.tv-ctrl-btn {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 1px solid var(--color-outline-dim);
  background: var(--color-bg);
  color: var(--color-text);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.12s, color 0.12s;
}

.tv-ctrl-btn:hover {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.tv-ctrl-btn .material-symbols-outlined {
  font-size: 26px;
}

.tv-footer-spacer {
  flex: 1;
}

/* Fade-переходы между слайдами */
.tv-fade-enter-active,
.tv-fade-leave-active {
  transition: opacity 0.45s ease, transform 0.45s ease;
}

.tv-fade-enter-from {
  opacity: 0;
  transform: translateY(14px);
}

.tv-fade-leave-to {
  opacity: 0;
  transform: translateY(-14px);
}

@media (max-width: 1100px) {
  .tv-numbers-row {
    grid-template-columns: repeat(3, 1fr);
  }
  .tv-stat-value { font-size: 44px; }
  .tv-stat--hours .tv-stat-value { font-size: 34px; }
  .tv-grid {
    grid-template-columns: 1fr;
  }
  .tv-card--wide {
    grid-column: auto;
  }
}

@media (max-width: 700px) {
  .tv-header { padding: 12px 18px; gap: 12px; }
  .tv-title { font-size: 16px; }
  .tv-slide-name { font-size: 18px; }
  .tv-clock { font-size: 18px; }
  .tv-numbers-row { grid-template-columns: repeat(2, 1fr); gap: 10px; }
  .tv-stat { padding: 14px; border-radius: 16px; }
  .tv-stat-value { font-size: 32px; }
  .tv-stat--hours .tv-stat-value { font-size: 26px; }
  .tv-list-item { grid-template-columns: 28px minmax(0, 1fr) auto; }
  .tv-list-item .tv-bar-track { display: none; }
}
</style>
