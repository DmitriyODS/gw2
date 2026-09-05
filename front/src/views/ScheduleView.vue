<template>
  <AppListDetail
    v-model:open="detailOpen"
    :loading="store.loadingList && !store.schedules.length"
    @narrow-change="narrow = $event"
  >
    <!-- Список расписаний. Их много и они независимы: у каждого свой цикл,
         свои категории и поля — папок и наборов у раздела нет. -->
    <template #list="{ toggle }">
      <AppPage
        embedded
        title="Расписания"
        show-title
        :menu="!narrow"
        menu-icon="left_panel_close"
        menu-label="Свернуть список"
        @menu="toggle"
      >
        <template #subhead>
          <!-- Вкладки делят строку поровну: колонка списка узка по замыслу, и
               пилюля по содержимому оставляла справа пустое место, а сами
               вкладки выходили разной ширины. -->
          <AppTabs
            :model-value="store.tab"
            :tabs="[
              { value: 'mine', label: 'Мои' },
              { value: 'shared', label: 'Поделились' },
            ]"
            class="sv-tabs"
            full-width
            @update:model-value="store.setTab($event)"
          />
        </template>

        <EmptyState
          v-if="!store.schedules.length"
          size="sm"
          icon="calendar_view_week"
          :title="store.tab === 'mine' ? 'Расписаний нет' : 'С вами не делились'"
          :subtitle="store.tab === 'mine'
            ? 'Заведите расписание — учёбы, работы или тренировок.'
            : 'Здесь появятся расписания, которые вам открыли.'"
        />
        <AppStack v-else :gap="6">
          <template v-for="s in store.schedules" :key="s.id">
            <AppInlineEdit
              v-if="renamingId === s.id"
              :model-value="s.name"
              placeholder="Название расписания"
              :maxlength="120"
              @save="applyRename(s, $event)"
              @cancel="renamingId = null"
            />
            <AppRow
              v-else
              :title="s.name"
              :hint="scheduleHint(s)"
              icon="calendar_view_week"
              dense
              clickable
              :selected="s.id === store.selectedId"
              @click="openSchedule(s.id)"
              @contextmenu.prevent="openRowMenu(s, $event)"
            />
          </template>
        </AppStack>

        <template v-if="store.tab === 'mine'" #footer>
          <AppButton variant="glass" icon="add" label="Новое расписание" @click="createOpen = true" />
        </template>
      </AppPage>
    </template>

    <!-- Шкала выбранного расписания. -->
    <template #detail="{ collapsed, toggle }">
      <AppPage
        embedded
        :title="store.selected?.name || ''"
        :back="narrow"
        back-label="К расписаниям"
        :menu="!narrow && collapsed"
        menu-icon="left_panel_open"
        menu-label="Показать список"
        :commands="commands"
        :scroll="false"
        @back="detailOpen = false"
        @menu="toggle"
        @command="onCommand"
      >
        <!-- Управление шкалой стоит В СТРОКЕ НАЗВАНИЯ: своя строка ради
             недельной пилюли и одной кнопки съедала у шкалы полсотни пикселей.
             Не поместилось — строка переносит его сама. -->
        <template v-if="store.selected && !narrow" #status>
          <ScheduleWeekNav
            :range="weekRange"
            :cycle="store.selected.cycle_weeks > 1 ? store.cycleWeekLabel : ''"
            :current="onCurrentWeek"
            @step="store.stepWeek($event)"
            @today="store.today()"
          />
        </template>

        <!-- Тесная панель и телефон: название занимает строку целиком, поэтому
             навигация уходит под него и растягивается на всю ширину. -->
        <template v-if="store.selected && narrow" #subhead>
          <ScheduleWeekNav
            wide
            :range="weekRange"
            :cycle="store.selected.cycle_weeks > 1 ? store.cycleWeekLabel : ''"
            :current="onCurrentWeek"
            @step="store.stepWeek($event)"
            @today="store.today()"
          />
        </template>

        <template v-if="store.selected">
          <BrandLoader v-if="store.loadingItems" :size="64" block />
          <div v-else class="sv-body">
            <!-- Телефон и тесная панель: один день и полоса дней. Неделя
                 колонками там нечитаема — колонка уже названия занятия. -->
            <AppTabs
              v-if="narrow"
              class="sv-days"
              :model-value="store.selectedWeekday"
              :tabs="dayTabs"
              variant="tint"
              dense
              full-width
              @update:model-value="store.selectedWeekday = $event"
            />

            <ScheduleTimeline
              :schedule="store.selected"
              :items="store.items"
              :monday="store.monday"
              :days="narrow ? [store.selectedWeekday] : days"
              :gap-min="store.gapMin"
              :mode="narrow ? 'day' : 'week'"
              :editing="store.editing && !store.readonly"
              :readonly="store.readonly"
              @open="openItem"
              @create="startItem"
            />

            <ScheduleSummary
              :items="summaryItems"
              :gap-min="store.gapMin"
              :label="summaryLabel"
            />
          </div>
        </template>
        <EmptyState
          v-else
          icon="calendar_view_week"
          title="Выберите расписание"
          subtitle="Слева — ваши расписания и те, которыми с вами поделились."
        />
      </AppPage>
    </template>
  </AppListDetail>

  <!-- Новое расписание -->
  <AppDialog v-model="createOpen" title="Новое расписание" size="sm" :busy="creating">
    <AppStack :gap="14">
      <AppField v-slot="{ id }" label="Название" required>
        <InputText
          :id="id"
          v-model="createForm.name"
          maxlength="120"
          placeholder="Учёба"
          autofocus
          @keyup.enter="doCreate"
        />
      </AppField>
      <AppField
        v-slot="{ id }"
        label="Недель в цикле"
        hint="Две недели — это числитель и знаменатель. Цикл можно изменить позже."
      >
        <Select
          :input-id="id"
          v-model="createForm.cycle_weeks"
          :options="cycleOptions"
          option-label="label"
          option-value="value"
        />
      </AppField>
    </AppStack>
    <template #footer>
      <span class="sv-spacer" />
      <AppButton label="Отмена" :disabled="creating" @click="createOpen = false" />
      <AppButton variant="filled" label="Создать" :loading="creating" @click="doCreate" />
    </template>
  </AppDialog>

  <ScheduleItemDialog
    v-if="store.selected"
    v-model="itemOpen"
    :schedule="store.selected"
    :item="editingItem"
    :preset="itemPreset"
    :submit="saveItem"
    :remove="deleteItem"
  />

  <ScheduleSetupDialog
    v-if="store.selected && !store.readonly"
    v-model="setupOpen"
    :schedule="store.selected"
    :submit="saveSettings"
    :save-fields="store.saveFields"
    :add-category-fn="store.createCategory"
    :update-category-fn="store.updateCategory"
    :remove-category-fn="store.removeCategory"
    :remove-fn="deleteSchedule"
  />

  <ScheduleShareDialog
    v-if="store.selected && !store.readonly"
    v-model="shareOpen"
    :schedule="store.selected"
  />

  <SchedulePrintPreview
    v-if="store.selected"
    v-model="printOpen"
    :schedule="store.selected"
    :items="store.items"
    :monday="store.monday"
    :days="days"
  />

  <ContextMenu
    :visible="rowMenuOpen"
    :x="rowMenuX"
    :y="rowMenuY"
    :items="rowMenuItems"
    @select="onRowMenu"
    @close="rowMenuOpen = false"
  />

  <ConfirmDialog
    :visible="!!scheduleToDelete"
    header="Удалить расписание?"
    :message="`«${scheduleToDelete?.name || ''}» удалится вместе со всеми занятиями и категориями. Действие необратимо.`"
    confirm-label="Удалить"
    danger-confirm
    @confirm="doDeleteSchedule"
    @cancel="scheduleToDelete = null"
  />

  <!-- Импорт: файл выбирается системным диалогом, поэтому input скрытый. -->
  <input ref="fileInput" type="file" accept="application/json,.json" hidden @change="onImportFile" />
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppInlineEdit from '@/components/ui/AppInlineEdit.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppField from '@/components/ui/AppField.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ScheduleItemDialog from '@/components/schedule/ScheduleItemDialog.vue'
import SchedulePrintPreview from '@/components/schedule/SchedulePrintPreview.vue'
import ScheduleSetupDialog from '@/components/schedule/ScheduleSetupDialog.vue'
import ScheduleShareDialog from '@/components/schedule/ScheduleShareDialog.vue'
import ScheduleSummary from '@/components/schedule/ScheduleSummary.vue'
import ScheduleTimeline from '@/components/schedule/ScheduleTimeline.vue'
import ScheduleWeekNav from '@/components/schedule/ScheduleWeekNav.vue'
import { useSchedulesStore } from '@/stores/schedules.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { exportSchedule, importInto } from '@/api/schedules.js'
import { MAX_CYCLE_WEEKS, WEEKDAYS, addDays, dateKey, itemsOfDay, mondayOf } from '@/utils/scheduleCycle.js'
import { visibleDays } from '@/utils/scheduleLayout.js'
import { saveBlob } from '@/utils/download.js'

const route = useRoute()
const store = useSchedulesStore()
const notif = useNotificationsStore()

// Узкая раскладка — свойство самой панели (раздел живёт окном рабочего стола),
// поэтому её сообщает AppListDetail, а не медиазапрос по ширине экрана.
const narrow = ref(false)
const detailOpen = ref(false)

const createOpen = ref(false)
const creating = ref(false)
const createForm = ref({ name: '', cycle_weeks: 1 })

const itemOpen = ref(false)
const editingItem = ref(null)
const itemPreset = ref(null)
const setupOpen = ref(false)
const shareOpen = ref(false)
const printOpen = ref(false)
const fileInput = ref(null)

/* Контекстное меню строки списка: те же действия, что и в шапке открытого
   расписания, но над ЛЮБЫМ из списка — не открывая его ради переименования
   или выгрузки. */
const rowMenuOpen = ref(false)
const rowMenuX = ref(0)
const rowMenuY = ref(0)
const rowMenuTarget = ref(null)
const renamingId = ref(null)
const scheduleToDelete = ref(null)

const days = computed(() => visibleDays(store.items, todayWeekday.value))
// Длина цикла — выбор из готового набора: 1..MAX_CYCLE_WEEKS недель.
const cycleOptions = Array.from({ length: MAX_CYCLE_WEEKS }, (_, i) => ({ value: i + 1, label: String(i + 1) }))

/* Дни недели на телефоне — вкладки во всю ширину, поэтому подпись короткая:
   при равных долях самая длинная задаёт предел, и «Сегодня» с числами резалось
   бы у всех. Что день сегодняшний, говорит сводка под шкалой. */
const dayTabs = computed(() => days.value.map((d) => ({
  value: d,
  label: WEEKDAYS[d].short,
})))
const todayWeekday = computed(() => (new Date().getDay() + 6) % 7)

const weekRange = computed(() => {
  const from = store.monday
  const to = addDays(from, days.value[days.value.length - 1] ?? 6)
  return `${dayMonth(from)} — ${dayMonth(to)}`
})

/* На текущей ли мы неделе: диапазон служит кнопкой возврата, и подсветка
   говорит, что возвращаться есть куда. */
const onCurrentWeek = computed(() => dateKey(store.monday) === dateKey(mondayOf(new Date())))

const summaryItems = computed(() => {
  if (!store.selected) return []
  const weekday = narrow.value ? store.selectedWeekday : todayWeekday.value
  return itemsOfDay(store.selected, store.items, addDays(store.monday, weekday))
})

const summaryLabel = computed(() => {
  const weekday = narrow.value ? store.selectedWeekday : todayWeekday.value
  const name = WEEKDAYS[weekday]?.full || ''
  return isToday(weekday) ? `${name}, сегодня` : name
})

/* Порог окна живёт в меню «ещё» и набором готовых значений: настройка редкая
   (выставил раз под свой день), а в строке управления она занимала место
   постоянно. Набор вместо шага «плюс-минус» — по мелким кнопкам ещё и пальцем
   попадать неудобно. */
const GAP_STEPS = [10, 15, 20, 30, 45, 60]

const gapCommand = computed(() => ({
  key: 'gap',
  label: `Окно от ${store.gapMin} мин`,
  icon: 'timelapse',
  children: GAP_STEPS.map((n) => ({
    key: `gap:${n}`,
    label: `от ${n} мин`,
    icon: n === store.gapMin ? 'check' : 'timelapse',
  })),
}))

const commands = computed(() => {
  if (!store.selected) return []
  const own = !store.readonly
  return [
    ...(own ? [{ key: 'add', label: 'Занятие', icon: 'add', variant: 'filled', primary: true, fab: true }] : []),
    /* Конструктор — режим редактирования, а не постоянный инструмент: в шапке
       он занимал место у каждого, кто просто смотрит расписание. */
    ...(own
      ? [{
          key: 'editing',
          label: store.editing ? 'Выйти из конструктора' : 'Конструктор',
          icon: store.editing ? 'edit_off' : 'edit',
        }]
      : []),
    gapCommand.value,
    ...(own ? [{ key: 'setup', label: 'Настройки расписания', icon: 'tune' }] : []),
    ...(own ? [{ key: 'share', label: 'Поделиться', icon: 'share' }] : []),
    {
      key: 'download',
      label: 'Выгрузка',
      icon: 'download',
      children: [
        { key: 'download:xlsx', label: 'Таблица XLSX', icon: 'table' },
        { key: 'download:json', label: 'Файл переноса JSON', icon: 'data_object' },
        { key: 'print', label: 'Печать недели', icon: 'print' },
        ...(own ? [{ key: 'import', label: 'Загрузить из файла', icon: 'upload' }] : []),
      ],
    },
  ]
})

const rowMenuItems = computed(() => {
  const s = rowMenuTarget.value
  if (!s) return []
  // Чужое расписание открыто только на чтение (уровней доступа у раздела нет).
  const own = !s.shared
  return [
    { label: 'Переименовать', icon: 'edit', action: 'rename', disabled: !own },
    { label: 'Настройки расписания', icon: 'tune', action: 'setup', disabled: !own },
    { label: 'Поделиться', icon: 'share', action: 'share', disabled: !own },
    {
      label: 'Выгрузка',
      icon: 'download',
      children: [
        { label: 'Таблица XLSX', icon: 'table', action: 'xlsx' },
        { label: 'Файл переноса JSON', icon: 'data_object', action: 'json' },
        { label: 'Печать недели', icon: 'print', action: 'print' },
        ...(own ? [{ label: 'Загрузить из файла', icon: 'upload', action: 'import' }] : []),
      ],
    },
    { divider: true },
    { label: 'Удалить', icon: 'delete', danger: true, action: 'delete', disabled: !own },
  ]
})

function openRowMenu(s, e) {
  rowMenuTarget.value = s
  rowMenuX.value = e.clientX
  rowMenuY.value = e.clientY
  rowMenuOpen.value = true
}

/* Диалоги настроек, ссылок и печати работают с ОТКРЫТЫМ расписанием (им нужны
   его занятия), поэтому пункт меню сперва открывает своё. Переименование и
   выгрузка идут по id — открывать ради них чужой экран незачем. */
async function onRowMenu(action) {
  const s = rowMenuTarget.value
  rowMenuOpen.value = false
  if (!s) return
  switch (action) {
    case 'rename': renamingId.value = s.id; break
    case 'delete': scheduleToDelete.value = s; break
    case 'xlsx': download('xlsx', s); break
    case 'json': download('json', s); break
    case 'setup': await openSchedule(s.id); setupOpen.value = true; break
    case 'share': await openSchedule(s.id); shareOpen.value = true; break
    case 'print': await openSchedule(s.id); printOpen.value = true; break
    case 'import': await openSchedule(s.id); fileInput.value?.click(); break
  }
}

async function applyRename(s, name) {
  renamingId.value = null
  const next = name.trim()
  if (!next || next === s.name) return
  try {
    await store.updateSchedule(s.id, { name: next })
  } catch (e) {
    notif.error(e?.message || 'Ошибка сервера', 'Не переименовано')
  }
}

async function doDeleteSchedule() {
  const s = scheduleToDelete.value
  scheduleToDelete.value = null
  if (!s) return
  try {
    await store.removeSchedule(s.id)
    if (store.selectedId === null) detailOpen.value = false
  } catch (e) {
    notif.error(e?.message || 'Ошибка сервера', 'Не удалено')
  }
}

function scheduleHint(s) {
  const parts = [`${s.item_count || 0} занятий`]
  if (s.cycle_weeks > 1) parts.push(`цикл ${s.cycle_weeks} нед.`)
  if (s.shared && s.owner_name) parts.push(s.owner_name)
  return parts.join(' · ')
}

function dayMonth(date) {
  const d = addDays(date, 0)
  return `${String(d.getUTCDate()).padStart(2, '0')}.${String(d.getUTCMonth() + 1).padStart(2, '0')}`
}

function dayNumber(weekday) { return addDays(store.monday, weekday).getUTCDate() }
function isToday(weekday) { return dateKey(addDays(store.monday, weekday)) === dateKey(new Date()) }

async function openSchedule(id) {
  detailOpen.value = true
  await store.select(id)
}

function openItem(item) {
  if (store.readonly) return
  editingItem.value = item
  itemPreset.value = null
  itemOpen.value = true
}

function startItem(preset) {
  editingItem.value = null
  itemPreset.value = preset
  itemOpen.value = true
}

async function saveItem(body) {
  if (editingItem.value) await store.updateItem(editingItem.value.id, body)
  else await store.createItem(body)
}

async function deleteItem(item) {
  await store.removeItem(item.id)
}

async function saveSettings(body) {
  const data = await store.updateSchedule(store.selectedId, body)
  // Сервер честно говорит, скольким занятиям укороченный цикл стёр недели —
  // молча менять чужие занятия нельзя.
  if (data.adjusted_items > 0) {
    notif.warn(`Занятий переведено на «каждую неделю»: ${data.adjusted_items}`, 'Цикл изменён')
  }
}

async function deleteSchedule() {
  await store.removeSchedule(store.selectedId)
  detailOpen.value = false
}

async function doCreate() {
  const name = createForm.value.name.trim()
  if (!name) return
  creating.value = true
  try {
    await store.createSchedule({ name, cycle_weeks: createForm.value.cycle_weeks })
    createOpen.value = false
    createForm.value = { name: '', cycle_weeks: 1 }
    detailOpen.value = true
  } catch (e) {
    notif.error(e?.message || 'Ошибка сервера', 'Не создано')
  } finally {
    creating.value = false
  }
}

async function download(format, target = null) {
  const schedule = target || store.selected
  try {
    const resp = await exportSchedule(schedule.id, format)
    await saveBlob(await resp.blob(), `${schedule.name}.${format}`)
  } catch (e) {
    notif.error(e?.message || 'Ошибка сервера', 'Не выгружено')
  }
}

async function onImportFile(event) {
  const file = event.target.files?.[0]
  event.target.value = ''
  if (!file) return
  try {
    store.applyView(await importInto(store.selectedId, file))
    notif.success('Расписание загружено')
  } catch (e) {
    notif.error(e?.message || 'Файл не похож на расписание', 'Не загружено')
  }
}

function onCommand(key) {
  if (typeof key === 'string' && key.startsWith('gap:')) {
    store.setGap(Number(key.slice(4)))
    return
  }
  switch (key) {
    case 'add': startItem({ weekday: narrow.value ? store.selectedWeekday : todayWeekday.value }); break
    case 'editing': store.editing = !store.editing; break
    case 'setup': setupOpen.value = true; break
    case 'share': shareOpen.value = true; break
    case 'download:xlsx': download('xlsx'); break
    case 'download:json': download('json'); break
    case 'print': printOpen.value = true; break
    case 'import': fileInput.value?.click(); break
  }
}

onMounted(async () => {
  await store.fetchSchedules()
  const id = Number(route.query.id)
  if (id && store.schedules.some((s) => s.id === id)) await openSchedule(id)
  else if (!narrow.value && store.schedules.length) await openSchedule(store.schedules[0].id)
})

// Deep-link из поиска Hola и «Моей активности»: ?id=… открывает расписание.
watch(() => route.query.id, async (value) => {
  const id = Number(value)
  if (id && id !== store.selectedId) await openSchedule(id)
})
</script>

<style scoped>
.sv-tabs { flex: 1 1 100%; }

.sv-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
}
/* Полоса дней не сжимается и не растягивается: место в колонке принадлежит
   шкале, а прижатая к ней вплотную полоса читалась как её часть. */
/* Полоса дней живёт в КОЛОНКЕ тела, а не в строке управления: доля строки
   (flex-basis) стала бы здесь высотой и раздула бы пилюлю на пол-экрана.
   Ширину даёт full-width самого AppTabs. */
.sv-days { flex: none; }


.sv-spacer { flex: 1; }
</style>
