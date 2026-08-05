<template>
  <!-- Карточка товара — внутренняя страница магазина (не модалка): слева
       описание, справа панель оформления с выбором периода и промокодом. -->
  <div class="card-page">
    <header class="card-head">
      <AppButton icon="arrow_back" label="Назад" size="sm" @click="$emit('back')" />
      <h2 class="card-head-title">Карточка товара</h2>
    </header>

    <div class="card-body">
      <section class="card-main">
        <h1 class="card-title">{{ heading }}</h1>
        <p v-if="offer.description" class="card-desc">{{ offer.description }}</p>

        <ul v-if="features.length" class="feature-list">
          <li v-for="f in features" :key="f.label">
            <span class="material-symbols-outlined">check_circle</span>
            <span>{{ f.label }}</span>
            <b>{{ f.value }}</b>
          </li>
        </ul>
      </section>

      <AppCard tag="aside" class="card-side" title="Оформление">
        <AppTabs v-if="hasYear" v-model="period" :tabs="PERIODS" variant="tint" dense full-width />

        <div class="side-price">
          <span class="side-cap">цена</span>
          <p class="price-value">{{ priceLabel }}</p>
          <p v-if="quoteResult && quoteResult.discount > 0" class="price-discount">
            скидка {{ formatPrice(quoteResult.discount) }} по промокоду
          </p>
        </div>

        <div class="promo-row">
          <InputText v-model="promo" placeholder="Промокод" class="promo-input" />
          <AppButton label="Применить" size="sm" :disabled="!promo || checking" @click="applyPromo" />
        </div>
        <p v-if="promoError" class="promo-error">{{ promoError }}</p>

        <AppButton
          :label="owned ? 'Уже куплено' : 'Оформить'"
          icon="shopping_bag"
          variant="filled"
          block
          :disabled="busy || owned"
          :loading="busy"
          @click="submit"
        />

        <p v-if="!paymentEnabled" class="pay-note">
          Оплата по СБП скоро подключится. Заказ появится в разделе «Заказы» —
          его подтвердит администратор платформы.
        </p>
      </AppCard>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import InputText from 'primevue/inputtext'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppTabs from '@/components/ui/AppTabs.vue'
import * as api from '@/api/billing.js'
import { formatBytes, formatCount, formatPrice } from '@/utils/money.js'

/* offer — нормализованное предложение витрины:
   { kind: 'subscription'|'addon'|'product', code, productId, title, description,
     priceMonth, priceYear, recurring, owned, limits } */
const props = defineProps({
  offer: { type: Object, required: true },
  paymentEnabled: { type: Boolean, default: false },
  busy: { type: Boolean, default: false },
})

const emit = defineEmits(['back', 'buy'])

const PERIODS = [
  { key: 'month', label: 'Оплата за месяц' },
  { key: 'year', label: 'Оплата за год' },
]

const period = ref('month')
const promo = ref('')
const promoError = ref('')
const checking = ref(false)
const quoteResult = ref(null)

const hasYear = computed(() => (props.offer.priceYear || 0) > 0)
const owned = computed(() => Boolean(props.offer.owned))

const heading = computed(() => {
  if (props.offer.kind === 'subscription') return `тариф «${props.offer.title}»`
  return props.offer.title
})

const basePrice = computed(() => {
  if (period.value === 'year' && hasYear.value) return props.offer.priceYear
  return props.offer.priceMonth
})

const priceLabel = computed(() => {
  const amount = quoteResult.value ? quoteResult.value.amount : basePrice.value
  if (!amount) return 'Бесплатно'
  const suffix = props.offer.recurring
    ? (period.value === 'year' ? ' / год' : ' / мес.')
    : ''
  return `${formatPrice(amount)}${suffix}`
})

// Что даёт тариф — лимиты приходят с сервера (plan_limits), клиент их не дублирует.
const features = computed(() => {
  const l = props.offer.limits
  if (!l) return []
  const num = (v) => (v == null || v < 0 ? 'без ограничений' : formatCount(v))
  const yn = (v) => (v ? 'есть' : 'нет')
  return [
    { label: 'Задачи', value: num(l.tasks) },
    { label: 'Компании', value: num(l.companies) },
    { label: 'Человек в компании', value: num(l.members) },
    { label: 'Хранилище', value: l.storage_bytes < 0 ? 'без ограничений' : formatBytes(l.storage_bytes) },
    { label: 'Токены ИИ в месяц', value: num(l.ai_tokens) },
    { label: 'Календари', value: num(l.calendars) },
    { label: 'Ежедневники', value: num(l.diaries) },
    { label: 'Доски', value: num(l.boards) },
    { label: 'Реестры', value: num(l.registries) },
    { label: 'Папки чатов', value: num(l.chat_folders) },
    { label: 'Участников в звонке', value: num(l.call_participants) },
    { label: 'Корпоративный портал', value: yn(l.portal) },
    { label: 'Расширенная статистика', value: yn(l.advanced_stats) },
    { label: 'Экспорт и импорт данных', value: yn(l.data_transfer) },
    { label: 'Статусы пользователей', value: yn(l.user_statuses) },
    { label: 'Премиум-темы и скины', value: yn(l.premium_themes) },
  ]
})

// Смена периода сбрасывает расчёт: цена другая, скидку надо пересчитать.
watch(period, () => { quoteResult.value = null })

function requestBody() {
  return {
    kind: props.offer.kind,
    item_code: props.offer.code || '',
    product_id: props.offer.productId || 0,
    period: period.value,
    qty: 1,
    promo: promo.value.trim() || undefined,
  }
}

async function applyPromo() {
  checking.value = true
  promoError.value = ''
  try {
    quoteResult.value = await api.quote(requestBody())
  } catch (e) {
    quoteResult.value = null
    promoError.value = e?.data?.message || 'Промокод не подошёл'
  } finally {
    checking.value = false
  }
}

function submit() {
  emit('buy', requestBody())
}
</script>

<style scoped>
.card-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-height: 0;
}

.card-head {
  display: flex;
  align-items: center;
  gap: 14px;
}

.card-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 320px;
  gap: 18px;
  align-items: start;
}

.card-title {
  margin: 0 0 12px;
  font-size: clamp(1.8rem, 5cqw, 3rem);
  font-weight: 800;
  line-height: 1.05;
  color: var(--color-primary);
}

.card-desc {
  margin: 0 0 18px;
  font-size: 0.95rem;
  line-height: 1.5;
  color: var(--color-text-dim);
}

.feature-list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(230px, 1fr));
  gap: 8px;
  margin: 0;
  padding: 0;
  list-style: none;
}

.feature-list li {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--radius-md);
  background: var(--color-surface-variant);
  font-size: 0.85rem;
}

.feature-list b { margin-left: auto; font-weight: 700; }
.feature-list .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }

/* Панель оформления идёт за прокруткой описания. */
.card-side {
  position: sticky;
  top: 0;
  align-self: start;
}

.side-price { display: flex; flex-direction: column; gap: 2px; }
.side-cap { font-size: 0.85rem; color: var(--color-text-dim); }

.price-value {
  margin: 0;
  font-size: clamp(1.4rem, 3cqw, 2rem);
  font-weight: 700;
}

.price-discount {
  margin: 0;
  font-size: 0.8rem;
  color: var(--color-success, var(--color-primary));
}

.promo-row { display: flex; gap: 8px; }
.promo-input { flex: 1; min-width: 0; }

.promo-error {
  margin: 0;
  font-size: 0.8rem;
  color: var(--color-error);
}

.card-head-title { margin: 0; font-size: 1.05rem; font-weight: 600; }

.pay-note { margin: 0; font-size: 0.85rem; color: var(--color-text-dim); }

@container (max-width: 820px) {
  .card-body { grid-template-columns: 1fr; }
  .card-side { position: static; }
}

@media (max-width: 820px) {
  .card-body { grid-template-columns: 1fr; }
}
</style>
