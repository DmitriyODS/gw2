<template>
  <!-- Заказы и выплаты. Пока платёжный шлюз — заглушка, оплата подтверждается
       здесь вручную: подтверждение выдаёт покупку ровно один раз. -->
  <div class="tab">
    <div class="filters">
      <Dropdown v-model="status" :options="STATUSES" option-label="label" option-value="value" @change="load" />
      <button class="gw-chip" type="button" @click="load">Обновить</button>
    </div>

    <section v-if="!orders.length" class="gw-banner">
      <h2>Заказов нет</h2>
    </section>
    <div v-else class="rows">
      <article v-for="o in orders" :key="o.id" class="gw-card gw-row row">
        <span class="gw-row-icon"><span class="material-symbols-outlined">receipt_long</span></span>
        <div class="row-main">
          <p class="gw-h">#{{ o.id }} · {{ o.title || o.item_code }}</p>
          <p class="gw-sub">
            пользователь {{ o.user_id }} · {{ formatPrice(o.amount) }} ·
            {{ formatUntil(o.created_at) }}
          </p>
        </div>
        <span class="chip-tint" :class="tone(o.status)">{{ STATUS_LABELS[o.status] || o.status }}</span>
        <button v-if="o.status === 'pending'" class="gw-chip" type="button" @click="confirm(o.id)">
          Подтвердить оплату
        </button>
      </article>
    </div>

    <h3 class="gw-h">Выплаты авторам</h3>
    <section v-if="!payouts.length" class="gw-banner">
      <h2>Заявок нет</h2>
    </section>
    <div v-else class="rows">
      <article v-for="p in payouts" :key="p.id" class="gw-card gw-row row">
        <span class="gw-row-icon"><span class="material-symbols-outlined">payments</span></span>
        <div class="row-main">
          <p class="gw-h">{{ p.user_name || `#${p.user_id}` }} · {{ formatPrice(p.amount) }}</p>
          <p class="gw-sub">{{ p.requisites }} · {{ formatUntil(p.created_at) }}</p>
        </div>
        <span class="chip-tint" :class="payoutTone(p.status)">{{ PAYOUT_LABELS[p.status] || p.status }}</span>
        <template v-if="p.status === 'requested'">
          <button class="gw-chip" type="button" @click="process(p.id, 'paid')">Выплачено</button>
          <button class="gw-chip" type="button" @click="process(p.id, 'rejected')">Отказать</button>
        </template>
      </article>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import Dropdown from 'primevue/dropdown'
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
  if (s === 'paid') return 'chip-tint--success'
  if (s === 'pending') return 'chip-tint--warning'
  if (s === 'canceled' || s === 'failed') return 'chip-tint--error'
  return 'chip-tint--primary'
}

function payoutTone(s) {
  if (s === 'paid') return 'chip-tint--success'
  if (s === 'requested') return 'chip-tint--warning'
  return 'chip-tint--error'
}
</script>

<style scoped>
.tab { display: flex; flex-direction: column; gap: 14px; }
.filters { display: flex; flex-wrap: wrap; gap: 10px; }
.rows { display: flex; flex-direction: column; gap: 10px; }
.row { padding: 14px; }
.row-main { flex: 1; min-width: 0; }
</style>
