<template>
  <div class="tv-resp-wrap">
    <div class="tv-stage-eyebrow">
      <span class="material-symbols-outlined" style="color: var(--color-primary)">assignment_ind</span>
      <span>{{ slide.heroEyebrow }}</span>
    </div>
    <div v-if="respList.length === 0" class="tv-stage-empty">Ответственные пока не назначены</div>
    <ol v-else class="tv-resp">
      <li
        v-for="(r, i) in respList"
        :key="r.user_id"
        class="tv-resp-item"
        :style="{ '--row-delay': i * 80 + 'ms' }"
      >
        <span class="tv-resp-rank">{{ i + 1 }}</span>
        <img class="tv-resp-avatar" :src="avatarSrc(r)" alt="" />
        <span class="tv-resp-names">
          <span class="tv-resp-fio">{{ r.fio }}</span>
          <span v-if="r.post" class="tv-resp-post">{{ r.post }}</span>
        </span>
        <div class="tv-resp-bar">
          <div class="tv-resp-bar-fill" :style="{ '--bar-width': barPercent(r.open_count, respMax) + '%' }"></div>
        </div>
        <span class="tv-resp-counts">
          <span class="tv-resp-open"><TvCount :value="r.open_count" /> в работе</span>
          <span class="tv-resp-closed">{{ r.closed_count }} закрыто</span>
        </span>
      </li>
    </ol>
  </div>
</template>

<script setup>
// «Ответственные»: у кого сколько задач в работе и сколько уже закрыто.
import { computed } from 'vue'
import TvCount from '../TvCount.vue'
import { num, barPercent } from '../tvFormat.js'

const props = defineProps({
  slide: { type: Object, required: true },
  responsibles: { type: Array, default: () => [] }, // /api/stats/responsibles
})

const respList = computed(() =>
  [...props.responsibles]
    .sort((a, b) => num(b.open_count) - num(a.open_count) || num(b.closed_count) - num(a.closed_count))
    .slice(0, 5))
const respMax = computed(() => Math.max(1, ...respList.value.map(r => num(r.open_count))))

function avatarSrc(r) {
  if (r.avatar_path) return `/uploads/${r.avatar_path}`
  return `/api/users/${r.user_id}/identicon`
}
</script>

<style scoped>
.tv-resp-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: clamp(12px, 1.6vmin, 20px);
  min-height: 0;
}

.tv-resp {
  list-style: none;
  padding: 0;
  margin: 0;
  flex: 1;
  display: grid;
  grid-template-rows: repeat(5, 1fr);
  gap: clamp(6px, 0.8vmin, 12px);
  min-height: 0;
}

.tv-resp-item {
  display: grid;
  grid-template-columns: clamp(40px, 4.6vmin, 60px)
                         clamp(40px, 5.4vmin, 70px)
                         minmax(0, 1.4fr)
                         minmax(0, 2fr)
                         auto;
  gap: clamp(10px, 1.4vmin, 18px);
  align-items: center;
  padding: clamp(6px, 0.8vmin, 12px) clamp(10px, 1.4vmin, 18px);
  background: color-mix(in oklch, var(--color-surface-high) 60%, transparent);
  border-radius: clamp(12px, 1.4vmin, 18px);
  animation: tv-row-in 0.6s cubic-bezier(0.34, 1.56, 0.64, 1) backwards;
  animation-delay: var(--row-delay, 0ms);
}

.tv-resp-rank {
  font-size: clamp(20px, 2.6vmin, 32px);
  font-weight: 900;
  color: var(--color-primary);
  text-align: center;
}

.tv-resp-avatar {
  width: clamp(40px, 5.4vmin, 70px);
  height: clamp(40px, 5.4vmin, 70px);
  border-radius: 50%;
  object-fit: cover;
  border: 2px solid var(--color-primary);
}

.tv-resp-names {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.tv-resp-fio {
  font-size: clamp(15px, 2vmin, 24px);
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-resp-post {
  font-size: clamp(10px, 1.3vmin, 16px);
  color: var(--color-text-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tv-resp-bar {
  height: clamp(10px, 1.4vmin, 16px);
  background: var(--color-outline-dim);
  border-radius: 999px;
  overflow: hidden;
}

.tv-resp-bar-fill {
  height: 100%;
  background: linear-gradient(90deg,
    color-mix(in oklch, var(--color-tertiary) 80%, transparent),
    var(--color-tertiary));
  border-radius: 999px;
  width: 0;
  animation: tv-bar-fill 0.9s cubic-bezier(0.34, 1.56, 0.64, 1) forwards;
  animation-delay: calc(var(--row-delay, 0ms) + 200ms);
}

.tv-resp-counts {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
  white-space: nowrap;
}

.tv-resp-open {
  font-size: clamp(14px, 1.9vmin, 23px);
  font-weight: 800;
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
}

.tv-resp-closed {
  font-size: clamp(10px, 1.3vmin, 16px);
  font-weight: 600;
  color: var(--color-success);
}
</style>
