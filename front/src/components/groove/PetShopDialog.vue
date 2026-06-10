<template>
  <AppDialog
    :model-value="modelValue"
    title="Гардероб Грувика"
    subtitle="Аксессуары за грувы — заработанные честным трудом"
    icon="storefront"
    tone="primary"
    size="md"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div class="shop-balance">
      <span class="shop-balance-chip">🫘 {{ pet?.beans ?? 0 }} грувов</span>
      <span v-if="groove.seasonalItem" class="shop-season-chip">
        {{ SHOP_ITEMS[groove.seasonalItem]?.emoji }} Сезон: {{ groove.seasonTitle }}
      </span>
    </div>

    <div class="shop-grid">
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
        >🫘 {{ item.price }}</button>
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
  </AppDialog>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { useGrooveStore } from '@/stores/groove.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { SHOP_ITEMS } from '@/utils/groove.js'

defineProps({
  modelValue: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])

const groove = useGrooveStore()
const notify = useNotificationsStore()
const buying = ref(false)

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
.shop-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(130px, 1fr));
  gap: 10px;
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
}
.shop-item.owned { background: var(--color-surface-high); }
.shop-item { position: relative; }
.shop-item.seasonal { border-color: color-mix(in oklch, var(--color-tertiary) 55%, transparent); }
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
