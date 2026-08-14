<template>
  <div class="page" :class="{ 'no-pad': bare || embedded, embedded, edge: flushShell }">
    <section ref="panelEl" class="page-panel" :class="{ bare, 'has-bg': !!slots.background }">
      <!-- Обои раздела (лента портала, переписка): слой лежит ПОД содержимым
           панели и клипается её скруглением. Панель для этого делается
           позиционированной и изолированной — иначе слой с z-index: -1 уехал бы
           под фон самой панели. -->
      <slot name="background" />

      <header v-if="!headless && hasHead()" class="page-head">
        <!-- Название занимает свою строку целиком: рядом с кнопками длинные
             имена (реестра, заметки, компании) обрезались до многоточия почти
             сразу. Кнопка возврата остаётся при названии — она о нём и есть. -->
        <div v-if="hasTitleRow()" class="head-title-row">
          <AppButton
            v-if="menu"
            variant="icon"
            :icon="menuIcon"
            size="sm"
            :aria-label="menuLabel"
            :title="menuLabel"
            @click="$emit('menu')"
          />

          <AppButton
            v-if="back"
            variant="icon"
            icon="arrow_back"
            size="sm"
            :aria-label="backLabel"
            :title="backLabel"
            @click="$emit('back')"
          />

          <slot name="title">
            <h1 v-if="showsTitle" class="page-title">{{ title }}</h1>
          </slot>

          <div v-if="slots.status" class="head-status"><slot name="status" :narrow="narrowPage" /></div>

          <!-- Поиск раздела: НА ТЕЛЕФОНЕ он свёрнут в лупу и встаёт ПРЯМО в
               строку названия — там отдельная строка ради одного поля съедала
               бы у содержимого полсотни пикселей. -->
          <div v-if="slots.search && compactSearch" class="head-search compact">
            <slot name="search" :narrow="true" />
          </div>

          <!-- Команды стоят при названии, а не при фильтрах: так строка ниже
               целиком достаётся вкладкам и поиску, и ничто не наезжает. В тесной
               панели они уходят на свою строку (см. ниже). -->
          <AppCommandBar
            v-if="barCommands.length && !commandsBelow()"
            class="head-commands"
            :commands="barCommands"
            @command="$emit('command', $event)"
          />

          <div v-else-if="slots.commands && !commandsBelow()" class="head-commands">
            <slot name="commands" />
          </div>
        </div>

        <!-- Тесная панель: команда занимает всю ширину — прижатая к углу кнопка
             там читается хуже, а места для соседей всё равно нет. -->
        <AppCommandBar
          v-if="commandsBelow()"
          :commands="barCommands"
          block
          @command="$emit('command', $event)"
        />

        <!-- Строка управления: вкладки, фильтры, поиск. Когда названия нет (оно
             уже в рамке окна), команды переезжают сюда. -->
        <div v-if="hasControlsRow()" class="head-line">
          <AppButton
            v-if="menu && !hasTitleRow()"
            variant="icon"
            :icon="menuIcon"
            size="sm"
            :aria-label="menuLabel"
            :title="menuLabel"
            @click="$emit('menu')"
          />

          <AppButton
            v-if="back && !hasTitleRow()"
            variant="icon"
            icon="arrow_back"
            size="sm"
            :aria-label="backLabel"
            :title="backLabel"
            @click="$emit('back')"
          />

          <div v-if="slots.search && !compactSearch" class="head-search"><slot name="search" :narrow="false" /></div>

          <div v-if="slots.subhead" class="head-sub"><slot name="subhead" :narrow="narrowPage" /></div>

          <div v-if="slots.status && !hasTitleRow()" class="head-status"><slot name="status" /></div>

          <template v-if="!hasTitleRow()">
            <AppCommandBar
              v-if="barCommands.length"
              class="head-commands"
              :commands="barCommands"
              @command="$emit('command', $event)"
            />

            <div v-else-if="slots.commands" class="head-commands"><slot name="commands" /></div>
          </template>
        </div>
      </header>

      <!-- `narrow` отдаём содержимому: раздел живёт окном, и вид (таблица или
           карточки) должен зависеть от ширины ПАНЕЛИ, а не экрана. -->
      <div class="page-body" :class="{ flush, 'no-scroll': !scroll }">
        <div v-if="loading" class="page-state"><BrandLoader /></div>
        <slot v-else :narrow="narrowPage" />
      </div>

      <footer v-if="slots.footer" class="page-foot"><slot name="footer" /></footer>
    </section>

    <!-- Тесная панель: главное действие уходит из шапки на плавающую кнопку,
         остальные команды сворачиваются в меню «ещё». -->
    <AppFab
      v-if="fabCommand"
      :icon="fabCommand.icon || 'add'"
      :label="fabCommand.label"
      :aria-label="fabCommand.label"
      @click="$emit('command', fabCommand.key)"
    />
  </div>
</template>

<script setup>
/* Каркас раздела — одна стеклянная панель со статичной шапкой и прокручиваемым
   телом. Свёл воедино четыре разошедшихся каркаса: `.admin-page`, `.gw-shell`,
   `.settings-shell` и самодельные шеллы разделов.

   Раздел открывается ОКНОМ рабочего стола либо занимает экран телефона целиком,
   поэтому панель объявляет `container-type` — вложенные блоки меряют её ширину,
   а не ширину экрана. На телефоне полей по краям нет, а рамка и скругления
   снимаются: их всё равно обрезала бы кромка экрана. */
import { computed, ref, useSlots, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppButton from './AppButton.vue'
import AppCommandBar from './AppCommandBar.vue'
import AppFab from '@/components/ui/AppFab.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import { appForPath } from '@/desktop/apps.js'
import { useFlushShell, useModalHost } from '@/desktop/windowHost.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useNarrowWidth } from '@/composables/useNarrowWidth.js'

const props = defineProps({
  title: { type: String, default: '' },
  /** Кнопка возврата слева от заголовка. */
  back: { type: Boolean, default: false },
  backLabel: { type: String, default: 'Назад' },
  /** Кнопка сворачивания боковой панели (её состояние даёт AppListDetail). */
  menu: { type: Boolean, default: false },
  menuIcon: { type: String, default: 'menu' },
  menuLabel: { type: String, default: 'Свернуть панель' },
  /** Команды шапки — см. AppCommandBar. */
  commands: { type: Array, default: () => [] },
  /** Показать лоадер вместо содержимого. */
  loading: { type: Boolean, default: false },
  /** Тело без внутренних полей — содержимое рисует их само (лента, холст, таблица). */
  flush: { type: Boolean, default: false },
  /** Скроллит содержимое, а не тело (колонки задач, редакторы). */
  scroll: { type: Boolean, default: true },
  /** Без шапки — раздел рисует свою (редакторы во весь экран). */
  headless: { type: Boolean, default: false },
  /** Без панели и полей — раздел сам себе фон (холст доски, лента чата). */
  bare: { type: Boolean, default: false },
  /** Колонка внутри AppListDetail: панель без внешних полей, растянута на ячейку. */
  embedded: { type: Boolean, default: false },
  /** Заставить показать/скрыть заголовок вопреки правилу «в окне он лишний». */
  showTitle: { type: Boolean, default: null },
  /** Когда поиск сворачивается в лупу при названии: 'phone' — только на
   *  телефоне (на столе он всегда полноценная строка), 'narrow' — в любой
   *  тесной панели (колонка списка чатов узка ПО ЗАМЫСЛУ, и строка поиска
   *  стоила бы списку целой строки). */
  searchCompact: {
    type: String,
    default: 'phone',
    validator: (v) => ['phone', 'narrow'].includes(v),
  },
})

const emit = defineEmits(['back', 'command', 'menu', 'narrow-change'])
const slots = useSlots()

/* Раздел, открытый окном рабочего стола, уже подписан заголовком самого окна —
   второй раз ТО ЖЕ название не пишем, а строку отдаём под управление. Скрывается
   только совпадающий заголовок: имя выбранного реестра или раздела настроек в
   рамке окна не написано, и без него панель осталась бы безымянной. В мобильном
   каркасе и на самостоятельных страницах окна нет — заголовок остаётся. */
const { inWindow } = useModalHost()
/* Сенсорный каркас: у экрана нет рамки, поэтому поля вокруг панели — пустая
   полоса вдоль кромки (а на планшете между двумя зонами — двойной зазор). */
const flushShell = useFlushShell()
const route = useRoute()

const windowTitle = computed(() => {
  if (!inWindow.value) return ''
  // Внутри окна useRoute() отдаёт маршрут ЭТОГО окна (см. desktop/windowRoute.js).
  const path = route?.path
  const app = path ? appForPath(path) : null
  return (app?.titleFor?.(route) || app?.title) ?? ''
})

const showsTitle = computed(() => {
  if (props.showTitle !== null) return props.showTitle && !!props.title
  if (!props.title) return false
  return !inWindow.value || props.title !== windowTitle.value
})

/* Наличие слотов проверяем ФУНКЦИЯМИ, а не computed: объект слотов не
   реактивен, и вычисляемое значение поверх него кэшируется навсегда. Раздел,
   объявляющий слот условно (`<template v-if="registry" #search>`), из-за этого
   оставался без строки поиска: на первом кадре слота ещё не было, а пересчёта
   потом уже не случалось. Функция зовётся из шаблона и считается на каждой
   отрисовке — а её каркас получает вместе с новыми слотами. */
function hasTitleRow() {
  return showsTitle.value || !!slots.title
}

/* Главное действие раздела (`fab: true`) в тесной панели уезжает из шапки на
   плавающую кнопку: в узкой строке оно всё равно осталось бы одной иконкой без
   подписи и делило бы место с второстепенными командами. */
const panelEl = ref(null)
const narrowPage = useNarrowWidth(panelEl, 640)

/* Свёрнутый в лупу поиск — приём ТЕЛЕФОНА, а не узкой панели: на столе окно
   бывает уже 640px сколько угодно, но вертикали там вдоволь, и поиск обязан
   оставаться полноценной строкой — искать по таблице приходится постоянно.
   Отсюда единственное место, где каркас смотрит на устройство, а не на себя
   (разделы со своей узкой колонкой просят прежнее правило `search-compact`). */
const { isMobile } = useBreakpoint()

const compactSearch = computed(() =>
  narrowPage.value && (props.searchCompact === 'narrow' || isMobile.value),
)

/* Тесноту панели знает не только шапка: от неё зависят и состав команд, и вид
   содержимого — а слот-проп доступен лишь внутри тела (см. AppListDetail). */
watch(narrowPage, (v) => emit('narrow-change', v))

/* Плавающая кнопка — приём ТЕЛЕФОНА, как и свёрнутый поиск: на рабочем столе
   окно бывает уже 640px сколько угодно, но там она просто накрывает содержимое
   собой, а главное действие и так стоит в шапке. Поэтому решает устройство, а
   не ширина панели. */
const fabCommand = computed(() =>
  (isMobile.value && props.commands.find((c) => c.fab && !c.hidden)) || null,
)

const barCommands = computed(() =>
  fabCommand.value ? props.commands.filter((c) => c !== fabCommand.value) : props.commands,
)

/* Отдельная строка нужна только главной команде: одинокое меню «ещё» на всю
   ширину выглядело бы странно и лучше остаётся при названии. */
function commandsBelow() {
  return hasTitleRow() && narrowPage.value && barCommands.value.some((c) => c.primary)
}

/* Команды считаются за строку управления, ТОЛЬКО когда рисуются в ней: при
   заголовке они стоят в его строке, и раньше шапка добавляла ниже пустую
   строку — лишний отступ на пустом месте (заметно на телефоне). */
function hasControlsRow() {
  return !!slots.subhead ||
    (!!slots.search && !compactSearch.value) ||
    (!hasTitleRow() && (
      !!slots.commands || barCommands.value.length > 0 || props.back || props.menu || !!slots.status
    ))
}

function hasHead() {
  return hasTitleRow() || hasControlsRow()
}
</script>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 16px;
  overflow: hidden;
}

.page.no-pad { padding: 0; }
.page.embedded { height: 100%; overflow: hidden; }

.page-panel {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-xl);
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  container-type: inline-size;
}

.page-panel.has-bg {
  position: relative;
  isolation: isolate;
  overflow: hidden;
}

.page-panel.bare {
  border: none;
  border-radius: 0;
  background: none;
  -webkit-backdrop-filter: none;
  backdrop-filter: none;
}

.page-head {
  display: flex;
  flex-direction: column;
  gap: 12px;
  flex-shrink: 0;
  padding: 18px 20px 12px;
}

/* Строка названия и команд. В тесной панели команды переносятся под название и
   встают во всю ширину — в UWP главное действие панели навигации выглядит
   именно так, а ужатая в угол кнопка читается хуже. */
.head-title-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px 12px;
  min-width: 0;
}

.head-title-row > .head-commands { flex: 0 0 auto; }

/* Строка управления. Переносится по мере надобности: в узкой панели вкладки и
   кнопка действия не должны налезать друг на друга — каждой достаётся своя
   строка. */
.head-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px 12px;
  min-width: 0;
}

/* Базис 0, а не auto: с «auto» гипотетический размер названия равен его тексту
   целиком, и длинное имя (реестра, календаря) выталкивало кнопку возврата и
   команды каждую на свою строку — шапка вырастала втрое. */
.page-title {
  flex: 1 1 0;
  min-width: 0;
  margin: 0;
  font-size: 1.65rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Поиск и вкладки делят строку с командами и забирают свободное место. Базис
   задан не «auto», а конкретной шириной: сжиматься до нечитаемой полоски им
   нельзя — при нехватке места они переезжают на свою строку целиком. */
.head-sub {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1 1 240px;
  min-width: 0;
}

/* Поиску нужен собственный минимум: со сжатием до нуля он превращался в
   бесполезную пилюлю с одной лупой. Не хватило места — переезжает на свою строку. */
.head-sub > :deep(.search-field) { flex: 1 1 200px; }

/* Поиск раздела: широко — тянется вместе с фильтрами, тесно — кнопка-лупа
   при названии. */
.head-search { display: flex; align-items: center; flex: 1 1 200px; min-width: 0; }
.head-search.compact { flex: 0 0 auto; }

.head-status {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
  min-width: 0;
}

/* Команды прижаты вправо и НЕ сжимаются — лишние уходят в меню «ещё» сами. */
.head-commands {
  flex: 0 0 auto;
  margin-left: auto;
  justify-content: flex-end;
}

.page-subhead {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
}

.page-body {
  flex: 1;
  min-height: 0;
  padding: 6px 20px 20px;
  overflow-y: auto;
}

.page-body.flush { padding: 0; }
.page-body.no-scroll { overflow: hidden; display: flex; flex-direction: column; }

.page-state {
  display: grid;
  place-items: center;
  min-height: 240px;
  height: 100%;
}

.page-foot {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 12px 20px;
  border-top: 1px solid var(--acrylic-border);
}

/* Тесная панель: шапка отдаёт содержимому всё, что может, — заголовок мельче,
   поля и промежутки между строками управления сжаты. На телефоне шапка раздела
   легко съедала треть экрана, а под запись оставалась пара строк. */
@container (max-width: 620px) {
  .page-head { gap: 8px; padding: 12px 14px 8px; }
  .head-title-row { gap: 8px; }
  .head-line { gap: 8px; }

  /* Тесная шапка: название (реестра, заметки, календаря) пишем ЦЕЛИКОМ, с
     переносом — многоточие съедало половину имени, а именно по нему человек и
     понимает, где он. Строка остаётся за названием: значок статуса переезжает
     под него, рядом остаются только кнопки-иконки. */
  .page-title {
    font-size: 1.22rem;
    white-space: normal;
    overflow: visible;
    text-overflow: clip;
    overflow-wrap: anywhere;
  }

  .head-title-row > .head-status { flex-basis: 100%; }
}

/* Каркас без рамки (телефон и планшет): панель занимает экран или зону
   целиком — кромка экрана и есть край раздела. */
.page.edge { padding: 0; }
.page.edge > .page-panel { border: none; border-radius: 0; }

@media (max-width: 768px) {
  .page { padding: 0; }
  .page-panel { border: none; border-radius: 0; }
  .page-head { gap: 8px; padding: 12px 14px 8px; }
  .head-title-row, .head-line { gap: 8px; }
  .page-body { padding: 4px 14px 16px; }
  .page-body.flush { padding: 0; }
  .page-foot { padding: 10px 14px; }

  /* Дубль правил выше для заводского WebView старых Android (@container он не
     знает): название целиком, статус — под ним. */
  .page-title {
    font-size: 1.22rem;
    white-space: normal;
    overflow: visible;
    text-overflow: clip;
    overflow-wrap: anywhere;
  }

  .head-title-row > .head-status { flex-basis: 100%; }
}
</style>
