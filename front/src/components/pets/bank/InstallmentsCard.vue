<template>
  <!-- Каркас — общий AppCard: своя копия `.kb-card*` разъезжалась с разделом
       (шапка без переноса давала горизонтальную прокрутку на телефоне). -->
  <AppCard title="Оплата частями" :gap="12">
    <template #head>
      <AppButton
        variant="icon"
        size="sm"
        icon="help"
        aria-label="Как работает оплата частями"
        @click="helpOpen = true"
      />
      <AppChip v-if="data" tone="primary" size="sm" :label="`${data.available} свободно`" />
    </template>

    <div v-if="data" class="ic-gauge">
      <div class="ic-gauge-bar"><div class="ic-gauge-fill" :style="{ width: usedPercent + '%' }"></div></div>
      <span class="ic-gauge-label">Занято {{ data.used }} из {{ data.limit }}</span>
    </div>

    <AppStack v-if="data && data.items.length" :gap="10">
      <div v-for="i in data.items" :key="i.id" class="ic-item" :class="{ overdue: i.overdue }">
        <div class="ic-item-top">
          <span class="ic-item-title"><EmojiGlyph :char="itemEmoji(i)" /> {{ itemLabel(i) }}</span>
          <span class="ic-item-out"><KudosCoin /> {{ i.outstanding }}</span>
        </div>
        <div class="ic-item-bar"><div class="ic-item-fill" :style="{ width: paidPercent(i) + '%' }"></div></div>
        <div class="ic-item-meta">
          <span v-if="i.overdue" class="ic-overdue"><span class="material-symbols-outlined">error</span> Просрочка — долг растёт</span>
          <span v-else>Платёж до {{ formatDue(i.due_at) }}</span>
          <span>{{ i.paid }} / {{ i.total }}</span>
        </div>
        <AppStack row :gap="8" class="ic-item-actions">
          <AppButton
            size="sm"
            class="ic-btn"
            :disabled="busy || walletShort(i.part_amount)"
            :label="`Доля ${i.part_amount}`"
            @click="pay(i, i.part_amount)"
          />
          <AppButton
            variant="filled"
            size="sm"
            class="ic-btn"
            :disabled="busy || walletShort(i.outstanding)"
            :label="`Погасить ${i.outstanding}`"
            @click="pay(i, i.outstanding)"
          />
        </AppStack>
      </div>
    </AppStack>

    <p v-else-if="data" class="ic-empty">Активных рассрочек нет. Выберите «Частями» при покупке в магазине или домике.</p>

    <AppDialog v-model="helpOpen" title="Оплата частями" tone="primary" size="sm">
      <ul class="ic-help-list">
        <li><b>Кредитный счёт на {{ data ? data.limit : 500 }} кудосов.</b> Любой не-акционный товар можно взять сейчас и оплачивать долями.</li>
        <li>Покупка делится на <b>{{ data ? data.parts : 4 }} части</b> — вносите их когда удобно.</li>
        <li>Платёж нужен <b>минимум раз в неделю</b>. Пропустили неделю — на остаток капает <b>+20%</b>.</li>
        <li>Пока счёт не погашен, новые покупки в рассрочку доступны в пределах свободного лимита.</li>
        <li>Акционные товары в рассрочку нельзя — их берут сразу или в кредит.</li>
      </ul>
    </AppDialog>
  </AppCard>
</template>

<script setup>
import { computed, onMounted, ref } from 'vue'
import AppButton from '@/components/ui/AppButton.vue'
import AppCard from '@/components/ui/AppCard.vue'
import AppChip from '@/components/ui/AppChip.vue'
import AppStack from '@/components/ui/AppStack.vue'
import AppDialog from '@/components/ui/AppDialog.vue'
import { usePetsStore } from '@/stores/pets'
import { useNotificationsStore } from '@/stores/notifications'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import EmojiGlyph from '@/components/common/EmojiGlyph.vue'
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
/* Только специфика рассрочки: шкала свободного лимита и строка покупки.
   Каркас, шапка, чип и кнопки — из ядра `components/ui`. */
.ic-help-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 9px; }
.ic-help-list li { font-size: 13.5px; line-height: 1.5; color: var(--color-text); }

.ic-gauge-bar {
  height: 8px; border-radius: var(--radius-full); overflow: hidden;
  background: color-mix(in oklch, var(--color-text) 10%, transparent);
}
.ic-gauge-fill { height: 100%; border-radius: inherit; background: linear-gradient(90deg, var(--color-primary), var(--color-tertiary)); }
.ic-gauge-label { display: block; margin-top: 4px; font-size: 12px; color: var(--color-text-dim); }

.ic-item {
  padding: 12px; border-radius: var(--radius-md);
  border: 1px solid var(--acrylic-border);
  background: color-mix(in oklch, var(--color-primary) 5%, transparent);
}
.ic-item.overdue { border-color: color-mix(in oklch, var(--color-error) 40%, transparent); }
.ic-item-top { display: flex; justify-content: space-between; align-items: center; gap: 8px; }
/* Название сжимается, сумма — нет: иначе длинное имя товара распирает карточку. */
.ic-item-title { display: inline-flex; align-items: center; gap: 6px; min-width: 0; font-weight: 600; font-size: 13.5px; overflow-wrap: anywhere; }
.ic-item-out { display: inline-flex; align-items: center; gap: 3px; flex-shrink: 0; font-weight: 700; }
.ic-item-bar {
  height: 5px; border-radius: var(--radius-full); overflow: hidden; margin: 8px 0 6px;
  background: color-mix(in oklch, var(--color-text) 10%, transparent);
}
.ic-item-fill { height: 100%; background: var(--color-success); border-radius: inherit; }
.ic-item-meta { display: flex; justify-content: space-between; flex-wrap: wrap; gap: 4px 10px; font-size: 12px; color: var(--color-text-dim); }
.ic-overdue { display: inline-flex; align-items: center; gap: 4px; color: var(--color-error); }
.ic-overdue .material-symbols-outlined { font-size: 15px; }
.ic-item-actions { margin-top: 10px; }
.ic-btn { flex: 1 1 auto; min-width: 0; }
.ic-empty { margin: 0; font-size: 13px; color: var(--color-text-dim); }
</style>
