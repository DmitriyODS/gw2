<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <div class="page-head">
        <div class="page-head-text">
          <h1 class="groove-title">Мой Groove</h1>
          <div class="groove-meta">
            <span v-if="pet" class="meta-stat">
              <GrooveCoin class="meta-emoji" />
              <strong>{{ pet.beans }}</strong> грувов
            </span>
            <span v-if="pet?.feed_streak" class="meta-stat warning">
              <span class="material-symbols-outlined">local_fire_department</span>
              стрик <strong>{{ pet.feed_streak }}</strong> дн.
            </span>
          </div>
        </div>
        <div class="head-actions desktop-only">
          <button class="wrapped-btn" @click="showWrapped = true">
            <span class="material-symbols-outlined">auto_awesome</span>
            <span>Моя неделя</span>
          </button>
          <button class="kudos-btn" @click="showKudos = true">
            <span class="material-symbols-outlined">volunteer_activism</span>
            <span>Поблагодарить</span>
          </button>
        </div>
      </div>

      <!-- Компактный питомец в шапке: появляется на узких экранах,
           когда PetCard уезжает из вьюпорта. Тап возвращает к карточке. -->
      <transition name="petstrip">
        <button v-if="showPetStrip" class="pet-strip" type="button" @click="scrollToTop">
          <span class="pet-strip-emoji">{{ petEmoji(pet) }}</span>
          <span class="pet-strip-name">{{ pet.name }}</span>
          <span v-if="pet.sick" class="pet-strip-sick" title="Грувик болеет">🤒</span>
          <span class="pet-strip-chip"><GrooveCoin /> {{ pet.beans }}</span>
          <span v-if="pet.feed_streak" class="pet-strip-chip">🔥 {{ pet.feed_streak }}</span>
          <span class="pet-strip-bar" aria-hidden="true">
            <span class="pet-strip-fill" :style="{ width: stripXpPercent + '%' }"></span>
          </span>
        </button>
      </transition>
    </header>

    <div ref="bodyRef" class="admin-body">
      <LiveNowBar class="groove-live" />

      <div class="groove-layout">
        <aside class="groove-aside">
          <div ref="petBoxRef">
            <PetCard @open-shop="showShop = true" />
          </div>
          <RaidCard />
        </aside>

        <main ref="grooveMainRef" class="groove-main">
          <ZooStrip />

          <div v-if="newCount" class="feed-newpill-wrap">
            <button class="feed-newpill" type="button" @click="scrollToTop">
              <span class="material-symbols-outlined">arrow_upward</span>
              Новые события<template v-if="newCount > 1"> · {{ newCount }}</template>
            </button>
          </div>

          <div v-if="groove.events.length" class="feed-filters" role="tablist" aria-label="Фильтр ленты">
            <button
              v-for="f in FEED_FILTERS"
              :key="f.key"
              type="button"
              class="feed-filter-chip"
              :class="{ active: feedFilter === f.key }"
              role="tab"
              :aria-selected="feedFilter === f.key"
              @click="feedFilter = f.key"
            >{{ f.title }}</button>
          </div>

          <div v-if="!groove.events.length && groove.loadingFeed" class="groove-empty">
            Загружаем ленту…
          </div>
          <div v-else-if="!groove.events.length" class="groove-empty">
            <span class="groove-empty-icon">👾</span>
            <h3>Лента пока пуста</h3>
            <p>Запустите юнит или закройте задачу — здесь появится первая опорная точка</p>
          </div>
          <div v-else-if="!filteredEvents.length" class="groove-empty">
            <span class="groove-empty-icon">🔍</span>
            <h3>Ничего не нашлось</h3>
            <p>В загруженной части ленты таких событий нет — листайте, история подгружается</p>
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
                  <FeedCard
                    v-for="(event, i) in zone.events"
                    :key="event.id"
                    :event="event"
                    :style="{ '--i': Math.min(i, 8) }"
                  />
                </template>
              </div>
            </section>
          </div>

          <!-- Sentinel вне реки: продолжает подгрузку и когда фильтр
               «спрятал» все загруженные события. -->
          <div v-if="groove.events.length" ref="sentinelEl" class="river-sentinel" aria-hidden="true">
            <span v-if="groove.loadingFeed" class="river-loading">…</span>
          </div>
        </main>
      </div>
    </div>

    <!-- Мобильное FAB-меню: Спасибо / Моя неделя / Магазин -->
    <Teleport to="body">
      <transition name="gfab-back">
        <div v-if="fabOpen" class="gfab-backdrop" @click="fabOpen = false"></div>
      </transition>
      <div class="gfab" :class="{ open: fabOpen }">
        <div class="gfab-items">
          <button class="gfab-item" type="button" @click="fabAction(() => showShop = true)">
            <span class="gfab-item-label">Магазин</span>
            <span class="gfab-item-btn"><span class="material-symbols-outlined">storefront</span></span>
          </button>
          <button class="gfab-item" type="button" @click="fabAction(() => showWrapped = true)">
            <span class="gfab-item-label">Моя неделя</span>
            <span class="gfab-item-btn"><span class="material-symbols-outlined">auto_awesome</span></span>
          </button>
          <button class="gfab-item" type="button" @click="fabAction(() => showKudos = true)">
            <span class="gfab-item-label">Поблагодарить</span>
            <span class="gfab-item-btn"><span class="material-symbols-outlined">volunteer_activism</span></span>
          </button>
        </div>
        <button
          class="gfab-main"
          type="button"
          :aria-expanded="fabOpen"
          aria-label="Действия Groove"
          @click="fabOpen = !fabOpen"
        >
          <span v-if="fabOpen" class="material-symbols-outlined">close</span>
          <span v-else class="gfab-main-emoji">👾</span>
        </button>
      </div>
    </Teleport>

    <PetShopDialog v-model="showShop" />
    <KudosDialog v-model="showKudos" />
    <WrappedDialog v-model="showWrapped" />
    <GrooveCelebration />
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useGrooveStore } from '@/stores/groove.js'
import { dayKey, dayTitle, petEmoji, timeZoneOf } from '@/utils/groove.js'
import GrooveCoin from '@/components/groove/GrooveCoin.vue'
import LiveNowBar from '@/components/groove/LiveNowBar.vue'
import PetCard from '@/components/groove/PetCard.vue'
import RaidCard from '@/components/groove/RaidCard.vue'
import ZooStrip from '@/components/groove/ZooStrip.vue'
import FeedCard from '@/components/groove/FeedCard.vue'
import PetShopDialog from '@/components/groove/PetShopDialog.vue'
import KudosDialog from '@/components/groove/KudosDialog.vue'
import WrappedDialog from '@/components/groove/WrappedDialog.vue'
import GrooveCelebration from '@/components/groove/GrooveCelebration.vue'

const groove = useGrooveStore()

const showShop = ref(false)
const showKudos = ref(false)
const showWrapped = ref(false)
const riverEl = ref(null)
const sentinelEl = ref(null)
const bodyRef = ref(null)
const grooveMainRef = ref(null)
const petBoxRef = ref(null)
const fabOpen = ref(false)

const pet = computed(() => groove.pet)

// ── Фильтр ленты (клиентский, по уже загруженным событиям) ────
const FEED_FILTERS = [
  { key: 'all', title: 'Все' },
  { key: 'mine', title: 'Мои' },
  { key: 'milestones', title: 'Вехи' },
  { key: 'kudos', title: 'Спасибо' },
]
const MILESTONE_FILTER_KINDS = new Set([
  'streak', 'pet_evolved', 'pet_recovered', 'raid_won', 'wrapped', 'quest_done',
])
const feedFilter = ref('all')

const filteredEvents = computed(() => {
  const list = groove.events
  switch (feedFilter.value) {
    case 'mine': return list.filter(e => e.user?.id === groove.myId)
    case 'milestones': return list.filter(e => MILESTONE_FILTER_KINDS.has(e.kind))
    case 'kudos': return list.filter(e => e.kind === 'kudos')
    default: return list
  }
})

// ── Pill «Новые события»: пришло по сокету, а лента проскроллена ──
const newCount = ref(0)

watch(() => groove.events[0]?.id, (id, old) => {
  if (id == null || old == null || id === old) return
  const scrollTop = isNarrow.value
    ? (bodyRef.value?.scrollTop ?? 0)
    : (grooveMainRef.value?.scrollTop ?? 0)
  if (scrollTop > 300) newCount.value++
})

function onBodyScroll() {
  const scrollTop = isNarrow.value
    ? (bodyRef.value?.scrollTop ?? 0)
    : (grooveMainRef.value?.scrollTop ?? 0)
  if (newCount.value && scrollTop < 200) newCount.value = 0
}

function scrollToTop() {
  if (isNarrow.value) {
    bodyRef.value?.scrollTo({ top: 0, behavior: 'smooth' })
  } else {
    grooveMainRef.value?.scrollTo({ top: 0, behavior: 'smooth' })
  }
  newCount.value = 0
}

// ── Компактный питомец в шапке (узкие экраны) ─────────────────
const isNarrow = ref(false)
const petBoxVisible = ref(true)
let narrowMq = null
let petObserver = null

const showPetStrip = computed(() => isNarrow.value && !petBoxVisible.value && !!pet.value)

const stripXpPercent = computed(() => {
  const p = pet.value
  if (!p?.next_stage_xp) return 100
  return Math.min(100, Math.round((p.xp / p.next_stage_xp) * 100))
})

function fabAction(open) {
  fabOpen.value = false
  open()
}

// События сгруппированы: день → временные зоны (Утро/День/Вечер).
const days = computed(() => {
  const map = new Map()
  for (const e of filteredEvents.value) {
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
  bodyRef.value?.addEventListener('scroll', onBodyScroll, { passive: true })
  grooveMainRef.value?.addEventListener('scroll', onBodyScroll, { passive: true })

  narrowMq = window.matchMedia('(max-width: 1100px)')
  isNarrow.value = narrowMq.matches
  narrowMq.addEventListener('change', onNarrowChange)

  // Полоска питомца в шапке появляется, когда PetCard скрылась за
  // sticky-шапкой (rootMargin компенсирует её высоту).
  if (typeof IntersectionObserver !== 'undefined' && petBoxRef.value) {
    petObserver = new IntersectionObserver(([entry]) => {
      petBoxVisible.value = entry.isIntersecting
    }, { rootMargin: '-72px 0px 0px 0px' })
    petObserver.observe(petBoxRef.value)
  }

  await Promise.allSettled([
    groove.fetchFeed(),
    groove.fetchLive(),
    groove.fetchPet(),
    groove.fetchRaid(),
    groove.fetchZoo(),
  ])
  setupObserver()
})

function onNarrowChange(e) {
  isNarrow.value = e.matches
  // Пересоздаём обсервер: на десктопе root = grooveMainRef, на мобильном = viewport
  observer?.disconnect()
  observer = null
  if (sentinelEl.value) setupObserver()
}

// Sentinel в конце реки: виден — догружаем старое (работает и для
// горизонтального, и для вертикального скролла — clipping учитывается).
function setupObserver() {
  if (observer || typeof IntersectionObserver === 'undefined') return
  // На десктопе правая колонка — самостоятельный scroll-контейнер,
  // поэтому sentinel нужно наблюдать именно внутри него, а не в viewport.
  const root = isNarrow.value ? null : grooveMainRef.value
  observer = new IntersectionObserver((entries) => {
    if (entries.some(e => e.isIntersecting)) {
      groove.loadMore().catch(() => {})
    }
  }, { root, rootMargin: '200px' })
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
  petObserver?.disconnect()
  petObserver = null
  narrowMq?.removeEventListener('change', onNarrowChange)
  bodyRef.value?.removeEventListener('scroll', onBodyScroll)
  grooveMainRef.value?.removeEventListener('scroll', onBodyScroll)
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

/* На десктопе admin-body становится flex-колонкой без собственного скролла:
   LiveNowBar сверху фиксированной высоты, groove-layout занимает остаток. */
.admin-body {
  display: flex;
  flex-direction: column;
  overflow-y: hidden;
}
.groove-live { flex-shrink: 0; margin-bottom: 16px; }

/* ── Компактный питомец в шапке (узкие экраны) ─────────────── */
.pet-strip { display: none; }
@media (max-width: 1100px) {
  .pet-strip {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    margin-top: 10px;
    border: 1px solid var(--color-outline-dim);
    background: var(--color-surface);
    border-radius: var(--radius-full);
    padding: 6px 14px;
    cursor: pointer;
    font: inherit;
    color: var(--color-text);
    text-align: left;
  }
  .pet-strip:active { background: var(--color-surface-high); }
  .pet-strip-emoji { font-size: 20px; line-height: 1; }
  .pet-strip-sick { font-size: 14px; }
  .pet-strip-name {
    font-weight: 700;
    font-size: 13.5px;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    max-width: 96px;
  }
  .pet-strip-chip { font-size: 12px; font-weight: 600; white-space: nowrap; }
  .pet-strip-bar {
    flex: 1;
    min-width: 36px;
    height: 6px;
    border-radius: var(--radius-full);
    background: var(--color-surface-high);
    overflow: hidden;
  }
  .pet-strip-fill {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: var(--color-primary);
    transition: width 0.4s ease;
  }
}
.petstrip-enter-active, .petstrip-leave-active { transition: opacity 0.2s, transform 0.2s; }
.petstrip-enter-from, .petstrip-leave-to { opacity: 0; transform: translateY(-6px); }

/* ── Фильтр-чипы ленты ─────────────────────────────────────── */
.feed-filters { display: flex; gap: 8px; flex-wrap: wrap; }
.feed-filter-chip {
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  color: var(--color-text);
  border-radius: var(--radius-full);
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}
.feed-filter-chip:hover { background: var(--color-surface-high); }
.feed-filter-chip.active {
  background: var(--color-secondary-container);
  border-color: transparent;
  color: var(--color-on-secondary-container);
}

/* ── Pill «Новые события» ──────────────────────────────────── */
.feed-newpill-wrap {
  position: sticky;
  top: 8px;
  z-index: 20;
  height: 0;
  display: flex;
  justify-content: center;
  overflow: visible;
}
.feed-newpill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 13px;
  font-weight: 600;
  padding: 8px 16px;
  cursor: pointer;
  box-shadow: 0 8px 24px color-mix(in oklch, var(--color-primary) 35%, transparent);
  animation: pill-in 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}
.feed-newpill .material-symbols-outlined { font-size: 17px; }
@keyframes pill-in {
  from { transform: translateY(-8px); opacity: 0; }
}

.groove-layout {
  flex: 1;
  min-height: 0;
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 16px;
  align-items: stretch;
}
.groove-aside {
  display: flex;
  flex-direction: column;
  gap: 16px;
  overflow-y: auto;
  padding-bottom: 24px;
  scrollbar-width: thin;
  scrollbar-color: var(--color-outline-dim) transparent;
}
.groove-main {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  overflow-y: auto;
  padding-bottom: 24px;
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
/* Stagger: карточки въезжают каскадом (новые от сокета — тоже). */
.river-day-body :deep(.feed-card) {
  animation: card-in 0.4s cubic-bezier(0.34, 1.2, 0.64, 1) backwards;
  animation-delay: calc(var(--i, 0) * 55ms);
}
@keyframes card-in {
  from { opacity: 0; transform: translateY(10px) scale(0.985); }
}
@media (prefers-reduced-motion: reduce) {
  .river-day-body :deep(.feed-card) { animation: none; }
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
  /* Возвращаем admin-body к стандартному поведению: весь контент в потоке */
  .admin-body {
    display: block;
    overflow-y: auto;
  }
  .groove-layout {
    flex: unset;
    min-height: unset;
    grid-template-columns: 1fr;
    align-items: start;
  }
  .groove-aside {
    overflow-y: visible;
    padding-bottom: 0;
  }
  .groove-main {
    overflow-y: visible;
    padding-bottom: 0;
  }
  .groove-river { gap: 16px; }
}

.desktop-only { display: inline-flex; }
@media (max-width: 768px) {
  .desktop-only { display: none; }
  .groove-title { font-size: 22px; }
}

/* ── Мобильное FAB-меню (M3 Expressive FAB menu) ───────────── */
.gfab-backdrop {
  position: fixed;
  inset: 0;
  z-index: 148;
  background: color-mix(in oklch, var(--color-scrim, var(--color-text)) 24%, transparent);
}
.gfab-back-enter-active, .gfab-back-leave-active { transition: opacity 0.2s; }
.gfab-back-enter-from, .gfab-back-leave-to { opacity: 0; }
.gfab {
  position: fixed;
  right: 16px;
  bottom: calc(64px + 16px + env(safe-area-inset-bottom, 0px));
  z-index: 150;
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 12px;
}
.gfab-items { display: flex; flex-direction: column; align-items: flex-end; gap: 10px; }
.gfab-item {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  border: none;
  background: none;
  padding: 0;
  cursor: pointer;
  font: inherit;
  opacity: 0;
  transform: translateY(10px) scale(0.92);
  pointer-events: none;
  transition: opacity 0.22s, transform 0.22s cubic-bezier(0.34, 1.36, 0.64, 1);
}
.gfab.open .gfab-item { opacity: 1; transform: none; pointer-events: auto; }
.gfab.open .gfab-item:nth-last-child(2) { transition-delay: 0.05s; }
.gfab.open .gfab-item:nth-last-child(3) { transition-delay: 0.1s; }
.gfab-item-label {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  color: var(--color-text);
  border-radius: var(--radius-full);
  padding: 6px 12px;
  font-size: 13px;
  font-weight: 600;
  box-shadow: var(--shadow-sm, none);
}
.gfab-item-btn {
  width: 44px;
  height: 44px;
  border-radius: 16px;
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
  display: grid;
  place-items: center;
  box-shadow: 0 4px 12px color-mix(in oklch, var(--color-secondary) 28%, transparent);
}
.gfab-item-btn .material-symbols-outlined { font-size: 22px; }
.gfab-main {
  width: 56px;
  height: 56px;
  border: none;
  border-radius: 18px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid;
  place-items: center;
  cursor: pointer;
  box-shadow:
    0 6px 16px color-mix(in oklch, var(--color-primary) 38%, transparent),
    0 2px 6px color-mix(in oklch, var(--color-primary) 20%, transparent);
  transition: border-radius 0.26s cubic-bezier(0.34, 1.36, 0.64, 1),
              transform 0.12s, background 0.15s;
}
.gfab.open .gfab-main { border-radius: 50%; }
.gfab-main:active { transform: scale(0.94); }
.gfab-main .material-symbols-outlined { font-size: 24px; }
.gfab-main-emoji { font-size: 26px; line-height: 1; }
@media (min-width: 769px) {
  .gfab, .gfab-backdrop { display: none; }
}
</style>
