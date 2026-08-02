<template>
  <div class="page" :class="{ 'no-pad': bare || embedded, embedded }">
    <section ref="panelEl" class="page-panel" :class="{ bare }">
      <header v-if="!headless && hasHead" class="page-head">
        <!-- Название занимает свою строку целиком: рядом с кнопками длинные
             имена (реестра, заметки, компании) обрезались до многоточия почти
             сразу. Кнопка возврата остаётся при названии — она о нём и есть. -->
        <div v-if="hasTitleRow" class="head-title-row">
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

          <div v-if="slots.status" class="head-status"><slot name="status" /></div>
        </div>

        <!-- Строка управления: поиск и вкладки слева, команды справа. Когда
             названия нет (оно уже в рамке окна), эта строка — единственная. -->
        <div v-if="hasControlsRow" class="head-line">
          <AppButton
            v-if="menu && !hasTitleRow"
            variant="icon"
            :icon="menuIcon"
            size="sm"
            :aria-label="menuLabel"
            :title="menuLabel"
            @click="$emit('menu')"
          />

          <AppButton
            v-if="back && !hasTitleRow"
            variant="icon"
            icon="arrow_back"
            size="sm"
            :aria-label="backLabel"
            :title="backLabel"
            @click="$emit('back')"
          />

          <div v-if="slots.subhead" class="head-sub"><slot name="subhead" /></div>

          <div v-if="slots.status && !hasTitleRow" class="head-status"><slot name="status" /></div>

          <AppCommandBar
            v-if="barCommands.length"
            class="head-commands"
            :commands="barCommands"
            @command="$emit('command', $event)"
          />

          <div v-else-if="slots.commands" class="head-commands"><slot name="commands" /></div>
        </div>
      </header>

      <div class="page-body" :class="{ flush, 'no-scroll': !scroll }">
        <div v-if="loading" class="page-state"><BrandLoader /></div>
        <slot v-else />
      </div>

      <footer v-if="slots.footer" class="page-foot"><slot name="footer" /></footer>
    </section>

    <!-- Тесная панель: главное действие уходит из шапки на плавающую кнопку,
         остальные команды сворачиваются в меню «ещё». -->
    <AppFab
      v-if="fabCommand && narrowPage"
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
import { computed, onBeforeUnmount, onMounted, ref, useSlots } from 'vue'
import { useRoute } from 'vue-router'
import AppButton from './AppButton.vue'
import AppCommandBar from './AppCommandBar.vue'
import AppFab from '@/components/ui/AppFab.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import { appForPath } from '@/desktop/apps.js'
import { useModalHost } from '@/desktop/windowHost.js'

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
})

defineEmits(['back', 'command', 'menu'])
const slots = useSlots()

/* Раздел, открытый окном рабочего стола, уже подписан заголовком самого окна —
   второй раз ТО ЖЕ название не пишем, а строку отдаём под управление. Скрывается
   только совпадающий заголовок: имя выбранного реестра или раздела настроек в
   рамке окна не написано, и без него панель осталась бы безымянной. В мобильном
   каркасе и на самостоятельных страницах окна нет — заголовок остаётся. */
const { inWindow } = useModalHost()
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

const hasTitleRow = computed(() => showsTitle.value || !!slots.title)

/* Главное действие раздела (`fab: true`) в тесной панели уезжает из шапки на
   плавающую кнопку: в узкой строке оно всё равно осталось бы одной иконкой без
   подписи и делило бы место с второстепенными командами. */
const panelEl = ref(null)
const narrowPage = ref(false)
const NARROW_AT = 640

let ro = null

onMounted(() => {
  if (typeof ResizeObserver === 'undefined' || !panelEl.value) return
  ro = new ResizeObserver(([entry]) => {
    narrowPage.value = entry.contentRect.width < NARROW_AT
  })
  ro.observe(panelEl.value)
})

onBeforeUnmount(() => ro?.disconnect())

const fabCommand = computed(() => props.commands.find((c) => c.fab && !c.hidden) || null)

const barCommands = computed(() =>
  narrowPage.value && fabCommand.value
    ? props.commands.filter((c) => c !== fabCommand.value)
    : props.commands,
)

const hasControlsRow = computed(() =>
  !!slots.subhead || !!slots.commands || barCommands.value.length > 0 ||
  (!hasTitleRow.value && (props.back || props.menu || !!slots.status)),
)

const hasHead = computed(() => hasTitleRow.value || hasControlsRow.value)
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

/* Строка названия: кнопок здесь нет, поэтому длинное имя занимает её целиком. */
.head-title-row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
}

/* Строка управления — мерка для панели команд: она сворачивается, когда
   содержимое перестаёт помещаться сюда. Обрезка нужна на время подгонки. */
.head-line {
  display: flex;
  align-items: center;
  gap: 12px;
  min-width: 0;
  overflow: hidden;
}

.page-title {
  flex: 1 1 auto;
  min-width: 0;
  margin: 0;
  font-size: 1.65rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Поиск и вкладки делят строку с командами: забирают свободное место, но
   уступают его командам — сжимаются первыми. */
.head-sub {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1 1 auto;
  min-width: 0;
}

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

/* Узкая панель: заголовок мельче, статус-чипы уходят под строку заголовка. */
@container (max-width: 620px) {
  .page-title { font-size: 1.3rem; }
}

@media (max-width: 768px) {
  .page { padding: 0; }
  .page-panel { border: none; border-radius: 0; }
  .page-head { padding: 14px 14px 10px; }
  .page-title { font-size: 1.3rem; }
  .page-body { padding: 4px 14px 16px; }
  .page-body.flush { padding: 0; }
  .page-foot { padding: 10px 14px; }
}
</style>
