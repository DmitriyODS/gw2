<template>
  <!-- Журнал действий супер-админа: тарифы, подписки, токены, промокоды,
       модерация товаров, подтверждение оплат. -->
  <div class="tab">
    <section v-if="!items.length" class="gw-banner">
      <h2>Записей пока нет</h2>
      <p class="gw-sub">Здесь останется след каждого действия с деньгами и тарифами.</p>
    </section>

    <div v-else class="rows">
      <article v-for="e in items" :key="e.id" class="gw-card gw-row row">
        <span class="gw-row-icon"><span class="material-symbols-outlined">history</span></span>
        <div class="row-main">
          <p class="gw-h">{{ e.summary || e.action }}</p>
          <p class="gw-sub">
            {{ e.actor_name || 'система' }} · {{ e.action }}
            <template v-if="e.target_kind"> · {{ e.target_kind }} {{ e.target_id }}</template>
            · {{ formatUntil(e.created_at) }}
          </p>
        </div>
      </article>

      <button v-if="items.length < total" class="gw-chip more" type="button" @click="loadMore">
        Показать ещё
      </button>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
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
.tab { display: flex; flex-direction: column; gap: 14px; }
.rows { display: flex; flex-direction: column; gap: 10px; }
.row { padding: 14px; }
.row-main { flex: 1; min-width: 0; }
.more { align-self: center; }
</style>
