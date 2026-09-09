<script setup>
/* Свойства выделенного — САМОСТОЯТЕЛЬНАЯ панель у правого края редактора, как
   инспектор в Figma: заливка и градиент, обводка, прозрачность, скругление,
   эффекты, булевы операции, выравнивание и порядок. Появляется, когда на
   холсте что-то выделено, и уходит вместе с выделением.

   Фон у панели ПЛОТНЫЙ, а не стеклянный: под ней рисунок, и просвечивающий
   холст мешал читать значения и попадать по ползункам.

   Панель НИЧЕГО не хранит: значения читаются у первого выделенного объекта, а
   правки уходят наверх патчем. Иначе при выделении нескольких объектов пришлось
   бы держать своё «сводное» состояние и разбираться, чьё оно. */
import { computed, ref } from 'vue'
import Slider from 'primevue/slider'
import BoardColorPopover from '@/components/boards/BoardColorPopover.vue'
import SizeField from '@/components/boards/SizeField.vue'
import { BOOL_OPS, OBJ, SCENE_COLORS, STROKE_WIDTHS, TEXT_SIZES, isHexColor } from '@/utils/boardScene.js'
import { EFFECT_CONTROLS, EFFECT_DEFAULTS } from '@/utils/boardPaint.js'

const props = defineProps({
  // Выделенные объекты; панель показывается, только когда их хотя бы один.
  selection: { type: Array, default: () => [] },
  readOnly: { type: Boolean, default: false },
})

const emit = defineEmits(['apply', 'align', 'bool', 'bool-release', 'mask', 'order', 'to-vector', 'text-path', 'close'])

const colorPopover = ref({ open: false, anchor: { x: 0, y: 0 }, target: 'color' })
const effectsOpen = ref(false)

const first = computed(() => props.selection[0] || null)
const many = computed(() => props.selection.length > 1)
const effects = computed(() => ({ ...EFFECT_DEFAULTS, ...(first.value?.effects || {}) }))
const shadow = computed(() => effects.value.shadow)
const hasFill = computed(() => !!(first.value?.fill || first.value?.gradient))
const fillLabel = computed(() => {
  const o = first.value
  if (o?.gradient) return o.gradient.type === 'radial' ? 'Радиальный градиент' : 'Линейный градиент'
  if (!o?.fill) return 'Нет'
  return isHexColor(o.fill) ? o.fill : (colorLabel(o.fill) || 'Заливка')
})
const isBoolean = computed(() => props.selection.some((o) => o.bool))
const isMask = computed(() => props.selection.some((o) => o.maskObject))
const hasText = computed(() => props.selection.some((o) => o.type === OBJ.text))
// Скругление есть у того, что рисуется прямоугольником.
const hasRadius = computed(() => props.selection.some((o) => [OBJ.rect, OBJ.image, OBJ.sticky].includes(o.type)))
const canVector = computed(() => props.selection.some(
  (o) => ![OBJ.text, OBJ.image, OBJ.comment, OBJ.sticky, OBJ.vector].includes(o.type),
))

const swatchStyle = (value) => {
  if (isHexColor(value)) return { background: value }
  const found = SCENE_COLORS.find((c) => c.key === value)
  if (!found) return { background: 'var(--color-surface-variant)' }
  return { background: found.token ? `var(${found.token})` : found.value }
}

const gradientStyle = computed(() => {
  const g = first.value?.gradient
  if (!g) return null
  const stops = g.stops.map((s) => `${isHexColor(s.color) ? s.color : `var(--tag-${s.color}-accent, ${s.color})`} ${Math.round(s.at * 100)}%`)
  return { background: g.type === 'radial' ? `radial-gradient(${stops.join(', ')})` : `linear-gradient(${g.angle}deg, ${stops.join(', ')})` }
})

function apply(patch) {
  if (props.readOnly) return
  emit('apply', patch)
}

function openColor(e, target) {
  const rect = e.currentTarget.getBoundingClientRect()
  colorPopover.value = { open: true, anchor: { x: rect.left, y: rect.bottom }, target }
}

function onColorSelect(value) {
  if (colorPopover.value.target === 'fill') apply({ fill: value, solid: true, filled: !!value, gradient: undefined })
  else if (colorPopover.value.target === 'shadow') apply({ effects: { ...effects.value, shadow: { ...shadowOrDefault(), color: value } } })
  else apply({ color: value })
}

function onGradientSelect(gradient) {
  if (!gradient) {
    clearFill()
    return
  }
  apply({ gradient, filled: true })
}

function shadowOrDefault() {
  return shadow.value || { x: 0, y: 4, blur: 8, color: 'ink', opacity: 0.35 }
}

function setEffect(key, value) {
  apply({ effects: { ...effects.value, [key]: value } })
}

function toggleShadow() {
  apply({ effects: { ...effects.value, shadow: shadow.value ? null : shadowOrDefault() } })
}

function setShadow(key, value) {
  apply({ effects: { ...effects.value, shadow: { ...shadowOrDefault(), [key]: value } } })
}

function clearEffects() {
  apply({ effects: undefined })
}

/** Убрать заливку целиком — и цвет, и градиент. */
function clearFill() {
  apply({ fill: '', filled: false, solid: false, gradient: undefined })
}

const colorLabel = (key) => SCENE_COLORS.find((c) => c.key === key)?.label || ''

const ALIGN_BUTTONS = [
  { key: 'left', icon: 'align_horizontal_left', label: 'По левому краю' },
  { key: 'hcenter', icon: 'align_horizontal_center', label: 'По центру' },
  { key: 'right', icon: 'align_horizontal_right', label: 'По правому краю' },
  { key: 'top', icon: 'align_vertical_top', label: 'По верху' },
  { key: 'vcenter', icon: 'align_vertical_center', label: 'По середине' },
  { key: 'bottom', icon: 'align_vertical_bottom', label: 'По низу' },
  { key: 'distribute-h', icon: 'horizontal_distribute', label: 'Разложить по горизонтали' },
  { key: 'distribute-v', icon: 'vertical_distribute', label: 'Разложить по вертикали' },
]
</script>

<template>
  <aside v-if="first" class="pp">
    <header class="pp-head">
      <span class="material-symbols-outlined">tune</span>
      <h3 class="pp-title">{{ many ? `Выделено: ${selection.length}` : 'Свойства' }}</h3>
      <button type="button" class="pp-icon" title="Скрыть панель" aria-label="Скрыть панель" @click="emit('close')">
        <span class="material-symbols-outlined">close</span>
      </button>
    </header>

    <div class="pp-body">
      <section class="pp-block">
        <span class="pp-label">Обводка</span>
        <div class="pp-row">
          <button
            type="button"
            class="pp-swatch"
            :style="swatchStyle(first.color)"
            title="Цвет обводки"
            aria-label="Цвет обводки"
            @click="openColor($event, 'color')"
          />
          <span class="pp-value">{{ isHexColor(first.color) ? first.color : (colorLabel(first.color) || 'Чернила') }}</span>
        </div>
      </section>

      <section class="pp-block">
        <span class="pp-label">Заливка</span>
        <div class="pp-row">
          <button
            type="button"
            class="pp-swatch"
            :class="{ 'is-empty': !hasFill }"
            :style="gradientStyle || swatchStyle(first.fill)"
            title="Выбрать заливку"
            aria-label="Выбрать заливку"
            @click="openColor($event, 'fill')"
          >
            <span v-if="!hasFill" class="material-symbols-outlined">block</span>
          </button>
          <span class="pp-value">{{ fillLabel }}</span>
          <button
            v-if="hasFill"
            type="button"
            class="pp-icon"
            title="Убрать заливку"
            aria-label="Убрать заливку"
            :disabled="readOnly"
            @click="clearFill"
          >
            <span class="material-symbols-outlined">close</span>
          </button>
        </div>
      </section>

      <section class="pp-block">
        <SizeField
          label="Толщина линии"
          zero-label="без обводки"
          :model-value="first.width ?? 3"
          :min="0"
          :max="200"
          :presets="STROKE_WIDTHS"
          :disabled="readOnly"
          @update:model-value="(v) => apply({ width: v })"
        />
      </section>

      <section v-if="hasText" class="pp-block">
        <SizeField
          label="Размер текста"
          :model-value="first.size || 18"
          :min="8"
          :max="200"
          :presets="TEXT_SIZES"
          :disabled="readOnly"
          @update:model-value="(v) => apply({ size: v })"
        />
      </section>

      <section class="pp-block">
        <SizeField
          label="Непрозрачность"
          unit="%"
          :model-value="Math.round((first.opacity ?? 1) * 100)"
          :min="5"
          :max="100"
          :presets="[100, 75, 50, 25]"
          :disabled="readOnly"
          @update:model-value="(v) => apply({ opacity: v / 100 })"
        />
      </section>

      <section v-if="hasRadius" class="pp-block">
        <SizeField
          label="Скругление углов"
          :model-value="first.radius ?? 8"
          :min="0"
          :max="200"
          :presets="[0, 8, 16, 32]"
          :disabled="readOnly"
          @update:model-value="(v) => apply({ radius: v })"
        />
      </section>

      <section class="pp-block">
        <button type="button" class="pp-toggle" @click="effectsOpen = !effectsOpen">
          <span class="material-symbols-outlined">auto_fix_high</span>
          Эффекты и коррекция
          <span class="material-symbols-outlined pp-caret">{{ effectsOpen ? 'expand_less' : 'expand_more' }}</span>
        </button>
        <template v-if="effectsOpen">
          <div v-for="c in EFFECT_CONTROLS" :key="c.key" class="pp-effect">
            <span class="pp-label">{{ c.label }} · {{ effects[c.key] }}{{ c.unit || '' }}</span>
            <Slider
              :model-value="effects[c.key]"
              :min="c.min"
              :max="c.max"
              :step="c.step"
              :disabled="readOnly"
              @update:model-value="(v) => setEffect(c.key, v)"
            />
          </div>
          <div class="pp-row">
            <button type="button" class="pp-chip" :class="{ 'is-on': !!shadow }" :disabled="readOnly" @click="toggleShadow">
              <span class="material-symbols-outlined">ev_shadow</span>
              Тень
            </button>
            <button
              v-if="shadow"
              type="button"
              class="pp-swatch"
              :style="swatchStyle(shadow.color)"
              title="Цвет тени"
              @click="openColor($event, 'shadow')"
            />
            <button type="button" class="pp-chip" :disabled="readOnly" @click="clearEffects">Сбросить</button>
          </div>
          <template v-if="shadow">
            <div class="pp-effect">
              <span class="pp-label">Смещение по X · {{ shadow.x }}</span>
              <Slider :model-value="shadow.x" :min="-60" :max="60" :disabled="readOnly" @update:model-value="(v) => setShadow('x', v)" />
            </div>
            <div class="pp-effect">
              <span class="pp-label">Смещение по Y · {{ shadow.y }}</span>
              <Slider :model-value="shadow.y" :min="-60" :max="60" :disabled="readOnly" @update:model-value="(v) => setShadow('y', v)" />
            </div>
            <div class="pp-effect">
              <span class="pp-label">Размытие тени · {{ shadow.blur }}</span>
              <Slider :model-value="shadow.blur" :min="0" :max="80" :disabled="readOnly" @update:model-value="(v) => setShadow('blur', v)" />
            </div>
          </template>
        </template>
      </section>

      <section v-if="many" class="pp-block">
        <span class="pp-label">Объединение фигур</span>
        <div class="pp-row pp-row--wrap">
          <button
            v-for="b in BOOL_OPS"
            :key="b.key"
            type="button"
            class="pp-chip"
            :disabled="readOnly"
            :title="b.label"
            @click="emit('bool', b.key)"
          >
            <span class="material-symbols-outlined">{{ b.icon }}</span>
            {{ b.label }}
          </button>
        </div>
      </section>

      <section v-if="isBoolean" class="pp-block">
        <button type="button" class="pp-chip" :disabled="readOnly" @click="emit('bool-release')">
          <span class="material-symbols-outlined">call_split</span>
          Разъединить фигуры
        </button>
      </section>

      <section v-if="many" class="pp-block">
        <span class="pp-label">Выравнивание</span>
        <div class="pp-row pp-row--wrap">
          <button
            v-for="a in ALIGN_BUTTONS"
            :key="a.key"
            type="button"
            class="pp-icon pp-icon--bordered"
            :title="a.label"
            :aria-label="a.label"
            :disabled="readOnly"
            @click="emit('align', a.key)"
          >
            <span class="material-symbols-outlined">{{ a.icon }}</span>
          </button>
        </div>
      </section>

      <section class="pp-block">
        <span class="pp-label">Порядок и маски</span>
        <div class="pp-row pp-row--wrap">
          <button type="button" class="pp-chip" :disabled="readOnly" @click="emit('order', true)">
            <span class="material-symbols-outlined">flip_to_front</span>
            Вперёд
          </button>
          <button type="button" class="pp-chip" :disabled="readOnly" @click="emit('order', false)">
            <span class="material-symbols-outlined">flip_to_back</span>
            Назад
          </button>
          <button type="button" class="pp-chip" :class="{ 'is-on': isMask }" :disabled="readOnly" @click="emit('mask')">
            <span class="material-symbols-outlined">photo_filter</span>
            {{ isMask ? 'Снять маску' : 'Как маска' }}
          </button>
          <button v-if="canVector" type="button" class="pp-chip" :disabled="readOnly" @click="emit('to-vector')">
            <span class="material-symbols-outlined">polyline</span>
            В кривые
          </button>
          <button v-if="hasText" type="button" class="pp-chip" :disabled="readOnly" @click="emit('text-path')">
            <span class="material-symbols-outlined">text_rotation_none</span>
            Текст по контуру
          </button>
        </div>
      </section>
    </div>

    <BoardColorPopover
      v-model="colorPopover.open"
      :anchor="colorPopover.anchor"
      :value="colorPopover.target === 'fill' ? (first.fill || '') : (colorPopover.target === 'shadow' ? shadow?.color : first.color)"
      :gradient="colorPopover.target === 'fill' ? first.gradient : null"
      :allow-none="colorPopover.target === 'fill'"
      :allow-gradient="colorPopover.target === 'fill'"
      :title="colorPopover.target === 'fill' ? 'Заливка' : (colorPopover.target === 'shadow' ? 'Цвет тени' : 'Цвет')"
      @select="onColorSelect"
      @gradient="onGradientSelect"
    />
  </aside>
</template>

<style scoped>
.pp {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 268px;
  /* Высоту задаёт место панели в редакторе (top/bottom), здесь — только
     потолок для случая, когда панель ставят в обычный поток. */
  max-height: 100%;
  padding: 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-lg);
  /* Плотная подложка: панель стоит поверх рисунка, и стекло мешало бы читать. */
  background: var(--color-surface);
  box-shadow: var(--shadow-2);
}

.pp-head { display: flex; align-items: center; gap: 6px; }
.pp-title { flex: 1; margin: 0; font-size: 14px; font-weight: 600; }

.pp-body {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  min-height: 0;
  gap: 12px;
  overflow-y: auto;
}

.pp-block { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.pp-label { font-size: 11px; color: var(--color-text-muted); }

.pp-value {
  flex: 1;
  min-width: 0;
  font-size: 12px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pp-row { display: flex; align-items: center; gap: 6px; min-width: 0; }
.pp-row--wrap { flex-wrap: wrap; }
.pp-effect { display: flex; flex-direction: column; gap: 4px; }

.pp-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  max-width: 28px;
  min-height: 28px;
  max-height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}

.pp-icon--bordered { border: 1px solid var(--color-outline-dim); }
.pp-icon:hover:not(:disabled) { background: var(--color-surface-variant); }
.pp-icon:disabled { opacity: 0.4; cursor: default; }
.pp-icon .material-symbols-outlined { font-size: 18px; }

.pp-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 28px;
  padding: 0 10px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text);
  font-size: 12px;
  cursor: pointer;
}

.pp-chip:hover:not(:disabled) { background: var(--color-surface-variant); }
.pp-chip:disabled { opacity: 0.5; cursor: default; }
.pp-chip.is-on { border-color: var(--color-primary); color: var(--color-primary); }
.pp-chip .material-symbols-outlined { font-size: 16px; }

.pp-toggle {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 100%;
  padding: 6px 4px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}

.pp-toggle:hover { background: var(--color-surface-variant); }
.pp-caret { margin-left: auto; }
.pp-toggle .material-symbols-outlined { font-size: 18px; }

.pp-swatch {
  flex: 0 0 auto;
  min-width: 26px;
  max-width: 26px;
  min-height: 26px;
  max-height: 26px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 2px solid var(--color-outline-dim);
  border-radius: 50%;
  color: var(--color-text-muted);
  cursor: pointer;
}

.pp-swatch.is-empty { background: var(--color-surface-variant); }
.pp-swatch .material-symbols-outlined { font-size: 14px; }
</style>
