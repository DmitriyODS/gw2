<template>
  <section
    v-if="raid"
    class="raid-card"
    :class="{ defeated: raid.defeated, expanded }"
    @click="expanded = !expanded"
  >
    <div class="raid-head">
      <span class="raid-boss-emoji" :class="{ shake: !raid.defeated }">{{ bossEmoji }}</span>
      <div class="raid-title-wrap">
        <h3 class="raid-title">{{ raid.defeated ? 'Босс повержен!' : 'Рейд недели' }}</h3>
        <p class="raid-subtitle">{{ raid.boss }}</p>
      </div>
      <span class="raid-chevron material-symbols-outlined" aria-hidden="true">expand_more</span>
    </div>

    <div class="raid-hp">
      <div class="raid-hp-meta">
        <span>{{ raid.defeated ? 'Победа!' : 'HP босса' }}</span>
        <span>{{ raid.progress }} / {{ raid.target }} задач</span>
      </div>
      <div class="raid-hp-bar">
        <div class="raid-hp-fill" :style="{ width: hpPercent + '%' }"></div>
      </div>
    </div>

    <p class="raid-note">
      <template v-if="raid.defeated">
        Команда справилась — всем Грувикам {{ rewardTitle }} {{ rewardEmoji }}
      </template>
      <template v-else>
        Бьём босса закрытыми задачами — вся компания вместе.
        Награда всем: {{ rewardTitle }} {{ rewardEmoji }}
        <span v-if="raid.days_left" class="raid-days">· осталось {{ daysLabel }}</span>
      </template>
    </p>
  </section>
</template>

<script setup>
import { computed, ref } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { BOSS_EMOJI, SHOP_ITEMS } from '@/utils/groove.js'

const groove = useGrooveStore()
const raid = computed(() => groove.raid)

// На мобиле карточка свёрнута в тонкую полосу — описание по тапу.
const expanded = ref(false)

const bossEmoji = computed(() => BOSS_EMOJI[raid.value?.boss] || '👾')
const rewardTitle = computed(() => SHOP_ITEMS[raid.value?.reward]?.title || 'награда')
const rewardEmoji = computed(() => SHOP_ITEMS[raid.value?.reward]?.emoji || '🎁')

// HP босса убывает с каждым закрытием.
const hpPercent = computed(() => {
  if (!raid.value?.target) return 100
  const left = Math.max(0, raid.value.target - raid.value.progress)
  return Math.round((left / raid.value.target) * 100)
})

const daysLabel = computed(() => {
  const d = raid.value?.days_left ?? 0
  if (d === 1) return '1 день'
  if (d >= 2 && d <= 4) return `${d} дня`
  return `${d} дней`
})
</script>

<style scoped>
.raid-card {
  background: var(--color-surface);
  border: 1px solid color-mix(in oklch, var(--color-error) 35%, transparent);
  border-radius: var(--radius-lg, 16px);
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.raid-card.defeated {
  border-color: color-mix(in oklch, var(--color-success) 45%, transparent);
}
.raid-head { display: flex; align-items: center; gap: 12px; }
.raid-boss-emoji {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: color-mix(in oklch, var(--color-error) 14%, transparent);
  display: grid;
  place-items: center;
  font-size: 26px;
  flex-shrink: 0;
}
.defeated .raid-boss-emoji {
  background: color-mix(in oklch, var(--color-success) 16%, transparent);
  filter: grayscale(0.6);
}
.raid-boss-emoji.shake { animation: raid-shake 4s ease-in-out infinite; }
@keyframes raid-shake {
  0%, 88%, 100% { transform: rotate(0); }
  90% { transform: rotate(-8deg) scale(1.06); }
  93% { transform: rotate(7deg); }
  96% { transform: rotate(-4deg); }
}
.raid-title { margin: 0; font-size: 14.5px; font-weight: 700; }
.raid-subtitle { margin: 0; font-size: 13px; color: var(--color-text-dim); }

.raid-hp-meta {
  display: flex;
  justify-content: space-between;
  font-size: 11.5px;
  color: var(--color-text-dim);
  margin-bottom: 4px;
}
.raid-hp-bar {
  height: 10px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.raid-hp-fill {
  height: 100%;
  border-radius: inherit;
  background: var(--color-error);
  transition: width 0.6s ease;
}
.defeated .raid-hp-fill { background: var(--color-success); }
.raid-note {
  margin: 0;
  font-size: 12.5px;
  line-height: 1.45;
  color: var(--color-text-dim);
}
.raid-days { white-space: nowrap; }
.raid-chevron {
  display: none;
  margin-left: auto;
  color: var(--color-text-dim);
  font-size: 22px;
  transition: transform 0.25s;
  flex-shrink: 0;
}

/* На мобильной вертикали рейд не должен отъедать экран у ленты:
   компактная полоса, подробности — по тапу. */
@media (max-width: 1100px) {
  .raid-card { padding: 10px 14px; gap: 6px; cursor: pointer; }
  .raid-boss-emoji { width: 36px; height: 36px; font-size: 20px; }
  .raid-title { font-size: 13.5px; }
  .raid-subtitle { font-size: 12px; }
  .raid-hp-bar { height: 7px; }
  .raid-chevron { display: block; }
  .raid-card.expanded .raid-chevron { transform: rotate(180deg); }
  .raid-card:not(.expanded) .raid-note { display: none; }
}
</style>
