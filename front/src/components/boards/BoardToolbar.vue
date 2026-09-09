<script setup>
/* Панель инструментов холста: выбор инструмента, цвет, толщина, заливка,
   прозрачность и масштаб. Плавающая — чтобы не отъедать рабочую площадь доски.

   Инструментов стало больше, чем помещается в ряд, поэтому родственные собраны
   в ГРУППЫ (кисти, фигуры, выделение): кнопка показывает выбранный инструмент
   группы, а повторный клик по активной кнопке открывает список остальных — тот
   же приём, что в панелях графических редакторов. */
import { computed, ref } from 'vue'
import ContextMenu from '@/components/common/ContextMenu.vue'
import BoardColorPopover from '@/components/boards/BoardColorPopover.vue'
import SizePopover from '@/components/boards/SizePopover.vue'
import { ERASER_SIZES, SCENE_COLORS, STROKE_WIDTHS, TEXT_SIZES, isHexColor } from '@/utils/boardScene.js'

const props = defineProps({
  tool: { type: String, required: true },
  color: { type: String, required: true },
  fill: { type: String, default: '' },
  width: { type: Number, required: true },
  opacity: { type: Number, default: 1 },
  textSize: { type: Number, required: true },
  eraserSize: { type: Number, default: 32 },
  eraseMode: { type: String, default: 'pixel' },
  polygonSides: { type: Number, default: 5 },
  polygonStar: { type: Boolean, default: false },
  zoom: { type: Number, default: 1 },
  hasSelection: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:tool', 'update:color', 'update:fill', 'update:width', 'update:textSize',
  'update:opacity', 'update:eraserSize', 'update:eraseMode', 'update:polygonSides', 'update:polygonStar',
  'zoom-in', 'zoom-out', 'fit', 'add-image', 'delete-selected',
])

/* Каждая запись — либо одиночный инструмент, либо группа с вариантами. Значок
   группы берётся у выбранного варианта: панель показывает, чем рисуют сейчас. */
const GROUPS = [
  {
    id: 'select',
    items: [
      { key: 'select', icon: 'arrow_selector_tool', label: 'Выделение' },
      { key: 'lasso', icon: 'gesture', label: 'Лассо' },
    ],
  },
  { id: 'pan', items: [{ key: 'pan', icon: 'pan_tool', label: 'Рука' }] },
  {
    id: 'draw',
    items: [
      { key: 'pencil', icon: 'draw', label: 'Карандаш' },
      { key: 'brush', icon: 'brush', label: 'Кисть' },
      { key: 'marker', icon: 'ink_highlighter', label: 'Маркер' },
    ],
  },
  {
    id: 'vector',
    items: [
      { key: 'pen', icon: 'stylus_note', label: 'Перо (кривые Безье)' },
      { key: 'nodes', icon: 'linear_scale', label: 'Правка узлов' },
      { key: 'curve', icon: 'polyline', label: 'Кривая по точкам' },
    ],
  },
  { id: 'eraser', items: [{ key: 'eraser', icon: 'ink_eraser', label: 'Ластик' }] },
  {
    id: 'shape',
    items: [
      { key: 'line', icon: 'horizontal_rule', label: 'Линия' },
      { key: 'arrow', icon: 'arrow_right_alt', label: 'Стрелка' },
      { key: 'rect', icon: 'crop_square', label: 'Прямоугольник' },
      { key: 'ellipse', icon: 'circle', label: 'Овал' },
      { key: 'diamond', icon: 'change_history', label: 'Ромб' },
      { key: 'polygon', icon: 'pentagon', label: 'Многоугольник' },
      { key: 'star', icon: 'star', label: 'Звезда' },
    ],
  },
  { id: 'fill', items: [{ key: 'fill', icon: 'format_color_fill', label: 'Заливка' }] },
  { id: 'eyedropper', items: [{ key: 'eyedropper', icon: 'colorize', label: 'Пипетка' }] },
  { id: 'text', items: [{ key: 'text', icon: 'title', label: 'Надпись' }] },
  { id: 'sticky', items: [{ key: 'sticky', icon: 'sticky_note_2', label: 'Липкая заметка' }] },
  { id: 'comment', items: [{ key: 'comment', icon: 'add_comment', label: 'Комментарий' }] },
]

// Выбранный вариант каждой группы: помним, чтобы кнопка не «сбрасывалась» на
// первый пункт после перехода к другому инструменту.
const chosen = ref({ select: 'select', draw: 'pencil', vector: 'pen', shape: 'rect' })
const menu = ref({ visible: false, x: 0, y: 0, items: [] })
const colorPopover = ref({ open: false, anchor: { x: 0, y: 0 }, target: 'color' })
/* Величины (толщина, ластик, кегль) задаются ползунком с полем — набор
   пресетов в меню не давал ни промежуточных значений, ни крупных. */
const sizePopover = ref({ open: false, anchor: { x: 0, y: 0 }, kind: 'width' })

// «Звезда» — тот же многоугольник, но с вершинами через одну: отдельного
// инструмента в сцене нет, различает флаг polygonStar.
const activeKey = computed(() => (props.tool === 'polygon' && props.polygonStar ? 'star' : props.tool))

const groupState = computed(() => GROUPS.map((g) => {
  const pickedKey = g.items.length > 1 ? (chosen.value[g.id] || g.items[0].key) : g.items[0].key
  const item = g.items.find((i) => i.key === pickedKey) || g.items[0]
  const active = g.items.some((i) => i.key === activeKey.value)
  // У активной группы показываем то, чем рисуют, а не то, что было выбрано.
  const shown = active ? (g.items.find((i) => i.key === activeKey.value) || item) : item
  return { ...g, shown, active }
}))

const showFill = computed(() => ['rect', 'ellipse', 'diamond', 'polygon', 'curve', 'pen', 'fill'].includes(props.tool)
  || props.hasSelection)
const showTextSize = computed(() => props.tool === 'text')
const showEraser = computed(() => props.tool === 'eraser')
const showWidth = computed(() => !['text', 'sticky', 'comment', 'eraser', 'eyedropper', 'fill', 'nodes'].includes(props.tool)
  || props.hasSelection)
const zoomPercent = computed(() => `${Math.round(props.zoom * 100)}%`)
const opacityPercent = computed(() => `${Math.round(props.opacity * 100)}%`)

const swatchStyle = (value) => {
  if (isHexColor(value)) return { background: value }
  const found = SCENE_COLORS.find((c) => c.key === value)
  if (!found) return { background: 'var(--color-surface-variant)' }
  return { background: found.token ? `var(${found.token})` : found.value }
}

function pickTool(key) {
  if (key === 'star') {
    emit('update:polygonStar', true)
    emit('update:tool', 'polygon')
    return
  }
  if (key === 'polygon') emit('update:polygonStar', false)
  emit('update:tool', key)
}

function onGroupClick(group, e) {
  if (group.items.length === 1) {
    pickTool(group.items[0].key)
    return
  }
  // Клик по неактивной группе просто выбирает её инструмент, по активной —
  // открывает остальные варианты.
  if (!group.active) {
    pickTool(group.shown.key)
    return
  }
  openMenu(e, group.items.map((i) => ({
    label: i.label, icon: i.icon, action: `tool:${i.key}`,
  })))
}

function openMenu(e, items) {
  const rect = e.currentTarget.getBoundingClientRect()
  menu.value = { visible: true, x: rect.left, y: rect.top - 6, items }
}

// Настройки величины для каждой кнопки: границы, пресеты и режимы ластика.
const SIZE_KINDS = {
  width: { title: 'Толщина линии', min: 0, max: 200, presets: STROKE_WIDTHS, zeroLabel: 'без обводки' },
  eraser: { title: 'Ластик', min: 2, max: 400, presets: ERASER_SIZES, modes: [
    { key: 'pixel', label: 'Пиксели' },
    { key: 'object', label: 'Объекты' },
  ] },
  text: { title: 'Размер надписи', min: 8, max: 200, presets: TEXT_SIZES },
}

const sizeKind = computed(() => SIZE_KINDS[sizePopover.value.kind] || SIZE_KINDS.width)

const sizeValue = computed(() => ({
  width: props.width,
  eraser: props.eraserSize,
  text: props.textSize,
}[sizePopover.value.kind] ?? 0))

function openSize(e, kind) {
  const rect = e.currentTarget.getBoundingClientRect()
  sizePopover.value = { open: true, anchor: { x: rect.left + rect.width / 2, y: rect.top }, kind }
}

function onSizeChange(value) {
  const event = { width: 'update:width', eraser: 'update:eraserSize', text: 'update:textSize' }[sizePopover.value.kind]
  if (event) emit(event, value)
}

const opacityItems = computed(() => [100, 75, 50, 25, 10].map((p) => ({
  label: `${p}%`,
  icon: Math.round(props.opacity * 100) === p ? 'radio_button_checked' : 'radio_button_unchecked',
  action: `opacity:${p}`,
})))

const sidesItems = computed(() => [3, 4, 5, 6, 8, 12].map((n) => ({
  label: `${n} вершин`,
  icon: props.polygonSides === n ? 'radio_button_checked' : 'radio_button_unchecked',
  action: `sides:${n}`,
})))

function onMenuSelect(action) {
  menu.value.visible = false
  const [kind, value] = action.split(':')
  switch (kind) {
    case 'tool':
      // Запоминаем выбор внутри группы — кнопка панели покажет его и потом.
      for (const g of GROUPS) {
        if (g.items.some((i) => i.key === value)) chosen.value = { ...chosen.value, [g.id]: value }
      }
      pickTool(value)
      break
    case 'opacity': emit('update:opacity', Number(value) / 100); break
    case 'sides': emit('update:polygonSides', Number(value)); break
    default: break
  }
}

function openColor(e, target) {
  const rect = e.currentTarget.getBoundingClientRect()
  colorPopover.value = { open: true, anchor: { x: rect.left + rect.width / 2, y: rect.bottom }, target }
}

function onColorSelect(value) {
  emit(colorPopover.value.target === 'fill' ? 'update:fill' : 'update:color', value)
}
</script>

<template>
  <div class="bt">
    <div class="bt-group bt-tools">
      <button
        v-for="g in groupState"
        :key="g.id"
        type="button"
        class="bt-btn"
        :class="{ 'is-active': g.active, 'has-more': g.items.length > 1 }"
        :title="g.items.length > 1 ? `${g.shown.label} — ещё раз для выбора` : g.shown.label"
        :aria-label="g.shown.label"
        :aria-pressed="g.active"
        @click="onGroupClick(g, $event)"
      >
        <span class="material-symbols-outlined">{{ g.shown.icon }}</span>
      </button>
      <button type="button" class="bt-btn" title="Картинка" aria-label="Картинка" @click="emit('add-image')">
        <span class="material-symbols-outlined">image</span>
      </button>
    </div>

    <div class="bt-group">
      <button
        type="button"
        class="bt-swatch"
        :style="swatchStyle(color)"
        :title="hasSelection ? 'Цвет — перекрасить выделенное' : 'Цвет'"
        aria-label="Цвет"
        @click="openColor($event, 'color')"
      />
      <button
        v-if="showFill"
        type="button"
        class="bt-swatch bt-swatch--fill"
        :class="{ 'bt-swatch--none': !fill }"
        :style="fill ? swatchStyle(fill) : {}"
        title="Заливка"
        aria-label="Заливка"
        @click="openColor($event, 'fill')"
      >
        <span v-if="!fill" class="material-symbols-outlined">block</span>
      </button>
      <button
        v-if="tool === 'polygon'"
        type="button"
        class="bt-btn bt-size"
        title="Число вершин"
        aria-label="Число вершин"
        @click="openMenu($event, sidesItems)"
      >{{ polygonSides }}</button>
    </div>

    <div class="bt-group">
      <button
        v-if="showWidth"
        type="button"
        class="bt-btn bt-width"
        :title="`Толщина ${width}`"
        aria-label="Толщина"
        @click="openSize($event, 'width')"
      >
        <span class="bt-width-dot" :style="{ width: `${Math.min(width + 2, 18)}px`, height: `${Math.min(width + 2, 18)}px` }" />
      </button>
      <button
        v-if="showEraser"
        type="button"
        class="bt-btn bt-size"
        :title="eraseMode === 'pixel' ? 'Ластик: пиксели' : 'Ластик: объекты'"
        aria-label="Настройки ластика"
        @click="openSize($event, 'eraser')"
      >{{ eraseMode === 'pixel' ? eraserSize : 'об' }}</button>
      <button
        v-if="showTextSize"
        type="button"
        class="bt-btn bt-size"
        title="Размер надписи"
        aria-label="Размер надписи"
        @click="openSize($event, 'text')"
      >{{ textSize }}</button>
      <button
        type="button"
        class="bt-btn bt-size"
        title="Непрозрачность"
        aria-label="Непрозрачность"
        @click="openMenu($event, opacityItems)"
      >{{ opacityPercent }}</button>
    </div>

    <div class="bt-group">
      <button type="button" class="bt-btn" title="Отдалить" aria-label="Отдалить" @click="emit('zoom-out')">
        <span class="material-symbols-outlined">zoom_out</span>
      </button>
      <span class="bt-zoom">{{ zoomPercent }}</span>
      <button type="button" class="bt-btn" title="Приблизить" aria-label="Приблизить" @click="emit('zoom-in')">
        <span class="material-symbols-outlined">zoom_in</span>
      </button>
      <button type="button" class="bt-btn" title="По размеру" aria-label="По размеру" @click="emit('fit')">
        <span class="material-symbols-outlined">fit_screen</span>
      </button>
      <button
        v-if="hasSelection"
        type="button"
        class="bt-btn bt-btn--danger"
        title="Удалить выделенное"
        aria-label="Удалить выделенное"
        @click="emit('delete-selected')"
      >
        <span class="material-symbols-outlined">delete</span>
      </button>
    </div>

    <ContextMenu
      :visible="menu.visible"
      :x="menu.x"
      :y="menu.y"
      :items="menu.items"
      @select="onMenuSelect"
      @close="menu.visible = false"
    />

    <SizePopover
      v-model="sizePopover.open"
      :anchor="sizePopover.anchor"
      :value="sizeValue"
      :min="sizeKind.min"
      :max="sizeKind.max"
      :title="sizeKind.title"
      :presets="sizeKind.presets"
      :zero-label="sizeKind.zeroLabel || ''"
      :modes="sizeKind.modes || []"
      :mode="eraseMode"
      @update:value="onSizeChange"
      @update:mode="(v) => emit('update:eraseMode', v)"
    />

    <BoardColorPopover
      v-model="colorPopover.open"
      :anchor="colorPopover.anchor"
      :value="colorPopover.target === 'fill' ? fill : color"
      :allow-none="colorPopover.target === 'fill'"
      :title="colorPopover.target === 'fill' ? 'Заливка' : 'Цвет'"
      @select="onColorSelect"
    />
  </div>
</template>

<style scoped>
.bt {
  /* Ширина по содержимому: панель ровно такая, сколько в ней кнопок, а при
     нехватке места кнопки прокручиваются внутри — как в панели задач. */
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 6px;
  width: fit-content;
  max-width: 100%;
  padding: 6px 8px;
  pointer-events: auto;
  overflow-x: auto;
  scrollbar-width: none;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  /* -webkit-версия ПЕРВОЙ: минификатор выбрасывает стандартное свойство, если
     оно стоит раньше префиксного, и панель остаётся без размытия. */
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  background: var(--acrylic-bg);
  box-shadow: var(--shadow-2);
}

.bt::-webkit-scrollbar { display: none; }

.bt-group {
  display: flex;
  flex: 0 0 auto;
  align-items: center;
  gap: 4px;
  padding-right: 6px;
  border-right: 1px solid var(--color-outline-variant);
}

.bt-group:last-child { border-right: none; padding-right: 0; }

.bt-tools { flex-wrap: nowrap; }

.bt-btn {
  position: relative;
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  min-width: 34px;
  max-width: 34px;
  min-height: 34px;
  max-height: 34px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.bt-btn:hover:not(.is-active) { background: var(--color-surface-variant); }
.bt-btn.is-active,
.bt-btn.is-active:hover { background: var(--color-primary); color: var(--color-on-primary); }
.bt-btn--danger { color: var(--color-error); }
.bt-btn .material-symbols-outlined { font-size: 20px; }

/* Уголок у кнопки-группы: подсказывает, что внутри есть ещё инструменты. */
.bt-btn.has-more::after {
  content: '';
  position: absolute;
  right: 3px;
  bottom: 3px;
  border-top: 4px solid transparent;
  border-right: 4px solid currentColor;
  opacity: 0.7;
}

.bt-size { font-size: 12px; font-weight: 600; }

.bt-width-dot {
  border-radius: 50%;
  background: currentColor;
}

.bt-swatch {
  flex: 0 0 auto;
  min-width: 24px;
  max-width: 24px;
  min-height: 24px;
  max-height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 2px solid var(--color-outline-dim);
  border-radius: 50%;
  background: var(--color-text);
  color: var(--color-text-muted);
  cursor: pointer;
}

.bt-swatch--fill { opacity: 0.75; }
.bt-swatch--none { background: var(--color-surface-variant); }
.bt-swatch--none .material-symbols-outlined { font-size: 14px; }

.bt-zoom {
  min-width: 44px;
  text-align: center;
  font-size: 12px;
  color: var(--color-text-muted);
}

@media (max-width: 768px) {
  .bt { gap: 4px; padding: 4px 6px; }
  .bt-group { padding-right: 4px; }
}
</style>
