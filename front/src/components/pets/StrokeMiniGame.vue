<template>
  <Teleport to="body">
    <div class="mg-overlay" @click.self="onClose">
      <div class="mg-panel">
        <header class="mg-head">
          <h3>Погладить питомца</h3>
          <button class="mg-close" type="button" @click="onClose" aria-label="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </header>
        <p class="mg-hint">
          {{ ownerName ? `«${petName}» у ${ownerName}` : `«${petName}»` }} — водите ладошкой по питомцу,
          каждое поглаживание <KudosCoin class="mg-hint-coin" /> 2
        </p>

        <div
          ref="zoneEl"
          class="mg-stroke-zone"
          :class="{ done: finished, house: hasHouse }"
          :style="sceneStyle"
          @pointerdown="onDown"
          @pointermove="onMove"
          @pointerup="onUp"
          @pointercancel="onUp"
        >
          <!-- Обставленная комната грувика: гладим прямо в его домике. -->
          <template v-if="hasHouse">
            <span
              v-for="item in houseItems"
              :key="'house-' + item.key"
              class="mg-prop mg-prop--house"
              :style="{ left: item.x + '%', top: item.y + '%' }"
              :title="decorTitle(item.key)"
              aria-hidden="true"
            ><EmojiGlyph :char="decorEmoji(item.key)" /></span>
          </template>
          <span
            v-for="(pr, i) in hasHouse ? [] : scene.props"
            :key="'prop-' + i"
            class="mg-prop"
            :style="{ left: pr.left + '%', top: pr.top + '%', fontSize: pr.size + 'px' }"
            aria-hidden="true"
          >{{ pr.emoji }}</span>

          <template v-if="!finished">
            <span ref="petEl" class="mg-stroke-emoji" :class="{ happy: rubbing && overPet, pulse: bigPulse }" :style="petPosStyle"><EmojiGlyph :char="emoji" /></span>
            <span class="mg-hand" :class="{ rubbing: rubbing && overPet }" :style="handStyle" aria-hidden="true">🫳</span>
            <transition-group name="mg-heart" tag="div" class="mg-hearts" aria-hidden="true">
              <span v-for="h in hearts" :key="h.id" class="mg-heart" :style="{ left: h.left + '%' }">💗</span>
            </transition-group>
          </template>

          <!-- Финал: лимит на сегодня выбран -->
          <div v-else class="mg-final">
            <span class="mg-final-emoji"><EmojiGlyph :char="emoji" /></span>
            <p class="mg-final-text">«{{ petName }}» наглажен до завтра — мурчит и сияет 💖</p>
          </div>
        </div>

        <!-- Прогресс текущего поглаживания + слоты выполненных -->
        <template v-if="!finished">
          <div class="mg-progress">
            <div class="mg-progress-fill" :style="{ width: rubProgress + '%' }"></div>
          </div>
          <div class="mg-slots" :title="`Поглаживаний за сеанс: ${strokesDone}`">
            <span v-for="i in MAX_STROKES" :key="i" class="mg-slot" :class="{ filled: i <= strokesDone }">💗</span>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, onBeforeUnmount, ref } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import { createRubTracker, isInHitZone } from '@/utils/miniGames.js'
import { decorEmoji, decorTitle, petEmoji } from '@/utils/pets.js'
import { houseThemeBackground } from '@/utils/houseThemes.js'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  pet: { type: Object, default: null },
  // Сколько раз этот питомец уже поглажен зрителем сегодня (strokes_today
  // из выдачи зоопарка) — слоты и лимит переживают перезагрузку страницы.
  initialStrokes: { type: Number, default: 0 },
})
const emit = defineEmits(['close', 'exhausted'])

const MAX_STROKES = 3 // = domain.StrokeDailyMaxPerPet
const RUB_THRESHOLD_PX = 420
const FINAL_CLOSE_MS = 2200

const pets = usePetsStore()
const notify = useNotificationsStore()

const emoji = computed(() => petEmoji(props.pet))
const petName = computed(() => props.pet?.name || 'Грувик')
const ownerName = computed(() => {
  const fio = props.pet?.user?.fio
  if (!fio) return ''
  const parts = fio.split(' ')
  return parts.length > 1 ? `${parts[0]} ${parts[1]}` : fio
})

// Уютные фоны-сценки (пастель только из токенов тегов, декор — эмодзи),
// каждый сеанс — случайная.
const SCENES = [
  {
    from: 'pink', to: 'violet',
    props: [
      { emoji: '🌸', left: 10, top: 18, size: 20 },
      { emoji: '✨', left: 84, top: 14, size: 16 },
      { emoji: '🌷', left: 88, top: 72, size: 20 },
      { emoji: '🦋', left: 16, top: 68, size: 16 },
    ],
  },
  {
    from: 'amber', to: 'orange',
    props: [
      { emoji: '🌅', left: 12, top: 16, size: 22 },
      { emoji: '☁️', left: 82, top: 20, size: 18 },
      { emoji: '🌾', left: 90, top: 70, size: 20 },
      { emoji: '✨', left: 14, top: 74, size: 14 },
    ],
  },
  {
    from: 'blue', to: 'teal',
    props: [
      { emoji: '🌙', left: 12, top: 16, size: 20 },
      { emoji: '⭐', left: 86, top: 14, size: 15 },
      { emoji: '☁️', left: 88, top: 70, size: 18 },
      { emoji: '💤', left: 14, top: 70, size: 16 },
    ],
  },
]
const scene = SCENES[Math.floor(Math.random() * SCENES.length)]

// Если хозяин обставил комнату грувика — гладим в ней: сцена мини-игры
// повторяет расстановку house_placed (координаты в % сцены, как в домике).
const houseItems = computed(() =>
  (props.pet?.house_placed || []).map((i, idx) =>
    typeof i === 'string' ? { key: i, x: 16 + (idx % 5) * 17, y: 78 } : i))
const hasHouse = computed(() => houseItems.value.length > 0)

const sceneStyle = computed(() => (hasHouse.value
  ? { background: houseThemeBackground(props.pet?.house_theme) }
  : { background: `linear-gradient(135deg, var(--tag-${scene.from}-surface), var(--tag-${scene.to}-surface))` }))

// В комнате грувик стоит там, куда его поставил хозяин.
const petPosStyle = computed(() => {
  if (!hasHouse.value || props.pet?.house_pet_x == null || props.pet?.house_pet_y == null) return {}
  return {
    position: 'absolute',
    left: props.pet.house_pet_x + '%',
    top: props.pet.house_pet_y + '%',
    transform: 'translate(-50%, -50%)',
  }
})

const zoneEl = ref(null)
const petEl = ref(null)
const rubbing = ref(false)
const overPet = ref(false)
const rubProgress = ref(0)
const strokesDone = ref(Math.min(MAX_STROKES, Math.max(0, props.initialStrokes)))
const finished = ref(false)
const bigPulse = ref(false)
const hearts = ref([])
const hand = ref({ x: 50, y: 55 })

const tracker = createRubTracker(RUB_THRESHOLD_PX)
let last = null
let heartSeq = 0
let heartDist = 0
let sending = false
let closeTimer = null

const handStyle = computed(() => ({ left: `${hand.value.x}%`, top: `${hand.value.y}%` }))

function relative(e) {
  const rect = zoneEl.value?.getBoundingClientRect()
  if (!rect || !rect.width) return null
  return {
    x: Math.min(96, Math.max(4, ((e.clientX - rect.left) / rect.width) * 100)),
    y: Math.min(90, Math.max(10, ((e.clientY - rect.top) / rect.height) * 100)),
  }
}

function spawnHeart(leftPercent) {
  const id = heartSeq++
  hearts.value.push({ id, left: leftPercent })
  if (hearts.value.length > 7) hearts.value.shift()
  setTimeout(() => { hearts.value = hearts.value.filter((h) => h.id !== id) }, 700)
}

function onDown(e) {
  if (finished.value) return
  rubbing.value = true
  overPet.value = isOverPet(e)
  last = overPet.value ? { x: e.clientX, y: e.clientY } : null
  const p = relative(e)
  if (p) hand.value = p
  try { e.currentTarget?.setPointerCapture?.(e.pointerId) } catch { /* noop */ }
}

// Трение засчитывается ТОЛЬКО над самим питомцем (круг вокруг эмодзи с
// небольшим запасом), а не по всей серой зоне.
function isOverPet(e) {
  const rect = petEl.value?.getBoundingClientRect()
  if (!rect || !rect.width) return false
  const cx = rect.left + rect.width / 2
  const cy = rect.top + rect.height / 2
  return isInHitZone(e.clientX, e.clientY, cx, cy, Math.max(rect.width, rect.height) / 2 + 14)
}

function onMove(e) {
  if (!rubbing.value || finished.value || sending) return
  const p = relative(e)
  if (p) hand.value = p

  overPet.value = isOverPet(e)
  if (!overPet.value) {
    // Вышли за питомца — «разрываем» жест, чтобы обратный вход не засчитал
    // прыжок курсора как дистанцию трения.
    last = null
    return
  }
  if (!last) { last = { x: e.clientX, y: e.clientY }; return }
  const dx = e.clientX - last.x
  const dy = e.clientY - last.y
  last = { x: e.clientX, y: e.clientY }

  // Мелкие сердечки — непрерывная обратная связь во время трения.
  heartDist += Math.abs(dx) + Math.abs(dy)
  if (heartDist > 110) {
    heartDist = 0
    spawnHeart(hand.value.x)
  }

  const r = tracker.add(dx, dy)
  rubProgress.value = r.completed ? 100 : r.progress
  if (r.completed) performStroke()
}

function onUp() {
  rubbing.value = false
  overPet.value = false
  last = null
}

// Полный цикл трения — одно платное поглаживание (StrokePet, domain.StrokeCost = 2 кудоса).
async function performStroke() {
  if (sending || finished.value) return
  sending = true
  try {
    await pets.strokePet(props.pet.user_id)
    strokesDone.value++
    bigPulse.value = true
    setTimeout(() => { bigPulse.value = false }, 500)
    if (strokesDone.value >= MAX_STROKES) {
      finishExhausted()
    }
  } catch (e) {
    if (e?.error === 'STROKED_ENOUGH') {
      finishExhausted()
    } else if (e?.error === 'NO_KUDOS') {
      notify.warn('Кудосы закончились — поглаживания на сегодня всё')
      emit('close')
    } else if (e?.error === 'PET_AWAY') {
      notify.warn(e?.message || 'Питомец в приключении')
      emit('close')
    } else {
      notify.warn(e?.message || 'Не получилось погладить')
      emit('close')
    }
  } finally {
    sending = false
    rubProgress.value = tracker.progress
  }
}

// Дневной лимит на этого питомца выбран — тёплый финал и автозакрытие.
function finishExhausted() {
  finished.value = true
  rubbing.value = false
  emit('exhausted')
  closeTimer = setTimeout(() => emit('close'), FINAL_CLOSE_MS)
}

function onClose() {
  clearTimeout(closeTimer)
  emit('close')
}

onBeforeUnmount(() => clearTimeout(closeTimer))
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

.mg-stroke-zone {
  position: relative;
  height: 170px;
  border-radius: 18px;
  display: grid;
  place-items: center;
  touch-action: none;
  cursor: grab;
  overflow: hidden;
  user-select: none;
}

.mg-prop {
  position: absolute;
  transform: translate(-50%, -50%);
  line-height: 1;
  pointer-events: none;
  opacity: 0.85;
}
/* Декор комнаты — как в PetHouseDialog: тот же размер и координаты. */
.mg-prop--house { font-size: 26px; opacity: 1; }
.mg-stroke-zone:active { cursor: grabbing; }

.mg-stroke-emoji { font-size: 60px; line-height: 1; transition: transform 0.2s; }
.mg-stroke-emoji.happy { transform: scale(1.06); }
.mg-stroke-emoji.pulse { animation: mg-stroke-pulse 0.5s cubic-bezier(0.34, 1.56, 0.64, 1); }
@keyframes mg-stroke-pulse {
  0% { transform: scale(1); }
  40% { transform: scale(1.22) rotate(-4deg); }
  100% { transform: scale(1); }
}
@media (prefers-reduced-motion: reduce) { .mg-stroke-emoji.pulse { animation: none; } }

.mg-hand {
  position: absolute;
  font-size: 30px;
  line-height: 1;
  transform: translate(-50%, -50%) rotate(-15deg);
  opacity: 0.45;
  pointer-events: none;
  transition: opacity 0.15s;
}
.mg-hand.rubbing { opacity: 1; }

.mg-hearts { position: absolute; inset: 0; pointer-events: none; }
.mg-heart {
  position: absolute;
  bottom: 12%;
  font-size: 18px;
  transform: translateX(-50%);
}
.mg-heart-enter-active { transition: opacity 0.15s, transform 0.6s ease-out; }
.mg-heart-enter-from { opacity: 0; }
.mg-heart-leave-active { transition: opacity 0.4s, transform 0.4s; }
.mg-heart-leave-to { opacity: 0; transform: translate(-50%, -34px); }

.mg-final {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 0 18px;
  text-align: center;
  animation: mg-final-in 0.35s cubic-bezier(0.34, 1.56, 0.64, 1);
}
@keyframes mg-final-in {
  from { opacity: 0; transform: scale(0.85); }
  to { opacity: 1; transform: scale(1); }
}
.mg-final-emoji { font-size: 54px; line-height: 1; }
.mg-final-text { margin: 0; font-size: 14px; font-weight: 600; line-height: 1.4; }

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

.mg-slots { display: flex; justify-content: center; gap: 8px; }
.mg-slot { font-size: 16px; opacity: 0.25; filter: grayscale(1); transition: opacity 0.2s, filter 0.2s; }
.mg-slot.filled { opacity: 1; filter: none; }
</style>
