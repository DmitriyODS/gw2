<template>
  <div class="admin-page">
    <header class="admin-sticky">
      <div class="kb-toolbar">
        <button class="btn-glass kb-back" type="button" title="К грувикам" @click="router.push('/pets')">
          <span class="material-symbols-outlined">arrow_back</span>
        </button>
        <h1 class="kb-title">Кудо-банк</h1>
        <span v-if="bank" class="chip-tint chip-tint--primary kb-tier-chip">
          {{ TIER_EMOJI[bank.tier.key] || '⭐' }} {{ bank.tier.title }}
        </span>
        <button class="btn-grad kb-transfer-btn" type="button" @click="openTransfer()">
          <span class="material-symbols-outlined">send_money</span>
          <span class="kb-btn-label">Перевести</span>
        </button>
      </div>
    </header>

    <div v-if="!bank" class="admin-body kb-loading">
      <BrandLoader :size="64" />
    </div>

    <div v-else class="admin-body">
      <!-- ── Карта клиента ─────────────────────────────────────── -->
      <section class="kb-hero">
        <span class="kb-hero-watermark" aria-hidden="true">{{ TIER_EMOJI[bank.tier.key] || '⭐' }}</span>
        <div class="kb-hero-main">
          <span class="kb-hero-label">Кошелёк</span>
          <span class="kb-hero-balance"><KudosCoin /> {{ bank.kudos }}</span>
          <div class="kb-hero-trend">
            <span class="kb-trend in"><span class="material-symbols-outlined">arrow_downward</span>+{{ bank.month_in }}</span>
            <span class="kb-trend out"><span class="material-symbols-outlined">arrow_upward</span>−{{ bank.month_out }}</span>
            <span class="kb-trend-hint">за 30 дней</span>
          </div>
        </div>
        <div class="kb-hero-side">
          <div class="kb-hero-cell">
            <span class="kb-hero-cell-label">Вклад</span>
            <span class="kb-hero-cell-value"><KudosCoin /> {{ bank.savings }}</span>
          </div>
          <div class="kb-hero-cell">
            <span class="kb-hero-cell-label">В копилках</span>
            <span class="kb-hero-cell-value"><KudosCoin /> {{ goalsTotal }}</span>
          </div>
          <div class="kb-hero-cell" :class="{ debt: bank.loan > 0 }">
            <span class="kb-hero-cell-label">{{ bank.loan > 0 ? 'Долг' : 'Кредит' }}</span>
            <span class="kb-hero-cell-value">
              <template v-if="bank.loan > 0"><KudosCoin /> {{ bank.loan }}</template>
              <template v-else>нет</template>
            </span>
          </div>
        </div>
        <div class="kb-hero-tier">
          <template v-if="bank.next_tier">
            <div class="kb-tier-bar"><div class="kb-tier-fill" :style="{ width: tierPercent + '%' }"></div></div>
            <span class="kb-tier-hint">
              {{ bank.earned }} / {{ bank.next_tier.threshold }} заработанных до уровня
              «{{ bank.next_tier.title }}» — ставка вклада {{ bank.next_tier.savings_rate_pct }}%
            </span>
          </template>
          <span v-else class="kb-tier-hint">Максимальный уровень — лучшие условия банка ✨</span>
        </div>
      </section>

      <!-- ── Быстрые действия ──────────────────────────────────── -->
      <section class="kb-actions">
        <button class="kb-action glass-hover" type="button" @click="openTransfer()">
          <span class="kb-action-icon material-symbols-outlined">send_money</span>
          <span>Перевести</span>
        </button>
        <button class="kb-action glass-hover" type="button" @click="focusDeposit">
          <span class="kb-action-icon material-symbols-outlined">savings</span>
          <span>На вклад</span>
        </button>
        <button class="kb-action glass-hover" type="button" @click="goalCreateOpen = true">
          <span class="kb-action-icon material-symbols-outlined">target</span>
          <span>Копилка</span>
        </button>
        <button v-if="bank.loan > 0" class="kb-action kb-action--alert glass-hover" type="button" @click="focusLoan">
          <span class="kb-action-icon material-symbols-outlined">credit_score</span>
          <span>Погасить долг</span>
        </button>
        <button v-if="isManager()" class="kb-action glass-hover" type="button" @click="fundCreateOpen = true">
          <span class="kb-action-icon material-symbols-outlined">volunteer_activism</span>
          <span>Объявить сбор</span>
        </button>
      </section>

      <div class="kb-grid">
        <!-- ── Вклад ────────────────────────────────────────────── -->
        <section ref="depositCard" class="kb-card">
          <header class="kb-card-head">
            <span class="kb-card-icon kb-card-icon--pink material-symbols-outlined">savings</span>
            <h3 class="kb-card-title">Вклад</h3>
            <span class="kb-badge kb-badge--success">{{ bank.tier.savings_rate_pct }}% в день</span>
          </header>
          <!-- Главное — состояние счёта: сколько лежит и что накапает. -->
          <div class="kb-stats">
            <div class="kb-stat">
              <span class="kb-stat-label">На вкладе</span>
              <span class="kb-stat-value">{{ bank.savings }} <KudosCoin /></span>
            </div>
            <div class="kb-stat-divider"></div>
            <div class="kb-stat">
              <span class="kb-stat-label">Доход завтра</span>
              <span class="kb-stat-value kb-stat-value--success">+{{ savingsTomorrow }} <KudosCoin /></span>
            </div>
          </div>
          <p class="kb-card-hint">
            Проценты капают за каждые полные сутки по ставке вашего уровня.
            <template v-if="bank.loan > 0"> Пока есть долг, вклад закрыт.</template>
          </p>
          <div class="kb-field">
            <label class="kb-field-label">Сумма</label>
            <AmountInput ref="depositInput" v-model="savingsAmount" placeholder="0" />
          </div>
          <div class="kb-row">
            <button
              class="btn-grad"
              :disabled="busy || bank.loan > 0 || !validAmount(savingsAmount)"
              @click="deposit"
            >
              <span class="material-symbols-outlined">download</span> Пополнить
            </button>
            <button
              class="btn-glass"
              :disabled="busy || !bank.savings || !validAmount(savingsAmount)"
              @click="withdraw"
            >
              <span class="material-symbols-outlined">upload</span> Снять
            </button>
          </div>
        </section>

        <!-- ── Кредит ───────────────────────────────────────────── -->
        <section ref="loanCard" class="kb-card">
          <header class="kb-card-head">
            <span class="kb-card-icon kb-card-icon--violet material-symbols-outlined">credit_card</span>
            <h3 class="kb-card-title">Кредит</h3>
            <span v-if="bank.credit" class="kb-badge kb-badge--violet">{{ RATING_EMOJI[bank.credit.tier.key] || '🏅' }} {{ bank.credit.tier.title }}</span>
          </header>

          <template v-if="bank.credit">
          <!-- Кредитный рейтинг: прогресс и условия. -->
          <div class="kb-credit-rating">
            <div class="kb-credit-row">
              <span class="kb-credit-label">Кредитный рейтинг</span>
              <span class="kb-credit-score">{{ bank.credit.score }}</span>
            </div>
            <div v-if="bank.credit.next_tier" class="kb-tier-bar kb-tier-bar--sm">
              <div class="kb-tier-fill" :style="{ width: creditPercent + '%' }"></div>
            </div>
            <p class="kb-card-hint kb-credit-hint">
              <template v-if="bank.credit.next_tier">
                Ещё {{ bank.credit.next_tier.min_score - bank.credit.score }} возврат(а) в срок до «{{ bank.credit.next_tier.title }}»:
                комиссия {{ bank.credit.next_tier.fee_pct }}%, лимит до {{ bank.credit.next_tier.loan_max }}.
              </template>
              <template v-else>Высший рейтинг — лучшие условия кредита ✨</template>
            </p>
            <p class="kb-card-hint kb-credit-perk">
              <span class="material-symbols-outlined">savings</span>
              Вернёте за {{ bank.credit.grace_days }} дн. без просрочек — кэшбэк {{ bank.credit.cashback_pct }}% от тела и +1 к рейтингу.
              <template v-if="bank.credit.fee_pct === 0"> Комиссия 0% — кэшбэк идёт в плюс!</template>
            </p>
          </div>

          <template v-if="bank.loan > 0">
            <p class="kb-card-hint">Остаток долга — {{ bank.loan }} кудосов. Погашение — с кошелька, вклад откроется после.</p>
            <div v-if="bank.credit.loan_due_at" class="kb-loan-due" :class="{ overdue: bank.credit.overdue }">
              <span class="material-symbols-outlined">{{ bank.credit.overdue ? 'error' : 'schedule' }}</span>
              <span v-if="bank.credit.overdue">Просрочка — на остаток капает +20%/нед. Внесите платёж, чтобы остановить рост долга.</span>
              <span v-else>Платёж нужен минимум раз в неделю — до {{ formatDue(bank.credit.loan_due_at) }}, иначе +20%/нед на остаток.</span>
            </div>
            <div class="kb-field">
              <label class="kb-field-label">Сумма платежа</label>
              <AmountInput ref="loanInput" v-model="loanAmount" placeholder="0" />
            </div>
            <div class="kb-row">
              <button class="btn-grad" :disabled="busy || !validAmount(loanAmount)" @click="repay(loanAmount)">
                Погасить
              </button>
              <button class="btn-glass" :disabled="busy || bank.kudos < bank.loan" @click="repay(bank.loan)">
                Погасить всё ({{ bank.loan }})
              </button>
            </div>
          </template>

          <template v-else>
            <p class="kb-card-hint">Один активный кредит; с долгом вклад недоступен.</p>
            <div class="kb-stats">
              <div class="kb-stat">
                <span class="kb-stat-label">Доступно к получению</span>
                <span class="kb-stat-value">до {{ bank.credit.loan_max }} <KudosCoin /></span>
              </div>
              <div class="kb-stat-divider"></div>
              <div class="kb-stat">
                <span class="kb-stat-label">Комиссия</span>
                <span class="kb-stat-value">{{ bank.credit.fee_pct }}%</span>
              </div>
            </div>
            <div class="kb-field">
              <label class="kb-field-label">Сумма кредита</label>
              <AmountInput ref="loanInput" v-model="loanAmount" :max="bank.credit.loan_max" placeholder="0" />
            </div>
            <button
              class="btn-grad kb-loan-take"
              :disabled="busy || !validAmount(loanAmount) || loanAmount > bank.credit.loan_max"
              @click="takeLoan"
            >
              <span class="material-symbols-outlined">account_balance_wallet</span>
              Взять кредит{{ validAmount(loanAmount) ? ` — вернуть ${loanDebt}` : '' }}
            </button>
          </template>
          </template>
          <p v-else class="kb-card-hint">Данные кредита обновляются — обновите страницу.</p>
        </section>

        <!-- ── Рассрочка (оплата частями) ───────────────────────── -->
        <InstallmentsCard />

        <!-- ── Копилки-цели ─────────────────────────────────────── -->
        <section class="kb-card">
          <header class="kb-card-head">
            <span class="kb-card-icon kb-card-icon--amber material-symbols-outlined">target</span>
            <h3 class="kb-card-title">Копилки</h3>
            <button
              v-if="(bank.goals?.length || 0) < (bank.goals_max || 4)"
              class="kb-head-action"
              type="button"
              @click="goalCreateOpen = true"
            ><span class="material-symbols-outlined">add</span> Новая</button>
          </header>

          <p v-if="!bank.goals?.length" class="kb-card-hint">
            Копите на мечту отдельно от кошелька: облик, декор, щедрый подарок коллеге.
          </p>
          <ul v-else class="kb-goals">
            <li v-for="g in bank.goals" :key="g.id" class="kb-goal" :class="{ achieved: g.achieved }">
              <button class="kb-goal-row" type="button" @click="toggleGoal(g.id)">
                <span class="kb-goal-emoji">{{ g.emoji }}</span>
                <span class="kb-goal-info">
                  <span class="kb-goal-title">{{ g.title }}</span>
                  <span class="kb-goal-bar"><span class="kb-goal-fill" :style="{ width: goalPercent(g) + '%' }"></span></span>
                </span>
                <span class="kb-goal-nums">
                  <template v-if="g.achieved">🎉 {{ g.saved }}</template>
                  <template v-else>{{ g.saved }} / {{ g.target }}</template>
                </span>
                <span class="material-symbols-outlined kb-goal-chevron">{{ expandedGoalId === g.id ? 'expand_less' : 'expand_more' }}</span>
              </button>
              <div v-if="expandedGoalId === g.id" class="kb-goal-ops">
                <AmountInput v-model="goalAmount" size="sm" class="kb-goal-input" />
                <button class="btn-grad kb-goal-btn" :disabled="busy || !validAmount(goalAmount)" @click="goalDeposit(g)">
                  Пополнить
                </button>
                <button class="btn-glass kb-goal-btn" :disabled="busy || !g.saved || !validAmount(goalAmount)" @click="goalWithdraw(g)">
                  Снять
                </button>
                <button
                  class="kb-goal-delete"
                  type="button"
                  :disabled="busy"
                  @click="deleteGoal(g)"
                >{{ confirmDeleteId === g.id ? 'Точно? Остаток вернётся' : 'Удалить' }}</button>
              </div>
            </li>
          </ul>
        </section>

        <!-- ── Сборы (благотворительность) ──────────────────────── -->
        <section class="kb-card">
          <header class="kb-card-head">
            <span class="kb-card-icon kb-card-icon--teal material-symbols-outlined">volunteer_activism</span>
            <h3 class="kb-card-title">Сборы компании</h3>
            <button
              v-if="isManager()"
              class="kb-head-action"
              type="button"
              @click="fundCreateOpen = true"
            ><span class="material-symbols-outlined">add</span> Объявить</button>
          </header>

          <p v-if="!bank.funds?.length" class="kb-card-hint">
            Здесь появляются общие цели: скинуться на подарок, пиццу за релиз или доброе дело.
          </p>
          <ul v-else class="kb-funds">
            <li v-for="f in bank.funds" :key="f.id" class="kb-fund" :class="'is-' + f.status">
              <div class="kb-fund-head">
                <span class="kb-fund-emoji">{{ f.emoji }}</span>
                <div class="kb-fund-info">
                  <span class="kb-fund-title">{{ f.title }}</span>
                  <span v-if="f.description" class="kb-fund-desc">{{ f.description }}</span>
                </div>
                <span v-if="f.status === 'done'" class="kb-badge kb-badge--success">Собран 🎉</span>
                <span v-else-if="f.status === 'closed'" class="kb-badge">Закрыт</span>
              </div>
              <div class="kb-fund-bar">
                <div class="kb-fund-fill" :style="{ width: fundPercent(f) + '%' }"></div>
              </div>
              <div class="kb-fund-meta">
                <span><strong>{{ f.collected }}</strong> / {{ f.target }} <KudosCoin /></span>
                <span v-if="f.donors_count">· {{ f.donors_count }} {{ donorsWord(f.donors_count) }}</span>
                <span v-if="f.my_donated" class="kb-fund-mine">· мой вклад {{ f.my_donated }}</span>
              </div>
              <div v-if="f.top_donors?.length" class="kb-fund-donors">
                <img
                  v-for="d in f.top_donors"
                  :key="d.user?.id"
                  class="kb-fund-donor"
                  :src="avatarUrl(d.user)"
                  :alt="d.user?.fio"
                  :title="`${d.user?.fio} — ${d.sent}`"
                />
              </div>
              <div v-if="f.status === 'active'" class="kb-fund-ops">
                <AmountInput :model-value="fundAmounts[f.id]" size="sm" class="kb-fund-input"
                  @update:model-value="(v) => { fundAmounts[f.id] = v }" />
                <button
                  class="btn-grad kb-fund-btn"
                  :disabled="busy || !validAmount(fundAmounts[f.id])"
                  @click="donate(f)"
                >Поддержать</button>
                <button
                  v-if="canCloseFund(f)"
                  class="kb-fund-close"
                  type="button"
                  :disabled="busy"
                  @click="closeFund(f)"
                >{{ confirmCloseId === f.id ? 'Точно закрыть?' : 'Завершить' }}</button>
              </div>
            </li>
          </ul>
        </section>

        <!-- ── Динамика и структура ─────────────────────────────── -->
        <section class="kb-card">
          <header class="kb-card-head">
            <span class="kb-card-icon kb-card-icon--blue material-symbols-outlined">monitoring</span>
            <h3 class="kb-card-title">Динамика</h3>
            <span class="kb-badge">{{ stats?.days || 14 }} дней</span>
          </header>
          <template v-if="stats">
            <div class="kb-chart" role="img" :aria-label="`Приход и расход за ${stats.days} дней`">
              <div v-for="d in chartDays" :key="d.day" class="kb-chart-day" :title="`${d.title}: +${d.in} / −${d.out}`">
                <div class="kb-chart-bars">
                  <span class="kb-chart-bar in" :style="{ height: d.inPct + '%' }"></span>
                  <span class="kb-chart-bar out" :style="{ height: d.outPct + '%' }"></span>
                </div>
                <span class="kb-chart-label">{{ d.label }}</span>
              </div>
            </div>
            <div class="kb-legend">
              <span class="kb-legend-item"><span class="kb-legend-dot in"></span>заработано</span>
              <span class="kb-legend-item"><span class="kb-legend-dot out"></span>потрачено</span>
            </div>
            <div v-if="topSpend.length || topEarn.length" class="kb-kinds">
              <div v-if="topEarn.length" class="kb-kinds-col">
                <h4 class="kb-kinds-title">Откуда приходят</h4>
                <div v-for="k in topEarn" :key="'in-' + k.kind" class="kb-kind">
                  <span class="kb-kind-name">{{ kindTitle(k.kind) }}</span>
                  <span class="kb-kind-bar"><span class="kb-kind-fill in" :style="{ width: k.pct + '%' }"></span></span>
                  <span class="kb-kind-val">+{{ k.value }}</span>
                </div>
              </div>
              <div v-if="topSpend.length" class="kb-kinds-col">
                <h4 class="kb-kinds-title">Куда уходят</h4>
                <div v-for="k in topSpend" :key="'out-' + k.kind" class="kb-kind">
                  <span class="kb-kind-name">{{ kindTitle(k.kind) }}</span>
                  <span class="kb-kind-bar"><span class="kb-kind-fill out" :style="{ width: k.pct + '%' }"></span></span>
                  <span class="kb-kind-val">−{{ k.value }}</span>
                </div>
              </div>
            </div>
          </template>
          <p v-else class="kb-card-hint">Считаем…</p>
        </section>

        <!-- ── Топ щедрости ─────────────────────────────────────── -->
        <section class="kb-card">
          <header class="kb-card-head">
            <span class="kb-card-icon kb-card-icon--pink material-symbols-outlined">favorite</span>
            <h3 class="kb-card-title">Самые щедрые за месяц</h3>
          </header>
          <p v-if="!bank.top_generous?.length" class="kb-card-hint">
            Пока никто не переводил кудосы — станьте первым, кого здесь увидят.
          </p>
          <ul v-else class="kb-generous">
            <li v-for="(g, i) in bank.top_generous" :key="g.user?.id ?? i" class="kb-generous-row">
              <span class="kb-generous-place">{{ ['🥇', '🥈', '🥉'][i] || '·' }}</span>
              <img class="kb-generous-avatar" :src="avatarUrl(g.user)" :alt="g.user?.fio" />
              <span class="kb-generous-name">{{ g.user?.fio }}</span>
              <span class="kb-generous-sent">{{ g.sent }} <KudosCoin /></span>
              <button
                v-if="g.user?.id !== pets.myId"
                class="kb-generous-thank"
                type="button"
                title="Отблагодарить переводом"
                @click="openTransfer(g.user?.id)"
              ><span class="material-symbols-outlined">send_money</span></button>
            </li>
          </ul>
        </section>

        <!-- ── История ──────────────────────────────────────────── -->
        <section class="kb-card kb-card--wide">
          <header class="kb-card-head">
            <span class="kb-card-icon kb-card-icon--blue material-symbols-outlined">receipt_long</span>
            <h3 class="kb-card-title">История операций</h3>
          </header>
          <div class="kb-filters">
            <button
              v-for="f in LEDGER_FILTERS"
              :key="f.value"
              class="kb-filter"
              :class="{ active: ledgerFilter === f.value }"
              type="button"
              @click="ledgerFilter = f.value"
            >{{ f.label }}</button>
          </div>

          <EmptyState
            v-if="!filteredDays.length"
            icon="receipt_long" size="sm" tone="soft"
            title="Операций пока нет"
            subtitle="Заработанные и потраченные кудосы появятся здесь."
          />
          <div v-else class="kb-days">
            <div v-for="day in filteredDays" :key="day.key" class="kb-day">
              <div class="kb-day-head">
                <span class="kb-day-title">{{ day.title }}</span>
                <span class="kb-day-totals">
                  <span v-if="day.in" class="in">+{{ day.in }}</span>
                  <span v-if="day.out" class="out">−{{ day.out }}</span>
                </span>
              </div>
              <ul class="kb-ledger">
                <li v-for="e in day.items" :key="e.id" class="kb-ledger-row">
                  <span class="material-symbols-outlined kb-ledger-icon">{{ ledgerIcon(e) }}</span>
                  <span class="kb-ledger-body">
                    <span class="kb-ledger-title">{{ ledgerText(e) }}</span>
                    <span class="kb-ledger-time">{{ fmtTime(e.created_at) }}</span>
                  </span>
                  <span class="kb-ledger-delta" :class="e.delta > 0 ? 'in' : 'out'">
                    {{ e.delta > 0 ? '+' : '−' }}{{ Math.abs(e.delta) }}
                  </span>
                </li>
              </ul>
            </div>
          </div>
          <button
            v-if="pets.ledgerNextBeforeId"
            class="btn-glass kb-more"
            :disabled="busy"
            @click="loadMore"
          >Показать ещё</button>
        </section>
      </div>
    </div>

    <TransferDialog v-model="transferOpen" :preset-user-id="transferPreset" />
    <GoalCreateDialog v-model="goalCreateOpen" />
    <FundCreateDialog v-model="fundCreateOpen" />
    <ConfettiBurst ref="confettiEl" />
  </div>
</template>

<script setup>
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import BrandLoader from '@/components/common/BrandLoader.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import KudosCoin from '@/components/pets/KudosCoin.vue'
import TransferDialog from '@/components/pets/bank/TransferDialog.vue'
import GoalCreateDialog from '@/components/pets/bank/GoalCreateDialog.vue'
import FundCreateDialog from '@/components/pets/bank/FundCreateDialog.vue'
import ConfettiBurst from '@/components/pets/bank/ConfettiBurst.vue'
import AmountInput from '@/components/pets/bank/AmountInput.vue'
import InstallmentsCard from '@/components/pets/bank/InstallmentsCard.vue'
import { ledgerIcon, ledgerText, ledgerGroup, kindTitle } from '@/components/pets/bank/ledgerMeta.js'
import { usePetsStore } from '@/stores/pets.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { usePermission } from '@/composables/usePermission.js'
import { avatarUrl } from '@/utils/pets.js'

const TIER_EMOJI = { start: '🌱', bronze: '🥉', silver: '🥈', gold: '🥇', platinum: '💎' }
const RATING_EMOJI = { none: '🆕', known: '🤝', trusted: '✅', prime: '🌟', premium: '👑' }
const LEDGER_FILTERS = [
  { value: 'all', label: 'Все' },
  { value: 'earn', label: 'Заработок' },
  { value: 'spend', label: 'Траты' },
  { value: 'social', label: 'Переводы' },
  { value: 'bank', label: 'Банк' },
]

const router = useRouter()
const pets = usePetsStore()
const notify = useNotificationsStore()
const { isManager, isAdmin } = usePermission()

const busy = ref(false)
const savingsAmount = ref(null)
const loanAmount = ref(null)
const goalAmount = ref(null)
const fundAmounts = reactive({})
const expandedGoalId = ref(null)
const confirmDeleteId = ref(null)
const confirmCloseId = ref(null)
const ledgerFilter = ref('all')

const transferOpen = ref(false)
const transferPreset = ref(null)
const goalCreateOpen = ref(false)
const fundCreateOpen = ref(false)

const depositCard = ref(null)
const depositInput = ref(null)
const loanCard = ref(null)
const loanInput = ref(null)
const confettiEl = ref(null)

const bank = computed(() => pets.bank)
const stats = computed(() => pets.bankStats)

const goalsTotal = computed(() =>
  (bank.value?.goals || []).reduce((sum, g) => sum + g.saved, 0))

const savingsTomorrow = computed(() => {
  const b = bank.value
  if (!b?.savings) return 0
  return Math.floor(b.savings * b.tier.savings_rate_pct / 100)
})

const tierPercent = computed(() => {
  const b = bank.value
  if (!b?.next_tier) return 100
  const from = b.tier.threshold
  const span = b.next_tier.threshold - from
  return Math.min(100, Math.max(2, Math.round(((b.earned - from) / Math.max(1, span)) * 100)))
})

const loanDebt = computed(() => {
  const b = bank.value
  if (!b?.credit || !validAmount(loanAmount.value)) return 0
  return loanAmount.value + Math.ceil((loanAmount.value * b.credit.fee_pct) / 100)
})

const creditPercent = computed(() => {
  const c = bank.value?.credit
  if (!c?.next_tier) return 100
  const from = c.tier.min_score
  const span = c.next_tier.min_score - from
  return Math.min(100, Math.max(4, Math.round(((c.score - from) / Math.max(1, span)) * 100)))
})

function formatDue(iso) {
  if (!iso) return ''
  return new Date(iso).toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })
}

const validAmount = (v) => Number.isFinite(v) && v >= 1

// ── Динамика: 14 дней с нулями, высоты в % от максимума окна ──
const chartDays = computed(() => {
  const s = stats.value
  if (!s) return []
  const byDay = new Map((s.daily || []).map((d) => [d.day, d]))
  const days = []
  const today = new Date()
  for (let i = s.days - 1; i >= 0; i--) {
    const d = new Date(today)
    d.setDate(today.getDate() - i)
    const key = d.toISOString().slice(0, 10)
    const row = byDay.get(key)
    days.push({
      day: key,
      label: d.toLocaleDateString('ru-RU', { day: 'numeric' }),
      title: d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'short' }),
      in: row?.in || 0,
      out: row?.out || 0,
    })
  }
  const max = Math.max(1, ...days.map((d) => Math.max(d.in, d.out)))
  return days.map((d) => ({
    ...d,
    inPct: Math.round((d.in / max) * 100),
    outPct: Math.round((d.out / max) * 100),
  }))
})

function topKinds(field) {
  const s = stats.value
  if (!s) return []
  const rows = (s.by_kind || [])
    .map((k) => ({ kind: k.kind, value: k[field] }))
    .filter((k) => k.value > 0)
    .sort((a, b) => b.value - a.value)
    .slice(0, 4)
  const max = Math.max(1, ...rows.map((r) => r.value))
  return rows.map((r) => ({ ...r, pct: Math.round((r.value / max) * 100) }))
}
const topEarn = computed(() => topKinds('in'))
const topSpend = computed(() => topKinds('out'))

// ── История: фильтр + группировка по дням ──
const filteredDays = computed(() => {
  const items = (pets.ledger || []).filter((e) =>
    ledgerFilter.value === 'all' || ledgerGroup(e) === ledgerFilter.value)
  const groups = []
  let current = null
  for (const e of items) {
    const key = dayKey(e.created_at)
    if (!current || current.key !== key) {
      current = { key, title: dayTitle(e.created_at), items: [], in: 0, out: 0 }
      groups.push(current)
    }
    current.items.push(e)
    if (e.delta > 0) current.in += e.delta
    else current.out -= e.delta
  }
  return groups
})

function dayKey(iso) {
  return String(iso).slice(0, 10)
}

function dayTitle(iso) {
  const d = new Date(iso)
  const today = new Date()
  const yesterday = new Date(today)
  yesterday.setDate(today.getDate() - 1)
  const same = (a, b) => a.toDateString() === b.toDateString()
  if (same(d, today)) return 'Сегодня'
  if (same(d, yesterday)) return 'Вчера'
  return d.toLocaleDateString('ru-RU', { day: 'numeric', month: 'long' })
}

function fmtTime(iso) {
  return new Date(iso).toLocaleTimeString('ru-RU', { hour: '2-digit', minute: '2-digit' })
}

function goalPercent(g) {
  return Math.min(100, Math.round((g.saved / Math.max(1, g.target)) * 100))
}

function fundPercent(f) {
  return Math.min(100, Math.round((f.collected / Math.max(1, f.target)) * 100))
}

function donorsWord(n) {
  const mod10 = n % 10
  const mod100 = n % 100
  if (mod10 === 1 && mod100 !== 11) return 'участник'
  if (mod10 >= 2 && mod10 <= 4 && (mod100 < 12 || mod100 > 14)) return 'участника'
  return 'участников'
}

function canCloseFund(f) {
  return isAdmin() || f.creator?.id === pets.myId
}

// ── Действия ──
async function run(action, successMsg) {
  busy.value = true
  try {
    const res = await action()
    if (successMsg) notify.success(successMsg)
    return res
  } catch (e) {
    notify.error(e?.message || 'Операция не удалась')
    return null
  } finally {
    busy.value = false
  }
}

// Разовые маркеры ответа — праздник (конфетти) за здоровые финансовые события.
function celebrate(res) {
  if (!res) return
  if (res.goal_achieved) {
    confettiEl.value?.burst()
    notify.success(`Копилка ${res.goal_achieved.emoji} «${res.goal_achieved.title}» полна — цель достигнута!`)
  }
  if (res.fund_completed) {
    confettiEl.value?.burst(36)
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
  const before = bank.value?.loan || 0
  const res = await pets.bankRepayLoan(amount)
  loanAmount.value = null
  if (before > 0 && res?.loan === 0) confettiEl.value?.burst()
  if (res?.loan_cashback) {
    notify.success(`Кредит закрыт в срок — кэшбэк +${res.loan_cashback} и рейтинг вырос!`)
  } else {
    notify.success('Платёж по кредиту принят')
  }
})

function toggleGoal(id) {
  expandedGoalId.value = expandedGoalId.value === id ? null : id
  goalAmount.value = null
  confirmDeleteId.value = null
}

const goalDeposit = (g) => run(async () => {
  celebrate(await pets.goalDeposit(g.id, goalAmount.value))
  goalAmount.value = null
})

const goalWithdraw = (g) => run(async () => {
  await pets.goalWithdraw(g.id, goalAmount.value)
  goalAmount.value = null
}, 'Снято из копилки')

async function deleteGoal(g) {
  if (confirmDeleteId.value !== g.id) {
    confirmDeleteId.value = g.id
    setTimeout(() => { if (confirmDeleteId.value === g.id) confirmDeleteId.value = null }, 3500)
    return
  }
  confirmDeleteId.value = null
  await run(() => pets.deleteGoal(g.id), 'Копилка удалена, остаток вернулся в кошелёк')
}

const donate = (f) => run(async () => {
  celebrate(await pets.donateFund(f.id, fundAmounts[f.id]))
  fundAmounts[f.id] = null
}, 'Взнос отправлен — спасибо!')

async function closeFund(f) {
  if (confirmCloseId.value !== f.id) {
    confirmCloseId.value = f.id
    setTimeout(() => { if (confirmCloseId.value === f.id) confirmCloseId.value = null }, 3500)
    return
  }
  confirmCloseId.value = null
  await run(() => pets.closeFund(f.id), 'Сбор завершён')
}

const loadMore = () => run(() => pets.fetchLedger({ more: true }))

function openTransfer(presetUserId = null) {
  transferPreset.value = presetUserId
  transferOpen.value = true
}

function focusDeposit() {
  depositCard.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  setTimeout(() => depositInput.value?.focus(), 350)
}

function focusLoan() {
  loanCard.value?.scrollIntoView({ behavior: 'smooth', block: 'center' })
  setTimeout(() => loanInput.value?.focus(), 350)
}

onMounted(() => {
  pets.fetchBank().catch(() => {})
  pets.fetchLedger().catch(() => {})
  pets.fetchBankStats().catch(() => {})
  if (!pets.zoo.length) pets.fetchZoo().catch(() => {})
})
</script>

<style scoped>
/* Прозрачная плавающая шапка — как у хаба грувиков. */
.admin-sticky { background: transparent; -webkit-backdrop-filter: none; backdrop-filter: none; }
.admin-sticky::after { display: none; }

.kb-toolbar { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.kb-back { display: inline-flex; align-items: center; padding: 8px 10px; }
.kb-title { margin: 0; font-size: 20px; font-weight: 800; }
.kb-tier-chip { font-weight: 700; }
.kb-transfer-btn { margin-left: auto; display: inline-flex; align-items: center; gap: 6px; }

.kb-loading { display: flex; justify-content: center; padding: 60px 0; }

/* ── Карта клиента: градиентная обложка по паттерну pdm-cover ── */
.kb-hero {
  position: relative;
  border-radius: var(--radius-lg, 20px);
  border: 1px solid var(--acrylic-border);
  padding: 22px 24px;
  overflow: hidden;
  display: grid;
  grid-template-columns: 1fr auto;
  grid-template-areas: 'main side' 'tier tier';
  gap: 16px 24px;
  background:
    radial-gradient(120% 140% at 85% -20%,
      color-mix(in oklch, var(--color-tertiary-container) 60%, transparent) 0%, transparent 60%),
    linear-gradient(120deg,
      color-mix(in oklch, var(--color-primary-container) 80%, var(--color-surface)) 0%,
      color-mix(in oklch, var(--color-primary-container) 80%, var(--color-surface)) 55%,
      color-mix(in oklch, var(--color-secondary-container) 80%, var(--color-surface)) 55.5%);
}
.kb-hero-watermark {
  position: absolute;
  right: -6px; bottom: -18px;
  font-size: 110px;
  opacity: 0.14;
  transform: rotate(-12deg);
  pointer-events: none;
}
.kb-hero-main { grid-area: main; display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.kb-hero-label { font-size: 12.5px; font-weight: 600; color: var(--color-text-dim); }
.kb-hero-balance {
  font-size: 40px;
  font-weight: 850;
  line-height: 1.1;
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
.kb-hero-trend { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.kb-trend {
  display: inline-flex; align-items: center; gap: 3px;
  font-size: 12.5px; font-weight: 700;
  border-radius: var(--radius-full);
  padding: 3px 10px;
}
.kb-trend .material-symbols-outlined { font-size: 14px; }
.kb-trend.in { background: color-mix(in oklch, var(--color-success) 18%, transparent); color: var(--color-success); }
.kb-trend.out { background: color-mix(in oklch, var(--color-text) 8%, transparent); color: var(--color-text-dim); }
.kb-trend-hint { font-size: 11.5px; color: var(--color-text-dim); }

.kb-hero-side { grid-area: side; display: flex; gap: 10px; align-items: flex-start; flex-wrap: wrap; }
.kb-hero-cell {
  background: var(--acrylic-bg-strong);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-md);
  padding: 10px 14px;
  display: flex; flex-direction: column; gap: 3px;
  min-width: 92px;
}
.kb-hero-cell.debt { background: var(--color-error-container); color: var(--color-on-error-container); border-color: transparent; }
.kb-hero-cell-label { font-size: 11px; color: var(--color-text-dim); }
.kb-hero-cell.debt .kb-hero-cell-label { color: inherit; opacity: 0.8; }
.kb-hero-cell-value { font-size: 15px; font-weight: 800; display: inline-flex; align-items: center; gap: 4px; }

.kb-hero-tier { grid-area: tier; display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.kb-tier-bar {
  flex: 1;
  min-width: 140px;
  height: 7px;
  border-radius: var(--radius-full);
  background: color-mix(in oklch, var(--color-text) 10%, transparent);
  overflow: hidden;
}
.kb-tier-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--color-primary), var(--color-tertiary));
}
.kb-tier-hint { font-size: 12px; color: var(--color-text-dim); }
.kb-tier-bar--sm { height: 6px; min-width: 0; margin: 6px 0; }

/* ── Кредитный рейтинг ── */
.kb-credit-rating {
  margin: 4px 0 14px;
  padding: 12px 14px;
  border-radius: var(--radius-md);
  background: color-mix(in oklch, var(--color-primary) 6%, transparent);
  border: 1px solid var(--acrylic-border);
}
.kb-credit-row { display: flex; align-items: baseline; justify-content: space-between; }
.kb-credit-label { font-size: 12.5px; color: var(--color-text-dim); }
.kb-credit-score { font-size: 20px; font-weight: 800; color: var(--color-primary); }
.kb-credit-hint { margin-top: 4px; }
.kb-credit-perk {
  display: flex; align-items: center; gap: 6px; margin-top: 8px;
  color: var(--color-success);
}
.kb-credit-perk .material-symbols-outlined { font-size: 17px; }
.kb-loan-due {
  display: flex; align-items: center; gap: 8px;
  margin: 10px 0; padding: 9px 12px;
  border-radius: var(--radius-md);
  font-size: 12.5px; line-height: 1.4;
  background: color-mix(in oklch, var(--color-tertiary) 10%, transparent);
  color: var(--color-text-dim);
}
.kb-loan-due .material-symbols-outlined { font-size: 18px; color: var(--color-tertiary); }
.kb-loan-due.overdue {
  background: color-mix(in oklch, var(--color-error) 12%, transparent);
  color: var(--color-error);
}
.kb-loan-due.overdue .material-symbols-outlined { color: var(--color-error); }

/* ── Быстрые действия ── */
.kb-actions { display: flex; gap: 10px; margin-top: 16px; flex-wrap: wrap; }
.kb-action {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  border: 1px solid var(--acrylic-border);
  background: var(--acrylic-card-bg);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border-radius: var(--radius-lg, 16px);
  padding: 12px 14px;
  flex: 1 1 0;
  min-width: 92px;
  font: inherit; font-size: 12px; font-weight: 600;
  color: var(--color-text);
  cursor: pointer;
}
.kb-action-icon {
  width: 40px; height: 40px;
  border-radius: 14px;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid; place-items: center;
  font-size: 22px;
}
.kb-action--alert .kb-action-icon { background: var(--color-error-container); color: var(--color-on-error-container); }

/* ── Сетка карточек ── */
.kb-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin-top: 16px;
}
.kb-card {
  background: var(--acrylic-card-bg);
  border: 1px solid var(--acrylic-border);
  border-radius: var(--radius-lg, 20px);
  padding: 18px 20px;
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}
.kb-card--wide { grid-column: 1 / -1; }

.kb-card-head { display: flex; align-items: center; gap: 10px; }
.kb-card-icon {
  width: 42px; height: 42px;
  border-radius: 14px;
  display: grid; place-items: center;
  font-size: 22px;
  flex-shrink: 0;
}
.kb-card-icon--pink { background: color-mix(in oklch, var(--color-error-container) 70%, var(--color-surface)); color: var(--color-error); }
.kb-card-icon--violet { background: var(--color-primary-container); color: var(--color-primary); }
.kb-card-icon--amber { background: color-mix(in oklch, var(--color-warning, var(--color-tertiary)) 22%, transparent); color: var(--color-warning, var(--color-tertiary)); }
.kb-card-icon--teal { background: var(--color-tertiary-container); color: var(--color-tertiary); }
.kb-card-icon--blue { background: var(--color-secondary-container); color: var(--color-secondary); }
.kb-card-title { margin: 0; font-size: 16px; font-weight: 800; flex: 1; min-width: 0; }

.kb-badge {
  font-size: 12px; font-weight: 700;
  border-radius: var(--radius-full);
  padding: 4px 11px;
  background: color-mix(in oklch, var(--color-text) 7%, transparent);
  color: var(--color-text-dim);
  white-space: nowrap;
}
.kb-badge--success { background: color-mix(in oklch, var(--color-success) 16%, transparent); color: var(--color-success); }
.kb-badge--violet { background: var(--color-primary-container); color: var(--color-on-primary-container); }

.kb-head-action {
  display: inline-flex; align-items: center; gap: 3px;
  border: none; background: none;
  color: var(--color-primary);
  font: inherit; font-size: 12.5px; font-weight: 700;
  cursor: pointer; padding: 4px 6px;
  border-radius: var(--radius-sm);
}
.kb-head-action:hover { background: var(--color-primary-container); }
.kb-head-action .material-symbols-outlined { font-size: 16px; }

.kb-card-hint { margin: 0; font-size: 12.5px; color: var(--color-text-dim); line-height: 1.45; }

/* Поле суммы в стиле референса: label + кастомный AmountInput с монетой. */
.kb-field { display: flex; flex-direction: column; gap: 6px; }
.kb-field-label { font-size: 12px; font-weight: 600; color: var(--color-text-dim); }

.kb-row { display: flex; gap: 10px; flex-wrap: wrap; }
.kb-row .btn-grad, .kb-row .btn-glass {
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  flex: 1 1 0;
}

.kb-stats {
  display: flex; align-items: center; gap: 16px;
  background: var(--color-surface);
  background: var(--glass-bg);
  box-shadow: var(--glass-edge);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 12px 16px;
}
.kb-stat { display: flex; flex-direction: column; gap: 3px; flex: 1; }
.kb-stat-label { font-size: 11.5px; color: var(--color-text-dim); }
.kb-stat-value { font-size: 18px; font-weight: 800; display: inline-flex; align-items: center; gap: 4px; }
.kb-stat-value--success { color: var(--color-success); }
.kb-stat-divider { width: 1px; align-self: stretch; background: var(--color-outline-dim); }
.kb-loan-take { display: inline-flex; align-items: center; justify-content: center; gap: 8px; width: 100%; }

/* ── Копилки ── */
.kb-goals { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 8px; }
.kb-goal {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  overflow: hidden;
}
.kb-goal.achieved { border-color: color-mix(in oklch, var(--color-success) 45%, transparent); }
.kb-goal-row {
  display: flex; align-items: center; gap: 10px;
  width: 100%;
  border: none; background: none;
  font: inherit; color: var(--color-text);
  padding: 10px 12px;
  cursor: pointer;
  text-align: left;
}
.kb-goal-emoji { font-size: 22px; flex-shrink: 0; }
.kb-goal-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 5px; }
.kb-goal-title { font-size: 13.5px; font-weight: 700; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kb-goal-bar { height: 5px; border-radius: var(--radius-full); background: var(--color-surface-high); overflow: hidden; display: block; }
.kb-goal-fill { display: block; height: 100%; border-radius: inherit; background: var(--color-primary); }
.kb-goal.achieved .kb-goal-fill { background: var(--color-success); }
.kb-goal-nums { font-size: 12px; font-weight: 700; color: var(--color-text-dim); white-space: nowrap; }
.kb-goal-chevron { font-size: 18px; color: var(--color-text-dim); }
.kb-goal-ops {
  display: flex; gap: 8px; align-items: center; flex-wrap: wrap;
  padding: 0 12px 12px;
}
.kb-goal-input { width: 110px; }
.kb-goal-btn { font-size: 12.5px; padding: 8px 14px; }
.kb-goal-delete {
  margin-left: auto;
  border: none; background: none;
  color: var(--color-error);
  font: inherit; font-size: 12px; font-weight: 600;
  cursor: pointer;
  padding: 4px 6px;
  border-radius: var(--radius-sm);
}
.kb-goal-delete:hover { background: var(--color-error-container); }

/* ── Сборы ── */
.kb-funds { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 12px; }
.kb-fund {
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-md);
  padding: 12px 14px;
  display: flex; flex-direction: column; gap: 8px;
}
.kb-fund.is-done { border-color: color-mix(in oklch, var(--color-success) 45%, transparent); }
.kb-fund.is-closed { opacity: 0.7; }
.kb-fund-head { display: flex; align-items: flex-start; gap: 10px; }
.kb-fund-emoji { font-size: 24px; flex-shrink: 0; }
.kb-fund-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
.kb-fund-title { font-size: 14px; font-weight: 700; }
.kb-fund-desc { font-size: 12px; color: var(--color-text-dim); line-height: 1.4; }
.kb-fund-bar {
  height: 10px;
  border-radius: var(--radius-full);
  background: var(--color-surface-high);
  overflow: hidden;
}
.kb-fund-fill {
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--color-tertiary), var(--color-primary));
  transition: width 0.4s ease;
}
.kb-fund.is-done .kb-fund-fill { background: var(--color-success); }
.kb-fund-meta { display: flex; gap: 6px; flex-wrap: wrap; font-size: 12px; color: var(--color-text-dim); align-items: center; }
.kb-fund-meta strong { color: var(--color-text); }
.kb-fund-mine { color: var(--color-primary); font-weight: 600; }
.kb-fund-donors { display: flex; gap: 4px; }
.kb-fund-donor { width: 24px; height: 24px; border-radius: 50%; object-fit: cover; border: 1.5px solid var(--color-surface); }
.kb-fund-ops { display: flex; gap: 8px; align-items: center; flex-wrap: wrap; }
.kb-fund-input { width: 110px; }
.kb-fund-btn { font-size: 12.5px; padding: 8px 16px; }
.kb-fund-close {
  margin-left: auto;
  border: none; background: none;
  color: var(--color-text-dim);
  font: inherit; font-size: 12px; font-weight: 600;
  cursor: pointer; padding: 4px 6px;
  border-radius: var(--radius-sm);
}
.kb-fund-close:hover { background: var(--color-surface-high); color: var(--color-error); }

/* ── Динамика ── */
.kb-chart {
  display: flex;
  align-items: flex-end;
  gap: 4px;
  height: 110px;
  padding-top: 6px;
}
.kb-chart-day { flex: 1; min-width: 0; display: flex; flex-direction: column; align-items: center; gap: 4px; height: 100%; }
.kb-chart-bars { flex: 1; width: 100%; display: flex; align-items: flex-end; justify-content: center; gap: 2px; }
.kb-chart-bar { width: 5px; border-radius: 3px 3px 0 0; min-height: 2px; display: block; }
.kb-chart-bar.in { background: var(--color-success); }
.kb-chart-bar.out { background: color-mix(in oklch, var(--color-text) 26%, transparent); }
.kb-chart-label { font-size: 9.5px; color: var(--color-text-dim); }
.kb-legend { display: flex; gap: 14px; }
.kb-legend-item { display: inline-flex; align-items: center; gap: 5px; font-size: 11.5px; color: var(--color-text-dim); }
.kb-legend-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.kb-legend-dot.in { background: var(--color-success); }
.kb-legend-dot.out { background: color-mix(in oklch, var(--color-text) 26%, transparent); }

.kb-kinds { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.kb-kinds-title { margin: 0 0 8px; font-size: 12px; font-weight: 700; color: var(--color-text-dim); }
.kb-kind { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.kb-kind-name { font-size: 12px; width: 40%; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kb-kind-bar { flex: 1; height: 6px; border-radius: var(--radius-full); background: var(--color-surface-high); overflow: hidden; }
.kb-kind-fill { display: block; height: 100%; border-radius: inherit; }
.kb-kind-fill.in { background: var(--color-success); }
.kb-kind-fill.out { background: var(--color-primary); }
.kb-kind-val { font-size: 11.5px; font-weight: 700; color: var(--color-text-dim); white-space: nowrap; }

/* ── Топ щедрости ── */
.kb-generous { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.kb-generous-row { display: flex; align-items: center; gap: 10px; font-size: 13.5px; }
.kb-generous-place { width: 22px; text-align: center; }
.kb-generous-avatar { width: 32px; height: 32px; border-radius: 50%; object-fit: cover; }
.kb-generous-name { font-weight: 600; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kb-generous-sent { color: var(--color-text-dim); font-size: 12.5px; display: inline-flex; gap: 3px; align-items: center; }
.kb-generous-thank {
  width: 32px; height: 32px; min-height: 0;
  border: none; border-radius: 50%;
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
  display: grid; place-items: center;
  cursor: pointer;
}
.kb-generous-thank .material-symbols-outlined { font-size: 17px; }

/* ── История ── */
.kb-filters { display: flex; gap: 6px; flex-wrap: wrap; }
.kb-filter {
  border: 1px solid var(--color-outline-dim);
  background: none;
  color: var(--color-text-dim);
  border-radius: var(--radius-full);
  font: inherit; font-size: 12px; font-weight: 600;
  padding: 5px 13px;
  cursor: pointer;
}
.kb-filter.active {
  border-color: var(--color-primary);
  background: var(--color-primary-container);
  color: var(--color-on-primary-container);
}

.kb-days { display: flex; flex-direction: column; gap: 4px; }
.kb-day-head {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 2px 4px;
}
.kb-day-title { font-size: 12.5px; font-weight: 700; color: var(--color-text-dim); }
.kb-day-totals { display: inline-flex; gap: 8px; font-size: 12px; font-weight: 700; }
.kb-day-totals .in { color: var(--color-success); }
.kb-day-totals .out { color: var(--color-text-dim); }

.kb-ledger { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; }
.kb-ledger-row {
  display: flex; align-items: center; gap: 12px;
  padding: 9px 2px;
  border-bottom: 1px solid var(--color-outline-dim);
}
.kb-ledger-row:last-child { border-bottom: none; }
.kb-ledger-icon { font-size: 20px; color: var(--color-text-dim); }
.kb-ledger-body { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 1px; }
.kb-ledger-title { font-size: 13px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kb-ledger-time { font-size: 11px; color: var(--color-text-dim); }
.kb-ledger-delta { font-size: 13.5px; font-weight: 800; flex-shrink: 0; }
.kb-ledger-delta.in { color: var(--color-success); }
.kb-ledger-delta.out { color: var(--color-text-dim); }

.kb-more { align-self: center; }

@media (max-width: 900px) {
  .kb-grid { grid-template-columns: 1fr; }
  .kb-hero { grid-template-columns: 1fr; grid-template-areas: 'main' 'side' 'tier'; }
  .kb-kinds { grid-template-columns: 1fr; }
}
@media (max-width: 560px) {
  .kb-btn-label { display: none; }
  .kb-hero-balance { font-size: 32px; }
  .kb-actions { display: grid; grid-template-columns: repeat(3, 1fr); }
  .kb-action { min-width: 0; }
}
</style>
