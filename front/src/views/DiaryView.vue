<template>
  <div class="dv split-view">
    <!-- ЛЕВАЯ ПАНЕЛЬ -->
    <aside class="split-side">
      <div class="dv-side-head">
        <NotesHubTabs full-width />
      </div>
      <div class="dv-side-tabs">
        <SegmentedTabs :model-value="store.tab" :tabs="tabs" full-width dense @update:model-value="store.setTab" />
      </div>
      <div class="split-side-list">
        <div v-if="store.loadingList" class="split-side-note">Загрузка…</div>
        <template v-else>
          <button
            v-for="d in store.diaries"
            :key="d.id"
            class="split-side-item"
            :class="{ active: d.id === store.selectedId, 'drop-target': dropDiaryId === d.id }"
            @click="store.select(d.id)"
            @dragover="onDiaryDragOver($event, d)"
            @dragleave="dropDiaryId === d.id && (dropDiaryId = null)"
            @drop="onDiaryDrop($event, d)"
          >
            <span class="split-item-tile">
              <span class="material-symbols-outlined">{{ store.tab === 'shared' ? 'folder_shared' : 'book' }}</span>
            </span>
            <span class="dv-side-main">
              <span class="split-side-name">{{ d.name }}</span>
              <span v-if="store.tab === 'shared'" class="dv-side-owner">{{ d.owner_name }}</span>
              <span v-if="diaryTotal(d)" class="dv-side-progress">
                <span class="dv-side-bar"><span class="dv-side-fill" :style="{ width: diaryPct(d) + '%' }" /></span>
                <span class="dv-side-count">{{ d.done_count || 0 }}/{{ diaryTotal(d) }}</span>
              </span>
            </span>
          </button>
          <p v-if="!store.diaries.length" class="split-side-note">
            {{ store.tab === 'shared' ? 'С вами пока не делились' : 'Ежедневников нет' }}
          </p>
        </template>
      </div>
      <button v-if="store.tab === 'mine'" class="split-side-add" @click="openCreateDiary">
        <span class="material-symbols-outlined">add</span> Новый ежедневник
      </button>
    </aside>

    <!-- ПРАВАЯ ПАНЕЛЬ -->
    <section class="split-main">
      <!-- Мобайл: левая колонка скрыта, переключатель Заметки/Ежедневник — здесь -->
      <div v-if="isMobile" class="dv-mobile-hub">
        <NotesHubTabs full-width />
      </div>
      <div v-if="isMobile && store.diaries.length" class="dv-regstrip">
        <button
          v-for="d in store.diaries" :key="d.id"
          class="dv-regchip" :class="{ active: d.id === store.selectedId }"
          @click="store.select(d.id)"
        >{{ d.name }}</button>
      </div>

      <template v-if="store.selected">
        <header class="dv-toolbar">
          <div class="dv-subtabs">
            <SegmentedTabs :model-value="store.subtab" :tabs="subtabs" dense @update:model-value="store.setSubtab" />
          </div>

          <div v-if="store.subtab === 'active'" class="dv-nav">
            <button class="dv-icon-btn" title="Назад" @click="store.step(-1)"><span class="material-symbols-outlined">chevron_left</span></button>
            <button class="dv-today" @click="store.today()">Сегодня</button>
            <button class="dv-icon-btn" title="Вперёд" @click="store.step(1)"><span class="material-symbols-outlined">chevron_right</span></button>
            <h2 class="dv-period">{{ periodLabel }}</h2>
          </div>

          <SearchField
            v-model="searchInput"
            class="dv-toolbar-search"
            placeholder="Поиск по записям…"
            hotkey
            @update:model-value="onSearch"
            @clear="clearSearch"
          />

          <div v-if="store.subtab === 'active'" class="dv-viewseg">
            <button v-for="v in viewModes" :key="v.value" :class="{ active: store.view === v.value }" @click="store.setView(v.value)">{{ v.label }}</button>
          </div>

          <div class="dv-actions">
            <!-- Мобайл: всё управление — в отдельном листе «Управление» -->
            <button class="dv-icon-btn dv-mobile-controls" title="Управление" @click="controlsOpen = true"><span class="material-symbols-outlined">tune</span></button>
            <template v-if="!store.readonly">
              <button class="dv-icon-btn dv-manage" title="Переименовать" @click="openRenameDiary"><span class="material-symbols-outlined">edit</span></button>
              <button class="dv-icon-btn dv-manage" title="Удалить ежедневник" @click="confirmDeleteDiary = true"><span class="material-symbols-outlined">delete</span></button>
              <button class="dv-icon-btn dv-manage" title="Поделиться" @click="shareOpen = true"><span class="material-symbols-outlined">share</span></button>
            </template>
            <button class="dv-icon-btn dv-manage" title="Экспорт в XLSX" @click="doExport"><span class="material-symbols-outlined">download</span></button>
            <button v-if="!store.readonly" class="btn-grad" @click="openCreate()">
              <span class="material-symbols-outlined">add</span><span class="dv-btn-label">Запись</span>
            </button>
          </div>
        </header>

        <div class="dv-body">
          <!-- АРХИВ — выполненные, сгруппированные по дням -->
          <div v-if="store.subtab === 'archive'" class="dv-archive">
            <div v-if="!store.archive.length" class="dv-empty">
              <span class="material-symbols-outlined">inventory_2</span>
              <p>Архив пуст — выполненные записи появятся здесь</p>
            </div>
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
            <div v-if="!store.entries.length" class="dv-empty">
              <span class="material-symbols-outlined">checklist</span>
              <p>Активных записей нет</p>
            </div>
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
            <div v-if="!isMobile && store.view !== 'day'" ref="weekGridRef" class="dv-grid" :class="store.view">
              <template v-if="store.view === 'month'">
                <div v-for="(wd, i) in weekdays" :key="'h' + i" class="dv-wd">{{ wd }}</div>
              </template>
              <div
                v-for="day in gridDays" :key="dayKey(day)"
                class="dv-day" :class="{ dim: store.view === 'month' && !inCurrentMonth(day), today: isToday(day), 'drop-target': dropDayKey === dayKey(day) }"
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

            <div v-else-if="isMobile && store.view !== 'day'" class="dv-agenda">
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
              <div v-if="!dayEntries(store.cursor).length && !store.dayDone.length" class="dv-empty">
                <span class="material-symbols-outlined">event_busy</span>
                <p>На этот день записей нет</p>
                <button v-if="!store.readonly" class="dv-btn-tonal" @click="openCreate(store.cursor)">
                  <span class="material-symbols-outlined">add</span> Добавить запись
                </button>
              </div>
              <template v-else>
                <template v-if="dayEntries(store.cursor).length">
                  <div class="dv-day-section">Активные</div>
                  <button
                    v-for="e in dayEntries(store.cursor)" :key="e.id" class="dv-dayrow"
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
                </template>
                <template v-if="store.dayDone.length">
                  <div class="dv-day-section">Выполнено</div>
                  <button v-for="e in store.dayDone" :key="e.id" class="dv-dayrow" @click="openEntry(e)">
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

          <div v-if="store.loadingEntries" class="dv-overlay"><span class="material-symbols-outlined spin">progress_activity</span></div>
        </div>
      </template>

      <EmptyState
        v-else
        class="split-empty"
        icon="event_note"
        tone="soft"
        :title="store.diaries.length ? (isMobile ? 'Выберите ежедневник сверху' : 'Выберите ежедневник слева') : 'Создайте свой первый ежедневник'"
        :subtitle="store.diaries.length
          ? 'Выберите ежедневник в списке, чтобы посмотреть записи'
          : 'Планируйте дела по дням и отмечайте выполненное'"
      >
        <button v-if="store.tab === 'mine' && !store.diaries.length" class="btn-grad" @click="openCreateDiary">
          <span class="material-symbols-outlined">add</span> Новый ежедневник
        </button>
      </EmptyState>
    </section>

    <AppFab
      :visible="isMobile && !!store.selected && !store.readonly && fabVisible"
      icon="add"
      aria-label="Добавить запись"
      @click="openCreate()"
    />

    <!-- Диалог дня -->
    <AppDialog v-model="dayOpen" :title="dayTitle" icon="today" size="md" :actions="dayActions" @cancel="dayOpen = false" @confirm="openCreate(dayDate)">
      <div class="dd">
        <p v-if="!dayActive.length && !dayDone.length" class="dd-empty">На этот день записей нет.</p>

        <div v-if="dayActive.length" class="dd-group">
          <span class="dd-grouplabel">Активные</span>
          <ul class="dd-list">
            <li
              v-for="e in dayOrdered" :key="e.id" class="dd-row"
              :class="{ dragging: ddDragId === e.id }"
              :draggable="canDrag && dayOrdered.length > 1"
              @dragstart="ddDragStart($event, e)" @dragend="ddDragEnd"
              @dragover="ddDragOver($event, e)" @drop.prevent="ddDrop"
            >
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
            <li v-for="e in dayDone" :key="e.id" class="dd-row">
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

    <!-- Мобайл: лист управления (вид, поиск, действия) -->
    <AppDialog v-model="controlsOpen" title="Управление" icon="tune" size="sm" mobile="sheet" :actions="[{ kind: 'cancel', label: 'Готово' }]" @cancel="controlsOpen = false">
      <div class="dv-controls">
        <div v-if="store.subtab === 'active'" class="dv-ctl-block">
          <span class="dv-ctl-label">Вид</span>
          <SegmentedTabs :model-value="store.view" :tabs="viewModes" full-width @update:model-value="store.setView" />
        </div>
        <div class="dv-ctl-block">
          <span class="dv-ctl-label">Поиск</span>
          <div class="dv-search dv-ctl-search">
            <span class="material-symbols-outlined">search</span>
            <input v-model="searchInput" type="text" placeholder="Поиск…" @input="onSearch" />
            <button v-if="searchInput" class="dv-search-clear" @click="clearSearch"><span class="material-symbols-outlined">close</span></button>
          </div>
        </div>
        <div class="dv-ctl-actions">
          <button v-if="!store.readonly" class="dv-ctl-btn" @click="controlsOpen = false; openRenameDiary()"><span class="material-symbols-outlined">edit</span> Переименовать</button>
          <button v-if="!store.readonly" class="dv-ctl-btn" @click="controlsOpen = false; shareOpen = true"><span class="material-symbols-outlined">share</span> Поделиться</button>
          <button class="dv-ctl-btn" @click="controlsOpen = false; doExport()"><span class="material-symbols-outlined">download</span> Экспорт в XLSX</button>
          <button v-if="!store.readonly" class="dv-ctl-btn danger" @click="controlsOpen = false; confirmDeleteDiary = true"><span class="material-symbols-outlined">delete</span> Удалить ежедневник</button>
        </div>
      </div>
    </AppDialog>

    <!-- Создание/переименование ежедневника -->
    <AppDialog
      v-model="nameOpen"
      :title="nameMode === 'create' ? 'Новый ежедневник' : 'Переименовать'"
      icon="book" size="sm" :busy="nameBusy"
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
  </div>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import SearchField from '@/components/common/SearchField.vue'
import AppFab from '@/components/common/AppFab.vue'
import { useFabOnScroll } from '@/composables/useFabOnScroll.js'
import NotesHubTabs from '@/components/notes/NotesHubTabs.vue'
import DiaryEntryDialog from '@/components/diary/DiaryEntryDialog.vue'
import DiaryShareDialog from '@/components/diary/DiaryShareDialog.vue'
import TaskForm from '@/components/tasks/TaskForm.vue'
import { useDiariesStore, dayKey } from '@/stores/diaries.js'
import { exportEntries, getEntries } from '@/api/diaries.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const store = useDiariesStore()
const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()
// Мобильный FAB «Добавить запись»: прячется/появляется по прокрутке.
const { fabVisible } = useFabOnScroll()

const tabs = [
  { value: 'mine', label: 'Мои', icon: 'book' },
  { value: 'shared', label: 'Поделились', icon: 'folder_shared' },
]
const subtabs = [
  { value: 'active', label: 'Активные', icon: 'checklist' },
  { value: 'all', label: 'Все задачи', icon: 'list' },
  { value: 'archive', label: 'Архив', icon: 'inventory_2' },
]
const viewModes = [
  { value: 'month', label: 'Месяц' },
  { value: 'week', label: 'Неделя' },
  { value: 'day', label: 'День' },
]
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
// Мобильный лист управления
const controlsOpen = ref(false)

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
  store.fetchDiaries()
  weekRO = new ResizeObserver(() => measureWeekColumn())
  if (weekGridRef.value) weekRO.observe(weekGridRef.value)
})
onBeforeUnmount(() => { weekRO?.disconnect(); weekRO = null })

// Грид появляется/исчезает при смене вида/подвкладки/устройства — переподключаем
// observer и пересчитываем после рендера.
watch([() => store.view, () => store.subtab, isMobile, () => store.selectedId], () => {
  nextTick(() => {
    if (weekRO && weekGridRef.value) { weekRO.disconnect(); weekRO.observe(weekGridRef.value) }
    measureWeekColumn()
  })
})
watch(() => store.loadingEntries, () => nextTick(measureWeekColumn))
</script>

<style scoped>
/* Каркас (стеклянные панели, список, кнопка добавления, мобильное скрытие
   левой панели) — глобальный паттерн .split-* (main.css). Здесь — только
   специфика ежедневников: шапка с NotesHubTabs, прогресс и drop-цель пункта. */
.dv-side-head { flex-shrink: 0; display: flex; align-items: center; gap: 8px; padding: 12px; border-bottom: 1px solid var(--color-outline-dim); }
.dv-side-head :deep(.seg-tabs) { flex: 1; }
.dv-mobile-hub { flex-shrink: 0; padding: 10px 12px 0; }
.dv-side-tabs { padding: 10px 10px 4px; }
.dv-side-main { flex: 1; display: flex; flex-direction: column; min-width: 0; }
.dv-side-owner { font-size: 12px; opacity: 0.8; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-side-progress { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.dv-side-bar { flex: 1; height: 4px; border-radius: var(--radius-full); background: var(--color-surface-highest); overflow: hidden; }
.split-side-item.active .dv-side-bar { background: color-mix(in oklch, var(--color-primary) 20%, transparent); }
.dv-side-fill { display: block; height: 100%; border-radius: inherit; background: var(--color-success); transition: width 0.25s; }
.dv-side-count { flex-shrink: 0; font-size: 11px; font-weight: 600; font-variant-numeric: tabular-nums; opacity: 0.85; }
.split-side-item.drop-target {
  outline: 2px dashed var(--color-primary);
  outline-offset: -2px;
  background: color-mix(in oklch, var(--color-primary) 8%, transparent);
}

/* Правая панель */
.dv-toolbar { flex-shrink: 0; display: flex; align-items: center; gap: 12px; flex-wrap: wrap; padding: 12px 16px; border-bottom: 1px solid var(--color-outline-dim); }
.dv-subtabs { flex-shrink: 0; }
.dv-nav { display: flex; align-items: center; gap: 8px; }
.dv-period { margin: 0 0 0 6px; font-size: 16px; font-weight: 700; color: var(--color-text); text-transform: capitalize; white-space: nowrap; }
.dv-today { height: 36px; padding: 0 14px; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--acrylic-card-bg); color: var(--color-text); font-weight: 600; font-size: 13px; cursor: pointer; }
.dv-today:hover { background: var(--color-surface-high); }
/* Сегмент вида — единый стиль с периодами статистики (StatsPeriodControl). */
.dv-viewseg { display: inline-flex; gap: 2px; padding: 4px; background: var(--color-surface-high); border-radius: var(--radius-full); }
.dv-viewseg button {
  min-height: 36px; padding: 8px 14px; border: none; background: transparent;
  border-radius: var(--radius-full); color: var(--color-text-dim); cursor: pointer;
  font-weight: 600; font-size: 13px; transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.dv-viewseg button:hover:not(.active) { color: var(--color-text); }
.dv-viewseg button.active { background: var(--acrylic-card-bg); color: var(--color-primary); font-weight: 700; box-shadow: var(--shadow-sm); }
.dv-search { flex: 1 1 auto; display: flex; align-items: center; gap: 8px; height: 38px; padding: 0 12px; min-width: 170px; background: var(--color-surface-low); border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); }
.dv-search > .material-symbols-outlined { color: var(--color-text-dim); font-size: 20px; }
.dv-search input { flex: 1; min-width: 0; border: none; background: none; outline: none; color: var(--color-text); font-size: 14px; }
.dv-search-clear { border: none; background: none; cursor: pointer; color: var(--color-text-dim); display: grid; place-items: center; }
.dv-actions { display: flex; align-items: center; gap: 8px; }
.dv-icon-btn { width: 38px; height: 38px; display: grid; place-items: center; border: 1px solid var(--color-outline-dim); border-radius: var(--radius-full); background: var(--acrylic-card-bg); color: var(--color-text-dim); cursor: pointer; }
.dv-icon-btn:hover { background: var(--color-surface-high); color: var(--color-text); }
.dv-mobile-controls { display: none; } /* кнопка «Управление» — только на мобайле */
.dv-btn-tonal { display: inline-flex; align-items: center; gap: 6px; height: 38px; padding: 0 16px; border: none; border-radius: var(--radius-full); background: var(--color-primary-container); color: var(--color-on-primary-container); font-weight: 600; font-size: 14px; cursor: pointer; }

/* Тело */
.dv-body { position: relative; flex: 1; min-height: 0; overflow: auto; }
.dv-grid { display: grid; grid-template-columns: repeat(7, 1fr); gap: 1px; background: var(--color-outline-dim); min-height: 100%; }
.dv-grid.month { grid-template-rows: auto repeat(6, 1fr); }
.dv-grid.week { grid-template-rows: 1fr; }
/* Sticky-шапка дней недели: записи прокручиваются под ней — полный акрил. */
.dv-wd { background: var(--acrylic-bg-strong); -webkit-backdrop-filter: var(--acrylic-blur); backdrop-filter: var(--acrylic-blur); padding: 8px 10px; text-align: center; font-size: 12px; font-weight: 700; color: var(--color-text-dim); text-transform: uppercase; position: sticky; top: 0; z-index: 1; }
.dv-day { background: var(--acrylic-card-bg); min-height: 104px; padding: 6px; display: flex; flex-direction: column; gap: 4px; cursor: pointer; overflow: hidden; }
.dv-grid.week .dv-day { min-height: 0; }
.dv-day:hover { background: var(--color-surface-high); }
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
.dv-dayrow-time { flex-shrink: 0; min-width: 56px; font-size: 15px; font-weight: 700; color: var(--color-primary); font-variant-numeric: tabular-nums; }
.dv-dayrow-body { flex: 1; min-width: 0; display: flex; flex-direction: column; }
.dv-dayrow-title { font-size: 15px; font-weight: 600; color: var(--color-text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-dayrow-sub { font-size: 13px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dv-dayrow-done { flex-shrink: 0; color: var(--color-success); display: grid; place-items: center; }
.dv-dayrow-done:hover { transform: scale(1.1); }
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

.dv-empty { display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 10px; padding: 48px 16px; color: var(--color-text-dim); }
.dv-empty .material-symbols-outlined { font-size: 44px; }
.dv-empty p { margin: 0; }
.dv-overlay { position: absolute; inset: 0; display: grid; place-items: center; background: color-mix(in oklch, var(--color-surface) 50%, transparent); }
.spin { animation: dvspin 1s linear infinite; font-size: 32px; color: var(--color-primary); }
@keyframes dvspin { to { transform: rotate(360deg); } }

/* Диалог дня */
.dd { display: flex; flex-direction: column; gap: 8px; }
.dd-empty { margin: 8px 0; color: var(--color-text-dim); text-align: center; }
.dd-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.dd-row { display: flex; align-items: stretch; gap: 6px; }
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

.dv-name-input { width: 100%; padding: 12px 14px; font: inherit; color: var(--color-text); background: var(--color-surface-high); border: 1px solid var(--color-outline-variant); border-radius: var(--radius-md); outline: none; }
.dv-name-input:focus { border-color: var(--color-primary); }

/* Мобайл */
.dv-regstrip { flex: none; display: flex; gap: 8px; padding: 10px 12px; overflow-x: auto; border-bottom: 1px solid var(--color-outline-dim); }
.dv-regchip { flex: none; padding: 8px 14px; border-radius: var(--radius-full); border: 1px solid var(--color-outline-dim); background: var(--acrylic-card-bg); color: var(--color-text-dim); font-size: 14px; font-weight: 600; cursor: pointer; white-space: nowrap; }
.dv-regchip.active { background: var(--color-primary); color: var(--color-on-primary); border-color: transparent; }

@media (max-width: 768px) {
  /* Скрытие левой панели и разворот правой — в глобальном .split-* */
  /* Резерв под нижнюю навигацию (64px) + 12px воздуха: сетка/списки
     скроллятся под стекло, последние записи не прячутся за навигацией. */
  .dv-body { padding-bottom: calc(76px + env(safe-area-inset-bottom, 0px)); }
  .dv-toolbar { padding: 10px 12px; gap: 8px; }
  .dv-period { font-size: 14px; }
  /* Вид, поиск и управляющие иконки уезжают в лист «Управление» — на узком
     экране в панели остаются только вкладки, навигация по периоду и «Запись». */
  .dv-toolbar .dv-viewseg,
  .dv-toolbar .dv-toolbar-search,
  .dv-manage { display: none; }
  /* Создание записи на мобильном — плавающий FAB, кнопка тулбара не нужна. */
  .dv-actions .btn-grad { display: none; }
  .dv-mobile-controls { display: grid; }
  .dv-subtabs { order: 0; flex-basis: 100%; }
  .dv-subtabs :deep(.seg-tabs) { width: 100%; }
  .dv-subtabs :deep(.seg-tab) { flex: 1; }
  /* Навигация по периоду и кнопка «Управление» — в ОДНОЙ строке (отдельная
     строка под одну иконку съедала слишком много места сверху). */
  .dv-nav { order: 1; flex: 1 1 auto; min-width: 0; }
  .dv-nav .dv-period { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; }
  .dv-actions { order: 2; flex-basis: auto; flex-shrink: 0; justify-content: flex-end; }
}

/* Лист управления (мобайл) */
.dv-controls { display: flex; flex-direction: column; gap: 18px; }
.dv-ctl-block { display: flex; flex-direction: column; gap: 8px; }
.dv-ctl-label { font-size: 13px; font-weight: 600; color: var(--color-text-dim); }
.dv-ctl-search { flex: 0 0 auto; width: 100%; height: 44px; }
.dv-ctl-actions { display: flex; flex-direction: column; gap: 4px; }
.dv-ctl-btn {
  display: flex; align-items: center; gap: 12px; width: 100%; padding: 12px 10px;
  border: none; background: none; cursor: pointer; border-radius: var(--radius-md);
  color: var(--color-text); font: inherit; font-weight: 600; font-size: 15px; text-align: left;
}
.dv-ctl-btn:hover { background: var(--color-surface-high); }
.dv-ctl-btn .material-symbols-outlined { color: var(--color-text-dim); }
.dv-ctl-btn.danger { color: var(--color-error); }
.dv-ctl-btn.danger .material-symbols-outlined { color: var(--color-error); }
</style>
