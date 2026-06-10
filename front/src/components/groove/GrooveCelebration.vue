<template>
  <Teleport to="body">
    <transition name="celebr">
      <div v-if="visible" class="celebration" role="status" @click="dismiss">
        <i
          v-for="(style, i) in particles"
          :key="`${seed}-${i}`"
          class="celebr-confetti"
          :style="style"
          aria-hidden="true"
        ></i>
        <div class="celebr-card">
          <span class="celebr-emoji">{{ content.emoji }}</span>
          <h2 class="celebr-title">{{ content.title }}</h2>
          <p v-if="content.subtitle" class="celebr-subtitle">{{ content.subtitle }}</p>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { PET_STAGES, PET_SPECIES, SHOP_ITEMS } from '@/utils/groove.js'

const AUTO_HIDE_MS = 4200
const PALETTE = [
  '--color-primary', '--color-secondary', '--color-tertiary',
  '--color-success', '--color-warning',
]

const groove = useGrooveStore()

const visible = ref(false)
const particles = ref([])
const seed = ref(0)
let hideTimer = null

const content = computed(() => {
  const c = groove.celebration
  if (!c) return { emoji: '', title: '' }
  const p = c.payload || {}
  switch (c.kind) {
    case 'pet_evolved':
      return {
        emoji: PET_SPECIES[p.species]?.emoji || '🌟',
        title: `«${p.pet_name || 'Грувик'}» эволюционировал!`,
        subtitle: [PET_STAGES[p.stage], PET_SPECIES[p.species]?.title]
          .filter(Boolean).join(' · '),
      }
    case 'streak':
      return {
        emoji: '🔥',
        title: `Стрик ${p.days} дней!`,
        subtitle: `«${p.pet_name || 'Грувик'}» сыт и доволен — так держать`,
      }
    case 'pet_recovered':
      return {
        emoji: '💚',
        title: `«${p.pet_name || 'Грувик'}» снова здоров!`,
        subtitle: 'Отличная забота — болезнь позади',
      }
    case 'raid_won':
      return {
        emoji: '🏆',
        title: `«${p.boss || 'Босс'}» повержен!`,
        subtitle: `Всем Грувикам — ${SHOP_ITEMS[p.reward]?.title || 'награда'} и +${p.beans || 0} грувов`,
      }
    default:
      return { emoji: '🎉', title: 'Веха!' }
  }
})

function makeParticles() {
  return Array.from({ length: 18 }, (_, i) => {
    const round = Math.random() > 0.5
    return {
      left: `${4 + Math.random() * 92}%`,
      width: `${6 + Math.random() * 6}px`,
      height: `${6 + Math.random() * 6}px`,
      background: `var(${PALETTE[i % PALETTE.length]})`,
      borderRadius: round ? '50%' : '2px',
      animationDelay: `${Math.random() * 0.6}s`,
      animationDuration: `${1.7 + Math.random() * 1.2}s`,
      '--dx': `${(Math.random() - 0.5) * 30}vw`,
      '--rot': `${360 + Math.random() * 540}deg`,
    }
  })
}

watch(() => groove.celebration, (c) => {
  // Сторе может хранить веху, показанную на другом маршруте, — протухшее не играем.
  if (!c || Date.now() - c.at > 8000) return
  particles.value = makeParticles()
  seed.value = c.at
  visible.value = true
  clearTimeout(hideTimer)
  hideTimer = setTimeout(dismiss, AUTO_HIDE_MS)
}, { immediate: true })

function dismiss() {
  clearTimeout(hideTimer)
  visible.value = false
  groove.clearCelebration()
}

onBeforeUnmount(() => clearTimeout(hideTimer))
</script>

<style scoped>
.celebration {
  position: fixed;
  inset: 0;
  z-index: 10800;
  display: grid;
  place-items: center;
  background: color-mix(in oklch, var(--color-scrim, var(--color-text)) 32%, transparent);
  overflow: hidden;
  cursor: pointer;
}
.celebr-confetti {
  position: absolute;
  top: -16px;
  animation: celebr-fall 2s ease-in forwards;
}
@keyframes celebr-fall {
  from { transform: translate3d(0, -4vh, 0) rotate(0deg); opacity: 1; }
  to { transform: translate3d(var(--dx, 0), 106vh, 0) rotate(var(--rot, 540deg)); opacity: 0.85; }
}

.celebr-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  text-align: center;
  background: var(--color-surface);
  border-radius: 28px;
  box-shadow: var(--shadow-lg, 0 12px 40px color-mix(in oklch, var(--color-scrim, var(--color-text)) 30%, transparent));
  padding: 30px 36px 26px;
  max-width: min(420px, calc(100vw - 48px));
  animation: celebr-pop 0.55s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
}
.celebr-emoji {
  font-size: 64px;
  line-height: 1;
  animation: celebr-pop 0.6s cubic-bezier(0.34, 1.8, 0.64, 1) 0.12s backwards;
}
@keyframes celebr-pop {
  from { transform: scale(0.55); opacity: 0; }
}
.celebr-title {
  margin: 6px 0 0;
  font-size: 25px;
  font-weight: 800;
  line-height: 1.2;
  word-break: break-word;
}
.celebr-subtitle {
  margin: 0;
  font-size: 14.5px;
  color: var(--color-text-dim);
  line-height: 1.4;
}

.celebr-enter-active, .celebr-leave-active { transition: opacity 0.3s; }
.celebr-enter-from, .celebr-leave-to { opacity: 0; }

@media (prefers-reduced-motion: reduce) {
  .celebr-confetti { display: none; }
  .celebr-card, .celebr-emoji { animation: none; }
}
</style>
