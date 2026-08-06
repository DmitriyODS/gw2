<template>
  <AppListDetail
    v-model:open="detailOpen"
    :loading="store.loadingList && !store.diaries.length"
    @narrow-change="narrow = $event"
  >
    <!-- Список ежедневников -->
    <template #list="{ toggle }">
      <AppPage
        embedded
        title="Ежедневники"
        show-title
        :menu="!narrow"
        menu-icon="left_panel_close"
        menu-label="Свернуть список"
        :commands="listCommands"
        @menu="toggle"
        @command="openCreateDiary"
      >
        <template #subhead>
          <AppTabs :model-value="store.tab" :tabs="tabs" full-width dense @update:model-value="store.setTab" />
        </template>

        <EmptyState
          v-if="!store.diaries.length"
          size="sm"
          icon="book"
          :title="store.tab === 'shared' ? 'С вами пока не делились' : 'Ежедневников нет'"
          :subtitle="store.tab === 'shared' ? 'Здесь появятся ежедневники, которыми с вами поделились.' : 'Создайте первый — он появится в этом списке.'"
        />
        <AppStack v-else :gap="6">
          <AppRow
            v-for="d in store.diaries"
            :key="d.id"
            :title="d.name"
            :icon="store.tab === 'shared' ? 'folder_shared' : 'book'"
            dense
            clickable
            :selected="d.id === store.selectedId"
            :tone="dropDiaryId === d.id ? 'primary' : 'neutral'"
            @click="selectDiary(d.id)"
            @dragover="onDiaryDragOver($event, d)"
            @dragleave="dropDiaryId === d.id && (dropDiaryId = null)"
            @drop="onDiaryDrop($event, d)"
          >
            <template v-if="store.tab === 'shared' || diaryTotal(d)" #hint>
              <span v-if="store.tab === 'shared'">{{ d.owner_name }}</span>
              <span v-if="diaryTotal(d)" class="dv-side-progress">
                <span class="dv-side-bar"><span class="dv-side-fill" :style="{ width: diaryPct(d) + '%' }" /></span>
                <span class="dv-side-count">{{ d.done_count || 0 }}/{{ diaryTotal(d) }}</span>
              </span>
            </template>
          </AppRow>
        </AppStack>
      </AppPage>
    </template>

    <!-- Выбранный ежедневник -->
    <template #detail="{ collapsed, toggle }">
      <AppPage
        embedded
        :title="store.selected?.name || ''"
        :back="narrow"
        back-label="К ежедневникам"
        :menu="!narrow && collapsed"
        menu-icon="left_panel_open"
        menu-label="Показать список"
        :commands="commands"
        flush
        :scroll="false"
        @back="detailOpen = false"
        @menu="toggle"
        @command="onCommand"
      >
        <!-- Поиск — в строку названия: в тесной панели он сворачивается в лупу
             и не отнимает у списка дел целую строку. -->
        <template v-if="store.selected" #search="{ narrow: tight }">
          <SearchField
            v-model="searchInput"
            placeholder="Поиск по записям…"
            hotkey
            :collapsed="tight"
            @update:model-value="onSearch"
            @clear="clearSearch"
          />
        </template>

        <!-- В тесной панели строка управления остаётся только под навигацию по
             дням: набор записей и вид ушли в меню «ещё». -->
        <template
          v-if="store.selected && (!narrow || store.subtab === 'active')"
          #subhead="{ narrow: tight }"
        >
          <AppTabs
            v-if="!tight"
            :model-value="store.subtab"
            :tabs="subtabs"
            dense
            @update:model-value="store.setSubtab"
          />

          <PeriodNav
            v-if="store.subtab === 'active'"
            :label="periodLabel"
            :view="store.view"
            :tight="tight"
            @step="store.step($event)"
            @today="store.today()"
            @update:view="store.setView($event)"
          />
        </template>

        <template v-if="store.selected">
        <div class="dv-body">
          <!-- АРХИВ — выполненные, сгруппированные по дням -->
          <div v-if="store.subtab === 'archive'" class="dv-archive">
            <EmptyState
              v-if="!store.archive.length"
              icon="inventory_2"
              title="Архив пуст"
              subtitle="Выполненные записи появятся здесь"
              size="sm"
            />
            <div v-for="g in archiveGroups" :key="g.date" class="dv-arc-group">
              <div class="dv-arc-daylabel">{{ g.label }}</div>
              <button v-for="e in g.items" :key="e.id" class="dv-arow" @click="openEntry(e)">
                <span class="material-symbols-outlined dv-arow-check">check_circle</span>
                <span class="dv-arow-body">
                  <span class="dv-arow-title">{{ e.title }}</span>
                  <span v-if="entryTime(e)" class="dv-arow-meta">{{ entryTime(e) }}</span>
                </span>
                <span v-if="store.canToggle" class="dv-arow-act" title="Вернуть в активные" @click.stop="toggleDone(e, false)">
                  <span class="material-symbols-outlined">undo</span>
                </span>
                <span class="material-symbols-outlined dv-arow-chev">chevron_right</span>
              </button>
            </div>
          </div>

          <!-- ВСЕ ЗАДАЧИ — все активные записи по всем дням единым списком -->
          <div v-else-if="store.subtab === 'all'" class="dv-all">
            <EmptyState
              v-if="!store.entries.length"
              icon="checklist"
              title="Активных записей нет"
              size="sm"
            />
            <div v-for="g in allGroups" :key="g.date" class="dv-all-group">
              <div class="dv-arc-daylabel">{{ g.label }}</div>
              <button
                v-for="e in g.items" :key="e.id" class="dv-dayrow"
                :class="{ dragging: dragEntryId === e.id }"
                :draggable="canDrag" @dragstart="onDragStart($event, e)" @dragend="onDragEnd"
                @click="openEntry(e)"
              >
                <span class="dv-dayrow-time">{{ entryTime(e) || '—' }}</span>
                <span class="dv-dayrow-body">
                  <span class="dv-dayrow-title">{{ e.title }}</span>
                  <span v-if="e.description" class="dv-dayrow-sub">{{ e.description }}</span>
                </span>
                <span v-if="store.canToggle" class="dv-dayrow-done" title="Выполнено" @click.stop="toggleDone(e, true)">
                  <span class="material-symbols-outlined">check_circle</span>
                </span>
                <span class="material-symbols-outlined dv-dayrow-chev">chevron_right</span>
              </button>
            </div>
          </div>

          <!-- АКТИВНЫЕ — календарные виды -->
          <template v-else>
            <div v-if="!narrow && store.view !== 'day'" ref="weekGridRef" class="dv-grid" :class="store.view">
              <template v-if="store.view === 'month'">
                <div v-for="(wd, i) in weekdays" :key="'h' + i" class="dv-wd">{{ wd }}</div>
              </template>
              <div
                v-for="day in gridDays" :key="dayKey(day)"
                class="dv-day glass-hover" :class="{ dim: store.view === 'month' && !inCurrentMonth(day), today: isToday(day), 'drop-target': dropDayKey === dayKey(day) }"
                @click="openDay(day)"
                @dragover="onDayDragOver($event, day)"
                @dragleave="dropDayKey === dayKey(day) && (dropDayKey = null)"
                @drop="onDayDrop($event, day)"
              >
                <div class="dv-day-head">
                  <span class="dv-day-num">{{ day.getDate() }}</span>
                  <span v-if="store.view === 'week'" class="dv-day-wd">{{ weekdayShort(day) }}</span>
                  <span v-if="dayEntries(day).length" class="dv-day-count">{{ dayEntries(day).length }}</span>
                </div>
                <div class="dv-day-events">
                  <div
                    v-for="e in dayPreview(day)" :key="e.id" class="dv-event"
                    :class="{ dragging: dragEntryId === e.id }"
                    :draggable="canDrag" @dragstart="onDragStart($event, e)" @dragend="onDragEnd"
                  >
                    <span v-if="entryTime(e)" class="dv-event-time">{{ entryTime(e) }}</span>
                    <span class="dv-event-title">{{ e.title }}</span>
                  </div>
                  <div v-if="dayEntries(day).length > dayPreview(day).length" class="dv-event-more">+{{ dayEntries(day).length - dayPreview(day).length }}</div>
                </div>
              </div>
            </div>

            <div v-else-if="narrow && store.view !== 'day'" class="dv-agenda">
              <button v-for="day in agendaDays" :key="dayKey(day)" class="dv-agenda-row" :class="{ today: isToday(day) }" @click="openDay(day)">
                <div class="dv-agenda-date">
                  <span class="dv-agenda-dnum">{{ day.getDate() }}</span>
                  <span class="dv-agenda-dwd">{{ weekdayShort(day) }}</span>
                </div>
                <div class="dv-agenda-body">
                  <span class="dv-agenda-month">{{ agendaMonth(day) }}</span>
                  <span v-if="dayEntries(day).length" class="dv-agenda-prev">{{ agendaPreview(day) }}</span>
                  <span v-else class="dv-agenda-empty">Нет записей</span>
                </div>
                <span v-if="dayEntries(day).length" class="dv-day-count">{{ dayEntries(day).length }}</span>
                <span class="material-symbols-outlined dv-agenda-chev">chevron_right</span>
              </button>
            </div>

            <div v-else class="dv-daylist">
              <EmptyState
                v-if="!dayEntries(store.cursor).length && !store.dayDone.length"
                icon="event_busy"
                title="На этот день записей нет"
                size="sm"
              >
                <AppButton
                  v-if="!store.readonly"
                  variant="filled"
                  icon="add"
                  label="Добавить запись"
                  @click="openCreate(store.cursor)"
                />
              </EmptyState>
              <template v-else>
                <template v-if="dayEntries(store.cursor).length">
                  <div class="dv-day-section">Активные</div>
                  <button
                    v-for="(e, i) in dayEntries(store.cursor)" :key="e.id" class="dv-dayrow"
                    :class="{ dragging: dragEntryId === e.id }"
                    :draggable="canDrag" @dragstart="onDragStart($event, e)" @dragend="onDragEnd"
                    @click="openEntry(e)"
                  >
                    <span class="dv-dayrow-num">{{ i + 1 }}</span>
                    <span class="dv-dayrow-time">{{ entryTime(e) || '—' }}</span>
                    <span class="dv-dayrow-body">
                      <span class="dv-dayrow-title">{{ e.title }}</span>
                      <span v-if="e.description" class="dv-dayrow-sub">{{ e.description }}</span>
                    </span>
                    <span v-if="store.canToggle" class="dv-dayrow-done" title="Выполнено" @click.stop="toggleDone(e, true)">
                      <span class="material-symbols-outlined">check_circle</span>
                    </span>
                    <span class="material-symbols-outlined dv-dayrow-chev">chevron_right</span>
                  </button>
                </template>
                <template v-if="store.dayDone.length">
                  <div class="dv-day-section">Выполнено</div>
                  <button v-for="(e, i) in store.dayDone" :key="e.id" class="dv-dayrow" @click="openEntry(e)">
                    <span class="dv-dayrow-num">{{ i + 1 }}</span>
                    <span class="dv-dayrow-time">{{ entryTime(e) || '—' }}</span>
                    <span class="dv-dayrow-body">
                      <span class="dv-dayrow-title done">{{ e.title }}</span>
                      <span v-if="e.description" class="dv-dayrow-sub">{{ e.description }}</span>
                    </span>
                    <span v-if="store.canToggle" class="dv-dayrow-done undo" title="Вернуть в активные" @click.stop="toggleDone(e, false)">
                      <span class="material-symbols-outlined">undo</span>
                    </span>
                    <span class="material-symbols-outlined dv-dayrow-chev">chevron_right</span>
                  </button>
                </template>
              </template>
            </div>
          </template>

          <div v-if="store.loadingEntries" class="dv-overlay"><BrandLoader :size="48" /></div>
        </div>
        </template>

        <!-- Ежедневник не выбран (широкая раскладка) -->
        <EmptyState
          v-else
          icon="event_note"
          tone="soft"
          :title="store.diaries.length ? 'Выберите ежедневник слева' : 'Создайте свой первый ежедневник'"
          :subtitle="store.diaries.length
            ? 'Выберите ежедневник в списке, чтобы посмотреть записи'
            : 'Планируйте дела по дням и отмечайте выполненное'"
        >
          <AppButton
            v-if="store.tab === 'mine' && !store.diaries.length"
            variant="filled"
            icon="add"
            label="Новый ежедневник"
            @click="openCreateDiary"
          />
        </EmptyState>
      </AppPage>
    </template>

    <!-- Диалог дня -->
    <AppDialog v-model="dayOpen" :title="dayTitle" size="md" :actions="dayActions" @cancel="dayOpen = false" @confirm="openCreate(dayDate)">
      <div class="dd">
        <p v-if="!dayActive.length && !dayDone.length" class="dd-empty">На этот день записей нет.</p>

        <div v-if="dayActive.length" class="dd-group">
          <span class="dd-grouplabel">Активные</span>
          <ul class="dd-list">
            <li
              v-for="(e, i) in dayOrdered" :key="e.id" class="dd-row"
              :class="{ dragging: ddDragId === e.id }"
              :draggable="canDrag && dayOrdered.length > 1"
              @dragstart="ddDragStart($event, e)" @dragend="ddDragEnd"
              @dragover="ddDragOver($event, e)" @drop.prevent="ddDrop"
            >
              <!-- Номер — позиция в списке, а не свойство записи: перестановка
                   оставляет нумерацию на месте. -->
              <span class="dd-num">{{ i + 1 }}</span>
              <span v-if="canDrag && dayOrdered.length > 1" class="dd-grip" title="Перетащите, чтобы изменить порядок">
                <span class="material-symbols-outlined">drag_indicator</span>
              </span>
              <button v-if="store.canToggle" class="dd-check" title="Выполнено" @click="dayToggle(e, true)"><span class="material-symbols-outlined">radio_button_unchecked</span></button>
              <button class="dd-main" @click="openEntry(e)">
                <span v-if="entryTime(e)" class="dd-time">{{ entryTime(e) }}</span>
                <span class="dd-title">{{ e.title }}</span>
                <span class="material-symbols-outlined dd-chev">chevron_right</span>
              </button>
            </li>
          </ul>
        </div>

        <div v-if="dayDone.length" class="dd-group">
          <span class="dd-grouplabel">Выполнено</span>
          <ul class="dd-list">
            <li v-for="(e, i) in dayDone" :key="e.id" class="dd-row">
              <span class="dd-num">{{ i + 1 }}</span>
              <button v-if="store.canToggle" class="dd-check done" title="Вернуть в активные" @click="dayToggle(e, false)"><span class="material-symbols-outlined">check_circle</span></button>
              <button class="dd-main" @click="openEntry(e)">
                <span v-if="entryTime(e)" class="dd-time">{{ entryTime(e) }}</span>
                <span class="dd-title done">{{ e.title }}</span>
                <span class="material-symbols-outlined dd-chev">chevron_right</span>
              </button>
            </li>
          </ul>
        </div>
      </div>
    </AppDialog>

    <DiaryEntryDialog
      v-model="entryOpen"
      :entry="activeEntry"
      :readonly="store.readonly"
      :can-toggle="store.canToggle"
      :default-date="defaultDate"
      @create-task="onCreateTask"
    />

    <DiaryShareDialog v-model="shareOpen" :diary-id="store.selectedId" />

    <!-- Создание/переименование ежедневника -->
    <AppDialog
      v-model="nameOpen"
      :title="nameMode === 'create' ? 'Новый ежедневник' : 'Переименовать'" size="sm" :busy="nameBusy"
      :actions="[{ kind: 'cancel', label: 'Отмена' }, { kind: 'confirm', label: 'Сохранить' }]"
      @cancel="nameOpen = false" @confirm="saveName"
    >
      <input ref="nameInput" v-model="nameValue" class="dv-name-input" type="text" placeholder="Например, Личные дела" maxlength="120" @keydown.enter="saveName" />
    </AppDialog>

    <ConfirmDialog
      :visible="confirmDeleteDiary"
      header="Удалить ежедневник?"
      message="Ежедневник и все его записи будут удалены безвозвратно."
      confirm-label="Удалить" danger-confirm
      @confirm="doDeleteDiary" @cancel="confirmDeleteDiary = false"
    />

    <!-- Создание задачи с юнитом из записи -->
    <TaskForm v-if="taskFormEntry" :preset-name="taskFormEntry.title" @close="taskFormEntry = null" @saved="onTaskSaved" />
  </AppListDetail>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import AppDialog from '@/components/ui/AppDialog.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppListDetail from '@/components/ui/AppListDetail.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import PeriodNav from '@/components/common/PeriodNav.vue'
import { periodViewCommand, parseViewCommand } from '@/utils/periodViews.js'
import SearchField from '@/components/common/SearchField.vue'
import DiaryEntryDialog from '@/components/diary/DiaryEntryDialog.vue'
import DiaryShareDialog from '@/components/diary/DiaryShareDialog.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import { useDiariesStore, dayKey } from '@/stores/diaries.js'
import { useAuthStore } from '@/stores/auth.js'
import { exportEntries, getEntries } from '@/api/diaries.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const store = useDiariesStore()
const route = useRoute()
const authStore = useAuthStore()
const notif = useNotificationsStore()
/* Узкая раскладка — свойство самой панели (раздел живёт окном рабочего стола),
   поэтому её сообщает AppListDetail, а не медиазапрос по ширине экрана. */
const narrow = ref(false)
const detailOpen = ref(false)

function selectDiary(id) {
  store.select(id)
  detailOpen.value = true
}

const listCommands = computed(() => (store.tab === 'mine'
  ? [{ key: 'new-diary', label: 'Ежедневник', icon: 'add', variant: 'filled', primary: true }]
  : []))

/* Команды шапки: создание записи — главное действие (в тесной панели уезжает на
   плавающую кнопку), управление ежедневником — в меню «ещё». */
const commands = computed(() => {
  if (!store.selected) return []
  const own = !store.readonly
  return [
    ...(own ? [{ key: 'add', label: 'Запись', icon: 'add', variant: 'filled', primary: true, fab: true }] : []),
    /* Тесная панель: набор записей и вид периода уезжают в меню — две строки
       вкладок стоили дороже, чем сам список дел. */
    ...(narrow.value ? [subtabCommand.value] : []),
    ...(narrow.value && store.subtab === 'active' ? [periodViewCommand(store.view)] : []),
    ...(own ? [{ key: 'rename', label: 'Переименовать', icon: 'edit' }] : []),
    ...(own ? [{ key: 'share', label: 'Поделиться', icon: 'share' }] : []),
    { key: 'export', label: 'Экспорт в XLSX', icon: 'download' },
    ...(own ? [{ key: 'delete', label: 'Удалить ежедневник', icon: 'delete', tone: 'danger' }] : []),
  ]
})

function onCommand(key) {
  const view = parseViewCommand(key)
  if (view) return store.setView(view)
  if (key.startsWith('subtab:')) return store.setSubtab(key.slice(7))
  if (key === 'add') openCreate()
  else if (key === 'rename') openRenameDiary()
  else if (key === 'share') shareOpen.value = true
  else if (key === 'export') doExport()
  else if (key === 'delete') confirmDeleteDiary.value = true
}

// Ежедневники личные (кросс-компанийные), но привязанные задачи скоупятся
// активной компанией — при её смене освежаем список и открытые записи.
watch(() => authStore.companyId, (id, prev) => {
  if (id === prev) return
  store.fetchDiaries()
  if (store.selectedId != null) store.fetchEntries({ silent: true })
})
// Мобильный FAB «Добавить запись»: прячется/появляется по прокрутке.

const tabs = [
  { value: 'mine', label: 'Мои', icon: 'book' },
  { value: 'shared', label: 'Поделились', icon: 'folder_shared' },
]
const subtabs = [
  { value: 'active', label: 'Активные', icon: 'checklist' },
  { value: 'all', label: 'Все задачи', icon: 'list' },
  { value: 'archive', label: 'Архив', icon: 'inventory_2' },
]

// То же самое пунктом меню — для тесной панели (см. commands).
const subtabCommand = computed(() => {
  const active = subtabs.find((t) => t.value === store.subtab) || subtabs[0]
  return {
    key: 'subtab',
    label: `Записи: ${active.label.toLowerCase()}`,
    icon: active.icon,
    children: subtabs.map((t) => ({
      key: `subtab:${t.value}`,
      label: t.label,
      icon: t.value === store.subtab ? 'check' : t.icon,
    })),
  }
})
const weekdays = ['Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб', 'Вс']

function addDays(d, n) { const x = new Date(d); x.setHours(0, 0, 0, 0); x.setDate(x.getDate() + n); return x }
const pad = (n) => String(n).padStart(2, '0')
function entryTime(e) {
  if (e.start_min == null) return ''
  const s = `${pad(Math.floor(e.start_min / 60))}:${pad(e.start_min % 60)}`
  if (e.end_min == null) return s
  return `${s}–${pad(Math.floor(e.end_min / 60))}:${pad(e.end_min % 60)}`
}

const gridDays = computed(() => {
  const { from, to } = store.range
  const n = Math.round((to.getTime() - from.getTime()) / 86400000)
  const start = new Date(from); start.setHours(0, 0, 0, 0)
  return Array.from({ length: n }, (_, i) => addDays(start, i))
})
function dayEntries(day) { return store.entriesByDay[dayKey(day)] || [] }
function inCurrentMonth(day) { return day.getMonth() === store.cursor.getMonth() }
function weekdayShort(day) { return weekdays[(day.getDay() + 6) % 7] }

// Прогресс ежедневника (выполнено/всего) в боковом списке.
function diaryTotal(d) { return (d.active_count || 0) + (d.done_count || 0) }
function diaryPct(d) { const t = diaryTotal(d); return t ? Math.round(((d.done_count || 0) / t) * 100) : 0 }

// ── Drag-and-drop: перенос записи на другой день (плитки сетки) или в другой
// свой ежедневник (боковой список). Перенос доступен только владельцу.
const dragEntryId = ref(null)
const dropDayKey = ref(null)
const dropDiaryId = ref(null)
const canDrag = computed(() => !store.readonly)

function onDragStart(ev, e) {
  dragEntryId.value = e.id
  ev.dataTransfer.effectAllowed = 'move'
  ev.dataTransfer.setData('text/plain', String(e.id))
}
function onDragEnd() { dragEntryId.value = null; dropDayKey.value = null; dropDiaryId.value = null }

function onDayDragOver(ev, day) {
  if (dragEntryId.value == null) return
  ev.preventDefault()
  ev.dataTransfer.dropEffect = 'move'
  dropDayKey.value = dayKey(day)
}
async function onDayDrop(ev, day) {
  if (dragEntryId.value == null) return
  ev.preventDefault()
  ev.stopPropagation()
  const id = dragEntryId.value
  onDragEnd()
  try { await store.moveEntry(id, { entryDate: dayKey(day) }) }
  catch (e) { notif.error(e?.message || 'Не удалось перенести запись') }
}

function onDiaryDragOver(ev, d) {
  if (dragEntryId.value == null || store.tab !== 'mine' || d.id === store.selectedId) return
  ev.preventDefault()
  ev.dataTransfer.dropEffect = 'move'
  dropDiaryId.value = d.id
}
async function onDiaryDrop(ev, d) {
  if (dragEntryId.value == null || d.id === store.selectedId) return
  ev.preventDefault()
  const id = dragEntryId.value
  onDragEnd()
  try {
    await store.moveEntry(id, { diaryId: d.id })
    notif.success(`Запись перенесена в «${d.name}»`)
  } catch (e) { notif.error(e?.message || 'Не удалось перенести запись') }
}

// Превью записей в плитке. Месяц — тесный (2). Неделя — столько, сколько влезает
// в высоту столбца; при переполнении одна строка уходит под «+N».
const EVENT_H = 22   // .dv-grid.week .dv-event height (var --dv-event-h)
const EVENT_GAP = 3  // .dv-day-events gap
const weekGridRef = ref(null)
const weekColEventsH = ref(0)
let weekRO = null

function measureWeekColumn() {
  const el = weekGridRef.value
  if (!el || store.view !== 'week') return
  // Неделя — один ряд (grid-template-rows: 1fr), высота сетки = высота столбца.
  // Вычитаем паддинги плитки (6×2), gap шапка→события (4) и высоту шапки (24).
  weekColEventsH.value = Math.max(0, el.clientHeight - 12 - 4 - 24)
}
function weekMaxVisible() {
  const h = weekColEventsH.value
  if (h <= 0) return 4 // фолбэк до первого замера
  return Math.max(1, Math.floor((h + EVENT_GAP) / (EVENT_H + EVENT_GAP)))
}
function dayPreview(day) {
  const entries = dayEntries(day)
  if (store.view !== 'week') return entries.slice(0, 2)
  const max = weekMaxVisible()
  if (entries.length <= max) return entries
  return entries.slice(0, Math.max(0, max - 1))
}

const agendaDays = computed(() => {
  if (store.view === 'week') return gridDays.value
  const c = store.cursor
  const days = new Date(c.getFullYear(), c.getMonth() + 1, 0).getDate()
  return Array.from({ length: days }, (_, i) => new Date(c.getFullYear(), c.getMonth(), i + 1))
})
function agendaMonth(day) { return day.toLocaleDateString('ru-RU', { month: 'short' }) }
function agendaPreview(day) {
  return dayEntries(day).slice(0, 2).map((e) => `${entryTime(e)} ${e.title}`.trim()).join(' · ')
}

const todayKey = dayKey(new Date())
function isToday(day) { return dayKey(day) === todayKey }

// Архив сгруппирован по дням (store.archive уже отсортирован по дате убыв.).
const archiveGroups = computed(() => {
  const map = new Map()
  for (const e of store.archive) {
    if (!map.has(e.entry_date)) map.set(e.entry_date, [])
    map.get(e.entry_date).push(e)
  }
  return [...map.entries()].map(([date, items]) => ({ date, label: archiveDayLabel(date), items }))
})
function archiveDayLabel(d) {
  const [y, m, day] = d.split('-').map(Number)
  const s = new Date(y, m - 1, day).toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  return s.charAt(0).toUpperCase() + s.slice(1)
}

// «Все задачи» — все активные записи по всем дням, сгруппированы по дню
// (store.entries отсортирован бэкендом по дате возр.).
const allGroups = computed(() => {
  const map = new Map()
  for (const e of store.entries) {
    if (!map.has(e.entry_date)) map.set(e.entry_date, [])
    map.get(e.entry_date).push(e)
  }
  return [...map.entries()].map(([date, items]) => ({ date, label: archiveDayLabel(date), items }))
})

const periodLabel = computed(() => {
  const c = store.cursor
  if (store.view === 'day') return c.toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
  if (store.view === 'week') {
    const { from } = store.range
    const start = new Date(from); const end = addDays(start, 6)
    const opts = { day: 'numeric', month: 'short' }
    return `${start.toLocaleDateString('ru-RU', opts)} – ${end.toLocaleDateString('ru-RU', opts)} ${end.getFullYear()}`
  }
  return c.toLocaleDateString('ru-RU', { month: 'long', year: 'numeric' })
})

// Поиск
const searchInput = ref('')
let searchTimer = null
function onSearch() { clearTimeout(searchTimer); searchTimer = setTimeout(() => store.setSearch(searchInput.value.trim()), 300) }
function clearSearch() { clearTimeout(searchTimer); searchInput.value = ''; store.setSearch('') }
watch(() => store.selectedId, () => { searchInput.value = '' })

// Диалог дня — день делится на активные и выполненные (архив этого дня).
const dayOpen = ref(false)
const dayDate = ref(null)
const dayDone = ref([])           // выполненные записи выбранного дня (догружаются)
const dayActive = computed(() => (dayDate.value ? dayEntries(dayDate.value) : []))
const dayTitle = computed(() => {
  if (!dayDate.value) return 'День'
  const s = new Date(dayDate.value).toLocaleDateString('ru-RU', { weekday: 'long', day: 'numeric', month: 'long' })
  return s.charAt(0).toUpperCase() + s.slice(1)
})
const dayActions = computed(() => store.readonly
  ? [{ kind: 'cancel', label: 'Закрыть' }]
  : [{ kind: 'cancel', label: 'Закрыть' }, { kind: 'confirm', label: 'Добавить запись', icon: 'add' }])

async function loadDayDone() {
  if (!dayDate.value || store.selectedId == null) { dayDone.value = []; return }
  const from = dayKey(dayDate.value)
  const to = dayKey(addDays(dayDate.value, 1))
  try {
    const data = await getEntries(store.selectedId, { archived: 1, from, to })
    dayDone.value = data.items ?? []
  } catch { dayDone.value = [] }
}

function openDay(day) {
  dayDate.value = new Date(day)
  dayOpen.value = true
  loadDayDone()
}

// ── Сортировка записей дня перетаскиванием (модалка дня, только владелец).
// Пока тянем — живой предпросмотр в ddOrder; на отпускании порядок сохраняется.
const ddDragId = ref(null)
const ddOrder = ref(null) // массив id в текущем (предпросмотровом) порядке
const dayOrdered = computed(() => {
  if (!ddOrder.value) return dayActive.value
  const byId = new Map(dayActive.value.map((e) => [e.id, e]))
  return ddOrder.value.map((id) => byId.get(id)).filter(Boolean)
})

function ddDragStart(ev, e) {
  ddDragId.value = e.id
  ddOrder.value = dayActive.value.map((x) => x.id)
  ev.dataTransfer.effectAllowed = 'move'
  ev.dataTransfer.setData('text/plain', String(e.id))
}
function ddDragOver(ev, target) {
  if (ddDragId.value == null || target.id === ddDragId.value || !ddOrder.value) return
  ev.preventDefault()
  ev.dataTransfer.dropEffect = 'move'
  const order = ddOrder.value.slice()
  const from = order.indexOf(ddDragId.value)
  const to = order.indexOf(target.id)
  if (from === -1 || to === -1 || from === to) return
  order.splice(from, 1)
  order.splice(to, 0, ddDragId.value)
  ddOrder.value = order
}
async function ddDrop() {
  if (ddDragId.value == null || !ddOrder.value || !dayDate.value) return
  const ids = ddOrder.value.slice()
  const changed = dayActive.value.some((e, i) => e.id !== ids[i])
  const date = dayKey(dayDate.value)
  ddDragId.value = null
  if (!changed) { ddOrder.value = null; return }
  try {
    await store.reorderDay(date, ids)
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить порядок')
  } finally {
    ddOrder.value = null
  }
}
function ddDragEnd() { ddDragId.value = null; ddOrder.value = null }

// Отметка/возврат прямо из модалки дня: обновляем и активные (в сторе), и
// выполненные этого дня.
async function dayToggle(e, done) {
  try {
    await store.toggleDone(e.id, done)
    await loadDayDone()
  } catch (err) { notif.error(err?.message || 'Не удалось изменить статус') }
}

// Диалог записи
const entryOpen = ref(false)
const activeEntry = ref(null)
const defaultDate = ref(null)
// Модалку дня НЕ закрываем: диалог записи открывается поверх неё, после закрытия
// записи остаёмся в модалке дня (для вызовов вне модалки dayOpen и так false).
function openEntry(e) { activeEntry.value = e; defaultDate.value = null; entryOpen.value = true }
function openCreate(day) {
  activeEntry.value = null
  defaultDate.value = day ? new Date(day) : new Date(store.cursor)
  entryOpen.value = true
  // Модалку дня НЕ закрываем: диалог записи открывается поверх неё, после
  // сохранения список активных в модалке дня обновится сам.
}

async function toggleDone(e, done) {
  try { await store.toggleDone(e.id, done) } catch (err) { notif.error(err?.message || 'Не удалось изменить статус') }
}

// Шаринг
const shareOpen = ref(false)
// Создание/переименование ежедневника
const nameOpen = ref(false)
const nameMode = ref('create')
const nameValue = ref('')
const nameBusy = ref(false)
const nameInput = ref(null)
function openCreateDiary() { nameMode.value = 'create'; nameValue.value = ''; nameOpen.value = true; nextTick(() => nameInput.value?.focus()) }
function openRenameDiary() { nameMode.value = 'rename'; nameValue.value = store.selected?.name || ''; nameOpen.value = true; nextTick(() => nameInput.value?.focus()) }
async function saveName() {
  const name = nameValue.value.trim()
  if (!name) { notif.error('Укажите название'); return }
  nameBusy.value = true
  try {
    if (nameMode.value === 'create') {
      const d = await store.createDiary(name)
      store.select(d.id)
    } else {
      await store.renameDiary(store.selectedId, name)
    }
    nameOpen.value = false
  } catch (e) {
    notif.error(e?.message || 'Не удалось сохранить')
  } finally {
    nameBusy.value = false
  }
}

const confirmDeleteDiary = ref(false)
async function doDeleteDiary() {
  confirmDeleteDiary.value = false
  try { await store.removeDiary(store.selectedId); notif.success('Ежедневник удалён') }
  catch (e) { notif.error(e?.message || 'Не удалось удалить') }
}

// Создание задачи с юнитом из записи
const taskFormEntry = ref(null)
function onCreateTask(entry) { entryOpen.value = false; taskFormEntry.value = entry }
async function onTaskSaved(task) {
  const entry = taskFormEntry.value
  taskFormEntry.value = null
  if (entry && task?.id) {
    try { await store.linkTask(entry.id, task.id); notif.success('Задача создана и привязана к записи') }
    catch { /* задача создана; связь не критична */ }
  }
}

// Экспорт
async function doExport() {
  try {
    let params
    if (store.subtab === 'archive') params = { archived: 1, search: store.search }
    else if (store.subtab === 'all') params = { search: store.search }
    else params = { from: dayKey(store.range.from), to: dayKey(store.range.to), search: store.search }
    const resp = await exportEntries(store.selectedId, params)
    if (!resp.ok) throw new Error('export_failed')
    const blob = await resp.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `${store.selected?.name || 'diary'}.xlsx`
    document.body.appendChild(a); a.click(); document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e) {
    notif.error(e?.message || 'Не удалось выгрузить')
  }
}

onMounted(() => {
  store.fetchDiaries().then(applySearchQuery)
  weekRO = new ResizeObserver(() => measureWeekColumn())
  if (weekGridRef.value) weekRO.observe(weekGridRef.value)
})
/* Переход из строки глобального поиска: открыть нужный ежедневник и
   подставить искомый текст — запись сразу видно в списке. */
function applySearchQuery() {
  const { diary, q } = route.query
  if (diary) store.select(Number(diary))
  if (q) store.setSearch(String(q))
}

watch(() => route.query, applySearchQuery)

onBeforeUnmount(() => {
  weekRO?.disconnect(); weekRO = null
})

// Грид появляется/исчезает при смене вида/подвкладки/устройства — переподключаем
// observer и пересчитываем после рендера.
watch([() => store.view, () => store.subtab, narrow, () => store.selectedId], () => {
  nextTick(() => {
    if (weekRO && weekGridRef.value) { weekRO.disconnect(); weekRO.observe(weekGridRef.value) }
    measureWeekColumn()
  })
})
watch(() => store.loadingEntries, () => nextTick(measureWeekColumn))
</script>

<style scoped>
.dv-side-owner { font-size: 12px; opacity: 0.8; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-side-progress { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.dv-side-bar { flex: 1; height: 4px; border-radius: var(--radius-full); background: var(--color-surface-highest); overflow: hidden; }
.dv-side-fill { display: block; height: 100%; border-radius: inherit; background: var(--color-success); transition: width 0.25s; }
.dv-side-count { flex-shrink: 0; font-size: 11px; font-weight: 600; font-variant-numeric: tabular-nums; opacity: 0.85; }
/* кнопка «Управление» — только на мобайле */

/* Тело */
.dv-body { position: relative; flex: 1; min-height: 0; overflow: auto; }
.dv-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px; background: var(--color-outline-dim); min-height: 100%; }
.dv-grid.month { grid-template-rows: auto repeat(6, 1fr); }
.dv-grid.week { grid-template-rows: 1fr; }
/* Sticky-шапка дней недели: записи прокручиваются под ней — полный акрил. */
.dv-wd { background: var(--acrylic-bg-strong); -webkit-backdrop-filter: var(--acrylic-blur); backdrop-filter: var(--acrylic-blur); padding: 8px 10px; text-align: center; font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; position: sticky; top: 0; z-index: 1; }
.dv-day { background: var(--acrylic-card-bg); min-height: 104px; padding: 6px; display: flex; flex-direction: column; gap: 4px; cursor: pointer; overflow: hidden; }
.dv-grid.week .dv-day { min-height: 0; }
/* Hover — глобальное «запотевание» .glass-hover (main.css), как в календаре. */
.dv-day.drop-target { background: var(--color-primary-container); outline: 2px dashed var(--color-primary); outline-offset: -2px; }
.dv-event[draggable='true'] { cursor: grab; }
.dv-event.dragging, .dv-dayrow.dragging { opacity: 0.4; }
.dv-day.dim { background: var(--color-surface-low); }
.dv-day.dim .dv-day-num { color: var(--color-text-dim); opacity: 0.6; }
.dv-day-head { display: flex; align-items: center; justify-content: space-between; }
.dv-day-num { font-size: 13px; font-weight: 700; color: var(--color-text); width: 24px; height: 24px; display: grid; place-items: center; }
.dv-day.today .dv-day-num { background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-full); }
.dv-day-wd { font-size: 11px; color: var(--color-text-dim); text-transform: uppercase; }
.dv-day-count { flex-shrink: 0; min-width: 18px; height: 18px; padding: 0 5px; display: inline-flex; align-items: center; justify-content: center; border-radius: var(--radius-full); background: var(--color-primary); color: var(--color-on-primary); font-size: 11px; font-weight: 700; }
.dv-day-events { display: flex; flex-direction: column; gap: 3px; min-height: 0; }
.dv-event { display: flex; align-items: baseline; gap: 6px; width: 100%; text-align: left; padding: 3px 6px; border-radius: var(--radius-sm); background: var(--color-primary-container); color: var(--color-on-primary-container); font-size: 12px; overflow: hidden; }
/* В неделе высота строки фиксирована — по ней считаем, сколько событий влезает в столбец. */
.dv-grid.week .dv-event { height: var(--dv-event-h, 22px); box-sizing: border-box; }
.dv-event-time { flex-shrink: 0; font-weight: 700; font-variant-numeric: tabular-nums; }
.dv-event-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-event-more { font-size: 11px; font-weight: 600; color: var(--color-text-dim); padding-left: 6px; }

.dv-agenda { display: flex; flex-direction: column; }
.dv-agenda-row { display: flex; align-items: center; gap: 14px; width: 100%; text-align: left; padding: 12px 16px; border: none; background: none; cursor: pointer; border-bottom: 1px solid var(--color-outline-dim); }
.dv-agenda-row:hover { background: var(--color-surface-high); }
.dv-agenda-date { flex-shrink: 0; width: 44px; display: flex; flex-direction: column; align-items: center; }
.dv-agenda-dnum { font-size: 18px; font-weight: 700; color: var(--color-text); }
.dv-agenda-row.today .dv-agenda-dnum { width: 30px; height: 30px; display: grid; place-items: center; background: var(--color-primary); color: var(--color-on-primary); border-radius: var(--radius-full); }
.dv-agenda-dwd { font-size: 11px; color: var(--color-text-dim); text-transform: uppercase; }
.dv-agenda-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.dv-agenda-month { font-size: 12px; color: var(--color-text-dim); }
.dv-agenda-prev { font-size: 14px; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-agenda-empty { font-size: 13px; color: var(--color-text-dim); }
.dv-agenda-chev { flex-shrink: 0; color: var(--color-text-dim); }

.dv-daylist { display: flex; flex-direction: column; gap: 8px; padding: 16px; }
.dv-dayrow { display: flex; align-items: center; gap: 14px; width: 100%; text-align: left; padding: 12px 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); background: var(--acrylic-card-bg); cursor: pointer; }
.dv-dayrow:hover { background: var(--color-surface-high); border-color: var(--color-outline); }
.dv-dayrow-num { flex-shrink: 0; min-width: 20px; text-align: right; font-size: 13px; font-weight: 700; color: var(--color-text-dim); font-variant-numeric: tabular-nums; }
.dv-dayrow-time { flex-shrink: 0; min-width: 56px; font-size: 15px; font-weight: 700; color: var(--color-primary); font-variant-numeric: tabular-nums; }
.dv-dayrow-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.dv-dayrow-title { font-size: 15px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-dayrow-sub { font-size: 13px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-dayrow-done { flex-shrink: 0; color: var(--color-success); display: grid; place-items: center; }
.dv-dayrow-done:hover { color: color-mix(in oklch, var(--color-success) 75%, var(--color-text)); }
.dv-dayrow-done.undo { color: var(--color-text-dim); }
.dv-dayrow-title.done { text-decoration: line-through; color: var(--color-text-dim); }
.dv-dayrow-chev { flex-shrink: 0; color: var(--color-text-dim); }
.dv-day-section { padding: 6px 4px 2px; font-size: 12px; font-weight: 700; text-transform: uppercase; letter-spacing: 0.04em; color: var(--color-text-dim); }

/* Все задачи */
.dv-all { display: flex; flex-direction: column; gap: 16px; padding: 16px; }
.dv-all-group { display: flex; flex-direction: column; gap: 8px; }

/* Архив */
.dv-archive { display: flex; flex-direction: column; gap: 8px; padding: 16px; }
.dv-arow { display: flex; align-items: center; gap: 12px; width: 100%; text-align: left; padding: 12px 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-lg); background: var(--acrylic-card-bg); cursor: pointer; }
.dv-arow:hover { background: var(--color-surface-high); }
.dv-arow-check { color: var(--color-success); flex-shrink: 0; }
.dv-arow-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.dv-arow-title { font-size: 15px; font-weight: 600; color: var(--color-text-dim); text-decoration: line-through; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-arow-meta { font-size: 12px; color: var(--color-text-dim); }
.dv-arow-act { flex-shrink: 0; display: grid; place-items: center; color: var(--color-text-dim); }
.dv-arow-act:hover { color: var(--color-primary); }
.dv-arow-chev { flex-shrink: 0; color: var(--color-text-dim); }

.dv-overlay { position: absolute; inset: 0; display: grid; place-items: center; background: color-mix(in oklch, var(--color-surface) 50%, transparent); }

/* Диалог дня */
.dd { display: flex; flex-direction: column; gap: 8px; }
.dd-empty { margin: 8px 0; color: var(--color-text-dim); text-align: center; }
.dd-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.dd-row { display: flex; align-items: stretch; gap: 6px; }
.dd-num { flex-shrink: 0; align-self: center; min-width: 22px; text-align: right; font-size: 13px; font-weight: 700; color: var(--color-text-dim); font-variant-numeric: tabular-nums; }
.dd-row[draggable='true'] { cursor: grab; }
.dd-row.dragging { opacity: 0.45; }
.dd-grip { flex-shrink: 0; display: grid; place-items: center; color: var(--color-text-dim); }
.dd-grip .material-symbols-outlined { font-size: 20px; }
.dd-check { flex-shrink: 0; width: 42px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--acrylic-card-bg); color: var(--color-text-dim); cursor: pointer; }
.dd-check:hover { color: var(--color-success); }
.dd-main { flex: 1; min-width: 0; display: flex; align-items: center; gap: 12px; text-align: left; padding: 10px 12px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-md); background: var(--acrylic-card-bg); cursor: pointer; }
.dd-main:hover { background: var(--color-surface-high); }
.dd-time { flex-shrink: 0; min-width: 48px; font-weight: 700; color: var(--color-primary); font-variant-numeric: tabular-nums; }
.dd-title { flex: 1; min-width: 0; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dd-chev { flex-shrink: 0; color: var(--color-text-dim); }
.dd-group { display: flex; flex-direction: column; gap: 8px; }
.dd-group + .dd-group { margin-top: 16px; }
.dd-grouplabel { font-size: 12px; font-weight: 700; text-transform: uppercase; color: var(--color-text-dim); letter-spacing: 0.04em; }
.dd-check.done { color: var(--color-success); }
.dd-title.done { text-decoration: line-through; color: var(--color-text-dim); }

/* Архив по дням */
.dv-arc-group + .dv-arc-group { margin-top: 6px; }
.dv-arc-daylabel { padding: 12px 4px 6px; font-size: 13px; font-weight: 700; color: var(--color-text-dim); text-transform: capitalize; }

.dv-name-input { width: 100%; padding: 12px 14px; font: inherit; color: var(--color-text); background: var(--color-surface-high); background: var(--glass-bg); box-shadow: var(--glass-edge); border: 1px solid var(--acrylic-border); border-radius: var(--radius-md); outline: none; }
.dv-name-input:focus { border-color: var(--color-primary); }

/* Мобайл: кнопка-селектор ежедневника (открывает шторку) */
.dv-mobile-bar { flex: none; padding: 8px 12px 4px; }
.dv-diary-select {
  display: flex; align-items: center; gap: 10px; width: 100%; padding: 11px 14px;
  border: 1px solid var(--acrylic-border); border-radius: var(--radius-full);
  background: var(--acrylic-card-bg); background: var(--glass-bg); box-shadow: var(--glass-edge);
  color: var(--color-text); font: inherit; font-weight: 600; font-size: 15px; cursor: pointer;
}
.dv-diary-select-icon { color: var(--color-primary); font-size: 20px; flex-shrink: 0; }
.dv-diary-select-name { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; text-align: left; }
.dv-diary-select-chev { color: var(--color-text-dim); flex-shrink: 0; }

/* Телефон: тело скроллится под панель задач каркаса — оставляем ей воздух. */
@media (max-width: 768px) {
  .dv-body { padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px)); }
}
</style>
