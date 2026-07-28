<template>
  <!-- «Аудит платформы» — единая точка управления платформой для супер-админа:
       тарифы и цены, подписки пользователей, промокоды, товары магазина,
       заказы и выплаты, журнал действий. Компании, пользователи и резервная
       копия открываются своими разделами — они живут отдельными маршрутами. -->
  <div class="audit">
    <PillTabs v-model="tab" :tabs="TABS" />

    <AuditPlans v-if="tab === 'plans'" />
    <AuditSubscriptions v-else-if="tab === 'subs'" />
    <AuditPromos v-else-if="tab === 'promos'" />
    <AuditProducts v-else-if="tab === 'products'" />
    <AuditOrders v-else-if="tab === 'orders'" />
    <AuditAi v-else-if="tab === 'ai'" />
    <AuditLog v-else-if="tab === 'log'" />

    <div v-else-if="tab === 'data'" class="links">
      <button
        v-for="link in DATA_LINKS"
        :key="link.to"
        class="gw-card gw-row link"
        type="button"
        @click="go(link.to)"
      >
        <span class="gw-row-icon"><span class="material-symbols-outlined">{{ link.icon }}</span></span>
        <div class="link-main">
          <p class="gw-h">{{ link.title }}</p>
          <p class="gw-sub">{{ link.desc }}</p>
        </div>
        <span class="material-symbols-outlined">chevron_right</span>
      </button>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import PillTabs from '@/components/common/PillTabs.vue'
import AuditPlans from '@/components/settings/audit/AuditPlans.vue'
import AuditSubscriptions from '@/components/settings/audit/AuditSubscriptions.vue'
import AuditPromos from '@/components/settings/audit/AuditPromos.vue'
import AuditProducts from '@/components/settings/audit/AuditProducts.vue'
import AuditOrders from '@/components/settings/audit/AuditOrders.vue'
import AuditLog from '@/components/settings/audit/AuditLog.vue'
import AuditAi from '@/components/settings/audit/AuditAi.vue'

const router = useRouter()

const TABS = [
  { key: 'plans', label: 'Тарифы и цены', icon: 'sell' },
  { key: 'subs', label: 'Подписки', icon: 'card_membership' },
  { key: 'promos', label: 'Промокоды', icon: 'local_activity' },
  { key: 'products', label: 'Товары', icon: 'inventory_2' },
  { key: 'orders', label: 'Заказы', icon: 'receipt_long' },
  { key: 'ai', label: 'ИИ возможности', icon: 'smart_toy' },
  { key: 'data', label: 'Данные', icon: 'database' },
  { key: 'log', label: 'Журнал', icon: 'history' },
]

const DATA_LINKS = [
  { to: '/companies', icon: 'business_center', title: 'Компании платформы', desc: 'Все компании, доступ и удаление' },
  { to: '/users', icon: 'group', title: 'Пользователи платформы', desc: 'Учётные записи, блокировка и сброс пароля' },
  { to: '/settings?section=backup', icon: 'backup', title: 'Резервная копия', desc: 'Экспорт и восстановление базы данных' },
]

const tab = ref('plans')

function go(to) {
  router.push(to).catch(() => {})
}
</script>

<style scoped>
.audit { display: flex; flex-direction: column; gap: 14px; }
.links { display: flex; flex-direction: column; gap: 10px; }
.link { padding: 14px; text-align: left; cursor: pointer; }
.link-main { flex: 1; min-width: 0; }
</style>
