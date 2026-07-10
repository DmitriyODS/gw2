<template>
  <Teleport to="body">
    <Transition name="msg-ctx">
      <div
        v-if="visible"
        ref="menuEl"
        class="msg-ctx-menu"
        :style="style"
        role="menu"
        @click.stop
      >
        <div v-if="showReactions" class="msg-ctx-reactions">
          <button
            v-for="e in QUICK_REACTIONS"
            :key="e"
            class="msg-ctx-react"
            :class="{ active: myReactions.includes(e) }"
            @click="emitReact(e)"
          >{{ e }}</button>
        </div>
        <button class="msg-ctx-item" @click="emitAction('reply')">
          <span class="material-symbols-outlined">reply</span>
          <span>Ответить</span>
        </button>
        <button v-if="showEdit" class="msg-ctx-item" @click="emitAction('edit')">
          <span class="material-symbols-outlined">edit</span>
          <span>Редактировать</span>
        </button>
        <button v-if="showCopy" class="msg-ctx-item" @click="emitAction('copy')">
          <span class="material-symbols-outlined">content_copy</span>
          <span>Скопировать</span>
        </button>
        <button v-if="showForward" class="msg-ctx-item" @click="emitAction('forward')">
          <span class="material-symbols-outlined">forward</span>
          <span>Переслать</span>
        </button>
        <button v-if="showPin" class="msg-ctx-item" @click="emitAction('pin')">
          <span class="material-symbols-outlined">{{ isPinned ? 'keep_off' : 'keep' }}</span>
          <span>{{ isPinned ? 'Открепить' : 'Закрепить' }}</span>
        </button>
        <div v-if="showDelete" class="msg-ctx-divider" />
        <button v-if="showDelete" class="msg-ctx-item danger" @click="emitAction('delete')">
          <span class="material-symbols-outlined">delete</span>
          <span>Удалить</span>
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
  isPinned: { type: Boolean, default: false },
  showEdit: { type: Boolean, default: false },
  showCopy: { type: Boolean, default: true },
  showForward: { type: Boolean, default: true },
  showPin: { type: Boolean, default: true },
  showDelete: { type: Boolean, default: true },
  showReactions: { type: Boolean, default: true },
  // Эмодзи, уже поставленные текущим пользователем (подсветка в быстром ряду).
  myReactions: { type: Array, default: () => [] },
})

const QUICK_REACTIONS = ['👍', '❤️', '🎉', '😂', '👏', '🔥']

const emit = defineEmits(['close', 'action', 'react'])
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

function emitReact(emoji) {
  emit('react', emoji)
  emit('close')
}

function onDocClick() { if (props.visible) emit('close') }
function onScroll() { if (props.visible) emit('close') }

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

function onKey(e) {
  if (e.key === 'Escape' && props.visible) emit('close')
}
</script>

<style scoped>
.msg-ctx-menu {
  min-width: 200px;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 6px;
  box-shadow: var(--shadow-lg);
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.msg-ctx-reactions {
  display: flex;
  gap: 2px;
  padding: 2px;
  margin-bottom: 2px;
}

.msg-ctx-react {
  width: 36px;
  height: 36px;
  display: grid;
  place-items: center;
  border: none;
  background: transparent;
  border-radius: var(--radius-full);
  font-size: 19px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.15s, transform 0.12s;
}

.msg-ctx-react:hover {
  background: var(--color-surface-low);
  transform: scale(1.15);
}

.msg-ctx-react.active {
  background: var(--color-primary-container);
}

.msg-ctx-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border: none;
  background: transparent;
  color: var(--color-text);
  font-size: 14px;
  font-weight: 500;
  text-align: left;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}

.msg-ctx-item:hover { background: var(--color-surface-low); }

.msg-ctx-item.danger { color: var(--color-error); }
.msg-ctx-item.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}

.msg-ctx-item .material-symbols-outlined { font-size: 18px; }

.msg-ctx-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}

.msg-ctx-enter-active, .msg-ctx-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.msg-ctx-enter-from, .msg-ctx-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
