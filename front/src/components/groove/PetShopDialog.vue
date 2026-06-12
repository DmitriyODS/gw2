<template>
  <AppDialog
    :model-value="modelValue"
    title="Гардероб Грувика"
    subtitle="Аксессуары и облики за грувы — заработанные честным трудом"
    icon="storefront"
    tone="primary"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="shop-balance">
      <span class="shop-balance-chip"><GrooveCoin /> {{ pet?.beans ?? 0 }} грувов</span>
      <span v-if="groove.seasonalItem" class="shop-season-chip">
        {{ SHOP_ITEMS[groove.seasonalItem]?.emoji }} Сезон: {{ groove.seasonTitle }}
      </span>
    </div>

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

    <div v-if="tab === 'accessories'" class="shop-grid">
      <div
        v-for="item in items"
        :key="item.key"
        class="shop-item"
        :class="{ owned: item.owned, seasonal: item.seasonal }"
      >
        <span v-if="item.seasonal" class="shop-season-tag">сезонный</span>
        <span class="shop-emoji">{{ item.emoji }}</span>
        <span class="shop-title">{{ item.title }}</span>
        <button
          v-if="!item.owned"
          class="shop-buy"
          type="button"
          :disabled="(pet?.beans ?? 0) < item.price || buying"
          @click="buy(item)"
        ><GrooveCoin /> {{ item.price }}</button>
        <span v-else class="shop-owned-tag">
          <span class="material-symbols-outlined">check</span> куплено
        </span>
      </div>

      <div class="shop-item special" :class="{ owned: hasHelmet }">
        <span class="shop-emoji">⛑️</span>
        <span class="shop-title">Каска дедлайнщика</span>
        <span class="shop-owned-tag special">
          {{ hasHelmet ? 'Награда за рейд — ваша!' : 'Только за победу в рейде' }}
        </span>
      </div>
    </div>

    <div v-else class="shop-grid">
      <p class="species-hint">
        Виды-зверюшки — это альтернативный облик. Стадия и опыт не сбрасываются,
        переключаться между разблокированными можно когда угодно.
      </p>
      <div
        v-for="sp in speciesItems"
        :key="sp.key"
        class="shop-item species"
        :class="{ owned: sp.unlocked, active: sp.current }"
      >
        <span v-if="sp.current" class="shop-season-tag active">сейчас</span>
        <span class="shop-emoji">{{ sp.emoji }}</span>
        <span class="shop-title">{{ sp.title }}</span>
        <button
          v-if="sp.unlocked && !sp.current"
          class="shop-buy ghost"
          type="button"
          :disabled="switching"
          @click="pickSpecies(sp)"
        >Надеть</button>
        <button
          v-else-if="!sp.unlocked"
          class="shop-buy"
          type="button"
          :disabled="(pet?.beans ?? 0) < sp.price || switching"
          @click="buySpecies(sp)"
        ><GrooveCoin /> {{ sp.price }}</button>
        <span v-else class="shop-owned-tag">
          <span class="material-symbols-outlined">check</span> надет
        </span>
      </div>
    </div>
  </AppDialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import GrooveCoin from '@/components/groove/GrooveCoin.vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { PET_SPECIES, SHOP_ITEMS } from '@/utils/groove.js'

defineProps({
  modelValue: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])

const groove = useGrooveStore()
const notify = useNotificationsStore()
const buying = ref(false)
const switching = ref(false)
const tab = ref('accessories')

const pet = computed(() => groove.pet)
const hasHelmet = computed(() => (pet.value?.accessories || []).includes('helmet'))

const items = computed(() =>
  Object.entries(groove.shopPrices)
    .map(([key, price]) => ({
      key,
      price,
      emoji: SHOP_ITEMS[key]?.emoji || '🎁',
      title: SHOP_ITEMS[key]?.title || key,
      owned: (pet.value?.accessories || []).includes(key),
      seasonal: key === groove.seasonalItem,
    }))
    .sort((a, b) => a.price - b.price)
)

const speciesItems = computed(() => {
  const unlocked = new Set(pet.value?.unlocked_species || [])
  if (pet.value?.species) unlocked.add(pet.value.species)
  return Object.entries(groove.speciesPrices || {})
    .map(([key, price]) => ({
      key,
      price,
      emoji: PET_SPECIES[key]?.emoji || '🐾',
      title: PET_SPECIES[key]?.title || key,
      unlocked: unlocked.has(key),
      current: pet.value?.species === key,
    }))
    .sort((a, b) => a.price - b.price)
})

onMounted(() => {
  if (!Object.keys(groove.shopPrices).length) groove.fetchShop().catch(() => {})
})

async function buy(item) {
  buying.value = true
  try {
    await groove.buyItem(item.key)
    notify.success(`«${item.title}» куплен и сразу надет ${item.emoji}`)
  } catch (e) {
    notify.warn(e?.message || 'Покупка не удалась')
  } finally {
    buying.value = false
  }
}

async function buySpecies(sp) {
  switching.value = true
  try {
    await groove.buySpecies(sp.key)
    notify.success(`Грувик перевоплотился в облик «${sp.title}» ${sp.emoji}`)
  } catch (e) {
    notify.warn(e?.message || 'Не получилось разблокировать вид')
  } finally {
    switching.value = false
  }
}

async function pickSpecies(sp) {
  if (sp.current) return
  switching.value = true
  try {
    await groove.switchSpecies(sp.key)
    notify.success(`Облик сменён: ${sp.emoji} ${sp.title}`)
  } catch (e) {
    notify.warn(e?.message || 'Не удалось сменить облик')
  } finally {
    switching.value = false
  }
}
</script>

<style scoped>
.shop-balance { display: flex; justify-content: center; gap: 8px; flex-wrap: wrap; margin-bottom: 14px; }
.shop-balance-chip {
  background: color-mix(in oklch, var(--color-success) 18%, transparent);
  border-radius: var(--radius-full);
  padding: 6px 14px;
  font-size: 14px;
  font-weight: 700;
}

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
.species-hint {
  grid-column: 1 / -1;
  margin: 0 0 4px;
  font-size: 12.5px;
  color: var(--color-text-dim);
  line-height: 1.45;
  text-align: center;
}
.shop-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  border: 1px solid var(--color-outline-dim);
  border-radius: 14px;
  padding: 12px 8px;
  text-align: center;
  position: relative;
}
.shop-item.owned { background: var(--color-surface-high); }
.shop-item.seasonal { border-color: color-mix(in oklch, var(--color-tertiary) 55%, transparent); }
.shop-item.species.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
}
.shop-season-tag {
  position: absolute;
  top: -8px;
  right: 8px;
  font-size: 10px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: var(--radius-full);
  padding: 2px 8px;
}
.shop-season-tag.active {
  background: var(--color-primary);
  color: var(--color-on-primary);
}
.shop-season-chip {
  background: var(--color-tertiary-container);
  color: var(--color-on-tertiary-container);
  border-radius: var(--radius-full);
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 600;
}
.shop-item.special { border-style: dashed; }
.shop-emoji { font-size: 30px; line-height: 1; }
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
}
.shop-buy.ghost {
  background: var(--color-secondary-container);
  color: var(--color-on-secondary-container);
}
.shop-buy:disabled { opacity: 0.45; cursor: default; }
.shop-owned-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  font-size: 11.5px;
  color: var(--color-text-dim);
}
.shop-owned-tag .material-symbols-outlined { font-size: 14px; color: var(--color-success); }
.shop-owned-tag.special { line-height: 1.3; }
</style>
