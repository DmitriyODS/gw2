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
          <AppTabs
            :model-value="store.tab"
            :tabs="[
              { value: 'mine', label: 'Мои' },
              { value: 'shared', label: 'Поделились' },
            ]"
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
          <AppRow
            v-for="s in store.schedules"
            :key="s.id"
            :title="s.name"
            :hint="scheduleHint(s)"
            icon="calendar_view_week"
            dense
            clickable
            :selected="s.id === store.selectedId"
            @click="openSchedule(s.id)"
          />
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
        <template v-if="store.selected" #subhead="{ narrow: tight }">
          <div class="sv-bar">
            <div class="sv-nav">
              <AppButton variant="icon" icon="chevron_left" label="Предыдущая неделя" @click="store.stepWeek(-1)" />
              <AppButton variant="text" :label="weekRange" title="Вернуться к текущей неделе" @click="store.today()" />
              <AppChip v-if="store.selected.cycle_weeks > 1" tone="primary" :label="store.cycleWeekLabel" />
              <AppButton variant="icon" icon="chevron_right" label="Следующая неделя" @click="store.stepWeek(1)" />
            </div>

            <!-- Порог окна крутится прямо здесь: подобрать его на глаз проще,
                 чем угадать в настройках. Значение личное — гость по ссылке
                 чужой порог менять не может. -->
            <div
              class="sv-gap"
              title="Промежуток, начиная с которого он считается окном"
              @wheel.prevent="onGapWheel"
            >
              <AppButton variant="icon" icon="remove" label="Короче окно" @click="store.setGap(store.gapMin - 5)" />
              <span class="sv-gap-value">окно от {{ store.gapMin }} мин</span>
              <AppButton variant="icon" icon="add" label="Длиннее окно" @click="store.setGap(store.gapMin + 5)" />
            </div>

            <AppSwitch
              v-if="!store.readonly && !tight"
              :model-value="store.editing"
              label="Конструктор"
              @update:model-value="store.editing = $event"
            />
          </div>
        </template>

        <template v-if="store.selected">
          <BrandLoader v-if="store.loadingItems" size="64" class="sv-loader" />
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

  <!-- Импорт: файл выбирается системным диалогом, поэтому input скрытый. -->
  <input ref="fileInput" type="file" accept="application/json,.json" hidden @change="onImportFile" />
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import InputText from 'primevue/inputtext'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppField from '@/components/ui/AppField.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitch from '@/components/ui/AppSwitch.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ScheduleItemDialog from '@/components/schedule/ScheduleItemDialog.vue'
import SchedulePrintPreview from '@/components/schedule/SchedulePrintPreview.vue'
import ScheduleSetupDialog from '@/components/schedule/ScheduleSetupDialog.vue'
import ScheduleShareDialog from '@/components/schedule/ScheduleShareDialog.vue'
import ScheduleSummary from '@/components/schedule/ScheduleSummary.vue'
import ScheduleTimeline from '@/components/schedule/ScheduleTimeline.vue'
import { useSchedulesStore } from '@/stores/schedules.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { exportSchedule, importInto } from '@/api/schedules.js'
import { MAX_CYCLE_WEEKS, WEEKDAYS, addDays, dateKey, itemsOfDay } from '@/utils/scheduleCycle.js'
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

const days = computed(() => visibleDays(store.items, todayWeekday.value))
// Длина цикла — выбор из готового набора: 1..MAX_CYCLE_WEEKS недель.
const cycleOptions = Array.from({ length: MAX_CYCLE_WEEKS }, (_, i) => ({ value: i + 1, label: String(i + 1) }))

// Дни недели на телефоне — вкладки. Сегодняшний подписан словом: значок-
// счётчик тут читался бы как «непрочитанное», а день и так один.
const dayTabs = computed(() => days.value.map((d) => ({
  value: d,
  label: isToday(d) ? 'Сегодня' : `${WEEKDAYS[d].short} ${dayNumber(d)}`,
})))
const todayWeekday = computed(() => (new Date().getDay() + 6) % 7)

const weekRange = computed(() => {
  const from = store.monday
  const to = addDays(from, days.value[days.value.length - 1] ?? 6)
  return `${dayMonth(from)} — ${dayMonth(to)}`
})

const summaryItems = computed(() => {
  if (!store.selected) return []
  const weekday = narrow.value ? store.selectedWeekday : todayWeekday.value
  return itemsOfDay(store.selected, store.items, addDays(store.monday, weekday))
})

const summaryLabel = computed(() => {
  const weekday = narrow.value ? store.selectedWeekday : todayWeekday.value
  return WEEKDAYS[weekday]?.full || ''
})

const commands = computed(() => {
  if (!store.selected) return []
  const own = !store.readonly
  return [
    ...(own ? [{ key: 'add', label: 'Занятие', icon: 'add', variant: 'filled', primary: true, fab: true }] : []),
    ...(own && narrow.value
      ? [{ key: 'editing', label: store.editing ? 'Выйти из конструктора' : 'Конструктор', icon: 'edit' }]
      : []),
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

async function download(format) {
  try {
    const resp = await exportSchedule(store.selectedId, format)
    await saveBlob(await resp.blob(), `${store.selected.name}.${format}`)
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

function onGapWheel(event) {
  store.setGap(store.gapMin + (event.deltaY < 0 ? 5 : -5))
}

function onCommand(key) {
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
.sv-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.sv-nav { display: flex; align-items: center; gap: 2px; }

.sv-gap {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 2px 4px;
  border-radius: var(--radius-full);
  background: var(--color-surface-low);
}
.sv-gap-value {
  font-size: 12px;
  color: var(--color-text-dim);
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}

.sv-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
  height: 100%;
  min-height: 0;
}
/* Полоса дней не сжимается и не растягивается: место в колонке принадлежит
   шкале, а прижатая к ней вплотную полоса читалась как её часть. */
.sv-days { flex: none; }
.sv-loader { margin: auto; }


.sv-spacer { flex: 1; }
</style>
