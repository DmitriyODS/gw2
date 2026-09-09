<script setup>
/* Выбор цвета рисования: палитра доски плюс ЛЮБОЙ оттенок — спектр, поле
   «#rrggbb» и недавние цвета. Ключ палитры следует теме приложения, а
   произвольный цвет хранится значением (см. utils/boardScene.js).

   Попап живёт в body и позиционируется экранными координатами кнопки — как
   контекстные меню: у окна рабочего стола свой transform, и телепорт в тело
   окна сместил бы попап относительно клика. */
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import Slider from 'primevue/slider'
import { SCENE_COLORS, isHexColor, resolveColor } from '@/utils/boardScene.js'
import { GRADIENT_TYPES, defaultGradient } from '@/utils/boardPaint.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  // Экранная точка привязки: правый нижний угол кнопки.
  anchor: { type: Object, default: () => ({ x: 0, y: 0 }) },
  value: { type: String, default: 'ink' },
  // Заливке нужен вариант «без цвета», обводке — нет.
  allowNone: { type: Boolean, default: false },
  // Градиент предлагается только заливке: обводку им доска не красит.
  allowGradient: { type: Boolean, default: false },
  gradient: { type: Object, default: null },
  title: { type: String, default: 'Цвет' },
})

const emit = defineEmits(['update:modelValue', 'select', 'gradient'])

const RECENT_KEY = 'gw_board_colors'
const RECENT_LIMIT = 10
const SV_W = 208
const SV_H = 132
const HUE_H = 14

const root = ref(null)
const rampEl = ref(null)
const svCanvas = ref(null)
const hueCanvas = ref(null)
const hsv = ref({ h: 210, s: 0.7, v: 0.9 })
const hex = ref('#3b74d6')
const recent = ref(loadRecent())
const style = ref({ left: '0px', top: '0px' })
// Вкладки попапа: сплошной цвет или градиент (у заливки).
const tab = ref('solid')
const draft = ref(null)      // правится копия градиента, оригинал — в объекте
const stopIndex = ref(0)

const swatchStyle = (c) => ({ background: c.token ? `var(${c.token})` : c.value })

const currentHex = computed(() => (isHexColor(props.value) ? props.value.toLowerCase() : ''))

function loadRecent() {
  try {
    const saved = JSON.parse(localStorage.getItem(RECENT_KEY) || '[]')
    return Array.isArray(saved) ? saved.filter(isHexColor).slice(0, RECENT_LIMIT) : []
  } catch {
    return []
  }
}

function rememberColor(value) {
  if (!isHexColor(value)) return
  const next = [value.toLowerCase(), ...recent.value.filter((c) => c !== value.toLowerCase())].slice(0, RECENT_LIMIT)
  recent.value = next
  try {
    localStorage.setItem(RECENT_KEY, JSON.stringify(next))
  } catch { /* приватный режим — недавние просто не переживут перезагрузку */ }
}

// ── Преобразования цвета ─────────────────────────────────────────

function hsvToHex({ h, s, v }) {
  const f = (n) => {
    const k = (n + h / 60) % 6
    const value = v - v * s * Math.max(0, Math.min(k, 4 - k, 1))
    return Math.round(value * 255).toString(16).padStart(2, '0')
  }
  return `#${f(5)}${f(3)}${f(1)}`
}

/* cssToHex — значение темы («oklch(…)», «rgb(…)») в «#rrggbb»: спектр обязан
   открыться на том цвете, которым рисуют сейчас, а токены палитры заданы в
   OKLCH — вручную их не пересчитать. Просим об этом сам браузер. */
function cssToHex(value) {
  if (isHexColor(value)) return value.toLowerCase()
  try {
    const probe = document.createElement('canvas')
    probe.width = 1
    probe.height = 1
    const ctx = probe.getContext('2d', { willReadFrequently: true })
    if (!ctx) return ''
    ctx.fillStyle = value
    ctx.fillRect(0, 0, 1, 1)
    const [r, g, b] = ctx.getImageData(0, 0, 1, 1).data
    return `#${[r, g, b].map((v) => v.toString(16).padStart(2, '0')).join('')}`
  } catch {
    return ''
  }
}

function hexToHsv(value) {
  const full = value.length === 4
    ? `#${value[1]}${value[1]}${value[2]}${value[2]}${value[3]}${value[3]}`
    : value
  const r = parseInt(full.slice(1, 3), 16) / 255
  const g = parseInt(full.slice(3, 5), 16) / 255
  const b = parseInt(full.slice(5, 7), 16) / 255
  const max = Math.max(r, g, b); const min = Math.min(r, g, b)
  const d = max - min
  let h = 0
  if (d) {
    if (max === r) h = 60 * (((g - b) / d) % 6)
    else if (max === g) h = 60 * ((b - r) / d + 2)
    else h = 60 * ((r - g) / d + 4)
  }
  return { h: (h + 360) % 360, s: max ? d / max : 0, v: max }
}

// ── Отрисовка спектра ────────────────────────────────────────────

function drawSv() {
  const ctx = svCanvas.value?.getContext('2d')
  if (!ctx) return
  ctx.fillStyle = hsvToHex({ h: hsv.value.h, s: 1, v: 1 })
  ctx.fillRect(0, 0, SV_W, SV_H)
  const white = ctx.createLinearGradient(0, 0, SV_W, 0)
  white.addColorStop(0, 'rgba(255,255,255,1)')
  white.addColorStop(1, 'rgba(255,255,255,0)')
  ctx.fillStyle = white
  ctx.fillRect(0, 0, SV_W, SV_H)
  const black = ctx.createLinearGradient(0, 0, 0, SV_H)
  black.addColorStop(0, 'rgba(0,0,0,0)')
  black.addColorStop(1, 'rgba(0,0,0,1)')
  ctx.fillStyle = black
  ctx.fillRect(0, 0, SV_W, SV_H)
}

function drawHue() {
  const ctx = hueCanvas.value?.getContext('2d')
  if (!ctx) return
  const grad = ctx.createLinearGradient(0, 0, SV_W, 0)
  for (let i = 0; i <= 6; i += 1) grad.addColorStop(i / 6, hsvToHex({ h: i * 60, s: 1, v: 1 }))
  ctx.fillStyle = grad
  ctx.fillRect(0, 0, SV_W, HUE_H)
}

/* Спектр и полоса — обычные canvas, а не градиенты в CSS: значение цвета всё
   равно считается формулой, и рисовать его дважды (в стилях и в коде) значит
   рано или поздно разойтись. */
const markerSv = computed(() => ({
  left: `${hsv.value.s * SV_W}px`,
  top: `${(1 - hsv.value.v) * SV_H}px`,
}))
const markerHue = computed(() => ({ left: `${(hsv.value.h / 360) * SV_W}px` }))
const previewStyle = computed(() => ({ background: hex.value }))

function pickSv(e) {
  const rect = svCanvas.value.getBoundingClientRect()
  const s = Math.min(1, Math.max(0, (e.clientX - rect.left) / rect.width))
  const v = 1 - Math.min(1, Math.max(0, (e.clientY - rect.top) / rect.height))
  hsv.value = { ...hsv.value, s, v }
  hex.value = hsvToHex(hsv.value)
  syncStop()
}

function pickHue(e) {
  const rect = hueCanvas.value.getBoundingClientRect()
  const h = Math.min(360, Math.max(0, ((e.clientX - rect.left) / rect.width) * 360))
  hsv.value = { ...hsv.value, h }
  hex.value = hsvToHex(hsv.value)
  drawSv()
  syncStop()
}

/* На вкладке градиента спектр правит цвет ВЫБРАННОЙ точки сразу: отдельная
   кнопка «применить» здесь только мешала бы — видно же прямо на полосе. */
function syncStop() {
  if (tab.value === 'gradient' && draft.value) setStopColor(hex.value)
}

function dragSv(e) {
  e.target.setPointerCapture?.(e.pointerId)
  pickSv(e)
}

function moveSv(e) {
  if (e.buttons) pickSv(e)
}

function dragHue(e) {
  e.target.setPointerCapture?.(e.pointerId)
  pickHue(e)
}

function moveHue(e) {
  if (e.buttons) pickHue(e)
}

// ── Выбор ────────────────────────────────────────────────────────

function choose(key) {
  if (tab.value === 'gradient') {
    setStopColor(key)
    hex.value = resolveHexOf(key)
    hsv.value = hexToHsv(hex.value)
    drawSv()
    return
  }
  emit('select', key)
  close()
}

function chooseCustom() {
  if (!isHexColor(hex.value)) return
  rememberColor(hex.value)
  choose(hex.value.toLowerCase())
}

function switchTab(next) {
  tab.value = next
  if (next === 'gradient' && !draft.value) {
    pushGradient(props.gradient || defaultGradient(isHexColor(props.value) ? props.value : (props.value || 'blue')))
  }
}

function onHexInput(value) {
  const text = String(value || '').trim()
  const normalized = text.startsWith('#') ? text : `#${text}`
  hex.value = normalized
  if (isHexColor(normalized)) {
    hsv.value = hexToHsv(normalized.toLowerCase())
    drawSv()
  }
}

// ── Градиент ─────────────────────────────────────────────────────

const stops = computed(() => draft.value?.stops || [])

const previewGradient = computed(() => {
  const g = draft.value
  if (!g) return {}
  const parts = g.stops.map((st) => `${resolveColor(st.color)} ${Math.round(st.at * 100)}%`)
  return {
    background: g.type === 'radial'
      ? `radial-gradient(circle, ${parts.join(', ')})`
      : `linear-gradient(${g.angle}deg, ${parts.join(', ')})`,
  }
})

function pushGradient(next) {
  draft.value = next
  emit('gradient', next)
}

function setGradientType(type) {
  pushGradient({ ...draft.value, type })
}

function setAngle(value) {
  pushGradient({ ...draft.value, angle: Number(value) })
}

/** Цвет выбранной точки градиента берётся из общей палитры и спектра. */
function setStopColor(color) {
  const next = stops.value.map((st, i) => (i === stopIndex.value ? { ...st, color } : st))
  pushGradient({ ...draft.value, stops: next })
}

// Позиция точки правится перетаскиванием маркера, отдельного ползунка нет.

/** Доля от левого края полосы по экранной координате. */
function rampRatio(clientX) {
  const rect = rampEl.value?.getBoundingClientRect()
  if (!rect?.width) return 0
  return Math.min(1, Math.max(0, (clientX - rect.left) / rect.width))
}

/* Точки всегда держим отсортированными по позиции (иначе градиент рисуется
   рывками), а перетаскиваемую помечаем на время сортировки: искать её по
   значению нельзя — каждый шаг создаёт новые объекты, и ссылка теряется. */
function applyStops(list, markedIndex) {
  const marked = list.map((st, i) => (i === markedIndex ? { ...st, drag: true } : st))
  const sorted = [...marked].sort((a, b) => a.at - b.at)
  stopIndex.value = Math.max(0, sorted.findIndex((st) => st.drag))
  pushGradient({ ...draft.value, stops: sorted.map(({ drag, ...st }) => st) })
}

/** Клик по полосе — новая точка в этом месте цветом ближайшей к ней. */
function onRampDown(e) {
  if (!draft.value) return
  const at = rampRatio(e.clientX)
  const nearest = stops.value.reduce((best, st, i) => (
    Math.abs(st.at - at) < Math.abs(stops.value[best].at - at) ? i : best), 0)
  const list = [...stops.value, { color: stops.value[nearest].color, at }]
  applyStops(list, list.length - 1)
  hex.value = resolveHexOf(stops.value[stopIndex.value].color)
  hsv.value = hexToHsv(hex.value)
  drawSv()
}

/** Перетаскивание маркера: точка едет за курсором. */
function onKnobDown(index, e) {
  stopIndex.value = index
  hex.value = resolveHexOf(stops.value[index].color)
  hsv.value = hexToHsv(hex.value)
  drawSv()
  e.target.setPointerCapture?.(e.pointerId)

  const move = (event) => {
    const at = rampRatio(event.clientX)
    applyStops(stops.value.map((st, i) => (i === stopIndex.value ? { ...st, at } : st)), stopIndex.value)
  }
  const up = () => {
    window.removeEventListener('pointermove', move)
    window.removeEventListener('pointerup', up)
  }
  window.addEventListener('pointermove', move)
  window.addEventListener('pointerup', up)
}

/** Развернуть градиент — самая частая правка после «не туда льётся». */
function reverseStops() {
  const next = stops.value.map((st) => ({ ...st, at: 1 - st.at })).sort((a, b) => a.at - b.at)
  stopIndex.value = stops.value.length - 1 - stopIndex.value
  pushGradient({ ...draft.value, stops: next })
}

/** Убрать заливку целиком: ни цвета, ни градиента. */
function clearFill() {
  // Один сигнал наружу: «заливки нет». Отдельный пустой градиент по пути
  // ставил бы объекту заливку без цвета.
  draft.value = null
  tab.value = 'solid'
  emit('select', '')
  close()
}

function removeStop(index) {
  // Двух точек градиенту хватает по определению — меньше не отдаём.
  if (stops.value.length <= 2) return
  const next = stops.value.filter((_, i) => i !== index)
  stopIndex.value = Math.min(stopIndex.value, next.length - 1)
  pushGradient({ ...draft.value, stops: next })
}

/** Значение цвета точки в «#rrggbb» — им открывается спектр. */
function resolveHexOf(color) {
  return isHexColor(color) ? color.toLowerCase() : (cssToHex(resolveColor(color)) || hex.value)
}

function close() {
  emit('update:modelValue', false)
}

function onOutside(e) {
  if (root.value && !root.value.contains(e.target)) close()
}

function place() {
  const width = SV_W + 24
  const height = props.allowGradient ? 400 : 330
  const x = Math.min(Math.max(8, props.anchor.x - width / 2), window.innerWidth - width - 8)
  // Не помещается под кнопкой — показываем над ней.
  const below = props.anchor.y + 8
  const y = below + height > window.innerHeight ? Math.max(8, props.anchor.y - height - 44) : below
  style.value = { left: `${Math.round(x)}px`, top: `${Math.round(y)}px`, width: `${width}px` }
}

watch(() => props.modelValue, async (open) => {
  if (!open) return
  draft.value = props.gradient ? { ...props.gradient, stops: props.gradient.stops.map((st) => ({ ...st })) } : null
  tab.value = props.allowGradient && props.gradient ? 'gradient' : 'solid'
  stopIndex.value = 0
  // Открылись на текущем цвете: произвольный — как есть, ключ палитры —
  // его фактическим значением в теме (иначе спектр показывал бы чужой цвет).
  const start = currentHex.value || cssToHex(resolveColor(props.value))
  if (isHexColor(start)) {
    hex.value = start
    hsv.value = hexToHsv(hex.value)
  }
  place()
  await nextTick()
  drawSv()
  drawHue()
})

watch(() => props.anchor, place, { deep: true })

onMounted(() => {
  document.addEventListener('pointerdown', onOutside, true)
  window.addEventListener('resize', place)
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', onOutside, true)
  window.removeEventListener('resize', place)
})
</script>

<template>
  <Teleport to="body">
    <div v-if="modelValue" ref="root" class="bcp" :style="style" @pointerdown.stop>
      <header class="bcp-head">
        <span class="bcp-title">{{ title }}</span>
        <button type="button" class="bcp-icon" title="Закрыть" aria-label="Закрыть" @click="close">
          <span class="material-symbols-outlined">close</span>
        </button>
      </header>

      <div v-if="allowGradient || allowNone" class="bcp-tabs">
        <button
          v-if="allowNone"
          type="button"
          class="bcp-tab"
          :class="{ 'is-on': !value && !gradient }"
          title="Убрать заливку"
          @click="clearFill"
        >Нет</button>
        <button
          type="button"
          class="bcp-tab"
          :class="{ 'is-on': tab === 'solid' }"
          @click="switchTab('solid')"
        >Цвет</button>
        <button
          v-if="allowGradient"
          type="button"
          class="bcp-tab"
          :class="{ 'is-on': tab === 'gradient' }"
          @click="switchTab('gradient')"
        >Градиент</button>
      </div>

      <template v-if="tab === 'gradient' && draft">
        <!-- Полоса градиента: маркеры тянутся мышью, клик по пустому месту
             добавляет точку — так это устроено во всех редакторах, и отдельные
             ползунки на каждую точку не нужны. -->
        <div
          ref="rampEl"
          class="bcp-ramp"
          :style="previewGradient"
          title="Клик — добавить точку, перетаскивание — сдвинуть"
          @pointerdown="onRampDown"
        >
          <button
            v-for="(st, i) in stops"
            :key="i"
            type="button"
            class="bcp-knob"
            :class="{ 'is-on': stopIndex === i }"
            :style="{ left: `${st.at * 100}%`, background: resolveColor(st.color) }"
            :title="`Точка ${i + 1} · ${Math.round(st.at * 100)}%`"
            :aria-label="`Точка ${i + 1}`"
            @pointerdown.stop="onKnobDown(i, $event)"
          />
        </div>

        <div class="bcp-row">
          <button
            v-for="t in GRADIENT_TYPES"
            :key="t.key"
            type="button"
            class="bcp-tab bcp-tab--sm"
            :class="{ 'is-on': draft.type === t.key }"
            @click="setGradientType(t.key)"
          >{{ t.label }}</button>
          <button type="button" class="bcp-icon" title="Перевернуть" aria-label="Перевернуть" @click="reverseStops">
            <span class="material-symbols-outlined">swap_horiz</span>
          </button>
          <button
            type="button"
            class="bcp-icon"
            title="Убрать точку"
            aria-label="Убрать точку"
            :disabled="stops.length <= 2"
            @click="removeStop(stopIndex)"
          >
            <span class="material-symbols-outlined">delete</span>
          </button>
        </div>

        <div v-if="draft.type === 'linear'" class="bcp-stop">
          <span class="bcp-hint">Угол · {{ draft.angle }}°</span>
          <Slider :model-value="draft.angle" :min="0" :max="360" @update:model-value="setAngle" />
        </div>

        <p class="bcp-hint">Цвет точки {{ stopIndex + 1 }} — палитрой и спектром ниже.</p>
      </template>

      <div class="bcp-palette">
        <button
          v-for="c in SCENE_COLORS"
          :key="c.key"
          type="button"
          class="bcp-swatch"
          :class="{ 'is-active': value === c.key }"
          :style="swatchStyle(c)"
          :title="c.label"
          :aria-label="c.label"
          @click="choose(c.key)"
        />
      </div>

      <div class="bcp-sv">
        <canvas
          ref="svCanvas"
          :width="SV_W"
          :height="SV_H"
          @pointerdown="dragSv"
          @pointermove="moveSv"
        />
        <span class="bcp-marker" :style="markerSv" />
      </div>

      <div class="bcp-hue">
        <canvas
          ref="hueCanvas"
          :width="SV_W"
          :height="HUE_H"
          @pointerdown="dragHue"
          @pointermove="moveHue"
        />
        <span class="bcp-marker bcp-marker--hue" :style="markerHue" />
      </div>

      <div class="bcp-row">
        <span class="bcp-preview" :style="previewStyle" />
        <InputText
          :model-value="hex"
          class="bcp-hex"
          maxlength="7"
          spellcheck="false"
          aria-label="Код цвета"
          @update:model-value="onHexInput"
          @keydown.enter="chooseCustom"
        />
        <button type="button" class="bcp-apply" :disabled="!isHexColor(hex)" @click="chooseCustom">Взять</button>
      </div>

      <div v-if="recent.length" class="bcp-palette bcp-palette--recent">
        <button
          v-for="c in recent"
          :key="c"
          type="button"
          class="bcp-swatch"
          :class="{ 'is-active': currentHex === c }"
          :style="{ background: c }"
          :title="c"
          :aria-label="`Цвет ${c}`"
          @click="choose(c)"
        />
      </div>
    </div>
  </Teleport>
</template>

<style scoped>
.bcp {
  position: fixed;
  z-index: 1200;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 10px;
  border: 1px solid var(--glass-edge);
  border-radius: var(--radius-lg);
  /* Плотная подложка, как у панелей редактора: под попапом рисунок, и
     просвечивающий холст мешал целиться в спектр. */
  background: var(--color-surface);
  box-shadow: var(--shadow-3);
}

.bcp-head { display: flex; align-items: center; gap: 6px; }

.bcp-tabs { display: flex; gap: 4px; }

.bcp-tab {
  flex: 1;
  height: 26px;
  padding: 0 8px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text);
  font-size: 12px;
  cursor: pointer;
}

.bcp-tab.is-on { border-color: var(--color-primary); background: var(--color-primary); color: var(--color-on-primary); }
.bcp-tab--sm { flex: 0 0 auto; }

.bcp-ramp {
  position: relative;
  height: 28px;
  margin-bottom: 8px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  cursor: copy;
  touch-action: none;
}

.bcp-knob {
  position: absolute;
  top: 100%;
  width: 14px;
  height: 14px;
  margin: -7px 0 0 -7px;
  padding: 0;
  border: 2px solid var(--color-surface);
  border-radius: 50%;
  box-shadow: 0 0 0 1px var(--color-text);
  cursor: ew-resize;
  touch-action: none;
}

.bcp-knob.is-on { box-shadow: 0 0 0 2px var(--color-primary); }

.bcp-row { display: flex; align-items: center; gap: 4px; }
.bcp-stop { display: flex; flex-direction: column; gap: 4px; }
.bcp-hint { margin: 0; font-size: 11px; color: var(--color-text-muted); }
.bcp-title { flex: 1; font-size: 13px; font-weight: 600; }

.bcp-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 26px;
  max-width: 26px;
  min-height: 26px;
  max-height: 26px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
}

.bcp-icon:hover { background: var(--color-surface-variant); }
.bcp-icon .material-symbols-outlined { font-size: 18px; }

.bcp-palette {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(24px, 1fr));
  gap: 6px;
}

.bcp-palette--recent { padding-top: 2px; border-top: 1px solid var(--color-outline-variant); }

.bcp-swatch {
  min-width: 24px;
  max-width: 24px;
  min-height: 24px;
  max-height: 24px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0;
  border: 1px solid var(--color-outline-dim);
  border-radius: 50%;
  color: var(--color-text-muted);
  cursor: pointer;
}

.bcp-swatch.is-active { box-shadow: inset 0 0 0 2px var(--color-surface), 0 0 0 2px var(--color-primary); }
.bcp-swatch--none { background: var(--color-surface-variant); }
.bcp-swatch--none .material-symbols-outlined { font-size: 14px; }

.bcp-sv,
.bcp-hue { position: relative; line-height: 0; }
.bcp-sv canvas { border-radius: var(--radius-sm); cursor: crosshair; touch-action: none; }
.bcp-hue canvas { border-radius: var(--radius-full); cursor: ew-resize; touch-action: none; }

.bcp-marker {
  position: absolute;
  width: 12px;
  height: 12px;
  margin: -6px 0 0 -6px;
  border: 2px solid var(--color-surface);
  border-radius: 50%;
  box-shadow: 0 0 0 1px var(--color-text);
  pointer-events: none;
}

.bcp-marker--hue { top: 7px; }

.bcp-row { display: flex; align-items: center; gap: 6px; }

.bcp-preview {
  flex: 0 0 auto;
  width: 26px;
  height: 26px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
}

.bcp-hex { flex: 1; min-width: 0; font-size: 12px; }

.bcp-apply {
  flex: 0 0 auto;
  padding: 6px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
}

.bcp-apply:disabled { opacity: 0.5; cursor: default; }
</style>
