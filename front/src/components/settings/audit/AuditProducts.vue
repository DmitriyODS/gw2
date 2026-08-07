<template>
  <!-- Товары магазина: очередь модерации авторских товаров и позиции самой
       платформы. Опубликованное сразу попадает на витрину. -->
  <AppStack :gap="14">
    <AppCard title="Позиция платформы">
      <div class="form-row">
        <label class="field">
          <span class="field-label">Вид</span>
          <Dropdown v-model="form.kind" :options="KINDS" option-label="label" option-value="value" />
        </label>
        <label class="field grow">
          <span class="field-label">Название</span>
          <InputText v-model="form.title" />
        </label>
        <label class="field">
          <span class="field-label">Цена, руб.</span>
          <InputNumber v-model="form.priceRub" :min="0" />
        </label>
        <AppButton label="Добавить" icon="add" variant="filled" @click="create" />
      </div>
      <Textarea v-model="form.description" rows="2" auto-resize placeholder="Описание для витрины" />
    </AppCard>

    <EmptyState
      v-if="!items.length"
      icon="sell"
      size="sm"
      title="Товаров нет"
      subtitle="Здесь появятся товары авторов, отправленные на проверку."
    />
    <AppStack v-else :gap="10">
      <AppRow v-for="p in items" :key="p.id" :title="p.title">
        <template #hint>
          {{ p.author_name || 'платформа' }} · {{ formatPrice(p.price) }} · продаж {{ p.sales_count }}
          <template v-if="p.reject_reason"> · {{ p.reject_reason }}</template>
        </template>
        <AppChip :tone="tone(p.status)" size="sm" :label="STATUS[p.status] || p.status" />
        <template v-if="p.status === 'review'">
          <AppButton label="Опубликовать" size="sm" @click="review(p, true)" />
          <AppButton label="Отклонить" size="sm" tone="danger" @click="reject(p)" />
        </template>
        <AppButton
          v-else-if="p.status === 'published'"
          label="Снять"
          size="sm"
          tone="neutral"
          @click="remove(p.id)"
        />
      </AppRow>
    </AppStack>
  </AppStack>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Textarea from 'primevue/textarea'
import Dropdown from 'primevue/dropdown'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppRow from '@/components/ui/AppRow.vue'
import AppStack from '@/components/ui/AppStack.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import * as api from '@/api/billing.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { formatPrice } from '@/utils/money.js'

const notif = useNotificationsStore()

const KINDS = [
  { value: 'theme', label: 'Тема' },
  { value: 'wallpaper', label: 'Обои' },
  { value: 'gradient', label: 'Градиент' },
  { value: 'pet_skin', label: 'Скин питомца' },
  { value: 'pet_decor', label: 'Декор домика' },
  { value: 'other', label: 'Другое' },
]

const STATUS = {
  draft: 'Черновик', review: 'На проверке', published: 'На витрине',
  rejected: 'Отклонён', removed: 'Снят',
}

const items = ref([])
const form = ref({ kind: 'theme', title: '', description: '', priceRub: 199 })

onMounted(load)

async function load() {
  const res = await api.adminProducts()
  items.value = res.items ?? []
}

async function create() {
  if (!form.value.title.trim()) return
  await api.adminCreateProduct({
    kind: form.value.kind,
    title: form.value.title.trim(),
    description: form.value.description,
    price: Math.round((form.value.priceRub || 0) * 100),
    payload: {},
  })
  form.value = { kind: 'theme', title: '', description: '', priceRub: 199 }
  await load()
}

async function review(product, approve) {
  await api.adminReviewProduct(product.id, approve)
  notif.notify({ severity: 'success', summary: approve ? 'Опубликован' : 'Отклонён', life: 3000 })
  await load()
}

async function reject(product) {
  const reason = window.prompt('Причина отказа (её увидит автор)') || ''
  await api.adminReviewProduct(product.id, false, reason)
  await load()
}

async function remove(id) {
  await api.adminDeleteProduct(id)
  await load()
}

function tone(status) {
  if (status === 'published') return 'success'
  if (status === 'review') return 'warning'
  if (status === 'rejected') return 'error'
  return 'primary'
}
</script>

<style scoped>
.form-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.field-label { font-size: 0.85rem; color: var(--color-text-dim); }
.grow { flex: 1; min-width: 180px; }
</style>
