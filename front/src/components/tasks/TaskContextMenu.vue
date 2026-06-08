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

const props = defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  canEdit: { type: Boolean, default: true },
  isArchived: { type: Boolean, default: false },
  isRunning: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'action'])
const menuEl = ref(null)
const pos = ref({ x: 0, y: 0 })

const style = computed(() => ({
  position: 'fixed',
  left: pos.value.x + 'px',
  top: pos.value.y + 'px',
  zIndex: 12000,
}))

watch(() => props.visible, async (v) => {
  if (!v) return
  pos.value = { x: props.x, y: props.y }
  await nextTick()
  // Кламп в вьюпорт, чтобы меню не выезжало за край.
  const el = menuEl.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const pad = 8
  let nx = pos.value.x
  let ny = pos.value.y
  if (nx + r.width > window.innerWidth - pad) nx = window.innerWidth - r.width - pad
  if (ny + r.height > window.innerHeight - pad) ny = window.innerHeight - r.height - pad
  if (nx < pad) nx = pad
  if (ny < pad) ny = pad
  pos.value = { x: nx, y: ny }
})

function emitAction(action) {
  emit('action', action)
  emit('close')
}

function onDocClick() { if (props.visible) emit('close') }
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
  background: var(--color-surface);
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

.task-ctx-enter-active, .task-ctx-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.task-ctx-enter-from, .task-ctx-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
