<template>
  <Teleport to="body">
    <Transition name="note-ctx">
      <div
        v-if="visible"
        ref="menuEl"
        class="note-ctx-menu"
        :style="style"
        role="menu"
        @click.stop
      >
        <!-- Цвет плитки — палитра тегов задач -->
        <div class="note-ctx-colors">
          <button
            v-for="c in TASK_COLORS"
            :key="c.id"
            class="note-ctx-swatch"
            :class="{ active: color === c.id }"
            :style="{ background: `var(--tag-${c.id}-surface)`, borderColor: `var(--tag-${c.id}-border)` }"
            :title="c.label"
            @click="pickColor(c.id)"
          />
          <button class="note-ctx-swatch off" :class="{ active: !color }" title="Без цвета" @click="pickColor('')">
            <span class="material-symbols-outlined">format_color_reset</span>
          </button>
        </div>
        <div class="note-ctx-divider" />
        <button class="note-ctx-item" @click="emitAction('open')">
          <span class="material-symbols-outlined">edit_note</span>
          <span>Открыть</span>
        </button>
        <button class="note-ctx-item" @click="emitAction('groups')">
          <span class="material-symbols-outlined">folder</span>
          <span>Группы</span>
        </button>
        <button class="note-ctx-item" @click="emitAction('pin')">
          <span class="material-symbols-outlined">{{ pinned ? 'keep_off' : 'keep' }}</span>
          <span>{{ pinned ? 'Открепить' : 'Закрепить' }}</span>
        </button>
        <button class="note-ctx-item" @click="emitAction('share')">
          <span class="material-symbols-outlined">share</span>
          <span>Поделиться</span>
        </button>
        <button class="note-ctx-item" @click="emitAction('send-chat')">
          <span class="material-symbols-outlined">send</span>
          <span>Отправить в чат</span>
        </button>
        <button v-if="canPost" class="note-ctx-item" @click="emitAction('publish')">
          <span class="material-symbols-outlined">campaign</span>
          <span>Опубликовать на портале</span>
        </button>
        <button class="note-ctx-item" @click="emitAction('export')">
          <span class="material-symbols-outlined">download</span>
          <span>Экспорт .txt</span>
        </button>
        <button class="note-ctx-item" @click="emitAction('archive')">
          <span class="material-symbols-outlined">{{ archived ? 'unarchive' : 'archive' }}</span>
          <span>{{ archived ? 'Вернуть из архива' : 'В архив' }}</span>
        </button>
        <div class="note-ctx-divider" />
        <button class="note-ctx-item danger" @click="emitAction('delete')">
          <span class="material-symbols-outlined">delete</span>
          <span>Удалить</span>
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup>
// Контекстное меню плитки заметки (ПКМ на десктопе, long-press на таче) —
// по образцу TaskContextMenu: teleport в body, кламп в вьюпорт, закрытие по
// клику мимо/скроллу/Esc.
import { computed, nextTick, ref, watch, onMounted, onBeforeUnmount } from 'vue'
import { TASK_COLORS } from '@/utils/taskColors.js'

const props = defineProps({
  visible: { type: Boolean, default: false },
  x: { type: Number, default: 0 },
  y: { type: Number, default: 0 },
  // Текущий цвет заметки ('' — без цвета) — для отметки в палитре.
  color: { type: String, default: '' },
  // Архивна ли заметка — меняет пункт «В архив» на «Вернуть из архива».
  archived: { type: Boolean, default: false },
  // Закреплена ли — меняет пункт «Закрепить» на «Открепить».
  pinned: { type: Boolean, default: false },
  // Есть активная компания — доступна публикация на корпоративном портале.
  canPost: { type: Boolean, default: false },
})

const emit = defineEmits(['close', 'action', 'color'])

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
.note-ctx-menu {
  min-width: 210px;
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

.note-ctx-item {
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

.note-ctx-item:hover { background: var(--color-surface-low); }
.note-ctx-item.danger { color: var(--color-error); }
.note-ctx-item.danger:hover {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
}
.note-ctx-item .material-symbols-outlined { font-size: 18px; }

.note-ctx-divider {
  height: 1px;
  background: var(--color-outline-dim);
  margin: 4px 4px;
}

.note-ctx-colors {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 10px 6px;
}
.note-ctx-swatch {
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  border: 1px solid;
  cursor: pointer;
  padding: 0;
  flex-shrink: 0;
}
.note-ctx-swatch.active {
  outline: 2px solid var(--color-primary);
  outline-offset: 1px;
}
.note-ctx-swatch.off {
  display: grid;
  place-items: center;
  background: var(--color-surface);
  border-color: var(--color-outline-variant);
  color: var(--color-text-dim);
}
.note-ctx-swatch.off .material-symbols-outlined { font-size: 15px; }

.note-ctx-enter-active, .note-ctx-leave-active {
  transition: opacity 0.14s, transform 0.14s;
  transform-origin: top left;
}
.note-ctx-enter-from, .note-ctx-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(-4px);
}
</style>
