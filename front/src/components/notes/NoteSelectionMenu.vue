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
            @mouseenter="openSub = null"
            @click="pickAi(item)"
          >
            <span class="material-symbols-outlined">{{ item.icon }}</span>
            <span>{{ item.label }}</span>
          </button>

          <div
            v-for="group in AI_GROUPS"
            :key="group.key"
            class="nsm-sub"
            @mouseenter="openSub = group.key"
            @mouseleave="openSub === group.key && (openSub = null)"
          >
            <button class="nsm-item" @click="toggleSub(group.key)">
              <span class="material-symbols-outlined">{{ group.icon }}</span>
              <span>{{ group.label }}</span>
              <span class="material-symbols-outlined nsm-arrow">chevron_right</span>
            </button>
            <div v-if="openSub === group.key" :ref="setFlyoutEl" class="nsm-flyout" :class="{ left: flyLeft, up: flyUp }">
              <button
                v-for="(opt, i) in group.options"
                :key="i"
                class="nsm-item"
                @click="pickAi(opt, group)"
              >
                <span v-if="opt.icon" class="material-symbols-outlined">{{ opt.icon }}</span>
                <span>{{ opt.label }}</span>
              </button>
            </div>
          </div>

          <div class="nsm-divider" />
        </template>

        <div
          class="nsm-sub"
          @mouseenter="openSub = 'create'"
          @mouseleave="openSub === 'create' && (openSub = null)"
        >
          <button class="nsm-item" @click="toggleSub('create')">
            <span class="material-symbols-outlined">add_circle</span>
            <span>Создать из выделенного</span>
            <span class="material-symbols-outlined nsm-arrow">chevron_right</span>
          </button>
          <div v-if="openSub === 'create'" :ref="setFlyoutEl" class="nsm-flyout" :class="{ left: flyLeft, up: flyUp }">
            <button v-if="canTask" class="nsm-item" @click="pickCreate('task')">
              <span class="material-symbols-outlined">task_alt</span>
              <span>Задачу</span>
            </button>
            <button class="nsm-item" @click="pickCreate('diary')">
              <span class="material-symbols-outlined">book</span>
              <span>Пункт в ежедневник</span>
            </button>
          </div>
        </div>

        <div class="nsm-divider" />
        <button class="nsm-item" @mouseenter="openSub = null" @click="pickCopy">
          <span class="material-symbols-outlined">content_copy</span>
          <span>Копировать</span>
        </button>
      </div>
    </Transition>
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
const openSub = ref(null)
// Куда раскрывать flyout: влево — когда справа не влезает, вверх — когда
// подменю упирается в нижний край.
const flyLeft = ref(false)
const flyUp = ref(false)
const flyoutEl = ref(null)
function setFlyoutEl(el) { if (el) flyoutEl.value = el }

const style = computed(() => ({
  position: 'fixed',
  left: pos.value.x + 'px',
  top: pos.value.y + 'px',
  zIndex: 12000,
}))

watch(() => props.visible, async (v) => {
  if (!v) { openSub.value = null; return }
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
  flyLeft.value = nx + r.width + 210 > window.innerWidth - PAD
})

// Открывшийся flyout меряем по факту: не влезает снизу — прижимаем к низу
// строки-родителя (класс up).
watch(openSub, async (v) => {
  flyUp.value = false
  if (!v) return
  await nextTick()
  const r = flyoutEl.value?.getBoundingClientRect()
  if (r && r.bottom > window.innerHeight - BOTTOM_GAP) flyUp.value = true
})

function toggleSub(key) { openSub.value = openSub.value === key ? null : key }

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
  if (props.visible && !menuEl.value?.contains(e.target)) emit('close')
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
  document.removeEventListener('mousedown', onDocClick, true)
  document.removeEventListener('touchstart', onDocClick, true)
  document.removeEventListener('scroll', onScroll, true)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.nsm {
  /* Без overflow: он обрезал бы flyout-подменю; в экран меню вписывает
     кламп позиции (с переворотом вверх у нижнего края). */
  min-width: 230px;
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
.nsm-item:hover { background: var(--color-surface-low); }
.nsm-item .material-symbols-outlined { font-size: 18px; color: var(--color-text-dim); }
.nsm-item:hover .material-symbols-outlined { color: var(--color-primary); }
.nsm-arrow { margin-left: auto; }

.nsm-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}

.nsm-sub { position: relative; }
.nsm-flyout {
  position: absolute;
  top: -6px;
  left: calc(100% + 2px);
  min-width: 190px;
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
  z-index: 1;
}
.nsm-flyout.left { left: auto; right: calc(100% + 2px); }
.nsm-flyout.up { top: auto; bottom: -6px; }

.nsm-enter-active, .nsm-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.nsm-enter-from, .nsm-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
