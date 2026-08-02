<template>
  <!-- «Аудит платформы» — единая точка управления платформой для супер-админа:
       тарифы и цены, подписки пользователей, промокоды, товары магазина,
       заказы и выплаты, журнал действий. Компании, пользователи и резервная
       копия открываются своими разделами — они живут отдельными маршрутами. -->
  <div class="audit">
    <AppTabs variant="tint" v-model="tab" :tabs="TABS" />

    <AuditPlans v-if="tab === 'plans'" />
    <AuditSubscriptions v-else-if="tab === 'subs'" />
    <AuditPromos v-else-if="tab === 'promos'" />
    <AuditProducts v-else-if="tab === 'products'" />
    <AuditOrders v-else-if="tab === 'orders'" />
    <AuditAi v-else-if="tab === 'ai'" />
    <AuditLog v-else-if="tab === 'log'" />

    <div v-else-if="tab === 'data'" class="links">
      <AppRow
        v-for="link in DATA_LINKS"
        :key="link.to"
        :title="link.title"
        :hint="link.desc"
        clickable
        @click="go(link.to)"
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import AppTabs from '@/components/ui/AppTabs.vue'
import AppRow from '@/components/ui/AppRow.vue'
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
  { to: '/companies', title: 'Компании платформы', desc: 'Все компании, доступ и удаление' },
  { to: '/users', title: 'Пользователи платформы', desc: 'Учётные записи, блокировка и сброс пароля' },
  { to: '/settings?section=backup', title: 'Резервная копия', desc: 'Экспорт и восстановление базы данных' },
]

const tab = ref('plans')

function go(to) {
  router.push(to).catch(() => {})
}
</script>

<style scoped>
.audit { display: flex; flex-direction: column; gap: 14px; }
.links { display: flex; flex-direction: column; gap: 10px; }
</style>
