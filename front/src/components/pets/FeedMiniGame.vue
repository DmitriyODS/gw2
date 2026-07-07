<template>
  <Teleport to="body">
    <div class="mg-overlay" @click.self="$emit('close')">
      <div class="mg-panel">
        <header class="mg-head">
          <h3>{{ sick ? 'Дать бульон' : 'Покормить' }}</h3>
          <button class="mg-close" type="button" @click="$emit('close')" aria-label="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </header>
        <p class="mg-hint">Питомец разыгрался — поймайте момент и дайте ему еду</p>

        <div ref="areaEl" class="mg-area">
          <div
            ref="mouthEl"
            class="mg-pet"
            :class="{ chew: state === 'success' }"
            :style="petStyle"
          >
            <span class="mg-pet-emoji">{{ petEmojiChar }}</span>
          </div>

          <div
            ref="foodEl"
            class="mg-food"
            :class="{ dragging, success: state === 'success', miss: state === 'miss' }"
            :style="foodStyle"
            @pointerdown="onPointerDown"
          >{{ foodEmoji }}</div>
        </div>

        <p v-if="state === 'miss'" class="mg-feedback miss">Мимо — попробуйте ещё раз</p>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, nextTick, onBeforeUnmount, reactive, ref } from 'vue'
import { isInHitZone } from '@/utils/miniGames.js'
import { petEmoji } from '@/utils/pets.js'

const props = defineProps({
  pet: { type: Object, default: null },
})
const emit = defineEmits(['success', 'close'])

const sick = computed(() => !!props.pet?.sick)
const foodEmoji = computed(() => (sick.value ? '🍲' : '🥕'))
const petEmojiChar = computed(() => petEmoji(props.pet))

const areaEl = ref(null)
const mouthEl = ref(null)
const foodEl = ref(null)
const dragging = ref(false)
const state = ref('idle') // idle | success | miss
const origin = reactive({ x: 0, y: 0 })
const offset = reactive({ x: 0, y: 0 })
let dragStart = { x: 0, y: 0 }

const foodStyle = computed(() => ({
  transform: `translate(${offset.x}px, ${offset.y}px)`,
}))

// Питомец «разыгрался»: непрерывно бегает по площадке — композиция двух
// синусоид даёт живую, слабо предсказуемую траекторию. Попадание считается
// по реальному положению рта (getBoundingClientRect) в момент броска.
const petX = ref(50)
let petRaf = null
function petTick(ts) {
  petX.value = 50 + 24 * Math.sin(ts * 0.0016) + 10 * Math.sin(ts * 0.0037 + 1.7)
  petRaf = requestAnimationFrame(petTick)
}
petRaf = requestAnimationFrame(petTick)

const petStyle = computed(() => ({
  left: `${petX.value}%`,
}))

function onPointerDown(e) {
  if (state.value === 'success') return
  dragging.value = true
  dragStart = { x: e.clientX - offset.x, y: e.clientY - offset.y }
  // Capture держит поток pointer-событий на еде даже при быстром движении
  // пальца за пределы элемента — без него drag на таче теряется.
  try { e.target.setPointerCapture(e.pointerId) } catch { /* не поддерживается — drag через window */ }
  window.addEventListener('pointermove', onPointerMove)
  window.addEventListener('pointerup', onPointerUp)
  window.addEventListener('pointercancel', onPointerCancel)
  e.preventDefault()
}

function onPointerMove(e) {
  if (!dragging.value) return
  offset.x = e.clientX - dragStart.x
  offset.y = e.clientY - dragStart.y
}

function removeDragListeners() {
  window.removeEventListener('pointermove', onPointerMove)
  window.removeEventListener('pointerup', onPointerUp)
  window.removeEventListener('pointercancel', onPointerCancel)
}

// Системная отмена жеста (скролл, звонок, потеря capture) — вернуть еду.
function onPointerCancel() {
  dragging.value = false
  removeDragListeners()
  offset.x = 0
  offset.y = 0
}

function onPointerUp(e) {
  dragging.value = false
  removeDragListeners()

  const mouthRect = mouthEl.value?.getBoundingClientRect()
  if (mouthRect) {
    const cx = mouthRect.left + mouthRect.width / 2
    const cy = mouthRect.top + mouthRect.height / 2
    const radius = Math.max(mouthRect.width, mouthRect.height) / 2 + 12
    if (isInHitZone(e.clientX, e.clientY, cx, cy, radius)) {
      state.value = 'success'
      if (petRaf) cancelAnimationFrame(petRaf) // поймали — питомец останавливается жевать
      setTimeout(() => emit('success'), 550)
      return
    }
  }
  // Промах — пружинка возвращает еду на место.
  state.value = 'miss'
  offset.x = 0
  offset.y = 0
  setTimeout(() => { if (state.value === 'miss') state.value = 'idle' }, 900)
}

nextTick(() => { origin.x = 0; origin.y = 0 })

onBeforeUnmount(() => {
  removeDragListeners()
  if (petRaf) cancelAnimationFrame(petRaf)
})
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
  width: min(360px, calc(100vw - 32px));
  background: var(--color-surface);
  border-radius: 24px;
  box-shadow: var(--shadow-lg);
  padding: 18px 18px 22px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.mg-head { display: flex; align-items: center; justify-content: space-between; }
.mg-head h3 { margin: 0; font-size: 16px; font-weight: 700; }
.mg-close {
  width: 32px; height: 32px; border-radius: 50%; border: none; background: none;
  color: var(--color-text-dim); display: grid; place-items: center; cursor: pointer;
}
.mg-close:hover { background: var(--color-surface-high); }
.mg-hint { margin: 0 0 10px; font-size: 12.5px; color: var(--color-text-dim); }

.mg-area {
  position: relative;
  height: 220px;
  border-radius: 18px;
  background: var(--color-surface-high);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-end;
  padding: 20px 0 28px;
  touch-action: none;
  overflow: hidden;
}
.mg-pet {
  position: absolute;
  top: 18px;
  transform: translateX(-50%);
  width: 88px;
  height: 88px;
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  font-size: 46px;
}
.mg-pet.chew { animation: mg-chew 0.55s ease; }
/* База transform — translateX(-50%) (центрирование): keyframes обязаны её
   сохранять, иначе питомец прыгнет в момент жевания. */
@keyframes mg-chew {
  0%, 100% { transform: translateX(-50%) scale(1); }
  40% { transform: translateX(-50%) scale(1.15); }
  70% { transform: translateX(-50%) scale(0.92); }
}
.mg-food {
  font-size: 40px;
  line-height: 1;
  cursor: grab;
  touch-action: none;
  user-select: none;
  transition: transform 0.25s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.mg-food.dragging { cursor: grabbing; transition: none; }
.mg-food.success { animation: mg-eaten 0.55s ease forwards; }
@keyframes mg-eaten {
  from { transform: scale(1); opacity: 1; }
  to { transform: scale(0.2); opacity: 0; }
}
.mg-feedback { margin: 8px 0 0; text-align: center; font-size: 12.5px; color: var(--color-error); }
</style>
