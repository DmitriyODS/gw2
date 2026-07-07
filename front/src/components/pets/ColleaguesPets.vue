<template>
  <section class="cp-wrap">
    <header class="cp-head">
      <h3 class="cp-title">
        <span class="material-symbols-outlined">pets</span>
        Питомцы коллег
      </h3>
      <span class="cp-hint">Тап по питомцу — погладить (<KudosCoin class="cp-hint-coin" /> 1, до 3 раз в день)</span>
    </header>

    <div v-if="pets.zoo.length" class="cp-grid">
      <component
        :is="p.user_id === pets.myId ? 'div' : 'button'"
        v-for="p in pets.zoo"
        :key="p.user_id"
        class="cp-card"
        :class="{
          mine: p.user_id === pets.myId,
          stroked: strokedOut[p.user_id],
          disabled: p.user_id !== pets.myId && (isAway(p) || (!canAfford && !strokedOut[p.user_id])),
        }"
        :type="p.user_id === pets.myId ? undefined : 'button'"
        :disabled="p.user_id === pets.myId ? undefined : (isAway(p) || !canAfford || !!strokedOut[p.user_id])"
        :aria-label="p.user_id === pets.myId ? undefined : `Погладить питомца «${p.name}»`"
        @click="stroke(p)"
      >
        <div class="cp-figure" :class="{ sick: p.sick, pulse: pulsing[p.user_id] }">
          <span class="cp-emoji">{{ petEmoji(p) }}</span>
          <span v-if="p.hat" class="cp-hat">{{ shopItemEmoji({ kind: 'accessory', key: p.hat }) }}</span>
          <span v-if="p.sick" class="cp-sick" title="Болеет">🤒</span>
          <span v-else-if="isAway(p)" class="cp-sick" title="В приключении">🧭</span>
          <transition-group name="cp-heart" tag="div" class="cp-hearts" aria-hidden="true">
            <span v-for="h in hearts[p.user_id] || []" :key="h.id" class="cp-heart" :style="{ left: h.left + '%' }">💗</span>
          </transition-group>
        </div>
        <span class="cp-name">{{ p.name }}</span>
        <span class="cp-owner">{{ firstName(p.user?.fio) }}</span>
        <span class="cp-stage">{{ PET_STAGES[p.stage] || '' }}</span>

        <span v-if="p.user_id === pets.myId" class="cp-tag">Ваш питомец</span>
        <span v-else-if="isAway(p)" class="cp-tag">🧭 В приключении</span>
        <span v-else-if="strokedOut[p.user_id]" class="cp-tag done">
          <span class="material-symbols-outlined">check</span> Поглажен
        </span>
        <span v-else class="cp-tag action">
          <span class="material-symbols-outlined">volunteer_activism</span>
          Погладить · <KudosCoin class="cp-tag-coin" /> 1
        </span>
      </component>
    </div>

    <EmptyState
      v-else
      icon="pets"
      title="Пока пусто"
      subtitle="Питомцы коллег появятся здесь, как только кто-то войдёт в раздел"
    />
  </section>
</template>

<script setup>
import { computed, onMounted, reactive } from 'vue'
import EmptyState from '@/components/common/EmptyState.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { petEmoji, shopItemEmoji, PET_STAGES } from '@/utils/pets.js'

const STROKE_COST = 1 // = domain.StrokeCost в petsvc

const pets = usePetsStore()
const notify = useNotificationsStore()

const canAfford = computed(() => (pets.pet?.kudos ?? 0) >= STROKE_COST)

// Поглаживание — в один тап по карточке: сердечки сразу, запрос следом.
const hearts = reactive({})
const pulsing = reactive({})
const strokedOut = reactive({})
const busy = reactive({})
let heartSeq = 0

// Питомец коллеги в приключении — поглаживание недоступно, пока не вернётся.
function isAway(p) {
  return !!(p?.adventure_until && new Date(p.adventure_until) > new Date())
}

function firstName(fio) {
  if (!fio) return ''
  const parts = fio.split(' ')
  return parts.length > 1 ? `${parts[0]} ${parts[1]}` : fio
}

function spawnHearts(userId) {
  const list = hearts[userId] || (hearts[userId] = [])
  for (let i = 0; i < 3; i++) {
    const id = heartSeq++
    list.push({ id, left: 20 + Math.random() * 60 })
    setTimeout(() => {
      const cur = hearts[userId] || []
      hearts[userId] = cur.filter((h) => h.id !== id)
    }, 750)
  }
  pulsing[userId] = true
  setTimeout(() => { pulsing[userId] = false }, 450)
}

async function stroke(p) {
  if (p.user_id === pets.myId || busy[p.user_id] || strokedOut[p.user_id] || isAway(p)) return
  if (!canAfford.value) return
  busy[p.user_id] = true
  spawnHearts(p.user_id)
  try {
    await pets.strokePet(p.user_id)
  } catch (e) {
    if (e?.error === 'STROKED_ENOUGH') {
      strokedOut[p.user_id] = true
    } else {
      notify.warn(e?.message || 'Не получилось погладить')
    }
  } finally {
    busy[p.user_id] = false
  }
}

onMounted(() => {
  if (!pets.zoo.length) pets.fetchZoo().catch(() => {})
})
</script>

<style scoped>
.cp-wrap { display: flex; flex-direction: column; gap: 12px; }
.cp-head { display: flex; flex-direction: column; gap: 2px; }
.cp-title {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: 8px;
}
.cp-title .material-symbols-outlined { font-size: 20px; color: var(--color-primary); }
.cp-hint { font-size: 12.5px; color: var(--color-text-dim); }
.cp-hint-coin { font-size: 12px; }

.cp-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 14px;
}
.cp-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 14px 10px;
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  font: inherit;
  color: var(--color-text);
  cursor: default;
  text-align: center;
}
button.cp-card { cursor: pointer; transition: transform 0.1s, border-color 0.15s; }
button.cp-card:hover:not(:disabled) { border-color: color-mix(in oklch, var(--color-primary) 45%, var(--color-outline-dim)); }
button.cp-card:active:not(:disabled) { transform: scale(0.97); }
button.cp-card:disabled { cursor: default; }
.cp-card.disabled { opacity: 0.6; }
.cp-card.mine { border-color: color-mix(in oklch, var(--color-primary) 45%, var(--color-outline-dim)); }

.cp-figure {
  position: relative;
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  margin-bottom: 4px;
}
.cp-figure.pulse { animation: cp-pulse 0.45s cubic-bezier(0.34, 1.56, 0.64, 1); }
@keyframes cp-pulse {
  0% { transform: scale(1); }
  40% { transform: scale(1.14) rotate(-3deg); }
  100% { transform: scale(1); }
}
@media (prefers-reduced-motion: reduce) { .cp-figure.pulse { animation: none; } }
.cp-emoji { font-size: 32px; line-height: 1; }
.cp-figure.sick .cp-emoji { filter: grayscale(0.55) brightness(0.92); }
.cp-hat { position: absolute; top: -8px; right: -2px; font-size: 18px; transform: rotate(12deg); }
.cp-sick { position: absolute; bottom: -4px; left: -4px; font-size: 16px; }

.cp-hearts { position: absolute; inset: 0; pointer-events: none; }
.cp-heart {
  position: absolute;
  bottom: 8%;
  font-size: 16px;
  transform: translateX(-50%);
}
.cp-heart-enter-active { transition: opacity 0.15s, transform 0.65s ease-out; }
.cp-heart-enter-from { opacity: 0; }
.cp-heart-leave-active { transition: opacity 0.4s, transform 0.4s; }
.cp-heart-leave-to { opacity: 0; transform: translate(-50%, -36px); }

.cp-name {
  font-size: 13px;
  font-weight: 700;
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cp-owner {
  font-size: 11px;
  color: var(--color-text-dim);
  max-width: 100%;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.cp-stage {
  font-size: 10.5px;
  font-weight: 700;
  padding: 1px 8px;
  border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  margin-top: 2px;
}

.cp-tag {
  margin-top: 8px;
  display: inline-flex;
  align-items: center;
  gap: 5px;
  font-size: 11.5px;
  font-weight: 700;
  color: var(--color-text-dim);
  min-height: 28px;
}
.cp-tag .material-symbols-outlined { font-size: 14px; }
.cp-tag-coin { font-size: 11px; }
.cp-tag.action {
  border-radius: var(--radius-full);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  padding: 6px 12px;
}
.cp-tag.done { color: var(--color-success); }
</style>
