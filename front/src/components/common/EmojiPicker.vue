<template>
  <div ref="rootEl" class="emoji-picker-wrap">
    <button
      type="button"
      class="emoji-btn"
      :class="{ active: open }"
      title="Эмодзи"
      aria-haspopup="dialog"
      :aria-expanded="open"
      @click="toggle"
    >
      <span class="material-symbols-outlined">mood</span>
    </button>

    <!-- Попап телепортируется в body и позиционируется fixed'ом: внутри
         диалогов (overflow:hidden) абсолютный попап обрезался контейнером. -->
    <Teleport to="body">
      <Transition name="emoji-pop">
        <div
          v-if="open"
          ref="popEl"
          class="emoji-pop"
          :class="{ 'place-below': placeBelow }"
          :style="popStyle"
          role="dialog"
          @mousedown.prevent
        >
          <div class="emoji-tabs">
            <button
              v-for="cat in CATEGORIES"
              :key="cat.key"
              type="button"
              class="emoji-tab"
              :class="{ active: activeCat === cat.key }"
              :title="cat.label"
              @click="activeCat = cat.key"
            >{{ cat.icon }}</button>
          </div>
          <div ref="gridEl" class="emoji-grid">
            <button
              v-for="e in activeEmojis"
              :key="e"
              type="button"
              class="emoji-cell"
              @click="pick(e)"
            >{{ e }}</button>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onBeforeUnmount, watch, nextTick } from 'vue'
import { EMOJI_CATEGORIES } from '@/utils/emojiCatalog.js'

const emit = defineEmits(['pick'])

const CATEGORIES = EMOJI_CATEGORIES
const open = ref(false)
const activeCat = ref(CATEGORIES[0].key)
const rootEl = ref(null)
const popEl = ref(null)
const gridEl = ref(null)
const placeBelow = ref(false)
const popStyle = ref({})

const POP_WIDTH = 320
const GAP = 8
const MARGIN = 12

const activeEmojis = computed(
  () => CATEGORIES.find(c => c.key === activeCat.value)?.emojis || [],
)

function toggle() {
  open.value = !open.value
  if (open.value) nextTick(reposition)
}
function close() { open.value = false }

// Позиционируем попап относительно кнопки: по умолчанию над ней (как раньше),
// но если сверху не хватает места — раскрываем вниз.
function reposition() {
  const btn = rootEl.value?.querySelector('.emoji-btn')
  if (!btn) return
  const r = btn.getBoundingClientRect()
  const vw = window.innerWidth
  const vh = window.innerHeight
  const width = Math.min(POP_WIDTH, vw - MARGIN * 2)
  const height = popEl.value?.offsetHeight || 300

  let left = r.left
  left = Math.min(Math.max(left, MARGIN), vw - width - MARGIN)

  const spaceAbove = r.top
  const below = spaceAbove < height + GAP && (vh - r.bottom) > spaceAbove
  placeBelow.value = below
  const top = below ? r.bottom + GAP : r.top - GAP - height

  popStyle.value = {
    left: `${Math.round(left)}px`,
    top: `${Math.round(top)}px`,
    width: `${width}px`,
  }
}

function pick(emoji) {
  // Не закрываем — можно набрать серию эмодзи (как в Telegram).
  emit('pick', emoji)
}

// Смена категории — прокрутка сетки в начало.
watch(activeCat, () => { if (gridEl.value) gridEl.value.scrollTop = 0 })

function onDocPointer(e) {
  if (!open.value) return
  if (rootEl.value?.contains(e.target)) return
  if (popEl.value?.contains(e.target)) return
  close()
}
function onKey(e) { if (e.key === 'Escape' && open.value) close() }
function onReflow() { if (open.value) reposition() }

onMounted(() => {
  document.addEventListener('mousedown', onDocPointer, true)
  document.addEventListener('touchstart', onDocPointer, { capture: true, passive: true })
  document.addEventListener('keydown', onKey)
  window.addEventListener('resize', onReflow)
  window.addEventListener('scroll', onReflow, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('mousedown', onDocPointer, true)
  document.removeEventListener('touchstart', onDocPointer, true)
  document.removeEventListener('keydown', onKey)
  window.removeEventListener('resize', onReflow)
  window.removeEventListener('scroll', onReflow, true)
})

defineExpose({ close })
</script>

<style scoped>
.emoji-picker-wrap {
  position: relative;
  flex-shrink: 0;
}

.emoji-btn {
  appearance: none;
  border: 1px solid var(--acrylic-border);
  width: 42px;
  height: 42px;
  display: grid;
  place-items: center;
  cursor: pointer;
  border-radius: 50%;
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s, transform 0.12s;
}

.emoji-btn:hover,
.emoji-btn.active {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.emoji-btn:active { transform: scale(0.94); }
.emoji-btn .material-symbols-outlined { font-size: 22px; }
</style>

<!-- Попап не scoped: он телепортирован в body, вне дерева компонента. -->
<style>
.emoji-pop {
  position: fixed;
  background: var(--acrylic-bg);
  -webkit-backdrop-filter: var(--acrylic-blur);
  backdrop-filter: var(--acrylic-blur);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  z-index: 11000;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.emoji-pop .emoji-tabs {
  display: flex;
  gap: 2px;
  padding: 6px;
  border-bottom: 1px solid var(--color-outline-dim);
  overflow-x: auto;
  scrollbar-width: none;
}
.emoji-pop .emoji-tabs::-webkit-scrollbar { display: none; }

.emoji-pop .emoji-tab {
  flex: 1 0 auto;
  min-width: 34px;
  height: 32px;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 18px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.15s;
}
.emoji-pop .emoji-tab:hover { background: var(--color-surface-low); }
.emoji-pop .emoji-tab.active { background: var(--color-primary-container); }

.emoji-pop .emoji-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  gap: 2px;
  padding: 6px;
  max-height: 240px;
  overflow-y: auto;
  overscroll-behavior: contain;
}

.emoji-pop .emoji-cell {
  aspect-ratio: 1;
  display: grid;
  place-items: center;
  border: none;
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 22px;
  line-height: 1;
  cursor: pointer;
  transition: background 0.12s, transform 0.1s;
}
.emoji-pop .emoji-cell:hover { background: var(--color-surface-low); transform: scale(1.12); }
.emoji-pop .emoji-cell:active { transform: scale(0.95); }

.emoji-pop-enter-active,
.emoji-pop-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
  transform-origin: bottom left;
}
.emoji-pop.place-below.emoji-pop-enter-active,
.emoji-pop.place-below.emoji-pop-leave-active {
  transform-origin: top left;
}
.emoji-pop-enter-from,
.emoji-pop-leave-to {
  opacity: 0;
  transform: scale(0.96) translateY(6px);
}
</style>
