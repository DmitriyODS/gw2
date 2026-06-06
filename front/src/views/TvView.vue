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
          <div class="tv-live">
            <span class="tv-live-dot"></span>
            LIVE
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
        v-for="(s, i) in slides"
        :key="s.id"
        class="tv-progress-bar"
        :class="{ active: i === activeIdx, done: i < activeIdx }"
      >
        <div
          class="tv-progress-fill"
          :style="i === activeIdx && !paused ? { animationDuration: SLIDE_MS + 'ms' } : {}"
        ></div>
      </div>
    </div>

    <!-- ═══ CANVAS: KPI RAIL | STAGE | ASIDE ════════════════════════════ -->
    <main class="tv-canvas">
      <!-- ─── KPI rail (always visible) ─────────────────────────────────── -->
      <aside class="tv-kpi-rail">
        <div class="tv-rail-title">
          <span class="material-symbols-outlined">monitoring</span>
          За {{ railPeriodLabel }}
        </div>

        <TvKpiTile
          tone="primary"
          icon="inbox"
          label="Поступило"
          :value="railCommon?.tasks?.received ?? 0"
          format="int"
          prefix="+"
        />
        <TvKpiTile
          tone="success"
          icon="task_alt"
          label="Закрыто"
          :value="railCommon?.tasks?.closed ?? 0"
          format="int"
          prefix="−"
        />
        <TvKpiTile
          tone="tertiary"
          icon="hourglass_top"
          label="В работе"
          :value="railCommon?.tasks?.remaining ?? 0"
          format="int"
        />
        <TvKpiTile
          tone="secondary"
          icon="schedule"
          label="Часы команды"
          :value="railTotalHours"
          format="hours"
        />
      </aside>

      <!-- ─── STAGE (main content) ─────────────────────────────────────── -->
      <transition name="tv-stage" mode="out-in">
        <section :key="currentSlide.id" class="tv-stage">

          <!-- Loader -->
          <div v-if="slideLoading && !commonData" class="tv-stage-loader">
            <ProgressSpinner />
          </div>

          <!-- HERO NUMBER ───────────────────────────────────────────── -->
          <div v-else-if="currentSlide.kind === 'hero-number'" class="tv-hero">
            <div class="tv-stage-eyebrow">
              <span class="material-symbols-outlined" :style="{ color: heroTone(currentSlide.tone) }">
                {{ currentSlide.heroIcon }}
              </span>
              <span>{{ currentSlide.heroEyebrow }}</span>
            </div>
            <div class="tv-hero-glow" :style="{ '--glow': heroTone(currentSlide.tone) }">
              <div class="tv-hero-number" :style="{ color: heroTone(currentSlide.tone) }">
                <TvCount :value="heroValue(currentSlide.heroKey)" :format="currentSlide.heroFormat || 'int'" />
              </div>
            </div>
            <div class="tv-hero-caption">{{ currentSlide.heroCaption }}</div>
            <div v-if="heroSecondaries(currentSlide).length" class="tv-hero-secondaries">
              <div
                v-for="(s, i) in heroSecondaries(currentSlide)"
                :key="i"
                class="tv-hero-sec"
              >
                <div class="tv-hero-sec-label">{{ s.label }}</div>
                <div class="tv-hero-sec-value" :style="{ color: s.tone ? heroTone(s.tone) : '' }">
                  <TvCount :value="s.value" :format="s.format || 'int'" :prefix="s.prefix || ''" />
                </div>
              </div>
            </div>
          </div>

          <!-- PODIUM ───────────────────────────────────────────────── -->
          <div v-else-if="currentSlide.kind === 'podium'" class="tv-podium-wrap">
            <div class="tv-stage-eyebrow">
              <span class="material-symbols-outlined" style="color: var(--color-warning)">workspace_premium</span>
              <span>{{ currentSlide.heroEyebrow }}</span>
            </div>
            <div v-if="podiumList.length === 0" class="tv-stage-empty">
              Пока никто не работал
            </div>
            <div v-else class="tv-podium">
              <!-- Order: 2nd, 1st, 3rd visually -->
              <div
                v-for="place in podiumOrder"
                :key="place"
                class="tv-podium-col"
                :class="['tv-podium-col--' + place, { 'tv-podium-col--empty': !podiumList[place - 1] }]"
              >
                <template v-if="podiumList[place - 1]">
                  <div class="tv-podium-medal">
                    <span v-if="place === 1" class="tv-fire material-symbols-outlined">local_fire_department</span>
                    <span class="tv-podium-place">{{ place }}</span>
                  </div>
                  <div class="tv-podium-avatar-wrap">
                    <img class="tv-podium-avatar" :src="avatarOf(podiumList[place - 1].user_id)" alt="" />
                  </div>
                  <div class="tv-podium-fio">{{ podiumList[place - 1].fio }}</div>
                  <div class="tv-podium-hours">
                    <TvCount :value="podiumList[place - 1].total_hours" format="hours" />
                  </div>
                  <div class="tv-podium-base">
                    <div class="tv-podium-base-num">{{ place }}</div>
                  </div>
                </template>
                <template v-else>
                  <div class="tv-podium-place-empty">{{ place }}</div>
                  <div class="tv-podium-empty-text">—</div>
                  <div class="tv-podium-base">
                    <div class="tv-podium-base-num">{{ place }}</div>
                  </div>
                </template>
              </div>
            </div>
          </div>

          <!-- RANKING (top-5 list) ─────────────────────────────────── -->
          <div v-else-if="currentSlide.kind === 'ranking'" class="tv-ranking-wrap">
            <div class="tv-stage-eyebrow">
              <span class="material-symbols-outlined" style="color: var(--color-primary)">leaderboard</span>
              <span>{{ currentSlide.heroEyebrow }}</span>
            </div>
            <div v-if="rankingList.length === 0" class="tv-stage-empty">За период работа не велась</div>
            <ol v-else class="tv-ranking">
              <li
                v-for="(e, i) in rankingList"
                :key="e.user_id || e.fio + i"
                class="tv-ranking-item"
                :class="{ 'is-leader': i === 0 }"
                :style="{ '--row-delay': i * 80 + 'ms' }"
              >
                <span class="tv-ranking-rank">
                  <span v-if="i === 0" class="tv-fire material-symbols-outlined">local_fire_department</span>
                  <span v-else>{{ i + 1 }}</span>
                </span>
                <img class="tv-ranking-avatar" :src="avatarOf(e.user_id)" alt="" />
                <span class="tv-ranking-fio">{{ e.fio }}</span>
                <div class="tv-ranking-bar">
                  <div
                    class="tv-ranking-bar-fill"
                    :style="{ '--bar-width': barPercent(e.total_hours, rankingMax) + '%' }"
                  ></div>
                </div>
                <span class="tv-ranking-value">
                  <TvCount :value="e.total_hours" format="hours" />
                </span>
              </li>
            </ol>
          </div>

          <!-- DEPARTMENTS ──────────────────────────────────────────── -->
          <div v-else-if="currentSlide.kind === 'departments'" class="tv-deps-wrap">
            <div class="tv-stage-eyebrow">
              <span class="material-symbols-outlined" style="color: var(--color-tertiary)">apartment</span>
              <span>{{ currentSlide.heroEyebrow }}</span>
            </div>
            <div v-if="deptList.length === 0" class="tv-stage-empty">Нет данных по отделам</div>
            <div v-else class="tv-deps">
              <div
                v-for="(d, i) in deptList"
                :key="d.dept_id || d.name"
                class="tv-dep-bar"
                :style="{ '--row-delay': i * 100 + 'ms' }"
              >
                <div class="tv-dep-bar-label">{{ d.name }}</div>
                <div class="tv-dep-bar-track">
                  <div
                    class="tv-dep-bar-fill"
                    :style="{ '--bar-width': barPercent(d.tasks_count, deptMax) + '%' }"
                  >
                    <span class="tv-dep-bar-num">{{ d.tasks_count }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- QUAD (4 big tiles) ───────────────────────────────────── -->
          <div v-else-if="currentSlide.kind === 'quad'" class="tv-quad-wrap">
            <div class="tv-stage-eyebrow">
              <span class="material-symbols-outlined" style="color: var(--color-secondary)">view_quilt</span>
              <span>{{ currentSlide.heroEyebrow }}</span>
            </div>
            <div class="tv-quad">
              <div class="tv-quad-tile tone-primary">
                <div class="tv-quad-icon"><span class="material-symbols-outlined">inbox</span></div>
                <div class="tv-quad-num"><TvCount :value="commonData?.tasks?.received ?? 0" format="int" prefix="+" /></div>
                <div class="tv-quad-label">поступило</div>
              </div>
              <div class="tv-quad-tile tone-success">
                <div class="tv-quad-icon"><span class="material-symbols-outlined">task_alt</span></div>
                <div class="tv-quad-num"><TvCount :value="commonData?.tasks?.closed ?? 0" format="int" prefix="−" /></div>
                <div class="tv-quad-label">закрыто</div>
              </div>
              <div class="tv-quad-tile tone-tertiary">
                <div class="tv-quad-icon"><span class="material-symbols-outlined">hourglass_top</span></div>
                <div class="tv-quad-num"><TvCount :value="commonData?.tasks?.remaining ?? 0" format="int" /></div>
                <div class="tv-quad-label">в работе</div>
              </div>
              <div class="tv-quad-tile tone-secondary">
                <div class="tv-quad-icon"><span class="material-symbols-outlined">schedule</span></div>
                <div class="tv-quad-num"><TvCount :value="totalHours" format="hours" /></div>
                <div class="tv-quad-label">часы команды</div>
              </div>
            </div>
          </div>

          <!-- BRAND / AI-FACT SLIDE ─────────────────────────────────── -->
          <div v-else-if="currentSlide.kind === 'brand' && aiFact" class="tv-ai-fact-stage">
            <div class="tv-ai-fact-glow"></div>
            <div class="tv-ai-fact-eyebrow">
              <span class="material-symbols-outlined">lightbulb_2</span>
              Факт дня
            </div>
            <div class="tv-ai-fact-text">{{ aiFact.text }}</div>
            <div class="tv-ai-fact-foot">{{ longDateLabel }} · Groove Work</div>
          </div>
          <div v-else-if="currentSlide.kind === 'brand'" class="tv-brand-stage">
            <div class="tv-brand-glow"></div>
            <img class="tv-brand-big-logo" src="/logo.svg" alt="" />
            <div class="tv-brand-big-name">Groove Work</div>
            <div class="tv-brand-quote">«{{ brandQuote }}»</div>
            <div class="tv-brand-date">{{ longDateLabel }}</div>
          </div>

        </section>
      </transition>

      <!-- ─── ASIDE rail ────────────────────────────────────────────── -->
      <aside class="tv-aside-rail">
        <transition name="tv-aside" mode="out-in">
          <div :key="currentSlide.id + '-aside'" class="tv-aside-card" :class="'tone-' + (currentSlide.asideTone || 'primary')">
            <div class="tv-aside-eyebrow">
              <span class="material-symbols-outlined">{{ currentSlide.asideIcon || 'auto_awesome' }}</span>
              {{ currentSlide.asideTitle || 'Контекст' }}
            </div>
            <div v-if="asideContent" class="tv-aside-body">
              <div v-if="asideContent.headline" class="tv-aside-headline">{{ asideContent.headline }}</div>
              <div v-if="asideContent.value != null" class="tv-aside-value">
                <TvCount :value="asideContent.value" :format="asideContent.format || 'int'" :prefix="asideContent.prefix || ''" />
              </div>
              <div v-if="asideContent.sub" class="tv-aside-sub">{{ asideContent.sub }}</div>

              <!-- Спарклайн -->
              <div v-if="asideContent.sparkline?.length" class="tv-spark">
                <svg viewBox="0 0 100 40" preserveAspectRatio="none">
                  <polyline
                    class="tv-spark-line"
                    :points="sparklinePoints(asideContent.sparkline)"
                  />
                  <polygon
                    class="tv-spark-area"
                    :points="sparklineArea(asideContent.sparkline)"
                  />
                </svg>
              </div>
            </div>
            <div v-else class="tv-aside-body">
              <div class="tv-aside-headline">—</div>
            </div>
          </div>
        </transition>
      </aside>
    </main>

    <!-- ═══ TICKER (бегущая строка) ═══════════════════════════════════ -->
    <footer class="tv-ticker">
      <span class="tv-ticker-mark">
        <span class="tv-live-dot tv-live-dot--small"></span>
        ЛЕНТА
      </span>
      <div class="tv-ticker-viewport">
        <div class="tv-ticker-track" :style="{ animationDuration: tickerDuration + 's' }">
          <span v-for="(item, i) in tickerItemsX2" :key="i" class="tv-ticker-item">
            <span class="tv-ticker-bullet">●</span>{{ item }}
          </span>
        </div>
      </div>
    </footer>

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
      <button class="tv-ctrl" @click="toggleFullscreen" :title="isFullscreen ? 'Свернуть' : 'Во весь экран'">
        <span class="material-symbols-outlined">{{ isFullscreen ? 'fullscreen_exit' : 'fullscreen' }}</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, h, defineComponent, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import { useThemeStore } from '@/stores/theme.js'
import { getStatsCommon, getStatsExtended } from '@/api/stats.js'
import { getDirectory } from '@/api/users.js'
import { getTvFact } from '@/api/ai.js'

const themeStore = useThemeStore()

const SLIDE_MS = 8_000
const REFRESH_MS = 60_000
const CONTROLS_HIDE_MS = 2_500

// ─── Цитаты для брендового слайда ────────────────────────────────────────
// Шуточные и тёплые, в разном настроении — выбираем случайную при каждом
// показе слайда (через :key на section мы заново монтируемся → новый ребус).
const BRAND_QUOTES = [
  'Команда — это сила',
  'Лучшая задача — закрытая задача',
  'Сегодня было неплохо. А завтра будет ещё лучше.',
  'Каждый юнит — кусочек большого дела',
  'Кофе допит, дедлайны побеждены',
  'Закрывайте задачи, как двери — с уверенностью',
  'Делаем — значит делаем хорошо',
  'Если задача не двигается, значит она копит энергию',
  'Один за всех — и все на одном Groove',
  'Сегодня выложились — завтра выложимся ещё',
  'Пусть бэклог тает, как снег весной',
  'Кто рано встал — тот рано закрыл',
  'Не бывает маленьких задач — бывают большие закрытия',
  'Считаем не часы, а сделанное. Но часы тоже считаем.',
  'Время — деньги. У нас в платформе и то и другое под учётом.',
  'Лучший юнит — тот, который начат',
  'Помните: даже Эйнштейн делал ошибки в дедлайнах',
  'Релиз ближе, чем кажется',
  'Хорошего дня, хорошей команды и хорошего кофе',
  'Дисциплина — это когда ты закрываешь юнит до обеда',
]

function pickRandomQuote() {
  return BRAND_QUOTES[Math.floor(Math.random() * BRAND_QUOTES.length)]
}

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

// ─── Слайды ───────────────────────────────────────────────────────────────
const slides = [
  // 1. Сегодня • закрытия
  {
    id: 'today-closed', period: 'day', kind: 'hero-number',
    icon: 'today', periodLabel: 'Сегодня',
    heroIcon: 'task_alt', heroEyebrow: 'Закрыто задач за день',
    heroKey: 'closed', heroFormat: 'int', tone: 'success',
    heroCaption: 'столько работ команда довела до финала сегодня',
    secondaries: [
      { label: 'Поступило', key: 'received', tone: 'primary', prefix: '+' },
      { label: 'В работе',  key: 'remaining', tone: 'tertiary' },
    ],
    asideTone: 'primary', asideIcon: 'schedule', asideTitle: 'Время команды',
    asideKind: 'hours-today',
  },
  // 2. Сегодня • подиум
  {
    id: 'today-podium', period: 'day', kind: 'podium',
    icon: 'today', periodLabel: 'Сегодня',
    heroEyebrow: 'Лидеры дня',
    asideTone: 'tertiary', asideIcon: 'apartment', asideTitle: 'Активный отдел',
    asideKind: 'top-dept',
  },
  // 3. Сегодня • отделы
  {
    id: 'today-departments', period: 'day', kind: 'departments',
    icon: 'today', periodLabel: 'Сегодня',
    heroEyebrow: 'Задачи по отделам',
    asideTone: 'success', asideIcon: 'task_alt', asideTitle: 'Сегодня закрыто',
    asideKind: 'closed-today',
  },
  // 4. Неделя • часы команды (heroNumber: hours)
  {
    id: 'week-hours', period: 'week', kind: 'hero-number',
    icon: 'date_range', periodLabel: 'Последние 7 дней',
    heroIcon: 'schedule', heroEyebrow: 'Часы команды за неделю',
    heroKey: 'total_hours', heroFormat: 'hours', tone: 'secondary',
    heroCaption: 'это суммарное время работы всей команды',
    secondaries: [
      { label: 'Закрыто', key: 'closed', tone: 'success', prefix: '−' },
      { label: 'Поступило', key: 'received', tone: 'primary', prefix: '+' },
    ],
    asideTone: 'primary', asideIcon: 'show_chart', asideTitle: 'Динамика',
    asideKind: 'sparkline-closed',
  },
  // 5. Неделя • топ-5 сотрудников
  {
    id: 'week-ranking', period: 'week', kind: 'ranking',
    icon: 'date_range', periodLabel: 'Последние 7 дней',
    heroEyebrow: 'Топ сотрудников недели',
    asideTone: 'secondary', asideIcon: 'schedule', asideTitle: 'Всего часов',
    asideKind: 'hours-period',
  },
  // 6. Месяц • четверть KPI
  {
    id: 'month-quad', period: 'month', kind: 'quad',
    icon: 'calendar_month', periodLabel: 'Последние 30 дней',
    heroEyebrow: 'Месяц одной картой',
    asideTone: 'tertiary', asideIcon: 'apartment', asideTitle: 'Топ-отдел месяца',
    asideKind: 'top-dept',
  },
  // 7. Месяц • MVP
  {
    id: 'month-podium', period: 'month', kind: 'podium',
    icon: 'calendar_month', periodLabel: 'Последние 30 дней',
    heroEyebrow: 'MVP месяца',
    asideTone: 'secondary', asideIcon: 'show_chart', asideTitle: 'Динамика',
    asideKind: 'sparkline-hours',
  },
  // 8. Брендовый слайд
  {
    id: 'brand', period: 'day', kind: 'brand',
    icon: 'auto_awesome', periodLabel: 'Хорошего дня',
    heroEyebrow: 'Groove Work',
    asideTone: 'primary', asideIcon: 'today', asideTitle: 'Сегодня',
    asideKind: 'today-snapshot',
  },
]

// ─── Состояние ────────────────────────────────────────────────────────────
const activeIdx = ref(0)
const paused = ref(false)
const isFullscreen = ref(false)
const clock = ref('')
const todayLabel = ref('')
const longDateLabel = ref('')
const brandQuote = ref(pickRandomQuote())
const aiFact = ref(null)   // {text, generated_at, kind, slot} | null

async function loadAiFact() {
  try {
    aiFact.value = await getTvFact()   // null если AI выключен / не сгенерён
  } catch { /* фолбэк на brand-цитату */ }
}

const commonByPeriod = ref({})   // { day: {...}, week: {...}, month: {...} }
const extendedByPeriod = ref({}) // { day: {...}, week: {...}, month: {...} }
const slideLoading = ref(false)
const controlsVisible = ref(false)
let controlsTimer = null

const currentSlide = computed(() => slides[activeIdx.value])
const commonData = computed(() => commonByPeriod.value[currentSlide.value.period])
const extendedData = computed(() => extendedByPeriod.value[currentSlide.value.period])

// KPI rail смотрит на ТОТ ЖЕ период, что и текущий слайд (динамически).
const railCommon = computed(() => commonData.value)
const railPeriodLabel = computed(() => {
  const p = currentSlide.value.period
  if (p === 'day') return 'сегодня'
  if (p === 'week') return 'неделю'
  return 'месяц'
})
const railTotalHours = computed(() => sumHours(railCommon.value?.tasks_by_employees))

// ─── Утилиты ──────────────────────────────────────────────────────────────
function num(v) {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

function sumHours(list) {
  if (!list) return 0
  return list.reduce((acc, e) => acc + num(e.total_hours), 0)
}

const totalHours = computed(() => sumHours(commonData.value?.tasks_by_employees))

function heroTone(tone) {
  switch (tone) {
    case 'primary':   return 'var(--color-primary)'
    case 'secondary': return 'var(--color-secondary)'
    case 'tertiary':  return 'var(--color-tertiary)'
    case 'success':   return 'var(--color-success)'
    case 'warning':   return 'var(--color-warning)'
    case 'error':     return 'var(--color-error)'
    default: return 'var(--color-primary)'
  }
}

function heroValue(key) {
  if (key === 'total_hours') return totalHours.value
  const t = commonData.value?.tasks
  return t ? num(t[key]) : 0
}

function heroSecondaries(slide) {
  if (!slide.secondaries) return []
  return slide.secondaries.map(s => ({
    label: s.label,
    value: heroValue(s.key),
    format: s.format || 'int',
    tone: s.tone,
    prefix: s.prefix || '',
  }))
}

// ─── Подиум: 1-2-3 в визуальном порядке (2,1,3) ──────────────────────────
const podiumList = computed(() => {
  const list = commonData.value?.tasks_by_employees || []
  return [...list].sort((a, b) => num(b.total_hours) - num(a.total_hours)).slice(0, 3)
})
const podiumOrder = [2, 1, 3]

// ─── Рейтинг (top-N) ──────────────────────────────────────────────────────
const rankingList = computed(() => {
  const list = commonData.value?.tasks_by_employees || []
  return [...list].sort((a, b) => num(b.total_hours) - num(a.total_hours)).slice(0, 5)
})
const rankingMax = computed(() => Math.max(1, ...rankingList.value.map(e => num(e.total_hours))))

// ─── Отделы ──────────────────────────────────────────────────────────────
const deptList = computed(() => {
  const list = extendedData.value?.by_departments || []
  return [...list].sort((a, b) => num(b.tasks_count) - num(a.tasks_count)).slice(0, 5)
})
const deptMax = computed(() => Math.max(1, ...deptList.value.map(d => num(d.tasks_count))))

function barPercent(val, max) {
  const m = num(max)
  if (!m) return 0
  return Math.max(6, Math.round((num(val) / m) * 100))
}

// ─── Aside content по типу слайда ─────────────────────────────────────────
const asideContent = computed(() => {
  const s = currentSlide.value
  const kind = s.asideKind
  if (!kind) return null

  if (kind === 'hours-today') {
    return {
      headline: 'Всего отработано',
      value: totalHours.value, format: 'hours',
      sub: 'все сотрудники, все юниты',
    }
  }
  if (kind === 'hours-period') {
    return {
      headline: 'Команда за период',
      value: totalHours.value, format: 'hours',
      sub: 'суммарно по всем сотрудникам',
    }
  }
  if (kind === 'closed-today') {
    return {
      headline: 'Закрыто',
      value: num(commonData.value?.tasks?.closed), format: 'int',
      prefix: '−',
      sub: 'задач за день',
    }
  }
  if (kind === 'top-dept') {
    const top = (extendedData.value?.by_departments || [])
      .slice().sort((a, b) => num(b.tasks_count) - num(a.tasks_count))[0]
    if (!top) return { headline: 'нет данных' }
    return {
      headline: top.name,
      value: num(top.tasks_count), format: 'int',
      sub: 'задач у лидера',
    }
  }
  if (kind === 'sparkline-closed') {
    const cal = extendedData.value?.calendar || []
    const arr = cal.map(d => num(d.closed))
    return {
      headline: 'Закрытий по дням',
      value: arr.reduce((a, b) => a + b, 0), format: 'int',
      sub: 'за период',
      sparkline: arr,
    }
  }
  if (kind === 'sparkline-hours') {
    const cal = extendedData.value?.calendar || []
    const arr = cal.map(d => num(d.total_hours))
    return {
      headline: 'Часы по дням',
      value: arr.reduce((a, b) => a + b, 0), format: 'hours',
      sub: 'за период',
      sparkline: arr,
    }
  }
  if (kind === 'today-snapshot') {
    const today = commonByPeriod.value['day']
    return {
      headline: 'Сегодня',
      value: num(today?.tasks?.closed), format: 'int',
      prefix: '−',
      sub: 'задач закрыто',
    }
  }
  return null
})

// ─── Спарклайн ────────────────────────────────────────────────────────────
function sparklinePoints(arr) {
  if (!arr || arr.length === 0) return ''
  const max = Math.max(1, ...arr)
  const n = arr.length
  return arr.map((v, i) => {
    const x = (i / Math.max(1, n - 1)) * 100
    const y = 38 - (num(v) / max) * 34
    return `${x.toFixed(2)},${y.toFixed(2)}`
  }).join(' ')
}
function sparklineArea(arr) {
  const pts = sparklinePoints(arr)
  if (!pts) return ''
  return `0,40 ${pts} 100,40`
}

// ─── Ticker items (живая лента) ───────────────────────────────────────────
const tickerItems = computed(() => {
  const out = []
  const day = commonByPeriod.value['day']
  const week = commonByPeriod.value['week']
  const month = commonByPeriod.value['month']
  const dayExt = extendedByPeriod.value['day']
  const weekExt = extendedByPeriod.value['week']

  if (day?.tasks) {
    out.push(`Сегодня закрыто ${day.tasks.closed} ${plural(day.tasks.closed, 'задача', 'задачи', 'задач')}`)
    out.push(`Сегодня поступило ${day.tasks.received} ${plural(day.tasks.received, 'задача', 'задачи', 'задач')}`)
  }
  const dayLeader = (day?.tasks_by_employees || []).slice().sort((a, b) => num(b.total_hours) - num(a.total_hours))[0]
  if (dayLeader) out.push(`Лидер дня — ${dayLeader.fio}, ${formatHoursShort(dayLeader.total_hours)}`)

  const dayDept = (dayExt?.by_departments || []).slice().sort((a, b) => num(b.tasks_count) - num(a.tasks_count))[0]
  if (dayDept) out.push(`Активный отдел дня — ${dayDept.name}`)

  if (week?.tasks) {
    out.push(`За неделю закрыто ${week.tasks.closed} ${plural(week.tasks.closed, 'задача', 'задачи', 'задач')}`)
  }
  const weekHours = sumHours(week?.tasks_by_employees)
  if (weekHours > 0) out.push(`Команда отработала ${formatHoursShort(weekHours)} за неделю`)

  const weekTopType = (weekExt?.by_unit_types || []).slice().sort((a, b) => num(b.total_hours) - num(a.total_hours))[0]
  if (weekTopType) out.push(`Главный тип работ недели — «${weekTopType.name}»`)

  if (month?.tasks) {
    const monthHours = sumHours(month?.tasks_by_employees)
    if (monthHours > 0) out.push(`За месяц команда наработала ${formatHoursShort(monthHours)}`)
    out.push(`Месяц: ${month.tasks.received} поступило, ${month.tasks.closed} закрыто`)
  }
  if (!out.length) out.push('Загружаем данные…')
  return out
})

// Дублируем дорожку, чтобы анимация была бесшовной.
const tickerItemsX2 = computed(() => [...tickerItems.value, ...tickerItems.value])

// Скорость прокрутки — пропорционально количеству пунктов.
const tickerDuration = computed(() => Math.max(20, tickerItems.value.length * 6))

function plural(n, one, few, many) {
  const a = Math.abs(n) % 100
  const b = a % 10
  if (a > 10 && a < 20) return many
  if (b > 1 && b < 5) return few
  if (b === 1) return one
  return many
}

// ─── Форматирование часов (для рендера и aside-sub) ──────────────────────
// При больших объёмах команды «440 ч» выглядит абстрактно — переводим в
// рабочие дни (по 8 часов) с порога 40 ч (5 рабочих дней): становится
// «55 дн» или «55 дн 4 ч», что куда нагляднее на табло.
const HOURS_PER_DAY = 8
const DAY_THRESHOLD = 40

function formatHoursShort(val) {
  const hours = num(val)
  if (hours <= 0) return '0 ч'

  if (hours >= DAY_THRESHOLD) {
    const days = Math.floor(hours / HOURS_PER_DAY)
    const remainHours = Math.round(hours - days * HOURS_PER_DAY)
    if (remainHours === 0) return `${days} дн`
    return `${days} дн ${remainHours} ч`
  }

  const totalMinutes = Math.round(hours * 60)
  const h = Math.floor(totalMinutes / 60)
  const m = totalMinutes % 60
  if (h === 0) return `${m} мин`
  if (m === 0) return `${h} ч`
  return `${h} ч ${m} мин`
}

// ─── Animated counter (inline child component) ────────────────────────────
const TvCount = defineComponent({
  name: 'TvCount',
  props: {
    value:  { type: [Number, String], default: 0 },
    format: { type: String, default: 'int' }, // 'int' | 'hours'
    prefix: { type: String, default: '' },
  },
  setup(props) {
    const display = ref(0)
    let raf = null
    function animateTo(target) {
      cancelAnimationFrame(raf)
      const start = performance.now()
      const startVal = display.value
      const duration = 900
      const step = (now) => {
        const t = Math.min(1, (now - start) / duration)
        const eased = 1 - Math.pow(1 - t, 3)
        display.value = startVal + (Number(target) - startVal) * eased
        if (t < 1) raf = requestAnimationFrame(step)
      }
      raf = requestAnimationFrame(step)
    }
    onMounted(() => animateTo(num(props.value)))
    // При смене props.value (например, переключение периода в KPI rail)
    // плавно «доезжаем» до нового значения вместо мгновенного скачка.
    watch(() => num(props.value), v => animateTo(v))
    onBeforeUnmount(() => cancelAnimationFrame(raf))

    return () => {
      const v = display.value
      let text = ''
      if (props.format === 'hours') text = formatHoursShort(v)
      else text = String(Math.round(v))
      return h('span', { class: 'tv-count' }, props.prefix + text)
    }
  },
})

// ─── KPI tile (inline) ────────────────────────────────────────────────────
const TvKpiTile = defineComponent({
  name: 'TvKpiTile',
  components: { TvCount },
  props: {
    tone:   { type: String, default: 'primary' },
    icon:   { type: String, required: true },
    label:  { type: String, required: true },
    value:  { type: [Number, String], default: 0 },
    format: { type: String, default: 'int' },
    prefix: { type: String, default: '' },
  },
  setup(props) {
    return () => h('div', { class: ['tv-kpi', 'tone-' + props.tone] }, [
      h('div', { class: 'tv-kpi-row' }, [
        h('span', { class: 'tv-kpi-ico material-symbols-outlined' }, props.icon),
        h('div', { class: 'tv-kpi-label' }, props.label),
      ]),
      h('div', { class: 'tv-kpi-value' }, [
        h(TvCount, {
          value: Number(props.value) || 0,
          format: props.format,
          prefix: props.prefix,
        }),
      ]),
    ])
  },
})

// ─── Loading & navigation ────────────────────────────────────────────────
async function loadPeriod(period, { silent = false } = {}) {
  const { from, to } = makeRange(period)
  if (!silent) slideLoading.value = true
  try {
    const [common, extended] = await Promise.all([
      getStatsCommon(from, to),
      getStatsExtended(from, to),
    ])
    commonByPeriod.value = { ...commonByPeriod.value, [period]: common }
    extendedByPeriod.value = { ...extendedByPeriod.value, [period]: extended }
  } catch { /* табло на стене — молчим */ }
  finally {
    if (!silent) slideLoading.value = false
  }
}

let slideTimer = null
let refreshTimer = null
let clockTimer = null
let aiFactTimer = null

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
  const period = slides[idx].period
  if (!commonByPeriod.value[period]) await loadPeriod(period)
  // При каждом заходе на брендовый слайд — берём новую цитату.
  if (slides[idx].kind === 'brand') brandQuote.value = pickRandomQuote()
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

function bumpControls() {
  controlsVisible.value = true
  clearTimeout(controlsTimer)
  controlsTimer = setTimeout(() => { controlsVisible.value = false }, CONTROLS_HIDE_MS)
}

function tickClock() {
  const d = new Date()
  clock.value = d.toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
  todayLabel.value = d.toLocaleDateString('ru-RU', { weekday: 'short', day: 'numeric', month: 'short' })
  longDateLabel.value = d.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

onMounted(async () => {
  themeStore.init()
  tickClock()
  clockTimer = setInterval(tickClock, 30_000)
  document.addEventListener('fullscreenchange', onFsChange)

  // Стартуем с первого слайда; параллельно грузим всех 3 периода.
  loadUsers()
  await loadPeriod('day')
  loadPeriod('week', { silent: true })
  loadPeriod('month', { silent: true })

  refreshTimer = setInterval(() => {
    loadPeriod('day', { silent: true })
    loadPeriod('week', { silent: true })
    loadPeriod('month', { silent: true })
  }, REFRESH_MS)

  // AI-факт обновляется на сервере до 6 раз в день — раз в час хватит,
  // чтобы поймать новый слот.
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
.tv-live-dot--small { width: 6px; height: 6px; }

@keyframes tv-live-pulse {
  0%   { box-shadow: 0 0 0 0 color-mix(in oklch, var(--color-error) 65%, transparent); }
  100% { box-shadow: 0 0 0 16px color-mix(in oklch, var(--color-error) 0%, transparent); }
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

.tv-kpi {
  border-radius: clamp(14px, 1.6vmin, 22px);
  padding: clamp(12px, 1.6vmin, 20px);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: 8px;
  min-height: 0;
  position: relative;
  overflow: hidden;
}

.tv-kpi::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 100% 0%, var(--kpi-glow, transparent), transparent 65%);
  opacity: 0.6;
  pointer-events: none;
}

.tv-kpi.tone-primary   { --kpi-glow: color-mix(in oklch, var(--color-primary) 20%, transparent); }
.tv-kpi.tone-secondary { --kpi-glow: color-mix(in oklch, var(--color-secondary) 20%, transparent); }
.tv-kpi.tone-tertiary  { --kpi-glow: color-mix(in oklch, var(--color-tertiary) 20%, transparent); }
.tv-kpi.tone-success   { --kpi-glow: color-mix(in oklch, var(--color-success) 22%, transparent); }
.tv-kpi.tone-warning   { --kpi-glow: color-mix(in oklch, var(--color-warning) 22%, transparent); }
.tv-kpi.tone-error     { --kpi-glow: color-mix(in oklch, var(--color-error) 22%, transparent); }

.tv-kpi-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.tv-kpi-ico {
  font-size: clamp(20px, 2.2vmin, 26px);
  color: var(--color-text-dim);
}

.tv-kpi.tone-primary   .tv-kpi-ico { color: var(--color-primary); }
.tv-kpi.tone-secondary .tv-kpi-ico { color: var(--color-secondary); }
.tv-kpi.tone-tertiary  .tv-kpi-ico { color: var(--color-tertiary); }
.tv-kpi.tone-success   .tv-kpi-ico { color: var(--color-success); }
.tv-kpi.tone-warning   .tv-kpi-ico { color: var(--color-warning); }
.tv-kpi.tone-error     .tv-kpi-ico { color: var(--color-error); }

.tv-kpi-label {
  font-size: clamp(11px, 1.3vmin, 15px);
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.06em;
  font-weight: 700;
}

.tv-kpi-value {
  font-size: clamp(28px, 4.2vmin, 56px);
  font-weight: 800;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}

.tv-kpi.tone-primary   .tv-kpi-value { color: var(--color-primary); }
.tv-kpi.tone-secondary .tv-kpi-value { color: var(--color-secondary); }
.tv-kpi.tone-tertiary  .tv-kpi-value { color: var(--color-tertiary); }
.tv-kpi.tone-success   .tv-kpi-value { color: var(--color-success); }
.tv-kpi.tone-warning   .tv-kpi-value { color: var(--color-warning); }
.tv-kpi.tone-error     .tv-kpi-value { color: var(--color-error); }

/* ════════════════ STAGE (main content) ═════════════════════════════ */
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

.tv-stage-eyebrow {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  font-size: clamp(14px, 1.8vmin, 20px);
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-text);
}

.tv-stage-eyebrow .material-symbols-outlined {
  font-size: clamp(22px, 2.6vmin, 30px);
}

.tv-stage-empty {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: clamp(16px, 2vmin, 22px);
  color: var(--color-text-dim);
}

/* ──────── HERO NUMBER ──────── */
.tv-hero {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: clamp(10px, 1.4vmin, 18px);
  text-align: center;
}

.tv-hero-glow {
  position: relative;
  padding: clamp(20px, 3vmin, 40px) clamp(40px, 6vmin, 80px);
}

.tv-hero-glow::before {
  content: '';
  position: absolute;
  inset: -10%;
  background: radial-gradient(circle, color-mix(in oklch, var(--glow) 28%, transparent), transparent 65%);
  filter: blur(8px);
  z-index: 0;
  pointer-events: none;
}

.tv-hero-number {
  position: relative;
  z-index: 1;
  font-size: clamp(72px, 16vmin, 240px);
  font-weight: 900;
  line-height: 0.9;
  letter-spacing: -0.04em;
  font-variant-numeric: tabular-nums;
}

.tv-hero-caption {
  font-size: clamp(14px, 1.8vmin, 22px);
  color: var(--color-text-dim);
  max-width: 560px;
  line-height: 1.4;
}

.tv-hero-secondaries {
  display: flex;
  gap: clamp(20px, 3vmin, 50px);
  margin-top: clamp(8px, 1.2vmin, 18px);
}

.tv-hero-sec {
  display: flex;
  flex-direction: column;
  gap: 4px;
  align-items: center;
}

.tv-hero-sec-label {
  font-size: clamp(11px, 1.2vmin, 14px);
  color: var(--color-text-dim);
  text-transform: uppercase;
  letter-spacing: 0.08em;
  font-weight: 700;
}

.tv-hero-sec-value {
  font-size: clamp(28px, 4vmin, 56px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  line-height: 1;
}

/* ──────── PODIUM ──────── */
.tv-podium-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(14px, 1.6vmin, 22px);
  min-height: 0;
}

.tv-podium {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: clamp(10px, 1.4vmin, 18px);
  align-items: end;
  min-height: 0;
}

.tv-podium-col {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: clamp(8px, 1vmin, 14px);
  text-align: center;
  min-width: 0;
  animation: tv-podium-rise 0.7s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
}

.tv-podium-col--1 { animation-delay: 0.25s; }
.tv-podium-col--2 { animation-delay: 0.1s; }
.tv-podium-col--3 { animation-delay: 0.4s; }

@keyframes tv-podium-rise {
  from { opacity: 0; transform: translateY(40px); }
  to   { opacity: 1; transform: translateY(0); }
}

.tv-podium-medal {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: clamp(36px, 4vmin, 56px);
  height: clamp(36px, 4vmin, 56px);
  border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  position: relative;
}

.tv-podium-col--1 .tv-podium-medal {
  background: color-mix(in oklch, var(--color-warning) 90%, white);
  color: var(--color-on-warning);
  box-shadow: 0 0 24px color-mix(in oklch, var(--color-warning) 50%, transparent);
}
.tv-podium-col--2 .tv-podium-medal {
  background: color-mix(in oklch, var(--color-outline) 30%, var(--color-surface-high));
  color: var(--color-text);
}
.tv-podium-col--3 .tv-podium-medal {
  background: color-mix(in oklch, var(--color-tertiary) 50%, var(--color-surface-high));
  color: var(--color-on-tertiary-container);
}

.tv-podium-place {
  font-size: clamp(16px, 2vmin, 24px);
  font-weight: 900;
}

.tv-fire {
  position: absolute;
  top: clamp(-16px, -1.6vmin, -12px);
  right: clamp(-14px, -1.4vmin, -10px);
  font-size: clamp(22px, 2.6vmin, 32px);
  color: var(--color-warning);
  filter: drop-shadow(0 0 6px color-mix(in oklch, var(--color-warning) 60%, transparent));
  animation: tv-flame 1.4s ease-in-out infinite;
  font-variation-settings: 'FILL' 1;
}

@keyframes tv-flame {
  0%, 100% { transform: scale(1) rotate(-3deg); }
  50%      { transform: scale(1.18) rotate(4deg); }
}

.tv-podium-avatar-wrap {
  width: clamp(64px, 10vmin, 140px);
  height: clamp(64px, 10vmin, 140px);
  border-radius: 50%;
  border: clamp(3px, 0.4vmin, 5px) solid var(--color-primary);
  overflow: hidden;
  background: var(--color-surface-high);
  flex-shrink: 0;
}

.tv-podium-col--1 .tv-podium-avatar-wrap {
  border-color: var(--color-warning);
  width: clamp(80px, 13vmin, 170px);
  height: clamp(80px, 13vmin, 170px);
  box-shadow: 0 0 36px color-mix(in oklch, var(--color-warning) 35%, transparent);
}

.tv-podium-col--2 .tv-podium-avatar-wrap { border-color: var(--color-outline); }
.tv-podium-col--3 .tv-podium-avatar-wrap { border-color: var(--color-tertiary); }

.tv-podium-avatar {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.tv-podium-fio {
  font-size: clamp(14px, 1.8vmin, 22px);
  font-weight: 700;
  color: var(--color-text);
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tv-podium-hours {
  font-size: clamp(16px, 2.4vmin, 30px);
  font-weight: 800;
  color: var(--color-primary);
  font-variant-numeric: tabular-nums;
}

.tv-podium-col--1 .tv-podium-hours { color: var(--color-warning); }
.tv-podium-col--3 .tv-podium-hours { color: var(--color-tertiary); }

.tv-podium-base {
  width: 100%;
  background: color-mix(in oklch, var(--color-primary) 14%, var(--color-surface-low));
  border-radius: 12px 12px 0 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 900;
  font-size: clamp(28px, 4vmin, 56px);
  color: color-mix(in oklch, var(--color-text) 35%, transparent);
  height: clamp(36px, 5vmin, 70px);
  margin-top: clamp(4px, 0.8vmin, 10px);
}

.tv-podium-col--1 .tv-podium-base {
  background: color-mix(in oklch, var(--color-warning) 22%, var(--color-surface-low));
  height: clamp(58px, 8vmin, 110px);
}
.tv-podium-col--2 .tv-podium-base {
  background: color-mix(in oklch, var(--color-outline) 30%, var(--color-surface-low));
  height: clamp(46px, 6vmin, 88px);
}
.tv-podium-col--3 .tv-podium-base {
  background: color-mix(in oklch, var(--color-tertiary) 22%, var(--color-surface-low));
  height: clamp(34px, 4.5vmin, 66px);
}

.tv-podium-place-empty {
  font-size: clamp(28px, 4vmin, 60px);
  font-weight: 900;
  color: var(--color-outline);
  margin: clamp(20px, 3vmin, 40px) 0;
}

.tv-podium-empty-text {
  color: var(--color-text-dim);
  font-size: clamp(14px, 1.6vmin, 18px);
}

/* ──────── RANKING ──────── */
.tv-ranking-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-ranking {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
  display: grid;
  grid-template-rows: repeat(5, 1fr);
  gap: clamp(6px, 0.8vmin, 12px);
  min-height: 0;
}

.tv-ranking-item {
  display: grid;
  grid-template-columns: clamp(40px, 4.6vmin, 60px)
                         clamp(40px, 5.4vmin, 70px)
                         minmax(0, 1.4fr)
                         minmax(0, 2.4fr)
                         auto;
  gap: clamp(10px, 1.4vmin, 18px);
  align-items: center;
  padding: clamp(6px, 0.8vmin, 12px) clamp(10px, 1.4vmin, 18px);
  background: color-mix(in oklch, var(--color-surface-high) 60%, transparent);
  border-radius: clamp(12px, 1.4vmin, 18px);
  animation: tv-row-in 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  animation-delay: var(--row-delay, 0ms);
}

.tv-ranking-item.is-leader {
  background: color-mix(in oklch, var(--color-warning) 16%, var(--color-surface));
  box-shadow: 0 0 20px color-mix(in oklch, var(--color-warning) 18%, transparent) inset;
}

@keyframes tv-row-in {
  from { opacity: 0; transform: translateX(-30px); }
  to   { opacity: 1; transform: translateX(0); }
}

.tv-ranking-rank {
  font-size: clamp(20px, 2.6vmin, 32px);
  font-weight: 900;
  color: var(--color-primary);
  text-align: center;
  position: relative;
}

.tv-ranking-item.is-leader .tv-ranking-rank {
  color: var(--color-warning);
}

.tv-ranking-rank .tv-fire {
  position: static;
  font-size: clamp(26px, 3vmin, 40px);
}

.tv-ranking-avatar {
  width: clamp(40px, 5.4vmin, 70px);
  height: clamp(40px, 5.4vmin, 70px);
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--color-primary);
}

.tv-ranking-item.is-leader .tv-ranking-avatar { border-color: var(--color-warning); }

.tv-ranking-fio {
  font-size: clamp(15px, 2vmin, 24px);
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-ranking-bar {
  height: clamp(10px, 1.4vmin, 16px);
  background: var(--color-outline-dim);
  border-radius: 999px;
  overflow: hidden;
}

.tv-ranking-bar-fill {
  height: 100%;
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-primary) 80%, transparent),
    var(--color-primary));
  border-radius: 999px;
  width: 0;
  animation: tv-bar-fill 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  animation-delay: calc(var(--row-delay, 0ms) + 200ms);
}

.tv-ranking-item.is-leader .tv-ranking-bar-fill {
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-warning) 80%, transparent),
    var(--color-warning));
}

@keyframes tv-bar-fill {
  from { width: 0; }
  to   { width: var(--bar-width); }
}

.tv-ranking-value {
  font-size: clamp(15px, 2vmin, 24px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
  white-space: nowrap;
}

/* ──────── DEPARTMENTS ──────── */
.tv-deps-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-deps {
  flex: 1;
  display: grid;
  grid-template-rows: repeat(auto-fit, 1fr);
  gap: clamp(8px, 1vmin, 14px);
  min-height: 0;
}

.tv-dep-bar {
  display: grid;
  grid-template-columns: minmax(120px, 26%) 1fr;
  gap: clamp(12px, 1.6vmin, 20px);
  align-items: center;
  animation: tv-row-in 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  animation-delay: var(--row-delay, 0ms);
}

.tv-dep-bar-label {
  font-size: clamp(15px, 1.9vmin, 22px);
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-dep-bar-track {
  height: clamp(28px, 4vmin, 50px);
  background: color-mix(in oklch, var(--color-surface-high) 70%, transparent);
  border-radius: 12px;
  overflow: hidden;
  position: relative;
}

.tv-dep-bar-fill {
  height: 100%;
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-tertiary) 60%, transparent),
    var(--color-tertiary));
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: flex-end;
  padding-right: clamp(10px, 1.4vmin, 18px);
  color: var(--color-on-tertiary-container);
  font-weight: 900;
  font-size: clamp(14px, 1.8vmin, 22px);
  width: 0;
  animation: tv-bar-fill 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  animation-delay: calc(var(--row-delay, 0ms) + 200ms);
  font-variant-numeric: tabular-nums;
}

/* ──────── QUAD (4 tiles) ──────── */
.tv-quad-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-quad {
  flex: 1;
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  grid-template-rows: repeat(2, 1fr);
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-quad-tile {
  border-radius: clamp(14px, 1.8vmin, 22px);
  padding: clamp(16px, 2vmin, 28px);
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  gap: clamp(6px, 0.8vmin, 12px);
  position: relative;
  overflow: hidden;
  animation: tv-tile-in 0.65s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
}

.tv-quad-tile:nth-child(1) { animation-delay: 80ms; }
.tv-quad-tile:nth-child(2) { animation-delay: 180ms; }
.tv-quad-tile:nth-child(3) { animation-delay: 280ms; }
.tv-quad-tile:nth-child(4) { animation-delay: 380ms; }

@keyframes tv-tile-in {
  from { opacity: 0; transform: scale(0.92); }
  to   { opacity: 1; transform: scale(1); }
}

.tv-quad-tile.tone-primary   { background: var(--color-primary-container);   color: var(--color-on-primary-container); }
.tv-quad-tile.tone-secondary { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.tv-quad-tile.tone-tertiary  { background: var(--color-tertiary-container);  color: var(--color-on-tertiary-container); }
.tv-quad-tile.tone-success   { background: var(--color-success-container);   color: var(--color-on-success-container); }

.tv-quad-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: clamp(36px, 4vmin, 56px);
  height: clamp(36px, 4vmin, 56px);
  border-radius: 50%;
  background: color-mix(in oklch, currentColor 14%, transparent);
}

.tv-quad-icon .material-symbols-outlined {
  font-size: clamp(22px, 2.6vmin, 32px);
}

.tv-quad-num {
  font-size: clamp(40px, 7.4vmin, 110px);
  font-weight: 900;
  line-height: 0.9;
  letter-spacing: -0.03em;
  font-variant-numeric: tabular-nums;
}

.tv-quad-label {
  font-size: clamp(13px, 1.6vmin, 18px);
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  opacity: 0.85;
}

/* ──────── BRAND SLIDE ──────── */
.tv-brand-stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: clamp(12px, 1.6vmin, 22px);
  position: relative;
  text-align: center;
}

.tv-brand-glow {
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 50% 50%,
    color-mix(in oklch, var(--color-primary) 22%, transparent),
    transparent 60%);
  filter: blur(20px);
  pointer-events: none;
}

.tv-brand-big-logo {
  position: relative;
  width: clamp(96px, 14vmin, 220px);
  height: clamp(96px, 14vmin, 220px);
  border-radius: 50%;
  animation: tv-brand-pulse 3.6s ease-in-out infinite;
}

@keyframes tv-brand-pulse {
  0%, 100% { transform: scale(1); filter: drop-shadow(0 0 18px color-mix(in oklch, var(--color-primary) 35%, transparent)); }
  50%      { transform: scale(1.04); filter: drop-shadow(0 0 32px color-mix(in oklch, var(--color-primary) 55%, transparent)); }
}

.tv-brand-big-name {
  position: relative;
  font-size: clamp(36px, 6.4vmin, 96px);
  font-weight: 900;
  letter-spacing: 0.02em;
  color: var(--color-text);
}

.tv-brand-quote {
  position: relative;
  font-size: clamp(16px, 2.2vmin, 28px);
  color: var(--color-text-dim);
  font-style: italic;
}

.tv-brand-date {
  position: relative;
  font-size: clamp(14px, 1.8vmin, 22px);
  color: var(--color-text-dim);
  text-transform: capitalize;
  font-weight: 600;
}

/* ════════════════ AI FACT SLIDE ═════════════════════════════════════ */
.tv-ai-fact-stage {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: clamp(20px, 2.6vmin, 36px);
  padding: clamp(24px, 3vmin, 56px);
  position: relative;
  text-align: center;
}

.tv-ai-fact-glow {
  position: absolute;
  inset: 0;
  background: radial-gradient(ellipse at 50% 50%,
    color-mix(in oklch, var(--color-tertiary) 28%, transparent),
    transparent 65%);
  filter: blur(28px);
  pointer-events: none;
}

.tv-ai-fact-eyebrow {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: clamp(14px, 1.8vmin, 22px);
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-tertiary);
}
.tv-ai-fact-eyebrow .material-symbols-outlined {
  font-size: clamp(22px, 2.6vmin, 32px);
  font-variation-settings: 'FILL' 1;
  animation: tv-ai-fact-pulse 2.4s ease-in-out infinite;
}

@keyframes tv-ai-fact-pulse {
  0%, 100% { transform: scale(1); filter: drop-shadow(0 0 6px color-mix(in oklch, var(--color-tertiary) 50%, transparent)); }
  50%      { transform: scale(1.08); filter: drop-shadow(0 0 14px color-mix(in oklch, var(--color-tertiary) 75%, transparent)); }
}

.tv-ai-fact-text {
  position: relative;
  font-size: clamp(28px, 4.6vmin, 72px);
  line-height: 1.18;
  font-weight: 800;
  color: var(--color-text);
  max-width: 22ch;
  text-wrap: balance;
  animation: tv-ai-fact-rise 0.7s cubic-bezier(0.16, 1, 0.3, 1);
}

@keyframes tv-ai-fact-rise {
  from { opacity: 0; transform: translateY(14px); }
  to   { opacity: 1; transform: translateY(0); }
}

.tv-ai-fact-foot {
  position: relative;
  font-size: clamp(13px, 1.6vmin, 20px);
  color: var(--color-text-dim);
  text-transform: capitalize;
  font-weight: 600;
}

/* ════════════════ ASIDE CARD ════════════════════════════════════════ */
.tv-aside-rail {
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.tv-aside-card {
  flex: 1;
  border-radius: clamp(18px, 2.4vmin, 28px);
  padding: clamp(18px, 2.2vmin, 28px);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  display: flex;
  flex-direction: column;
  gap: clamp(10px, 1.4vmin, 18px);
  position: relative;
  overflow: hidden;
  min-height: 0;
}

.tv-aside-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(circle at 100% 0%,
    var(--aside-glow, color-mix(in oklch, var(--color-primary) 22%, transparent)),
    transparent 60%);
  pointer-events: none;
}

.tv-aside-card.tone-primary   { --aside-glow: color-mix(in oklch, var(--color-primary) 22%, transparent); }
.tv-aside-card.tone-secondary { --aside-glow: color-mix(in oklch, var(--color-secondary) 22%, transparent); }
.tv-aside-card.tone-tertiary  { --aside-glow: color-mix(in oklch, var(--color-tertiary) 22%, transparent); }
.tv-aside-card.tone-success   { --aside-glow: color-mix(in oklch, var(--color-success) 24%, transparent); }
.tv-aside-card.tone-warning   { --aside-glow: color-mix(in oklch, var(--color-warning) 24%, transparent); }

.tv-aside-eyebrow {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: clamp(11px, 1.3vmin, 14px);
  font-weight: 700;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

.tv-aside-eyebrow .material-symbols-outlined {
  font-size: clamp(16px, 1.8vmin, 22px);
  color: var(--color-primary);
}

.tv-aside-card.tone-secondary .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-success); }
.tv-aside-card.tone-warning   .tv-aside-eyebrow .material-symbols-outlined { color: var(--color-warning); }

.tv-aside-body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: clamp(8px, 1.2vmin, 14px);
  flex: 1;
  min-height: 0;
}

.tv-aside-headline {
  font-size: clamp(16px, 2.2vmin, 26px);
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.2;
  word-break: break-word;
}

.tv-aside-value {
  font-size: clamp(40px, 6.4vmin, 96px);
  font-weight: 900;
  line-height: 0.95;
  letter-spacing: -0.03em;
  color: var(--color-primary);
  font-variant-numeric: tabular-nums;
}

.tv-aside-card.tone-secondary .tv-aside-value { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-aside-value { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-aside-value { color: var(--color-success); }
.tv-aside-card.tone-warning   .tv-aside-value { color: var(--color-warning); }

.tv-aside-sub {
  font-size: clamp(12px, 1.4vmin, 16px);
  color: var(--color-text-dim);
}

.tv-spark {
  flex: 1;
  min-height: clamp(60px, 8vmin, 110px);
  margin-top: auto;
}

.tv-spark svg { width: 100%; height: 100%; display: block; }

.tv-spark-line {
  fill: none;
  stroke: currentColor;
  stroke-width: 1.6;
  vector-effect: non-scaling-stroke;
  color: var(--color-primary);
  stroke-linecap: round;
  stroke-linejoin: round;
}

.tv-aside-card.tone-secondary .tv-spark-line { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-spark-line { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-spark-line { color: var(--color-success); }

.tv-spark-area {
  fill: currentColor;
  color: var(--color-primary);
  opacity: 0.18;
}

.tv-aside-card.tone-secondary .tv-spark-area { color: var(--color-secondary); }
.tv-aside-card.tone-tertiary  .tv-spark-area { color: var(--color-tertiary); }
.tv-aside-card.tone-success   .tv-spark-area { color: var(--color-success); }

/* ════════════════ TICKER ════════════════════════════════════════════ */
.tv-ticker {
  display: flex;
  align-items: center;
  gap: clamp(12px, 1.6vmin, 20px);
  padding: clamp(8px, 1.2vmin, 14px) clamp(20px, 2.6vmin, 36px);
  background: color-mix(in oklch, var(--color-surface) 80%, transparent);
  border-top: 1px solid var(--color-outline-dim);
  overflow: hidden;
}

.tv-ticker-mark {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: clamp(11px, 1.2vmin, 14px);
  font-weight: 800;
  letter-spacing: 0.16em;
  color: var(--color-error);
  text-transform: uppercase;
  flex-shrink: 0;
}

.tv-ticker-viewport {
  flex: 1;
  overflow: hidden;
  min-width: 0;
  mask-image: linear-gradient(90deg, transparent 0%, #000 4%, #000 96%, transparent 100%);
}

.tv-ticker-track {
  display: inline-flex;
  gap: clamp(28px, 4vmin, 60px);
  white-space: nowrap;
  animation: tv-ticker-scroll linear infinite;
  will-change: transform;
}

@keyframes tv-ticker-scroll {
  from { transform: translateX(0); }
  to   { transform: translateX(-50%); }
}

.tv-ticker-item {
  display: inline-flex;
  align-items: center;
  gap: clamp(8px, 1vmin, 12px);
  font-size: clamp(14px, 1.6vmin, 20px);
  color: var(--color-text);
  font-weight: 600;
}

.tv-ticker-bullet {
  color: var(--color-primary);
  font-size: 8px;
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
