import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/billing.js'

/* Подписка, лимиты и магазин.

   Лимиты нужны не только магазину: разделы спрашивают у стора, можно ли
   создать ещё одну доску/компанию и доступна ли возможность на тарифе, чтобы
   гасить кнопки ДО запроса к серверу (сервер всё равно проверит сам —
   клиентская проверка это подсказка, а не защита).

   Сокет-события billing:* (покупка, продление, начисление токенов) обновляют
   витрину без перезагрузки. */
export const useBillingStore = defineStore('billing', () => {
  const entitlements = ref(null)
  const subscription = ref(null)
  const plans = ref([])
  const addons = ref([])
  const myAddons = ref([])
  const storage = ref([])
  const planLimits = ref({})
  const settings = ref(null)
  const loading = ref(false)
  const loadedAt = ref(0)

  const plan = computed(() => entitlements.value?.plan || 'junior')
  const planName = computed(() => entitlements.value?.plan_name || 'Джун')
  const limits = computed(() => entitlements.value?.limits || {})
  const storageUsed = computed(() => entitlements.value?.storage_used ?? 0)
  const storageLimit = computed(() => limits.value.storage_bytes ?? -1)
  const tokensLeft = computed(() => entitlements.value?.tokens_left ?? 0)
  const tokensUsed = computed(() => entitlements.value?.tokens_used ?? 0)
  const tokensLimit = computed(() => limits.value.ai_tokens ?? 0)
  const expiresAt = computed(() => subscription.value?.expires_at || null)
  const isFree = computed(() => plan.value === 'junior')

  /** Есть ли возможность на текущем тарифе: has('portal'), has('premium_themes'). */
  function has(feature) {
    return Boolean(limits.value[feature])
  }

  /** Влезает ли ещё одна сущность: room('boards', boards.length). */
  function room(kind, current) {
    const limit = limits.value[kind]
    if (limit == null || limit < 0) return true
    return current < limit
  }

  async function fetchShowcase(force = false) {
    // Лимиты спрашивают многие экраны — не дёргаем сервер чаще раза в минуту.
    if (!force && loadedAt.value && Date.now() - loadedAt.value < 60_000) return
    loading.value = true
    try {
      const data = await api.getShowcase()
      entitlements.value = data.entitlements ?? null
      subscription.value = data.subscription ?? null
      plans.value = data.plans ?? []
      addons.value = data.addons ?? []
      myAddons.value = data.my_addons ?? []
      storage.value = data.storage ?? []
      planLimits.value = data.plan_limits ?? {}
      settings.value = data.settings ?? null
      loadedAt.value = Date.now()
    } finally {
      loading.value = false
    }
  }

  async function purchase(body) {
    const order = await api.purchase(body)
    await fetchShowcase(true)
    return order
  }

  async function activatePromo(code) {
    const res = await api.activatePromo(code)
    await fetchShowcase(true)
    return res
  }

  async function setAutoRenew(enabled) {
    subscription.value = await api.setAutoRenew(enabled)
  }

  async function cancelAddon(id) {
    await api.cancelAddon(id)
    await fetchShowcase(true)
  }

  /** Сокет-события billingsvc: подписка, токены, докупки, покупки товаров. */
  function applyEvent() {
    loadedAt.value = 0
    return fetchShowcase(true)
  }

  function reset() {
    entitlements.value = null
    subscription.value = null
    plans.value = []
    addons.value = []
    myAddons.value = []
    storage.value = []
    planLimits.value = {}
    settings.value = null
    loadedAt.value = 0
  }

  return {
    entitlements, subscription, plans, addons, myAddons, storage, planLimits, settings, loading,
    plan, planName, limits, storageUsed, storageLimit, tokensLeft, tokensUsed, tokensLimit,
    expiresAt, isFree,
    has, room, fetchShowcase, purchase, activatePromo, setAutoRenew, cancelAddon, applyEvent, reset,
  }
})
