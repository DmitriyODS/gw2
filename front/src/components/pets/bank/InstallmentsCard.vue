<template>
  <section class="kb-card ic-card">
    <header class="kb-card-head">
      <span class="kb-card-icon kb-card-icon--violet material-symbols-outlined">splitscreen</span>
      <h3 class="kb-card-title ic-title">Оплата частями</h3>
      <button class="ic-help" type="button" aria-label="Как работает оплата частями" @click="helpOpen = true">
        <span class="material-symbols-outlined">help</span>
      </button>
      <span v-if="data" class="kb-badge kb-badge--violet ic-avail">{{ data.available }} свободно</span>
    </header>

    <div v-if="data" class="ic-gauge">
      <div class="ic-gauge-bar"><div class="ic-gauge-fill" :style="{ width: usedPercent + '%' }"></div></div>
      <span class="ic-gauge-label">Занято {{ data.used }} из {{ data.limit }}</span>
    </div>

    <div v-if="data && data.items.length" class="ic-list">
      <div v-for="i in data.items" :key="i.id" class="ic-item" :class="{ overdue: i.overdue }">
        <div class="ic-item-top">
          <span class="ic-item-title"><EmojiGlyph :char="itemEmoji(i)" /> {{ itemLabel(i) }}</span>
          <span class="ic-item-out"><KudosCoin /> {{ i.outstanding }}</span>
        </div>
        <div class="ic-item-bar"><div class="ic-item-fill" :style="{ width: paidPercent(i) + '%' }"></div></div>
        <div class="ic-item-meta">
          <span v-if="i.overdue" class="ic-overdue"><span class="material-symbols-outlined">error</span> Просрочка — долг растёт</span>
          <span v-else>Платёж до {{ formatDue(i.due_at) }}</span>
          <span class="ic-item-progress">{{ i.paid }} / {{ i.total }}</span>
        </div>
        <div class="ic-item-actions">
          <button class="btn-glass ic-btn" :disabled="busy || walletShort(i.part_amount)" @click="pay(i, i.part_amount)">
            Доля {{ i.part_amount }}
          </button>
          <button class="btn-grad ic-btn" :disabled="busy || walletShort(i.outstanding)" @click="pay(i, i.outstanding)">
            Погасить {{ i.outstanding }}
          </button>
        </div>
      </div>
    </div>

    <p v-else-if="data" class="ic-empty">Активных рассрочек нет. Выберите «Частями» при покупке в магазине или домике.</p>

    <AppDialog v-model="helpOpen" title="Оплата частями" icon="splitscreen" tone="primary" size="sm">
      <ul class="ic-help-list">
        <li><b>Кредитный счёт на {{ data ? data.limit : 500 }} кудосов.</b> Любой не-акционный товар можно взять сейчас и оплачивать долями.</li>
        <li>Покупка делится на <b>{{ data ? data.parts : 4 }} части</b> — вносите их когда удобно.</li>
        <li>Платёж нужен <b>минимум раз в неделю</b>. Пропустили неделю — на остаток капает <b>+20%</b>.</li>
        <li>Пока счёт не погашен, новые покупки в рассрочку доступны в пределах свободного лимита.</li>
        <li>Акционные товары в рассрочку нельзя — их берут сразу или в кредит.</li>
      </ul>
    </AppDialog>
  </section>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import { usePetsStore } from '@/stores/pets'
import { useNotificationsStore } from '@/stores/notifications'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
import AppDialog from '@/components/common/AppDialog.vue'
import { SHOP_ITEMS, DECOR_ITEMS } from '@/utils/pets'

const pets = usePetsStore()
const notify = useNotificationsStore()
const busy = ref(false)
const helpOpen = ref(false)

const data = computed(() => pets.installments)

const usedPercent = computed(() => {
  const d = data.value
  if (!d?.limit) return 0
  return Math.min(100, Math.round((d.used / d.limit) * 100))
})

const paidPercent = (i) => (i.total ? Math.round((i.paid / i.total) * 100) : 0)
const walletShort = (amount) => (pets.bank?.kudos ?? pets.pet?.kudos ?? 0) < amount

function catalog(i) {
  return i.category === 'house' ? DECOR_ITEMS[i.item_key] : SHOP_ITEMS[i.item_key]
}
function itemEmoji(i) {
  return catalog(i)?.emoji || '🛍️'
}
function itemLabel(i) {
  return catalog(i)?.title || i.item_title
}
function formatDue(iso) {
  return iso ? new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' }) : ''
}

async function pay(i, amount) {
  busy.value = true
  try {
    await pets.payInstallment(i.id, amount)
  } catch (e) {
    notify.error(e?.message || 'Платёж не прошёл')
  } finally {
    busy.value = false
  }
}

onMounted(() => pets.fetchInstallments().catch(() => {}))
</script>

<style scoped>
/* Карточка/шапка/иконка/бейдж: дублируем стили KudosBankView, т.к. те scoped и
   на этот отдельный компонент не распространяются (иначе шапка не flex-строка). */
.kb-card {
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 20px);
  padding: 18px 20px;
  display: flex; flex-direction: column; gap: 12px; min-width: 0;
}
.kb-card-head { display: flex; align-items: center; gap: 10px; }
.kb-card-icon {
  width: 42px; height: 42px; border-radius: 14px;
  display: grid; place-items: center; font-size: 22px; flex-shrink: 0;
}
.kb-card-icon--violet { background: var(--color-primary-container); color: var(--color-primary); }
.kb-card-title { margin: 0; font-size: 16px; font-weight: 800; }
.kb-badge {
  font-size: 12px; font-weight: 700; border-radius: var(--radius-full);
  padding: 4px 11px; background: color-mix(in oklch, var(--color-text) 7%, transparent);
  color: var(--color-text-dim); white-space: nowrap;
}
.kb-badge--violet { background: var(--color-primary-container); color: var(--color-on-primary-container); }
.kb-card-hint { margin: 0; font-size: 12.5px; color: var(--color-text-dim); line-height: 1.45; }

.ic-help {
  display: inline-flex; align-items: center; justify-content: center;
  width: 24px; height: 24px; min-width: 24px; min-height: 24px;
  border-radius: var(--radius-full); border: none; cursor: pointer;
  background: color-mix(in oklch, var(--color-primary) 12%, transparent);
  color: var(--color-primary);
}
.ic-help:hover { background: color-mix(in oklch, var(--color-primary) 22%, transparent); }
.ic-help .material-symbols-outlined { font-size: 16px; }
/* Название по своей ширине, чтобы «?» встал сразу справа от него, не уезжая вниз. */
.ic-title { flex: 0 0 auto; }
.ic-avail { margin-left: auto; }

.ic-help-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 9px; }
.ic-help-list li { font-size: 13.5px; line-height: 1.5; color: var(--color-text); }

.ic-gauge { margin: 12px 0 4px; }
.ic-gauge-bar {
  height: 8px; border-radius: var(--radius-full); overflow: hidden;
  background: color-mix(in oklch, var(--color-text) 10%, transparent);
}
.ic-gauge-fill { height: 100%; border-radius: inherit; background: linear-gradient(90deg, var(--color-primary), var(--color-tertiary)); }
.ic-gauge-label { display: block; margin-top: 4px; font-size: 12px; color: var(--color-text-dim); }

.ic-list { display: flex; flex-direction: column; gap: 10px; margin-top: 12px; }
.ic-item {
  padding: 12px; border-radius: var(--radius-md);
  border: 1px solid var(--acrylic-border);
  background: color-mix(in oklch, var(--color-primary) 5%, transparent);
}
.ic-item.overdue { border-color: color-mix(in oklch, var(--color-error) 40%, transparent); }
.ic-item-top { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
.ic-item-title { display: inline-flex; align-items: center; gap: 6px; font-weight: 600; font-size: 13.5px; }
.ic-item-out { display: inline-flex; align-items: center; gap: 3px; font-weight: 700; }
.ic-item-bar {
  height: 5px; border-radius: var(--radius-full); overflow: hidden; margin: 8px 0 6px;
  background: color-mix(in oklch, var(--color-text) 10%, transparent);
}
.ic-item-fill { height: 100%; background: var(--color-success); border-radius: inherit; }
.ic-item-meta { display: flex; justify-content: space-between; font-size: 12px; color: var(--color-text-dim); }
.ic-overdue { display: inline-flex; align-items: center; gap: 4px; color: var(--color-error); }
.ic-overdue .material-symbols-outlined { font-size: 15px; }
.ic-item-actions { display: flex; gap: 8px; margin-top: 10px; }
.ic-btn { flex: 1; min-height: 34px; font-size: 13px; }
.ic-empty { margin-top: 12px; font-size: 13px; color: var(--color-text-dim); }
</style>
