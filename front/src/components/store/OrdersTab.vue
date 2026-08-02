<template>
  <!-- Заказы: покупки подписок, докупок и товаров с их статусом оплаты. -->
  <div class="orders">
    <BrandLoader v-if="loading" :size="48" />
    <section v-else-if="!items.length" class="gw-banner">
      <h2>Заказов пока нет</h2>
      <p class="gw-sub">Здесь появятся покупки подписок, дополнений и товаров.</p>
    </section>

    <section v-else class="gw-group order-list">
      <article v-for="o in items" :key="o.id" class="gw-card gw-row order">
        <div class="order-main">
          <p class="gw-h">{{ o.title || o.item_code }}</p>
          <p class="gw-sub">
            {{ formatUntil(o.created_at) }} · {{ periodLabel(o) }} · {{ formatPrice(o.amount) }}
            <template v-if="o.discount > 0"> · скидка {{ formatPrice(o.discount) }}</template>
          </p>
        </div>
        <span class="chip-tint" :class="statusTone(o.status)">{{ STATUS[o.status] || o.status }}</span>
        <button
          v-if="o.status === 'pending'"
          class="gw-chip"
          type="button"
          @click="cancel(o.id)"
        >
          Отменить
        </button>
      </article>
    </section>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import * as api from '@/api/billing.js'
import { formatPrice, formatUntil } from '@/utils/money.js'

const items = ref([])
const loading = ref(true)

const STATUS = {
  pending: 'Ждёт оплаты',
  paid: 'Оплачен',
  canceled: 'Отменён',
  failed: 'Не прошёл',
  refunded: 'Возвращён',
}

onMounted(load)

async function load() {
  loading.value = true
  try {
    const res = await api.getOrders()
    items.value = res.items ?? []
  } finally {
    loading.value = false
  }
}

async function cancel(id) {
  await api.cancelOrder(id)
  await load()
}

function periodLabel(o) {
  if (o.kind === 'product') return 'разовая покупка'
  return o.period === 'year' ? 'на год' : 'на месяц'
}

function statusTone(status) {
  if (status === 'paid') return 'chip-tint--success'
  if (status === 'pending') return 'chip-tint--warning'
  if (status === 'canceled' || status === 'failed') return 'chip-tint--error'
  return 'chip-tint--primary'
}
</script>

<style scoped>
.orders { display: flex; flex-direction: column; gap: 12px; }
.order-list { display: flex; flex-direction: column; gap: 10px; }
.order { padding: 14px; }
.order-main { flex: 1; min-width: 0; }
</style>
