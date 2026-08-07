<template>
  <!-- Заказы и выплаты. Пока платёжный шлюз — заглушка, оплата подтверждается
       здесь вручную: подтверждение выдаёт покупку ровно один раз. -->
  <AppStack :gap="14">
    <AppStack row :gap="10">
      <Dropdown v-model="status" :options="STATUSES" option-label="label" option-value="value" @change="load" />
      <AppButton label="Обновить" icon="refresh" size="sm" @click="load" />
    </AppStack>

    <EmptyState v-if="!orders.length" icon="receipt_long" size="sm" title="Заказов нет" />
    <AppStack v-else :gap="10">
      <AppRow v-for="o in orders" :key="o.id" :title="`#${o.id} · ${o.title || o.item_code}`">
        <template #hint>
          пользователь {{ o.user_id }} · {{ formatPrice(o.amount) }} ·
          {{ formatUntil(o.created_at) }}
        </template>
        <AppChip :tone="tone(o.status)" size="sm" :label="STATUS_LABELS[o.status] || o.status" />
        <AppButton
          v-if="o.status === 'pending'"
          label="Подтвердить оплату"
          size="sm"
          @click="confirm(o.id)"
        />
      </AppRow>
    </AppStack>

    <AppCard title="Выплаты авторам" :gap="10">
      <EmptyState v-if="!payouts.length" icon="payments" size="sm" title="Заявок нет" />
      <AppRow
        v-for="p in payouts"
        v-else
        :key="p.id"
        :title="`${p.user_name || `#${p.user_id}`} · ${formatPrice(p.amount)}`"
        :hint="`${p.requisites} · ${formatUntil(p.created_at)}`"
      >
        <AppChip :tone="payoutTone(p.status)" size="sm" :label="PAYOUT_LABELS[p.status] || p.status" />
        <template v-if="p.status === 'requested'">
          <AppButton label="Выплачено" size="sm" @click="process(p.id, 'paid')" />
          <AppButton label="Отказать" size="sm" tone="danger" @click="process(p.id, 'rejected')" />
        </template>
      </AppRow>
    </AppCard>
  </AppStack>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import Dropdown from 'primevue/dropdown'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import * as api from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatPrice, formatUntil } from '@/utils/money.js'

const notif = useNotificationsStore()

const STATUSES = [
  { value: '', label: 'Все заказы' },
  { value: 'pending', label: 'Ждут оплаты' },
  { value: 'paid', label: 'Оплаченные' },
  { value: 'canceled', label: 'Отменённые' },
]

const STATUS_LABELS = {
  pending: 'Ждёт оплаты', paid: 'Оплачен', canceled: 'Отменён',
  failed: 'Не прошёл', refunded: 'Возвращён',
}

const PAYOUT_LABELS = { requested: 'Ждёт решения', paid: 'Выплачено', rejected: 'Отказано' }

const orders = ref([])
const payouts = ref([])
const status = ref('')

onMounted(load)

async function load() {
  const [o, p] = await Promise.all([api.adminOrders({ status: status.value }), api.adminPayouts()])
  orders.value = o.items ?? []
  payouts.value = p.items ?? []
}

async function confirm(id) {
  try {
    await api.adminConfirmOrder(id)
    notif.notify({ severity: 'success', summary: 'Оплата подтверждена', life: 3000 })
    await load()
  } catch (e) {
    notif.notify({ severity: 'error', summary: 'Не получилось', detail: e?.data?.message || '', life: 5000 })
  }
}

async function process(id, decision) {
  const note = decision === 'rejected' ? (window.prompt('Причина отказа') || '') : ''
  await api.adminProcessPayout(id, decision, note)
  await load()
}

function tone(s) {
  if (s === 'paid') return 'success'
  if (s === 'pending') return 'warning'
  if (s === 'canceled' || s === 'failed') return 'error'
  return 'primary'
}

function payoutTone(s) {
  if (s === 'paid') return 'success'
  if (s === 'requested') return 'warning'
  return 'error'
}
</script>

