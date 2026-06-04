<template>
  <div class="hc">
    <!-- Поиск + быстрые действия -->
    <div class="hc-toolbar">
      <div class="hc-search">
        <span class="material-symbols-outlined">search</span>
        <input
          v-model="query"
          type="search"
          placeholder="О чём хотите узнать? — например, «юнит», «звонок», «отдел»"
        />
        <button v-if="query" class="hc-clear" @click="query = ''" title="Очистить">
          <span class="material-symbols-outlined">close</span>
        </button>
      </div>
      <button class="btn-tonal" @click="startFullTour" title="Начать обзорный тур">
        <span class="material-symbols-outlined">tour</span>
        Обзорный тур
      </button>
    </div>

    <!-- Текущий открытый раздел или список -->
    <Transition name="hc-swap" mode="out-in">
      <!-- Открытый раздел -->
      <div v-if="activeArticle" key="article" class="hc-article">
        <button class="hc-back" @click="activeArticle = null">
          <span class="material-symbols-outlined">arrow_back</span>
          К списку
        </button>

        <header class="hc-article-head">
          <div class="hc-article-icon" :data-tone="activeArticle.tone">
            <span class="material-symbols-outlined">{{ activeArticle.icon }}</span>
          </div>
          <div>
            <h3 class="hc-article-title">{{ activeArticle.title }}</h3>
            <p class="hc-article-sub">{{ activeArticle.subtitle }}</p>
          </div>
        </header>

        <div class="hc-article-body">
          <p
            v-for="(p, i) in activeArticle.paragraphs"
            :key="`p${i}`"
            class="hc-p"
          >{{ p }}</p>

          <div v-if="activeArticle.steps?.length" class="hc-steps">
            <h4>Как этим пользоваться</h4>
            <ol>
              <li v-for="(s, i) in activeArticle.steps" :key="`s${i}`">{{ s }}</li>
            </ol>
          </div>

          <div v-if="activeArticle.tips?.length" class="hc-tips">
            <h4>
              <span class="material-symbols-outlined">tips_and_updates</span>
              Полезно знать
            </h4>
            <ul>
              <li v-for="(t, i) in activeArticle.tips" :key="`t${i}`">{{ t }}</li>
            </ul>
          </div>

          <div class="hc-article-cta">
            <button
              v-if="activeArticle.route"
              class="btn-filled"
              @click="goTo(activeArticle.route)"
            >
              <span class="material-symbols-outlined">arrow_forward</span>
              {{ activeArticle.ctaLabel || 'Перейти в раздел' }}
            </button>
            <button
              v-if="activeArticle.tourTarget"
              class="btn-tonal"
              @click="startTour(activeArticle.tourTarget)"
            >
              <span class="material-symbols-outlined">school</span>
              Показать в туре
            </button>
          </div>
        </div>
      </div>

      <!-- Каталог разделов -->
      <div v-else key="list" class="hc-list">
        <header class="hc-intro hc-intro--spaced">
          <div class="hc-intro-icon">
            <span class="material-symbols-outlined">help_center</span>
          </div>
          <div>
            <h3>Справка по платформе</h3>
            <p>Карточки разделов с пояснениями. Нажмите на любой, чтобы узнать, как этим пользоваться.</p>
          </div>
        </header>

        <div v-for="group in filteredGroups" :key="group.key" class="hc-group">
          <div class="hc-group-label">{{ group.label }}</div>
          <div class="hc-grid">
            <button
              v-for="a in group.articles"
              :key="a.id"
              class="hc-card"
              @click="activeArticle = a"
            >
              <div class="hc-card-icon" :data-tone="a.tone">
                <span class="material-symbols-outlined">{{ a.icon }}</span>
              </div>
              <div class="hc-card-text">
                <span class="hc-card-title">{{ a.title }}</span>
                <span class="hc-card-sub">{{ a.subtitle }}</span>
              </div>
              <span class="material-symbols-outlined hc-card-chev">chevron_right</span>
            </button>
          </div>
        </div>

        <div v-if="!filteredGroups.length" class="hc-empty">
          <div class="hc-empty-icon">
            <span class="material-symbols-outlined">search_off</span>
          </div>
          <h4>Ничего не нашли</h4>
          <p>Попробуйте другие слова — например, «время», «отдел» или «звонок».</p>
        </div>
      </div>
    </Transition>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useTutorial } from '@/composables/useTutorial.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'

const router = useRouter()
const tutorial = useTutorial()
const { isAtLeast } = usePermission()

const query = ref('')
const activeArticle = ref(null)

/* ── Каталог разделов ──────────────────────────────────────────
   Группировка по логическим блокам, чтобы пользователь быстро находил
   нужное. paragraphs — основной текст; steps — пронумерованные действия;
   tips — короткие подсказки; route — кнопка перехода, tourTarget — id
   шага в туре, на который можно «перепрыгнуть». */
const GROUPS = computed(() => [
  {
    key: 'core',
    label: 'Основная работа',
    articles: [
      {
        id: 'tasks',
        title: 'Задачи',
        subtitle: 'Главный экран с задачами команды',
        icon: 'grid_view',
        tone: 'primary',
        route: '/tasks',
        tourTarget: 'tasks-board',
        paragraphs: [
          'Доска задач — центральное место в платформе. Здесь собраны все задачи вашей команды: активные, любимые и архивные.',
          'У каждой задачи свой набор юнитов — отрезков рабочего времени. Над одной задачей могут работать несколько сотрудников параллельно.',
        ],
        steps: [
          'Откройте раздел «Задачи» в боковом меню.',
          'Нажмите «Добавить» в правом верхнем углу, чтобы создать задачу.',
          'Кликните по карточке, чтобы открыть подробности и начать юнит.',
        ],
        tips: [
          'Звёздочка на карточке добавляет задачу в «Избранное» — удобно для приоритетных.',
          'Тег-цвет задачи личный: ваш цвет видите только вы.',
        ],
      },
      {
        id: 'units',
        title: 'Юниты — рабочее время',
        subtitle: 'Как считать часы и над чем именно работали',
        icon: 'timer',
        tone: 'secondary',
        tourTarget: 'units-concept',
        paragraphs: [
          'Юнит — это один логический кусочек работы над задачей: «Дизайн макета», «Монтаж», «Корректура». У каждого юнита свой таймер.',
          'Юниты нельзя ставить на паузу — только начать и завершить. Это сделано намеренно, чтобы статистика отражала реальное время «руки на клавиатуре».',
        ],
        steps: [
          'Откройте задачу и нажмите «Начать».',
          'Введите название юнита и выберите его тип.',
          'Когда работа выполнена — нажмите «Завершить».',
        ],
        tips: [
          'Одновременно может быть только один активный юнит — нельзя «работать над двумя задачами сразу».',
          'Пока юнит запущен, навигация по другим разделам заблокирована — для концентрации.',
        ],
      },
      {
        id: 'stats',
        title: 'Статистика',
        subtitle: 'Сколько времени потрачено и на что',
        icon: 'query_stats',
        tone: 'tertiary',
        route: '/stats',
        tourTarget: 'stats-nav',
        paragraphs: [
          'Статистика — это срез по часам команды: что закрыто, что в работе, кто сколько отработал. Доступна всем сотрудникам, экспорт — менеджерам и выше.',
          'Два режима: «Общая» — сводные таблицы и рейтинг, «Расширенная» — разбивка по типам юнитов, отделам, тепловая карта дней.',
        ],
        tips: [
          'Период переключается кнопками день/неделя/месяц/год или произвольным диапазоном через «Свой период».',
          'ТВ-режим (кнопка на статистике) показывает три слайда подряд для офисного экрана.',
        ],
      },
    ],
  },
  {
    key: 'communication',
    label: 'Общение',
    articles: [
      {
        id: 'employees',
        title: 'Сотрудники',
        subtitle: 'Каталог коллег с онлайн-статусом',
        icon: 'group',
        tone: 'primary',
        route: '/employees',
        paragraphs: [
          'Доска со всеми коллегами: аватар, ФИО, должность. Зелёная точка означает «сейчас в сети», под именем — точное время последнего захода для офлайн-коллег.',
          'Карточка профиля открывается по клику — оттуда можно написать сообщение или сразу позвонить.',
        ],
        tips: [
          'Поиск ищет одновременно по фамилии и по логину.',
          'Кнопки звонка появились в v2.6 — пробуйте аудио или сразу с видео.',
        ],
      },
      {
        id: 'messenger',
        title: 'Мессенджер',
        subtitle: 'Чаты, файлы и история переписки',
        icon: 'chat',
        tone: 'secondary',
        route: '/messenger',
        paragraphs: [
          'Встроенный чат: текст, картинки, видео, аудио, документы до 25 МБ за одно вложение.',
          'Можно отвечать на конкретное сообщение с цитатой, пересылать сообщения и файлы одному или нескольким коллегам.',
          'Удаление — у себя или у обоих. Закрепление пинит важный чат вверх списка лично для вас.',
        ],
        steps: [
          'Откройте «Сотрудники» → карточка коллеги → «Написать».',
          'Перетащите файл прямо в окно чата или вставьте скриншот из буфера (Ctrl+V).',
          'Наведите курсор на сообщение для контекстных действий: ответить, переслать, удалить.',
        ],
        tips: [
          'Маленькая круглая кнопка в правом нижнем углу — мини-чат поверх всего экрана. Работает даже над запущенным юнитом.',
          'Прочтение ставится автоматически, как только вы возвращаетесь к открытому чату.',
        ],
      },
      {
        id: 'calls',
        title: 'Звонки и видеоконференции',
        subtitle: 'Голосовая и видеосвязь до 6 человек',
        icon: 'call',
        tone: 'tertiary',
        paragraphs: [
          'Звонки 1:1 и групповые до 6 человек — прямо в платформе, без сторонних приложений. Аудио, видео или сначала аудио с возможностью включить камеру.',
          'Звонок можно свернуть в маленькое окошко в углу и продолжать работать в других разделах. Микрофон и камеру можно включать/выключать в любой момент.',
        ],
        steps: [
          'Откройте чат с коллегой или его профиль в «Сотрудниках».',
          'Нажмите кнопку с трубкой (аудио) или с камерой (видео).',
          'У собеседника появится входящий звонок с вашим именем и аватаром.',
        ],
        tips: [
          'Свернуть звонок — кнопка в правом верхнем углу окна звонка.',
          'Если коллега уже в звонке — он увидит уведомление о пропущенном.',
        ],
      },
    ],
  },
  {
    key: 'personal',
    label: 'Личное и настройки',
    articles: [
      {
        id: 'profile',
        title: 'Профиль',
        subtitle: 'Ваши данные, пароль и аватар',
        icon: 'account_circle',
        tone: 'primary',
        route: '/profile',
        paragraphs: [
          'В профиле можно сменить пароль, загрузить или сменить аватар. Если аватара нет — система рисует уникальный цветной identicon по вашему ID.',
          'Личная статистика тоже здесь: ваши часы за период, типы юнитов, любимые задачи.',
        ],
      },
      {
        id: 'theme',
        title: 'Внешний вид',
        subtitle: 'Тема, цвет и оформление',
        icon: 'palette',
        tone: 'secondary',
        paragraphs: [
          'Выберите готовую тему или соберите свою из четырёх ключевых цветов. Палитра пересчитывается мгновенно и применяется во всём интерфейсе.',
          'Светлая или тёмная — переключается отдельным сегментом. Своя сохранённая тема появляется в списке «Мои темы».',
        ],
        tips: [
          'Кнопка «Мне повезёт» подбирает случайную гармоничную тему по правилам цветовой теории.',
          'Тему можно экспортировать в JSON и поделиться с коллегой.',
        ],
      },
      {
        id: 'tutorial',
        title: 'Обучение',
        subtitle: 'Интерактивный тур по платформе',
        icon: 'school',
        tone: 'tertiary',
        paragraphs: [
          'Обучающий тур показывает основные экраны: задачи, юниты, статистику, сотрудников, мессенджер. Тур интерактивный — на части шагов реально создаётся тестовая задача и юнит, потом всё аккуратно удаляется.',
        ],
        tips: [
          'Тур можно прервать в любой момент кнопкой «Пропустить» — изменения откатятся.',
          'Запустить тур заново можно отсюда же.',
        ],
      },
    ],
  },
  ...(isAtLeast(ROLES.DIRECTOR) ? [{
    key: 'admin',
    label: 'Администрирование',
    articles: [
      {
        id: 'users-admin',
        title: 'Пользователи',
        subtitle: 'Создание сотрудников и роли',
        icon: 'manage_accounts',
        tone: 'primary',
        paragraphs: [
          'Создание новых сотрудников, назначение ролей, скрытие и удаление. Доступно с уровня «Администратор».',
          'Роль определяет, что человек может в системе: смотреть, редактировать или администрировать. Нельзя назначить роль выше своей.',
        ],
        tips: [
          'Новый сотрудник получает пароль по умолчанию — при первом входе платформа попросит его сменить.',
          'Скрытый пользователь не показывается в каталоге, но история его работы сохраняется.',
        ],
      },
      {
        id: 'lists-admin',
        title: 'Списки',
        subtitle: 'Отделы, типы юнитов и этапы',
        icon: 'list_alt',
        tone: 'secondary',
        paragraphs: [
          'В разделе «Списки» ведутся справочники компании: отделы (для группировки сотрудников и фильтров статистики), типы юнитов (категории работы — дизайн, монтаж, написание кода) и этапы задач (колонки канбан-режима).',
          'Создавать и редактировать — с уровня «Менеджер».',
        ],
        tips: [
          'Удаление типа юнита удаляет все юниты этого типа безвозвратно.',
          'Порядок этапов настраивается перетаскиванием — он определяет порядок колонок в канбан-режиме задач.',
        ],
        route: '/lists',
      },
    ],
  }] : []),
  ...(isAtLeast(ROLES.ADMIN) ? [{
    key: 'system',
    label: 'Система',
    articles: [
      {
        id: 'backup',
        title: 'Резервная копия',
        subtitle: 'Экспорт и восстановление',
        icon: 'backup',
        tone: 'error',
        paragraphs: [
          'Резервная копия — это полный архив базы данных и вложений в одном zip-файле. Доступно только суперадминистратору.',
          'Восстановление полностью заменяет текущие данные содержимым архива. Мы дважды переспросим перед началом — действие необратимо.',
        ],
        tips: [
          'Делайте копию регулярно перед крупными изменениями.',
          'Архив включает аватары и вложения мессенджера — может быть большим.',
        ],
      },
    ],
  }] : []),
])

const allArticles = computed(() => GROUPS.value.flatMap(g => g.articles.map(a => ({ ...a, group: g.label }))))

const filteredGroups = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return GROUPS.value
  return GROUPS.value
    .map(g => ({
      ...g,
      articles: g.articles.filter(a => {
        const haystack = [
          a.title, a.subtitle,
          ...(a.paragraphs || []),
          ...(a.tips || []),
          ...(a.steps || []),
        ].join(' ').toLowerCase()
        return haystack.includes(q)
      }),
    }))
    .filter(g => g.articles.length)
})

function goTo(path) {
  router.push(path)
}

function startTour(target) {
  tutorial.open({ startAt: target })
}

function startFullTour() {
  tutorial.open()
}
</script>

<style scoped>
.hc {
  display: flex;
  flex-direction: column;
  gap: 18px;
}

/* ── Toolbar ─────────────────────────────────────────────────── */
.hc-toolbar {
  display: flex;
  gap: 10px;
  align-items: center;
}

.hc-search {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 16px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 999px;
  transition: border-color 0.15s, background 0.15s;
}

.hc-search:focus-within {
  border-color: var(--color-primary);
  background: var(--color-surface-low);
}

.hc-search > .material-symbols-outlined {
  font-size: 20px;
  color: var(--color-text-dim);
}

.hc-search input {
  flex: 1;
  min-width: 0;
  background: transparent;
  border: 0;
  outline: 0;
  padding: 12px 0;
  font-size: 14px;
  color: var(--color-text);
}

.hc-search input::placeholder { color: var(--color-text-dim); }

.hc-clear {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  border: 0;
  background: transparent;
  display: grid;
  place-items: center;
  cursor: pointer;
  color: var(--color-text-dim);
}

.hc-clear:hover { background: var(--color-surface-high); color: var(--color-text); }
.hc-clear .material-symbols-outlined { font-size: 16px; }

.btn-tonal, .btn-filled {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 11px 20px;
  border-radius: 999px;
  border: 0;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: background 0.15s, transform 0.15s;
}

.btn-tonal {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.btn-tonal:hover { background: color-mix(in oklch, var(--color-secondary-container) 80%, var(--color-on-secondary-container) 20%); }
.btn-tonal .material-symbols-outlined { font-size: 18px; }

.btn-filled {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.btn-filled:hover { background: color-mix(in oklch, var(--color-primary) 88%, var(--color-on-primary) 12%); }
.btn-filled .material-symbols-outlined { font-size: 18px; }

/* ── Intro card ──────────────────────────────────────────────── */
/* hc-list (контейнер списка групп) — отдельный увеличенный gap, чтобы
   между intro-карточкой и первым лейблом группы была заметная воздушная
   пауза. .hc используется и в article-режиме, поэтому общий gap не растим. */
.hc-list {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.hc-intro--spaced { margin-bottom: 6px; }

.hc-intro {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 18px 20px;
  background: linear-gradient(
    135deg,
    color-mix(in oklch, var(--color-primary-container) 90%, transparent),
    color-mix(in oklch, var(--color-tertiary-container) 90%, transparent)
  );
  border-radius: 22px;
  color: var(--color-on-primary-container);
}

.hc-intro-icon {
  width: 52px;
  height: 52px;
  border-radius: 16px;
  background: color-mix(in oklch, var(--color-on-primary-container) 10%, transparent);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}

.hc-intro-icon .material-symbols-outlined { font-size: 28px; }

.hc-intro h3 {
  margin: 0;
  font-size: 18px;
  font-weight: 800;
  letter-spacing: -0.01em;
}

.hc-intro p {
  margin: 2px 0 0;
  font-size: 13px;
  line-height: 1.5;
  color: color-mix(in oklch, var(--color-on-primary-container) 78%, transparent);
}

/* ── Group / Grid ────────────────────────────────────────────── */
.hc-group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.hc-group-label {
  font-size: 11px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--color-text-dim);
  font-weight: 700;
  padding: 0 4px;
}

.hc-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 10px;
}

.hc-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px;
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: 18px;
  cursor: pointer;
  text-align: left;
  transition: background 0.15s, border-color 0.15s, transform 0.15s;
}

.hc-card:hover {
  background: var(--color-surface-low);
  border-color: color-mix(in oklch, var(--color-primary) 30%, var(--color-outline-dim));
  transform: translateY(-2px);
}

.hc-card-icon {
  flex-shrink: 0;
  width: 44px;
  height: 44px;
  border-radius: 14px;
  display: grid;
  place-items: center;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}

.hc-card-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.hc-card-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.hc-card-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.hc-card-icon[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-on-error-container); }

.hc-card-icon .material-symbols-outlined { font-size: 22px; }

.hc-card-text {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.hc-card-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--color-text);
  line-height: 1.2;
}

.hc-card-sub {
  font-size: 12px;
  color: var(--color-text-dim);
  line-height: 1.3;
}

.hc-card-chev {
  font-size: 20px;
  color: var(--color-text-dim);
  opacity: 0.7;
}

/* ── Empty ───────────────────────────────────────────────────── */
.hc-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 40px 20px;
  text-align: center;
  color: var(--color-text-dim);
}

.hc-empty-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: var(--color-surface-high);
  display: grid;
  place-items: center;
}

.hc-empty-icon .material-symbols-outlined { font-size: 32px; opacity: 0.6; }

.hc-empty h4 { margin: 0; font-size: 16px; font-weight: 700; color: var(--color-text); }
.hc-empty p { margin: 0; font-size: 13px; max-width: 320px; line-height: 1.5; }

/* ── Article ─────────────────────────────────────────────────── */
.hc-article {
  display: flex;
  flex-direction: column;
  gap: 18px;
  max-width: 720px;
}

.hc-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  align-self: flex-start;
  padding: 8px 14px;
  background: var(--color-surface-high);
  border: 0;
  border-radius: 999px;
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s;
}

.hc-back:hover { background: var(--color-surface-highest); }
.hc-back .material-symbols-outlined { font-size: 16px; }

.hc-article-head {
  display: flex;
  align-items: center;
  gap: 14px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--color-outline-dim);
}

.hc-article-icon {
  width: 56px;
  height: 56px;
  border-radius: 18px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
  background: var(--tone-bg, var(--color-primary-container));
  color: var(--tone-fg, var(--color-on-primary-container));
}

.hc-article-icon[data-tone="primary"]   { --tone-bg: var(--color-primary-container);   --tone-fg: var(--color-on-primary-container); }
.hc-article-icon[data-tone="secondary"] { --tone-bg: var(--color-secondary-container); --tone-fg: var(--color-on-secondary-container); }
.hc-article-icon[data-tone="tertiary"]  { --tone-bg: var(--color-tertiary-container);  --tone-fg: var(--color-on-tertiary-container); }
.hc-article-icon[data-tone="error"]     { --tone-bg: var(--color-error-container);     --tone-fg: var(--color-on-error-container); }

.hc-article-icon .material-symbols-outlined { font-size: 30px; }

.hc-article-title {
  margin: 0;
  font-size: 22px;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--color-text);
}

.hc-article-sub {
  margin: 4px 0 0;
  font-size: 13px;
  color: var(--color-text-dim);
}

.hc-article-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

.hc-p {
  margin: 0;
  font-size: 14px;
  line-height: 1.65;
  color: var(--color-text);
}

.hc-steps,
.hc-tips {
  padding: 16px 18px;
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: 16px;
}

.hc-steps h4,
.hc-tips h4 {
  margin: 0 0 10px;
  font-size: 13px;
  font-weight: 700;
  color: var(--color-text);
  display: flex;
  align-items: center;
  gap: 6px;
}

.hc-tips h4 { color: var(--color-on-tertiary-container); }
.hc-tips { background: var(--color-tertiary-container); border-color: transparent; color: var(--color-on-tertiary-container); }
.hc-tips ul { margin: 0; padding-left: 24px; display: flex; flex-direction: column; gap: 6px; font-size: 13px; line-height: 1.55; }
.hc-tips h4 .material-symbols-outlined { font-size: 16px; }

.hc-steps ol { margin: 0; padding-left: 22px; display: flex; flex-direction: column; gap: 6px; font-size: 13px; line-height: 1.55; color: var(--color-text); }

.hc-article-cta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  padding-top: 4px;
}

/* ── Transitions ─────────────────────────────────────────────── */
.hc-swap-enter-active, .hc-swap-leave-active { transition: opacity 0.18s, transform 0.18s; }
.hc-swap-enter-from { opacity: 0; transform: translateY(8px); }
.hc-swap-leave-to { opacity: 0; transform: translateY(-8px); }

/* ── Mobile ─────────────────────────────────────────────────── */
@media (max-width: 700px) {
  .hc-toolbar { flex-direction: column; align-items: stretch; }
  .hc-toolbar .btn-tonal { width: 100%; justify-content: center; }
  .hc-grid { grid-template-columns: 1fr; }
  .hc-intro { padding: 16px; gap: 12px; }
  .hc-intro-icon { width: 44px; height: 44px; border-radius: 14px; }
  .hc-intro-icon .material-symbols-outlined { font-size: 22px; }
  .hc-intro h3 { font-size: 16px; }
  .hc-intro p { font-size: 12px; }
  .hc-article-head { flex-direction: row; align-items: flex-start; }
  .hc-article-title { font-size: 18px; }
  .hc-article-icon { width: 44px; height: 44px; border-radius: 14px; }
  .hc-article-icon .material-symbols-outlined { font-size: 24px; }
}
</style>
