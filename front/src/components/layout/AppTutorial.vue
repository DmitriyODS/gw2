<template>
  <Teleport to="body">
    <Transition name="tour-fade">
      <div v-if="tutorial.isOpen.value" class="tour" role="dialog" aria-modal="true">
        <!-- Затенение с «дыркой» для подсвеченного элемента (если есть таргет).
             Если таргета нет — просто полупрозрачный фон. -->
        <div class="tour-backdrop" @click="skip">
          <svg
            v-if="spotRect"
            class="tour-spot-svg"
            preserveAspectRatio="none"
            :width="vw"
            :height="vh"
            :viewBox="`0 0 ${vw} ${vh}`"
          >
            <defs>
              <mask id="tour-spot-mask">
                <rect :width="vw" :height="vh" fill="white" />
                <rect
                  :x="spotRect.x - PAD"
                  :y="spotRect.y - PAD"
                  :width="spotRect.width + PAD * 2"
                  :height="spotRect.height + PAD * 2"
                  :rx="RADIUS" :ry="RADIUS"
                  fill="black"
                />
              </mask>
            </defs>
            <rect :width="vw" :height="vh" class="tour-spot-bg" mask="url(#tour-spot-mask)" />
            <rect
              :x="spotRect.x - PAD"
              :y="spotRect.y - PAD"
              :width="spotRect.width + PAD * 2"
              :height="spotRect.height + PAD * 2"
              :rx="RADIUS" :ry="RADIUS"
              class="tour-spot-ring"
            />
          </svg>
        </div>

        <!-- Карточка с шагом: шапка и футер фиксированы, скроллится только текст -->
        <div ref="cardEl" class="tour-card" :style="cardStyle" @click.stop>
          <header class="tour-head">
            <div class="tour-icon" :data-tone="step.tone || 'primary'">
              <span class="material-symbols-outlined">{{ step.icon }}</span>
            </div>
            <div class="tour-head-text">
              <span class="tour-step-label">Шаг {{ stepIndex + 1 }} из {{ steps.length }}</span>
              <h3 class="tour-title">{{ step.title }}</h3>
            </div>
            <button class="tour-close" @click="skip" aria-label="Пропустить">
              <span class="material-symbols-outlined">close</span>
            </button>
          </header>

          <div class="tour-body">
            <p class="tour-text">{{ step.text }}</p>
            <p v-if="step.tip" class="tour-tip">
              <span class="material-symbols-outlined">tips_and_updates</span>
              {{ step.tip }}
            </p>
          </div>

          <footer class="tour-actions">
            <button v-if="stepIndex > 0" class="btn-text" @click="prev">
              <span class="material-symbols-outlined">arrow_back</span>
              Назад
            </button>
            <button class="btn-filled" @click="next">
              {{ isLast ? 'Готово' : 'Дальше' }}
              <span v-if="!isLast" class="material-symbols-outlined">arrow_forward</span>
              <span v-else class="material-symbols-outlined">check</span>
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import { useRouter } from 'vue-router'
import { useTutorial } from '@/composables/useTutorial.js'
import { usePermission } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'

const tutorial = useTutorial()
const router = useRouter()
const { isAtLeast, canManageCompanies, ROLES: R } = usePermission()
const { usesGroove } = useCompanySettings()

// Роль ≥ Сотрудник означает «есть активная компания» (у супер-админа её нет).
const hasCompany = () => isAtLeast(R.EMPLOYEE)

const PAD = 8
const RADIUS = 12
const CARD_W = 440
const CARD_GAP = 16

const vw = ref(window.innerWidth)
const vh = ref(window.innerHeight)
const spotRect = ref(null)
const stepIndex = ref(0)

// Реальная высота карточки (следит ResizeObserver) — позиционирование по ней,
// а не по константе: тексты шагов разной длины, и карточка не должна уезжать
// за нижнюю границу окна.
const cardEl = ref(null)
const cardH = ref(320)
let cardRO = null
watch(cardEl, (el) => {
  cardRO?.disconnect()
  cardRO = null
  if (!el) return
  cardRO = new ResizeObserver(() => { cardH.value = el.offsetHeight })
  cardRO.observe(el)
})

// Каталог шагов. Каждый — id, иконка, тон, заголовок, текст, опц. подсказка,
// target (CSS-селектор, может быть null — карточка по центру), navigateTo и
// check (роль). Без onEnter-побочек: новый тур ничего не создаёт в системе.
const allSteps = computed(() => [
  {
    id: 'welcome',
    icon: 'waving_hand', tone: 'primary',
    title: 'Добро пожаловать в Groove Work',
    text: 'Платформа для работы команды: задачи и учёт времени, мессенджер со звонками, корпоративный портал и личные инструменты — заметки, ежедневники, календари. Короткий тур по разделам; пропустить можно в любой момент — крестик справа сверху.',
    target: null,
  },
  {
    id: 'tasks',
    icon: 'dashboard_customize', tone: 'primary',
    title: 'Задачи',
    text: 'Главный рабочий раздел: задачи компании, поиск, фильтры, избранное и архив. Внутри задачи запускаются юниты — отрезки рабочего времени, из которых складывается вся статистика.',
    tip: 'Одновременно может быть только один активный юнит. Цвет-тег задачи — личный: ваш цвет видите только вы.',
    target: '[data-tutorial="nav-tasks"]',
    navigateTo: '/tasks',
    check: hasCompany,
  },
  {
    id: 'registries',
    icon: 'view_agenda', tone: 'secondary',
    title: 'Реестры',
    text: 'Настраиваемые таблицы-справочники компании: клиенты, оборудование, договоры — что угодно. Администратор собирает структуру карточки из полей разных типов, записи ведут все. Есть сквозной поиск, экспорт в Excel и публичные ссылки только для чтения.',
    target: '[data-tutorial="nav-registries"]',
    navigateTo: '/registries',
    check: hasCompany,
  },
  {
    id: 'notes',
    icon: 'note_stack', tone: 'tertiary',
    title: 'Заметки и ежедневники',
    text: 'Личное пространство, не привязанное к компании. Заметки — полноценный редактор с форматированием, картинками и таблицами, группы-фильтры и публичные ссылки. Ежедневники — задачи по дням с архивом выполненного; ежедневником можно поделиться с коллегой.',
    tip: 'Запись ежедневника перетаскивается на другой день, а из карточки создаётся полноценная задача компании.',
    target: '[data-tutorial="nav-diaries"]',
    navigateTo: '/notes',
  },
  {
    id: 'calendars',
    icon: 'calendar_month', tone: 'primary',
    title: 'Календари',
    text: 'Общие календари компании: события с настраиваемыми карточками, просмотр по дню, неделе и месяцу, экспорт за период и публичные ссылки. Удобно для дежурств, брони переговорок и общих планов.',
    target: '[data-tutorial="nav-calendars"]',
    navigateTo: '/calendars',
    check: hasCompany,
  },
  {
    id: 'messenger',
    icon: 'chat', tone: 'secondary',
    title: 'Мессенджер',
    text: 'Личные чаты: файлы, ответы, пересылка, закрепление сообщений и чатов. Отсюда же звонки — голосом или с видео, с демонстрацией экрана и приглашением гостей по ссылке, прямо внутри платформы.',
    tip: 'Мессенджер и звонки работают даже без компании — это ваш личный аккаунт.',
    target: '[data-tutorial="nav-messenger"]',
    navigateTo: '/messenger',
  },
  {
    id: 'portal',
    icon: 'brand_awareness', tone: 'tertiary',
    title: 'Портал',
    text: 'Внутренняя жизнь компании: лента постов с комментариями и реакциями, тематические разделы, закрепление важного и пересылка поста в мессенджер. Рядом, на вкладке «Сотрудники», — все коллеги: кто в сети, профиль, быстрые «Написать» и «Позвонить».',
    tip: 'Бейдж на пункте «Портал» показывает непрочитанные посты.',
    target: '[data-tutorial="nav-portal"]',
    navigateTo: '/portal',
    check: hasCompany,
  },
  {
    id: 'groove',
    icon: 'pets', tone: 'primary',
    title: 'Грувики',
    text: 'Питомец, который растёт от вашей работы: юниты и закрытые задачи приносят опыт и кудосы. Кормите его, гуляйте, отправляйте в приключения и гладьте питомцев коллег. Живёт плавающим компаньоном поверх интерфейса, а магазин, рейтинг недели и питомцы коллег — на отдельной странице.',
    tip: 'Грувик может заболеть, если долго не работать. Лечится юнитами, задачами, прогулкой и заботой коллег.',
    target: '[data-tutorial="nav-groove"]',
    navigateTo: '/pets',
    check: () => hasCompany() && usesGroove.value,
  },
  {
    id: 'assistant',
    icon: 'smart_toy', tone: 'secondary',
    title: 'Плавающий хаб: ассистент и сообщения',
    text: 'Кнопка в правом нижнем углу открывает мини-хаб, доступный из любого раздела. Внутри две вкладки: ИИ-ассистент, который отвечает на вопросы о задачах и статистике компании по реальным данным, и компактный мессенджер, чтобы не уходить со страницы.',
    target: '[data-tutorial="mini-hub"]',
  },
  {
    id: 'stats',
    icon: 'bar_chart', tone: 'primary',
    title: 'Статистика',
    text: 'Сколько часов команда отработала за период: по сотрудникам, отделам и типам юнитов, с карточкой «Ответственные по задачам» и экспортом в Excel. Отсюда же включается ТВ-режим — живое табло для офисного экрана.',
    target: '[data-tutorial="nav-stats"]',
    navigateTo: '/stats',
    check: hasCompany,
  },
  {
    id: 'companies',
    icon: 'business_center', tone: 'tertiary',
    title: 'Мои компании',
    text: 'Управление компаниями, где вы администратор: настройки, участники и роли, приглашения по ссылке или на email. Один аккаунт может состоять в нескольких компаниях — активная переключается в шапке бокового меню.',
    tip: 'Любой пользователь может создать свою компанию и стать её администратором.',
    target: '[data-tutorial="nav-companies"]',
    navigateTo: '/companies',
    check: () => canManageCompanies(),
  },
  {
    id: 'settings',
    icon: 'settings', tone: 'secondary',
    title: 'Настройки',
    text: 'Внешний вид, справка по разделам и «О приложении» — версия, что нового и быстрая кнопка «Написать в техподдержку».',
    target: '[data-tutorial="nav-settings"]',
    navigateTo: '/settings',
  },
  {
    id: 'theme',
    icon: 'palette', tone: 'primary',
    title: 'Свой стиль',
    text: 'В разделе «Внешний вид» соберите тему из четырёх цветов или выберите готовую, включите градиентный фон и его вариации. «Мне повезёт» — для смелых.',
    tip: 'Светлая, тёмная или системная тема переключается отдельным сегментом.',
    target: '[data-tutorial="settings-section-theme"]',
    onEnter: () => document.querySelector('[data-tutorial="settings-section-theme"]')?.click(),
  },
  {
    id: 'help',
    icon: 'help_center', tone: 'tertiary',
    title: 'Справка',
    text: 'Подробные статьи по каждому разделу: как устроены задачи и юниты, статистика, реестры и остальное. Если что-то забудется после тура — ответ здесь.',
    target: '[data-tutorial="settings-section-help"]',
    onEnter: () => document.querySelector('[data-tutorial="settings-section-help"]')?.click(),
  },
  {
    id: 'profile',
    icon: 'account_circle', tone: 'primary',
    title: 'Профиль и аккаунт',
    text: 'Кликните по аватару в сайдбаре или нижней навигации — попадёте в свой аккаунт. Там фото, телефон, email и пароль.',
    target: '[data-tutorial="profile-avatar"]',
  },
  {
    id: 'done',
    icon: 'celebration', tone: 'primary',
    title: 'Готово',
    text: 'Это всё базовое. Тур всегда можно повторить из «О приложении», а подробная справка по разделам — в Настройках → Справка. Удачной работы!',
    target: null,
  },
])

const steps = computed(() => allSteps.value.filter(s => !s.check || s.check()))
const step = computed(() => steps.value[stepIndex.value] || steps.value[0])
const isLast = computed(() => stepIndex.value >= steps.value.length - 1)

// Один и тот же data-tutorial-якорь есть и в sidebar, и в bottom-nav.
// Возвращаем первый ВИДИМЫЙ узел — иначе на мобильном бы вернулся скрытый
// sidebar с rect 0×0.
function findVisible(selector) {
  if (!selector) return null
  for (const el of document.querySelectorAll(selector)) {
    const r = el.getBoundingClientRect()
    if (r.width > 0 && r.height > 0) return el
  }
  return null
}

async function refreshSpot() {
  await nextTick()
  const s = step.value
  if (!s?.target) { spotRect.value = null; return }
  const el = findVisible(s.target)
  if (!el) { spotRect.value = null; return }
  const r = el.getBoundingClientRect()
  const outside = r.bottom < 0 || r.top > vh.value || r.right < 0 || r.left > vw.value
  if (outside) {
    el.scrollIntoView({ block: 'center', inline: 'center', behavior: 'smooth' })
    await new Promise(r => setTimeout(r, 220))
  }
  const r2 = el.getBoundingClientRect()
  spotRect.value = r2.width && r2.height ? r2 : null
}

async function applyStep() {
  const s = step.value
  if (!s) return
  if (s.navigateTo && router.currentRoute.value.path !== s.navigateTo) {
    try { await router.push(s.navigateTo) } catch {}
    await new Promise(r => setTimeout(r, 200))
  }
  if (s.onEnter) {
    try { s.onEnter() } catch {}
    await nextTick()
    await new Promise(r => setTimeout(r, 60))
  }
  await refreshSpot()
}

watch(stepIndex, applyStep)
watch(() => tutorial.isOpen.value, async (v) => {
  if (!v) return
  if (tutorial.startAtId.value) {
    const idx = steps.value.findIndex(s => s.id === tutorial.startAtId.value)
    if (idx >= 0) stepIndex.value = idx
    else stepIndex.value = 0
  } else {
    stepIndex.value = 0
  }
  await applyStep()
})

function next() {
  if (isLast.value) { tutorial.close(); return }
  stepIndex.value++
}
function prev() { if (stepIndex.value > 0) stepIndex.value-- }
function skip() { tutorial.close() }

function onResize() {
  vw.value = window.innerWidth
  vh.value = window.innerHeight
  refreshSpot()
}
function onKeydown(e) {
  if (!tutorial.isOpen.value) return
  if (e.key === 'Escape') skip()
  else if (e.key === 'ArrowRight' || e.key === 'Enter') next()
  else if (e.key === 'ArrowLeft') prev()
}

onMounted(() => {
  window.addEventListener('resize', onResize)
  window.addEventListener('scroll', refreshSpot, true)
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', onResize)
  window.removeEventListener('scroll', refreshSpot, true)
  window.removeEventListener('keydown', onKeydown)
  cardRO?.disconnect()
})

// Позиционируем карточку: рядом с подсвеченным элементом (если влезает) —
// иначе по центру экрана. На мобильных (узких) — всегда снизу с safe-area.
const cardStyle = computed(() => {
  const mobile = vw.value <= 720
  if (mobile) {
    return {
      left: '12px',
      right: '12px',
      bottom: 'calc(12px + env(safe-area-inset-bottom, 0px))',
      width: 'auto',
      maxWidth: 'none',
    }
  }
  const h = cardH.value
  if (!spotRect.value) {
    return {
      left: `${Math.max(16, (vw.value - CARD_W) / 2)}px`,
      top: `${Math.max(16, (vh.value - h) / 2)}px`,
      width: `${CARD_W}px`,
    }
  }
  const r = spotRect.value
  // Пытаемся справа от spot, потом слева, потом снизу, потом сверху.
  // Вертикаль всегда клампится по реальной высоте — кнопки не должны
  // оказаться ниже края окна.
  const fitsRight = r.right + CARD_GAP + CARD_W < vw.value - 12
  const fitsLeft = r.left - CARD_GAP - CARD_W > 12
  const fitsBelow = r.bottom + CARD_GAP + h < vh.value - 12
  let left, top
  if (fitsRight) {
    left = r.right + CARD_GAP
    top = clamp(r.top, 16, vh.value - h - 16)
  } else if (fitsLeft) {
    left = r.left - CARD_GAP - CARD_W
    top = clamp(r.top, 16, vh.value - h - 16)
  } else if (fitsBelow) {
    left = clamp(r.left, 16, vw.value - CARD_W - 16)
    top = clamp(r.bottom + CARD_GAP, 16, vh.value - h - 16)
  } else {
    left = clamp(r.left, 16, vw.value - CARD_W - 16)
    top = Math.max(16, Math.min(r.top - CARD_GAP - h, vh.value - h - 16))
  }
  return { left: `${left}px`, top: `${top}px`, width: `${CARD_W}px` }
})

function clamp(v, lo, hi) { return Math.min(Math.max(v, lo), hi) }
</script>

<style scoped>
.tour {
  position: fixed;
  inset: 0;
  z-index: 11000;
}

.tour-backdrop {
  position: absolute;
  inset: 0;
  cursor: pointer;
}

.tour-spot-svg {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
}

.tour-spot-bg {
  fill: var(--color-scrim);
  opacity: 0.62;
}

.tour-spot-ring {
  fill: none;
  stroke: var(--color-primary);
  stroke-width: 3;
  filter: drop-shadow(0 0 12px color-mix(in oklch, var(--color-primary) 60%, transparent));
}

/* Когда таргета нет — равномерное затенение всей области. */
.tour-backdrop:not(:has(.tour-spot-svg)) {
  background: color-mix(in oklch, var(--color-scrim) 62%, transparent);
}

.tour-card {
  position: absolute;
  background: var(--acrylic-bg);
  backdrop-filter: var(--acrylic-blur);
  -webkit-backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-xl);
  padding: 20px 22px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-height: calc(100dvh - 32px);
  overflow: hidden;
}

.tour-head {
  display: flex;
  align-items: flex-start;
  gap: 14px;
  flex-shrink: 0;
}

.tour-head-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.tour-step-label {
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.3px;
  text-transform: uppercase;
  color: var(--color-primary);
}

.tour-close {
  flex-shrink: 0;
  width: 32px;
  height: 32px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  color: var(--color-text-dim);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.tour-close:hover { background: var(--color-error-container); color: var(--color-on-error-container); }
.tour-close .material-symbols-outlined { font-size: 18px; }

.tour-icon {
  flex-shrink: 0;
  width: 48px;
  height: 48px;
  border-radius: var(--radius-md);
  display: flex;
  align-items: center;
  justify-content: center;
}
.tour-icon[data-tone="primary"]   { background: var(--color-primary-container);   color: var(--color-on-primary-container); }
.tour-icon[data-tone="secondary"] { background: var(--color-secondary-container); color: var(--color-on-secondary-container); }
.tour-icon[data-tone="tertiary"]  { background: var(--color-tertiary-container);  color: var(--color-on-tertiary-container); }
.tour-icon .material-symbols-outlined { font-size: 26px; }

.tour-title {
  margin: 0;
  font-size: 19px;
  font-weight: 800;
  letter-spacing: -0.2px;
  line-height: 1.25;
  color: var(--color-text);
}

/* Скроллится только тело — шапка и кнопки всегда на экране. */
.tour-body {
  min-height: 0;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.tour-text {
  margin: 0;
  font-size: 14.5px;
  line-height: 1.55;
  color: var(--color-text);
}

.tour-tip {
  margin: 4px 0 0;
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 13px;
  line-height: 1.45;
  color: var(--color-on-secondary-container);
  background: var(--color-secondary-container);
  padding: 10px 12px;
  border-radius: var(--radius-md);
}
.tour-tip .material-symbols-outlined { font-size: 18px; flex-shrink: 0; }

.tour-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  flex-shrink: 0;
}
.tour-actions .btn-text { margin-right: auto; }

.btn-text, .btn-filled {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  cursor: pointer;
  padding: 8px 16px;
  border-radius: var(--radius-full);
  font-size: 14px;
  font-weight: 600;
  font-family: inherit;
  transition: background 0.15s;
}

.btn-text {
  background: transparent;
  color: var(--color-text);
}
.btn-text:hover { background: var(--color-surface-low); }

.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.btn-filled:hover { background: var(--color-primary-hover); }

.btn-text .material-symbols-outlined,
.btn-filled .material-symbols-outlined { font-size: 18px; }

/* Адаптив: на узких экранах карточка снизу, ширина — на весь viewport
   с боковыми отступами. */
@media (max-width: 720px) {
  .tour-card {
    padding: 18px 18px 16px;
    border-radius: var(--radius-xl) var(--radius-xl) var(--radius-lg) var(--radius-lg);
  }
  .tour-title { font-size: 17px; }
}

/* Транзишн появления тура. */
.tour-fade-enter-active, .tour-fade-leave-active { transition: opacity 0.25s; }
.tour-fade-enter-from, .tour-fade-leave-to { opacity: 0; }
</style>
