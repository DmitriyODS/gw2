<template>
  <Teleport to="body">
    <Transition name="task-ctx">
      <div
        v-if="visible"
        ref="menuEl"
        class="task-ctx-menu"
        :style="style"
        role="menu"
        @click.stop
      >
        <!-- Цвет карточки — личная палитра тегов (механика как у плиток заметок) -->
        <div class="task-ctx-colors">
          <button
            v-for="c in TASK_COLORS"
            :key="c.id"
            class="task-ctx-swatch"
            :class="{ active: color === c.id }"
            :style="{ background: `var(--tag-${c.id}-surface)`, borderColor: `var(--tag-${c.id}-border)` }"
            :title="c.label"
            @click="pickColor(c.id)"
          />
          <button class="task-ctx-swatch off" :class="{ active: !color }" title="Без цвета" @click="pickColor('')">
            <span class="material-symbols-outlined">format_color_reset</span>
          </button>
        </div>
        <!-- Теги задачи — мультивыбор, меню не закрывается (можно отметить
             сразу несколько) -->
        <template v-if="tags.length">
          <div class="task-ctx-divider" />
          <button class="task-ctx-item" @click.stop="tagsOpen = !tagsOpen">
            <span class="material-symbols-outlined">sell</span>
            <span>Теги</span>
            <span v-if="taskTagIds.length" class="task-ctx-count">{{ taskTagIds.length }}</span>
            <span class="material-symbols-outlined task-ctx-chevron">
              {{ tagsOpen ? 'expand_less' : 'expand_more' }}
            </span>
          </button>
          <div v-if="tagsOpen" class="task-ctx-tags">
            <button
              v-for="t in tags"
              :key="t.id"
              class="task-ctx-tag"
              :class="{ active: taskTagIds.includes(t.id) }"
              :style="{ background: `var(--tag-${t.color}-surface)`, color: `var(--tag-${t.color}-accent)` }"
              @click.stop="$emit('toggle-tag', t.id)"
            >
              <span class="material-symbols-outlined task-ctx-tag-check">
                {{ taskTagIds.includes(t.id) ? 'check_box' : 'check_box_outline_blank' }}
              </span>
              {{ t.name }}
            </button>
          </div>
        </template>
        <div class="task-ctx-divider" />
        <button class="task-ctx-item" @click="emitAction('open')">
          <span class="material-symbols-outlined">open_in_new</span>
          <span>Открыть</span>
        </button>
        <button v-if="canEdit" class="task-ctx-item" @click="emitAction('edit')">
          <span class="material-symbols-outlined">edit</span>
          <span>Изменить</span>
        </button>
        <button
          v-if="!isArchived"
          class="task-ctx-item"
          @click="emitAction(isRunning ? 'stop-unit' : 'start-unit')"
        >
          <span class="material-symbols-outlined">{{ isRunning ? 'stop' : 'play_arrow' }}</span>
          <span>{{ isRunning ? 'Остановить юнит' : 'Начать юнит' }}</span>
        </button>
        <button class="task-ctx-item" @click="emitAction('send')">
          <span class="material-symbols-outlined">send</span>
          <span>Отправить</span>
        </button>
        <div v-if="canEdit && !isArchived" class="task-ctx-divider" />
        <button
          v-if="canEdit && !isArchived"
          class="task-ctx-item danger"
          @click="emitAction('archive')"
        >
          <span class="material-symbols-outlined">archive</span>
          <span>В архив</span>
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { TASK_COLORS } from '@/utils/taskColors.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  canEdit: { type: Boolean, default: true },
  isArchived: { type: Boolean, default: false },
  isRunning: { type: Boolean, default: false },
  // Текущий личный цвет задачи ('' — без цвета) — для отметки в палитре.
  color: { type: String, default: '' },
  // Справочник тегов компании + отмеченные у задачи (мультивыбор в меню).
  tags: { type: Array, default: () => [] },
  taskTagIds: { type: Array, default: () => [] },
})

const emit = defineEmits(['close', 'action', 'color', 'toggle-tag'])

const tagsOpen = ref(false)

function pickColor(id) {
  emit('color', id)
  emit('close')
}
const menuEl = ref(null)
const pos = ref({ x: 0, y: 0 })

const style = computed(() => ({
  position: 'fixed',
  left: pos.value.x + 'px',
  top: pos.value.y + 'px',
  zIndex: 12000,
}))

// Кламп в вьюпорт, чтобы меню не выезжало за край; снизу отступ больше —
// меню у нижней кромки поджимается вверх, а не обрезается.
async function clampToViewport() {
  await nextTick()
  const el = menuEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const pad = 8
  const padBottom = 16
  let nx = pos.value.x
  let ny = pos.value.y
  if (nx + r.width > window.innerWidth - pad) nx = window.innerWidth - r.width - pad
  if (ny + r.height > window.innerHeight - padBottom) ny = window.innerHeight - r.height - padBottom
  if (nx < pad) nx = pad
  if (ny < pad) ny = pad
  pos.value = { x: nx, y: ny }
}

watch(() => props.visible, (v) => {
  if (!v) {
    tagsOpen.value = false
    return
  }
  pos.value = { x: props.x, y: props.y }
  clampToViewport()
})

// Раскрытие секции тегов меняет высоту меню — переклампливаем, иначе низ
// уезжает за экран.
watch(tagsOpen, (v) => { if (v) clampToViewport() })

function emitAction(action) {
  emit('action', action)
  emit('close')
}

// Клики ВНУТРИ меню не закрывают его — мультивыбор тегов требует серии
// кликов (раньше спасал только зазор transition-анимации).
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
.task-ctx-menu {
  min-width: 220px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md, 12px);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.task-ctx-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
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

.task-ctx-item:hover { background: var(--color-surface-low); }
.task-ctx-item.danger { color: var(--color-error); }
.task-ctx-item.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.task-ctx-item .material-symbols-outlined { font-size: 18px; }

.task-ctx-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}

/* ── Теги ── */
.task-ctx-count {
  margin-left: auto;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  font-size: 11px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.task-ctx-chevron { margin-left: 2px; color: var(--color-text-dim); }
.task-ctx-item .task-ctx-chevron { font-size: 18px; }

.task-ctx-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 5px;
  padding: 4px 10px 8px;
  max-width: 260px;
}

.task-ctx-tag {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  border: 1px solid transparent;
  border-radius: var(--radius-full);
  padding: 5px 10px;
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  cursor: pointer;
  opacity: 0.6;
  transition: opacity 0.12s;
}
.task-ctx-tag.active { opacity: 1; outline: 2px solid currentColor; outline-offset: -2px; }
.task-ctx-tag-check { font-size: 14px; }

.task-ctx-colors {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px 6px;
}
.task-ctx-swatch {
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  border: 1px solid;
  cursor: pointer;
  padding: 0;
  flex-shrink: 0;
}
.task-ctx-swatch.active {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
}
.task-ctx-swatch.off {
  display: grid;
  place-items: center;
  background: var(--color-surface);
  border-color: var(--color-outline-variant);
  color: var(--color-text-dim);
}
.task-ctx-swatch.off .material-symbols-outlined { font-size: 15px; }

.task-ctx-enter-active, .task-ctx-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.task-ctx-enter-from, .task-ctx-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
