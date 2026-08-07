<template>
  <!-- «Мои товары»: купленное на витрине и СВОИ товары на продажу. Автор
       назначает цену, отправляет товар на модерацию и выводит выручку. -->
  <BrandLoader v-if="loading" block :size="64" />

  <AppStack v-else :gap="16">
    <AppRow
      title="Выручка автора"
      :hint="`Платформа удерживает ${commission}% с каждой продажи.
        Всего заработано: ${formatPrice(balance.total_earned, { free: '0 руб.' })}`"
    >
      <strong class="wallet-sum">{{ formatPrice(balance.balance, { free: '0 руб.' }) }}</strong>
      <AppButton label="Вывести" size="sm" :disabled="!balance.balance" @click="payoutOpen = true" />
    </AppRow>

    <AppCard title="Мои товары" :gap="10">
      <template #head>
        <AppButton
          label="Выставить товар"
          icon="add"
          variant="filled"
          size="sm"
          @click="createProduct"
        />
      </template>

      <EmptyState
        v-if="!products.length"
        icon="sell"
        size="sm"
        title="Вы ещё ничего не продаёте"
        subtitle="Выставьте свою тему или обои — после проверки они появятся на витрине."
      />
      <AppRow v-for="p in products" v-else :key="p.id" :title="p.title">
        <template #hint>
          {{ formatPrice(p.price) }} · продаж: {{ p.sales_count }}
          <template v-if="p.status === 'rejected' && p.reject_reason"> · {{ p.reject_reason }}</template>
        </template>

        <AppChip :tone="statusTone(p.status)" size="sm" :label="STATUS[p.status]" />
        <AppButton
          v-if="p.status === 'draft' || p.status === 'rejected'"
          label="На проверку"
          size="sm"
          @click="submit(p.id)"
        />
        <AppButton
          v-else-if="p.status === 'published'"
          label="Снять"
          size="sm"
          tone="neutral"
          @click="withdraw(p.id)"
        />
        <AppButton label="Изменить" size="sm" tone="neutral" @click="editProduct(p)" />
      </AppRow>
    </AppCard>

    <AppCard title="Купленное" :gap="10">
      <EmptyState
        v-if="!purchases.length"
        icon="shopping_bag"
        size="sm"
        title="Покупок пока нет"
        subtitle="Товары с витрины появятся здесь и сразу станут доступны в оформлении."
      />
      <AppRow
        v-for="pu in purchases"
        v-else
        :key="pu.id"
        :title="pu.product?.title"
        :hint="`${formatUntil(pu.created_at)} · ${formatPrice(pu.amount)}`"
      >
        <AppButton label="Открыть" size="sm" @click="$emit('open', pu.product)" />
      </AppRow>
    </AppCard>
  </AppStack>

  <ProductEditDialog v-model:visible="editorOpen" :product="editing" @saved="load" />

  <AppDialog
    v-model="payoutOpen"
    title="Вывод выручки"
    :actions="PAYOUT_ACTIONS"
    @confirm="sendPayout"
  >
    <div class="payout-form">
      <label class="field">
        <span class="field-label">Сумма, руб.</span>
        <InputNumber v-model="payoutRub" :min="1" :max="Math.floor(balance.balance / 100)" />
      </label>
      <label class="field">
        <span class="field-label">Реквизиты для перевода</span>
        <InputText v-model="payoutRequisites" placeholder="Телефон СБП или счёт" />
      </label>
      <p class="field-label">Заявку рассматривает администратор платформы.</p>
    </div>
  </AppDialog>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
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
  if (status === 'published') return 'success'
  if (status === 'review') return 'warning'
  if (status === 'rejected') return 'error'
  return 'primary'
}
</script>

<style scoped>
.wallet-sum { font-size: 1.15rem; font-weight: 700; }
.payout-form { display: flex; flex-direction: column; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label { font-size: 0.85rem; color: var(--color-text-dim); }
</style>
