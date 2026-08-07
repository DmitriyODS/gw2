<template>
  <!-- Журнал действий супер-админа: тарифы, подписки, токены, промокоды,
       модерация товаров, подтверждение оплат. -->
  <EmptyState
    v-if="!items.length"
    icon="history"
    title="Записей пока нет"
    subtitle="Здесь останется след каждого действия с деньгами и тарифами."
  />

  <AppStack v-else :gap="10">
    <AppRow v-for="e in items" :key="e.id" :title="e.summary || e.action">
      <template #hint>
        {{ e.actor_name || 'система' }} · {{ e.action }}
        <template v-if="e.target_kind"> · {{ e.target_kind }} {{ e.target_id }}</template>
        · {{ formatUntil(e.created_at) }}
      </template>
    </AppRow>

    <AppButton
      v-if="items.length < total"
      class="more"
      label="Показать ещё"
      size="sm"
      @click="loadMore"
    />
  </AppStack>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import * as api from '@/api/billing.js'
import { formatUntil } from '@/utils/money.js'

const items = ref([])
const total = ref(0)

onMounted(() => load(true))

async function load(reset = false) {
  const offset = reset ? 0 : items.value.length
  const res = await api.adminAudit({ limit: 50, offset })
  items.value = reset ? (res.items ?? []) : [...items.value, ...(res.items ?? [])]
  total.value = res.total ?? items.value.length
}

function loadMore() {
  return load(false)
}
</script>

<style scoped>
.more { align-self: center; }
</style>
