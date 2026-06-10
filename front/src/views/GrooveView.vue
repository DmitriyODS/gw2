<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <div class="page-head">
        <div class="page-head-text">
          <h1 class="groove-title">Мой Groove</h1>
          <div class="groove-meta">
            <span v-if="pet" class="meta-stat">
              <span class="meta-emoji">🫘</span>
              <strong>{{ pet.beans }}</strong> грувов
            </span>
            <span v-if="pet?.feed_streak" class="meta-stat warning">
              <span class="material-symbols-outlined">local_fire_department</span>
              стрик <strong>{{ pet.feed_streak }}</strong> дн.
            </span>
          </div>
        </div>
        <div class="head-actions">
          <button class="wrapped-btn" @click="showWrapped = true">
            <span class="material-symbols-outlined">auto_awesome</span>
            <span class="desktop-only-label">Моя неделя</span>
          </button>
          <button class="kudos-btn desktop-only" @click="showKudos = true">
            <span class="material-symbols-outlined">volunteer_activism</span>
            <span>Поблагодарить</span>
          </button>
        </div>
      </div>
    </header>

    <div class="admin-body">
      <LiveNowBar class="groove-live" />

      <div class="groove-layout">
        <aside class="groove-aside">
          <PetCard @open-shop="showShop = true" />
          <RaidCard />
        </aside>

        <main class="groove-main">
          <ZooStrip />

          <div v-if="!groove.events.length && groove.loadingFeed" class="groove-empty">
            Загружаем ленту…
          </div>
          <div v-else-if="!groove.events.length" class="groove-empty">
            <span class="groove-empty-icon">👾</span>
            <h3>Лента пока пуста</h3>
            <p>Запустите юнит или закройте задачу — здесь появится первая опорная точка</p>
          </div>

          <!-- Река дня: дни — полноширинные секции, события внутри
               раскладываются адаптивным гридом и заполняют всю ширину. -->
          <div v-else ref="riverEl" class="groove-river">
            <section v-for="day in days" :key="day.key" class="river-day">
              <header class="river-day-head">
                <span class="river-day-dot" aria-hidden="true"></span>
                <h2 class="river-day-title">{{ day.title }}</h2>
                <span class="river-day-line" aria-hidden="true"></span>
                <span class="river-day-count">{{ dayCount(day) }}</span>
              </header>
              <div class="river-day-body">
                <template v-for="zone in day.zones" :key="day.key + zone.key + zone.events[0].id">
                  <div class="river-zone">
                    <span class="material-symbols-outlined">{{ zone.icon }}</span>
                    {{ zone.title }}
                  </div>
                  <FeedCard v-for="event in zone.events" :key="event.id" :event="event" />
                </template>
              </div>
            </section>

            <div ref="sentinelEl" class="river-sentinel" aria-hidden="true">
              <span v-if="groove.loadingFeed" class="river-loading">…</span>
            </div>
          </div>
        </main>
      </div>
    </div>

    <AppFab
      icon="volunteer_activism"
      label="Спасибо"
      aria-label="Поблагодарить коллегу"
      @click="showKudos = true"
    />

    <PetShopDialog v-model="showShop" />
    <KudosDialog v-model="showKudos" />
    <WrappedDialog v-model="showWrapped" />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { dayKey, dayTitle, timeZoneOf } from '@/utils/groove.js'
import AppFab from '@/components/common/AppFab.vue'
import LiveNowBar from '@/components/groove/LiveNowBar.vue'
import PetCard from '@/components/groove/PetCard.vue'
import RaidCard from '@/components/groove/RaidCard.vue'
import ZooStrip from '@/components/groove/ZooStrip.vue'
import FeedCard from '@/components/groove/FeedCard.vue'
import PetShopDialog from '@/components/groove/PetShopDialog.vue'
import KudosDialog from '@/components/groove/KudosDialog.vue'
import WrappedDialog from '@/components/groove/WrappedDialog.vue'

const groove = useGrooveStore()

const showShop = ref(false)
const showKudos = ref(false)
const showWrapped = ref(false)
const riverEl = ref(null)
const sentinelEl = ref(null)

const pet = computed(() => groove.pet)

// События сгруппированы: день → временные зоны (Утро/День/Вечер).
const days = computed(() => {
  const map = new Map()
  for (const e of groove.events) {
    const key = dayKey(e.created_at)
    if (!map.has(key)) map.set(key, [])
    map.get(key).push(e)
  }
  return [...map.entries()].map(([key, events]) => ({
    key,
    title: dayTitle(key),
    zones: groupZones(events),
  }))
})

function dayCount(day) {
  const n = day.zones.reduce((sum, z) => sum + z.events.length, 0)
  if (n === 1) return '1 событие'
  if (n >= 2 && n <= 4) return `${n} события`
  return `${n} событий`
}

function groupZones(events) {
  const zones = []
  for (const e of events) {
    const z = timeZoneOf(e.created_at)
    const last = zones[zones.length - 1]
    if (last && last.key === z.key) last.events.push(e)
    else zones.push({ ...z, events: [e] })
  }
  return zones
}

let observer = null

onMounted(async () => {
  await Promise.allSettled([
    groove.fetchFeed(),
    groove.fetchLive(),
    groove.fetchPet(),
    groove.fetchRaid(),
    groove.fetchZoo(),
  ])
  setupObserver()
})

// Sentinel в конце реки: виден — догружаем старое (работает и для
// горизонтального, и для вертикального скролла — clipping учитывается).
function setupObserver() {
  if (observer || typeof IntersectionObserver === 'undefined') return
  observer = new IntersectionObserver((entries) => {
    if (entries.some(e => e.isIntersecting)) {
      groove.loadMore().catch(() => {})
    }
  }, { rootMargin: '200px' })
  if (sentinelEl.value) observer.observe(sentinelEl.value)
}

watch(sentinelEl, (el, old) => {
  if (!observer) return
  if (old) observer.unobserve(old)
  if (el) observer.observe(el)
})

onBeforeUnmount(() => {
  observer?.disconnect()
  observer = null
})
</script>

<style scoped>
.groove-title { margin: 0; font-size: 26px; font-weight: 800; }
.page-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}
.page-head-text { display: flex; flex-direction: column; gap: 8px; }
.groove-meta { display: flex; gap: 8px; flex-wrap: wrap; font-size: 13px; }
.meta-emoji { font-size: 14px; }

.kudos-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 14px;
  font-weight: 600;
  padding: 11px 20px;
  cursor: pointer;
  transition: transform 0.1s;
}
.kudos-btn:active { transform: scale(0.97); }
.head-actions { display: flex; gap: 8px; align-items: center; }
.wrapped-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  font-size: 14px;
  font-weight: 600;
  padding: 11px 18px;
  cursor: pointer;
  transition: transform 0.1s;
}
.wrapped-btn:active { transform: scale(0.97); }
.wrapped-btn .material-symbols-outlined { font-size: 18px; }

.groove-live { margin-bottom: 16px; }

.groove-layout {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 16px;
  align-items: start;
}
.groove-aside {
  display: flex;
  flex-direction: column;
  gap: 16px;
  position: sticky;
  top: 0;
}
.groove-main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}

/* ── Река дня: полноширинные дни, грид карточек ───────────── */
.groove-river {
  display: flex;
  flex-direction: column;
  gap: 22px;
  padding: 4px 2px 14px;
}
.river-day-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-bottom: 12px;
}
.river-day-dot {
  width: 14px;
  height: 14px;
  border-radius: 50%;
  background: var(--color-surface);
  border: 3px solid var(--color-primary);
  flex-shrink: 0;
}
.river-day-title {
  margin: 0;
  font-size: 15px;
  font-weight: 700;
  white-space: nowrap;
}
.river-day-line {
  flex: 1;
  height: 2px;
  background: var(--color-outline-dim);
  border-radius: 1px;
}
.river-day-count {
  font-size: 12px;
  color: var(--color-text-dim);
  white-space: nowrap;
}
.river-day-body {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 12px;
  align-items: start;
}
.river-zone {
  grid-column: 1 / -1;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11.5px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-dim);
  margin-top: 2px;
}
.river-zone .material-symbols-outlined { font-size: 15px; }
.river-sentinel {
  width: 100%;
  height: 32px;
  display: grid;
  place-items: center;
}
.river-loading { color: var(--color-text-dim); }

.groove-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  padding: 48px 16px;
  text-align: center;
  color: var(--color-text-dim);
}
.groove-empty-icon {
  width: 72px;
  height: 72px;
  border-radius: 50%;
  background: var(--color-primary-container);
  display: grid;
  place-items: center;
  font-size: 34px;
  margin-bottom: 6px;
}
.groove-empty h3 { margin: 0; color: var(--color-text); }
.groove-empty p { margin: 0; font-size: 13.5px; max-width: 360px; }

/* ── Мобильная вертикаль ──────────────────────────────────── */
@media (max-width: 1100px) {
  .groove-layout { grid-template-columns: 1fr; }
  .groove-aside { position: static; }
  .groove-river { gap: 16px; }
}

.desktop-only { display: inline-flex; }
@media (max-width: 768px) {
  .desktop-only { display: none; }
  .desktop-only-label { display: none; }
  .wrapped-btn { padding: 10px 12px; }
  .groove-title { font-size: 22px; }
}
</style>
