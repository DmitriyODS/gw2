<template>
  <Teleport to="body">
    <div
      class="tut-backdrop"
      :class="{ 'tut-backdrop--dim': !hasTarget && !step.transparent }"
      :style="step.transparent ? { pointerEvents: 'none' } : {}"
    />

    <div
      v-if="hasTarget && spotRect && !step.transparent"
      class="tut-spotlight"
      :style="spotStyle"
    />

    <Transition name="tut-fade" mode="out-in">
      <div :key="step.id" class="tut-card" :style="cardStyle">

        <div class="tut-header" :style="{ background: step.bgColor }">
          <div class="tut-icon-wrap" :style="{ background: step.iconBg }">
            <span class="material-symbols-outlined tut-icon" :style="{ color: step.iconColor }">
              {{ step.icon }}
            </span>
          </div>
          <div class="tut-header-right">
            <span class="tut-counter">{{ stepIndex + 1 }} / {{ activeSteps.length }}</span>
            <button class="tut-skip" @click="closeTutorial">Пропустить</button>
          </div>
        </div>

        <div class="tut-progress-track">
          <div
            class="tut-progress-fill"
            :style="{
              width: `${((stepIndex + 1) / activeSteps.length) * 100}%`,
              background: step.iconColor,
            }"
          />
        </div>

        <div class="tut-body">
          <h3 class="tut-title">{{ step.title }}</h3>
          <p class="tut-text">{{ step.text }}</p>
          <div
            v-if="step.tip"
            class="tut-tip"
            :style="{ borderColor: step.iconColor, background: step.bgColor }"
          >
            <span class="material-symbols-outlined tut-tip-icon" :style="{ color: step.iconColor }">
              lightbulb
            </span>
            <span>{{ step.tip }}</span>
          </div>
        </div>

        <div class="tut-footer">
          <button
            v-if="stepIndex > 0"
            class="tut-btn tut-btn--secondary"
            :disabled="transitioning"
            @click="prev"
          >
            <span class="material-symbols-outlined">arrow_back</span>
            Назад
          </button>
          <span v-else />

          <div class="tut-dots">
            <span
              v-for="(_, i) in activeSteps"
              :key="i"
              class="tut-dot"
              :class="{ 'tut-dot--active': i === stepIndex }"
              :style="i === stepIndex ? { background: step.iconColor } : {}"
            />
          </div>

          <button
            class="tut-btn tut-btn--primary"
            :style="{ background: step.iconColor }"
            :disabled="transitioning"
            @click="next"
          >
            {{ isLast ? 'Начать работу' : 'Далее' }}
            <span class="material-symbols-outlined">
              {{ isLast ? 'rocket_launch' : 'arrow_forward' }}
            </span>
          </button>
        </div>

      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { ref, computed, watch, onMounted, onUnmounted, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { useTutorial } from '@/composables/useTutorial.js'
import { useTasksStore } from '@/stores/tasks.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { createTask, deleteTask } from '@/api/tasks.js'
import { createUnit, stopUnit } from '@/api/units.js'
import { getDepartments } from '@/api/departments.js'
import { getUnitTypes } from '@/api/unitTypes.js'
import { useUnitsStore } from '@/stores/units.js'

const { close, startAtId } = useTutorial()
const router = useRouter()
const { isAtLeast } = usePermission()
const tasksStore = useTasksStore()
const unitsStore = useUnitsStore()

const demoTaskId = ref(null)
const demoUnitId = ref(null)

async function cleanupDemoUnit() {
  if (!demoUnitId.value) return
  const id = demoUnitId.value
  demoUnitId.value = null
  try { await stopUnit(id) } catch {}
  unitsStore.clearActiveUnit()
}

async function cleanupDemoTask() {
  if (!demoTaskId.value) return
  const id = demoTaskId.value
  demoTaskId.value = null
  tasksStore.closeTask()
  try { await deleteTask(id) } catch {}
  tasksStore.removeTask(id)
}

async function cleanupAll() {
  await cleanupDemoUnit()
  await cleanupDemoTask()
}

async function closeTutorial() {
  await cleanupAll()
  close()
}

// ─── Цветовые схемы ────────────────────────────────────────────────────────
const C = {
  primary:   { bgColor: 'color-mix(in oklch, var(--color-primary) 12%, var(--color-surface))',   iconBg: 'var(--color-primary-container)',   iconColor: 'var(--color-primary)' },
  secondary: { bgColor: 'color-mix(in oklch, var(--color-secondary) 12%, var(--color-surface))', iconBg: 'var(--color-secondary-container)', iconColor: 'var(--color-secondary)' },
  tertiary:  { bgColor: 'color-mix(in oklch, var(--color-tertiary) 12%, var(--color-surface))',  iconBg: 'var(--color-tertiary-container)',  iconColor: 'var(--color-tertiary)' },
  success:   { bgColor: 'color-mix(in oklch, var(--color-success) 12%, var(--color-surface))',   iconBg: 'var(--color-success-container)',   iconColor: 'var(--color-success)' },
  warning:   { bgColor: 'color-mix(in oklch, var(--color-warning) 12%, var(--color-surface))',   iconBg: 'var(--color-warning-container)',   iconColor: 'var(--color-warning)' },
}

// ─── Все шаги (фильтруются по правам при открытии) ────────────────────────
const ALL_STEPS = [
  {
    id: 'welcome',
    icon: 'waving_hand',
    ...C.primary,
    title: 'Добро пожаловать в Groove Work!',
    text: 'Groove Work — платформа для командного трекинга задач и рабочего времени. Это обучение проведёт вас по всем разделам — мы будем переходить между ними прямо во время руководства.',
    tip: null,
    target: null,
  },
  {
    id: 'tasks-board',
    icon: 'grid_view',
    ...C.primary,
    title: 'Доска задач',
    text: 'Это главный экран — здесь собраны все задачи вашей команды. Ищите задачи по названию, фильтруйте по исполнителям и переключайтесь между разделами с помощью вкладок справа.',
    tip: null,
    target: '[data-tutorial="nav-tasks"]',
    navigateTo: '/tasks',
  },
  {
    id: 'tab-active',
    icon: 'checklist',
    ...C.primary,
    title: 'Вкладка «Активные»',
    text: 'Здесь отображаются все текущие задачи команды — не закрытые и не перемещённые в архив. Это вкладка по умолчанию, вы всегда начинаете работу отсюда.',
    tip: null,
    target: '[data-tutorial="tab-active"]',
    onEnter: () => tasksStore.setTab('active'),
  },
  {
    id: 'tab-favorites',
    icon: 'star',
    ...C.warning,
    title: 'Вкладка «Избранное»',
    text: 'Задачи, отмеченные звёздочкой, попадают сюда. Удобно держать здесь высокоприоритетные задачи или те, к которым часто возвращаетесь.',
    tip: 'Нажмите на звёздочку прямо на карточке задачи, чтобы добавить её в избранное — открывать задачу не нужно.',
    target: '[data-tutorial="tab-favorites"]',
    onEnter: () => tasksStore.setTab('favorites'),
  },
  {
    id: 'tab-archive',
    icon: 'inventory_2',
    ...C.secondary,
    title: 'Вкладка «Архив»',
    text: 'Завершённые задачи архивируются и хранятся здесь. Создавать новые юниты для них нельзя, но вся история работы сохраняется.',
    tip: 'Задачу с активным (запущенным) юнитом заархивировать нельзя — сначала нужно завершить юнит.',
    target: '[data-tutorial="tab-archive"]',
    onEnter: () => tasksStore.setTab('archive'),
  },
  {
    id: 'task-create',
    icon: 'add_task',
    ...C.tertiary,
    title: 'Создаём тестовую задачу',
    text: 'Мы только что создали тестовую задачу — она уже появилась в списке. На следующем шаге откроем её и посмотрим, что внутри. После обучения задача будет удалена автоматически.',
    tip: 'В реальной работе вы нажимаете «Добавить» в правом верхнем углу доски и заполняете форму самостоятельно.',
    target: '[data-tutorial="task-add-btn"]',
    check: (isAtLeast) => isAtLeast(ROLES.EMPLOYEE),
    onEnter: async () => {
      tasksStore.setTab('active')
      try {
        const depts = await getDepartments()
        const deptId = depts?.[0]?.id
        if (!deptId) return
        const task = await createTask({ name: 'Демо-задача (обучение)', department_id: deptId })
        demoTaskId.value = task.id
        tasksStore.upsertTask(task)
      } catch {}
    },
  },
  {
    id: 'task-card',
    icon: 'task_alt',
    ...C.tertiary,
    title: 'Что внутри задачи',
    text: 'Вот как выглядит задача изнутри: название, описание, статус и исполнитель. Над одной задачей могут работать сразу несколько сотрудников — у каждого своя история работы.',
    tip: 'Подробное описание помогает всем участникам сразу понять, что нужно сделать.',
    target: null,
    transparent: true,
    onEnter: async () => {
      await nextTick()
      const task = demoTaskId.value
        ? tasksStore.tasks.find(t => t.id === demoTaskId.value)
        : tasksStore.tasks[0]
      if (task) tasksStore.openTask(task)
    },
  },
  {
    id: 'units-concept',
    icon: 'timer',
    ...C.warning,
    title: 'Юниты — отрезки вашей работы',
    text: 'Юнит — это один логический кусочек задачи, который выполняет один сотрудник. Откройте задачу и нажмите «Начать»: введите название и выберите тип. Примеры для медиа-команды: «Дизайн макета», «Написание текста», «Монтаж», «Публикация», «Копирайтинг», «Корректура».',
    tip: 'Каждый сотрудник может создать любое количество юнитов для одной задачи. Юнит нельзя поставить на паузу — только начать и завершить.',
    target: null,
    check: (isAtLeast) => isAtLeast(ROLES.EMPLOYEE),
    onEnter: () => tasksStore.closeTask(),
  },
  {
    id: 'unit-create',
    icon: 'play_circle',
    ...C.success,
    title: 'Запускаем демо-юнит',
    text: 'Мы только что запустили юнит для тестовой задачи. Посмотрите на экран — по центру появился таймер. Именно так выглядит рабочий сеанс: название, тип работы и живой отсчёт времени.',
    tip: 'В реальной работе вы открываете задачу, нажимаете «Начать», вводите название юнита и выбираете его тип — например, «Дизайн», «Копирайтинг» или «Монтаж».',
    target: null,
    transparent: true,
    check: (isAtLeast) => isAtLeast(ROLES.EMPLOYEE),
    onEnter: async () => {
      if (!demoTaskId.value) return
      try {
        const types = await getUnitTypes()
        const typeId = types?.[0]?.id
        if (!typeId) return
        const unit = await createUnit(demoTaskId.value, { name: 'Демо-юнит (обучение)', unit_type_id: typeId })
        demoUnitId.value = unit.id
        unitsStore.setActiveUnit({ ...unit, task_name: 'Демо-задача (обучение)' })
      } catch {}
    },
  },
  {
    id: 'active-unit',
    icon: 'radio_button_checked',
    ...C.success,
    title: 'Активный юнит — вы сейчас работаете',
    text: 'Пока таймер идёт — идёт учёт рабочего времени. Все разделы системы заблокированы: вы сфокусированы на задаче. Нажмите «Завершить» только когда этот кусочек работы действительно выполнен.',
    tip: 'Одновременно может быть только один активный юнит. Завершите текущий, прежде чем начинать следующий.',
    target: null,
    transparent: true,
    check: (isAtLeast) => isAtLeast(ROLES.EMPLOYEE),
  },
  {
    id: 'employees-nav',
    icon: 'group',
    ...C.primary,
    title: 'Сотрудники — все коллеги в одном месте',
    text: 'Раздел «Сотрудники» — доска со всеми коллегами компании. Видно, кто сейчас в сети (зелёная точка на аватаре) и когда последний раз заходил тот, кого нет. Удобно, когда нужно быстро понять, кому написать или позвонить.',
    tip: 'Карточка коллеги открывает профиль, оттуда — кнопки «Написать» и «Позвонить».',
    target: '[data-tutorial="nav-employees"]',
    navigateTo: '/employees',
    onEnter: async () => { await cleanupAll() },
  },
  {
    id: 'messenger-nav',
    icon: 'chat',
    ...C.secondary,
    title: 'Мессенджер — переписка прямо в платформе',
    text: 'Встроенный чат: текст, картинки, видео, документы. Можно отвечать на конкретное сообщение, пересылать, удалять у себя или у обоих. Маленький мини-чат в углу экрана работает даже поверх запущенного юнита.',
    tip: 'Файл можно бросить перетаскиванием в любое место чата или вставить из буфера (Ctrl+V для скриншотов).',
    target: '[data-tutorial="nav-messenger"]',
    navigateTo: '/messenger',
  },
  {
    id: 'calls-info',
    icon: 'videocam',
    ...C.tertiary,
    title: 'Звонки и видеоконференции',
    text: 'Голосом или с видео, один на один или вшестером — прямо в платформе, без Zoom и Telegram. В шапке любого чата и в карточке коллеги есть две кнопки: трубка (аудио) и камера (видео). У собеседника всплывёт входящий звонок с вашим именем и аватаром.',
    tip: 'Звонок можно свернуть в маленькое окошко в углу и продолжать листать задачи — собеседник останется на связи. Микрофон и камеру можно выключать в любой момент.',
    target: null,
  },
  {
    id: 'stats-nav',
    icon: 'query_stats',
    ...C.secondary,
    title: 'Переходим в статистику',
    text: 'Раздел «Статистика» показывает, сколько времени потрачено на задачи — в разрезе по сотрудникам, типам юнитов и периодам. Есть два режима просмотра.',
    tip: null,
    target: '[data-tutorial="nav-stats"]',
    navigateTo: '/stats',
    check: (isAtLeast) => isAtLeast(ROLES.EMPLOYEE),
    onEnter: async () => { await cleanupAll() },
  },
  {
    id: 'stats-common',
    icon: 'bar_chart',
    ...C.secondary,
    title: 'Общая статистика',
    text: 'Сводка за период: сколько задач поступило, закрыто и осталось. Таблица отработки по сотрудникам и рейтинг задач по затраченным часам. При наличии прав — доступен экспорт.',
    tip: null,
    target: '[data-tutorial="stats-tab-common"]',
    onEnter: () => document.querySelector('[data-tutorial="stats-tab-common"]')?.click(),
    check: (isAtLeast) => isAtLeast(ROLES.EMPLOYEE),
  },
  {
    id: 'stats-extended',
    icon: 'analytics',
    ...C.secondary,
    title: 'Расширенная статистика',
    text: 'Детальная аналитика: разбивка по типам юнитов, по отделам, тепловая карта загруженности сотрудников по дням. Полезно для анализа работы команды по направлениям.',
    tip: null,
    target: '[data-tutorial="stats-tab-extended"]',
    onEnter: () => document.querySelector('[data-tutorial="stats-tab-extended"]')?.click(),
    check: (isAtLeast) => isAtLeast(ROLES.EMPLOYEE),
  },
  {
    id: 'settings-nav',
    icon: 'settings',
    ...C.tertiary,
    title: 'Переходим в настройки',
    text: 'Раздел «Настройки» содержит управление пользователями, ролями, правами доступа, отделами, типами юнитов, резервными копиями и темой интерфейса.',
    tip: null,
    target: '[data-tutorial="nav-settings"]',
    navigateTo: '/settings',
    onEnter: async () => { await cleanupAll() },
  },
  {
    id: 'settings-theme',
    icon: 'palette',
    ...C.tertiary,
    title: 'Внешний вид и темы',
    text: 'В разделе «Внешний вид» можно выбрать готовую тему или собрать свою из четырёх ключевых цветов. Палитра пересчитывается мгновенно — попробуйте кнопку «Мне повезёт».',
    tip: 'Светлая или тёмная тема переключается отдельным сегментом — внешний вид меняется одним кликом.',
    target: '[data-tutorial="settings-section-theme"]',
    onEnter: () => document.querySelector('[data-tutorial="settings-section-theme"]')?.click(),
  },
  {
    id: 'settings-help',
    icon: 'help_center',
    ...C.secondary,
    title: 'Справка по всем разделам',
    text: 'В справке собраны подробные описания всех разделов платформы с примерами и подсказками. Если что-то непонятно — заходите туда, можно искать по словам.',
    tip: 'Из карточки раздела в справке можно сразу перейти к этому шагу в туре — кнопка «Показать в туре».',
    target: '[data-tutorial="settings-section-help"]',
  },
  {
    id: 'profile',
    icon: 'account_circle',
    ...C.primary,
    title: 'Ваш профиль',
    text: 'Нажмите на аватар в нижней части панели, чтобы перейти в профиль. Здесь можно сменить пароль и загрузить фотографию — она отображается рядом с вашим именем по всему интерфейсу.',
    tip: 'Нет фото — система создаёт уникальный цветной identicon по вашему ID.',
    target: '[data-tutorial="profile-avatar"]',
    navigateTo: '/profile',
  },
  {
    id: 'changelog',
    icon: 'newsmode',
    ...C.primary,
    title: 'История обновлений',
    text: 'Нажмите на логотип Groove Work в верхней части боковой панели — откроется список всех версий платформы с описанием изменений в каждом обновлении.',
    tip: null,
    target: '[data-tutorial="logo"]',
  },
  {
    id: 'done',
    icon: 'celebration',
    ...C.success,
    title: 'Всё готово — удачной работы!',
    text: 'Теперь вы знаете всё необходимое для работы в Groove Work. Создавайте задачи, запускайте юниты, анализируйте результаты вместе с командой. Повторить обучение — Настройки → Персонализация.',
    tip: null,
    target: null,
  },
]

// ─── Фильтрация по правам ──────────────────────────────────────────────────
const activeSteps = computed(() => ALL_STEPS.filter(s => !s.check || s.check(isAtLeast)))

// ─── Состояние ────────────────────────────────────────────────────────────
const CARD_W        = 460
const CARD_APPROX_H = 440
const SPOT_PAD      = 10
const CARD_GAP      = 24

const stepIndex     = ref(0)
const spotRect      = ref(null)
const transitioning = ref(false)
const windowWidth   = ref(window.innerWidth)

const step      = computed(() => activeSteps.value[stepIndex.value] ?? activeSteps.value[0])
const hasTarget = computed(() => !!step.value?.target)
const isLast    = computed(() => stepIndex.value === activeSteps.value.length - 1)
const isMobile  = computed(() => windowWidth.value <= 768)

// ─── Стили ────────────────────────────────────────────────────────────────
const spotStyle = computed(() => {
  if (!spotRect.value) return {}
  return {
    top:       `${spotRect.value.top  - SPOT_PAD}px`,
    left:      `${spotRect.value.left - SPOT_PAD}px`,
    width:     `${spotRect.value.width  + SPOT_PAD * 2}px`,
    height:    `${spotRect.value.height + SPOT_PAD * 2}px`,
    boxShadow: `0 0 0 3px ${step.value.iconColor}, 0 0 0 100vmax var(--color-scrim)`,
  }
})

const cardStyle = computed(() => {
  if (isMobile.value) return {
    bottom: 'calc(60px + env(safe-area-inset-bottom, 0px))',
    left: '0',
    right: '0',
    width: '100%',
    borderRadius: '20px 20px 0 0',
    maxHeight: '80dvh',
  }

  if (step.value?.transparent) {
    return { bottom: '24px', right: '24px', width: `${CARD_W}px` }
  }
  if (!hasTarget.value || !spotRect.value) {
    return { top: '50%', left: '50%', transform: 'translate(-50%, -50%)', width: `${CARD_W}px` }
  }
  const rect = spotRect.value
  const spotRight   = rect.left + rect.width + SPOT_PAD
  const spotLeft    = rect.left - SPOT_PAD
  const spotCenterY = rect.top  + rect.height / 2

  let cardTop = spotCenterY - CARD_APPROX_H / 2
  cardTop = Math.max(16, Math.min(window.innerHeight - CARD_APPROX_H - 16, cardTop))

  const fitsRight = spotRight + CARD_GAP + CARD_W <= window.innerWidth - 8
  const fitsLeft  = spotLeft  - CARD_GAP - CARD_W >= 8

  let cardLeft
  if (fitsRight) {
    cardLeft = spotRight + CARD_GAP
  } else if (fitsLeft) {
    cardLeft = spotLeft - CARD_GAP - CARD_W
  } else {
    cardLeft = Math.max(8, (window.innerWidth - CARD_W) / 2)
  }

  return { top: `${cardTop}px`, left: `${cardLeft}px`, width: `${CARD_W}px` }
})

// ─── Логика шагов ─────────────────────────────────────────────────────────
async function updateSpotRect() {
  await nextTick()
  const target = step.value?.target
  if (!target) { spotRect.value = null; return }
  const el = document.querySelector(target)
  spotRect.value = el ? el.getBoundingClientRect() : null
}

async function applyStep(idx) {
  if (transitioning.value) return
  transitioning.value = true
  try {
    const s = activeSteps.value[idx]
    if (!s) return

    if (s.navigateTo && router.currentRoute.value.path !== s.navigateTo) {
      await router.push(s.navigateTo)
      await new Promise(r => setTimeout(r, 250))
    }

    if (s.onEnter) {
      const r = s.onEnter()
      if (r instanceof Promise) await r
      await nextTick()
      await new Promise(r => setTimeout(r, 80))
    }

    await updateSpotRect()
  } finally {
    transitioning.value = false
  }
}

watch(stepIndex, applyStep)

function next() {
  if (transitioning.value) return
  if (isLast.value) { closeTutorial(); return }
  stepIndex.value++
}

function prev() {
  if (transitioning.value) return
  if (stepIndex.value > 0) stepIndex.value--
}

function onKeydown(e) {
  if (e.key === 'Escape')     close()
  if (e.key === 'ArrowRight') next()
  if (e.key === 'ArrowLeft')  prev()
}

function onResize() {
  windowWidth.value = window.innerWidth
  updateSpotRect()
}

onMounted(async () => {
  // Если тур открыт «прыжком» из справки на конкретный шаг — стартуем оттуда.
  if (startAtId.value) {
    const idx = activeSteps.value.findIndex(s => s.id === startAtId.value)
    if (idx >= 0) stepIndex.value = idx
  }
  await applyStep(stepIndex.value)
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('resize',  onResize)
})
onUnmounted(() => {
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('resize',  onResize)
  cleanupAll()
})
</script>

<style scoped>
.tut-backdrop {
  position: fixed;
  inset: 0;
  z-index: 10000;
  pointer-events: all;
  background: transparent;
}

.tut-backdrop--dim {
  background: var(--color-scrim);
  backdrop-filter: blur(3px);
}

.tut-spotlight {
  position: fixed;
  z-index: 10001;
  border-radius: 16px;
  background: transparent;
  pointer-events: none;
  transition:
    top    0.4s cubic-bezier(.4, 0, .2, 1),
    left   0.4s cubic-bezier(.4, 0, .2, 1),
    width  0.4s cubic-bezier(.4, 0, .2, 1),
    height 0.4s cubic-bezier(.4, 0, .2, 1);
}

/* ── Карточка ───────────────────────────────────── */
.tut-card {
  position: fixed;
  z-index: 10002;
  pointer-events: all;
  background: var(--color-surface);
  border-radius: 20px;
  box-shadow: var(--shadow-xl), var(--shadow-lg);
  overflow: hidden;
  display: flex;
  flex-direction: column;
  max-height: calc(100dvh - 32px);
  overflow-y: auto;
  overscroll-behavior: contain;
}

/* ── Шапка ──────────────────────────────────────── */
.tut-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 20px 20px 16px;
  flex-shrink: 0;
}

.tut-icon-wrap {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.tut-icon {
  font-size: 30px;
}

.tut-header-right {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 6px;
  flex: 1;
}

.tut-counter {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text-dim);
  letter-spacing: 0.3px;
}

.tut-skip {
  font-size: 12px;
  color: var(--color-text-dim);
  background: none;
  border: none;
  cursor: pointer;
  padding: 3px 8px;
  border-radius: 6px;
  transition: color 0.15s, background 0.15s;
}

.tut-skip:hover {
  color: var(--color-text);
  background: var(--color-surface-high);
}

/* ── Прогресс-бар ───────────────────────────────── */
.tut-progress-track {
  height: 3px;
  background: var(--color-surface-highest);
  flex-shrink: 0;
}

.tut-progress-fill {
  height: 100%;
  transition: width 0.4s cubic-bezier(.4, 0, .2, 1);
  border-radius: 0 2px 2px 0;
}

/* ── Контент ────────────────────────────────────── */
.tut-body {
  padding: 20px 24px 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex: 1;
}

.tut-title {
  font-size: 20px;
  font-weight: 800;
  color: var(--color-text);
  line-height: 1.25;
  margin: 0;
}

.tut-text {
  font-size: 14px;
  line-height: 1.7;
  color: var(--color-text-dim);
  margin: 0;
}

.tut-tip {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 10px;
  border-left: 3px solid;
  font-size: 13px;
  line-height: 1.55;
  color: var(--color-text);
}

.tut-tip-icon {
  font-size: 18px;
  flex-shrink: 0;
  margin-top: 1px;
}

/* ── Подвал ─────────────────────────────────────── */
.tut-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 20px 20px;
  border-top: 1px solid var(--color-outline-dim);
  gap: 12px;
  flex-shrink: 0;
}

.tut-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  border-radius: 12px;
  padding: 10px 18px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s, transform 0.1s;
  white-space: nowrap;
}

.tut-btn:disabled {
  opacity: 0.45;
  cursor: not-allowed;
  transform: none !important;
}

.tut-btn:not(:disabled):hover {
  opacity: 0.88;
  transform: translateY(-1px);
}

.tut-btn:not(:disabled):active {
  transform: translateY(0);
}

.tut-btn .material-symbols-outlined {
  font-size: 18px;
}

.tut-btn--primary {
  color: var(--color-on-primary);
}

.tut-btn--secondary {
  background: var(--color-surface-high);
  color: var(--color-text);
}

/* ── Точки прогресса ────────────────────────────── */
.tut-dots {
  display: flex;
  align-items: center;
  gap: 5px;
  flex-wrap: wrap;
  justify-content: center;
  max-width: 140px;
}

.tut-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--color-outline-dim);
  transition: background 0.25s, width 0.25s, border-radius 0.25s;
}

.tut-dot--active {
  width: 16px;
  border-radius: 4px;
}

/* ── Переход между шагами ───────────────────────── */
.tut-fade-enter-active,
.tut-fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.tut-fade-enter-from {
  opacity: 0;
  transform: translateY(8px) scale(0.98);
}

.tut-fade-leave-to {
  opacity: 0;
  transform: translateY(-8px) scale(0.98);
}

/* ── Мобильная адаптация ────────────────────────── */
@media (max-width: 768px) {
  .tut-header {
    padding: 16px 16px 12px;
  }

  .tut-icon-wrap {
    width: 44px;
    height: 44px;
    border-radius: 12px;
  }

  .tut-icon {
    font-size: 24px;
  }

  .tut-body {
    padding: 16px 16px 12px;
    gap: 10px;
  }

  .tut-title {
    font-size: 18px;
  }

  .tut-footer {
    padding: 12px 16px 14px;
    gap: 8px;
  }

  .tut-dots {
    display: none;
  }

  .tut-btn {
    padding: 10px 16px;
  }

  .tut-fade-enter-from {
    opacity: 0;
    transform: translateY(24px) scale(0.99);
  }

  .tut-fade-leave-to {
    opacity: 0;
    transform: translateY(12px) scale(0.99);
  }
}
</style>
