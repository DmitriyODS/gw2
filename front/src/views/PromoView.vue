<template>
  <div class="pr">
    <AuthBackdrop />

    <!-- ── Шапка: плавающая панель того же материала, что панель задач ── -->
    <header class="pr-top">
      <RouterLink to="/" class="pr-brand" aria-label="Groove Work">
        <BrandLogo :size="28" />
        <BrandWordmark :size="17" />
      </RouterLink>
      <nav class="pr-nav">
        <AppButton tag="a" href="#how" variant="text" size="sm" label="Как это работает" />
        <AppButton tag="a" href="#tools" variant="text" size="sm" label="Инструменты" />
        <AppButton tag="a" href="#devices" variant="text" size="sm" label="Устройства" />
        <AppButton tag="a" href="#faq" variant="text" size="sm" label="Вопросы" />
      </nav>
      <div class="pr-top-actions">
        <AppButton tag="router-link" to="/login" variant="text" label="Войти" class="pr-top-login" />
        <AppButton tag="router-link" to="/welcome" variant="filled" label="Начать" />
      </div>
    </header>

    <main class="pr-main">
      <!-- ── Герой ──────────────────────────────────────────────── -->
      <section class="pr-hero">
        <div class="pr-hero-text">
          <h1 class="pr-display reveal">
            Ваш timeline,<br>
            <span class="pr-accent">обёрнутый в удобный интерфейс</span>
          </h1>
          <p class="pr-lead reveal">
            Положите дело во время — и оно вернётся к вам вовремя. Заметки, файлы,
            календари, переписка и звонки собираются вокруг одной оси дня:
            что сейчас, что дальше и что потом.
          </p>
          <div class="pr-actions reveal">
            <AppButton tag="router-link" to="/welcome" variant="filled" size="lg" trailing-icon="arrow_forward" label="Начать бесплатно" />
            <AppButton tag="router-link" to="/login" size="lg" label="У меня есть аккаунт" />
          </div>
        </div>

        <!-- Витрина: экран «Сегодня», собранный из тех же компонентов, что и
             само приложение, — поэтому он всегда выглядит как настоящий. -->
        <div class="pr-today reveal" aria-hidden="true" inert>
          <AppCard class="pr-today-card" :gap="14">
            <div class="pr-today-head">
              <span class="pr-today-title">Сегодня</span>
              <span class="pr-today-date">{{ todayLabel }}</span>
            </div>

            <div class="pr-now">
              <span class="pr-label">Сейчас · 11:00–12:30</span>
              <strong class="pr-now-title">Подготовить презентацию для клиента</strong>
              <AppStack row :gap="6">
                <AppChip size="sm" icon="description">Тезисы</AppChip>
                <AppChip size="sm" icon="draw">Доска макета</AppChip>
                <AppChip size="sm" icon="attach_file">brief.pdf</AppChip>
              </AppStack>
              <div class="pr-progress"><span /></div>
            </div>

            <div class="pr-next">
              <span class="pr-label">Дальше</span>
              <AppRow v-for="n in NEXT" :key="n.title" plain dense :title="n.title" :hint="n.hint">
                <template #lead><span class="pr-time">{{ n.time }}</span></template>
              </AppRow>
            </div>

            <div class="pr-later">
              <span class="pr-label">Потом</span>
              <span class="pr-later-text">Купить подарок · Позвонить в сервис · Прочитать статью</span>
            </div>

            <div class="pr-today-foot">
              <AppChip size="sm" tone="success" icon="check_circle">Сделано: 4</AppChip>
              <div class="pr-hola">
                <span class="material-symbols-outlined">blur_on</span>
                Созвон с Аней завтра в 15:00
              </div>
            </div>
          </AppCard>
        </div>
      </section>

      <!-- ── Как это работает ───────────────────────────────────── -->
      <section id="how" class="pr-section">
        <header class="pr-section-head reveal">
          <h2 class="pr-h2">Положите во время — и оно вернётся вовремя</h2>
          <p class="pr-section-lead">
            Не учёт времени и не табель. Timeline — просто место, где у каждого дела
            есть своё «когда», а у вас — спокойная голова.
          </p>
        </header>
        <AppGrid :min="260" :gap="16">
          <AppCard v-for="(s, i) in STEPS" :key="s.title" class="reveal">
            <span class="pr-step-num">{{ i + 1 }}</span>
            <h3 class="pr-h3">{{ s.title }}</h3>
            <p class="pr-text">{{ s.text }}</p>
          </AppCard>
        </AppGrid>
      </section>

      <!-- ── Инструменты ────────────────────────────────────────── -->
      <section id="tools" class="pr-section">
        <header class="pr-section-head reveal">
          <h2 class="pr-h2">Разделы — инструменты вашего timeline</h2>
          <p class="pr-section-lead">
            Любую вещь любого раздела можно положить «во время…»: заметку, доску,
            файл, запись календаря. На оси дня окажется ссылка, а не копия.
          </p>
        </header>
        <AppGrid :min="240" :gap="16">
          <AppCard v-for="f in TOOLS" :key="f.title" class="reveal">
            <span class="material-symbols-outlined pr-tool-icon">{{ f.icon }}</span>
            <h3 class="pr-h3">{{ f.title }}</h3>
            <p class="pr-text">{{ f.text }}</p>
          </AppCard>
        </AppGrid>
      </section>

      <!-- ── Моё и команды ─────────────────────────────────────── -->
      <section class="pr-split">
        <div class="pr-split-text reveal">
          <AppChip icon="group">Пространства</AppChip>
          <h2 class="pr-h2">Своё — отдельно, общее — с командой</h2>
          <p class="pr-text">
            У каждой вещи одно место: «Моё» или команда. Личное не прячется, когда
            вы переключаетесь на работу, а общее остаётся у команды, даже если кто-то
            ушёл. Перенести вещь к себе или в команду — одно действие.
          </p>
        </div>
        <AppCard class="pr-split-visual reveal" :gap="8" aria-hidden="true" inert>
          <AppRow title="Моё" hint="Заметки, файлы, дела дня" icon="person" selected dense />
          <AppRow title="Дизайн" hint="Команда · 6 человек" icon="group" dense />
          <AppRow title="Продажи" hint="Команда · 12 человек" icon="group" dense />
        </AppCard>
      </section>

      <section class="pr-split pr-split--rev">
        <div class="pr-split-text reveal">
          <AppChip icon="desktop_windows">Рабочий стол</AppChip>
          <h2 class="pr-h2">Окна на компьютере, один экран в телефоне</h2>
          <p class="pr-text">
            На большом экране разделы открываются окнами рядом друг с другом, а
            «Пуск» показывает живые плитки. В телефоне тот же timeline помещается
            в один экран — без потери возможностей.
          </p>
        </div>
        <div class="pr-desk reveal" aria-hidden="true">
          <span class="pr-desk-win a"><i /><i /><i /></span>
          <span class="pr-desk-win b"><i /><i /><i /></span>
          <span class="pr-desk-win c"><i /><i /><i /></span>
          <span class="pr-desk-bar"><b /><b /><b /><b /><b /></span>
        </div>
      </section>

      <!-- ── Устройства и скачивание ────────────────────────────── -->
      <section id="devices" class="pr-section">
        <header class="pr-section-head reveal">
          <h2 class="pr-h2">Одинаково на всех устройствах</h2>
          <p class="pr-section-lead">Вход один, данные общие: начните за компьютером, продолжите с телефона.</p>
        </header>
        <AppGrid :min="240" :gap="16">
          <AppCard class="pr-device reveal" title="Браузер" hint="Ничего не нужно ставить — откройте адрес и работайте.">
            <template #footer>
              <AppButton tag="router-link" to="/welcome" icon="open_in_new" label="Открыть" />
            </template>
          </AppCard>

          <AppCard class="pr-device reveal" title="Компьютер" hint="Отдельное окно, значок в трее и уведомления, даже когда браузер закрыт.">
            <p class="pr-dl-alt">
              <template v-if="showDesktop">
                <a :href="desktopFileHref('mac')" download>macOS</a> ·
                <a :href="desktopFileHref('win')" download>Windows</a> ·
                <a :href="desktopFileHref('linux')" download>Linux</a>
              </template>
              <template v-else>macOS · Windows · Linux</template>
            </p>
            <template v-if="showDesktop" #footer>
              <AppButton
                tag="a"
                :href="desktopFileHref(desktopOs)"
                download
                variant="filled"
                icon="download"
                :label="`Скачать для ${DESKTOP_OS_LABELS[desktopOs]}`"
              />
            </template>
          </AppCard>

          <AppCard class="pr-device reveal" title="Телефон" hint="Дела, чаты и звонки под рукой — с пуш-уведомлениями.">
            <p class="pr-dl-alt">Android · на iPhone — в браузере</p>
            <template v-if="showApk" #footer>
              <AppButton tag="a" :href="APK_HREF" :download="apkDownloadName" variant="filled" icon="download" label="Скачать APK" />
            </template>
          </AppCard>
        </AppGrid>
      </section>

      <!-- ── Вопросы ────────────────────────────────────────────── -->
      <section id="faq" class="pr-section pr-faq">
        <header class="pr-section-head reveal">
          <h2 class="pr-h2">Частые вопросы</h2>
        </header>
        <AppStack :gap="10">
          <AppCard v-for="q in FAQ" :key="q.q" tag="details" class="pr-q reveal" :gap="0">
            <summary>
              {{ q.q }}
              <span class="material-symbols-outlined">expand_more</span>
            </summary>
            <p class="pr-text">{{ q.a }}</p>
          </AppCard>
        </AppStack>
      </section>

      <!-- ── Финальный призыв ───────────────────────────────────── -->
      <AppCard tag="section" tone="primary" class="pr-final reveal" :gap="16">
        <h2 class="pr-h2">Соберите свой день на одной оси</h2>
        <p class="pr-text">Регистрация займёт минуту. Команду можно позвать позже — ссылкой-приглашением.</p>
        <AppButton tag="router-link" to="/welcome" variant="filled" size="lg" trailing-icon="arrow_forward" label="Начать бесплатно" />
      </AppCard>
    </main>

    <footer class="pr-foot">
      <span>© Groove Work</span>
      <RouterLink to="/welcome">Вход и регистрация</RouterLink>
      <RouterLink v-if="LEGAL_CONSENT_VISIBLE" to="/legal">Правовые документы</RouterLink>
    </footer>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted } from 'vue'
import { LEGAL_CONSENT_VISIBLE } from '@/utils/release.js'
import BrandLogo from '@/components/common/BrandLogo.vue'
import BrandWordmark from '@/components/common/BrandWordmark.vue'
import AuthBackdrop from '@/components/auth/AuthBackdrop.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppGrid from '@/components/ui/AppGrid.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import { useAppDownloads } from '@/composables/useAppDownloads.js'

// Скачивание клиентов — общая обвязка с разделом «О приложении»: имена
// артефактов и номер сборки приезжают из version.json рядом с файлами.
const {
  APK_HREF, apkDownloadName, desktopFileHref, desktopOs, DESKTOP_OS_LABELS,
  showApk, showDesktop,
} = useAppDownloads()

const todayLabel = new Date().toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' })

const NEXT = [
  { time: '13:00', title: 'Обед с командой', hint: 'Кафе на углу' },
  { time: '15:00', title: 'Созвон по запуску', hint: 'Звонок · 4 участника' },
  { time: '18:30', title: 'Тренировка', hint: 'Из расписания' },
]

const STEPS = [
  {
    title: 'Положите во время',
    text: 'Дело, заметку или файл — на конкретный час, на «потом» или просто в сегодня. Hola поймёт фразу «созвон завтра в три».',
  },
  {
    title: 'Приложите нужное',
    text: 'К делу цепляются тезисы, доска, документ с диска и переписка — всё, что понадобится в момент «сейчас».',
  },
  {
    title: 'Получите вовремя',
    text: 'Когда придёт время, дело само окажется наверху оси, а напоминание придёт на компьютер и телефон.',
  },
]

const TOOLS = [
  { icon: 'edit_note', title: 'Заметки и доски', text: 'Текст с совместным редактированием и бесконечный холст с анимацией.' },
  { icon: 'calendar_month', title: 'Календари и расписания', text: 'События, регулярные занятия и ежедневники — всё со своим временем.' },
  { icon: 'folder_open', title: 'Диск', text: 'Файлы и папки с доступом по ссылке, корзиной и избранным.' },
  { icon: 'chat', title: 'Переписка и звонки', text: 'Личные и групповые чаты, видеозвонки с демонстрацией экрана.' },
  { icon: 'dynamic_form', title: 'Формы и реестры', text: 'Опросы, записи на время и настраиваемые таблицы-справочники.' },
  { icon: 'blur_on', title: 'Hola и ИИ', text: 'Поиск по всем разделам, быстрые команды и деловой помощник.' },
]

const FAQ = [
  {
    q: 'Сколько стоит?',
    a: 'Сейчас бесплатно и целиком: все разделы доступны сразу после регистрации. Ограничено только место в хранилище — 5 Гб на человека.',
  },
  {
    q: 'Это учёт рабочего времени?',
    a: 'Нет. Timeline не считает часы и не строит табели — он помогает решить, что делать сейчас, и не забыть о том, что будет потом.',
  },
  {
    q: 'Это для работы или для личных дел?',
    a: 'Прежде всего для вас: ваши дела, заметки и файлы принадлежат вам и никуда не пропадают. Появится команда — общие вещи будут жить у неё, а ваши останутся в «Моём».',
  },
  {
    q: 'А если я работаю в нескольких командах?',
    a: 'Аккаунт принадлежит человеку, а не команде: в каждой у вас своя роль, а личный timeline один на всех.',
  },
  {
    q: 'Кто видит мои дела?',
    a: 'То, что лежит в «Моём», видите только вы и те, с кем вы поделились сами. Общее видят участники команды по своей роли.',
  },
]

// Плавное появление секций при прокрутке (с уважением к reduced-motion).
let observer = null
onMounted(() => {
  const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)')?.matches
  const els = document.querySelectorAll('.pr .reveal')
  if (reduced || !('IntersectionObserver' in window)) {
    els.forEach((el) => el.classList.add('is-visible'))
    return
  }
  observer = new IntersectionObserver((entries) => {
    for (const e of entries) {
      if (e.isIntersecting) {
        e.target.classList.add('is-visible')
        observer.unobserve(e.target)
      }
    }
  }, { threshold: 0.12 })
  els.forEach((el) => observer.observe(el))
})
onBeforeUnmount(() => observer?.disconnect())
</script>

<style scoped>
/* Витрина живёт на ТЕХ ЖЕ токенах и компонентах ядра, что и приложение: до
   входа тема всегда флагманская, а светлая/тёмная берётся у системы (роутер
   помечает маршрут meta.authScreen). Своих кнопок, карточек и палитры здесь
   нет — только раскладка. */
.pr {
  position: relative;
  min-height: 100dvh;
  color: var(--color-text);
  /* clip, а не hidden: hidden делает .pr контейнером прокрутки, и шапка
     прилипала бы к нему, а не к реально прокручиваемому .main-content. */
  overflow-x: clip;
}

.pr-main,
.pr-top,
.pr-foot {
  position: relative;
  z-index: 1;
}

/* ── Шапка ───────────────────────────────────────────────────── */
.pr-top {
  position: sticky;
  top: 12px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 16px;
  width: calc(100% - 32px);
  max-width: 1160px;
  margin: 12px auto 0;
  padding: 8px 8px 8px 16px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-xl);
  background: var(--acrylic-bg-strong);
  box-shadow: var(--sk-panel-shadow);
}

.pr-brand {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  text-decoration: none;
}

.pr-nav {
  flex: 1;
  display: flex;
  justify-content: center;
  gap: 2px;
  min-width: 0;
}

.pr-top-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-left: auto;
}

/* ── Общая сетка секций ──────────────────────────────────────── */
.pr-main {
  display: flex;
  flex-direction: column;
  gap: 104px;
  max-width: 1160px;
  margin: 0 auto;
  padding: 72px 24px 96px;
}

.pr-section { display: flex; flex-direction: column; gap: 32px; scroll-margin-top: 96px; }

.pr-section-head {
  display: flex;
  flex-direction: column;
  gap: 12px;
  max-width: 720px;
}

.pr-h2 {
  margin: 0;
  font-size: clamp(26px, 3.4vw, 40px);
  font-weight: 600;
  line-height: 1.15;
  letter-spacing: -0.02em;
}

.pr-h3 {
  margin: 0;
  font-size: 1.08rem;
  font-weight: 600;
}

.pr-section-lead,
.pr-text {
  margin: 0;
  font-size: 15px;
  line-height: 1.6;
  color: var(--color-text-dim);
}

.pr-section-lead { font-size: 16.5px; }

/* ── Герой ───────────────────────────────────────────────────── */
.pr-hero {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(0, 0.95fr);
  align-items: center;
  gap: 56px;
}

.pr-hero-text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 22px;
}

.pr-display {
  margin: 0;
  font-size: clamp(36px, 5.4vw, 64px);
  font-weight: 700;
  line-height: 1.04;
  letter-spacing: -0.035em;
}

.pr-accent { color: var(--color-primary); }

.pr-lead {
  margin: 0;
  max-width: 560px;
  font-size: clamp(16px, 1.5vw, 18.5px);
  line-height: 1.6;
  color: var(--color-text-dim);
}

.pr-actions { display: flex; flex-wrap: wrap; gap: 10px; }

.pr-hint {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-dim);
}

/* ── Витрина «Сегодня» ───────────────────────────────────────── */
.pr-today { min-width: 0; }

.pr .pr-today-card {
  padding: 22px;
  border-radius: var(--radius-xl);
  box-shadow: var(--sk-panel-shadow), var(--shadow-xl);
}

.pr-today-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
}

.pr-today-title { font-size: 22px; font-weight: 700; letter-spacing: -0.01em; }
.pr-today-date { font-size: 13px; color: var(--color-text-dim); }

.pr-label {
  font-size: 11.5px;
  font-weight: 700;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--color-text-dim);
}

/* «Сейчас» — углубление в листе: главное дело лежит в своей нише. */
.pr-now {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 16px;
  border: 1px solid color-mix(in oklch, var(--color-primary) 35%, var(--sk-edge));
  border-radius: var(--radius-lg);
  background: var(--sk-well-bg);
  box-shadow: var(--sk-well-shadow);
}

.pr-now .pr-label { color: var(--color-primary); }
.pr-now-title { font-size: 18px; font-weight: 600; line-height: 1.3; }

.pr-progress {
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-surface-highest);
  overflow: hidden;
}

.pr-progress span {
  display: block;
  width: 58%;
  height: 100%;
  border-radius: inherit;
  background: var(--grad-primary);
}

.pr-next { display: flex; flex-direction: column; gap: 2px; }

.pr-time {
  min-width: 46px;
  font-size: 13px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: var(--color-primary);
}

.pr-later { display: flex; flex-direction: column; gap: 4px; }

.pr-later-text {
  font-size: 13.5px;
  color: var(--color-text-dim);
  overflow-wrap: anywhere;
}

.pr-today-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 10px;
  padding-top: 12px;
  border-top: 1px solid var(--color-outline-dim);
}

/* Строка Hola — та же «скважина» поля ввода, что и в приложении. */
.pr-hola {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 7px 14px 7px 10px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-full);
  background: var(--sk-well-bg);
  box-shadow: var(--sk-well-shadow);
  font-size: 13px;
  color: var(--color-text-dim);
}

.pr-hola .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }

/* ── Шаги и инструменты ──────────────────────────────────────── */
.pr-step-num {
  display: grid;
  place-items: center;
  width: 36px; min-width: 36px; max-width: 36px;
  height: 36px; min-height: 36px; max-height: 36px;
  border: 1px solid var(--sk-edge);
  border-radius: 50%;
  background: var(--sk-raised-bg);
  box-shadow: var(--sk-raised-shadow);
  font-weight: 700;
  color: var(--color-primary);
}

.pr-tool-icon {
  font-size: 30px;
  color: var(--color-primary);
}

/* ── Чередующиеся разделы ────────────────────────────────────── */
.pr-split {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  align-items: center;
  gap: 56px;
}

.pr-split--rev .pr-split-text { order: 2; }

.pr-split-text {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 16px;
}

.pr .pr-split-visual { padding: 14px; border-radius: var(--radius-xl); }

/* Схема рабочего стола: окна и панель задач — листы того же материала. */
.pr-desk {
  position: relative;
  aspect-ratio: 16 / 10;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-xl);
  background: var(--sk-well-bg);
  box-shadow: var(--sk-well-shadow);
  overflow: hidden;
}

.pr-desk-win {
  position: absolute;
  display: flex;
  gap: 5px;
  padding: 10px 12px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  box-shadow: var(--sk-panel-shadow);
}

.pr-desk-win i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--color-surface-highest);
}

.pr-desk-win i:first-child { background: var(--color-primary); }

.pr-desk-win.a { left: 6%; top: 8%; width: 46%; height: 56%; }
.pr-desk-win.b { left: 40%; top: 18%; width: 52%; height: 50%; }
.pr-desk-win.c { left: 14%; top: 52%; width: 34%; height: 26%; }

.pr-desk-bar {
  position: absolute;
  left: 50%;
  bottom: 5%;
  transform: translateX(-50%);
  display: flex;
  gap: 8px;
  padding: 8px 10px;
  border: 1px solid var(--sk-edge);
  border-radius: var(--radius-md);
  background: var(--acrylic-card-bg);
  box-shadow: var(--sk-panel-shadow);
}

.pr-desk-bar b {
  width: 18px;
  height: 18px;
  border-radius: var(--radius-xs);
  background: var(--sk-raised-bg);
  box-shadow: var(--sk-raised-shadow);
}

.pr-desk-bar b:first-child { background: var(--grad-primary); }

/* ── Устройства ──────────────────────────────────────────────── */
.pr-dl-alt {
  margin: 0;
  font-size: 13px;
  color: var(--color-text-dim);
}

.pr-dl-alt a {
  color: var(--color-primary);
  text-decoration: none;
  font-weight: 600;
}

.pr-dl-alt a:hover { text-decoration: underline; }

/* Подвал карточки прижат к низу: кнопки скачивания стоят в одну линию. */
.pr-device :deep(.card-foot) { margin-top: auto; }

/* ── Вопросы ─────────────────────────────────────────────────── */
.pr-faq { max-width: 820px; width: 100%; margin: 0 auto; }

.pr-q summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  font-size: 16px;
  font-weight: 600;
  cursor: pointer;
  list-style: none;
}

.pr-q summary::-webkit-details-marker { display: none; }

.pr-q summary .material-symbols-outlined {
  color: var(--color-text-dim);
  transition: transform 0.2s ease;
}

.pr-q[open] summary .material-symbols-outlined { transform: rotate(180deg); }
.pr-q[open] .pr-text { margin-top: 12px; }

/* ── Финал ───────────────────────────────────────────────────── */
.pr .pr-final {
  align-items: center;
  padding: 48px 24px;
  border-radius: var(--radius-xl);
  text-align: center;
}

.pr-final .pr-text { max-width: 520px; }

/* ── Подвал ──────────────────────────────────────────────────── */
.pr-foot {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px 22px;
  padding: 0 24px 32px;
  font-size: 13px;
  color: var(--color-text-dim);
}

.pr-foot a { color: inherit; text-decoration: none; }
.pr-foot a:hover { color: var(--color-primary); }

/* ── Появление при прокрутке: только transform и opacity ─────── */
.reveal {
  opacity: 0;
  transform: translateY(18px);
  transition: opacity 0.6s ease, transform 0.6s cubic-bezier(0.2, 0.8, 0.3, 1);
}

.reveal.is-visible { opacity: 1; transform: none; }

@media (prefers-reduced-motion: reduce) {
  .reveal { opacity: 1; transform: none; transition: none; }
}

/* ── Узкие экраны ────────────────────────────────────────────── */
@media (max-width: 960px) {
  .pr-hero,
  .pr-split { grid-template-columns: minmax(0, 1fr); gap: 36px; }
  .pr-split--rev .pr-split-text { order: 0; }
  .pr-nav { display: none; }
}

@media (max-width: 560px) {
  .pr-top { width: calc(100% - 16px); top: 8px; margin-top: 8px; padding-left: 12px; }
  .pr-top-login { display: none; }
  .pr-main { gap: 72px; padding: 40px 16px 64px; }
  .pr-actions { width: 100%; }
  .pr-actions > * { flex: 1 1 100%; }
  .pr .pr-today-card { padding: 16px; }
  .pr .pr-final { padding: 32px 18px; }
}
</style>
