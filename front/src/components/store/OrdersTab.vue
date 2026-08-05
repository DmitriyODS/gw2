<template>
  <!-- Заказы: покупки подписок, докупок и товаров с их статусом оплаты. -->
  <BrandLoader v-if="loading" :size="48" />

  <EmptyState
    v-else-if="!items.length"
    icon="receipt_long"
    title="Заказов пока нет"
    subtitle="Здесь появятся покупки подписок, дополнений и товаров."
  />

  <AppStack v-else :gap="10">
    <AppRow v-for="o in items" :key="o.id" :title="o.title || o.item_code">
      <template #hint>
        {{ formatUntil(o.created_at) }} · {{ periodLabel(o) }} · {{ formatPrice(o.amount) }}
        <template v-if="o.discount > 0"> · скидка {{ formatPrice(o.discount) }}</template>
      </template>

      <AppChip :tone="statusTone(o.status)" size="sm" :label="STATUS[o.status] || o.status" />
      <AppButton
        v-if="o.status === 'pending'"
        label="Отменить"
        size="sm"
        tone="neutral"
        @click="cancel(o.id)"
      />
    </AppRow>
  </AppStack>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
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
  if (status === 'paid') return 'success'
  if (status === 'pending') return 'warning'
  if (status === 'canceled' || status === 'failed') return 'error'
  return 'primary'
}
</script>
