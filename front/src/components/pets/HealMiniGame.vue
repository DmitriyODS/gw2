<template>
  <Teleport to="body">
    <div class="mg-overlay" @click.self="$emit('close')">
      <div class="mg-panel">
        <header class="mg-head">
          <h3>Дать лекарство</h3>
          <button class="mg-close" type="button" @click="$emit('close')" aria-label="Закрыть">
            <span class="material-symbols-outlined">close</span>
          </button>
        </header>
        <p class="mg-hint">Тапните по полосе, когда маркер в зелёной зоне — нужно {{ HITS_NEEDED }} попаданий</p>

        <button class="mg-bar" type="button" @click="onTap">
          <span class="mg-zone" :style="{ left: zoneStart + '%', width: zoneWidthNow + '%' }"></span>
          <span class="mg-marker" :style="{ left: markerPercent + '%' }"></span>
        </button>

        <div class="mg-dots">
          <span
            v-for="i in HITS_NEEDED"
            :key="i"
            class="mg-dot"
            :class="{ filled: i <= hits }"
          ></span>
        </div>
        <p class="mg-feedback" :class="feedback">{{ feedbackText }}</p>
      </div>
    </div>
  </Teleport>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { isInGreenZone, pingPongPercent } from '@/utils/miniGames.js'

const emit = defineEmits(['success', 'close'])

const HITS_NEEDED = 4
const PERIOD_MS = 1100
// Зона сужается с каждым попаданием (18 → 12%) — прогрессия сложности.
const BASE_ZONE_WIDTH = 18
const MIN_ZONE_WIDTH = 12

const hits = ref(0)
const zoneWidthNow = computed(() => Math.max(MIN_ZONE_WIDTH, BASE_ZONE_WIDTH - hits.value * 2))
const feedback = ref('')
const feedbackText = computed(() => {
  if (feedback.value === 'hit') return 'Есть! Продолжайте'
  if (feedback.value === 'miss') return 'Мимо — ещё разок'
  return 'Ловите момент'
})

const zoneStart = ref(randomZoneStart())
const markerPercent = ref(0)
let raf = null
let startTs = 0
let finished = false
// Недавние позиции маркера: тап честно засчитываем, если маркер был в зоне
// в последние GRACE_MS — компенсирует реакцию и задержку рендера (иначе
// «кликнул в зелёное, а не сработало»).
const GRACE_MS = 90
let trail = []

function randomZoneStart() {
  return 10 + Math.random() * (90 - zoneWidthNow.value - 10)
}

function tick(ts) {
  if (!startTs) startTs = ts
  markerPercent.value = pingPongPercent(ts - startTs, PERIOD_MS)
  trail.push({ t: ts, pos: markerPercent.value })
  while (trail.length && ts - trail[0].t > GRACE_MS) trail.shift()
  raf = requestAnimationFrame(tick)
}

function hitWithinGrace() {
  if (isInGreenZone(markerPercent.value, zoneStart.value, zoneWidthNow.value)) return true
  return trail.some((p) => isInGreenZone(p.pos, zoneStart.value, zoneWidthNow.value))
}

function onTap() {
  if (finished) return
  if (hitWithinGrace()) {
    hits.value++
    feedback.value = 'hit'
    zoneStart.value = randomZoneStart()
    trail = [] // хвост старой зоны не должен «попадать» в новую
    if (hits.value >= HITS_NEEDED) {
      finished = true
      if (raf) cancelAnimationFrame(raf)
      setTimeout(() => emit('success'), 400)
    }
  } else {
    hits.value = Math.max(0, hits.value - 1)
    feedback.value = 'miss'
  }
}

onMounted(() => { raf = requestAnimationFrame(tick) })
onBeforeUnmount(() => { if (raf) cancelAnimationFrame(raf) })
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
.mg-hint { margin: 0; font-size: 12.5px; color: var(--color-text-dim); }

.mg-bar {
  position: relative;
  width: 100%;
  height: 52px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  cursor: pointer;
  padding: 0;
  touch-action: manipulation;
}
.mg-zone {
  position: absolute;
  top: 0; bottom: 0;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-success) 45%, transparent);
}
.mg-marker {
  position: absolute;
  top: -4px;
  width: 6px;
  height: 60px;
  border-radius: 3px;
  background: var(--color-primary);
  transform: translateX(-50%);
}

.mg-dots { display: flex; justify-content: center; gap: 8px; }
.mg-dot {
  width: 14px; height: 14px; border-radius: 50%;
  background: var(--color-surface-high);
  border: 1.5px solid var(--color-outline-dim);
}
.mg-dot.filled { background: var(--color-success); border-color: var(--color-success); }

.mg-feedback { margin: 0; text-align: center; font-size: 12.5px; color: var(--color-text-dim); }
.mg-feedback.hit { color: var(--color-success); font-weight: 700; }
.mg-feedback.miss { color: var(--color-error); }
</style>
