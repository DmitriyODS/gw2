<template>
  <!-- Тарифы и докупки: супер-админ правит ЦЕНЫ и подводки. Лимиты тарифа
       конечны и заданы в коде сервиса — здесь они только показаны. -->
  <div class="tab">
    <section class="gw-card">
      <div class="gw-row">
        <div class="row-main">
          <p class="gw-h">Комиссия платформы</p>
          <p class="gw-sub">Удерживается с продажи товара автора</p>
        </div>
        <InputNumber v-model="settings.commission_pct" :min="0" :max="100" suffix=" %" class="num" />
      </div>
      <div class="gw-row toggles">
        <label class="toggle">
          <InputSwitch v-model="settings.store_enabled" />
          <span>Магазин открыт</span>
        </label>
        <label class="toggle">
          <InputSwitch v-model="settings.payment_enabled" />
          <span>Приём оплаты включён</span>
        </label>
        <button class="gw-chip" type="button" @click="saveSettings">Сохранить</button>
      </div>
      <p class="gw-sub">
        Платёжный шлюз: {{ settings.payment_provider || 'manual' }}. Пока это
        заглушка — заказы подтверждаются вручную во вкладке «Заказы».
      </p>
    </section>

    <h3 class="gw-h">Тарифы</h3>
    <section v-for="plan in plans" :key="plan.code" class="gw-card plan">
      <div class="plan-head">
        <InputText v-model="plan.name" class="plan-name" />
        <label class="toggle">
          <InputSwitch v-model="plan.is_active" />
          <span>Продаётся</span>
        </label>
        <button class="gw-chip" type="button" @click="savePlan(plan)">Сохранить</button>
      </div>
      <Textarea v-model="plan.tagline" rows="2" auto-resize class="plan-tagline" />
      <div class="price-row">
        <label class="field">
          <span class="gw-sub">Цена за месяц, руб.</span>
          <InputNumber :model-value="plan.price_month / 100" :min="0" @update:model-value="v => plan.price_month = Math.round((v || 0) * 100)" />
        </label>
        <label class="field">
          <span class="gw-sub">Цена за год, руб.</span>
          <InputNumber :model-value="plan.price_year / 100" :min="0" @update:model-value="v => plan.price_year = Math.round((v || 0) * 100)" />
        </label>
      </div>
      <p class="gw-sub limits">{{ limitsSummary(plan.code) }}</p>
    </section>

    <h3 class="gw-h">Дополнения</h3>
    <section v-for="addon in addons" :key="addon.code" class="gw-card plan">
      <div class="plan-head">
        <InputText v-model="addon.name" class="plan-name" />
        <label class="toggle">
          <InputSwitch v-model="addon.is_active" />
          <span>Продаётся</span>
        </label>
        <button class="gw-chip" type="button" @click="saveAddon(addon)">Сохранить</button>
      </div>
      <Textarea v-model="addon.description" rows="2" auto-resize class="plan-tagline" />
      <div class="price-row">
        <label class="field">
          <span class="gw-sub">Цена за месяц, руб.</span>
          <InputNumber :model-value="addon.price_month / 100" :min="0" @update:model-value="v => addon.price_month = Math.round((v || 0) * 100)" />
        </label>
        <label class="field">
          <span class="gw-sub">Цена за год, руб.</span>
          <InputNumber :model-value="addon.price_year / 100" :min="0" @update:model-value="v => addon.price_year = Math.round((v || 0) * 100)" />
        </label>
        <label class="field">
          <span class="gw-sub">{{ amountLabel(addon.kind) }}</span>
          <InputNumber :model-value="amountValue(addon)" :min="1" @update:model-value="v => setAmount(addon, v)" />
        </label>
      </div>
    </section>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import InputSwitch from 'primevue/inputswitch'
import Textarea from 'primevue/textarea'
import * as api from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatBytes, formatCount } from '@/utils/money.js'

const notif = useNotificationsStore()

const plans = ref([])
const addons = ref([])
const planLimits = ref({})
const settings = ref({ commission_pct: 10, payment_provider: 'manual', payment_enabled: false, store_enabled: true })

const GIB = 1024 ** 3

onMounted(load)

async function load() {
  const data = await api.adminOverview()
  plans.value = data.plans ?? []
  addons.value = data.addons ?? []
  planLimits.value = data.plan_limits ?? {}
  settings.value = data.settings ?? settings.value
}

function limitsSummary(code) {
  const l = planLimits.value[code]
  if (!l) return ''
  const n = (v) => (v == null || v < 0 ? '∞' : formatCount(v))
  return [
    `задачи ${n(l.tasks)}`,
    `компании ${n(l.companies)}`,
    `человек ${n(l.members)}`,
    `место ${l.storage_bytes < 0 ? '∞' : formatBytes(l.storage_bytes)}`,
    `токены ${n(l.ai_tokens)}`,
    `календари ${n(l.calendars)}`,
    `ежедневники ${n(l.diaries)}`,
    `доски ${n(l.boards)}`,
    `реестры ${n(l.registries)}`,
  ].join(' · ')
}

function amountLabel(kind) {
  if (kind === 'storage') return 'Объём, Гб'
  if (kind === 'tokens') return 'Токенов в пачке'
  return 'Сколько добавляет'
}

function amountValue(addon) {
  return addon.kind === 'storage' ? Math.round(addon.amount / GIB) : addon.amount
}

function setAmount(addon, value) {
  addon.amount = addon.kind === 'storage' ? Math.round((value || 0) * GIB) : (value || 0)
}

async function savePlan(plan) {
  try {
    await api.adminUpdatePlan(plan.code, plan)
    notif.notify({ severity: 'success', summary: 'Тариф сохранён', life: 3000 })
  } catch (e) {
    notif.notify({ severity: 'error', summary: 'Не сохранилось', detail: e?.data?.message || '', life: 5000 })
  }
}

async function saveAddon(addon) {
  try {
    await api.adminUpdateAddon(addon.code, addon)
    notif.notify({ severity: 'success', summary: 'Дополнение сохранено', life: 3000 })
  } catch (e) {
    notif.notify({ severity: 'error', summary: 'Не сохранилось', detail: e?.data?.message || '', life: 5000 })
  }
}

async function saveSettings() {
  try {
    settings.value = await api.adminUpdateSettings(settings.value)
    notif.notify({ severity: 'success', summary: 'Настройки сохранены', life: 3000 })
  } catch (e) {
    notif.notify({ severity: 'error', summary: 'Не сохранилось', detail: e?.data?.message || '', life: 5000 })
  }
}
</script>

<style scoped>
.tab { display: flex; flex-direction: column; gap: 14px; }
.row-main { flex: 1; min-width: 0; }
.num { max-width: 130px; }
.toggles { flex-wrap: wrap; gap: 16px; margin-top: 12px; }
.toggle { display: flex; align-items: center; gap: 8px; font-size: 0.88rem; }
.plan { display: flex; flex-direction: column; gap: 10px; }
.plan-head { display: flex; flex-wrap: wrap; align-items: center; gap: 12px; }
.plan-name { flex: 1; min-width: 160px; font-weight: 600; }
.plan-tagline { width: 100%; }
.price-row { display: flex; flex-wrap: wrap; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.limits { line-height: 1.5; }
</style>
