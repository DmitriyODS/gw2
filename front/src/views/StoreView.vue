<template>
  <!-- Магазин: витрина товаров, свои товары на продажу, заказы и подписки.
       Подписки и товары продаются за рубли (оплата — СБП, см. billingsvc).
       Карточка товара — ВНУТРЕННЯЯ страница раздела, а не модалка. -->
  <AppPage title="Магазин" :loading="billing.loading && !billing.entitlements">
    <template #subhead>
      <AppTabs v-model="tab" :tabs="TABS" variant="tint" />
    </template>

    <!-- Карточка товара перекрывает вкладку целиком (как «назад» в проводнике). -->
    <OfferPage
      v-if="openedOffer"
      :offer="openedOffer"
      :payment-enabled="paymentEnabled"
      :busy="buying"
      @back="openedOffer = null"
      @buy="buy"
    />

    <AppStack v-else-if="tab === 'subs'" :gap="16">
      <StoreStatCards
        :plan-name="billing.planName"
        :is-free="billing.isFree"
        :expires-at="billing.expiresAt"
        :storage-used="billing.storageUsed"
        :storage-limit="billing.storageLimit"
        :tokens-used="billing.tokensUsed"
        :tokens-limit="billing.tokensLimit"
        @manage="manage"
      />

      <AppSwitchRow
        v-if="billing.subscription"
        title="Автопродление"
        :hint="billing.subscription.auto_renew
          ? 'Тариф продлится автоматически в конце периода'
          : 'Тариф закончится в конце оплаченного периода'"
        :model-value="billing.subscription.auto_renew"
        @update:model-value="toggleAutoRenew"
      />

      <AppCard title="Доступные подписки">
        <AppGrid :min="240">
          <OfferCard
            v-for="offer in subscriptionOffers"
            :key="offer.key"
            :title="offer.title"
            :description="offer.description"
            :price-month="offer.priceMonth"
            :price-year="offer.priceYear"
            :recurring="offer.recurring"
            @open="openedOffer = offer"
          />
        </AppGrid>
      </AppCard>

      <AppCard v-if="billing.myAddons.length" title="Мои дополнения" :gap="10">
        <AppRow
          v-for="a in billing.myAddons"
          :key="a.id"
          :title="a.name"
          :hint="a.expires_at ? `действует до ${formatUntil(a.expires_at)}` : 'бессрочно'"
        >
          <AppButton label="Отключить" size="sm" tone="neutral" @click="dropAddon(a.id)" />
        </AppRow>
      </AppCard>
    </AppStack>

    <AppStack v-else-if="tab === 'shop'" :gap="16">
      <!-- Категорий ровно четыре и они делят ширину поровну, поэтому сетка своя:
           auto-fill оставил бы половину ряда пустой. -->
      <div class="store-cats">
        <AppTile
          v-for="cat in CATEGORIES"
          :key="cat.key"
          :icon="cat.icon"
          :label="cat.label"
          :selected="kind === cat.key"
          @click="selectKind(cat.key)"
        />
      </div>

      <EmptyState
        v-if="!products.length"
        icon="storefront"
        title="Витрина пока пуста"
        subtitle="Товары появятся здесь — их публикуют платформа и авторы тем."
      />
      <AppGrid v-else :min="240">
        <OfferCard
          v-for="p in products"
          :key="p.id"
          :title="p.title"
          :description="p.description"
          :price-month="p.price"
          :price-year="0"
          :recurring="false"
          :owned="p.owned"
          @open="openProduct(p)"
        />
      </AppGrid>
    </AppStack>

    <MyGoodsTab v-else-if="tab === 'mine'" @open="openProduct" />

    <OrdersTab v-else-if="tab === 'orders'" />
  </AppPage>
</template>

<script setup>
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppGrid from '@/components/ui/AppGrid.vue'
import AppPage from '@/components/ui/AppPage.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppSwitchRow from '@/components/ui/AppSwitchRow.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import AppTile from '@/components/ui/AppTile.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import StoreStatCards from '@/components/store/StoreStatCards.vue'
import OfferCard from '@/components/store/OfferCard.vue'
import OfferPage from '@/components/store/OfferPage.vue'
import MyGoodsTab from '@/components/store/MyGoodsTab.vue'
import OrdersTab from '@/components/store/OrdersTab.vue'
import * as api from '@/api/billing.js'
import { useBillingStore } from '@/stores/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatUntil } from '@/utils/money.js'

const billing = useBillingStore()
const notif = useNotificationsStore()
const route = useRoute()
const router = useRouter()

const TABS = [
  { key: 'shop', label: 'Магазин', icon: 'storefront' },
  { key: 'mine', label: 'Мои товары', icon: 'sell' },
  { key: 'orders', label: 'Заказы', icon: 'receipt_long' },
  { key: 'subs', label: 'Подписки', icon: 'card_membership' },
]

const CATEGORIES = [
  { key: 'theme', label: 'Темы', icon: 'palette' },
  { key: 'wallpaper', label: 'Обои', icon: 'image' },
  { key: 'gradient', label: 'Градиенты', icon: 'gradient' },
  { key: 'pet_skin', label: 'Питомцу', icon: 'pets' },
]

const tabKeys = new Set(TABS.map((t) => t.key))
const tab = ref(tabKeys.has(route.query.tab) ? route.query.tab : 'shop')
const kind = ref('')
const products = ref([])
const openedOffer = ref(null)
const buying = ref(false)

const paymentEnabled = computed(() => Boolean(billing.settings?.payment_enabled))

// Витрина подписок: платные тарифы + докупки. Бесплатный «Джун» не оформляется.
const subscriptionOffers = computed(() => {
  const plans = billing.plans
    .filter((p) => p.price_month > 0 || p.price_year > 0)
    .map((p) => ({
      key: `plan:${p.code}`,
      kind: 'subscription',
      code: p.code,
      title: p.name,
      description: p.tagline,
      priceMonth: p.price_month,
      priceYear: p.price_year,
      recurring: true,
      limits: billing.planLimits[p.code] || null,
    }))
  const addons = billing.addons.map((a) => ({
    key: `addon:${a.code}`,
    kind: 'addon',
    code: a.code,
    title: a.name,
    description: a.description,
    priceMonth: a.price_month,
    priceYear: a.price_year,
    recurring: a.recurring,
  }))
  return [...plans, ...addons]
})

watch(() => route.query.tab, (key) => {
  if (tabKeys.has(key)) tab.value = key
})

watch(tab, () => { openedOffer.value = null })

onMounted(async () => {
  await billing.fetchShowcase()
  await loadProducts()
})

async function loadProducts() {
  const res = await api.getProducts({ kind: kind.value })
  products.value = res.items ?? []
}

function selectKind(next) {
  kind.value = kind.value === next ? '' : next
  loadProducts()
}

function openProduct(p) {
  openedOffer.value = {
    key: `product:${p.id}`,
    kind: 'product',
    productId: p.id,
    title: p.title,
    description: p.description,
    priceMonth: p.price,
    priceYear: 0,
    recurring: false,
    owned: p.owned,
  }
  tab.value = 'shop'
}

async function buy(body) {
  buying.value = true
  try {
    const order = await billing.purchase(body)
    openedOffer.value = null
    if (order.status === 'paid') {
      notif.notify({ severity: 'success', summary: 'Готово', detail: 'Покупка активирована', life: 4000 })
    } else {
      tab.value = 'orders'
      notif.notify({
        severity: 'info',
        summary: 'Заказ создан',
        detail: paymentEnabled.value
          ? 'Осталось оплатить его по СБП'
          : 'Оплата подключается — заказ подтвердит администратор платформы',
        life: 6000,
      })
    }
    await loadProducts()
  } catch (e) {
    notif.notify({
      severity: 'error',
      summary: 'Не получилось',
      detail: e?.data?.message || 'Попробуйте ещё раз',
      life: 5000,
    })
  } finally {
    buying.value = false
  }
}

async function toggleAutoRenew(value) {
  try {
    await billing.setAutoRenew(value)
  } catch {
    notif.notify({ severity: 'error', summary: 'Не удалось изменить автопродление', life: 4000 })
  }
}

async function dropAddon(id) {
  await billing.cancelAddon(id)
}

// Кнопки карточек состояния: тариф — к линейке, место и токены — к докупкам.
function manage(what) {
  tab.value = 'subs'
  if (what === 'tokens') {
    const offer = subscriptionOffers.value.find((o) => o.code === 'tokens_1000')
    if (offer) openedOffer.value = offer
    return
  }
  if (what === 'storage') {
    const offer = subscriptionOffers.value.find((o) => o.code?.startsWith('storage_'))
    if (offer) openedOffer.value = offer
    return
  }
  if (billing.isFree) {
    const offer = subscriptionOffers.value.find((o) => o.kind === 'subscription')
    if (offer) openedOffer.value = offer
    return
  }
  router.push('/store?tab=subs').catch(() => {})
}
</script>

<style scoped>
.store-cats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

/* Раскладка считается от ширины ОКНА раздела (панель AppPage — container). */
@container (max-width: 720px) {
  .store-cats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

/* Старый WebView (chrome87) не знает @container — дублируем media-запросом. */
@media (max-width: 720px) {
  .store-cats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
