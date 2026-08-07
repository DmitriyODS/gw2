<template>
  <!-- Карточка предложения витрины: тариф, докупка или товар. Цена месяца —
       светлой плашкой, годовая (выгоднее) — акцентной, со сноской. -->
  <button class="offer" type="button" @click="$emit('open')">
    <h3 class="offer-title">{{ title }}</h3>
    <p v-if="description" class="offer-desc">{{ description }}</p>

    <div class="offer-prices">
      <span v-if="owned" class="price price--owned">Уже куплено</span>
      <template v-else>
        <span v-if="showMonth" class="price price--plain">{{ formatPrice(priceMonth) }}{{ perMonthSuffix }}</span>
        <span v-if="priceYear > 0" class="price price--accent">{{ formatPerMonth(priceYear) }}*</span>
      </template>
    </div>
    <p v-if="priceYear > 0 && !owned" class="offer-note">*при оплате за год</p>
  </button>
</template>

<script setup>
import { computed } from 'vue'
import { formatPerMonth, formatPrice } from '@/utils/money.js'

const props = defineProps({
  title: { type: String, required: true },
  description: { type: String, default: '' },
  priceMonth: { type: Number, default: 0 },
  priceYear: { type: Number, default: 0 },
  // recurring=false — разовая покупка (пачка токенов): «руб.» без «/мес».
  recurring: { type: Boolean, default: true },
  owned: { type: Boolean, default: false },
})

defineEmits(['open'])

const showMonth = computed(() => props.priceMonth > 0 || props.priceYear === 0)
const perMonthSuffix = computed(() => (props.recurring && props.priceMonth > 0 ? '/мес' : ''))
</script>

<style scoped>
.offer {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 20px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg);
  background: var(--glass-bg), var(--acrylic-card-bg);
  box-shadow: var(--glass-edge);
  color: var(--color-text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.2s ease, transform 0.2s ease;
}

.offer:hover {
  border-color: var(--color-primary);
  }

.offer-title {
  margin: 0;
  font-size: 1.5rem;
  font-weight: 700;
  color: var(--color-primary);
}

.offer-desc {
  flex: 1;
  margin: 0;
  font-size: 0.84rem;
  line-height: 1.45;
  color: var(--color-text-dim);
}

.offer-prices {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
}

.price {
  padding: 8px 16px;
  border-radius: 999px;
  font-size: 0.9rem;
  font-weight: 600;
  white-space: nowrap;
}

.price--plain {
  background: var(--color-surface-variant);
  color: var(--color-text);
}

.price--accent {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.price--owned {
  background: var(--color-success-container, var(--color-surface-variant));
  color: var(--color-text);
}

.offer-note {
  margin: 0;
  font-size: 0.7rem;
  color: var(--color-text-dim);
}
</style>
