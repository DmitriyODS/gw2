<template>
  <Teleport to="body">
    <div class="mg-overlay" @click.self="onClose">
      <div class="mg-panel">
        <header class="mg-head">
          <h3>Прогулка · {{ scene.title }}</h3>
          <button class="mg-close" type="button" @click="onClose" aria-label="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </header>
        <p class="mg-hint">Потаскайте питомца по локации и соберите {{ scene.itemName }}</p>

        <div
          ref="sceneEl"
          class="mg-scene"
          :style="sceneStyle"
          @pointerdown="onDown"
          @pointermove="onMove"
          @pointerup="onUp"
          @pointercancel="onUp"
        >
          <span
            v-for="(p, i) in scene.props"
            :key="'prop-' + i"
            class="mg-prop"
            :style="{ left: p.left + '%', bottom: p.bottom + '%', fontSize: p.size + 'px' }"
            aria-hidden="true"
          >{{ p.emoji }}</span>

          <span
            v-for="it in items"
            :key="it.id"
            class="mg-item"
            :class="{ collected: it.collected }"
            :style="{ left: it.left + '%' }"
          >{{ it.collected ? '✨' : scene.item }}</span>

          <div class="mg-walker" :class="{ dragging }" :style="{ left: walkerLeft + '%' }">
            <EmojiGlyph :char="petEmojiChar" />
          </div>
          <div class="mg-ground"></div>
        </div>

        <p class="mg-score">Собрано: {{ collectedCount }}/{{ items.length }}</p>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, ref } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { petEmoji } from '@/utils/pets.js'
import { storageGet, storageSet } from '@/utils/storage.js'

const props = defineProps({
  pet: { type: Object, default: null },
})
const emit = defineEmits(['success', 'close'])

const petEmojiChar = computed(() => petEmoji(props.pet))

// Локации: пастельные фоны только из токенов тегов (--tag-*-surface),
// декор и предметы — эмодзи. Каждая прогулка — новая случайная локация.
const SCENES = [
  {
    key: 'forest', title: 'Лес', item: '🍄', itemName: 'грибы',
    sky: 'teal', ground: 'green',
    props: [
      { emoji: '🌲', left: 8, bottom: 22, size: 34 },
      { emoji: '🌳', left: 30, bottom: 24, size: 30 },
      { emoji: '🦋', left: 55, bottom: 62, size: 16 },
      { emoji: '🌲', left: 78, bottom: 23, size: 38 },
      { emoji: '🐿️', left: 92, bottom: 24, size: 18 },
    ],
  },
  {
    key: 'river', title: 'Речка', item: '🐚', itemName: 'ракушки',
    sky: 'blue', ground: 'teal',
    props: [
      { emoji: '🌾', left: 6, bottom: 22, size: 26 },
      { emoji: '🦆', left: 34, bottom: 55, size: 20 },
      { emoji: '🌊', left: 58, bottom: 24, size: 28 },
      { emoji: '🐟', left: 76, bottom: 50, size: 18 },
      { emoji: '🌾', left: 93, bottom: 22, size: 26 },
    ],
  },
  {
    key: 'mountains', title: 'Горы', item: '🌸', itemName: 'цветы',
    sky: 'violet', ground: 'amber',
    props: [
      { emoji: '⛰️', left: 12, bottom: 24, size: 40 },
      { emoji: '☁️', left: 38, bottom: 66, size: 24 },
      { emoji: '🦅', left: 62, bottom: 60, size: 18 },
      { emoji: '🏔️', left: 84, bottom: 25, size: 36 },
    ],
  },
  {
    key: 'beach', title: 'Пляж', item: '⭐', itemName: 'морские звёзды',
    sky: 'amber', ground: 'orange',
    props: [
      { emoji: '🌴', left: 10, bottom: 22, size: 36 },
      { emoji: '⛱️', left: 40, bottom: 24, size: 30 },
      { emoji: '🦀', left: 66, bottom: 22, size: 18 },
      { emoji: '🌴', left: 90, bottom: 23, size: 32 },
    ],
  },
  {
    key: 'winter', title: 'Зимний лес', item: '❄️', itemName: 'снежинки',
    sky: 'blue', ground: 'pink',
    props: [
      { emoji: '🌲', left: 9, bottom: 23, size: 34 },
      { emoji: '⛄', left: 36, bottom: 23, size: 28 },
      { emoji: '☁️', left: 60, bottom: 68, size: 22 },
      { emoji: '🌲', left: 82, bottom: 24, size: 38 },
    ],
  },
]

const LAST_SCENE_KEY = 'gw_pet_walk_scene'

function pickScene() {
  const last = storageGet(LAST_SCENE_KEY, null)
  const pool = SCENES.filter((s) => s.key !== last)
  const scene = pool[Math.floor(Math.random() * pool.length)] || SCENES[0]
  storageSet(LAST_SCENE_KEY, scene.key)
  return scene
}

const scene = pickScene()

const sceneStyle = computed(() => ({
  background: `linear-gradient(to bottom,
    var(--tag-${scene.sky}-surface) 0%,
    var(--tag-${scene.sky}-surface) 55%,
    var(--tag-${scene.ground}-surface) 55%,
    var(--tag-${scene.ground}-surface) 100%)`,
}))

// 4 предмета на случайных позициях (не ближе 12% друг к другу).
function spawnItems() {
  const out = []
  let guard = 0
  while (out.length < 4 && guard++ < 100) {
    const left = 10 + Math.random() * 80
    if (out.every((o) => Math.abs(o.left - left) >= 12)) {
      out.push({ id: out.length + 1, left, collected: false })
    }
  }
  return out
}

const items = ref(spawnItems())
const collectedCount = computed(() => items.value.filter((i) => i.collected).length)

const walkerLeft = ref(8)
const dragging = ref(false)
const sceneEl = ref(null)
let finished = false

// «Игрок начал играть» = собрал хотя бы один предмет: до этого закрытие —
// бесплатная отмена (API ещё не вызывался); после — закрыть/пропустить =
// досрочно завершить, прогулка засчитывается (иначе можно было бы бесплатно
// отменять неудачную попытку).
const hasActed = computed(() => collectedCount.value > 0)

const HIT_RADIUS = 7 // % ширины сцены

function relativeX(e) {
  const rect = sceneEl.value?.getBoundingClientRect()
  if (!rect || !rect.width) return walkerLeft.value
  return Math.min(94, Math.max(4, ((e.clientX - rect.left) / rect.width) * 100))
}

function tryCollect() {
  for (const it of items.value) {
    if (!it.collected && Math.abs(it.left - walkerLeft.value) <= HIT_RADIUS) {
      it.collected = true
    }
  }
  if (items.value.every((i) => i.collected)) finish()
}

function onDown(e) {
  if (finished) return
  dragging.value = true
  try { e.currentTarget?.setPointerCapture?.(e.pointerId) } catch { /* noop */ }
  walkerLeft.value = relativeX(e)
  tryCollect()
}

function onMove(e) {
  if (!dragging.value || finished) return
  walkerLeft.value = relativeX(e)
  tryCollect()
}

function onUp() {
  dragging.value = false
}

function finish() {
  if (finished) return
  finished = true
  dragging.value = false
  setTimeout(() => emit('success'), 450)
}

function onClose() {
  if (finished || !hasActed.value) {
    emit('close')
    return
  }
  finish()
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
  width: min(420px, calc(100vw - 32px));
  background: var(--color-surface);
  border-radius: 24px;
  box-shadow: var(--shadow-lg);
  padding: 18px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.mg-head { display: flex; align-items: center; justify-content: space-between; }
.mg-head h3 { margin: 0; font-size: 16px; font-weight: 700; }
.mg-close {
  width: 32px; height: 32px; min-height: 0; border-radius: 50%; border: none; background: none;
  color: var(--color-text-dim); display: grid; place-items: center; cursor: pointer;
}
.mg-close:hover { background: var(--color-surface-high); }
.mg-hint { margin: 0 0 10px; font-size: 12.5px; color: var(--color-text-dim); }

.mg-scene {
  position: relative;
  height: 170px;
  border-radius: 18px;
  overflow: hidden;
  touch-action: none;
  cursor: grab;
  user-select: none;
}
.mg-scene:active { cursor: grabbing; }

.mg-prop { position: absolute; transform: translateX(-50%); line-height: 1; pointer-events: none; }

.mg-item {
  position: absolute;
  bottom: 14%;
  transform: translateX(-50%);
  font-size: 22px;
  line-height: 1;
  pointer-events: none;
  transition: transform 0.2s, opacity 0.4s;
}
.mg-item.collected { opacity: 0; transform: translate(-50%, -26px) scale(1.4); }

.mg-walker {
  position: absolute;
  bottom: 9%;
  font-size: 36px;
  line-height: 1;
  transform: translateX(-50%);
  z-index: 1;
  pointer-events: none;
  transition: left 0.06s linear;
}
.mg-walker.dragging { animation: mg-walk-bob 0.35s ease-in-out infinite; }
@keyframes mg-walk-bob {
  0%, 100% { transform: translateX(-50%) translateY(0) rotate(-3deg); }
  50% { transform: translateX(-50%) translateY(-5px) rotate(3deg); }
}
@media (prefers-reduced-motion: reduce) { .mg-walker.dragging { animation: none; } }

/* Полоска «земли» поверх стыка градиента — даёт сцене глубину. */
.mg-ground {
  position: absolute;
  left: 0; right: 0; bottom: 0;
  height: 8%;
  background: color-mix(in oklch, var(--color-text) 8%, transparent);
  pointer-events: none;
}

.mg-score { margin: 8px 0 0; font-size: 12.5px; color: var(--color-text-dim); text-align: center; }
</style>
