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
import { fitToViewport } from '@/utils/menuPlacement.js'

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

// Расширенный набор: позитивные, нейтральные и негативные реакции (как в
// Telegram/Slack). Одна горизонтально прокручиваемая строка (см. .msg-ctx-reactions).
const QUICK_REACTIONS = [
  '👍', '👎', '❤️', '🔥', '😂', '🎉', '👏', '🙏', '😮', '🤔',
  '😢', '😡', '🥰', '😍', '🤯', '💯', '🤝', '👀', '💩', '🤣',
]

const emit = defineEmits(['close', 'action', 'react'])
const menuEl = ref(null)
const pos = ref({ x: 0, y: 0 })
const maxH = ref(null)

const style = computed(() => ({
  position: 'fixed',
  left: pos.value.x + 'px',
  top: pos.value.y + 'px',
  zIndex: 12000,
  ...(maxH.value ? { maxHeight: maxH.value + 'px', overflowY: 'auto' } : {}),
}))

let openedAt = 0

watch(() => props.visible, async (v) => {
  if (!v) return
  openedAt = Date.now()
  pos.value = { x: props.x, y: props.y }
  maxH.value = null
  await nextTick()
  // Размещаем так, чтобы не обрезаться краем; если выше вьюпорта — ужимаем со
  // скроллом (общий хелпер, что и у контекстного меню чата).
  const el = menuEl.value
  if (!el) return
  const p = fitToViewport(el, { x: props.x, y: props.y, pad: 8 })
  pos.value = { x: p.left, y: p.top }
  maxH.value = p.maxHeight
})

// Тап-открытие: следом за touchend браузер шлёт ЭМУЛИРОВАННЫЙ click в ту же
// точку. У нижнего края экрана меню клампится ВВЕРХ и накрывает точку тапа —
// призрачный click попадал в пункт («Редактировать»/«Скопировать» у своих
// сообщений) и срабатывал сам. Первые мгновения после открытия пункты глухие.
function ghostClick() {
  return Date.now() - openedAt < 400
}

function emitAction(action) {
  if (ghostClick()) return
  emit('action', action)
  emit('close')
}

function emitReact(emoji) {
  if (ghostClick()) return
  emit('react', emoji)
  emit('close')
}

function onDocClick(e) {
  if (!props.visible) return
  // Клики внутри меню не закрывают его (mousedown не гасится @click.stop).
  if (menuEl.value?.contains(e.target)) return
  // Тап-открытие: после touchend браузер шлёт ЭМУЛИРОВАННЫЙ mousedown в ту же
  // точку — без грейса он закрывал меню в момент открытия.
  if (Date.now() - openedAt < 400) return
  emit('close')
}
function onScroll(e) {
  if (!props.visible || Date.now() - openedAt < 400) return
  // Прокрутка ленты реакций ВНУТРИ меню не должна его закрывать — закрываемся
  // только на скролл СТРАНИЦЫ/ленты чата под меню.
  if (menuEl.value?.contains(e.target)) return
  emit('close')
}

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

/* Одна строка эмодзи, прокручиваемая по горизонтали. Скролл-бар скрыт —
   ряд не разрастается во всё меню (как реакции в Telegram). */
.msg-ctx-reactions {
  display: flex;
  flex-wrap: nowrap;
  gap: 2px;
  padding: 2px;
  margin-bottom: 2px;
  max-width: 260px;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;             /* Firefox */
  -ms-overflow-style: none;          /* старый Edge */
  -webkit-overflow-scrolling: touch;
  overscroll-behavior-x: contain;
}

.msg-ctx-reactions::-webkit-scrollbar { display: none; }

.msg-ctx-react {
  flex: 0 0 auto;
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
