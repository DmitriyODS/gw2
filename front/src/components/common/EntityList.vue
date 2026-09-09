<template>
  <AppPage
    embedded
    :title="title"
    show-title
    :menu="!narrow"
    menu-icon="left_panel_close"
    menu-label="Свернуть список"
    :scroll="false"
    @menu="$emit('toggle')"
  >
    <template v-if="scopes.length" #subhead>
      <AppTabs
        :model-value="scope"
        :tabs="scopes"
        variant="tint"
        dense
        full-width
        @update:model-value="$emit('update:scope', $event)"
      />
    </template>

    <div class="el">
      <!-- Фильтр появляется, только когда список длинный: у трёх пунктов он
           отнимал бы строку, ничего не давая взамен. -->
      <div v-if="showFilter" class="el-filter">
        <SearchField v-model="filter" :placeholder="filterPlaceholder" />
      </div>

      <div class="el-scroll" @dragleave="onListDragLeave">
        <!-- В пустой области показываем ТОЛЬКО заглушку: папки общие на раздел
             (вкладка «Мои» и «Поделились» — разные наборы одного списка), и
             висеть под «здесь пока пусто» им незачем — складывать в них нечего. -->
        <EmptyState
          v-if="!items.length"
          size="sm"
          :icon="empty.icon || icon"
          :title="empty.title || 'Пусто'"
          :subtitle="empty.subtitle || ''"
        />
        <p v-else-if="!visibleCount" class="el-nomatch">Ничего не найдено</p>

        <template v-if="items.length">
        <div v-for="g in groups" :key="g.key" class="el-group">
          <!-- Заголовок группы: у «остальных» он появляется, только когда есть
               папки — иначе это просто список, и подпись над ним лишняя. -->
          <div
            v-if="g.kind !== 'loose' || hasFolders"
            class="el-ghead"
            :class="{ drop: dropGroup === g.key, folder: g.kind === 'folder' }"
            :draggable="g.kind === 'folder' && organizable && !renamingFolder"
            @dragstart="onFolderDragStart($event, g)"
            @dragend="endDrag"
            @dragover="onGroupDragOver($event, g)"
            @dragleave="dropGroup === g.key && (dropGroup = null)"
            @drop="onGroupDrop($event, g)"
            @contextmenu.prevent="g.kind === 'folder' && openFolderMenu(g, $event)"
          >
            <!-- Сравниваем и вид группы: у «закреплённых» и «остальных» id
                 пустой, и одного сравнения с ним хватало, чтобы правка имени
                 открылась не в той шапке. -->
            <AppInlineEdit
              v-if="g.kind === 'folder' && renamingFolder === g.id"
              :model-value="g.name"
              placeholder="Название папки"
              @save="saveFolderName(g, $event)"
              @cancel="cancelFolderName(g)"
            />
            <template v-else>
              <button class="el-gtoggle" type="button" @click="toggleGroup(g)">
                <span class="material-symbols-outlined el-gchev" :class="{ closed: g.collapsed }">expand_more</span>
                <span class="material-symbols-outlined el-gicon">{{ groupIcon(g) }}</span>
                <span class="el-gname">{{ g.name }}</span>
                <span class="el-gcount">{{ g.items.length }}</span>
              </button>
              <AppButton
                v-if="g.kind === 'folder'"
                variant="icon"
                size="sm"
                icon="more_horiz"
                title="Действия с папкой"
                aria-label="Действия с папкой"
                @click="openFolderMenu(g, $event)"
              />
            </template>
          </div>

          <AppStack v-if="!g.collapsed" :gap="6" class="el-items">
            <div
              v-for="item in g.items"
              :key="keyOf(item, idKey)"
              class="el-item"
              :class="itemClass(item)"
              :draggable="canDragItem && renamingId !== item[idKey]"
              @dragstart="onItemDragStart($event, item)"
              @dragend="endDrag"
              @dragover="onItemDragOver($event, item)"
              @dragleave="onItemDragLeave($event, item)"
              @drop="onItemDrop($event, item)"
            >
              <AppInlineEdit
                v-if="renamingId != null && String(renamingId) === keyOf(item, idKey)"
                :model-value="String(item[labelKey] ?? '')"
                :placeholder="renamePlaceholder"
                :maxlength="renameMaxlength"
                @save="$emit('rename', item, $event)"
                @cancel="$emit('rename-cancel')"
              />
              <AppRow
                v-else
                :title="String(item[labelKey] ?? '')"
                :icon="iconOf ? iconOf(item) : icon"
                :hint="hint ? hint(item) : ''"
                dense
                clickable
                inline
                :selected="isSelected(item)"
                :tone="String(dropId ?? '') === keyOf(item, idKey) ? 'primary' : 'neutral'"
                @click="$emit('select', item[idKey])"
                @contextmenu.prevent="openItemMenu(item, $event)"
              >
                <template v-if="slots.hint" #hint><slot name="hint" :item="item" /></template>
                <!-- Управление отдаём строке, только когда есть что показать:
                     пустой слот всё равно занял бы отступ справа. -->
                <template v-if="slots.chips || isPinned(item)" #default>
                  <slot name="chips" :item="item" />
                  <span
                    v-if="isPinned(item)"
                    class="material-symbols-outlined el-pin"
                    title="Закреплено"
                  >keep</span>
                </template>
              </AppRow>
            </div>

            <!-- Пустая папка сама говорит, что с ней делать, — но только пока
                 что-то тащат: постоянная подсказка засоряла бы список. -->
            <div
              v-if="!g.items.length && g.kind === 'folder' && drag.kind === 'item'"
              class="el-drop-hint"
              :class="{ drop: dropGroup === g.key }"
              @dragover="onGroupDragOver($event, g)"
              @dragleave="dropGroup === g.key && (dropGroup = null)"
              @drop="onGroupDrop($event, g)"
            >Перетащите сюда</div>
          </AppStack>
        </div>
        </template>
      </div>

      <!-- Кнопка прижата к низу и видна всегда: заводить записи — самое частое
           действие панели, и прокручивать до конца ради неё незачем. -->
      <div v-if="canCreate || organizable" class="el-foot">
        <AppButton
          v-if="canCreate"
          variant="filled"
          icon="add"
          :label="createLabel"
          full-width
          @click="$emit('create')"
        />
        <!-- Рядом с «создать» папка — кнопка-значок, без неё — обычная кнопка с
             подписью: одинокий значок в углу подвала читался случайным. -->
        <AppButton
          v-if="organizable && canCreate"
          variant="icon"
          icon="create_new_folder"
          title="Новая папка"
          aria-label="Новая папка"
          @click="createFolder"
        />
        <AppButton
          v-else-if="organizable"
          variant="glass"
          icon="create_new_folder"
          label="Новая папка"
          full-width
          @click="createFolder"
        />
      </div>
    </div>
  </AppPage>

  <!-- Папка — только раскладка, и её удаление ничего не уносит: об этом и
       говорит подтверждение, иначе человек ждёт потери самих записей. -->
  <ConfirmDialog
    :visible="!!folderToDelete"
    header="Удалить папку?"
    :message="`Папка «${folderToDelete?.name || ''}» удалится, а записи из неё останутся в списке.`"
    confirm-label="Удалить"
    danger-confirm
    @confirm="doRemoveFolder"
    @cancel="folderToDelete = null"
  />

  <ContextMenu
    :visible="menu.open"
    :x="menu.x"
    :y="menu.y"
    :items="menuItemsList"
    @select="onMenuSelect"
    @close="menu.open = false"
  />
</template>

<script setup>
/* Боковой список раздела: области, закрепление, папки и ручной порядок.

   Один компонент на все разделы со списком слева (ежедневники, реестры, формы,
   расписания, календари) — раньше каждая панель повторяла свою разметку, а
   организации у них не было вовсе. Своих запросов не делает: сами записи
   приходят пропсами, а организация — ЛИЧНАЯ и живёт в stores/listPrefs.js
   (закрепление и папки — про взгляд человека на список, а не про сущность:
   в одном списке соседствуют свои и расшаренные записи).

   Порядок хранится плоской последовательностью ключей показа, поэтому
   перетаскивание внутри группы, между папками и в «закреплённые» — одна и та
   же операция: убрать ключ и вставить его перед соседом.

   Тач-устройства перетаскивания не знают, поэтому те же действия есть в
   контекстном меню («Вверх»/«Вниз» показываются только там, где нет мыши). */
import { computed, reactive, ref, useSlots, watch } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppInlineEdit from '@/components/ui/AppInlineEdit.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import ConfirmDialog from '@/components/ui/ConfirmDialog.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import SearchField from '@/components/common/SearchField.vue'
import { useListPrefsStore } from '@/stores/listPrefs.js'
import { flattenGroups, keyOf, moveKey, organizeList, shiftKey } from '@/utils/listOrganize.js'

const props = defineProps({
  /** Ключ раздела в личных настройках списков (diaries, registries, …). */
  section: { type: String, required: true },
  items: { type: Array, default: () => [] },
  idKey: { type: String, default: 'id' },
  labelKey: { type: String, default: 'name' },
  selectedId: { type: [Number, String, null], default: null },
  title: { type: String, default: '' },
  /** Значок пункта: один на список либо своя функция (у форм он зависит от вида). */
  icon: { type: String, default: 'description' },
  iconOf: { type: Function, default: null },
  /** Подпись под названием (владелец, состояние). Богатая — через слот `hint`. */
  hint: { type: Function, default: null },
  /** Вкладки областей ([{ value, label }]); пусто — вкладок нет. */
  scopes: { type: Array, default: () => [] },
  scope: { type: String, default: '' },
  /** id записи, название которой сейчас правят прямо в списке. */
  renamingId: { type: [Number, String, null], default: null },
  renamePlaceholder: { type: String, default: 'Название' },
  renameMaxlength: { type: Number, default: 120 },
  narrow: { type: Boolean, default: false },
  createLabel: { type: String, default: 'Создать' },
  canCreate: { type: Boolean, default: true },
  /** Организация списка (закрепление, папки, порядок) — можно выключить. */
  organizable: { type: Boolean, default: true },
  filterable: { type: Boolean, default: true },
  filterPlaceholder: { type: String, default: 'Фильтр по названию…' },
  empty: { type: Object, default: () => ({}) },
  /** Пункты меню самой записи: (item) => [{ label, icon, action, danger, children }]. */
  menuItems: { type: Function, default: null },
  /** Подсветка пункта при внешнем перетаскивании (запись в другой ежедневник). */
  dropId: { type: [Number, String, null], default: null },
})

const emit = defineEmits([
  'select', 'create', 'toggle', 'update:scope', 'command', 'rename', 'rename-cancel',
  'item-dragover', 'item-dragleave', 'item-drop',
])

const slots = useSlots()
const prefs = useListPrefsStore()
prefs.load()

const filter = ref('')
const dropGroup = ref(null)
const dropAt = reactive({ key: null, before: true })
const drag = reactive({ kind: null, key: null })
const renamingFolder = ref(null)
const newFolderId = ref(null)
const folderToDelete = ref(null)

/* Сенсорный экран жеста перетаскивания не даёт (HTML5 DnD там просто молчит),
   поэтому те же действия дублируются пунктами меню. Сам атрибут `draggable`
   при этом не снимаем: «primary pointer coarse» ошибается на гибридах —
   ноутбук с сенсорным экраном терял перенос мышью. */
const coarse = typeof window !== 'undefined' && window.matchMedia
  ? window.matchMedia('(pointer: coarse)').matches
  : false
const canDragItem = computed(() => props.organizable)

const sectionPrefs = computed(() => prefs.section(props.section))
const hasFolders = computed(() => sectionPrefs.value.folders.length > 0)

const shown = computed(() => {
  const q = filter.value.trim().toLowerCase()
  if (!q) return props.items
  return props.items.filter((i) => String(i[props.labelKey] ?? '').toLowerCase().includes(q))
})
const visibleCount = computed(() => shown.value.length)
const showFilter = computed(() => props.filterable && props.items.length >= 8)

const groups = computed(() => organizeList(shown.value, sectionPrefs.value, props.idKey))
/* Порядок сохраняем по ПОЛНОМУ списку, а не по отфильтрованному: иначе
   перестановка при активном фильтре выкинула бы скрытые пункты в конец. */
const fullOrder = computed(() =>
  flattenGroups(organizeList(props.items, sectionPrefs.value, props.idKey), props.idKey),
)

function isSelected(item) {
  return props.selectedId != null && String(props.selectedId) === keyOf(item, props.idKey)
}
function isPinned(item) {
  return prefs.isPinned(props.section, keyOf(item, props.idKey))
}
function groupIcon(g) {
  return g.kind === 'pinned' ? 'keep' : g.kind === 'folder' ? 'folder' : 'inbox'
}
function itemClass(item) {
  const key = keyOf(item, props.idKey)
  return {
    dragging: drag.kind === 'item' && drag.key === key,
    'drop-before': dropAt.key === key && dropAt.before,
    'drop-after': dropAt.key === key && !dropAt.before,
  }
}

function toggleGroup(g) {
  prefs.toggleCollapsed(props.section, g.collapseKey)
}

// ── Перетаскивание ────────────────────────────────────────────────
// Свой тип данных отличает наши пункты от чужой ноши (запись ежедневника,
// файл): чужую мы не перехватываем, а отдаём разделу событиями.
const DRAG_TYPE = 'application/x-gw-entity'

/* Ушли из списка целиком (курсор за его границами) — линия вставки не нужна.
   Проверяем по относительной цели: переход между строками её не считает. */
function onListDragLeave(ev) {
  if (drag.kind !== 'item') return
  if (!ev.currentTarget.contains(ev.relatedTarget)) dropAt.key = null
}

function endDrag() {
  drag.kind = null
  drag.key = null
  dropAt.key = null
  dropGroup.value = null
}

function ours(ev) {
  return drag.kind !== null || Array.from(ev.dataTransfer?.types || []).includes(DRAG_TYPE)
}

function onItemDragStart(ev, item) {
  if (!canDragItem.value) return
  drag.kind = 'item'
  drag.key = keyOf(item, props.idKey)
  ev.dataTransfer.effectAllowed = 'move'
  ev.dataTransfer.setData(DRAG_TYPE, drag.key)
  ev.dataTransfer.setData('text/plain', String(item[props.labelKey] ?? ''))
}

/* Дроп разрешаем БЕЗУСЛОВНО, как только тащат наш пункт, — даже над самим
   собой: строка состоит из вложенных элементов, и по пути курсора браузер
   успевает прислать dragleave между dragover'ами. Ставить `preventDefault`
   только «над правильным соседом» означало ловить эту гонку и время от времени
   получать перечёркнутый курсор вместо переноса. */
function onItemDragOver(ev, item) {
  if (!ours(ev)) { emit('item-dragover', ev, item); return }
  if (drag.kind !== 'item') return
  ev.preventDefault()
  ev.dataTransfer.dropEffect = 'move'
  const key = keyOf(item, props.idKey)
  if (key === drag.key) { dropAt.key = null; return }
  const r = ev.currentTarget.getBoundingClientRect()
  dropAt.key = key
  dropAt.before = ev.clientY < r.top + r.height / 2
}

// Подсветку места вставки снимает только уход из САМОГО списка: dragleave по
// вложенным элементам строки — обычное дело и к отмене отношения не имеет.
function onItemDragLeave(ev, item) {
  if (!ours(ev)) emit('item-dragleave', ev, item)
}

function onItemDrop(ev, item) {
  if (!ours(ev)) { emit('item-drop', ev, item); return }
  if (drag.kind !== 'item') { endDrag(); return }
  ev.preventDefault()
  ev.stopPropagation()
  const moving = drag.key
  // Цель — пункт, НА который отпустили; подсказка dropAt лишь уточняет сторону.
  const target = keyOf(item, props.idKey)
  const before = dropAt.key === target ? dropAt.before : false
  const group = groupOf(target)
  endDrag()
  if (moving === target) return
  // Пункт переезжает в ГРУППУ соседа: перетащив в папку, человек ждёт, что он
  // там и останется, а не просто встанет рядом в списке.
  applyGroup(moving, group)
  prefs.setOrder(props.section, moveKey(fullOrder.value, moving, target, before))
}

function groupOf(key) {
  return groups.value.find((g) => g.items.some((i) => keyOf(i, props.idKey) === key)) || null
}

function applyGroup(key, group) {
  if (!group) return
  if (group.kind === 'pinned') {
    if (!prefs.isPinned(props.section, key)) prefs.togglePin(props.section, key)
    return
  }
  if (prefs.isPinned(props.section, key)) prefs.togglePin(props.section, key)
  prefs.setFolder(props.section, key, group.kind === 'folder' ? group.id : '')
}

function onFolderDragStart(ev, g) {
  if (g.kind !== 'folder') return
  drag.kind = 'folder'
  drag.key = g.id
  ev.dataTransfer.effectAllowed = 'move'
  ev.dataTransfer.setData(DRAG_TYPE, `f:${g.id}`)
}

function onGroupDragOver(ev, g) {
  if (drag.kind === 'item') {
    ev.preventDefault()
    ev.dataTransfer.dropEffect = 'move'
    dropGroup.value = g.key
    dropAt.key = null
    return
  }
  if (drag.kind === 'folder' && g.kind === 'folder' && g.id !== drag.key) {
    ev.preventDefault()
    dropGroup.value = g.key
  }
}

function onGroupDrop(ev, g) {
  if (drag.kind === 'item') {
    ev.preventDefault()
    ev.stopPropagation()
    const moving = drag.key
    endDrag()
    applyGroup(moving, g)
    // В конец группы: заголовок — это «положить сюда», а не «встать первым».
    const last = g.items.length ? keyOf(g.items[g.items.length - 1], props.idKey) : null
    if (last && last !== moving) prefs.setOrder(props.section, moveKey(fullOrder.value, moving, last, false))
    return
  }
  if (drag.kind === 'folder' && g.kind === 'folder') {
    ev.preventDefault()
    const moving = drag.key
    endDrag()
    prefs.moveFolder(props.section, moving, g.id, true)
  }
}

// ── Папки ─────────────────────────────────────────────────────────
function createFolder() {
  // Новая папка сразу открывается на переименование: диалог ради одного поля
  // здесь лишний, а имя всё равно вводят немедленно.
  const id = prefs.addFolder(props.section, 'Новая папка')
  newFolderId.value = id
  renamingFolder.value = id
}

function saveFolderName(g, name) {
  prefs.renameFolder(props.section, g.id, name.trim() || g.name)
  renamingFolder.value = null
  newFolderId.value = null
}

function cancelFolderName(g) {
  // Отмена у ТОЛЬКО ЧТО созданной папки убирает её саму — иначе в списке
  // оставалась бы «Новая папка», которую человек заводить передумал.
  if (newFolderId.value === g.id) prefs.removeFolder(props.section, g.id)
  renamingFolder.value = null
  newFolderId.value = null
}

function doRemoveFolder() {
  const g = folderToDelete.value
  folderToDelete.value = null
  if (g) prefs.removeFolder(props.section, g.id)
}

// ── Контекстное меню ──────────────────────────────────────────────
const menu = reactive({ open: false, x: 0, y: 0, item: null, folder: null })

function openItemMenu(item, ev) {
  menu.item = item
  menu.folder = null
  // Пустое меню не открываем: без организации списка и без действий раздела
  // (чужая запись) показывать нечего, а пустая панель у курсора — брак.
  if (!menuItemsList.value.length) { menu.item = null; return }
  menu.x = ev.clientX
  menu.y = ev.clientY
  menu.open = true
}

function openFolderMenu(g, ev) {
  menu.item = null
  menu.folder = g
  const r = ev.currentTarget?.getBoundingClientRect?.()
  menu.x = ev.clientX || r?.left || 0
  menu.y = ev.clientY || (r ? r.bottom + 4 : 0)
  menu.open = true
}

const menuItemsList = computed(() => {
  if (menu.folder) {
    return [
      { label: 'Переименовать папку', icon: 'drive_file_rename_outline', action: 'folder:rename' },
      { label: 'Удалить папку', icon: 'folder_delete', action: 'folder:remove', danger: true },
    ]
  }
  const item = menu.item
  if (!item) return []
  const key = keyOf(item, props.idKey)
  const own = props.menuItems ? props.menuItems(item) || [] : []
  if (!props.organizable) return own
  const pinned = prefs.isPinned(props.section, key)
  const group = groupOf(key)
  const folders = sectionPrefs.value.folders
  const organize = [
    { label: pinned ? 'Открепить' : 'Закрепить', icon: pinned ? 'keep_off' : 'keep', action: 'org:pin' },
    {
      label: 'Переместить в…',
      icon: 'drive_file_move',
      children: [
        ...folders.map((f) => ({
          label: f.name,
          icon: group?.id === f.id ? 'check' : 'folder',
          action: `org:folder:${f.id}`,
        })),
        ...(group?.kind === 'folder' ? [{ label: 'Убрать из папки', icon: 'folder_off', action: 'org:folder:' }] : []),
        { label: 'Новая папка…', icon: 'create_new_folder', action: 'org:newfolder' },
      ],
    },
    ...(coarse
      ? [
        { label: 'Выше', icon: 'arrow_upward', action: 'org:up' },
        { label: 'Ниже', icon: 'arrow_downward', action: 'org:down' },
      ]
      : []),
  ]
  return own.length ? [...organize, { divider: true }, ...own] : organize
})

function onMenuSelect(action) {
  menu.open = false
  if (menu.folder) {
    const g = menu.folder
    if (action === 'folder:rename') renamingFolder.value = g.id
    if (action === 'folder:remove') folderToDelete.value = g
    return
  }
  const item = menu.item
  if (!item) return
  const key = keyOf(item, props.idKey)
  if (action === 'org:pin') return prefs.togglePin(props.section, key)
  if (action === 'org:newfolder') {
    const id = prefs.addFolder(props.section, 'Новая папка')
    prefs.setFolder(props.section, key, id)
    newFolderId.value = id
    renamingFolder.value = id
    return
  }
  if (action?.startsWith('org:folder:')) return prefs.setFolder(props.section, key, action.slice(11))
  if (action === 'org:up' || action === 'org:down') {
    const group = groupOf(key)
    if (!group) return
    const keys = group.items.map((i) => keyOf(i, props.idKey))
    return prefs.setOrder(props.section, shiftKey(fullOrder.value, key, action === 'org:up' ? -1 : 1, keys))
  }
  emit('command', action, item)
}

// Сменилась область (вкладка) — фильтр относится к прежнему набору.
watch(() => props.scope, () => { filter.value = '' })
</script>

<style scoped>
/* Тело панели прокручивается само, а подвал с кнопкой остаётся на месте —
   поэтому AppPage идёт без своего скролла (:scroll="false"). */
.el {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 0;
}

.el-filter { flex: none; padding-bottom: 8px; }

.el-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding-bottom: 8px;
}

.el-group + .el-group { margin-top: 10px; }

.el-ghead {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 30px;
  padding: 0 4px 0 2px;
  border-radius: var(--radius-md);
}

.el-ghead.folder { cursor: grab; }
.el-ghead.drop { background: color-mix(in oklch, var(--color-primary) 14%, transparent); }

.el-gtoggle {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
  padding: 4px 2px;
  border: none;
  background: none;
  color: var(--color-text-dim);
  font: inherit;
  font-size: 12px;
  font-weight: 700;
  letter-spacing: 0.02em;
  text-align: left;
  cursor: pointer;
}

.el-gtoggle:hover { color: var(--color-text); }
.el-gchev { font-size: 18px; transition: transform 0.18s ease; }
.el-gchev.closed { transform: rotate(-90deg); }
.el-gicon { font-size: 17px; }

.el-gname {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.el-gcount { flex: none; font-variant-numeric: tabular-nums; opacity: 0.7; }

.el-items { padding: 2px 0 0; }

/* Место вставки показываем линией, а не сдвигом строк: список не «дёргается»
   под курсором, а видно ровно то место, куда пункт встанет. */
.el-item { position: relative; }
.el-item.dragging { opacity: 0.45; }

.el-item.drop-before::before,
.el-item.drop-after::after {
  content: '';
  position: absolute;
  left: 6px;
  right: 6px;
  height: 2px;
  border-radius: var(--radius-full);
  background: var(--color-primary);
}

.el-item.drop-before::before { top: -4px; }
.el-item.drop-after::after { bottom: -4px; }

.el-pin { font-size: 17px; color: var(--color-primary); }

.el-drop-hint {
  margin: 2px 4px;
  padding: 10px;
  border: 1px dashed var(--color-outline-dim);
  border-radius: var(--radius-md);
  color: var(--color-text-dim);
  font-size: 12px;
  text-align: center;
}

.el-drop-hint.drop { border-color: var(--color-primary); color: var(--color-primary); }

.el-nomatch { margin: 12px 4px; color: var(--color-text-dim); font-size: 13px; text-align: center; }

.el-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 10px;
  border-top: 1px solid var(--acrylic-border);
}

.el-foot > :deep(.btn:not(.v-icon)) { flex: 1; min-width: 0; }
</style>
