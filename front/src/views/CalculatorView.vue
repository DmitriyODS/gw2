<template>
  <div class="calc">
    <!-- Табло: набранное выражение и живой результат под ним. -->
    <div class="calc-screen">
      <div class="calc-flags">
        <span v-if="memory !== 0" class="calc-flag">M</span>
        <button
          v-if="scientific"
          class="calc-flag as-btn"
          type="button"
          :title="angle === 'deg' ? 'Градусы — переключить на радианы' : 'Радианы — переключить на градусы'"
          @click="toggleAngle"
        >{{ angle === 'deg' ? 'DEG' : 'RAD' }}</button>
      </div>
      <input
        ref="exprEl"
        v-model="expr"
        class="calc-expr"
        type="text"
        spellcheck="false"
        placeholder="0"
        aria-label="Выражение"
        @keydown.enter.prevent="equals"
      />
      <div class="calc-preview" :class="{ error: expr.trim() && preview === null }">
        {{ previewText }}
      </div>
    </div>

    <div class="calc-tools">
      <AppTabs variant="tint" v-model="mode" :tabs="MODES" dense />
      <button
        class="calc-hist-toggle"
        type="button"
        :class="{ active: historyOpen }"
        :title="historyOpen ? 'Скрыть историю' : 'История вычислений'"
        aria-label="История вычислений"
        @click="historyOpen = !historyOpen"
      >
        <span class="material-symbols-outlined">history</span>
      </button>
    </div>

    <!-- Клавиатура делит остаток высоты: ряды тянутся, поэтому калькулятор
         никогда не прокручивается и не обрезается — только меняет масштаб. -->
    <div class="calc-pads">
      <!-- Инженерные функции требуют скобку, поэтому вставляются сразу с ней —
           курсор оказывается внутри. -->
      <div v-if="scientific" class="calc-fnpad">
        <button
          v-for="k in SCI_KEYS"
          :key="k.label"
          class="calc-key fn"
          type="button"
          @click="press(k)"
        >{{ k.label }}</button>
      </div>

      <div class="calc-keys">
        <button
          v-for="k in MEM_KEYS"
          :key="k.label"
          class="calc-key mem"
          type="button"
          @click="press(k)"
        >{{ k.label }}</button>

        <button
          v-for="k in KEYS"
          :key="k.label"
          class="calc-key"
          :class="k.kind"
          type="button"
          @click="press(k)"
        >{{ k.label }}</button>
      </div>

      <!-- История — слой поверх клавиатуры: в маленьком окне ей негде стоять
           рядом, а прокрутка уместна только внутри неё самой. -->
      <transition name="calc-hist">
        <aside v-if="historyOpen" class="calc-history">
          <header class="calc-history-head">
            <h3 class="calc-history-title">История</h3>
            <button
              v-if="history.length"
              class="calc-history-icon"
              type="button"
              title="Очистить историю"
              aria-label="Очистить историю"
              @click="clearHistory"
            >
              <span class="material-symbols-outlined">delete_sweep</span>
            </button>
            <button
              class="calc-history-icon"
              type="button"
              title="Закрыть"
              aria-label="Закрыть историю"
              @click="historyOpen = false"
            >
              <span class="material-symbols-outlined">close</span>
            </button>
          </header>
          <div class="calc-history-list">
            <p v-if="!history.length" class="calc-history-empty">Здесь появятся вычисления</p>
            <button
              v-for="(row, i) in history"
              :key="`${row.expr}-${i}`"
              class="calc-history-row"
              type="button"
              title="Подставить в табло"
              @click="expr = row.expr"
            >
              <span class="calc-history-expr">{{ row.expr }}</span>
              <span class="calc-history-value">= {{ row.value }}</span>
            </button>
          </div>
        </aside>
      </transition>
    </div>
  </div>
</template>

<script setup>
/**
 * Калькулятор — компактное окно рабочего стола (плитка «Пуска» и команда Hola).
 *
 * Считает тем же разбором, что строка Hola (utils/calc.js), поэтому «15% от
 * 2000» и «sqrt(9)» здесь дают тот же ответ. Табло — редактируемая строка:
 * выражение правится руками и набирается с клавиатуры, а не только кнопками.
 * Раскладка тянется за размером окна (единицы контейнера), прокрутки нет.
 */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { evaluate, formatResult } from '@/utils/calc.js'
import { storageGet, storageSet, storageGetJSON, storageSetJSON } from '@/utils/storage.js'
import AppTabs from '@/components/ui/AppTabs.vue'

const MODE_KEY = 'gw_calc_mode'
const ANGLE_KEY = 'gw_calc_angle'
const HISTORY_KEY = 'gw_calc_history'
const HISTORY_MAX = 20

const MODES = [
  { key: 'basic', label: 'Обычный' },
  { key: 'scientific', label: 'Инженерный' },
]

const SCI_KEYS = [
  { label: 'sin', insert: 'sin(' },
  { label: 'cos', insert: 'cos(' },
  { label: 'tan', insert: 'tan(' },
  { label: 'ln', insert: 'ln(' },
  { label: 'log', insert: 'log(' },
  { label: '√', insert: 'sqrt(' },
  { label: 'x²', insert: '^2' },
  { label: 'xʸ', insert: '^' },
  { label: 'n!', insert: '!' },
  { label: 'π', insert: 'pi' },
  { label: 'e', insert: 'e' },
  { label: 'eˣ', insert: 'exp(' },
]

const MEM_KEYS = [
  { label: 'MC', action: 'mc' },
  { label: 'MR', action: 'mr' },
  { label: 'M+', action: 'm+' },
  { label: 'M−', action: 'm-' },
]

const KEYS = [
  { label: 'C', action: 'clear', kind: 'act' },
  { label: '( )', action: 'paren', kind: 'act' },
  { label: '%', insert: '%', kind: 'act' },
  { label: '⌫', action: 'back', kind: 'act' },
  { label: '7', insert: '7' },
  { label: '8', insert: '8' },
  { label: '9', insert: '9' },
  { label: '÷', insert: '/', kind: 'op' },
  { label: '4', insert: '4' },
  { label: '5', insert: '5' },
  { label: '6', insert: '6' },
  { label: '×', insert: '*', kind: 'op' },
  { label: '1', insert: '1' },
  { label: '2', insert: '2' },
  { label: '3', insert: '3' },
  { label: '−', insert: '-', kind: 'op' },
  { label: '±', action: 'negate' },
  { label: '0', insert: '0' },
  { label: '.', insert: '.' },
  { label: '+', insert: '+', kind: 'op' },
  { label: '=', action: 'equals', kind: 'eq' },
]

const expr = ref('')
const exprEl = ref(null)
const memory = ref(0)
const historyOpen = ref(false)
const history = ref(loadHistory())
const mode = ref(storageGet(MODE_KEY, 'basic') === 'scientific' ? 'scientific' : 'basic')
const angle = ref(storageGet(ANGLE_KEY, 'deg') === 'rad' ? 'rad' : 'deg')

const scientific = computed(() => mode.value === 'scientific')

watch(mode, (v) => storageSet(MODE_KEY, v))

const preview = computed(() => evaluate(expr.value, { angle: angle.value }))

const previewText = computed(() => {
  if (!expr.value.trim()) return '0'
  return preview.value === null ? 'Не получается посчитать' : formatResult(preview.value)
})

function loadHistory() {
  const raw = storageGetJSON(HISTORY_KEY, [])
  return Array.isArray(raw) ? raw.filter((r) => r?.expr && r?.value) : []
}

function toggleAngle() {
  angle.value = angle.value === 'deg' ? 'rad' : 'deg'
  storageSet(ANGLE_KEY, angle.value)
}

/* Вставка идёт в позицию курсора — табло остаётся обычным текстовым полем,
   которое можно править руками. */
function insert(chunk) {
  const el = exprEl.value
  const start = el?.selectionStart ?? expr.value.length
  const end = el?.selectionEnd ?? expr.value.length
  expr.value = expr.value.slice(0, start) + chunk + expr.value.slice(end)
  const caret = start + chunk.length
  requestAnimationFrame(() => {
    el?.focus()
    el?.setSelectionRange(caret, caret)
  })
}

function press(key) {
  if (key.insert) return insert(key.insert)
  const actions = {
    clear: () => { expr.value = '' },
    back: backspace,
    paren,
    negate,
    equals,
    mc: () => { memory.value = 0 },
    mr: () => insert(String(memory.value)),
    'm+': () => { memory.value += preview.value ?? 0 },
    'm-': () => { memory.value -= preview.value ?? 0 },
  }
  actions[key.action]?.()
}

function backspace() {
  const el = exprEl.value
  const start = el?.selectionStart ?? expr.value.length
  const end = el?.selectionEnd ?? expr.value.length
  if (start !== end) {
    expr.value = expr.value.slice(0, start) + expr.value.slice(end)
    return
  }
  if (!start) return
  expr.value = expr.value.slice(0, start - 1) + expr.value.slice(start)
  requestAnimationFrame(() => {
    el?.focus()
    el?.setSelectionRange(start - 1, start - 1)
  })
}

/* Одна кнопка на обе скобки: закрываем, пока есть незакрытые, иначе открываем. */
function paren() {
  const opened = (expr.value.match(/\(/g) || []).length
  const closed = (expr.value.match(/\)/g) || []).length
  insert(opened > closed && /[\d)]\s*$/.test(expr.value) ? ')' : '(')
}

function negate() {
  const value = preview.value
  if (value === null) return
  expr.value = formatNumber(-value)
}

function equals() {
  const value = preview.value
  const text = expr.value.trim()
  if (value === null || !text) return
  const result = formatNumber(value)
  if (text !== result) {
    history.value = [{ expr: text, value: formatResult(value) }, ...history.value].slice(0, HISTORY_MAX)
    storageSetJSON(HISTORY_KEY, history.value)
  }
  expr.value = result
}

// В табло числа держим машинными (без разрядных пробелов) — иначе следующий
// шаг вычисления не разберёт собственный результат.
function formatNumber(value) {
  return String(Math.round(value * 1e10) / 1e10)
}

function clearHistory() {
  history.value = []
  storageSetJSON(HISTORY_KEY, history.value)
}

/* Клавиатура: цифры и операторы попадают в поле сами, здесь — только то, что
   поле не умеет (Esc — очистка). */
function onKeydown(e) {
  if (e.key === 'Escape') {
    e.preventDefault()
    expr.value = ''
  }
}

onMounted(() => {
  exprEl.value?.focus()
  window.addEventListener('keydown', onKeydown)
})

onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
/* Размеры считаем от САМОГО окна (container-type: size), поэтому калькулятор
   одинаково целен и в узкой колонке, и на пол-экрана: ряды клавиш делят
   остаток высоты, шрифты берут масштаб из cqmin. Каждому такому свойству
   предшествует px-значение — заводской WebView старых Android единиц
   контейнера не знает и возьмёт его. */
.calc {
  container-type: size;
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  padding: 12px;
  padding: clamp(8px, 3cqmin, 16px);
  gap: 8px;
  gap: clamp(6px, 2cqmin, 12px);
}

/* ── Табло ── */
.calc-screen {
  position: relative;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  padding: 12px 14px;
  padding: clamp(10px, 3cqmin, 16px) clamp(12px, 3.5cqmin, 18px);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
}

.calc-flags {
  position: absolute;
  top: 8px;
  left: 12px;
  display: flex;
  gap: 6px;
}

.calc-flag {
  padding: 2px 8px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-full);
  background: transparent;
  color: var(--color-text-dim);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.4px;
}

.calc-flag.as-btn { cursor: pointer; }
.calc-flag.as-btn:hover { color: var(--color-primary); border-color: var(--color-primary); }

.calc-expr {
  width: 100%;
  margin-top: 8px;
  border: none;
  outline: none;
  background: transparent;
  color: var(--color-text);
  font-size: 24px;
  font-size: clamp(18px, 7cqmin, 32px);
  font-weight: 600;
  font-family: inherit;
  font-variant-numeric: tabular-nums;
  text-align: right;
}

.calc-preview {
  font-size: 13px;
  font-size: clamp(11px, 3.6cqmin, 16px);
  font-weight: 500;
  color: var(--color-text-dim);
  font-variant-numeric: tabular-nums;
  text-align: right;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.calc-preview.error { color: color-mix(in oklch, var(--color-error) 80%, transparent); }

/* ── Переключатели ── */
.calc-tools {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  flex-shrink: 0;
}

/* Переключатель режима — ровно по строке инструментов: раздутая дорожка
   спорила с клавиатурой и ломала ритм окна. */
.calc-tools :deep(.app-tabs) {
  flex: 1;
  min-width: 0;
  padding: 3px;
}

.calc-tools :deep(.app-tab) {
  flex: 1;
  min-width: 0;
  justify-content: center;
  padding: 6px 10px;
  font-size: 12px;
  font-size: clamp(11px, 3.2cqmin, 13px);
  white-space: nowrap;
}

.calc-hist-toggle {
  width: 34px;
  min-width: 34px;
  max-width: 34px;
  height: 34px;
  min-height: 34px;
  max-height: 34px;
  display: grid;
  place-items: center;
  padding: 0;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text-dim);
  cursor: pointer;
  transition: color 0.15s, border-color 0.15s;
}

.calc-hist-toggle:hover,
.calc-hist-toggle.active {
  color: var(--color-primary);
  border-color: color-mix(in oklch, var(--color-primary) 32%, var(--acrylic-border));
}

.calc-hist-toggle .material-symbols-outlined { font-size: 18px; }

/* ── Клавиатура ──
   Обе сетки делят высоту пропорционально числу своих рядов (функции — 2,
   основной блок — 7), поэтому клавиши остаются одинаковыми по высоте. */
.calc-pads {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  gap: clamp(4px, 1.6cqmin, 8px);
}

.calc-fnpad,
.calc-keys {
  display: grid;
  gap: 6px;
  gap: clamp(4px, 1.6cqmin, 8px);
  grid-auto-rows: 1fr;
}

.calc-fnpad { flex: 3; grid-template-columns: repeat(4, 1fr); }
.calc-keys { flex: 7; grid-template-columns: repeat(4, 1fr); }

.calc-key {
  min-height: 0;
  padding: 0 2px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  font-size: 16px;
  font-size: clamp(13px, 4.4cqmin, 20px);
  font-weight: 600;
  line-height: 1;
  cursor: pointer;
  transition: background 0.12s, border-color 0.12s, color 0.12s;
}

.calc-key:hover { border-color: color-mix(in oklch, var(--color-primary) 34%, var(--acrylic-border)); }
.calc-key:active { background: color-mix(in oklch, var(--color-primary) 14%, var(--glass-bg)); }

.calc-key.op { color: var(--color-primary); }
.calc-key.act { color: var(--color-text-dim); }

.calc-key.fn,
.calc-key.mem {
  color: var(--color-text-dim);
  font-size: 12px;
  font-size: clamp(10px, 3.2cqmin, 14px);
}

/* «=» замыкает клавиатуру во всю ширину — последний ряд иначе оставался бы
   с тремя пустыми клетками. */
.calc-key.eq {
  grid-column: 1 / -1;
  background: var(--grad-primary-soft);
  border-color: color-mix(in oklch, var(--color-primary) 36%, transparent);
  color: var(--color-primary);
}

/* ── История ── */
.calc-history {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--acrylic-bg-strong);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  box-shadow: var(--shadow-md);
}

.calc-history-head { display: flex; align-items: center; gap: 6px; }

.calc-history-title {
  flex: 1;
  margin: 0;
  font-size: 12px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.4px;
  color: var(--color-text-dim);
}

.calc-history-icon {
  width: 26px;
  min-width: 26px;
  max-width: 26px;
  height: 26px;
  min-height: 26px;
  max-height: 26px;
  display: grid;
  place-items: center;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text-dim);
  cursor: pointer;
}

.calc-history-icon:hover { color: var(--color-primary); }
.calc-history-icon .material-symbols-outlined { font-size: 17px; }

.calc-history-list {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  scrollbar-width: thin;
}

.calc-history-empty { margin: 16px 0; text-align: center; font-size: 12.5px; color: var(--color-text-dim); }

.calc-history-row {
  display: flex;
  flex-direction: column;
  gap: 2px;
  width: 100%;
  padding: 7px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  text-align: right;
  cursor: pointer;
}

.calc-history-row:hover { background: color-mix(in oklch, var(--color-primary) 10%, transparent); }

.calc-history-expr { font-size: 11.5px; color: var(--color-text-dim); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.calc-history-value { font-size: 13.5px; font-weight: 700; font-variant-numeric: tabular-nums; }

.calc-hist-enter-active,
.calc-hist-leave-active { transition: opacity 0.16s ease, translate 0.18s cubic-bezier(0.2, 0, 0, 1); }
.calc-hist-enter-from,
.calc-hist-leave-to { opacity: 0; translate: 0 10px; }
</style>
