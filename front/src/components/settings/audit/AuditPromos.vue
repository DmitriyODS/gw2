<template>
  <!-- Промокоды: скидки к покупке (процент/сумма) и подарки (дни тарифа,
       пачка токенов). Подарочные активируются в магазине отдельной кнопкой. -->
  <AppStack :gap="14">
    <AppCard title="Новый промокод">
      <div class="form-row">
        <label class="field">
          <span class="field-label">Код</span>
          <InputText v-model="form.code" placeholder="SUMMER" />
        </label>
        <label class="field">
          <span class="field-label">Что даёт</span>
          <Dropdown v-model="form.kind" :options="KINDS" option-label="label" option-value="value" />
        </label>
        <label class="field">
          <span class="field-label">{{ valueLabel }}</span>
          <InputNumber v-model="form.value" :min="0" />
        </label>
        <label v-if="form.kind === 'days'" class="field">
          <span class="field-label">Тариф</span>
          <Dropdown v-model="form.plan_code" :options="PLANS" option-label="label" option-value="value" />
        </label>
      </div>
      <div class="form-row">
        <label class="field">
          <span class="field-label">Всего активаций (0 — без предела)</span>
          <InputNumber v-model="form.max_uses" :min="0" />
        </label>
        <label class="field">
          <span class="field-label">На одного пользователя</span>
          <InputNumber v-model="form.per_user_limit" :min="1" />
        </label>
        <label class="field">
          <span class="field-label">Действует до</span>
          <InputText v-model="form.expires_at" type="date" />
        </label>
        <AppButton label="Создать" icon="add" variant="filled" @click="create" />
      </div>
      <p v-if="error" class="form-error">{{ error }}</p>
    </AppCard>

    <EmptyState v-if="!items.length" icon="confirmation_number" size="sm" title="Промокодов пока нет" />
    <AppStack v-else :gap="10">
      <AppRow v-for="p in items" :key="p.id" :title="p.code">
        <template #hint>
          {{ describe(p) }} · активаций {{ p.used_count }}<template v-if="p.max_uses"> из {{ p.max_uses }}</template>
          <template v-if="p.expires_at"> · до {{ formatUntil(p.expires_at) }}</template>
        </template>
        <InputSwitch :model-value="p.is_active" @update:model-value="toggle(p, $event)" />
        <AppButton label="Удалить" size="sm" tone="danger" @click="remove(p.id)" />
      </AppRow>
    </AppStack>
  </AppStack>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import InputSwitch from 'primevue/inputswitch'
import Dropdown from 'primevue/dropdown'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import * as api from '@/api/billing.js'
import { formatPrice, formatUntil } from '@/utils/money.js'

const KINDS = [
  { value: 'percent', label: 'Скидка, %' },
  { value: 'amount', label: 'Скидка, руб.' },
  { value: 'days', label: 'Дни тарифа' },
  { value: 'tokens', label: 'Токены ИИ' },
]

const PLANS = [
  { value: 'middle', label: 'Мидл' },
  { value: 'senior', label: 'Синьор' },
]

const items = ref([])
const error = ref('')
const form = ref(emptyForm())

const valueLabel = computed(() => KINDS.find((k) => k.value === form.value.kind)?.label || 'Значение')

onMounted(load)

function emptyForm() {
  return {
    code: '', kind: 'percent', value: 20, plan_code: 'middle',
    applies_to: 'any', max_uses: 0, per_user_limit: 1, expires_at: '', is_active: true, comment: '',
  }
}

async function load() {
  const res = await api.adminPromos()
  items.value = res.items ?? []
}

function describe(p) {
  if (p.kind === 'percent') return `скидка ${p.value}%`
  if (p.kind === 'amount') return `скидка ${formatPrice(p.value)}`
  if (p.kind === 'days') return `${p.value} дней тарифа ${p.plan_code || 'middle'}`
  return `${p.value} токенов ИИ`
}

async function create() {
  error.value = ''
  const body = { ...form.value }
  // Скидка в рублях приходит в копейках — сервер считает деньги только в них.
  if (body.kind === 'amount') body.value = Math.round(body.value * 100)
  try {
    await api.adminCreatePromo(body)
    form.value = emptyForm()
    await load()
  } catch (e) {
    error.value = e?.data?.message || 'Не удалось создать промокод'
  }
}

async function toggle(promo, value) {
  await api.adminUpdatePromo(promo.id, { ...promo, is_active: value })
  await load()
}

async function remove(id) {
  await api.adminDeletePromo(id)
  await load()
}
</script>

<style scoped>
.form-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label { font-size: 0.85rem; color: var(--color-text-dim); }
.form-error { margin: 0; font-size: 0.85rem; color: var(--color-error); }
</style>
