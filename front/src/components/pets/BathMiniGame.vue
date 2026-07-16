<template>
  <Teleport to="body">
    <div class="mg-overlay" @click.self="emit('close')">
      <div class="mg-panel">
        <header class="mg-head">
          <h3>Купание</h3>
          <button class="mg-close" type="button" @click="emit('close')" aria-label="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </header>
        <p class="mg-hint">
          Водите губкой по питомцу, пока не отмоется — купание
          <KudosCoin class="mg-hint-coin" /> {{ cost }}
        </p>

        <div
          ref="zoneEl"
          class="mg-bath-zone"
          @pointerdown="onDown"
          @pointermove="onMove"
          @pointerup="onUp"
          @pointercancel="onUp"
        >
          <span class="mg-water" aria-hidden="true"></span>
          <span
            ref="petEl"
            class="mg-bath-emoji"
            :class="{ scrubbing: rubbing && overPet, clean: progress >= 100 }"
          ><EmojiGlyph :char="emoji" /></span>
          <span class="mg-sponge" :class="{ scrubbing: rubbing && overPet }" :style="spongeStyle" aria-hidden="true">🧽</span>
          <transition-group name="mg-bubble" tag="div" class="mg-bubbles" aria-hidden="true">
            <span v-for="b in bubbles" :key="b.id" class="mg-bubble" :style="{ left: b.left + '%' }">🫧</span>
          </transition-group>
        </div>

        <div class="mg-progress"><div class="mg-progress-fill" :style="{ width: progress + '%' }"></div></div>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, ref } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import { createRubTracker, isInHitZone } from '@/utils/miniGames.js'
import { petEmoji } from '@/utils/pets.js'

const props = defineProps({
  pet: { type: Object, default: null },
  cost: { type: Number, required: true },
})
const emit = defineEmits(['success', 'close'])

const RUB_THRESHOLD_PX = 520 // чуть дольше поглаживания — отмыть сложнее

const emoji = computed(() => petEmoji(props.pet))

const zoneEl = ref(null)
const petEl = ref(null)
const rubbing = ref(false)
const overPet = ref(false)
const progress = ref(0)
const bubbles = ref([])
const sponge = ref({ x: 50, y: 55 })

const tracker = createRubTracker(RUB_THRESHOLD_PX)
let last = null
let bubbleSeq = 0
let bubbleDist = 0
let done = false

const spongeStyle = computed(() => ({ left: `${sponge.value.x}%`, top: `${sponge.value.y}%` }))

function relative(e) {
  const rect = zoneEl.value?.getBoundingClientRect()
  if (!rect || !rect.width) return null
  return {
    x: Math.min(96, Math.max(4, ((e.clientX - rect.left) / rect.width) * 100)),
    y: Math.min(90, Math.max(10, ((e.clientY - rect.top) / rect.height) * 100)),
  }
}

// Трение засчитывается только над самим питомцем — как в поглаживании.
function isOverPet(e) {
  const rect = petEl.value?.getBoundingClientRect()
  if (!rect || !rect.width) return false
  const cx = rect.left + rect.width / 2
  const cy = rect.top + rect.height / 2
  return isInHitZone(e.clientX, e.clientY, cx, cy, Math.max(rect.width, rect.height) / 2 + 14)
}

function spawnBubble(left) {
  const id = bubbleSeq++
  bubbles.value.push({ id, left })
  if (bubbles.value.length > 8) bubbles.value.shift()
  setTimeout(() => { bubbles.value = bubbles.value.filter((b) => b.id !== id) }, 700)
}

function onDown(e) {
  if (done) return
  rubbing.value = true
  overPet.value = isOverPet(e)
  last = overPet.value ? { x: e.clientX, y: e.clientY } : null
  const p = relative(e)
  if (p) sponge.value = p
  try { e.currentTarget?.setPointerCapture?.(e.pointerId) } catch { /* noop */ }
}

function onMove(e) {
  if (!rubbing.value || done) return
  const p = relative(e)
  if (p) sponge.value = p

  overPet.value = isOverPet(e)
  if (!overPet.value) {
    // Вышли за питомца — рвём жест, чтобы прыжок курсора не засчитался трением.
    last = null
    return
  }
  if (!last) { last = { x: e.clientX, y: e.clientY }; return }
  const dx = e.clientX - last.x
  const dy = e.clientY - last.y
  last = { x: e.clientX, y: e.clientY }

  bubbleDist += Math.abs(dx) + Math.abs(dy)
  if (bubbleDist > 100) {
    bubbleDist = 0
    spawnBubble(sponge.value.x)
  }

  const r = tracker.add(dx, dy)
  progress.value = r.completed ? 100 : r.progress
  if (r.completed) {
    done = true
    rubbing.value = false
    setTimeout(() => emit('success'), 320) // дать увидеть отмытого питомца
  }
}

function onUp() {
  rubbing.value = false
  overPet.value = false
  last = null
}
</script>

<style scoped>
.mg-overlay {
  position: fixed;
  inset: 0;
  z-index: 10900;
  background: color-mix(in oklch, var(--color-scrim, var(--color-text)) 32%, transparent);
  display: grid;
  place-items: center;
}
.mg-panel {
  width: min(380px, calc(100vw - 32px));
  background: var(--color-surface);
  border-radius: 24px;
  box-shadow: var(--shadow-lg);
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.mg-head { display: flex; align-items: center; justify-content: space-between; }
.mg-head h3 { margin: 0; font-size: 16px; font-weight: 700; }
.mg-close {
  width: 32px; height: 32px; min-height: 0; border-radius: 50%; border: none; background: none;
  color: var(--color-text-dim); display: grid; place-items: center; cursor: pointer;
}
.mg-close:hover { background: var(--color-surface-high); }
.mg-hint { margin: 0; font-size: 12.5px; color: var(--color-text-dim); line-height: 1.4; }
.mg-hint-coin { font-size: 12px; }

.mg-bath-zone {
  position: relative;
  height: 170px;
  border-radius: 18px;
  display: grid;
  place-items: center;
  touch-action: none;
  cursor: grab;
  overflow: hidden;
  user-select: none;
  background: linear-gradient(135deg, var(--tag-blue-surface), var(--tag-teal-surface));
}
.mg-bath-zone:active { cursor: grabbing; }

/* Вода в тазике — статичный слой снизу, чтобы сцена читалась как купание. */
.mg-water {
  position: absolute;
  left: 0; right: 0; bottom: 0;
  height: 34%;
  background: color-mix(in oklch, var(--tag-blue-accent) 22%, transparent);
  border-top: 2px solid color-mix(in oklch, var(--tag-blue-accent) 45%, transparent);
}

.mg-bath-emoji {
  position: relative;
  font-size: 60px;
  line-height: 1;
  transition: transform 0.2s, filter 0.4s;
  filter: sepia(0.35) saturate(0.8);
}
.mg-bath-emoji.scrubbing { transform: scale(1.05) rotate(-3deg); }
.mg-bath-emoji.clean { filter: none; transform: scale(1.12); }

.mg-sponge {
  position: absolute;
  font-size: 28px;
  line-height: 1;
  transform: translate(-50%, -50%) rotate(-12deg);
  opacity: 0.5;
  pointer-events: none;
  transition: opacity 0.15s;
}
.mg-sponge.scrubbing { opacity: 1; }

.mg-bubbles { position: absolute; inset: 0; pointer-events: none; }
.mg-bubble {
  position: absolute;
  bottom: 16%;
  font-size: 18px;
  transform: translateX(-50%);
}
.mg-bubble-enter-active { transition: opacity 0.15s, transform 0.6s ease-out; }
.mg-bubble-enter-from { opacity: 0; }
.mg-bubble-leave-active { transition: opacity 0.4s, transform 0.4s; }
.mg-bubble-leave-to { opacity: 0; transform: translate(-50%, -40px); }

.mg-progress {
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.mg-progress-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--color-primary);
  transition: width 0.08s linear;
}
</style>
