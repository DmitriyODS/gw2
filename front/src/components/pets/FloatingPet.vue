<template>
  <Teleport to="body">
    <div
      v-if="pet"
      v-show="!anyModalOpen"
      class="fp-root float-fade"
      :class="{ dragging, 'float-hidden': floatingHidden }"
      :style="rootStyle"
    >
      <transition name="fp-bubble">
        <button v-if="bubble" class="fp-bubble" type="button" @click="onBubbleTap">
          {{ bubble.text }}
        </button>
      </transition>

      <button
        class="fp-avatar float-spring"
        type="button"
        aria-label="Открыть питомца"
        @pointerdown="onPointerDown"
        @click="onClick"
      >
        <span class="fp-emoji" :class="{ sick: pet.sick }">{{ petEmoji(pet) }}</span>
        <span v-if="pet.sick" class="fp-sick-badge" title="Питомец болеет">🤒</span>
        <span v-else-if="petAway" class="fp-adventure-badge" title="В приключении">🧭</span>
      </button>
    </div>

    <PetDetailModal
      v-if="modalOpen"
      :initial-action="initialAction"
      @close="modalOpen = false"
    />
  </Teleport>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useDraggable } from '@/composables/useDraggable.js'
import { anyModalOpen } from '@/composables/useOpenModals.js'
import { floatingHidden, installFloatingHide } from '@/composables/useFloatingHide.js'
import { usePetsStore } from '@/stores/pets.js'
import { petEmoji } from '@/utils/pets.js'
import { storageGet, storageSet } from '@/utils/storage.js'
import PetDetailModal from '@/components/pets/PetDetailModal.vue'

const WIDGET_SIZE = { w: 60, h: 60 }
const BUBBLE_HIDE_MS = 4500
const HUNGER_SHOWN_KEY = 'gw_pet_hunger_shown_date'
const MOBILE_BP = 768
const isNarrow = () => typeof window !== 'undefined' && window.innerWidth <= MOBILE_BP

const pets = usePetsStore()
const pet = computed(() => pets.pet)
const petAway = computed(() => {
  const until = pet.value?.adventure_until
  return !!until && new Date(until) > new Date()
})

const { pos, dragging, onPointerDown, wasDragged } = useDraggable({
  storageKey: 'gw_pet_widget_pos',
  size: WIDGET_SIZE,
  defaultCorner: 'bottom-left',
  margin: 16,
  // Запас под AppBottomNav (64px) + safe-area на мобильном — функции
  // пересчитываются на каждый clamp, так что resize/поворот учитываются.
  bottomInset: () => (isNarrow() ? 104 : 0),
  // Правый нижний угол занят FAB мини-хаба (56px + отступы) — туда не снапаем.
  rightBottomReserve: () => (isNarrow() ? 76 : 90),
})

// transform вместо left/top: композит-слой, без layout на каждый кадр драга.
const rootStyle = computed(() => ({
  transform: `translate3d(${pos.value.x}px, ${pos.value.y}px, 0)`,
}))

const modalOpen = ref(false)
const initialAction = ref(null)
const bubble = ref(null)
let bubbleTimer = null

function showBubble(text, action = null) {
  clearTimeout(bubbleTimer)
  bubble.value = { text, action }
  bubbleTimer = setTimeout(() => { bubble.value = null }, BUBBLE_HIDE_MS)
}

function onClick() {
  if (wasDragged()) return // клик сразу после перетаскивания — игнорируем
  initialAction.value = null
  modalOpen.value = true
}

function onBubbleTap() {
  initialAction.value = bubble.value?.action || null
  bubble.value = null
  clearTimeout(bubbleTimer)
  modalOpen.value = true
}

function todayKey() {
  const d = new Date()
  return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`
}

// Дата «сегодня» — ЛОКАЛЬНАЯ в обеих проверках (toISOString дал бы UTC и
// ложный «голодный» пузырь ночью в поясах восточнее Гринвича).
function localDateISO() {
  const d = new Date()
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`
}

function checkHungry(p) {
  // В приключении кормление недоступно — «голодный» пузырь не показываем.
  if (!p || p.sick || (p.adventure_until && new Date(p.adventure_until) > new Date())) return
  const fedToday = p.last_fed_date && String(p.last_fed_date).slice(0, 10) === localDateISO()
  if (fedToday) return
  if (storageGet(HUNGER_SHOWN_KEY, null) === todayKey()) return
  storageSet(HUNGER_SHOWN_KEY, todayKey())
  showBubble('Проголодался — покормишь?', 'feed')
}

function nearNextStage(p) {
  if (!p?.next_stage_xp) return false
  return p.xp / p.next_stage_xp >= 0.9
}

// Реагируем только на ПЕРЕХОДЫ состояния (сравнение с предыдущим снимком),
// не на таймер-опрос — приходит через сокет pet:update или после своих же
// действий (стор обновляется оптимистично из ответа API).
// deep не нужен: стор всегда заменяет pet новым объектом.
watch(pet, (next, prev) => {
  if (!next) return
  if (prev) {
    if (!prev.sick && next.sick) {
      showBubble('Кажется, я заболел — помоги мне поправиться', 'heal')
    } else if (prev.sick && !next.sick) {
      showBubble('Ура, я снова здоров!')
    } else if (prev.adventure_until && !next.adventure_until) {
      showBubble('Я вернулся с прогулки!')
    } else if (next.stage > prev.stage) {
      showBubble('Я эволюционировал! Загляни посмотреть')
    } else if (nearNextStage(next) && !nearNextStage(prev)) {
      showBubble('Ещё немного — и я эволюционирую!')
    }
  }
  checkHungry(next)
})

onMounted(() => {
  if (!pets.pet) pets.fetchPet().catch(() => {})
  installFloatingHide()
})

onBeforeUnmount(() => clearTimeout(bubbleTimer))
</script>

<style scoped>
.fp-root {
  position: fixed;
  left: 0;
  top: 0; /* позиция — через transform (см. rootStyle) */
  z-index: 10055; /* выше ActiveUnitModal (9999), рядом с MiniMessenger (10050) */
  display: flex;
  flex-direction: column;
  align-items: center;
  touch-action: none;
}
.fp-root.dragging .fp-avatar { cursor: grabbing; transform: scale(1.05); }

.fp-avatar {
  width: 60px;
  height: 60px;
  border-radius: 50%;
  border: none;
  background: var(--color-surface);
  box-shadow: var(--shadow-lg, 0 8px 24px color-mix(in oklch, var(--color-primary) 30%, transparent));
  display: grid;
  place-items: center;
  cursor: grab;
  touch-action: none;
  position: relative;
  transition: transform 0.15s;
}
.fp-avatar:active { transform: scale(0.95); }
.fp-emoji { font-size: 32px; line-height: 1; }
.fp-emoji.sick { filter: grayscale(0.55) brightness(0.92); }
.fp-sick-badge { position: absolute; bottom: -2px; right: -2px; font-size: 16px; }
.fp-adventure-badge { position: absolute; bottom: -2px; right: -2px; font-size: 16px; }

.fp-bubble {
  margin-bottom: 8px;
  max-width: 200px;
  background: var(--color-surface-high);
  border: 1px solid var(--color-outline-dim);
  border-radius: 14px;
  padding: 8px 12px;
  font-size: 12.5px;
  text-align: center;
  line-height: 1.35;
  box-shadow: var(--shadow-sm, none);
  cursor: pointer;
  color: var(--color-text);
}
.fp-bubble-enter-active, .fp-bubble-leave-active { transition: opacity 0.25s, transform 0.25s; }
.fp-bubble-enter-from, .fp-bubble-leave-to { opacity: 0; transform: translateY(6px); }

/* Мобильные safe-area отступы и приоритет над нижней навигацией. */
@media (max-width: 768px) {
  .fp-root {
    padding-bottom: env(safe-area-inset-bottom, 0px);
    padding-left: env(safe-area-inset-left, 0px);
  }
}
</style>
