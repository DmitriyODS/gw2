<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <div class="ps-toolbar">
        <button class="btn-glass ps-back" type="button" title="К грувикам" @click="router.push('/pets')">
          <span class="material-symbols-outlined">arrow_back</span>
        </button>
        <h1 class="ps-title">Магазин</h1>
        <button
          class="chip-tint chip-tint--success ps-balance"
          type="button"
          title="Открыть кудо-банк"
          @click="router.push('/pets/bank')"
        >
          <KudosCoin /> <strong>{{ pet?.kudos ?? 0 }}</strong>
        </button>
        <span class="chip-tint ps-refresh" title="Скидка дня и витрина меняются в полночь по Москве">
          <span class="material-symbols-outlined">schedule</span>
          обновится через {{ midnightLeft }}
        </span>
      </div>
    </header>

    <div class="admin-body">
      <!-- Сюрприз дня -->
      <button
        class="ps-mystery"
        type="button"
        :disabled="mysteryClaiming || mysteryDone"
        @click="claimMystery"
      >
        <span class="ps-mystery-emoji" :class="{ shake: !mysteryDone }">🎁</span>
        <span class="ps-mystery-text">
          <strong>{{ mysteryDone ? 'Сюрприз дня получен' : 'Сюрприз дня ждёт!' }}</strong>
          <small>{{ mysteryDone ? 'Загляните завтра за новым' : 'Бесплатный бонус-предмет — раз в день, чем реже, тем приятнее' }}</small>
        </span>
        <span v-if="!mysteryDone" class="material-symbols-outlined ps-mystery-arrow">redeem</span>
        <span v-else class="material-symbols-outlined ps-mystery-check">check_circle</span>
      </button>

      <!-- Скидка дня: главная сцена витрины -->
      <section v-if="saleItems.length" class="ps-featured">
        <h3 class="ps-section-title">
          <span class="material-symbols-outlined">local_fire_department</span>
          Скидка дня
          <span class="ps-timer">до полуночи · {{ midnightLeft }}</span>
        </h3>
        <div class="ps-featured-grid">
          <button
            v-for="item in saleItems"
            :key="'sale-' + item.key"
            class="ps-hero-card"
            type="button"
            :style="rarityStyle(item)"
            @click="openItem(item)"
          >
            <span class="ps-hero-emoji"><EmojiGlyph :char="shopItemEmoji(item)" /></span>
            <span class="ps-hero-info">
              <span class="ps-hero-title">{{ shopItemTitle(item) }}</span>
              <span class="ps-hero-rarity">{{ RARITY_TITLE[item.rarity] || item.rarity }}</span>
            </span>
            <span class="ps-hero-price">
              <span class="ps-sale-badge">−{{ item.discount_pct }}%</span>
              <s class="ps-old-price">{{ item.price_kudos }}</s>
              <strong><KudosCoin /> {{ effectivePrice(item) }}</strong>
            </span>
          </button>
        </div>
      </section>

      <!-- Скоро уйдёт: ротационные с таймером -->
      <section v-if="leavingItems.length" class="ps-leaving">
        <h3 class="ps-section-title">
          <span class="material-symbols-outlined">hourglass_bottom</span>
          Скоро уйдёт из продажи
        </h3>
        <div class="ps-leaving-row">
          <button
            v-for="item in leavingItems"
            :key="'rot-' + item.key"
            class="ps-leaving-card"
            type="button"
            :style="rarityStyle(item)"
            @click="openItem(item)"
          >
            <span class="ps-leaving-emoji"><EmojiGlyph :char="shopItemEmoji(item)" /></span>
            <span class="ps-leaving-title">{{ shopItemTitle(item) }}</span>
            <span class="ps-leaving-timer">{{ countdown(item) }}</span>
          </button>
        </div>
      </section>

      <!-- Фильтры: одна акриловая панель, ряды с подписями -->
      <div class="ps-filters">
        <div class="ps-filter-row">
          <span class="ps-filter-label">Витрина</span>
          <div class="ps-filter-chips">
            <button
              v-for="c in CATEGORIES"
              :key="c.value"
              class="ps-cat"
              :class="{ active: category === c.value }"
              type="button"
              @click="category = c.value"
            >{{ c.label }}</button>
          </div>
        </div>
        <div class="ps-filter-divider"></div>
        <div class="ps-filter-row">
          <span class="ps-filter-label">Редкость</span>
          <div class="ps-filter-chips">
            <button
              v-for="r in RARITIES"
              :key="r"
              class="ps-rarity-chip"
              :class="{ active: rarityFilter === r }"
              :style="rarityStyle({ rarity: r })"
              type="button"
              @click="rarityFilter = rarityFilter === r ? null : r"
            >{{ RARITY_TITLE[r] }}</button>
            <button
              class="ps-cat ps-afford"
              :class="{ active: affordOnly }"
              type="button"
              @click="affordOnly = !affordOnly"
            ><KudosCoin /> По карману</button>
          </div>
        </div>
      </div>

      <button
        v-if="category === 'species' && boughtSpeciesOn"
        class="btn-glass ps-reset"
        type="button"
        :disabled="switching"
        @click="resetSpeciesToNatural"
      >
        <span class="material-symbols-outlined">restart_alt</span>
        Вернуть природный облик
      </button>

      <!-- Витрина -->
      <div v-if="loading" class="ps-loading"><BrandLoader :size="64" /></div>
      <div v-else class="ps-grid">
        <button
          v-for="(item, idx) in visibleItems"
          :key="item.key"
          class="ps-item"
          :class="{ owned: item.owned, locked: item.unlock_kind === 'achievement', 'sold-out': item.sold_out }"
          :style="{ ...rarityStyle(item), '--i': Math.min(idx, 14) }"
          type="button"
          @click="openItem(item)"
        >
          <span class="ps-rarity-tag">{{ RARITY_TITLE[item.rarity] || item.rarity }}</span>
          <span v-if="item.discount_pct" class="ps-sale-tag">−{{ item.discount_pct }}%</span>
          <span v-else-if="item.limited_quota != null" class="ps-limited-tag">
            {{ item.sold_out ? 'Распродано' : `Осталось ${item.remaining}` }}
          </span>

          <span class="ps-item-emoji"><EmojiGlyph :char="shopItemEmoji(item)" /></span>
          <span class="ps-item-title">{{ shopItemTitle(item) }}</span>

          <span v-if="item.unlock_kind === 'achievement'" class="ps-item-note">
            <span class="material-symbols-outlined">emoji_events</span> Достижение
          </span>
          <span v-else-if="item.owned && item.kind === 'species' && pet?.species === item.key" class="ps-item-note owned-note">
            <span class="material-symbols-outlined">check</span> сейчас надет
          </span>
          <span v-else-if="item.owned" class="ps-item-note owned-note">
            <span class="material-symbols-outlined">check</span> {{ item.kind === 'species' ? 'куплен' : 'куплено' }}
          </span>
          <span v-else-if="item.sold_out" class="ps-item-note">Распродано</span>
          <span v-else-if="!canAfford(item)" class="ps-item-note lack">
            не хватает {{ effectivePrice(item) - (pet?.kudos ?? 0) }} <KudosCoin />
          </span>
          <span v-else class="ps-item-price">
            <s v-if="item.sale_price_kudos" class="ps-old-price">{{ item.price_kudos }}</s>
            <KudosCoin /> {{ effectivePrice(item) }}
          </span>
        </button>

        <p v-if="!visibleItems.length" class="ps-empty">Пока пусто в этой витрине — загляните позже</p>
      </div>

      <p class="ps-collection">
        <span class="material-symbols-outlined">collections_bookmark</span>
        Коллекция: собрано {{ ownedCount }} из {{ totalCount }}
      </p>
    </div>

    <!-- Карточка товара: примерка на своём грувике + покупка -->
    <AppDialog
      :model-value="!!selected"
      :title="selected ? shopItemTitle(selected) : ''"
      :subtitle="selected ? RARITY_TITLE[selected.rarity] || '' : ''"
      icon="storefront"
      tone="primary"
      size="sm"
      mobile="sheet"
      @update:model-value="(v) => { if (!v) selected = null }"
    >
      <div v-if="selected" class="ps-tryon" :style="rarityStyle(selected)">
        <!-- Примерка: мой грувик в этом облике/с этим аксессуаром -->
        <div class="ps-tryon-figure">
          <span class="ps-tryon-pet"><EmojiGlyph :char="tryOnPetEmoji" /></span>
          <span v-if="tryOnHat" class="ps-tryon-hat"><EmojiGlyph :char="tryOnHat" /></span>
        </div>
        <p class="ps-tryon-hint">
          {{ selected.kind === 'species' ? `Так будет выглядеть «${pet?.name || 'Грувик'}»` : `«${pet?.name || 'Грувик'}» примеряет обновку` }}
        </p>

        <div class="ps-tryon-meta">
          <span v-if="selected.discount_pct" class="ps-sale-badge">−{{ selected.discount_pct }}% до полуночи</span>
          <span v-if="selected.limited_quota != null && !selected.sold_out" class="ps-limited-tag static">
            Тираж: осталось {{ selected.remaining }}
          </span>
          <span v-if="countdown(selected)" class="ps-limited-tag static">{{ countdown(selected) }}</span>
        </div>

        <template v-if="selected.unlock_kind === 'achievement'">
          <p class="ps-tryon-note">Не продаётся — выдаётся за достижение.</p>
        </template>
        <template v-else-if="selected.owned">
          <button
            v-if="selected.kind === 'species' && pet?.species !== selected.key"
            class="btn-grad ps-tryon-buy"
            :disabled="switching"
            @click="pickSpecies(selected)"
          >Надеть облик</button>
          <p v-else class="ps-tryon-note">
            <span class="material-symbols-outlined">check_circle</span>
            {{ selected.kind === 'species' ? 'Облик сейчас надет' : 'Уже в вашей коллекции' }}
          </p>
          <!-- Продажа за полцены (текущий облик продать нельзя — снимите сначала). -->
          <button
            v-if="canSell(selected)"
            class="btn-glass ps-tryon-installment"
            :disabled="selling"
            @click="sell(selected)"
          >
            <span class="material-symbols-outlined">sell</span>
            Продать за {{ Math.floor(selected.price_kudos / 2) }}
          </button>
        </template>
        <template v-else>
          <button
            class="btn-grad ps-tryon-buy"
            :disabled="buying || switching || selected.sold_out || !canAfford(selected)"
            @click="selected.kind === 'species' ? buySpeciesItem(selected) : buy(selected)"
          >
            <template v-if="selected.sold_out">Распродано</template>
            <template v-else-if="!canAfford(selected)">Не хватает {{ effectivePrice(selected) - (pet?.kudos ?? 0) }}</template>
            <template v-else>
              Купить за
              <s v-if="selected.sale_price_kudos" class="ps-old-price">{{ selected.price_kudos }}</s>
              {{ effectivePrice(selected) }}
            </template>
            <KudosCoin />
          </button>
          <button
            v-if="canInstallment(selected)"
            class="btn-glass ps-tryon-installment"
            :disabled="buying || switching"
            @click="buyInstallment(selected)"
          >
            <span class="material-symbols-outlined">splitscreen</span>
            Оплатить частями ({{ Math.ceil(effectivePrice(selected) / 4) }} × 4)
          </button>
          <p v-if="!canAfford(selected) && !selected.sold_out && !canInstallment(selected)" class="ps-tryon-note">
            Кудосы приносят юниты, задачи и квесты — или загляните в копилки банка.
          </p>
        </template>
      </div>
    </AppDialog>

    <ConfettiBurst ref="confettiEl" />
  </div>
</template>

<script setup>
import { computed, onMounted, onBeforeUnmount, ref } from 'vue'
import { useRouter } from 'vue-router'
import BrandLoader from '@/components/common/BrandLoader.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import ConfettiBurst from '@/components/pets/bank/ConfettiBurst.vue'
import { getShop } from '@/api/pets.js'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import {
  NATURAL_SPECIES, PET_SPECIES, RARITY_TAG, RARITY_TITLE,
  petEmoji, shopItemEmoji, shopItemTitle,
} from '@/utils/pets.js'

const CATEGORIES = [
  { value: 'all', label: 'Все' },
  { value: 'accessories', label: 'Аксессуары' },
  { value: 'species', label: 'Облики' },
]
const RARITIES = ['common', 'rare', 'epic', 'legendary']

const router = useRouter()
const pets = usePetsStore()
const notify = useNotificationsStore()

const loading = ref(false)
const buying = ref(false)
const switching = ref(false)
const selling = ref(false)
const mysteryClaiming = ref(false)
const mysteryDone = ref(false)
const category = ref('all')
const rarityFilter = ref(null)
const affordOnly = ref(false)
const selected = ref(null)
const confettiEl = ref(null)

const pet = computed(() => pets.pet)

// ── Таймер до полуночи МСК (скидка дня и детерминированная витрина). ──
const now = ref(Date.now())
let tick = null
onMounted(() => { tick = setInterval(() => { now.value = Date.now() }, 30_000) })
onBeforeUnmount(() => clearInterval(tick))

const midnightLeft = computed(() => {
  const msk = new Date(new Date(now.value).toLocaleString('en-US', { timeZone: 'Europe/Moscow' }))
  const mins = (24 * 60) - (msk.getHours() * 60 + msk.getMinutes())
  const h = Math.floor(mins / 60)
  const m = mins % 60
  return h > 0 ? `${h} ч ${String(m).padStart(2, '0')} мин` : `${m} мин`
})

// ── Выборки витрины ──
const allItems = computed(() => pets.shop || [])

const saleItems = computed(() => allItems.value.filter((i) => i.discount_pct && !i.owned && !i.sold_out))

const leavingItems = computed(() => allItems.value
  .filter((i) => i.active_to && !i.owned && !i.sold_out)
  .sort((a, b) => new Date(a.active_to) - new Date(b.active_to)))

const visibleItems = computed(() => {
  const rarityOrder = { legendary: 0, epic: 1, rare: 2, common: 3 }
  return allItems.value
    .filter((i) => {
      if (category.value === 'species' && i.kind !== 'species') return false
      if (category.value === 'accessories' && i.kind === 'species') return false
      if (rarityFilter.value && i.rarity !== rarityFilter.value) return false
      if (affordOnly.value && !i.owned && !canAfford(i)) return false
      return true
    })
    .slice()
    .sort((a, b) => {
      const r = (rarityOrder[a.rarity] ?? 9) - (rarityOrder[b.rarity] ?? 9)
      return r !== 0 ? r : a.price_kudos - b.price_kudos
    })
})

const ownedCount = computed(() => allItems.value.filter((i) => i.owned).length)
const totalCount = computed(() => allItems.value.length)

// ── Примерка ──
const tryOnPetEmoji = computed(() => {
  const item = selected.value
  if (!item) return '🦊'
  if (item.kind === 'species') return PET_SPECIES[item.key]?.emoji || shopItemEmoji(item)
  return petEmoji(pet.value)
})
const tryOnHat = computed(() =>
  (selected.value && selected.value.kind !== 'species') ? shopItemEmoji(selected.value) : null)

function openItem(item) {
  selected.value = item
}

// Скидка дня: фактическая цена приходит с бэка (sale_price_kudos) и по ней
// же спишет покупка — фронт ничего не пересчитывает.
const effectivePrice = (item) => item.sale_price_kudos ?? item.price_kudos
const canAfford = (item) => (pet.value?.kudos ?? 0) >= effectivePrice(item)

function rarityStyle(item) {
  const tag = RARITY_TAG[item.rarity] || 'teal'
  return {
    '--rarity-border': `var(--tag-${tag}-border)`,
    '--rarity-accent': `var(--tag-${tag}-accent)`,
    '--rarity-surface': `var(--tag-${tag}-surface)`,
  }
}

function countdown(item) {
  if (!item.active_to) return null
  const ms = new Date(item.active_to).getTime() - now.value
  if (ms <= 0) return 'Скоро закончится'
  const hours = Math.floor(ms / 3_600_000)
  const days = Math.floor(hours / 24)
  if (days > 0) return `Осталось ${days} д ${hours % 24} ч`
  return `Осталось ${hours} ч`
}

// Праздник по редкости: чем реже — тем жирнее залп.
const CONFETTI_BY_RARITY = { common: 14, rare: 22, epic: 32, legendary: 48 }
function celebrate(item) {
  confettiEl.value?.burst(CONFETTI_BY_RARITY[item.rarity] || 18)
}

onMounted(async () => {
  // mystery_taken живёт только в ответе GET /shop, а стор кладёт лишь items.
  getShop()
    .then((res) => { mysteryDone.value = !!res.mystery_taken })
    .catch(() => {})
  if (!pets.pet) pets.fetchPet().catch(() => {})
  if (!pets.shopLoaded) {
    loading.value = true
    try {
      await pets.fetchShop()
    } catch { /* витрина покажет «пусто» */ } finally {
      loading.value = false
    }
  }
})

async function buy(item, installment = false) {
  buying.value = true
  try {
    await pets.buyItem(item.key, installment)
    celebrate(item)
    notify.success(installment
      ? `«${shopItemTitle(item)}» ваш — оплата долями в разделе «Банк» ${shopItemEmoji(item)}`
      : `«${shopItemTitle(item)}» куплен и сразу надет ${shopItemEmoji(item)}`)
    selected.value = null
  } catch (e) {
    notify.warn(e?.message || 'Покупка не удалась')
  } finally {
    buying.value = false
  }
}

async function buySpeciesItem(item, installment = false) {
  switching.value = true
  try {
    await pets.buySpecies(item.key, installment)
    celebrate(item)
    notify.success(installment
      ? `Облик «${shopItemTitle(item)}» разблокирован — оплата долями в разделе «Банк» ${shopItemEmoji(item)}`
      : `Грувик перевоплотился в облик «${shopItemTitle(item)}» ${shopItemEmoji(item)}`)
    selected.value = null
  } catch (e) {
    notify.warn(e?.message || 'Не получилось разблокировать вид')
  } finally {
    switching.value = false
  }
}

// Рассрочка доступна только на не-акционные товары (акционные — сразу/в кредит).
const canInstallment = (item) => item && !item.sold_out && !item.discount_pct
function buyInstallment(item) {
  return item.kind === 'species' ? buySpeciesItem(item, true) : buy(item, true)
}

// Продать можно купленное (не достижение); текущий облик — только сняв его.
const canSell = (item) =>
  item && item.owned && item.unlock_kind !== 'achievement' &&
  item.price_kudos > 0 && !(item.kind === 'species' && pet.value?.species === item.key)

async function sell(item) {
  selling.value = true
  try {
    await pets.sellItem(item.key)
    notify.success(`«${shopItemTitle(item)}» продан за ${Math.floor(item.price_kudos / 2)} кудосов`)
    selected.value = null
  } catch (e) {
    notify.warn(e?.message || 'Продать не удалось')
  } finally {
    selling.value = false
  }
}

const boughtSpeciesOn = computed(() => {
  const s = pet.value?.species
  return !!s && s !== 'egg' && !NATURAL_SPECIES.has(s)
})

async function resetSpeciesToNatural() {
  switching.value = true
  try {
    await pets.resetSpecies()
    notify.success('Грувик вернулся к природному облику')
  } catch (e) {
    notify.warn(e?.message || 'Не удалось сбросить облик')
  } finally {
    switching.value = false
  }
}

async function pickSpecies(item) {
  switching.value = true
  try {
    await pets.switchSpecies(item.key)
    notify.success(`Облик сменён: ${shopItemEmoji(item)} ${shopItemTitle(item)}`)
    selected.value = null
  } catch (e) {
    notify.warn(e?.message || 'Не удалось сменить облик')
  } finally {
    switching.value = false
  }
}

async function claimMystery() {
  mysteryClaiming.value = true
  try {
    const item = await pets.claimMystery()
    mysteryDone.value = true
    confettiEl.value?.burst(24)
    notify.success(`Сюрприз дня: «${shopItemTitle(item)}» ${shopItemEmoji(item)}`)
  } catch (e) {
    if (e?.error === 'ALREADY_TAKEN') mysteryDone.value = true
    notify.warn(e?.message || 'Сюрприз недоступен')
  } finally {
    mysteryClaiming.value = false
  }
}
</script>

<style scoped>
.admin-sticky { background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; }
.admin-sticky::after { display: none; }

.ps-toolbar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.ps-back { display: inline-flex; align-items: center; padding: 8px 10px; }
.ps-title { margin: 0; font-size: 20px; font-weight: 800; }
.ps-balance { border: none; font: inherit; cursor: pointer; display: inline-flex; align-items: center; gap: 5px; }
.ps-refresh { margin-left: auto; display: inline-flex; align-items: center; gap: 5px; font-size: 12px; }
.ps-refresh .material-symbols-outlined { font-size: 15px; }

.ps-section-title {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 800;
  display: flex;
  align-items: center;
  gap: 7px;
}
.ps-section-title .material-symbols-outlined { font-size: 19px; color: var(--color-primary); }
.ps-timer { font-size: 11.5px; font-weight: 600; color: var(--color-text-dim); margin-left: auto; }

/* ── Сюрприз дня ── */
.ps-mystery {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  border: 1.5px dashed color-mix(in oklch, var(--color-tertiary) 55%, transparent);
  border-radius: var(--radius-lg, 18px);
  background: color-mix(in oklch, var(--color-tertiary-container) 40%, transparent);
  padding: 14px 16px;
  margin-bottom: 20px;
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: inherit;
  transition: transform 0.1s, opacity 0.15s;
}
.ps-mystery:active:not(:disabled) { transform: scale(0.995); }
.ps-mystery:disabled { opacity: 0.65; cursor: default; }
.ps-mystery-emoji { font-size: 34px; line-height: 1; }
.ps-mystery-emoji.shake { animation: ps-gift-shake 2.6s ease-in-out infinite; }
@keyframes ps-gift-shake {
  0%, 86%, 100% { transform: rotate(0); }
  88% { transform: rotate(-8deg); }
  91% { transform: rotate(8deg); }
  94% { transform: rotate(-5deg); }
  97% { transform: rotate(4deg); }
}
@media (prefers-reduced-motion: reduce) { .ps-mystery-emoji.shake { animation: none; } }
.ps-mystery-text { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.ps-mystery-text strong { font-size: 14px; }
.ps-mystery-text small { font-size: 12px; color: var(--color-text-dim); }
.ps-mystery-arrow { color: var(--color-tertiary); font-size: 26px; }
.ps-mystery-check { color: var(--color-success); font-size: 24px; }

/* ── Featured: скидка дня ── */
.ps-featured { margin-bottom: 20px; }
.ps-featured-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(250px, 1fr));
  gap: 12px;
}
.ps-hero-card {
  display: flex;
  align-items: center;
  gap: 14px;
  border: 1.5px solid var(--rarity-border, var(--color-outline-dim));
  border-radius: var(--radius-lg, 18px);
  background:
    radial-gradient(120% 150% at 100% 0%,
      color-mix(in oklch, var(--rarity-surface) 60%, transparent) 0%, transparent 65%),
    var(--acrylic-card-bg);
  padding: 14px 16px;
  cursor: pointer;
  font: inherit;
  color: var(--color-text);
  text-align: left;
  transition: transform 0.12s;
}
.ps-hero-card:hover { background: var(--glass-hover-bg); }
.ps-hero-card:hover .ps-hero-emoji { animation: ps-wiggle 0.5s ease-in-out; }
.ps-hero-emoji { font-size: 42px; line-height: 1; animation: ps-float 4.5s ease-in-out infinite; }
.ps-hero-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.ps-hero-title { font-size: 14.5px; font-weight: 800; }
.ps-hero-rarity { font-size: 11.5px; color: var(--color-text-dim); text-transform: uppercase; letter-spacing: 0.05em; }
.ps-hero-price { display: flex; flex-direction: column; align-items: flex-end; gap: 3px; }
.ps-hero-price strong { display: inline-flex; align-items: center; gap: 4px; font-size: 16px; }

.ps-sale-badge {
  font-size: 11px; font-weight: 800;
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-radius: var(--radius-full);
  padding: 2px 9px;
}
.ps-old-price { opacity: 0.6; font-weight: 500; }

/* ── Скоро уйдёт ── */
.ps-leaving { margin-bottom: 20px; }
.ps-leaving-row {
  display: flex;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 6px;
}
.ps-leaving-card {
  flex: 0 0 130px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  border: 1.5px solid var(--rarity-border, var(--color-outline-dim));
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--rarity-surface, transparent) 30%, var(--color-surface));
  padding: 12px 8px;
  cursor: pointer;
  font: inherit;
  color: var(--color-text);
}
.ps-leaving-emoji { font-size: 26px; line-height: 1; }
.ps-leaving-title { font-size: 11.5px; font-weight: 700; text-align: center; }
.ps-leaving-timer { font-size: 10px; color: var(--color-error); font-weight: 700; animation: ps-blink 2.4s ease-in-out infinite; }
@keyframes ps-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.55; }
}
.ps-leaving-card:hover .ps-leaving-emoji { animation: ps-wiggle 0.5s ease-in-out; }
.ps-leaving-emoji { display: inline-block; }

/* ── Фильтры: единая акриловая панель, ряды с подписями и разделителем ── */
.ps-filters {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 6px 0 22px;
  padding: 14px 16px;
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 18px);
}
.ps-filter-row { display: flex; align-items: center; gap: 14px; }
.ps-filter-label {
  flex-shrink: 0;
  width: 72px;
  font-size: 11.5px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--color-text-dim);
}
.ps-filter-chips { display: flex; gap: 10px; flex-wrap: wrap; align-items: center; }
.ps-filter-divider { height: 1px; background: var(--color-outline-dim); }
.ps-cat {
  border: 1px solid var(--color-outline-dim);
  background: none;
  color: var(--color-text-dim);
  border-radius: var(--radius-full);
  font: inherit; font-size: 13px; font-weight: 600;
  padding: 7px 16px;
  cursor: pointer;
}
.ps-cat.active {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.ps-afford { display: inline-flex; align-items: center; gap: 5px; }
.ps-afford.active {
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-color: var(--color-primary);
}
.ps-rarity-chip {
  border: 1.5px solid var(--rarity-border);
  background: color-mix(in oklch, var(--rarity-surface) 40%, transparent);
  color: var(--color-text);
  border-radius: var(--radius-full);
  font: inherit; font-size: 12px; font-weight: 700;
  padding: 7px 14px;
  cursor: pointer;
  opacity: 0.6;
  transition: opacity 0.12s, box-shadow 0.12s;
}
.ps-rarity-chip:hover { opacity: 0.85; }
.ps-rarity-chip.active { opacity: 1; box-shadow: 0 0 0 2px var(--rarity-border); }

.ps-reset { display: inline-flex; align-items: center; gap: 6px; margin-bottom: 12px; }

/* ── Сетка ── */
.ps-loading { display: flex; justify-content: center; padding: 40px 0; }
.ps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
}
.ps-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  border: 1.5px solid var(--rarity-border, var(--color-outline-dim));
  border-radius: var(--radius-md);
  padding: 18px 10px 14px;
  text-align: center;
  position: relative;
  background: color-mix(in oklch, var(--rarity-surface, transparent) 35%, var(--color-surface));
  cursor: pointer;
  font: inherit;
  color: var(--color-text);
  transition: transform 0.15s, box-shadow 0.15s;
  /* Стаггер-появление витрины: каждая следующая карточка чуть позже. */
  animation: ps-pop-in 0.4s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--i, 0) * 35ms);
}
.ps-item:hover {
  box-shadow: 0 10px 26px color-mix(in oklch, var(--rarity-accent, var(--color-text)) 22%, transparent);
}
.ps-item:hover .ps-item-emoji { animation: ps-wiggle 0.5s ease-in-out; }
.ps-item:active { transform: translateY(-1px) scale(0.99); }

@keyframes ps-pop-in {
  from { opacity: 0; transform: translateY(14px) scale(0.96); }
  to { opacity: 1; transform: none; }
}
@keyframes ps-wiggle {
  0%, 100% { transform: rotate(0) scale(1); }
  30% { transform: rotate(-8deg) scale(1.15); }
  65% { transform: rotate(7deg) scale(1.1); }
}
@keyframes ps-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-5px); }
}
@media (prefers-reduced-motion: reduce) {
  .ps-item, .ps-hero-emoji, .ps-item-emoji, .ps-tryon-pet { animation: none !important; }
}
.ps-item.owned { background: var(--color-surface-high); }
.ps-item.locked { opacity: 0.75; }
.ps-item.sold-out { opacity: 0.55; }
.ps-rarity-tag {
  position: absolute;
  top: -9px; left: 10px;
  font-size: 9.5px; font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--rarity-surface, var(--color-surface-high));
  border: 1px solid var(--rarity-border, var(--color-outline-dim));
  color: var(--color-text);
  border-radius: var(--radius-full);
  padding: 2px 8px;
}
.ps-sale-tag, .ps-limited-tag {
  position: absolute;
  top: -9px; right: 10px;
  font-size: 9.5px; font-weight: 800;
  border-radius: var(--radius-full);
  padding: 2px 8px;
  white-space: nowrap;
}
.ps-sale-tag { background: var(--color-error-container); color: var(--color-on-error-container); }
.ps-limited-tag { background: var(--color-tertiary-container); color: var(--color-on-tertiary-container); }
.ps-limited-tag.static { position: static; }
.ps-item-emoji { font-size: 34px; line-height: 1; margin-top: 2px; display: inline-block; }
.ps-item-title { font-size: 12.5px; font-weight: 700; line-height: 1.25; }
.ps-item-price {
  display: inline-flex; align-items: center; gap: 4px;
  font-size: 13px; font-weight: 800;
  color: var(--color-primary);
}
.ps-item-note {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 11px; color: var(--color-text-dim);
}
.ps-item-note.lack { color: var(--color-error); font-weight: 600; }
.ps-item-note.owned-note .material-symbols-outlined { color: var(--color-success); }
.ps-item-note .material-symbols-outlined { font-size: 14px; }
.ps-empty {
  grid-column: 1 / -1;
  margin: 0; text-align: center;
  font-size: 13px; color: var(--color-text-dim);
  padding: 24px 0;
}

.ps-collection {
  margin: 18px 0 0;
  display: flex; align-items: center; gap: 6px;
  font-size: 12.5px; color: var(--color-text-dim);
}
.ps-collection .material-symbols-outlined { font-size: 17px; }

/* ── Примерка ── */
/* Нижний отступ крупнее: без футера кнопка покупки не должна липнуть к краю. */
.ps-tryon { display: flex; flex-direction: column; align-items: center; gap: 10px; padding: 4px 0 22px; }
.ps-tryon-figure {
  position: relative;
  width: 130px; height: 130px;
  border-radius: 50%;
  background:
    radial-gradient(100% 100% at 50% 100%,
      color-mix(in oklch, var(--rarity-surface) 70%, transparent) 0%, transparent 75%),
    var(--color-surface-high);
  border: 1.5px solid var(--rarity-border);
  display: grid; place-items: center;
}
.ps-tryon-pet { font-size: 64px; line-height: 1; display: inline-block; animation: ps-float 3.5s ease-in-out infinite; }
.ps-tryon-hat {
  position: absolute;
  top: 2px; right: 12px;
  font-size: 34px;
  transform: rotate(14deg);
}
.ps-tryon-hint { margin: 0; font-size: 12.5px; color: var(--color-text-dim); }
.ps-tryon-meta { display: flex; gap: 8px; flex-wrap: wrap; justify-content: center; }
.ps-tryon-buy {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 14px;
  padding: 12px 26px;
}
.ps-tryon-installment {
  display: inline-flex; align-items: center; gap: 6px;
  font-size: 13px; padding: 9px 20px; margin-top: 8px;
}
.ps-tryon-installment .material-symbols-outlined { font-size: 18px; }
.ps-tryon-note {
  margin: 0;
  display: inline-flex; align-items: center; gap: 5px;
  font-size: 12.5px; color: var(--color-text-dim);
  text-align: center;
}
.ps-tryon-note .material-symbols-outlined { font-size: 16px; color: var(--color-success); }

@media (max-width: 560px) {
  .ps-grid { grid-template-columns: repeat(2, 1fr); }
  .ps-refresh { margin-left: 0; }
  .ps-featured-grid { grid-template-columns: 1fr; }
  /* На узком экране подпись ряда — над чипами, не сбоку. */
  .ps-filter-row { flex-direction: column; align-items: flex-start; gap: 8px; }
  .ps-filter-label { width: auto; }
}
</style>
