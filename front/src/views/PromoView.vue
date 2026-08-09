<template>
  <div class="pr">
    <!-- Маячок верха страницы: пока он в кадре, шапка прозрачная и «лежит»
         на герое; ушёл за экран — она собирается в стеклянный островок.
         IntersectionObserver, а не scroll-слушатель: страница прокручивается
         внутри .main-content, и window-скролл там не срабатывает. -->
    <span ref="sentinelEl" class="pr-sentinel" aria-hidden="true" />

    <!-- ── Шапка ────────────────────────────────────────────────── -->
    <div class="pr-top-wrap" :class="{ stuck }">
    <header class="pr-top">
      <RouterLink to="/" class="pr-brand" aria-label="Groove Work">
        <Logo :size="26" />
        <span class="pr-wordmark">
          <span class="wm-groove">Groove</span>
          <span class="wm-work">Work</span>
        </span>
      </RouterLink>
      <nav class="pr-nav">
        <a href="#features">Возможности</a>
        <a href="#devices">Устройства</a>
        <a href="#faq">Вопросы</a>
      </nav>
      <RouterLink to="/welcome" class="pr-top-cta">Начать</RouterLink>
    </header>
    </div>

    <!-- ── Герой ────────────────────────────────────────────────── -->
    <section class="pr-hero">
      <p class="pr-eyebrow reveal">ваш личный менеджер дел</p>
      <h1 class="pr-display reveal">Все ваши дела — в одном месте</h1>
      <p class="pr-lead reveal">
        Задачи и время, заметки и файлы, напоминания, переписка и звонки.
        Groove Work ведёт ваш день от первого дела до последнего — и остаётся
        тем же помощником, когда рядом появляется команда.
      </p>
      <div class="pr-actions reveal">
        <RouterLink to="/welcome" class="pr-btn">Попробовать уже сейчас</RouterLink>
        <RouterLink to="/login" class="pr-btn pr-btn--ghost">Войти</RouterLink>
      </div>
      <p class="pr-hint reveal">бесплатно · регистрация за минуту · без карты</p>

      <!-- Витрина продукта: схематичный рабочий стол с окнами разделов -->
      <div class="pr-shot reveal" aria-hidden="true">
        <div class="pr-win pr-win--back">
          <span class="pr-win-bar"><i /><i /><i /></span>
          <span class="pr-line w-60" />
          <span class="pr-line w-40" />
          <span class="pr-line w-75" />
        </div>
        <div class="pr-win pr-win--main">
          <span class="pr-win-bar"><i /><i /><i /></span>
          <div class="pr-win-body">
            <div class="pr-col">
              <span class="pr-chip">в работе</span>
              <span class="pr-card"><span class="pr-line w-70" /><span class="pr-line w-45" /></span>
              <span class="pr-card"><span class="pr-line w-55" /><span class="pr-line w-35" /></span>
            </div>
            <div class="pr-col">
              <span class="pr-chip pr-chip--alt">на проверке</span>
              <span class="pr-card"><span class="pr-line w-60" /><span class="pr-line w-40" /></span>
            </div>
          </div>
        </div>
        <div class="pr-win pr-win--front">
          <span class="pr-dot" />
          <span class="pr-line w-70" />
          <span class="pr-line w-50" />
        </div>
      </div>

      <!-- Фирменная волна — тот же мотив, что на экранах входа: слои шире
           холста и бегут, сдвигаясь ровно на свой период. -->
      <div class="pr-wave" aria-hidden="true">
        <svg viewBox="0 0 1440 120" preserveAspectRatio="none">
          <g class="wv wv-far"><path :d="WAVE_FAR" /></g>
          <g class="wv wv-mid"><path :d="WAVE_MID" /></g>
          <g class="wv wv-near"><path :d="WAVE_NEAR" /></g>
        </svg>
      </div>
    </section>

    <!-- ── Три опоры ────────────────────────────────────────────── -->
    <section class="pr-pillars">
      <article v-for="p in PILLARS" :key="p.title" class="pr-pillar reveal">
        <span class="material-symbols-outlined">{{ p.icon }}</span>
        <h3>{{ p.title }}</h3>
        <p>{{ p.text }}</p>
      </article>
    </section>

    <!-- ── Сетка возможностей ───────────────────────────────────── -->
    <section id="features" class="pr-section">
      <h2 class="pr-h2 reveal">Всё, из чего состоит ваш день</h2>
      <p class="pr-section-lead reveal">
        Разделы знают друг о друге: задача уезжает в чат, заметка становится
        публикацией, файл с диска — вложением, а звонок оставляет след в переписке.
      </p>
      <div class="pr-grid">
        <article v-for="f in FEATURES" :key="f.title" class="pr-card-feature reveal">
          <span class="material-symbols-outlined">{{ f.icon }}</span>
          <h3>{{ f.title }}</h3>
          <p>{{ f.text }}</p>
        </article>
      </div>
    </section>

    <!-- ── Чередующиеся разделы ─────────────────────────────────── -->
    <section class="pr-split reveal">
      <div class="pr-split-text">
        <p class="pr-kicker">рабочий стол</p>
        <h2 class="pr-h2">Окна вместо вкладок</h2>
        <p>
          Разделы открываются окнами: дела рядом с перепиской, календарь поверх
          заметок. Панель задач, меню «Пуск» с живыми плитками и обои — рабочее
          место настраивается один раз и переезжает за вами на любое устройство.
        </p>
      </div>
      <div class="pr-split-visual" aria-hidden="true">
        <div class="pr-desk">
          <span class="pr-desk-win a" />
          <span class="pr-desk-win b" />
          <span class="pr-desk-win c" />
          <span class="pr-desk-bar"><i /><i /><i /><i /></span>
        </div>
      </div>
    </section>

    <section class="pr-split pr-split--rev reveal">
      <div class="pr-split-text">
        <p class="pr-kicker">статистика</p>
        <h2 class="pr-h2">Часы считаются сами</h2>
        <p>
          Счётчик запускается одной кнопкой и останавливается вместе с делом — время
          попадает в отчёт без табелей и напоминаний. Появится команда — та же
          статистика покажет загрузку отделов, а на офисном экране пойдёт табло.
        </p>
      </div>
      <div class="pr-split-visual" aria-hidden="true">
        <div class="pr-chart">
          <span v-for="(h, i) in BARS" :key="i" class="pr-bar" :style="{ height: h + '%' }" />
        </div>
      </div>
    </section>

    <!-- ── Геймификация: коротко и честно ───────────────────────── -->
    <section class="pr-fun reveal">
      <div class="pr-fun-emoji" aria-hidden="true">
        <EmojiGlyph v-for="(e, i) in PETS" :key="i" :char="e" />
      </div>
      <h2 class="pr-h2">И немного игры — по желанию</h2>
      <p>
        У каждого сотрудника есть грувик: питомец растёт от закрытых задач и отработанных
        часов, а команда обменивается кудосами за помощь. Механику выключает администратор
        одной настройкой — на работу платформы это не влияет.
      </p>
    </section>

    <!-- ── Устройства и скачивание ──────────────────────────────── -->
    <section id="devices" class="pr-section">
      <h2 class="pr-h2 reveal">Одинаково на всех устройствах</h2>
      <p class="pr-section-lead reveal">
        Данные общие, вход один: начните на компьютере, продолжите в дороге с телефона.
      </p>
      <div class="pr-devices">
        <article class="pr-device reveal">
          <span class="material-symbols-outlined">language</span>
          <h3>Браузер</h3>
          <p>Ничего не нужно ставить — откройте адрес и работайте.</p>
          <p class="pr-dl-alt">любой современный браузер</p>
          <RouterLink to="/welcome" class="pr-dl pr-dl--ghost">Открыть</RouterLink>
        </article>

        <article class="pr-device reveal">
          <span class="material-symbols-outlined">desktop_windows</span>
          <h3>Компьютер</h3>
          <p>Отдельное окно, значок в трее и уведомления — даже когда браузер закрыт.</p>
          <p class="pr-dl-alt">
            <template v-if="showDesktop">
              <a :href="desktopFileHref('mac')" download>macOS</a> ·
              <a :href="desktopFileHref('win')" download>Windows</a> ·
              <a :href="desktopFileHref('linux')" download>Linux</a>
            </template>
            <template v-else>macOS · Windows · Linux</template>
          </p>
          <a v-if="showDesktop" class="pr-dl" :href="desktopFileHref(desktopOs)" download>
            <span class="material-symbols-outlined">download</span>
            Скачать для {{ DESKTOP_OS_LABELS[desktopOs] }}
          </a>
        </article>

        <article class="pr-device reveal">
          <span class="material-symbols-outlined">smartphone</span>
          <h3>Телефон</h3>
          <p>Задачи, чаты и звонки под рукой — с пуш-уведомлениями.</p>
          <p class="pr-dl-alt">Android · на iPhone — в браузере</p>
          <a v-if="showApk" class="pr-dl" :href="APK_HREF" :download="apkDownloadName">
            <span class="material-symbols-outlined">download</span>
            Скачать APK
          </a>
        </article>
      </div>
    </section>

    <!-- ── Вопросы ──────────────────────────────────────────────── -->
    <section id="faq" class="pr-section pr-faq">
      <h2 class="pr-h2 reveal">Частые вопросы</h2>
      <details v-for="q in FAQ" :key="q.q" class="pr-q reveal">
        <summary>
          {{ q.q }}
          <span class="material-symbols-outlined">expand_more</span>
        </summary>
        <p>{{ q.a }}</p>
      </details>
    </section>

    <!-- ── Финальный призыв ─────────────────────────────────────── -->
    <section class="pr-final reveal">
      <h2 class="pr-h2">Соберите рабочий день в одном окне</h2>
      <p>Создайте компанию за минуту и позовите команду ссылкой-приглашением.</p>
      <RouterLink to="/welcome" class="pr-btn">Попробовать уже сейчас</RouterLink>
    </section>

    <footer class="pr-foot">
      <span>Groove Work</span>
      <RouterLink to="/welcome">Вход для команд</RouterLink>
    </footer>
  </div>
</template>

<script setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import Logo from '@/components/common/Logo.vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { useAppDownloads } from '@/composables/useAppDownloads.js'

// Скачивание клиентов — общая обвязка с разделом «О приложении»: имена
// артефактов и номер сборки приезжают из version.json рядом с файлами.
const {
  APK_HREF, apkDownloadName, desktopFileHref, desktopOs, DESKTOP_OS_LABELS,
  showApk, showDesktop,
} = useAppDownloads()

const PETS = ['🦊', '🐼', '🐧', '🦄']

/* Синусоида в count полуволн: путь начинается левее холста и заканчивается
   правее, поэтому сдвиг ровно на период (2 × half) края не обнажает —
   цикл получается бесшовным. */
function wavePath(baseY, half, amp, count) {
  let d = `M-720 ${baseY} q${half / 2} ${-amp} ${half} 0`
  for (let i = 1; i < count; i++) d += ` t${half} 0`
  return `${d} V120 H-720 Z`
}

const WAVE_FAR = wavePath(54, 180, 40, 16)   // период 360
const WAVE_MID = wavePath(76, 240, 34, 12)   // период 480
const WAVE_NEAR = wavePath(96, 150, 26, 20)  // период 300

// Высоты столбиков декоративного графика (в процентах) — фиксированные,
// чтобы картинка не «дёргалась» между рендерами.
const BARS = [38, 62, 45, 78, 54, 88, 66]

const PILLARS = [
  {
    icon: 'checklist', title: 'День под рукой',
    text: 'Задачи, ежедневник и напоминания в одном списке — ничего не теряется между приложениями.',
  },
  {
    icon: 'timer', title: 'Время считается само',
    text: 'Пока дело в работе, идёт счётчик: к вечеру видно, на что ушёл день, — без табелей и секундомеров.',
  },
  {
    icon: 'inventory_2', title: 'Всё своё с собой',
    text: 'Заметки, доски, файлы на диске и переписка — на компьютере, в браузере и в телефоне.',
  },
]

const FEATURES = [
  {
    icon: 'task_alt', title: 'Задачи и юниты',
    text: 'Ответственные, этапы, теги и комментарии. Время идёт, пока задача в работе.',
  },
  {
    icon: 'chat', title: 'Мессенджер и звонки',
    text: 'Личные и групповые чаты, файлы, реакции, видеозвонки с демонстрацией экрана.',
  },
  {
    icon: 'campaign', title: 'Корпоративный портал',
    text: 'Новости компании, разделы, обсуждения и реакции — вместо рассылок «всем».',
  },
  {
    icon: 'edit_note', title: 'Заметки и доски',
    text: 'Совместное редактирование текста и рисование на бесконечном холсте.',
  },
  {
    icon: 'calendar_month', title: 'Календари и реестры',
    text: 'Настраиваемые справочники и события компании, личные ежедневники и напоминания.',
  },
  {
    icon: 'blur_on', title: 'Hola и ИИ-ассистент',
    text: 'Поиск по всем разделам, быстрые команды и деловой помощник по данным компании.',
  },
]

const FAQ = [
  {
    q: 'Сколько стоит?',
    a: 'Сейчас бесплатно и целиком: дела, заметки, диск, переписка и звонки доступны сразу после регистрации. Ограничено только место в хранилище — 5 Гб на человека.',
  },
  {
    q: 'Это только для работы или для личных дел тоже?',
    a: 'Прежде всего для ваших личных дел: список задач, ежедневник, напоминания, заметки и файлы принадлежат вам и не зависят ни от какой компании. Появится команда — те же разделы начнут работать и на неё.',
  },
  {
    q: 'А если я работаю в нескольких местах?',
    a: 'Аккаунт принадлежит человеку, а не компании: в каждой у вас своя роль, переключение — одним нажатием. Личные дела, заметки, файлы и переписка при этом одни на всех.',
  },
  {
    q: 'Кто видит мои дела и часы?',
    a: 'Личные разделы видны только вам. В компании свои часы видит каждый, сводку по отделам и людям — менеджеры и администраторы. Администратор платформы к рабочим данным компаний доступа не имеет.',
  },
  {
    q: 'А если геймификация не нужна?',
    a: 'Питомцев-грувиков выключает администратор компании одной настройкой — остальные разделы работают как работали.',
  },
]

// Шапка «прилипла»: маячок верха ушёл из кадра.
const sentinelEl = ref(null)
const stuck = ref(false)
let stickyObserver = null

// Плавное появление секций при скролле (с уважением к reduced-motion).
let observer = null
onMounted(() => {
  if ('IntersectionObserver' in window && sentinelEl.value) {
    stickyObserver = new IntersectionObserver(([entry]) => {
      stuck.value = !entry.isIntersecting
    })
    stickyObserver.observe(sentinelEl.value)
  }
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
onBeforeUnmount(() => {
  observer?.disconnect()
  stickyObserver?.disconnect()
})
</script>

<style scoped>
/* Витрина живёт на ТЕХ ЖЕ токенах, что и приложение: до входа тема всегда
   классическая, а светлая/тёмная берётся у системы (роутер помечает маршрут
   meta.authScreen). Поэтому своей палитры здесь нет — только --color-*. */
.pr {
  min-height: 100dvh;
  background:
    radial-gradient(120% 70% at 12% -8%,
      color-mix(in oklch, var(--color-primary-container) 70%, transparent), transparent 60%),
    radial-gradient(90% 60% at 100% 2%,
      color-mix(in oklch, var(--color-tertiary-container) 45%, transparent), transparent 62%),
    var(--color-bg);
  color: var(--color-text);
  /* Именно clip, а НЕ hidden: hidden делает корень scroll-контейнером, и
     position: sticky у шапки перестаёт работать. */
  overflow-x: clip;
}

.reveal {
  opacity: 0;
  transform: translateY(16px);
  transition: opacity 0.6s ease, transform 0.6s cubic-bezier(0.2, 0.8, 0.3, 1);
}

.reveal.is-visible { opacity: 1; transform: none; }

@media (prefers-reduced-motion: reduce) {
  .reveal { transition: none; }
}

/* ── Шапка ─────────────────────────────────────────────────────────
   Плавающий островок: сверху страницы он прозрачный и не спорит с героем,
   а после прокрутки собирается в стеклянную пилюлю с тенью. Липнет
   обёртка (она во всю ширину), сама пилюля ограничена по ширине контента. */
.pr-sentinel {
  display: block;
  height: 1px;
  margin-bottom: -1px;
}

.pr-top-wrap {
  position: sticky;
  top: 0;
  z-index: 5;
  padding: 10px clamp(12px, 4vw, 40px);
  /* Клики ловит только сама пилюля — прозрачные поля обёртки не перехватывают
     указатель у контента под ней. */
  pointer-events: none;
  transition: padding 0.24s ease;
}

.pr-top {
  pointer-events: auto;
  max-width: 1120px;
  margin: 0 auto;
  display: flex;
  align-items: center;
  gap: 20px;
  padding: 10px 12px 10px 18px;
  border-radius: var(--radius-full);
  border: 1px solid transparent;
  background: transparent;
  box-shadow: none;
  transition: background 0.24s ease, border-color 0.24s ease,
    box-shadow 0.24s ease, padding 0.24s ease;
}

.pr-top-wrap.stuck { padding-top: 12px; }

.pr-top-wrap.stuck .pr-top {
  border-color: var(--acrylic-border);
  background: var(--glass-bg), var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: var(--shadow-lg), var(--glass-edge);
}

@media (prefers-reduced-motion: reduce) {
  .pr-top-wrap,
  .pr-top { transition: none; }
}

.pr-brand {
  display: flex;
  align-items: center;
  gap: 9px;
  text-decoration: none;
}

.pr-wordmark {
  display: flex;
  align-items: baseline;
  gap: 5px;
  font-family: 'Roboto Flex', 'Roboto', sans-serif;
  font-size: 17px;
  font-weight: 1000;
  font-variation-settings: 'wght' 1000;
  letter-spacing: 0.2px;
}

.wm-groove { color: var(--color-primary); }
.wm-work { color: var(--color-text); }

.pr-nav {
  display: flex;
  gap: 22px;
  margin-left: auto;
}

.pr-nav a {
  font-size: 14px;
  font-weight: 500;
  color: var(--color-text-dim);
  text-decoration: none;
  transition: color 0.15s;
}

.pr-nav a:hover { color: var(--color-primary); }

.pr-top-cta {
  padding: 9px 18px;
  border-radius: var(--radius-full);
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  transition: filter 0.15s;
}

.pr-top-cta:hover { filter: brightness(1.07); }

/* ── Герой ─────────────────────────────────────────────────────── */
.pr-hero {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  text-align: center;
  --pr-pad: clamp(16px, 5vw, 56px);
  padding: clamp(28px, 6vw, 84px) var(--pr-pad) 0;
}

.pr-eyebrow {
  margin: 0 0 14px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-primary);
}

.pr-display {
  margin: 0;
  max-width: 16ch;
  font-size: clamp(38px, 7vw, 84px);
  font-weight: 300;
  line-height: 1.03;
  letter-spacing: -0.035em;
}

.pr-lead {
  margin: 22px 0 0;
  max-width: 62ch;
  font-size: clamp(15px, 1.5vw, 18px);
  line-height: 1.6;
  color: var(--color-text-dim);
}

.pr-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 12px;
  margin-top: 32px;
}

.pr-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 50px;
  padding: 0 30px;
  border-radius: var(--radius-full);
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-size: 15px;
  font-weight: 600;
  text-decoration: none;
  box-shadow: var(--shadow-md);
  transition: filter 0.15s, box-shadow 0.15s;
}

.pr-btn:hover { filter: brightness(1.07); box-shadow: var(--shadow-lg); }

.pr-btn--ghost {
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 50%, transparent);
  border: 1px solid var(--acrylic-border);
  color: var(--color-text);
  box-shadow: var(--glass-edge);
}

.pr-btn--ghost:hover {
  background: var(--glass-bg), color-mix(in oklch, var(--color-primary) 12%, transparent);
  box-shadow: var(--glass-edge);
}

.pr-hint {
  margin: 14px 0 0;
  font-size: 13px;
  color: var(--color-text-dim);
}

/* ── Витрина продукта: схематичные окна ────────────────────────── */
.pr-shot {
  position: relative;
  width: min(100%, 980px);
  margin: clamp(40px, 6vw, 72px) auto 0;
  aspect-ratio: 16 / 9;
}

.pr-win {
  position: absolute;
  border: 1px solid var(--acrylic-border);
  border-radius: 18px;
  background: var(--glass-bg), var(--acrylic-bg-strong);
  box-shadow: var(--shadow-xl), var(--glass-edge);
  padding: 14px;
  display: flex;
  flex-direction: column;
  gap: 9px;
}

.pr-win--back { left: 0; top: 6%; width: 42%; height: 56%; opacity: 0.75; }
.pr-win--main { left: 16%; top: 16%; width: 74%; height: 78%; }
.pr-win--front {
  right: 2%;
  bottom: 2%;
  width: 30%;
  padding: 14px 16px;
  background: var(--glass-bg), color-mix(in oklch, var(--color-primary) 14%, var(--acrylic-bg-strong));
}

.pr-win-bar { display: flex; gap: 5px; }
.pr-win-bar i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: color-mix(in oklch, var(--color-outline) 55%, transparent);
}

.pr-win-body { display: flex; gap: 12px; flex: 1; min-height: 0; }
.pr-col { flex: 1; display: flex; flex-direction: column; gap: 8px; min-width: 0; }

.pr-chip {
  align-self: flex-start;
  padding: 3px 10px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-primary) 16%, transparent);
  color: var(--color-primary);
  font-size: 11px;
  font-weight: 600;
}

.pr-chip--alt {
  background: color-mix(in oklch, var(--color-tertiary) 18%, transparent);
  color: var(--color-tertiary);
}

.pr-card {
  display: flex;
  flex-direction: column;
  gap: 7px;
  padding: 12px;
  border-radius: 12px;
  border: 1px solid color-mix(in oklch, var(--acrylic-border) 70%, transparent);
  background: color-mix(in oklch, var(--color-surface) 55%, transparent);
}

.pr-line {
  height: 7px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-text) 12%, transparent);
}

.pr-dot {
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: color-mix(in oklch, var(--color-primary) 45%, transparent);
}

.w-35 { width: 35%; }
.w-40 { width: 40%; }
.w-45 { width: 45%; }
.w-50 { width: 50%; }
.w-55 { width: 55%; }
.w-60 { width: 60%; }
.w-70 { width: 70%; }
.w-75 { width: 75%; }

/* ── Волна под героем ──────────────────────────────────────────────
   Во всю ширину экрана (выходит за поля героя отрицательными отступами) и
   растворяется к углам: радиальная маска гасит слои по краям, поэтому вода
   не обрывается ровным срезом, а «утекает» в фон. */
.pr-wave {
  width: calc(100% + var(--pr-pad) * 2);
  margin-left: calc(var(--pr-pad) * -1);
  margin-top: clamp(36px, 6vw, 72px);
  line-height: 0;
  -webkit-mask-image: radial-gradient(118% 135% at 50% 100%,
    black 48%, color-mix(in oklch, black 45%, transparent) 76%, transparent 96%);
  mask-image: radial-gradient(118% 135% at 50% 100%,
    black 48%, color-mix(in oklch, black 45%, transparent) 76%, transparent 96%);
}

.pr-wave svg { width: 100%; height: clamp(80px, 12vw, 150px); display: block; }

.wv path { fill: var(--color-primary); }
.wv-far path { fill: color-mix(in oklch, var(--color-primary) 62%, black); }
.wv-near path { fill: color-mix(in oklch, var(--color-secondary) 45%, white); }

.wv-far { opacity: 0.5; }
.wv-mid { opacity: 0.55; }
.wv-near { opacity: 0.75; }

[data-dark='true'] .wv-far path { fill: color-mix(in oklch, var(--color-primary) 38%, black); }
[data-dark='true'] .wv-near path { fill: color-mix(in oklch, var(--color-secondary) 42%, black); }

/* Ход волны: сдвиг равен периоду слоя, поэтому конец цикла совпадает с
   началом. Разные скорости и направления дают ощущение живой воды. */
.wv { will-change: transform; }
.wv-far { animation: pr-roll-far 22s linear infinite; }
.wv-mid { animation: pr-roll-mid 15s linear infinite reverse; }
.wv-near { animation: pr-roll-near 30s linear infinite; }

@keyframes pr-roll-far {
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-360px, 0, 0); }
}

@keyframes pr-roll-mid {
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-480px, 0, 0); }
}

@keyframes pr-roll-near {
  from { transform: translate3d(0, 0, 0); }
  to { transform: translate3d(-300px, 0, 0); }
}

@media (prefers-reduced-motion: reduce) {
  .wv { animation: none; }
}

/* ── Секции ────────────────────────────────────────────────────── */
.pr-section {
  max-width: 1120px;
  margin: 0 auto;
  padding: clamp(56px, 8vw, 110px) clamp(16px, 5vw, 40px) 0;
}

.pr-h2 {
  margin: 0;
  font-size: clamp(26px, 3.6vw, 44px);
  font-weight: 300;
  line-height: 1.12;
  letter-spacing: -0.025em;
  text-align: center;
}

.pr-section-lead {
  margin: 16px auto 0;
  max-width: 64ch;
  text-align: center;
  font-size: 15px;
  line-height: 1.6;
  color: var(--color-text-dim);
}

.pr-kicker {
  margin: 0 0 10px;
  font-size: 12.5px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--color-primary);
}

/* Три опоры */
.pr-pillars {
  max-width: 1120px;
  margin: 0 auto;
  padding: clamp(40px, 6vw, 76px) clamp(16px, 5vw, 40px) 0;
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(240px, 100%), 1fr));
  gap: clamp(16px, 2.4vw, 28px);
}

.pr-pillar .material-symbols-outlined {
  font-size: 30px;
  color: var(--color-primary);
  font-variation-settings: 'wght' 300;
}
.pr-pillar h3 { margin: 12px 0 8px; font-size: 19px; font-weight: 600; }
.pr-pillar p { margin: 0; font-size: 14.5px; line-height: 1.6; color: var(--color-text-dim); }

/* Сетка возможностей */
.pr-grid {
  margin-top: clamp(28px, 4vw, 48px);
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(280px, 100%), 1fr));
  gap: clamp(14px, 1.8vw, 20px);
}

.pr-card-feature {
  padding: clamp(20px, 2.4vw, 28px);
  border: 1px solid var(--acrylic-border);
  border-radius: 22px;
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 46%, transparent);
  box-shadow: var(--glass-edge);
  transition: background 0.18s, border-color 0.18s, box-shadow 0.18s;
}

.pr-card-feature:hover {
  border-color: color-mix(in oklch, var(--color-primary) 32%, var(--acrylic-border));
  background: var(--glass-hover-bg), color-mix(in oklch, var(--color-primary) 10%, transparent);
  box-shadow: var(--shadow-md), var(--glass-edge);
}

.pr-card-feature .material-symbols-outlined {
  font-size: 28px;
  color: var(--color-primary);
  font-variation-settings: 'wght' 300;
}

.pr-card-feature h3 { margin: 12px 0 8px; font-size: 18px; font-weight: 600; }
.pr-card-feature p { margin: 0; font-size: 14px; line-height: 1.6; color: var(--color-text-dim); }

/* ── Чередующиеся секции ───────────────────────────────────────── */
.pr-split {
  max-width: 1120px;
  margin: 0 auto;
  padding: clamp(56px, 8vw, 110px) clamp(16px, 5vw, 40px) 0;
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  align-items: center;
  gap: clamp(28px, 5vw, 64px);
}

.pr-split--rev .pr-split-visual { order: -1; }

.pr-split-text .pr-h2 { text-align: left; }
.pr-split-text p {
  margin: 16px 0 0;
  font-size: 15px;
  line-height: 1.65;
  color: var(--color-text-dim);
}

.pr-split-visual { min-width: 0; }

/* Схема рабочего стола */
.pr-desk {
  position: relative;
  aspect-ratio: 4 / 3;
  border: 1px solid var(--acrylic-border);
  border-radius: 22px;
  background:
    radial-gradient(80% 60% at 20% 10%, color-mix(in oklch, var(--color-primary) 18%, transparent), transparent 70%),
    color-mix(in oklch, var(--color-surface) 55%, transparent);
  box-shadow: var(--shadow-lg), var(--glass-edge);
  overflow: hidden;
}

.pr-desk-win {
  position: absolute;
  border-radius: 12px;
  border: 1px solid var(--acrylic-border);
  background: var(--glass-bg), var(--acrylic-bg-strong);
  box-shadow: var(--shadow-md);
}

.pr-desk-win.a { left: 8%; top: 12%; width: 46%; height: 42%; }
.pr-desk-win.b { right: 7%; top: 22%; width: 42%; height: 46%; }
.pr-desk-win.c { left: 20%; bottom: 22%; width: 40%; height: 32%; }

.pr-desk-bar {
  position: absolute;
  left: 50%;
  bottom: 5%;
  transform: translateX(-50%);
  display: flex;
  gap: 8px;
  padding: 7px 12px;
  border-radius: var(--radius-full);
  border: 1px solid var(--acrylic-border);
  background: var(--acrylic-bg-strong);
  box-shadow: var(--shadow-md);
}

.pr-desk-bar i {
  width: 12px;
  height: 12px;
  border-radius: 4px;
  background: color-mix(in oklch, var(--color-primary) 45%, transparent);
}

/* Схема статистики */
.pr-chart {
  display: flex;
  align-items: flex-end;
  gap: clamp(8px, 1.4vw, 16px);
  aspect-ratio: 4 / 3;
  padding: clamp(18px, 2.4vw, 30px);
  border: 1px solid var(--acrylic-border);
  border-radius: 22px;
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 48%, transparent);
  box-shadow: var(--shadow-lg), var(--glass-edge);
}

.pr-bar {
  flex: 1;
  border-radius: 8px 8px 4px 4px;
  background: linear-gradient(180deg,
    var(--color-primary),
    color-mix(in oklch, var(--color-primary) 45%, transparent));
}

/* ── Геймификация ──────────────────────────────────────────────── */
.pr-fun {
  max-width: 720px;
  margin: 0 auto;
  padding: clamp(56px, 8vw, 110px) clamp(16px, 5vw, 40px) 0;
  text-align: center;
}

.pr-fun-emoji {
  display: flex;
  justify-content: center;
  gap: 10px;
  margin-bottom: 18px;
  font-size: 30px;
}

.pr-fun p {
  margin: 16px 0 0;
  font-size: 15px;
  line-height: 1.65;
  color: var(--color-text-dim);
}

/* ── Устройства ────────────────────────────────────────────────── */
.pr-devices {
  margin-top: clamp(28px, 4vw, 44px);
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(220px, 100%), 1fr));
  gap: clamp(16px, 2.4vw, 28px);
  text-align: center;
}

.pr-device > .material-symbols-outlined {
  font-size: 32px;
  color: var(--color-primary);
  font-variation-settings: 'wght' 300;
}

.pr-device {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: clamp(22px, 2.6vw, 30px) clamp(18px, 2.2vw, 26px);
  border: 1px solid var(--acrylic-border);
  border-radius: 22px;
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 46%, transparent);
  box-shadow: var(--glass-edge);
}

.pr-device h3 { margin: 12px 0 8px; font-size: 18px; font-weight: 600; }

/* Описание забирает свободную высоту, поэтому подпись и кнопка у всех
   карточек ряда стоят на одной линии, а не «плавают» каждая по-своему. */
.pr-device p {
  flex: 1;
  margin: 0;
  max-width: 30ch;
  font-size: 14px;
  line-height: 1.6;
  color: var(--color-text-dim);
}

.pr-device .pr-dl-alt {
  flex: 0 0 auto;
  margin: 14px 0 12px;
  font-size: 12.5px;
  color: var(--color-text-dim);
}

.pr-device .pr-dl-alt a { color: var(--color-primary); font-weight: 600; text-decoration: none; }
.pr-device .pr-dl-alt a:hover { text-decoration: underline; }

.pr-dl {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  width: 100%;
  max-width: 260px;
  height: 44px;
  border-radius: var(--radius-full);
  background: var(--grad-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  text-decoration: none;
  box-shadow: var(--shadow-sm);
  transition: filter 0.15s, box-shadow 0.15s;
}

.pr-dl:hover { filter: brightness(1.07); box-shadow: var(--shadow-md); }
/* Значок кнопки — цветом самой кнопки: иначе синий значок карточки
   сливается с градиентной заливкой. */
.pr-dl .material-symbols-outlined { font-size: 19px; color: inherit; }

.pr-dl--ghost {
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 50%, transparent);
  border: 1px solid var(--acrylic-border);
  color: var(--color-text);
  box-shadow: var(--glass-edge);
}

.pr-dl--ghost:hover {
  background: var(--glass-bg), color-mix(in oklch, var(--color-primary) 12%, transparent);
  box-shadow: var(--glass-edge);
}

/* ── Вопросы ───────────────────────────────────────────────────── */
.pr-faq { max-width: 820px; }

.pr-q {
  margin-top: 12px;
  border: 1px solid var(--acrylic-border);
  border-radius: 18px;
  background: var(--glass-bg), color-mix(in oklch, var(--color-surface) 44%, transparent);
  box-shadow: var(--glass-edge);
  overflow: hidden;
}

.pr-q:first-of-type { margin-top: clamp(24px, 3vw, 40px); }

.pr-q summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 16px 20px;
  font-size: 15.5px;
  font-weight: 600;
  cursor: pointer;
  list-style: none;
}

.pr-q summary::-webkit-details-marker { display: none; }

.pr-q summary .material-symbols-outlined {
  color: var(--color-text-dim);
  transition: transform 0.2s;
}

.pr-q[open] summary .material-symbols-outlined { transform: rotate(180deg); }

.pr-q p {
  margin: 0;
  padding: 0 20px 18px;
  font-size: 14.5px;
  line-height: 1.65;
  color: var(--color-text-dim);
}

/* ── Финал и подвал ────────────────────────────────────────────── */
.pr-final {
  max-width: 720px;
  margin: 0 auto;
  padding: clamp(64px, 9vw, 120px) clamp(16px, 5vw, 40px) 0;
  text-align: center;
}

.pr-final p {
  margin: 14px 0 26px;
  font-size: 15px;
  line-height: 1.6;
  color: var(--color-text-dim);
}

.pr-foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  max-width: 1120px;
  margin: clamp(56px, 8vw, 110px) auto 0;
  padding: 22px clamp(16px, 5vw, 40px) 32px;
  border-top: 1px solid color-mix(in oklch, var(--acrylic-border) 60%, transparent);
  font-size: 13.5px;
  color: var(--color-text-dim);
}

.pr-foot a { color: var(--color-primary); font-weight: 600; text-decoration: none; }
.pr-foot a:hover { text-decoration: underline; }

/* ── Адаптив ───────────────────────────────────────────────────── */
@media (max-width: 860px) {
  .pr-split { grid-template-columns: 1fr; text-align: center; }
  .pr-split--rev .pr-split-visual { order: 0; }
  .pr-split-text .pr-h2 { text-align: center; }
}

@media (max-width: 720px) {
  .pr-nav { display: none; }
  .pr-top-cta { margin-left: auto; }
  /* Схема продукта на узком экране — одно окно без наложений. */
  .pr-shot { aspect-ratio: 4 / 3; }
  .pr-win--back, .pr-win--front { display: none; }
  .pr-win--main { left: 0; top: 0; width: 100%; height: 100%; }
  .pr-actions { flex-direction: column; align-items: stretch; width: min(100%, 320px); }
  .pr-btn { width: 100%; }
}
</style>
