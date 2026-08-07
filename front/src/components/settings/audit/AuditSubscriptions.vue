<template>
  <!-- Подписки пользователей: выдать тариф, снять его, начислить или обнулить
       токены ИИ. Всё это пишется в журнал действий. -->
  <AppStack :gap="14">
    <AppCard title="Выдать тариф">
      <div class="grant-row">
        <div class="field user-field">
          <span class="field-label">Пользователь</span>
          <InputText v-model="query" placeholder="Имя или логин" @input="searchUsers" />
          <ul v-if="found.length" class="suggest">
            <li v-for="u in found" :key="u.id">
              <button type="button" @click="pick(u)">{{ u.fio }} · {{ u.login }}</button>
            </li>
          </ul>
          <p v-if="picked" class="field-label">Выбран: {{ picked.fio }}</p>
        </div>
        <label class="field">
          <span class="field-label">Тариф</span>
          <Dropdown v-model="grantPlan" :options="PLANS" option-label="label" option-value="value" />
        </label>
        <label class="field">
          <span class="field-label">Дней</span>
          <InputNumber v-model="grantDays" :min="1" :max="3650" />
        </label>
        <AppButton label="Выдать" variant="filled" :disabled="!picked" @click="grant" />
      </div>

      <div class="grant-row">
        <label class="field">
          <span class="field-label">Токены ИИ (можно отрицательные)</span>
          <InputNumber v-model="tokens" :min="-1000000" :max="1000000" />
        </label>
        <AppButton label="Начислить" size="sm" :disabled="!picked" @click="grantTokens" />
        <AppButton
          label="Обнулить токены"
          size="sm"
          tone="danger"
          :disabled="!picked"
          @click="resetTokens"
        />
      </div>
    </AppCard>

    <AppStack row :gap="10">
      <InputText v-model="search" placeholder="Поиск по подпискам" @keyup.enter="load" />
      <Dropdown v-model="planFilter" :options="FILTERS" option-label="label" option-value="value" @change="load" />
      <AppButton label="Обновить" icon="refresh" size="sm" @click="load" />
    </AppStack>

    <EmptyState v-if="!items.length" icon="card_membership" size="sm" title="Платных подписок пока нет" />
    <AppStack v-else :gap="10">
      <AppRow
        v-for="s in items"
        :key="s.user_id"
        :title="`${s.user_name || `#${s.user_id}`} · ${s.user_login}`"
      >
        <template #hint>
          {{ s.plan_code }} · {{ SOURCES[s.source] || s.source }} ·
          {{ s.expires_at ? `до ${formatUntil(s.expires_at)}` : 'бессрочно' }} ·
          место {{ formatBytes(s.storage_used) }}
        </template>
        <AppButton label="Снять" size="sm" tone="danger" @click="revoke(s.user_id)" />
      </AppRow>
    </AppStack>
  </AppStack>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Dropdown from 'primevue/dropdown'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'
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
.grant-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label { font-size: 0.85rem; color: var(--color-text-dim); }
.user-field { position: relative; flex: 1; min-width: 220px; }

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
