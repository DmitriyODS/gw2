<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <!-- Тулбар в стиле «Задач»: статы-чипы слева, главное действие справа. -->
      <div class="pets-toolbar">
        <span v-if="pet" class="chip-tint chip-tint--primary">
          <KudosCoin class="meta-emoji" />
          <strong>{{ pet.kudos }}</strong>&nbsp;кудосов
        </span>
        <span v-if="pet?.feed_streak" class="chip-tint chip-tint--warning">
          <span class="material-symbols-outlined">local_fire_department</span>
          стрик&nbsp;<strong>{{ pet.feed_streak }}</strong>&nbsp;дн.
        </span>
        <button class="btn-grad pets-shop-btn" type="button" @click="showShop = true">
          <span class="material-symbols-outlined">storefront</span>
          <span class="shop-cta-label">Магазин</span>
        </button>
      </div>
    </header>

    <div class="admin-body">
      <LiveNowBar class="pets-live" />

      <section class="pets-overview">
        <div v-if="pet" class="pet-summary">
          <div class="pet-summary-figure" :class="{ sick: pet.sick }">
            <span class="pet-summary-emoji">{{ petEmoji(pet) }}</span>
            <span v-if="hatEmoji" class="pet-summary-hat">{{ hatEmoji }}</span>
            <span v-if="pet.sick" class="pet-summary-sick" title="Болеет">🤒</span>
          </div>
          <div class="pet-summary-info">
            <h2 class="pet-summary-name">{{ pet.name }}</h2>
            <p class="pet-summary-stage">
              {{ stageTitle }}<template v-if="speciesTitle"> · {{ speciesTitle }}</template>
            </p>
            <div v-if="!pet.sick" class="pet-summary-xp">
              <div class="pet-summary-xp-bar"><div class="pet-summary-xp-fill" :style="{ width: xpPercent + '%' }"></div></div>
              <span v-if="pet.next_stage_xp" class="pet-summary-xp-label">{{ pet.xp }} / {{ pet.next_stage_xp }} XP</span>
            </div>
            <p v-else class="pet-summary-sick-hint">
              Болеет — {{ pet.recovery }}/{{ pet.recovery_target }} к выздоровлению
            </p>
          </div>
          <button class="pet-summary-open" type="button" @click="detailOpen = true">
            Открыть питомца
          </button>
        </div>

        <div v-if="quest" class="quest-card" :class="{ done: quest.done, claimed: quest.claimed }">
          <div class="quest-head">
            <span class="material-symbols-outlined">
              {{ quest.claimed ? 'check_circle' : (quest.done ? 'rocket_launch' : 'flag') }}
            </span>
            <span class="quest-title">{{ quest.title }}</span>
            <span class="quest-reward">+{{ quest.reward }} <KudosCoin /></span>
          </div>
          <div class="quest-bar"><div class="quest-fill" :style="{ width: questPercent + '%' }"></div></div>
          <div class="quest-meta">
            <span>{{ quest.progress }} / {{ quest.target }} {{ quest.unit }}</span>
            <span v-if="quest.claimed" class="quest-claimed">Награда получена</span>
            <span v-else-if="!quest.done" class="quest-hint">{{ quest.hint }}</span>
          </div>
        </div>
      </section>

      <section class="pets-section">
        <SeasonTrackCard />
      </section>

      <section ref="colleaguesEl" class="pets-section">
        <ColleaguesPets />
      </section>

      <section ref="ratingEl" class="pets-section">
        <RatingCard />
      </section>
    </div>

    <PetShopDialog v-model="showShop" />
    <PetDetailModal v-if="detailOpen" @close="detailOpen = false" />
  </div>
</template>

<script setup>
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import LiveNowBar from '@/components/pets/LiveNowBar.vue'
import RatingCard from '@/components/pets/RatingCard.vue'
import SeasonTrackCard from '@/components/pets/SeasonTrackCard.vue'
import ColleaguesPets from '@/components/pets/ColleaguesPets.vue'
import PetShopDialog from '@/components/pets/PetShopDialog.vue'
import PetDetailModal from '@/components/pets/PetDetailModal.vue'
import { usePetsStore } from '@/stores/pets.js'
import { petEmoji, PET_STAGES, PET_SPECIES, shopItemEmoji } from '@/utils/pets.js'

const route = useRoute()
const pets = usePetsStore()

const showShop = ref(false)
const detailOpen = ref(false)
const colleaguesEl = ref(null)
const ratingEl = ref(null)

// Страница — единый скролл без вкладок; ?tab= остаётся рабочим (ссылки из
// PetDetailModal и старые закладки): shop открывает магазин, остальные —
// прокрутка к секции.
function applyTabQuery(tab) {
  if (!tab) return
  if (tab === 'shop') {
    showShop.value = true
    return
  }
  const el = tab === 'colleagues' ? colleaguesEl.value : (tab === 'rating' ? ratingEl.value : null)
  if (el) nextTick(() => el.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}
watch(() => route.query.tab, applyTabQuery)

const pet = computed(() => pets.pet)
const stageTitle = computed(() => PET_STAGES[pet.value?.stage] || '')
const speciesTitle = computed(() => (pet.value?.stage >= 2 ? PET_SPECIES[pet.value.species]?.title : ''))
const hatEmoji = computed(() => (pet.value?.hat ? shopItemEmoji({ kind: 'accessory', key: pet.value.hat }) : null))
const quest = computed(() => pet.value?.quest || null)

const xpPercent = computed(() => {
  if (!pet.value?.next_stage_xp) return 100
  return Math.min(100, Math.round((pet.value.xp / pet.value.next_stage_xp) * 100))
})
const questPercent = computed(() => {
  const q = quest.value
  if (!q || !q.target) return 0
  return Math.min(100, Math.round((q.progress / q.target) * 100))
})

onMounted(async () => {
  await Promise.allSettled([
    pets.pet ? Promise.resolve() : pets.fetchPet(),
    pets.fetchZoo(),
    pets.fetchRating(),
    pets.fetchLive(),
  ])
  applyTabQuery(route.query.tab)
})
</script>

<style scoped>
/* Тулбар без подложки — прозрачная «плавающая» шапка как в «Задачах». */
.admin-sticky { background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; }
.admin-sticky::after { display: none; }

.pets-toolbar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.pets-shop-btn { margin-left: auto; }
.meta-emoji { font-size: 14px; }

.pets-live { margin-bottom: 16px; }

.pets-overview { display: flex; flex-direction: column; gap: 16px; }
.pets-section { margin-top: 24px; scroll-margin-top: 90px; }

.pet-summary {
  display: flex;
  align-items: center;
  gap: 16px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 20px);
  padding: 18px 20px;
  flex-wrap: wrap;
}
.pet-summary-figure {
  position: relative;
  width: 76px;
  height: 76px;
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.pet-summary-emoji { font-size: 40px; }
.pet-summary-figure.sick .pet-summary-emoji { filter: grayscale(0.55) brightness(0.92); }
.pet-summary-hat { position: absolute; top: -8px; right: -2px; font-size: 20px; transform: rotate(12deg); }
.pet-summary-sick { position: absolute; bottom: -4px; left: -4px; font-size: 18px; }
.pet-summary-info { flex: 1; min-width: 180px; display: flex; flex-direction: column; gap: 4px; }
.pet-summary-name { margin: 0; font-size: 18px; font-weight: 700; }
.pet-summary-stage { margin: 0; font-size: 13px; color: var(--color-text-dim); }
.pet-summary-xp { display: flex; align-items: center; gap: 8px; margin-top: 4px; }
.pet-summary-xp-bar { flex: 1; height: 7px; border-radius: var(--radius-full); background: var(--color-surface-high); overflow: hidden; max-width: 240px; }
.pet-summary-xp-fill { height: 100%; border-radius: inherit; background: var(--color-primary); }
.pet-summary-xp-label { font-size: 11.5px; color: var(--color-text-dim); white-space: nowrap; }
.pet-summary-sick-hint { margin: 4px 0 0; font-size: 12.5px; color: var(--color-error); }
.pet-summary-open {
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  font-size: 13px;
  font-weight: 700;
  padding: 10px 18px;
  cursor: pointer;
  flex-shrink: 0;
}

.quest-card {
  border: 1px solid color-mix(in oklch, var(--color-tertiary) 35%, var(--color-outline-dim));
  border-radius: var(--radius-lg, 16px);
  padding: 14px 16px;
  background: color-mix(in oklch, var(--color-tertiary-container) 30%, transparent);
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.quest-card.done { border-color: var(--color-success); background: color-mix(in oklch, var(--color-success) 12%, transparent); }
.quest-card.claimed { border-color: var(--color-outline-dim); background: var(--color-surface-high); opacity: 0.85; }
.quest-head { display: flex; align-items: center; gap: 8px; font-size: 14px; font-weight: 700; }
.quest-title { flex: 1; min-width: 0; }
.quest-reward { font-size: 12.5px; font-weight: 700; color: var(--color-text-dim); }
.quest-bar { height: 6px; border-radius: var(--radius-full); background: var(--color-surface); overflow: hidden; }
.quest-fill { height: 100%; border-radius: inherit; background: var(--color-tertiary); }
.quest-card.done .quest-fill { background: var(--color-success); }
.quest-meta { display: flex; justify-content: space-between; font-size: 12px; color: var(--color-text-dim); }
.quest-claimed { font-weight: 700; color: var(--color-success); }

@media (max-width: 768px) {
  .shop-cta-label { display: none; }
}
</style>
