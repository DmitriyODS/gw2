<template>
  <AppDialog
    :model-value="modelValue"
    title="Кудо-банк"
    :subtitle="subtitle"
    icon="account_balance"
    tone="primary"
    size="lg"
    @update:model-value="$emit('update:modelValue', $event)"
  >
    <div v-if="!bank" class="bank-loading"><ProgressSpinner style="width:32px;height:32px" /></div>
    <template v-else>
      <!-- Балансы + уровень -->
      <div class="bank-balances">
        <div class="bank-balance-card">
          <span class="bank-balance-label">Кошелёк</span>
          <span class="bank-balance-value"><KudosCoin /> {{ bank.kudos }}</span>
        </div>
        <div class="bank-balance-card">
          <span class="bank-balance-label">Вклад · {{ bank.tier.savings_rate_pct }}%/день</span>
          <span class="bank-balance-value"><KudosCoin /> {{ bank.savings }}</span>
        </div>
        <div class="bank-balance-card" :class="{ debt: bank.loan > 0 }">
          <span class="bank-balance-label">{{ bank.loan > 0 ? 'Долг по кредиту' : 'Кредит' }}</span>
          <span class="bank-balance-value">
            <template v-if="bank.loan > 0"><KudosCoin /> {{ bank.loan }}</template>
            <template v-else>нет</template>
          </span>
        </div>
      </div>

      <!-- Уровень клиента (loyalty-tier) с прогрессом -->
      <div class="bank-tier">
        <span class="bank-tier-badge">{{ TIER_EMOJI[bank.tier.key] || '⭐' }} {{ bank.tier.title }}</span>
        <template v-if="bank.next_tier">
          <div class="bank-tier-bar">
            <div class="bank-tier-fill" :style="{ width: tierPercent + '%' }"></div>
          </div>
          <span class="bank-tier-hint">
            {{ bank.earned }} / {{ bank.next_tier.threshold }} заработанных до уровня «{{ bank.next_tier.title }}»
          </span>
        </template>
        <span v-else class="bank-tier-hint">Максимальный уровень — лучшие условия банка</span>
      </div>

      <SegmentedTabs v-model="tab" :tabs="TABS" full-width dense />

      <!-- ── Обзор ── -->
      <div v-if="tab === 'overview'" class="bank-pane">
        <div class="bank-month">
          <span class="bank-month-item in">
            <span class="material-symbols-outlined">arrow_downward</span>
            +{{ bank.month_in }} за 30 дней
          </span>
          <span class="bank-month-item out">
            <span class="material-symbols-outlined">arrow_upward</span>
            −{{ bank.month_out }} за 30 дней
          </span>
        </div>

        <!-- Вклад -->
        <div class="bank-block">
          <h4 class="bank-block-title"><span class="material-symbols-outlined">savings</span> Вклад</h4>
          <p class="bank-block-hint">
            {{ bank.tier.savings_rate_pct }}% в день (не больше {{ bank.savings_daily_max }} кудосов/день).
            Проценты капают за каждые полные сутки.
            <template v-if="bank.loan > 0"> Пока есть долг, вклад закрыт.</template>
          </p>
          <div class="bank-row">
            <input
              v-model.number="savingsAmount"
              type="number" min="1" class="bank-input" placeholder="Сумма"
            />
            <button class="btn-glass" :disabled="busy || bank.loan > 0 || !validAmount(savingsAmount)" @click="deposit">
              Пополнить
            </button>
            <button class="btn-glass" :disabled="busy || !bank.savings || !validAmount(savingsAmount)" @click="withdraw">
              Снять
            </button>
          </div>
        </div>

        <!-- Кредит -->
        <div class="bank-block">
          <h4 class="bank-block-title"><span class="material-symbols-outlined">credit_score</span> Кредит</h4>
          <template v-if="bank.loan > 0">
            <p class="bank-block-hint">Остаток долга — {{ bank.loan }} кудосов. Погашение — с кошелька.</p>
            <div class="bank-row">
              <input v-model.number="loanAmount" type="number" min="1" class="bank-input" placeholder="Сумма" />
              <button class="btn-glass" :disabled="busy || !validAmount(loanAmount)" @click="repay(loanAmount)">
                Погасить
              </button>
              <button class="btn-glass" :disabled="busy || bank.kudos < bank.loan" @click="repay(bank.loan)">
                Погасить всё
              </button>
            </div>
          </template>
          <template v-else>
            <p class="bank-block-hint">
              До {{ bank.tier.loan_max }} кудосов, комиссия {{ bank.tier.loan_fee_pct }}%.
              Один активный кредит; с долгом вклад недоступен.
            </p>
            <div class="bank-row">
              <input v-model.number="loanAmount" type="number" min="1" :max="bank.tier.loan_max" class="bank-input" placeholder="Сумма" />
              <button class="btn-glass" :disabled="busy || !validAmount(loanAmount) || loanAmount > bank.tier.loan_max" @click="takeLoan">
                Взять кредит
              </button>
            </div>
          </template>
        </div>

        <!-- Топ щедрости -->
        <div v-if="bank.top_generous?.length" class="bank-block">
          <h4 class="bank-block-title"><span class="material-symbols-outlined">volunteer_activism</span> Самые щедрые за месяц</h4>
          <ul class="bank-generous">
            <li v-for="(g, i) in bank.top_generous" :key="g.user?.id ?? i" class="bank-generous-row">
              <span class="bank-generous-place">{{ ['🥇', '🥈', '🥉'][i] || '·' }}</span>
              <img class="bank-generous-avatar" :src="avatarUrl(g.user)" :alt="g.user?.fio" />
              <span class="bank-generous-name">{{ g.user?.fio }}</span>
              <span class="bank-generous-sent">подарил(а) {{ g.sent }} <KudosCoin /></span>
            </li>
          </ul>
        </div>
      </div>

      <!-- ── Перевод ── -->
      <div v-else-if="tab === 'transfer'" class="bank-pane">
        <p class="bank-block-hint">
          За один раз — до {{ bank.tier.transfer_max }} кудосов; сегодня осталось {{ bank.transfer_left_today }}.
        </p>
        <div class="bank-recipients">
          <button
            v-for="p in colleagues"
            :key="p.user_id"
            class="bank-recipient"
            :class="{ active: recipientId === p.user_id }"
            type="button"
            @click="recipientId = p.user_id"
          >
            <img class="bank-recipient-avatar" :src="avatarUrl(p.user)" :alt="p.user?.fio" />
            <span class="bank-recipient-name">{{ firstName(p.user?.fio) }}</span>
          </button>
          <p v-if="!colleagues.length" class="bank-block-hint">В компании пока нет коллег с питомцами.</p>
        </div>
        <div class="bank-amount-chips">
          <button
            v-for="a in quickAmounts"
            :key="a"
            class="bank-chip"
            :class="{ active: transferAmount === a }"
            type="button"
            @click="transferAmount = a"
          ><KudosCoin /> {{ a }}</button>
          <input
            v-model.number="transferAmount"
            type="number" min="1" :max="bank.tier.transfer_max"
            class="bank-input bank-input--amount" placeholder="Сумма"
          />
        </div>
        <input
          v-model="transferComment"
          type="text" maxlength="120"
          class="bank-input bank-input--comment"
          placeholder="Спасибо за… (комментарий увидит получатель)"
        />
        <button
          class="btn-grad bank-send"
          :disabled="busy || !recipientId || !validAmount(transferAmount) || transferAmount > bank.tier.transfer_max"
          @click="send"
        >
          <span class="material-symbols-outlined">send</span>
          Перевести {{ validAmount(transferAmount) ? transferAmount : '' }} <KudosCoin />
        </button>
      </div>

      <!-- ── История ── -->
      <div v-else class="bank-pane">
        <EmptyState
          v-if="!pets.ledger.length"
          icon="receipt_long" size="sm" tone="soft"
          title="Операций пока нет"
          subtitle="Заработанные и потраченные кудосы появятся здесь."
        />
        <ul v-else class="bank-ledger">
          <li v-for="e in pets.ledger" :key="e.id" class="bank-ledger-row">
            <span class="material-symbols-outlined bank-ledger-icon">{{ ledgerIcon(e) }}</span>
            <span class="bank-ledger-body">
              <span class="bank-ledger-title">{{ ledgerText(e) }}</span>
              <span class="bank-ledger-date">{{ fmtDate(e.created_at) }}</span>
            </span>
            <span class="bank-ledger-delta" :class="e.delta > 0 ? 'in' : 'out'">
              {{ e.delta > 0 ? '+' : '−' }}{{ Math.abs(e.delta) }}
            </span>
          </li>
        </ul>
        <button
          v-if="pets.ledgerNextBeforeId"
          class="btn-glass bank-more"
          :disabled="busy"
          @click="loadMore"
        >Показать ещё</button>
      </div>
    </template>
  </AppDialog>
</template>

<script setup>
import { computed, ref, watch } from 'vue'
import ProgressSpinner from 'primevue/progressspinner'
import AppDialog from '@/components/common/AppDialog.vue'
import SegmentedTabs from '@/components/common/SegmentedTabs.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { avatarUrl } from '@/utils/pets.js'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
})
defineEmits(['update:modelValue'])

const pets = usePetsStore()
const notify = useNotificationsStore()

const TABS = [
  { value: 'overview', label: 'Обзор', icon: 'account_balance_wallet' },
  { value: 'transfer', label: 'Перевод', icon: 'send_money' },
  { value: 'history', label: 'История', icon: 'receipt_long' },
]
const TIER_EMOJI = { start: '🌱', bronze: '🥉', silver: '🥈', gold: '🥇', platinum: '💎' }

const tab = ref('overview')
const busy = ref(false)
const savingsAmount = ref(null)
const loanAmount = ref(null)
const recipientId = ref(null)
const transferAmount = ref(null)
const transferComment = ref('')

const bank = computed(() => pets.bank)
const subtitle = computed(() =>
  bank.value ? `Уровень «${bank.value.tier.title}» — переводы, вклад и кредит` : 'Переводы, вклад и кредит')

const tierPercent = computed(() => {
  const b = bank.value
  if (!b?.next_tier) return 100
  const from = b.tier.threshold
  const span = b.next_tier.threshold - from
  return Math.min(100, Math.max(2, Math.round(((b.earned - from) / Math.max(1, span)) * 100)))
})

const colleagues = computed(() =>
  (pets.zoo || []).filter((p) => p.user_id !== pets.myId && p.user))

const quickAmounts = computed(() => {
  const max = bank.value?.tier.transfer_max ?? 20
  return [5, 10, 25, 50].filter((a) => a <= max)
})

const validAmount = (v) => Number.isFinite(v) && v >= 1

watch(() => props.modelValue, (open) => {
  if (!open) return
  pets.fetchBank().catch(() => {})
  pets.fetchLedger().catch(() => {})
  if (!pets.zoo.length) pets.fetchZoo().catch(() => {})
})

async function run(action, successMsg) {
  busy.value = true
  try {
    await action()
    if (successMsg) notify.success(successMsg)
  } catch (e) {
    notify.error(e?.message || 'Операция не удалась')
  } finally {
    busy.value = false
  }
}

const deposit = () => run(async () => {
  await pets.bankDeposit(savingsAmount.value)
  savingsAmount.value = null
}, 'Вклад пополнен')

const withdraw = () => run(async () => {
  await pets.bankWithdraw(savingsAmount.value)
  savingsAmount.value = null
}, 'Снято с вклада')

const takeLoan = () => run(async () => {
  await pets.bankTakeLoan(loanAmount.value)
  loanAmount.value = null
}, 'Кредит зачислен на кошелёк')

const repay = (amount) => run(async () => {
  await pets.bankRepayLoan(amount)
  loanAmount.value = null
}, 'Платёж по кредиту принят')

const send = () => run(async () => {
  await pets.transferKudos(recipientId.value, transferAmount.value, transferComment.value)
  transferAmount.value = null
  transferComment.value = ''
}, 'Перевод отправлен')

const loadMore = () => run(() => pets.fetchLedger({ more: true }))

// ── Тексты выписки ──
const LEDGER_META = {
  unit: { icon: 'timer', text: () => 'Работа: завершённые юниты' },
  task_closed: { icon: 'task_alt', text: () => 'Работа: закрытая задача' },
  quest: { icon: 'flag', text: () => 'Награда дневного квеста' },
  adventure: { icon: 'explore', text: (e) => `Приключение${e.comment ? ` ${e.comment}` : ''}` },
  season: { icon: 'military_tech', text: () => 'Награда сезонного трека' },
  feed: { icon: 'restaurant', text: (e) => (e.comment === 'бульон' ? 'Лечебный бульон' : 'Кормление питомца') },
  walk: { icon: 'directions_walk', text: () => 'Прогулка с питомцем' },
  heal: { icon: 'healing', text: () => 'Лечение питомца' },
  stroke: { icon: 'pets', text: (e) => `Поглаживание питомца${e.counterparty ? ` — ${e.counterparty.fio}` : ''}` },
  shop: { icon: 'shopping_bag', text: () => 'Покупка в магазине' },
  house: { icon: 'chair', text: () => 'Декор для домика' },
  transfer_in: { icon: 'call_received', text: (e) => `Перевод от ${e.counterparty?.fio || 'коллеги'}${e.comment ? ` — «${e.comment}»` : ''}` },
  transfer_out: { icon: 'call_made', text: (e) => `Перевод: ${e.counterparty?.fio || 'коллеге'}${e.comment ? ` — «${e.comment}»` : ''}` },
  bank_deposit: { icon: 'savings', text: () => 'Пополнение вклада' },
  bank_withdraw: { icon: 'savings', text: () => 'Снятие с вклада' },
  bank_interest: { icon: 'trending_up', text: () => 'Проценты по вкладу' },
  loan_taken: { icon: 'credit_score', text: (e) => `Кредит${e.comment ? ` (${e.comment})` : ''}` },
  loan_repaid: { icon: 'credit_score', text: () => 'Погашение кредита' },
}

const ledgerIcon = (e) => LEDGER_META[e.kind]?.icon || 'receipt_long'
const ledgerText = (e) => {
  try { return LEDGER_META[e.kind]?.text(e) || e.kind } catch { return e.kind }
}

function fmtDate(iso) {
  const d = new Date(iso)
  return d.toLocaleString('ru-RU', { day: 'numeric', month: 'short', hour: '2-digit', minute: '2-digit' })
}

function firstName(fio = '') {
  const parts = fio.trim().split(/\s+/)
  return parts.length > 1 ? parts[1] : parts[0] || ''
}
</script>

<style scoped>
.bank-loading { display: flex; justify-content: center; padding: 40px 0; }

.bank-balances {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 10px;
  margin-bottom: 12px;
}
.bank-balance-card {
  background: var(--color-surface-low);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 10px 12px;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.bank-balance-card.debt {
  background: var(--color-error-container);
  color: var(--color-on-error-container);
  border-color: transparent;
}
.bank-balance-label { font-size: 11.5px; color: var(--color-text-dim); }
.bank-balance-card.debt .bank-balance-label { color: inherit; opacity: 0.8; }
.bank-balance-value {
  font-size: 18px;
  font-weight: 800;
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.bank-tier {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}
.bank-tier-badge {
  font-size: 12.5px;
  font-weight: 700;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  border-radius: var(--radius-full);
  padding: 4px 12px;
  white-space: nowrap;
}
.bank-tier-bar {
  flex: 1;
  min-width: 90px;
  height: 6px;
  border-radius: var(--radius-full);
  background: var(--color-surface-low);
  overflow: hidden;
}
.bank-tier-fill { height: 100%; background: var(--color-primary); border-radius: inherit; }
.bank-tier-hint { font-size: 11.5px; color: var(--color-text-dim); }

.bank-pane { display: flex; flex-direction: column; gap: 14px; margin-top: 14px; }

.bank-month { display: flex; gap: 10px; flex-wrap: wrap; }
.bank-month-item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 12.5px;
  font-weight: 600;
  border-radius: var(--radius-full);
  padding: 5px 12px;
}
.bank-month-item .material-symbols-outlined { font-size: 15px; }
.bank-month-item.in { background: var(--color-success-container); color: var(--color-on-success-container); }
.bank-month-item.out { background: var(--color-surface-low); color: var(--color-text-dim); }

.bank-block {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.bank-block-title {
  margin: 0;
  font-size: 13.5px;
  font-weight: 700;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.bank-block-title .material-symbols-outlined { font-size: 18px; color: var(--color-primary); }
.bank-block-hint { margin: 0; font-size: 12px; color: var(--color-text-dim); line-height: 1.4; }

.bank-row { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }

.bank-input {
  width: 110px;
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-sm);
  background: var(--color-surface);
  color: var(--color-text);
  font: inherit;
  font-size: 13px;
  padding: 8px 10px;
}
.bank-input--comment { width: 100%; }
.bank-input--amount { width: 90px; }
.bank-input:focus { outline: none; border-color: var(--color-primary); }

.bank-generous { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.bank-generous-row { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.bank-generous-place { width: 20px; text-align: center; }
.bank-generous-avatar { width: 28px; height: 28px; border-radius: 50%; object-fit: cover; }
.bank-generous-name { font-weight: 600; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bank-generous-sent { color: var(--color-text-dim); font-size: 12px; display: inline-flex; gap: 3px; align-items: center; }

.bank-recipients {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}
.bank-recipient {
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  border-radius: var(--radius-md);
  padding: 8px 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 5px;
  width: 76px;
  cursor: pointer;
  transition: border-color 0.12s, background 0.12s;
}
.bank-recipient.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}
.bank-recipient-avatar { width: 36px; height: 36px; border-radius: 50%; object-fit: cover; }
.bank-recipient-name {
  font-size: 11.5px;
  font-weight: 600;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.bank-amount-chips { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.bank-chip {
  border: 1px solid var(--color-outline-dim);
  background: var(--color-surface);
  color: var(--color-text);
  border-radius: var(--radius-full);
  font: inherit;
  font-size: 12.5px;
  font-weight: 600;
  padding: 6px 12px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
}
.bank-chip.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.bank-send {
  align-self: flex-start;
  display: inline-flex;
  align-items: center;
  gap: 6px;
}

.bank-ledger { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; }
.bank-ledger-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 2px;
  border-bottom: 1px solid var(--color-outline-dim);
}
.bank-ledger-row:last-child { border-bottom: none; }
.bank-ledger-icon { font-size: 20px; color: var(--color-text-dim); }
.bank-ledger-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
.bank-ledger-title { font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.bank-ledger-date { font-size: 11px; color: var(--color-text-dim); }
.bank-ledger-delta { font-size: 13.5px; font-weight: 800; flex-shrink: 0; }
.bank-ledger-delta.in { color: var(--color-success); }
.bank-ledger-delta.out { color: var(--color-text-dim); }

.bank-more { align-self: center; }

@media (max-width: 560px) {
  .bank-balances { grid-template-columns: 1fr 1fr; }
}
</style>
