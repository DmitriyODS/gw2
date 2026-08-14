<template>
  <div class="rr">
    <!-- Подразделы: варианты спискового поля, назначенного источником. Вкладка
         «Все» показывает записи целиком; в узкой панели ряд прокручивается. -->
    <AppTabs
      v-if="sections.length"
      class="rr-sections"
      :model-value="section"
      :tabs="sectionTabs"
      variant="tint"
      dense
      @update:model-value="$emit('update:section', $event)"
    />

    <!-- Карточки: сортировка контролом — заголовков колонок там нет. -->
    <div v-if="asCards && fields.length" class="rr-sortbar">
      <span class="material-symbols-outlined">sort</span>
      <Select
        class="rr-sortbar-select"
        :model-value="sort"
        :options="sortOptions"
        option-label="label"
        option-value="value"
        @update:model-value="pickSort"
      />
      <AppButton
        variant="icon"
        size="sm"
        :icon="order === 'asc' ? 'arrow_upward' : 'arrow_downward'"
        :title="order === 'asc' ? 'По возрастанию' : 'По убыванию'"
        aria-label="Направление сортировки"
        @click="flipOrder"
      />
    </div>

    <div class="rr-box">
      <div v-if="!asCards" class="rr-scroll">
        <table class="rr-table" :class="{ sized: hasWidths }">
          <!-- Ширины задаются colgroup'ом: так они не зависят от содержимого
               ячеек и не «плывут» при перелистывании страниц. -->
          <colgroup>
            <col class="rr-col-check" />
            <col v-for="f in fields" :key="f.id" :style="colStyle(f.id)" />
            <col v-if="accounting" />
            <col class="rr-col-date" />
          </colgroup>
          <thead>
            <tr>
              <th class="rr-th-check">
                <Checkbox
                  :model-value="allSelected"
                  binary
                  :disabled="!records.length"
                  @update:model-value="$emit('toggle-all')"
                />
              </th>
              <!-- Колонки переставляются перетаскиванием заголовка. Тянем сам
                   th, а не отдельную рукоятку: в шапке таблицы ей негде встать,
                   а текст там и так не выделяется. -->
              <th
                v-for="f in fields"
                :key="f.id"
                :data-col-id="f.id"
                :class="{ dragging: dragCol === f.id, over: overCol === f.id }"
                draggable="true"
                @dragstart="onColDragStart(f.id, $event)"
                @dragover.prevent="onColDragOver(f.id)"
                @dragleave="onColDragLeave(f.id)"
                @drop.prevent="onColDrop(f.id)"
                @dragend="onColDragEnd"
              >
                <span class="rr-th-inner">
                  <button
                    class="rr-th-sort"
                    type="button"
                    :title="f.label"
                    :disabled="!isSortable(f.type)"
                    @click="isSortable(f.type) && headerSort(String(f.id))"
                  >
                    <!-- Название сжимается многоточием, стрелка и воронка — нет:
                         иначе узкая колонка выталкивала кнопку фильтра из
                         заголовка, и фильтр становился недоступен. -->
                    <span class="rr-th-text">{{ f.label }}</span>
                    <span v-if="sort === String(f.id)" class="material-symbols-outlined rr-sort">
                      {{ order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                    </span>
                  </button>
                  <!-- Фильтр колонки: воронка загорается, когда условие задано. -->
                  <button
                    class="rr-th-filter"
                    :class="{ on: !!filterOf(f.id) }"
                    type="button"
                    title="Фильтр по колонке"
                    aria-label="Фильтр по колонке"
                    @click.stop="openFilter(f, $event)"
                  >
                    <span class="material-symbols-outlined">filter_alt</span>
                  </button>
                </span>
                <!-- Правый край колонки: за него её и растягивают. Двойной клик
                     возвращает автоматическую ширину. -->
                <span
                  class="rr-th-resize"
                  :class="{ active: resizeId === f.id }"
                  title="Потяните, чтобы изменить ширину"
                  @pointerdown.stop.prevent="startResize(f.id, $event)"
                  @dblclick.stop="$emit('reset-widths')"
                  @dragstart.prevent
                />
              </th>
              <th v-if="accounting" class="rr-th-state">Состояние</th>
              <th class="rr-th-date sortable" @click="headerSort('created_at')">
                <span class="rr-th-inner">
                  Создано
                  <span v-if="sort === 'created_at'" class="material-symbols-outlined rr-sort">
                    {{ order === 'asc' ? 'arrow_upward' : 'arrow_downward' }}
                  </span>
                </span>
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="rec in records"
              :key="rec.id"
              class="rr-row"
              :class="{ selected: isSelected(rec.id) }"
              @click="$emit('open', rec)"
              @contextmenu.prevent="openRowMenu(rec, $event)"
            >
              <td class="rr-td-check" @click.stop>
                <Checkbox
                  :model-value="isSelected(rec.id)"
                  binary
                  @update:model-value="$emit('toggle', rec.id)"
                />
              </td>
              <td v-for="f in fields" :key="f.id">
                <button
                  v-if="f.type === 'image' && thumbSrc(rec.data?.[String(f.id)])"
                  class="rr-thumb"
                  type="button"
                  title="Открыть картинку"
                  @click.stop="openImage(f, rec)"
                >
                  <img
                    :src="thumbSrc(rec.data?.[String(f.id)])"
                    :alt="rec.data?.[String(f.id)]?.name || ''"
                    loading="lazy"
                    decoding="async"
                  />
                </button>
                <span
                  v-else-if="f.type === 'stock'"
                  class="rr-stock"
                  :class="{ taken: !!rec.data?.[String(f.id)]?.taken }"
                >{{ textValue(f, rec.data?.[String(f.id)]) }}</span>
                <span v-else class="rr-cell">{{ textValue(f, rec.data?.[String(f.id)]) }}</span>
              </td>
              <td v-if="accounting" class="rr-td-state">
                <AppChip :label="stateLabel(rec)" :tone="stateTone(rec)" />
              </td>
              <td class="rr-td-date">{{ shortDate(rec.created_at) }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Карточки — «галерея» записи: обложка сверху, первичное поле
           заголовком, ниже несколько поддерживающих полей, служебное — в
           подвал. Пустые поля НЕ показываем: прочерк в каждой строке
           превращает карточку в ту же таблицу, только хуже читаемую. -->
      <div v-else class="rr-cards-scroll">
        <label v-if="records.length" class="rr-cards-selall">
          <Checkbox :model-value="allSelected" binary @update:model-value="$emit('toggle-all')" />
          <span>Выбрать все на странице</span>
        </label>

        <div class="rr-grid">
          <article
            v-for="rec in records"
            :key="rec.id"
            class="rr-card glass-hover"
            :class="{ selected: isSelected(rec.id) }"
            tabindex="0"
            @click="$emit('open', rec)"
            @keydown.enter="$emit('open', rec)"
            @contextmenu.prevent="openRowMenu(rec, $event)"
          >
            <!-- Обложка держит ПОСТОЯННУЮ пропорцию: разновысокие картинки
                 рвали бы ряд, а карточки в сетке должны стоять ровно. Её место
                 резервируется, только если обложка в реестре вообще есть. -->
            <div v-if="hasCovers" class="rr-card-cover">
              <button
                v-if="coverOf(rec)"
                class="rr-card-cover-btn"
                type="button"
                title="Открыть картинку"
                @click.stop="openImage(coverField, rec)"
              >
                <img :src="coverOf(rec)" :alt="cardTitle(rec)" loading="lazy" decoding="async" />
              </button>
              <span v-else class="rr-card-cover-empty material-symbols-outlined">image</span>
            </div>

            <span class="rr-card-check" :class="{ shown: isSelected(rec.id) }" @click.stop>
              <Checkbox
                :model-value="isSelected(rec.id)"
                binary
                @update:model-value="$emit('toggle', rec.id)"
              />
            </span>

            <div class="rr-card-body">
              <h3 class="rr-card-title">{{ cardTitle(rec) }}</h3>

              <dl v-if="cardRows(rec).length" class="rr-card-fields">
                <div v-for="row in cardRows(rec)" :key="row.id" class="rr-card-field">
                  <dt>{{ row.label }}</dt>
                  <dd>{{ row.value }}</dd>
                </div>
              </dl>

              <footer class="rr-card-foot">
                <AppChip v-if="accounting" :label="stateLabel(rec)" :tone="stateTone(rec)" />
                <span class="rr-card-date">{{ shortDate(rec.created_at) }}</span>
              </footer>
            </div>
          </article>
        </div>
      </div>

      <div v-if="loading" class="rr-overlay">
        <BrandLoader :size="48" />
      </div>
      <EmptyState
        v-else-if="!records.length"
        class="rr-empty"
        icon="inbox"
        tone="soft"
        :title="search ? 'Ничего не найдено' : 'Записей пока нет'"
        :subtitle="search ? 'Попробуйте другой запрос.' : emptyHint"
      />

      <!-- Выбор переживает страницы, поэтому плашка висит над списком. Своих
           действий в ней намеренно нет: только счётчик, «выбрать всё» и сброс —
           массовые операции живут в командах раздела. -->
      <AppSelectionBar
        :count="selectionCount"
        :total="total"
        :all-selected="selectionAll"
        @select-all="$emit('select-all-matching')"
        @clear="$emit('clear-selection')"
      />
    </div>

    <!-- Клик по превью открывает картинку целиком, не трогая карточку записи. -->
    <ImageLightbox v-model="lightbox" :src="lightboxSrc" :caption="lightboxCaption" />

    <ContextMenu
      :visible="rowMenuOpen"
      :x="rowMenuX"
      :y="rowMenuY"
      :items="rowMenuItems"
      @select="onRowMenu"
      @close="rowMenuOpen = false"
    />

    <RegistryColumnFilter
      v-model="filterOpen"
      :field="filterField"
      :filter="filterField ? filterOf(filterField.id) : null"
      :anchor="filterAnchor"
      @apply="applyFilter"
    />
  </div>
</template>

<script setup>
/* Записи реестра: широко — таблица с сортировкой по заголовкам, тесно —
   карточки и сортировка отдельным контролом. Один и тот же вид показывают
   раздел «Реестры» и публичная страница внешней ссылки, поэтому он живёт
   здесь, а данные и запросы остаются у них: сюда приходят готовые записи,
   отсюда уходят намерения (открыть, выбрать, пересортировать).

   Узкую раскладку задаёт вызывающий: она про ширину ПАНЕЛИ (раздел живёт
   окном рабочего стола), а не про ширину экрана. */
import { computed, ref } from 'vue'
import Checkbox from 'primevue/checkbox'
import Select from 'primevue/select'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppSelectionBar from '@/components/ui/AppSelectionBar.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import ImageLightbox from '@/components/common/ImageLightbox.vue'
import RegistryColumnFilter from './RegistryColumnFilter.vue'
import { fileUrl, isSortable, textValue, thumbUrl } from '@/utils/registryFields.js'

const props = defineProps({
  /** Видимые поля в порядке реестра (см. useRegistryColumns). */
  fields: { type: Array, default: () => [] },
  records: { type: Array, default: () => [] },
  loading: { type: Boolean, default: false },
  /** Ключ сортировки: 'created_at' либо строковый id поля. */
  sort: { type: String, default: 'created_at' },
  order: { type: String, default: 'desc' },
  /** Отмечена ли запись — предикат из useRowSelection (выбор живёт между
      страницами, поэтому множеством id он не описывается). */
  isSelected: { type: Function, default: () => false },
  /** Вся текущая страница отмечена — состояние галочки в шапке. */
  allSelected: { type: Boolean, default: false },
  /** Сколько записей отмечено всего и сколько их по фильтру — для плашки. */
  selectionCount: { type: Number, default: 0 },
  total: { type: Number, default: 0 },
  /** Выбрано всё по фильтру — предлагать «выбрать все» больше нечего. */
  selectionAll: { type: Boolean, default: false },
  /** Тесная панель: таблица в неё не помещается, поэтому там всегда карточки. */
  narrow: { type: Boolean, default: false },
  /** Вид, выбранный человеком: 'table' | 'cards'. В тесной панели не влияет. */
  view: { type: String, default: 'table' },
  /** Активный поисковый запрос — от него зависит текст пустого состояния. */
  search: { type: String, default: '' },
  /** Сам реестр — нужен учётному режиму (плашки состояния позиций). */
  registry: { type: Object, default: null },
  /** Варианты поля-источника подразделов (вкладки) и выбранная из них. */
  sections: { type: Array, default: () => [] },
  section: { type: String, default: '' },
  /** Активные фильтры колонок: [{field_id, op, values}]. */
  columnFilters: { type: Array, default: () => [] },
  /** Ширины колонок в пикселях по id поля (пусто — раскладка автоматическая). */
  widths: { type: Object, default: () => ({}) },
  /** Можно ли вести записи — от этого зависит состав контекстного меню. */
  canEdit: { type: Boolean, default: false },
  /** Доступен ли учётный режим: по внешней ссылке выдач нет вовсе. */
  allowManage: { type: Boolean, default: true },
  emptyHint: { type: String, default: '' },
})

const emit = defineEmits([
  'update:sort', 'update:section', 'update:column-filter',
  'move-column', 'resize-columns', 'reset-widths',
  'open', 'edit', 'manage', 'remove', 'toggle', 'toggle-all',
  'select-all-matching', 'clear-selection',
])

const accounting = computed(() => !!props.registry?.accounting)

/* Карточки — либо по выбору человека, либо вынужденно: в тесной панели таблица
   не помещается, и выбор вида там ничего не решает. */
const asCards = computed(() => props.narrow || props.view === 'cards')

// Вкладка «Все» идёт первой и означает «без фильтра» — пустое значение.
const sectionTabs = computed(() => [
  { value: '', label: 'Все' },
  ...props.sections.map((s) => ({ value: s, label: s })),
])

/* ── Состояние позиции (учётный реестр) ──
   Открытую выдачу присылает сервер вместе с записью, он же считает просрочку:
   часы клиента не должны решать, просрочена ли вещь. */
function stateLabel(rec) {
  const issue = rec.issue
  if (!issue) return 'В наличии'
  // В плашке — ПОЛУЧАТЕЛЬ: он отвечает на вопрос «куда ушло», а ответственного
  // видно в карточке выдачи.
  if (!issue.due_at) return `У «${issue.issued_to || issue.holder_name}»`
  const overdue = overdueDays(issue.due_at)
  if (overdue > 0) return `Просрочено на ${overdue} дн.`
  return `Выдано до ${new Date(issue.due_at).toLocaleDateString('ru-RU')}`
}

function stateTone(rec) {
  const issue = rec.issue
  if (!issue) return 'success'
  if (!issue.due_at) return 'neutral'
  return overdueDays(issue.due_at) > 0 ? 'danger' : 'warning'
}

function overdueDays(due) {
  const diff = Date.now() - new Date(due).getTime()
  return diff > 0 ? Math.ceil(diff / 86400000) : 0
}

// ── Контекстное меню записи ──
const rowMenuOpen = ref(false)
const rowMenuX = ref(0)
const rowMenuY = ref(0)
const rowMenuTarget = ref(null)

/* Пункты описываются полем `action` — по нему ContextMenu и опознаёт
   выполнимый пункт: без него он молча закрывается, ничего не сообщая наверх. */
const rowMenuItems = computed(() => [
  { label: 'Открыть', icon: 'open_in_full', action: 'open' },
  ...(props.canEdit
    ? [{ label: 'Редактировать', icon: 'edit', action: 'edit' }]
    : []),
  ...(accounting.value && props.canEdit && props.allowManage
    ? [{ label: 'Управлять', icon: 'inventory', action: 'manage' }]
    : []),
  ...(props.canEdit
    ? [{ divider: true }, { label: 'Удалить', icon: 'delete', danger: true, action: 'remove' }]
    : []),
])

function openRowMenu(rec, e) {
  rowMenuTarget.value = rec
  rowMenuX.value = e.clientX
  rowMenuY.value = e.clientY
  rowMenuOpen.value = true
}

function onRowMenu(action) {
  const rec = rowMenuTarget.value
  rowMenuOpen.value = false
  if (rec) emit(action, rec)
}

/* ── Ширина колонок ──
   Первое же растягивание переводит таблицу в ФИКСИРОВАННУЮ раскладку, поэтому
   вместе с тянутой колонкой запоминаются текущие ширины остальных: иначе они
   схлопнулись бы до содержимого. Во время жеста ширина живёт локально
   (draftWidths) и уходит наверх один раз — на отпускании, чтобы не писать в
   localStorage на каждый пиксель. */
const MIN_COL_WIDTH = 72
const resizeId = ref(null)
const draftWidths = ref({})

const hasWidths = computed(() =>
  Object.keys(props.widths).length > 0 || Object.keys(draftWidths.value).length > 0)

function colStyle(id) {
  const w = draftWidths.value[id] ?? props.widths[id]
  return w ? { width: `${w}px` } : {}
}

function startResize(id, e) {
  const th = e.currentTarget.closest('th')
  if (!th) return
  // Снимок ВСЕХ колонок: дальше таблица станет фиксированной.
  const snapshot = {}
  for (const cell of th.parentElement.children) {
    const colId = cell.dataset.colId
    if (colId) snapshot[colId] = cell.getBoundingClientRect().width
  }
  const startX = e.clientX
  const startWidth = th.getBoundingClientRect().width
  resizeId.value = id
  draftWidths.value = { ...props.widths, ...snapshot }

  const move = (ev) => {
    const next = Math.max(MIN_COL_WIDTH, Math.round(startWidth + (ev.clientX - startX)))
    draftWidths.value = { ...draftWidths.value, [id]: next }
  }
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
    emit('resize-columns', { ...draftWidths.value })
    draftWidths.value = {}
    resizeId.value = null
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
}

/* ── Перестановка колонок ──
   Перетаскивается ИДЕНТИФИКАТОР колонки, а не индекс: пока тянут, набор мог
   поменяться (пришло сокет-событие о правке структуры), и индекс указал бы уже
   на другое поле. Сам порядок хранит useRegistryColumns — это личная настройка
   устройства, как и состав колонок. */
const dragCol = ref(null)
const overCol = ref(null)

function onColDragStart(id, e) {
  dragCol.value = id
  e.dataTransfer.effectAllowed = 'move'
  // Safari не начинает перетаскивание без данных в буфере.
  e.dataTransfer.setData('text/plain', String(id))
}

function onColDragOver(id) {
  if (dragCol.value !== null && id !== dragCol.value) overCol.value = id
}

function onColDragLeave(id) {
  if (overCol.value === id) overCol.value = null
}

function onColDrop(id) {
  const from = dragCol.value
  onColDragEnd()
  if (from !== null && from !== id) emit('move-column', from, id)
}

function onColDragEnd() {
  dragCol.value = null
  overCol.value = null
}

// ── Фильтры колонок ──
const filterOpen = ref(false)
const filterField = ref(null)
const filterAnchor = ref(null)

function filterOf(fieldId) {
  return props.columnFilters.find((f) => f.field_id === fieldId) || null
}

function openFilter(field, e) {
  filterField.value = field
  filterAnchor.value = e.currentTarget
  filterOpen.value = true
}

function applyFilter(filter) {
  filterOpen.value = false
  emit('update:column-filter', filterField.value.id, filter)
}

/* Сортировку решаем здесь целиком: клик по заголовку той же колонки
   переворачивает порядок, выбор в узком контроле его сохраняет. Наружу уходит
   готовая пара — вызывающему остаётся перечитать первую страницу. */
function headerSort(key) {
  if (props.sort === key) emit('update:sort', { sort: key, order: props.order === 'asc' ? 'desc' : 'asc' })
  else emit('update:sort', { sort: key, order: 'asc' })
}
function pickSort(key) {
  emit('update:sort', { sort: key, order: props.order })
}
function flipOrder() {
  emit('update:sort', { sort: props.sort, order: props.order === 'asc' ? 'desc' : 'asc' })
}

const sortOptions = computed(() => {
  const opts = [{ value: 'created_at', label: 'Дате создания' }]
  for (const f of props.fields) {
    if (isSortable(f.type)) opts.push({ value: String(f.id), label: f.label })
  }
  return opts
})

/* На карточке первое видимое поле — заголовок, остальные — тело. Обложка
   заголовком быть не может: в строке от неё осталось бы имя файла, а показать
   её самоё гораздо полезнее — она уходит наверх карточки картинкой. */
const titleField = computed(() => props.fields.find((f) => f.type !== 'image') || null)
const coverField = computed(() => props.fields.find((f) => f.type === 'image') || null)

/* В теле карточки — несколько поддерживающих полей, а не все подряд: карточка
   должна оставаться карточкой, а не таблицей боком. Берём первые НЕПУСТЫЕ —
   так три коротких значения полезнее, чем три прочерка и одно значение. */
const CARD_ROWS_MAX = 3
const cardFields = computed(() => props.fields
  .filter((f) => f !== titleField.value && f.type !== 'image' && f.type !== 'file'))

function cardRows(rec) {
  const out = []
  for (const f of cardFields.value) {
    const value = textValue(f, rec.data?.[String(f.id)])
    if (!value) continue
    out.push({ id: f.id, label: f.label, value })
    if (out.length === CARD_ROWS_MAX) break
  }
  return out
}

// Место под обложку резервируем на весь реестр, а не по записи: иначе карточки
// без картинки «подпрыгивали» бы в ряду относительно соседей.
const hasCovers = computed(() => !!coverField.value)

function coverOf(rec) {
  const f = coverField.value
  return f ? thumbSrc(rec.data?.[String(f.id)]) : ''
}

function cardTitle(rec) {
  const f = titleField.value
  const v = f ? textValue(f, rec.data?.[String(f.id)]) : ''
  return v || `Запись #${rec.id}`
}

function shortDate(v) {
  if (!v) return ''
  const d = new Date(v)
  return isNaN(d.getTime()) ? '' : d.toLocaleDateString('ru-RU')
}

/* Превью картинок прямо в строке: до него о содержимом поля говорило только имя
   файла. В таблице открывается лайтбокс с оригиналом — карточку записи ради
   одной картинки открывать незачем. */
const lightbox = ref(false)
const lightboxSrc = ref('')
const lightboxCaption = ref('')

const thumbSrc = (value) => thumbUrl(value)

function openImage(field, rec) {
  const value = rec.data?.[String(field.id)]
  lightboxSrc.value = fileUrl(value)
  lightboxCaption.value = value?.name || field.label
  lightbox.value = true
}
</script>

<style scoped>
.rr {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
}

/* ── Подразделы над таблицей ── */
.rr-sections {
  flex: none;
  /* Сверху почти вплотную к строке поиска: свой отступ шапка уже дала, и два
     подряд читались как провал между поиском и вкладками. */
  margin: 2px 14px 10px;
  /* Вкладок бывает больше, чем влезает: AppTabs прокручивает их сам, но для
     этого ему нужна конечная ширина, а не «сколько попросит содержимое». */
  max-width: calc(100% - 28px);
}

/* ── Сортировка в узкой раскладке ── */
.rr-sortbar {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 14px;
  border-bottom: 1px solid var(--color-outline-dim);
  color: var(--color-text-dim);
}

.rr-sortbar > .material-symbols-outlined { font-size: 20px; }
.rr-sortbar-select { flex: 1; min-width: 0; }

/* ── Таблица: собственный скролл, sticky-шапка ── */
.rr-box { position: relative; flex: 1; min-height: 0; display: flex; }
.rr-scroll { position: relative; flex: 1; min-height: 0; overflow: auto; }

.rr-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 14px;
}

/* Строки уезжают под шапку, поэтому фон ей нужен ПЛОТНЫЙ и без backdrop-filter:
   панель раздела сама backdrop root — размывать за шапкой всё равно нечего,
   зато blur выносил её в отдельный composited-слой, и после прокрутки клик по
   галочке в шапке проваливался в строку под ней. Матовость даёт «иней». */
.rr-table thead th {
  position: sticky;
  top: 0;
  z-index: 3;
  padding: 12px 18px 12px 14px;
  border-bottom: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  background: var(--glass-bg), var(--color-surface);
  text-align: left;
  font-weight: 700;
  color: var(--color-text);
  white-space: nowrap;
  user-select: none;
}

/* Перетаскиваемый заголовок: сам бледнеет, а цель показывает, куда встанет
   колонка, — полосой по её левому краю. */
.rr-table thead th[draggable='true'] { cursor: grab; }
.rr-table thead th.dragging { opacity: 0.45; }

.rr-table thead th.over {
  box-shadow: inset 2px 0 0 var(--color-primary);
}

.rr-table thead th.sortable { cursor: pointer; }
.rr-table thead th.sortable:hover { color: var(--color-primary); }
.rr-th-inner { display: flex; align-items: center; gap: 4px; min-width: 0; }
.rr-sort { font-size: 16px; }

/* Заголовок колонки — две кнопки: сама сортировка и воронка фильтра. Кнопки, а
   не кликабельный th: у несортируемой колонки фильтр остаётся рабочим, и путать
   эти два действия одним хитбоксом нельзя. */
.rr-th-sort {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
  align-items: center;
  gap: 4px;
  padding: 0;
  border: none;
  background: none;
  color: inherit;
  font: inherit;
  font-weight: 700;
  text-align: left;
  cursor: pointer;
}

.rr-th-text {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rr-sort, .rr-th-filter { flex: none; }

.rr-th-sort:disabled { cursor: default; }
.rr-th-sort:not(:disabled):hover { color: var(--color-primary); }

.rr-th-filter {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  /* Круглая кнопка: без явных min/max глобальный мобильный button{min-height}
     растянул бы её в овал. */
  width: 22px; min-width: 22px; max-width: 22px;
  height: 22px; min-height: 22px; max-height: 22px;
  padding: 0;
  border: none;
  border-radius: var(--radius-full);
  background: none;
  color: var(--color-text-dim);
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s, color 0.15s, background 0.15s;
}

.rr-th-filter .material-symbols-outlined { font-size: 16px; }

/* Воронка проявляется по наведению на заголовок, но заданный фильтр виден
   всегда — иначе о нём забывают и удивляются «пропавшим» записям. */
.rr-table thead th:hover .rr-th-filter,
.rr-th-filter:focus-visible { opacity: 1; }

.rr-th-filter.on {
  opacity: 1;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

/* Состояние — плашка, и она читается как единый цветной блок: соседней дате
   нужен заметный зазор, иначе они выглядят слипшимися. Ширины хватает и на
   «Просрочено на N дн.». */
.rr-th-state, .rr-td-state { width: 220px; padding-right: 22px; }

/* Очень длинное состояние обрезаем, а не выталкиваем соседнюю колонку. */
.rr-td-state :deep(.chip) { max-width: 100%; }
/* Фиксированная раскладка включается только после первого растягивания —
   нетронутая таблица по-прежнему подбирает ширины сама. */
.rr-table.sized { table-layout: fixed; }
.rr-col-check { width: 48px; }
.rr-col-date { width: 130px; }

/* Рукоятка ширины у правого края заголовка. Шире своей полоски, чтобы в неё
   попадали, не целясь. */
.rr-th-resize {
  position: absolute;
  top: 0;
  right: 0;
  width: 10px;
  height: 100%;
  cursor: col-resize;
  touch-action: none;
}

.rr-th-resize::after {
  content: '';
  position: absolute;
  top: 25%;
  right: 3px;
  width: 2px;
  height: 50%;
  border-radius: 1px;
  background: transparent;
  transition: background 0.15s;
}

.rr-table thead th:hover .rr-th-resize::after,
.rr-th-resize.active::after { background: var(--color-outline-dim); }
.rr-th-resize.active::after { background: var(--color-primary); }

.rr-th-check, .rr-td-check { width: 48px; text-align: center; padding-left: 16px; padding-right: 0; }
.rr-th-date, .rr-td-date { width: 130px; white-space: nowrap; color: var(--color-text-dim); }

.rr-row { cursor: pointer; }
.rr-row:hover { background: var(--color-surface-high); }
.rr-row.selected { background: var(--color-primary-container); }

.rr-table tbody td {
  padding: 11px 14px;
  border-bottom: 1px solid var(--color-outline-dim);
  color: var(--color-text);
}

/* Ширину колонки задал человек — упираться в свой потолок значению незачем:
   в фиксированной раскладке его и так обрезает сама ячейка. */
.rr-table.sized .rr-cell { max-width: none; }

.rr-cell {
  display: block;
  max-width: 320px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Наличие: состояние строки видно, не читая текст. */
.rr-stock {
  display: inline-block;
  padding: 2px 10px;
  border-radius: var(--radius-full);
  background: var(--color-success-container);
  color: var(--color-on-success-container);
  font-size: 13px;
  font-weight: 600;
  white-space: nowrap;
}

.rr-stock.taken { background: var(--color-warning-container); color: var(--color-on-warning-container); }

/* Превью картинки. Размер задан всеми тремя свойствами: глобальный мобильный
   `button { min-height: 36px }` иначе растянул бы плитку. */
.rr-thumb {
  display: block;
  width: 44px; min-width: 44px;
  height: 44px; min-height: 44px;
  padding: 0;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface-low);
  overflow: hidden;
  cursor: zoom-in;
}

.rr-thumb:hover { border-color: var(--color-primary); }
.rr-thumb img { width: 100%; height: 100%; object-fit: cover; display: block; }

/* В карточке (телефон, узкая панель) превью занимает ВСЮ ширину карточки и
   становится прямоугольным: снимки почти всегда горизонтальные, а места вширь
   там достаточно — подпись поля уходит над картинкой. */
.rr-card-row.image {
  flex-direction: column;
  align-items: stretch;
  gap: 6px;
}

.rr-card-row.image .rr-card-label { width: auto; max-width: none; }

.rr-thumb.card {
  width: 100%; min-width: 0;
  height: 160px; min-height: 160px;
}

.rr-overlay {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--color-surface) 60%, transparent);
}

.rr-empty { position: absolute; inset: 0; pointer-events: none; }

/* ── Карточки записей ──
   Сетка, а не колонка: на просторе карточки идут рядами, в тесной панели
   `auto-fill` сам сводит их в одну колонку. min(240px, 100%) — иначе в панели
   уже 240px колонка перестаёт сжиматься и раздел уезжает горизонтальной
   прокруткой. Отступы кратны 8 — общий ритм интерфейса. */
.rr-cards-scroll { flex: 1; min-height: 0; overflow-y: auto; padding: 12px 14px 16px; }

.rr-cards-selall {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 2px 10px;
  font-size: 13px;
  color: var(--color-text-dim);
}

.rr-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(min(240px, 100%), 1fr));
  gap: 14px;
  align-content: start;
}

.rr-card {
  position: relative;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  cursor: pointer;
  user-select: none;
}

/* Подсветку под курсором даёт общий `.glass-hover` (как у карточек заметок):
   карточка «загорается», а не прыгает — сдвиг в плотной сетке дёргает соседей
   и мешает целиться. Здесь остаётся только рамка фокуса с клавиатуры. */
.rr-card:focus-visible {
  border-color: color-mix(in oklch, var(--color-primary) 40%, var(--acrylic-border));
  outline: none;
}

.rr-card.selected {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 1px var(--color-primary), var(--shadow-sm);
}

/* Обложка — постоянная пропорция: разновысокие картинки рвали бы ряд. */
.rr-card-cover {
  position: relative;
  aspect-ratio: 16 / 10;
  background: var(--color-surface-high);
}

.rr-card-cover-btn {
  display: block;
  width: 100%;
  height: 100%;
  padding: 0;
  border: none;
  background: none;
  cursor: zoom-in;
}

.rr-card-cover img { display: block; width: 100%; height: 100%; object-fit: cover; }

/* Карточка без картинки не должна выглядеть сломанной: место занимает
   спокойная заглушка, а не пустая дыра. */
.rr-card-cover-empty {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  font-size: 34px;
  color: var(--color-outline-dim);
}

/* Галочка появляется под курсором и остаётся видимой у отмеченных: постоянно
   висящий чекбокс в галерее только мешает смотреть на карточки. */
.rr-card-check {
  position: absolute;
  top: 8px;
  right: 8px;
  z-index: 2;
  display: inline-flex;
  padding: 3px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-surface) 85%, transparent);
  -webkit-backdrop-filter: blur(6px);
  backdrop-filter: blur(6px);
  opacity: 0;
  transition: opacity 0.15s;
}

.rr-card:hover .rr-card-check,
.rr-card:focus-within .rr-card-check,
.rr-card-check.shown { opacity: 1; }

.rr-card-body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px 12px;
}

.rr-card-title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  line-height: 1.3;
  color: var(--color-text);
  /* Длинное название не растягивает карточку: две строки и многоточие. */
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  overflow-wrap: anywhere;
}

.rr-card-fields { display: flex; flex-direction: column; gap: 5px; margin: 0; }

/* Подпись НАД значением, а не рядом: колонка подписей у разных полей разной
   ширины, и в узкой карточке значения расползались по рваной сетке. */
.rr-card-field { display: flex; flex-direction: column; gap: 1px; min-width: 0; }

.rr-card-field dt {
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.02em;
  text-transform: uppercase;
  color: var(--color-text-dim);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rr-card-field dd {
  margin: 0;
  font-size: 13.5px;
  line-height: 1.35;
  color: var(--color-text);
  /* Значение — одна строка: карточки в ряду должны оставаться одной высоты. */
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.rr-card-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: auto;
  padding-top: 8px;
  border-top: 1px solid color-mix(in oklch, var(--acrylic-border) 70%, transparent);
  font-size: 12px;
  color: var(--color-text-dim);
}

.rr-card-date { margin-left: auto; white-space: nowrap; }
</style>
