<template>
  <section v-if="season" class="stc">
    <div class="stc-head">
      <h3 class="stc-title">
        <span class="material-symbols-outlined">military_tech</span>
        Сезон {{ seasonLabel }}
      </h3>
      <span class="stc-ends">до {{ endsLabel }}</span>
    </div>
    <p class="stc-hint">Кудосы за работу двигают по треку — награды остаются навсегда.</p>

    <div class="stc-progress">
      <div class="stc-bar">
        <div class="stc-fill" :style="{ width: fillPercent + '%' }"></div>
      </div>
      <span class="stc-progress-label">
        <KudosCoin /> <strong>{{ season.kudos }}</strong>&nbsp;/ {{ maxThreshold }}
      </span>
    </div>

    <!-- Награды — равномерная сетка карточек-этапов (как трек в батл-пассах). -->
    <div class="stc-rewards">
      <div
        v-for="r in season.rewards"
        :key="r.threshold"
        class="stc-reward"
        :class="{ reached: r.reached, claimed: r.claimed }"
      >
        <div class="stc-reward-figure">
          <span class="stc-reward-emoji">{{ seasonRewardMeta(r).emoji }}</span>
          <span v-if="r.claimed" class="stc-reward-check material-symbols-outlined">check</span>
        </div>
        <span class="stc-reward-name">{{ seasonRewardMeta(r).title }}</span>

        <button
          v-if="r.reached && !r.claimed"
          class="stc-reward-claim"
          type="button"
          :disabled="claiming"
          @click="claim(r)"
        >Забрать</button>
        <span v-else-if="r.claimed" class="stc-reward-state done">Получено</span>
        <span v-else class="stc-reward-state">
          <KudosCoin class="stc-reward-coin" /> {{ r.threshold }}
        </span>
      </div>
    </div>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { seasonRewardMeta } from '@/utils/pets.js'

const pets = usePetsStore()
const notify = useNotificationsStore()
const claiming = ref(false)

const season = computed(() => pets.season)

const SEASON_TITLES = { Q1: 'зима–весна', Q2: 'весна–лето', Q3: 'лето–осень', Q4: 'осень–зима' }
const seasonLabel = computed(() => {
  const [year, q] = (season.value?.season || '').split('-')
  return q ? `${SEASON_TITLES[q] || q} ${year}` : ''
})
const endsLabel = computed(() => {
  if (!season.value?.ends_at) return ''
  return new Date(season.value.ends_at).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })
})

// Шкала до последнего порога; заработанное сверх — полная полоса.
const maxThreshold = computed(() => {
  const rewards = season.value?.rewards || []
  return rewards.length ? rewards[rewards.length - 1].threshold : 1
})
const fillPercent = computed(() =>
  Math.min(100, Math.round(((season.value?.kudos || 0) / maxThreshold.value) * 100)))

function rewardTitle(r) {
  const meta = seasonRewardMeta(r)
  return r.kind === 'kudos' ? meta.title : `«${meta.title}»`
}

async function claim(r) {
  if (!r.reached || r.claimed || claiming.value) return
  claiming.value = true
  try {
    await pets.claimSeasonReward(r.threshold)
    notify.success(`Награда сезона: ${rewardTitle(r)}`)
  } catch (e) {
    notify.warn(e?.message || 'Не удалось забрать награду')
  } finally {
    claiming.value = false
  }
}

onMounted(() => {
  if (!pets.season) pets.fetchSeason().catch(() => {})
})
</script>

<style scoped>
.stc {
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 20px);
  padding: 18px 20px;
}

.stc-head { display: flex; align-items: baseline; justify-content: space-between; gap: 10px; flex-wrap: wrap; }
.stc-title { margin: 0; display: flex; align-items: center; gap: 8px; font-size: 16px; font-weight: 800; }
.stc-title .material-symbols-outlined { font-size: 20px; color: var(--color-tertiary); }
.stc-ends { font-size: 12px; font-weight: 600; color: var(--color-text-dim); }
.stc-hint { margin: 4px 0 0; font-size: 12.5px; color: var(--color-text-dim); }

.stc-progress { display: flex; align-items: center; gap: 12px; margin-top: 14px; }
.stc-bar {
  flex: 1;
  height: 10px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.stc-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--color-primary), var(--color-tertiary));
  transition: width 0.6s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.stc-progress-label {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 12.5px; font-weight: 600; color: var(--color-text-dim);
  white-space: nowrap;
}
.stc-progress-label strong { color: var(--color-text); }

.stc-rewards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(104px, 1fr));
  gap: 10px;
  margin-top: 14px;
}

.stc-reward {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 12px 8px 10px;
  border-radius: 16px;
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  text-align: center;
  opacity: 0.72;
  transition: opacity 0.2s, border-color 0.2s, box-shadow 0.2s, transform 0.15s;
}
.stc-reward.reached { opacity: 1; }
/* Достигнута, но не забрана — «светится» и просится в руки. */
.stc-reward.reached:not(.claimed) {
  border-color: color-mix(in oklch, var(--color-tertiary) 60%, transparent);
  background: linear-gradient(160deg,
    color-mix(in oklch, var(--color-tertiary-container) 45%, var(--color-surface)),
    var(--color-surface));
  box-shadow: 0 4px 18px color-mix(in oklch, var(--color-tertiary) 22%, transparent);
  transform: translateY(-1px);
}
.stc-reward.claimed { opacity: 0.92; }

.stc-reward-figure {
  position: relative;
  width: 52px; height: 52px;
  border-radius: 50%;
  background: var(--color-surface-high);
  display: grid; place-items: center;
}
.stc-reward.reached:not(.claimed) .stc-reward-figure {
  background: color-mix(in oklch, var(--color-tertiary-container) 70%, transparent);
}
.stc-reward-emoji { font-size: 26px; line-height: 1; }
.stc-reward.claimed .stc-reward-emoji { filter: saturate(1.1); }
.stc-reward:not(.reached) .stc-reward-emoji { filter: grayscale(0.7); opacity: 0.8; }
.stc-reward-check {
  position: absolute;
  right: -3px; bottom: -3px;
  font-size: 15px;
  padding: 1px;
  border-radius: 50%;
  background: var(--color-success);
  color: var(--color-on-success);
}

.stc-reward-name {
  font-size: 11.5px;
  font-weight: 600;
  line-height: 1.25;
  color: var(--color-text);
  min-height: 2.5em;
  display: flex;
  align-items: center;
}
.stc-reward:not(.reached) .stc-reward-name { color: var(--color-text-dim); }

.stc-reward-claim {
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-tertiary);
  color: var(--color-on-tertiary);
  font-size: 11.5px;
  font-weight: 700;
  padding: 5px 14px;
  cursor: pointer;
  transition: filter 0.15s, transform 0.1s;
}
.stc-reward-claim:hover:not(:disabled) { filter: brightness(1.08); }
.stc-reward-claim:active:not(:disabled) { transform: scale(0.97); }
.stc-reward-claim:disabled { opacity: 0.6; cursor: progress; }

.stc-reward-state {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 11.5px; font-weight: 700; color: var(--color-text-dim);
}
.stc-reward-state.done { color: var(--color-success); }
.stc-reward-coin { font-size: 11px; }
</style>
