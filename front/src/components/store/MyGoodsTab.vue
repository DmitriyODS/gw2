<template>
  <!-- «Мои товары»: купленное на витрине и СВОИ товары на продажу. Автор
       назначает цену, отправляет товар на модерацию и выводит выручку. -->
  <div class="mine">
    <BrandLoader v-if="loading" :size="48" />

    <template v-else>
      <section class="gw-card wallet">
        <div class="gw-row">
          <div class="wallet-main">
            <p class="gw-h">Выручка автора</p>
            <p class="gw-sub">
              Платформа удерживает {{ commission }}% с каждой продажи.
              Всего заработано: {{ formatPrice(balance.total_earned, { free: '0 руб.' }) }}
            </p>
          </div>
          <strong class="wallet-sum">{{ formatPrice(balance.balance, { free: '0 руб.' }) }}</strong>
          <button class="gw-chip" type="button" :disabled="!balance.balance" @click="payoutOpen = true">
            Вывести
          </button>
        </div>
      </section>

      <div class="mine-head">
        <h2 class="gw-title section-title">Мои товары</h2>
        <button class="btn-grad" type="button" @click="createProduct">
          <span class="material-symbols-outlined">add</span>
          Выставить товар
        </button>
      </div>

      <section v-if="!products.length" class="gw-banner">
        <h2>Вы ещё ничего не продаёте</h2>
        <p class="gw-sub">Выставьте свою тему или обои — после проверки они появятся на витрине.</p>
      </section>
      <section v-else class="gw-group rows">
        <article v-for="p in products" :key="p.id" class="gw-card gw-row row">
          <div class="row-main">
            <p class="gw-h">{{ p.title }}</p>
            <p class="gw-sub">
              {{ formatPrice(p.price) }} · продаж: {{ p.sales_count }}
              <template v-if="p.status === 'rejected' && p.reject_reason"> · {{ p.reject_reason }}</template>
            </p>
          </div>
          <span class="chip-tint" :class="statusTone(p.status)">{{ STATUS[p.status] }}</span>
          <button
            v-if="p.status === 'draft' || p.status === 'rejected'"
            class="gw-chip"
            type="button"
            @click="submit(p.id)"
          >
            На проверку
          </button>
          <button
            v-else-if="p.status === 'published'"
            class="gw-chip"
            type="button"
            @click="withdraw(p.id)"
          >
            Снять
          </button>
          <button class="gw-chip" type="button" @click="editProduct(p)">Изменить</button>
        </article>
      </section>

      <h2 class="gw-title section-title">Купленное</h2>
      <section v-if="!purchases.length" class="gw-banner">
        <h2>Покупок пока нет</h2>
        <p class="gw-sub">Товары с витрины появятся здесь и сразу станут доступны в оформлении.</p>
      </section>
      <section v-else class="gw-group rows">
        <article v-for="pu in purchases" :key="pu.id" class="gw-card gw-row row">
          <div class="row-main">
            <p class="gw-h">{{ pu.product?.title }}</p>
            <p class="gw-sub">{{ formatUntil(pu.created_at) }} · {{ formatPrice(pu.amount) }}</p>
          </div>
          <button class="gw-chip" type="button" @click="$emit('open', pu.product)">Открыть</button>
        </article>
      </section>
    </template>

    <ProductEditDialog v-model:visible="editorOpen" :product="editing" @saved="load" />

    <AppDialog
      v-model="payoutOpen"
      title="Вывод выручки"
      icon="payments"
      :actions="PAYOUT_ACTIONS"
      @confirm="sendPayout"
    >
      <div class="payout-form">
        <label class="field">
          <span class="gw-sub">Сумма, руб.</span>
          <InputNumber v-model="payoutRub" :min="1" :max="Math.floor(balance.balance / 100)" />
        </label>
        <label class="field">
          <span class="gw-sub">Реквизиты для перевода</span>
          <InputText v-model="payoutRequisites" placeholder="Телефон СБП или счёт" />
        </label>
        <p class="gw-sub">Заявку рассматривает администратор платформы.</p>
      </div>
    </AppDialog>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import AppDialog from '@/components/common/AppDialog.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import ProductEditDialog from '@/components/store/ProductEditDialog.vue'
import * as api from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatPrice, formatUntil } from '@/utils/money.js'

defineEmits(['open'])

const notif = useNotificationsStore()

const STATUS = {
  draft: 'Черновик',
  review: 'На проверке',
  published: 'На витрине',
  rejected: 'Отклонён',
  removed: 'Снят',
}

const loading = ref(true)
const products = ref([])
const purchases = ref([])
const balance = ref({ balance: 0, total_earned: 0 })
const commission = ref(10)

const editorOpen = ref(false)
const editing = ref(null)

const payoutOpen = ref(false)
const payoutRub = ref(null)
const payoutRequisites = ref('')

const PAYOUT_ACTIONS = [
  { kind: 'cancel', label: 'Отмена' },
  { kind: 'confirm', label: 'Отправить заявку', icon: 'send' },
]

onMounted(load)

async function load() {
  loading.value = true
  try {
    const data = await api.getMyStore()
    products.value = data.products ?? []
    purchases.value = data.purchases ?? []
    balance.value = data.balance ?? { balance: 0, total_earned: 0 }
    commission.value = data.settings?.commission_pct ?? 10
  } finally {
    loading.value = false
  }
}

function createProduct() {
  editing.value = null
  editorOpen.value = true
}

function editProduct(p) {
  editing.value = p
  editorOpen.value = true
}

async function submit(id) {
  await api.submitMyProduct(id)
  notif.notify({ severity: 'info', summary: 'Отправлено на проверку', life: 4000 })
  await load()
}

async function withdraw(id) {
  await api.withdrawMyProduct(id)
  await load()
}

async function sendPayout() {
  try {
    await api.requestPayout(Math.round((payoutRub.value || 0) * 100), payoutRequisites.value)
    payoutOpen.value = false
    payoutRub.value = null
    payoutRequisites.value = ''
    notif.notify({ severity: 'success', summary: 'Заявка отправлена', life: 4000 })
    await load()
  } catch (e) {
    notif.notify({
      severity: 'error',
      summary: 'Не удалось отправить заявку',
      detail: e?.data?.message || '',
      life: 5000,
    })
  }
}

function statusTone(status) {
  if (status === 'published') return 'chip-tint--success'
  if (status === 'review') return 'chip-tint--warning'
  if (status === 'rejected') return 'chip-tint--error'
  return 'chip-tint--primary'
}
</script>

<style scoped>
.mine { display: flex; flex-direction: column; gap: 14px; }
.section-title { font-size: 1.35rem; }
.mine-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.rows { display: flex; flex-direction: column; gap: 10px; }
.row { padding: 14px; }
.row-main { flex: 1; min-width: 0; }
.wallet { padding: 14px; }
.wallet-main { flex: 1; min-width: 0; }
.wallet-sum { font-size: 1.15rem; font-weight: 700; }
.payout-form { display: flex; flex-direction: column; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
</style>
