<template>
  <!-- Подписки пользователей: выдать тариф, снять его, начислить или обнулить
       токены ИИ. Всё это пишется в журнал действий. -->
  <div class="tab">
    <section class="gw-card grant">
      <p class="gw-h">Выдать тариф</p>
      <div class="grant-row">
        <div class="field user-field">
          <span class="gw-sub">Пользователь</span>
          <InputText v-model="query" placeholder="Имя или логин" @input="searchUsers" />
          <ul v-if="found.length" class="suggest">
            <li v-for="u in found" :key="u.id">
              <button type="button" @click="pick(u)">{{ u.fio }} · {{ u.login }}</button>
            </li>
          </ul>
          <p v-if="picked" class="gw-sub">Выбран: {{ picked.fio }}</p>
        </div>
        <label class="field">
          <span class="gw-sub">Тариф</span>
          <Dropdown v-model="grantPlan" :options="PLANS" option-label="label" option-value="value" />
        </label>
        <label class="field">
          <span class="gw-sub">Дней</span>
          <InputNumber v-model="grantDays" :min="1" :max="3650" />
        </label>
        <button class="btn-grad" type="button" :disabled="!picked" @click="grant">Выдать</button>
      </div>

      <div class="grant-row">
        <label class="field">
          <span class="gw-sub">Токены ИИ (можно отрицательные)</span>
          <InputNumber v-model="tokens" :min="-1000000" :max="1000000" />
        </label>
        <button class="gw-chip" type="button" :disabled="!picked" @click="grantTokens">Начислить</button>
        <button class="gw-chip" type="button" :disabled="!picked" @click="resetTokens">Обнулить токены</button>
      </div>
    </section>

    <div class="filters">
      <InputText v-model="search" placeholder="Поиск по подпискам" @keyup.enter="load" />
      <Dropdown v-model="planFilter" :options="FILTERS" option-label="label" option-value="value" @change="load" />
      <button class="gw-chip" type="button" @click="load">Обновить</button>
    </div>

    <section v-if="!items.length" class="gw-banner">
      <h2>Платных подписок пока нет</h2>
    </section>
    <div v-else class="rows">
      <article v-for="s in items" :key="s.user_id" class="gw-card gw-row row">
        <span class="gw-row-icon"><span class="material-symbols-outlined">card_membership</span></span>
        <div class="row-main">
          <p class="gw-h">{{ s.user_name || `#${s.user_id}` }} <span class="gw-sub">· {{ s.user_login }}</span></p>
          <p class="gw-sub">
            {{ s.plan_code }} · {{ SOURCES[s.source] || s.source }} ·
            {{ s.expires_at ? `до ${formatUntil(s.expires_at)}` : 'бессрочно' }} ·
            место {{ formatBytes(s.storage_used) }}
          </p>
        </div>
        <button class="gw-chip" type="button" @click="revoke(s.user_id)">Снять</button>
      </article>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Dropdown from 'primevue/dropdown'
import * as api from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatBytes, formatUntil } from '@/utils/money.js'

const notif = useNotificationsStore()

const PLANS = [
  { value: 'junior', label: 'Джун' },
  { value: 'middle', label: 'Мидл' },
  { value: 'senior', label: 'Синьор' },
]

const FILTERS = [{ value: '', label: 'Все тарифы' }, ...PLANS]

const SOURCES = { purchase: 'оплачен', grant: 'выдан', grace: 'переходный период' }

const items = ref([])
const search = ref('')
const planFilter = ref('')

const query = ref('')
const found = ref([])
const picked = ref(null)
const grantPlan = ref('middle')
const grantDays = ref(30)
const tokens = ref(1000)

let searchTimer = null

onMounted(load)

async function load() {
  const res = await api.adminSubscriptions({ search: search.value, plan: planFilter.value })
  items.value = res.items ?? []
}

function searchUsers() {
  clearTimeout(searchTimer)
  searchTimer = setTimeout(async () => {
    if (!query.value.trim()) {
      found.value = []
      return
    }
    const res = await api.adminSearchUsers(query.value.trim())
    found.value = res.items ?? []
  }, 300)
}

function pick(u) {
  picked.value = u
  query.value = u.fio
  found.value = []
}

async function grant() {
  try {
    await api.adminGrantSubscription({
      user_id: picked.value.id,
      plan: grantPlan.value,
      days: grantDays.value,
      note: 'Выдано из «Аудита платформы»',
    })
    notif.notify({ severity: 'success', summary: 'Тариф выдан', life: 3000 })
    await load()
  } catch (e) {
    notif.notify({ severity: 'error', summary: 'Не получилось', detail: e?.data?.message || '', life: 5000 })
  }
}

async function revoke(userId) {
  await api.adminRevokeSubscription(userId)
  notif.notify({ severity: 'info', summary: 'Подписка снята', life: 3000 })
  await load()
}

async function grantTokens() {
  await api.adminGrantTokens(picked.value.id, tokens.value)
  notif.notify({ severity: 'success', summary: 'Токены начислены', life: 3000 })
}

async function resetTokens() {
  await api.adminResetTokens(picked.value.id)
  notif.notify({ severity: 'info', summary: 'Токены обнулены', life: 3000 })
}
</script>

<style scoped>
.tab { display: flex; flex-direction: column; gap: 14px; }
.grant { display: flex; flex-direction: column; gap: 12px; }
.grant-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.user-field { position: relative; flex: 1; min-width: 220px; }
.filters { display: flex; flex-wrap: wrap; gap: 10px; }
.rows { display: flex; flex-direction: column; gap: 10px; }
.row { padding: 14px; }
.row-main { flex: 1; min-width: 0; }

.suggest {
  position: absolute;
  top: 100%;
  z-index: 2;
  width: 100%;
  margin: 4px 0 0;
  padding: 4px;
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  list-style: none;
  box-shadow: var(--shadow-md);
}

.suggest button {
  width: 100%;
  padding: 8px 10px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  font-size: 0.85rem;
  text-align: left;
  cursor: pointer;
}

.suggest button:hover { background: var(--color-surface-variant); }
</style>
