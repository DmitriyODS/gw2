<template>
  <!-- Магазин оформления. Витрины ещё нет: раздел выкатан каркасом, чтобы
       занять своё место в рабочем столе и в разделе «Темы» — товары,
       покупки и подписки появятся отдельной итерацией.
       Каркас собран из общих классов оформления (main.css, .gw-*). -->
  <div class="gw-shell store">
    <PillTabs v-model="tab" :tabs="TABS" />

    <div class="gw-panel gw-panel-body store-body">
      <section class="gw-banner">
        <h2>{{ BANNERS[tab] }}</h2>
        <p v-if="tab === 'shop'" class="gw-sub">Витрина ещё наполняется — загляните чуть позже.</p>
      </section>

      <template v-if="tab === 'shop'">
        <div class="store-cats">
          <button
            v-for="cat in CATEGORIES"
            :key="cat.key"
            class="gw-tile"
            type="button"
            @click="soon(cat.label)"
          >
            <span class="material-symbols-outlined">{{ cat.icon }}</span>
            <span>{{ cat.label }}</span>
          </button>
        </div>

        <h3 class="gw-h">Рекомендуем</h3>
        <div class="store-grid">
          <button
            v-for="n in 10"
            :key="n"
            class="gw-tile"
            type="button"
            @click="soon('Товары')"
          >
            <span class="material-symbols-outlined">redeem</span>
            <span>Template</span>
          </button>
        </div>
      </template>
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue'
import PillTabs from '@/components/common/PillTabs.vue'
import { useNotificationsStore } from '@/stores/notifications.js'

const notif = useNotificationsStore()

const TABS = [
  { key: 'shop', label: 'Магазин', icon: 'storefront' },
  { key: 'mine', label: 'Мои товары', icon: 'sell' },
  { key: 'orders', label: 'Заказы', icon: 'receipt_long' },
  { key: 'subs', label: 'Подписки', icon: 'card_membership' },
]

const BANNERS = {
  shop: 'Скоро здесь будет что-то интересное',
  mine: 'Купленных товаров пока нет',
  orders: 'Заказов пока нет',
  subs: 'Подписок пока нет',
}

const CATEGORIES = [
  { key: 'themes', label: 'Темы', icon: 'palette' },
  { key: 'wallpapers', label: 'Обои', icon: 'image' },
  { key: 'gradients', label: 'Градиенты', icon: 'gradient' },
  { key: 'pets', label: 'Питомцу', icon: 'pets' },
]

const tab = ref('shop')

function soon(what) {
  notif.notify({
    severity: 'info',
    summary: what,
    detail: 'Витрина ещё готовится — товары появятся в одном из ближайших обновлений.',
    life: 4000,
  })
}
</script>

<style scoped>
.store-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Ряды плиток меряют ширину ОКНА раздела (.gw-shell — container). */
.store-cats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}

.store-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  gap: 12px;
  padding-bottom: 4px;
}

@container (max-width: 720px) {
  .store-cats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

/* Старый WebView (chrome87) не знает @container — для телефонов дублируем
   правило обычным media-запросом: там окно всё равно во весь экран. */
@media (max-width: 720px) {
  .store-cats { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}
</style>
