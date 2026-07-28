<template>
  <!-- Товары магазина: очередь модерации авторских товаров и позиции самой
       платформы. Опубликованное сразу попадает на витрину. -->
  <div class="tab">
    <section class="gw-card form">
      <p class="gw-h">Позиция платформы</p>
      <div class="form-row">
        <label class="field">
          <span class="gw-sub">Вид</span>
          <Dropdown v-model="form.kind" :options="KINDS" option-label="label" option-value="value" />
        </label>
        <label class="field grow">
          <span class="gw-sub">Название</span>
          <InputText v-model="form.title" />
        </label>
        <label class="field">
          <span class="gw-sub">Цена, руб.</span>
          <InputNumber v-model="form.priceRub" :min="0" />
        </label>
        <button class="btn-grad" type="button" @click="create">Добавить</button>
      </div>
      <Textarea v-model="form.description" rows="2" auto-resize placeholder="Описание для витрины" />
    </section>

    <section v-if="!items.length" class="gw-banner">
      <h2>Товаров нет</h2>
      <p class="gw-sub">Здесь появятся товары авторов, отправленные на проверку.</p>
    </section>
    <div v-else class="rows">
      <article v-for="p in items" :key="p.id" class="gw-card gw-row row">
        <span class="gw-row-icon"><span class="material-symbols-outlined">inventory_2</span></span>
        <div class="row-main">
          <p class="gw-h">{{ p.title }}</p>
          <p class="gw-sub">
            {{ p.author_name || 'платформа' }} · {{ formatPrice(p.price) }} · продаж {{ p.sales_count }}
            <template v-if="p.reject_reason"> · {{ p.reject_reason }}</template>
          </p>
        </div>
        <span class="chip-tint" :class="tone(p.status)">{{ STATUS[p.status] || p.status }}</span>
        <template v-if="p.status === 'review'">
          <button class="gw-chip" type="button" @click="review(p, true)">Опубликовать</button>
          <button class="gw-chip" type="button" @click="reject(p)">Отклонить</button>
        </template>
        <button v-else-if="p.status === 'published'" class="gw-chip" type="button" @click="remove(p.id)">
          Снять
        </button>
      </article>
    </div>
  </div>
</template>

<script setup>
import { onMounted, ref } from 'vue'
import InputText from 'primevue/inputtext'
import InputNumber from 'primevue/inputnumber'
import Textarea from 'primevue/textarea'
import Dropdown from 'primevue/dropdown'
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
  if (status === 'published') return 'chip-tint--success'
  if (status === 'review') return 'chip-tint--warning'
  if (status === 'rejected') return 'chip-tint--error'
  return 'chip-tint--primary'
}
</script>

<style scoped>
.tab { display: flex; flex-direction: column; gap: 14px; }
.form { display: flex; flex-direction: column; gap: 12px; }
.form-row { display: flex; flex-wrap: wrap; align-items: flex-end; gap: 12px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.grow { flex: 1; min-width: 180px; }
.rows { display: flex; flex-direction: column; gap: 10px; }
.row { padding: 14px; }
.row-main { flex: 1; min-width: 0; }
</style>
