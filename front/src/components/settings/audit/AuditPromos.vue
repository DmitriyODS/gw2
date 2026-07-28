<template>
  <!-- Промокоды: скидки к покупке (процент/сумма) и подарки (дни тарифа,
       пачка токенов). Подарочные активируются в магазине отдельной кнопкой. -->
  <div class="tab">
    <section class="gw-card form">
      <p class="gw-h">Новый промокод</p>
      <div class="form-row">
        <label class="field">
          <span class="gw-sub">Код</span>
          <InputText v-model="form.code" placeholder="SUMMER" />
        </label>
        <label class="field">
          <span class="gw-sub">Что даёт</span>
          <Dropdown v-model="form.kind" :options="KINDS" option-label="label" option-value="value" />
        </label>
        <label class="field">
          <span class="gw-sub">{{ valueLabel }}</span>
          <InputNumber v-model="form.value" :min="0" />
        </label>
        <label v-if="form.kind === 'days'" class="field">
          <span class="gw-sub">Тариф</span>
          <Dropdown v-model="form.plan_code" :options="PLANS" option-label="label" option-value="value" />
        </label>
      </div>
      <div class="form-row">
        <label class="field">
          <span class="gw-sub">Всего активаций (0 — без предела)</span>
          <InputNumber v-model="form.max_uses" :min="0" />
        </label>
        <label class="field">
          <span class="gw-sub">На одного пользователя</span>
          <InputNumber v-model="form.per_user_limit" :min="1" />
        </label>
        <label class="field">
          <span class="gw-sub">Действует до</span>
          <InputText v-model="form.expires_at" type="date" />
        </label>
        <button class="btn-grad" type="button" @click="create">Создать</button>
      </div>
      <p v-if="error" class="form-error">{{ error }}</p>
    </section>

    <section v-if="!items.length" class="gw-banner">
      <h2>Промокодов пока нет</h2>
    </section>
    <div v-else class="rows">
      <article v-for="p in items" :key="p.id" class="gw-card gw-row row">
        <span class="gw-row-icon"><span class="material-symbols-outlined">local_activity</span></span>
        <div class="row-main">
          <p class="gw-h">{{ p.code }}</p>
          <p class="gw-sub">
            {{ describe(p) }} · активаций {{ p.used_count }}<template v-if="p.max_uses"> из {{ p.max_uses }}</template>
            <template v-if="p.expires_at"> · до {{ formatUntil(p.expires_at) }}</template>
          </p>
        </div>
        <InputSwitch :model-value="p.is_active" @update:model-value="toggle(p, $event)" />
        <button class="gw-chip" type="button" @click="remove(p.id)">Удалить</button>
      </article>
    </div>
  </div>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import InputSwitch from 'primevue/inputswitch'
import Dropdown from 'primevue/dropdown'
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
.tab { display: flex; flex-direction: column; gap: 14px; }
.form { display: flex; flex-direction: column; gap: 12px; }
.form-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.form-error { margin: 0; font-size: 0.85rem; color: var(--color-error); }
.rows { display: flex; flex-direction: column; gap: 10px; }
.row { padding: 14px; }
.row-main { flex: 1; min-width: 0; }
</style>
