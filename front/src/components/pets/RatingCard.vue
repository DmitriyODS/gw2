<template>
  <section class="rating-card">
    <div class="rating-head">
      <span class="rating-icon material-symbols-outlined">social_leaderboard</span>
      <div class="rating-title-wrap">
        <h3 class="rating-title">Топ недели</h3>
        <p v-if="rows.length" class="rating-subtitle">Кудосы с начала недели среди {{ totalLabel }}</p>
      </div>
    </div>

    <EmptyState
      v-if="!rows.length"
      icon="social_leaderboard"
      size="sm"
      title="Рейтинг пока пуст"
      subtitle="Кудосы за работу появятся здесь в течение недели"
    />
    <ol v-else class="rating-list">
      <li
        v-for="row in rows"
        :key="row.user?.id ?? row.position"
        class="rating-row"
        :class="{ mine: isMine(row), gap: row.gapBefore }"
      >
        <span class="rating-pos" :class="'p' + row.position">{{ row.position }}</span>
        <img class="rating-avatar" :src="avatarUrl(row.user)" alt="" loading="lazy" />
        <div class="rating-info">
          <span class="rating-pet">
            <span class="rating-pet-emoji"><EmojiGlyph :char="petEmoji(row)" /></span>
            {{ row.pet_name }}
            <span v-if="(row.generation || 1) >= 2" class="rating-gen" :title="`${row.generation}-е поколение`">
              🌟{{ row.generation }}
            </span>
            <span v-if="row.sick" title="Болеет">🤒</span>
          </span>
          <span class="rating-owner">{{ row.user?.fio || '—' }}</span>
          <span class="rating-bar" aria-hidden="true">
            <span class="rating-fill" :style="{ width: barPercent(row) + '%' }"></span>
          </span>
        </div>
        <span class="rating-scores">
          <span class="rating-kudos" title="Кудосы с начала недели">
            <span class="material-symbols-outlined">favorite</span>
            {{ row.kudos_week || 0 }}
          </span>
          <span class="rating-xp">{{ row.xp }} XP</span>
        </span>
      </li>
    </ol>
  </section>
</template>

<script setup>
import { computed } from 'vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import { usePetsStore } from '@/stores/pets.js'
import { avatarUrl, petEmoji } from '@/utils/pets.js'
import EmptyState from '@/components/common/EmptyState.vue'

const pets = usePetsStore()
const TOP_LIMIT = 10

const rating = computed(() => pets.rating)

// Топ-10 плюс собственная строка отдельно, если не попал в топ.
const rows = computed(() => {
  const r = rating.value
  if (!r?.items?.length) return []
  const top = r.items.slice(0, TOP_LIMIT)
  const me = r.me
  if (me && !top.some(i => i.user?.id === me.user?.id)) {
    return [...top, { ...me, gapBefore: me.position > TOP_LIMIT + 1 }]
  }
  return top
})

const totalLabel = computed(() => {
  const n = rating.value?.total ?? 0
  if (n % 10 === 1 && n % 100 !== 11) return `${n} питомца`
  return `${n} питомцев`
})

// Бар — по кудосам текущей ISO-недели (механика рейтинга), не по XP.
const maxKudos = computed(() => Math.max(1, ...rows.value.map(r => r.kudos_week || 0)))

function barPercent(row) {
  return Math.max(4, Math.round(((row.kudos_week || 0) / maxKudos.value) * 100))
}

function isMine(row) {
  return row.user?.id === pets.myId
}
</script>

<style scoped>
.rating-card {
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 16px);
  padding: 14px 16px;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.rating-head { display: flex; align-items: center; gap: 12px; }
.rating-icon {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: color-mix(in oklch, var(--color-tertiary) 16%, transparent);
  color: var(--color-tertiary);
  display: grid;
  place-items: center;
  font-size: 22px;
  flex-shrink: 0;
}
.rating-title { margin: 0; font-size: 14.5px; font-weight: 700; }
.rating-subtitle { margin: 0; font-size: 12px; color: var(--color-text-dim); }

.rating-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.rating-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border-radius: 12px;
}
.rating-row.mine {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
/* Разрыв перед строкой владельца, выпавшего из топа. */
.rating-row.gap {
  position: relative;
  margin-top: 10px;
}
.rating-row.gap::before {
  content: '···';
  position: absolute;
  top: -13px;
  left: 14px;
  font-size: 12px;
  letter-spacing: 2px;
  color: var(--color-text-dim);
}
.rating-pos {
  width: 22px;
  font-size: 12.5px;
  font-weight: 800;
  text-align: center;
  color: var(--color-text-dim);
  flex-shrink: 0;
}
.rating-pos.p1 { color: var(--color-warning); }
.rating-pos.p2 { color: var(--color-text); }
.rating-pos.p3 { color: var(--color-tertiary); }
.rating-avatar {
  width: 30px;
  height: 30px;
  border-radius: 50%;
  object-fit: cover;
  flex-shrink: 0;
  background: var(--color-surface-high);
}
.rating-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.rating-pet {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rating-pet-emoji { font-size: 15px; }
.rating-gen {
  font-size: 10px;
  font-weight: 800;
  padding: 0 6px;
  border-radius: var(--radius-full);
  background: linear-gradient(120deg,
    color-mix(in oklch, var(--color-tertiary-container) 90%, transparent),
    color-mix(in oklch, var(--color-primary-container) 90%, transparent));
  color: var(--color-on-tertiary-container);
}
.rating-owner {
  font-size: 11px;
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.rating-row.mine .rating-owner { color: inherit; opacity: 0.75; }
.rating-bar {
  height: 4px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
  margin-top: 2px;
}
.rating-row.mine .rating-bar {
  background: color-mix(in oklch, var(--color-on-primary-container) 14%, transparent);
}
.rating-fill {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--color-primary);
  transition: width 0.5s ease;
}
.rating-row.mine .rating-fill { background: var(--color-primary); }
.rating-scores {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}
.rating-kudos {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11.5px;
  font-weight: 700;
  color: var(--color-success);
  white-space: nowrap;
}
.rating-kudos .material-symbols-outlined { font-size: 13px; }
.rating-xp {
  font-size: 12px;
  font-weight: 700;
  white-space: nowrap;
  color: var(--color-text-dim);
}
.rating-row.mine .rating-xp { color: inherit; }
</style>
