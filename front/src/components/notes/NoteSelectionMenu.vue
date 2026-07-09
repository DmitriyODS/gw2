<template>
  <Teleport to="body">
    <Transition name="nsm">
      <div
        v-if="visible"
        ref="menuEl"
        class="nsm"
        :style="style"
        role="menu"
        @click.stop
      >
        <template v-if="aiAvailable">
          <div class="nsm-caption">
            <span class="material-symbols-outlined">auto_awesome</span>
            <span>ИИ с выделенным</span>
          </div>
          <button
            v-for="item in AI_TOP"
            :key="item.action"
            class="nsm-item"
            @mouseenter="closeFly"
            @click="pickAi(item)"
          >
            <span class="material-symbols-outlined">{{ item.icon }}</span>
            <span>{{ item.label }}</span>
          </button>

          <button
            v-for="group in AI_GROUPS"
            :key="group.key"
            class="nsm-item"
            :class="{ open: openSub === group.key }"
            @mouseenter="openFly(group.key, $event)"
            @mouseleave="scheduleFlyClose"
            @click="openFly(group.key, $event, true)"
          >
            <span class="material-symbols-outlined">{{ group.icon }}</span>
            <span>{{ group.label }}</span>
            <span class="material-symbols-outlined nsm-arrow">chevron_right</span>
          </button>

          <div class="nsm-divider" />
        </template>

        <button
          class="nsm-item"
          :class="{ open: openSub === 'create' }"
          @mouseenter="openFly('create', $event)"
          @mouseleave="scheduleFlyClose"
          @click="openFly('create', $event, true)"
        >
          <span class="material-symbols-outlined">add_circle</span>
          <span>Создать из выделенного</span>
          <span class="material-symbols-outlined nsm-arrow">chevron_right</span>
        </button>

        <div class="nsm-divider" />
        <button class="nsm-item" @mouseenter="closeFly" @click="pickCopy">
          <span class="material-symbols-outlined">content_copy</span>
          <span>Копировать</span>
        </button>
      </div>
    </Transition>

    <!-- Flyout — НЕ вложен в .nsm: backdrop-filter родителя делает его
         backdrop root'ом, и блюр вложенного подменю не захватывал бы страницу
         за пределами меню (стекло «пропадает»). Позиция — от строки-родителя. -->
    <div
      v-if="visible && activeGroup"
      ref="flyEl"
      class="nsm-flyout"
      :style="flyStyle"
      role="menu"
      @click.stop
      @mouseenter="cancelFlyClose"
      @mouseleave="scheduleFlyClose"
    >
      <button
        v-for="(opt, i) in activeGroup.options"
        :key="i"
        class="nsm-item"
        @click="pickOption(opt)"
      >
        <span v-if="opt.icon" class="material-symbols-outlined">{{ opt.icon }}</span>
        <span>{{ opt.label }}</span>
      </button>
    </div>
  </Teleport>
</template>

<script setup>
// Контекстное меню выделенного текста в редакторе заметки (ПКМ). Пункты
// сгруппированы во вложенные flyout-подменю, чтобы верхний уровень оставался
// коротким; часто нужные «Улучшить»/«Исправить» — сразу на верхнем уровне.
// Позиционирование: у нижнего края меню переворачивается вверх от курсора
// (как нативные меню), flyout при нехватке места уходит влево/вверх.
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'

const props = defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  // Есть активная компания → доступны ИИ-действия и создание задачи.
  aiAvailable: { type: Boolean, default: false },
  canTask: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'ai', 'create', 'copy'])

const AI_TOP = [
  { action: 'improve', icon: 'auto_fix_high', label: 'Улучшить текст' },
  { action: 'fix', icon: 'spellcheck', label: 'Исправить ошибки' },
]

// options: { action, style?, icon?, label } — заголовок диалога результата
// собирается из label группы и пункта (pickAi).
const AI_GROUPS = [
  {
    key: 'rewrite', icon: 'cached', label: 'Переписать',
    options: [
      { action: 'rephrase', icon: 'sync_alt', label: 'Переформулировать' },
      { action: 'shorten', icon: 'compress', label: 'Сократить' },
      { action: 'expand', icon: 'expand_content', label: 'Развернуть' },
      { action: 'simplify', icon: 'lightbulb', label: 'Упростить' },
      { action: 'bullets', icon: 'format_list_bulleted', label: 'В тезисы' },
    ],
  },
  {
    key: 'tone', icon: 'theater_comedy', label: 'Сменить тон',
    options: [
      { action: 'tone', style: 'formal', label: 'Деловой' },
      { action: 'tone', style: 'friendly', label: 'Дружелюбный' },
      { action: 'tone', style: 'confident', label: 'Уверенный' },
      { action: 'tone', style: 'casual', label: 'Непринуждённый' },
    ],
  },
  {
    key: 'translate', icon: 'translate', label: 'Перевести',
    options: [
      { action: 'translate', style: 'en', label: 'На английский' },
      { action: 'translate', style: 'ru', label: 'На русский' },
    ],
  },
  {
    key: 'compose', icon: 'edit_note', label: 'Сочинить',
    options: [
      { action: 'continue', icon: 'resume', label: 'Продолжить текст' },
      { action: 'summarize', icon: 'summarize', label: 'Резюме' },
    ],
  },
]

const PAD = 8        // отступ от краёв вьюпорта
const BOTTOM_GAP = 16 // отступ меню от нижнего края

const menuEl = ref(null)
const pos = ref({ x: 0, y: 0 })

const style = computed(() => ({
  position: 'fixed',
  left: pos.value.x + 'px',
  top: pos.value.y + 'px',
  zIndex: 12000,
}))

watch(() => props.visible, async (v) => {
  if (!v) { closeFly(); return }
  pos.value = { x: props.x, y: props.y }
  await nextTick()
  const el = menuEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  let nx = pos.value.x
  let ny = pos.value.y
  if (nx + r.width > window.innerWidth - PAD) nx = window.innerWidth - r.width - PAD
  if (nx < PAD) nx = PAD
  // Снизу не влезает с отступом — переворачиваем вверх от курсора.
  if (ny + r.height > window.innerHeight - BOTTOM_GAP) ny = pos.value.y - r.height - 6
  if (ny < PAD) ny = Math.max(PAD, window.innerHeight - BOTTOM_GAP - r.height)
  pos.value = { x: nx, y: ny }
})

// ── Flyout-подменю ──
const openSub = ref(null)
const flyEl = ref(null)
const flyPos = ref({ x: 0, y: 0 })
let flyCloseTimer = null

const createGroup = computed(() => ({
  key: 'create',
  options: [
    ...(props.canTask ? [{ create: 'task', icon: 'task_alt', label: 'Задачу' }] : []),
    { create: 'diary', icon: 'book', label: 'Пункт в ежедневник' },
  ],
}))

const activeGroup = computed(() => {
  if (!openSub.value) return null
  if (openSub.value === 'create') return createGroup.value
  return AI_GROUPS.find((g) => g.key === openSub.value) || null
})

const flyStyle = computed(() => ({
  position: 'fixed',
  left: flyPos.value.x + 'px',
  top: flyPos.value.y + 'px',
  zIndex: 12001,
}))

// openFly — открыть подменю от строки-родителя (hover или тап; toggle — тап
// по уже открытой строке закрывает). Позиция меряется по факту: не влезает
// справа — уходит влево от меню, снизу — прижимается к низу строки.
async function openFly(key, e, toggle = false) {
  cancelFlyClose()
  if (toggle && openSub.value === key) { openSub.value = null; return }
  const anchor = e.currentTarget.getBoundingClientRect()
  openSub.value = key
  flyPos.value = { x: anchor.right + 2, y: anchor.top - 6 }
  await nextTick()
  const r = flyEl.value?.getBoundingClientRect()
  if (!r) return
  let nx = flyPos.value.x
  let ny = flyPos.value.y
  if (nx + r.width > window.innerWidth - PAD) nx = anchor.left - r.width - 2
  if (nx < PAD) nx = PAD
  if (ny + r.height > window.innerHeight - BOTTOM_GAP) ny = anchor.bottom - r.height + 6
  if (ny < PAD) ny = PAD
  flyPos.value = { x: nx, y: ny }
}

function closeFly() {
  cancelFlyClose()
  openSub.value = null
}

// Между строкой и flyout есть зазор в пару пикселей — закрываем с задержкой,
// чтобы курсор успел переехать.
function scheduleFlyClose() {
  cancelFlyClose()
  flyCloseTimer = setTimeout(() => { openSub.value = null }, 150)
}

function cancelFlyClose() {
  clearTimeout(flyCloseTimer)
  flyCloseTimer = null
}

function pickOption(opt) {
  if (opt.create) pickCreate(opt.create)
  else pickAi(opt, activeGroup.value)
}

function pickAi(opt, group = null) {
  const label = group && opt.style ? `${group.label}: ${opt.label.toLowerCase()}` : opt.label
  emit('ai', { action: opt.action, style: opt.style ?? null, label })
  emit('close')
}

function pickCreate(kind) {
  emit('create', kind)
  emit('close')
}

function pickCopy() {
  emit('copy')
  emit('close')
}

function onDocClick(e) {
  if (!props.visible) return
  if (menuEl.value?.contains(e.target) || flyEl.value?.contains(e.target)) return
  emit('close')
}
function onScroll() { if (props.visible) emit('close') }
function onKey(e) { if (e.key === 'Escape' && props.visible) emit('close') }

onMounted(() => {
  document.addEventListener('mousedown', onDocClick, true)
  document.addEventListener('touchstart', onDocClick, { passive: true, capture: true })
  document.addEventListener('scroll', onScroll, true)
  document.addEventListener('keydown', onKey)
})

onBeforeUnmount(() => {
  cancelFlyClose()
  document.removeEventListener('mousedown', onDocClick, true)
  document.removeEventListener('touchstart', onDocClick, true)
  document.removeEventListener('scroll', onScroll, true)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.nsm,
.nsm-flyout {
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md, 12px);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  gap: 1px;
}
.nsm { min-width: 230px; }
.nsm-flyout { min-width: 190px; }

.nsm-caption {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px 4px;
  color: var(--color-primary);
  font-size: 11.5px;
  font-weight: 800;
  text-transform: uppercase;
  letter-spacing: 0.4px;
}
.nsm-caption .material-symbols-outlined { font-size: 16px; }

.nsm-item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 9px 12px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font: inherit;
  font-size: 14px;
  font-weight: 500;
  text-align: left;
  border-radius: var(--radius-sm, 8px);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.nsm-item:hover,
.nsm-item.open { background: var(--color-surface-low); }
.nsm-item .material-symbols-outlined { font-size: 18px; color: var(--color-text-dim); }
.nsm-item:hover .material-symbols-outlined,
.nsm-item.open .material-symbols-outlined { color: var(--color-primary); }
.nsm-arrow { margin-left: auto; }

.nsm-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}

.nsm-enter-active, .nsm-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.nsm-enter-from, .nsm-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
