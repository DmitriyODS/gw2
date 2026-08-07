<script setup>
/* Панель инструментов холста: выбор инструмента, цвет, толщина, заливка и
   масштаб. Плавающая — чтобы не отъедать рабочую площадь доски. */
import { computed } from 'vue'
import { SCENE_COLORS, STROKE_WIDTHS, TEXT_SIZES } from '@/utils/boardScene.js'

const props = defineProps({
  tool: { type: String, required: true },
  color: { type: String, required: true },
  fill: { type: String, default: '' },
  width: { type: Number, required: true },
  textSize: { type: Number, required: true },
  zoom: { type: Number, default: 1 },
  hasSelection: { type: Boolean, default: false },
})

const emit = defineEmits([
  'update:tool', 'update:color', 'update:fill', 'update:width', 'update:textSize',
  'zoom-in', 'zoom-out', 'fit', 'add-image', 'delete-selected',
])

const TOOLS = [
  { key: 'select', icon: 'arrow_selector_tool', label: 'Выделение' },
  { key: 'pan', icon: 'pan_tool', label: 'Рука' },
  { key: 'pen', icon: 'draw', label: 'Перо' },
  { key: 'marker', icon: 'ink_highlighter', label: 'Маркер' },
  { key: 'eraser', icon: 'ink_eraser', label: 'Ластик' },
  { key: 'line', icon: 'horizontal_rule', label: 'Линия' },
  { key: 'arrow', icon: 'arrow_right_alt', label: 'Стрелка' },
  { key: 'rect', icon: 'crop_square', label: 'Прямоугольник' },
  { key: 'ellipse', icon: 'circle', label: 'Овал' },
  { key: 'diamond', icon: 'change_history', label: 'Ромб' },
  { key: 'text', icon: 'title', label: 'Надпись' },
  { key: 'sticky', icon: 'sticky_note_2', label: 'Липкая заметка' },
  { key: 'comment', icon: 'add_comment', label: 'Комментарий' },
]

// Заливка и толщина нужны не всем инструментам — панель не должна пестрить.
const showFill = computed(() => ['rect', 'ellipse', 'diamond'].includes(props.tool) || props.hasSelection)
const showWidth = computed(() => !['text', 'sticky', 'comment'].includes(props.tool) || props.hasSelection)
const showTextSize = computed(() => props.tool === 'text')
const zoomPercent = computed(() => `${Math.round(props.zoom * 100)}%`)
</script>

<template>
  <div class="bt">
    <div class="bt-group bt-tools">
      <button
        v-for="t in TOOLS"
        :key="t.key"
        type="button"
        class="bt-btn"
        :class="{ 'is-active': tool === t.key }"
        :title="t.label"
        :aria-label="t.label"
        :aria-pressed="tool === t.key"
        @click="emit('update:tool', t.key)"
      >
        <span class="material-symbols-outlined">{{ t.icon }}</span>
      </button>
      <button type="button" class="bt-btn" title="Картинка" aria-label="Картинка" @click="emit('add-image')">
        <span class="material-symbols-outlined">image</span>
      </button>
    </div>

    <div class="bt-group bt-colors">
      <button
        v-for="c in SCENE_COLORS"
        :key="c.key"
        type="button"
        class="bt-swatch"
        :class="{ 'is-active': color === c.key }"
        :style="{ '--sw': `var(${c.token})` }"
        :title="hasSelection ? `${c.label} — перекрасить выделенное` : c.label"
        :aria-label="c.label"
        :aria-pressed="color === c.key"
        @click="emit('update:color', c.key)"
      />
    </div>

    <div v-if="showFill" class="bt-group bt-colors">
      <button
        type="button"
        class="bt-swatch bt-swatch--none"
        :class="{ 'is-active': !fill }"
        title="Без заливки"
        aria-label="Без заливки"
        @click="emit('update:fill', '')"
      >
        <span class="material-symbols-outlined">block</span>
      </button>
      <button
        v-for="c in SCENE_COLORS.slice(1)"
        :key="`f-${c.key}`"
        type="button"
        class="bt-swatch bt-swatch--fill"
        :class="{ 'is-active': fill === c.key }"
        :style="{ '--sw': `var(${c.token})` }"
        :title="`Заливка: ${c.label.toLowerCase()}`"
        :aria-label="`Заливка: ${c.label.toLowerCase()}`"
        @click="emit('update:fill', c.key)"
      />
    </div>

    <div v-if="showWidth" class="bt-group">
      <button
        v-for="w in STROKE_WIDTHS"
        :key="w"
        type="button"
        class="bt-btn bt-width"
        :class="{ 'is-active': width === w }"
        :title="`Толщина ${w}`"
        :aria-label="`Толщина ${w}`"
        @click="emit('update:width', w)"
      >
        <span class="bt-width-dot" :style="{ width: `${Math.min(w + 2, 18)}px`, height: `${Math.min(w + 2, 18)}px` }" />
      </button>
    </div>

    <div v-if="showTextSize" class="bt-group">
      <button
        v-for="s in TEXT_SIZES"
        :key="s"
        type="button"
        class="bt-btn bt-size"
        :class="{ 'is-active': textSize === s }"
        :title="`Размер ${s}`"
        @click="emit('update:textSize', s)"
      >{{ s }}</button>
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

.bt-size { font-size: 12px; font-weight: 600; }

.bt-width-dot {
  border-radius: 50%;
  background: currentColor;
}

.bt-swatch {
  flex: 0 0 auto;
  min-width: 22px;
  max-width: 22px;
  min-height: 22px;
  max-height: 22px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 50%;
  background: var(--sw, var(--color-text));
  color: var(--color-text-muted);
  cursor: pointer;
}

/* Рамку выделения рисуем внутренней тенью: внешнюю срезает прокрутка панели. */
.bt-swatch.is-active { box-shadow: inset 0 0 0 2px var(--color-surface), 0 0 0 2px var(--color-primary); }
.bt-swatch--fill { opacity: 0.55; }
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
