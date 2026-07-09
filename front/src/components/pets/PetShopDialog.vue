<template>
  <AppDialog
    :model-value="modelValue"
    title="Магазин Грувика"
    subtitle="Аксессуары и облики за кудосы — заработанные честным трудом"
    icon="storefront"
    tone="primary"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="shop-balance">
      <span class="shop-balance-chip"><KudosCoin /> {{ pet?.kudos ?? 0 }} кудосов</span>
    </div>

    <!-- Сюрприз дня: бесплатный взвешенный по редкости бонус раз в день. -->
    <button
      class="mystery-card"
      type="button"
      :disabled="mysteryClaiming || mysteryDone"
      @click="claimMystery"
    >
      <span class="mystery-emoji">🎁</span>
      <span class="mystery-text">
        <strong>{{ mysteryDone ? 'Сюрприз дня получен' : 'Сюрприз дня' }}</strong>
        <small>{{ mysteryDone ? 'Загляните завтра за новым' : 'Бесплатный бонус-предмет — раз в день' }}</small>
      </span>
      <span v-if="!mysteryDone" class="material-symbols-outlined mystery-arrow">chevron_right</span>
      <span v-else class="material-symbols-outlined mystery-check">check_circle</span>
    </button>

    <div class="shop-tabs">
      <button
        class="shop-tab"
        :class="{ active: tab === 'accessories' }"
        type="button"
        @click="tab = 'accessories'"
      >Аксессуары</button>
      <button
        class="shop-tab"
        :class="{ active: tab === 'species' }"
        type="button"
        @click="tab = 'species'"
      >Облики</button>
    </div>

    <div v-if="loading" class="shop-loading">
      <ProgressSpinner style="width:32px;height:32px" />
    </div>
    <div v-else class="shop-grid">
      <div
        v-for="item in visibleItems"
        :key="item.key"
        class="shop-item"
        :class="[`rarity-${item.rarity}`, { owned: item.owned, locked: item.unlock_kind === 'achievement', 'sold-out': item.sold_out }]"
        :style="rarityStyle(item)"
      >
        <span class="rarity-tag">{{ RARITY_TITLE[item.rarity] || item.rarity }}</span>
        <span v-if="item.limited_quota != null" class="limited-tag">
          {{ item.sold_out ? 'Распродано' : `Осталось: ${item.remaining}` }}
        </span>
        <span v-else-if="countdown(item)" class="rotation-tag">{{ countdown(item) }}</span>

        <span class="shop-emoji">{{ shopItemEmoji(item) }}</span>
        <span class="shop-title">{{ shopItemTitle(item) }}</span>

        <template v-if="item.unlock_kind === 'achievement'">
          <span class="shop-locked-tag">
            <span class="material-symbols-outlined">emoji_events</span> Достижение
          </span>
        </template>
        <template v-else-if="item.kind === 'species'">
          <button
            v-if="item.owned && pet?.species !== item.key"
            class="shop-buy ghost"
            type="button"
            :disabled="switching"
            @click="pickSpecies(item)"
          >Надеть</button>
          <button
            v-else-if="!item.owned"
            class="shop-buy"
            type="button"
            :disabled="!canAfford(item) || switching || item.sold_out"
            @click="buySpeciesItem(item)"
          ><KudosCoin /> {{ item.price_kudos }}</button>
          <span v-else class="shop-owned-tag">
            <span class="material-symbols-outlined">check</span> сейчас надет
          </span>
        </template>
        <template v-else>
          <button
            v-if="!item.owned"
            class="shop-buy"
            type="button"
            :disabled="!canAfford(item) || buying || item.sold_out"
            @click="buy(item)"
          ><KudosCoin /> {{ item.price_kudos }}</button>
          <span v-else class="shop-owned-tag">
            <span class="material-symbols-outlined">check</span> куплено
          </span>
        </template>
      </div>

      <p v-if="!visibleItems.length" class="shop-empty">
        Пока пусто в этой витрине — загляните позже
      </p>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import { getShop } from '@/api/pets.js'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { RARITY_TAG, RARITY_TITLE, shopItemEmoji, shopItemTitle } from '@/utils/pets.js'

defineProps({
  modelValue: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])

const pets = usePetsStore()
const notify = useNotificationsStore()
const buying = ref(false)
const switching = ref(false)
const mysteryClaiming = ref(false)
const mysteryDone = ref(false)
const loading = ref(false)
const tab = ref('accessories')

const pet = computed(() => pets.pet)

const visibleItems = computed(() => {
  const wantKind = tab.value === 'species' ? 'species' : null
  return (pets.shop || [])
    .filter((i) => (wantKind ? i.kind === wantKind : i.kind !== 'species'))
    .slice()
    .sort((a, b) => {
      const rarityOrder = { legendary: 0, epic: 1, rare: 2, common: 3 }
      const r = (rarityOrder[a.rarity] ?? 9) - (rarityOrder[b.rarity] ?? 9)
      return r !== 0 ? r : a.price_kudos - b.price_kudos
    })
})

function canAfford(item) {
  return (pet.value?.kudos ?? 0) >= item.price_kudos
}

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
  const ms = new Date(item.active_to).getTime() - Date.now()
  if (ms <= 0) return 'Скоро закончится'
  const hours = Math.floor(ms / 3_600_000)
  const days = Math.floor(hours / 24)
  if (days > 0) return `Осталось ${days} д ${hours % 24} ч`
  return `Осталось ${hours} ч`
}

onMounted(async () => {
  // mystery_taken живёт только в ответе GET /shop, а стор кладёт лишь items —
  // флаг берём прямым вызовом API (диалог открывается редко, запрос дешёвый).
  getShop()
    .then((res) => { mysteryDone.value = !!res.mystery_taken })
    .catch(() => {})
  if (!pets.shopLoaded) {
    loading.value = true
    try {
      await pets.fetchShop()
    } catch { /* витрина покажет «пусто», покупка всё равно перезапросит */ } finally {
      loading.value = false
    }
  }
})

async function buy(item) {
  buying.value = true
  try {
    await pets.buyItem(item.key)
    notify.success(`«${shopItemTitle(item)}» куплен и сразу надет ${shopItemEmoji(item)}`)
  } catch (e) {
    notify.warn(e?.message || 'Покупка не удалась')
  } finally {
    buying.value = false
  }
}

async function buySpeciesItem(item) {
  switching.value = true
  try {
    await pets.buySpecies(item.key)
    notify.success(`Грувик перевоплотился в облик «${shopItemTitle(item)}» ${shopItemEmoji(item)}`)
  } catch (e) {
    notify.warn(e?.message || 'Не получилось разблокировать вид')
  } finally {
    switching.value = false
  }
}

async function pickSpecies(item) {
  switching.value = true
  try {
    await pets.switchSpecies(item.key)
    notify.success(`Облик сменён: ${shopItemEmoji(item)} ${shopItemTitle(item)}`)
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
.shop-balance { display: flex; justify-content: center; gap: 8px; flex-wrap: wrap; margin-bottom: 12px; }
.shop-balance-chip {
  background: color-mix(in oklch, var(--color-success) 18%, transparent);
  border-radius: var(--radius-full);
  padding: 6px 14px;
  font-size: 14px;
  font-weight: 700;
}

.mystery-card {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  border: 1.5px dashed color-mix(in oklch, var(--color-tertiary) 55%, transparent);
  border-radius: 16px;
  background: color-mix(in oklch, var(--color-tertiary-container) 40%, transparent);
  padding: 12px 14px;
  margin-bottom: 14px;
  cursor: pointer;
  text-align: left;
  font: inherit;
  color: inherit;
  transition: transform 0.1s, opacity 0.15s;
}
.mystery-card:active:not(:disabled) { transform: scale(0.99); }
.mystery-card:disabled { opacity: 0.6; cursor: default; }
.mystery-emoji { font-size: 30px; line-height: 1; }
.mystery-text { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.mystery-text strong { font-size: 13.5px; }
.mystery-text small { font-size: 11.5px; color: var(--color-text-dim); }
.mystery-arrow { color: var(--color-tertiary); }
.mystery-check { color: var(--color-success); }

.shop-tabs {
  display: inline-flex;
  background: var(--color-surface-high);
  border-radius: var(--radius-full);
  padding: 4px;
  margin: 0 auto 14px;
  width: max-content;
  max-width: 100%;
}
.shop-tab {
  border: none;
  background: none;
  font-size: 13px;
  font-weight: 600;
  padding: 7px 16px;
  border-radius: var(--radius-full);
  cursor: pointer;
  color: var(--color-text-dim);
  transition: background 0.15s, color 0.15s;
}
.shop-tab.active {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.shop-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 10px;
}
.shop-loading {
  display: flex;
  justify-content: center;
  padding: 28px 0;
}
.shop-empty {
  grid-column: 1 / -1;
  margin: 0;
  text-align: center;
  font-size: 13px;
  color: var(--color-text-dim);
  padding: 20px 0;
}
.shop-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  border: 1.5px solid var(--rarity-border, var(--color-outline-dim));
  border-radius: 14px;
  padding: 14px 8px 12px;
  text-align: center;
  position: relative;
  background: color-mix(in oklch, var(--rarity-surface, transparent) 35%, var(--color-surface));
}
.shop-item.owned { background: var(--color-surface-high); }
.shop-item.locked { opacity: 0.75; }
.shop-item.sold-out { opacity: 0.55; }
.rarity-tag {
  position: absolute;
  top: -9px;
  left: 8px;
  font-size: 9.5px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--rarity-surface, var(--color-surface-high));
  border: 1px solid var(--rarity-border, var(--color-outline-dim));
  color: var(--color-text);
  border-radius: var(--radius-full);
  padding: 2px 8px;
}
.limited-tag, .rotation-tag {
  position: absolute;
  top: -9px;
  right: 8px;
  font-size: 9.5px;
  font-weight: 700;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: var(--radius-full);
  padding: 2px 8px;
  white-space: nowrap;
}
.shop-emoji { font-size: 30px; line-height: 1; margin-top: 4px; }
.shop-title { font-size: 12.5px; font-weight: 600; line-height: 1.25; }
.shop-buy {
  border: none;
  border-radius: var(--radius-full);
  background: var(--color-primary);
  color: var(--color-on-primary);
  font-size: 12.5px;
  font-weight: 700;
  padding: 6px 14px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.shop-buy.ghost {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.shop-buy:disabled { opacity: 0.45; cursor: default; }
.shop-owned-tag, .shop-locked-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11.5px;
  color: var(--color-text-dim);
}
.shop-owned-tag .material-symbols-outlined { font-size: 14px; color: var(--color-success); }
.shop-locked-tag .material-symbols-outlined { font-size: 14px; }
</style>
