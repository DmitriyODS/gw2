import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/pets.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'
import { useNotificationsStore } from '@/stores/notifications.js'

export const usePetsStore = defineStore('pets', () => {
  const pet = ref(null)
  const shop = ref([])
  const shopLoaded = ref(false)
  const zoo = ref([])
  const rating = ref(null)
  const live = ref([])
  const liveLoaded = ref(false)
  const activityLog = ref([])
  const activityLoaded = ref(false)

  const myId = computed(() => useAuthStore().user?.id ?? null)

  const myCompanyId = computed(() => {
    const auth = useAuthStore()
    if (auth.user?.company_id != null) return auth.user.company_id
    try { return useCompaniesStore().activeCompanyId ?? null } catch { return null }
  })

  // Сокеты вещают в комнату all — события чужих компаний отфильтровываем тут.
  function isMine(companyId) {
    return companyId == null || myCompanyId.value == null || companyId === myCompanyId.value
  }

  // ─────────────────────────── питомец ──────────────────────────

  async function fetchPet() {
    const res = await api.getMyPet()
    pet.value = res
    // Разовое поле ответа: этот GET зафиксировал возврат из приключения.
    if (res?.adventure_reward) {
      const { kudos, xp } = res.adventure_reward
      try {
        useNotificationsStore().success(`Вернулся из приключения: +${kudos} кудосов, +${xp} XP`)
      } catch { /* вне Pinia-контекста (тесты) — уведомление не критично */ }
    }
  }

  async function startAdventure() {
    const res = await api.startAdventure()
    pet.value = { ...pet.value, ...res }
    return res
  }

  async function feedPet() {
    const res = await api.feedPet()
    pet.value = { ...pet.value, ...res }
    return res
  }

  async function renamePet(name) {
    pet.value = { ...pet.value, ...(await api.renamePet(name)) }
  }

  async function equipItem(item) {
    pet.value = { ...pet.value, ...(await api.equipItem(item)) }
  }

  async function switchSpecies(species) {
    pet.value = { ...pet.value, ...(await api.switchSpecies(species)) }
  }

  async function claimQuest() {
    pet.value = { ...pet.value, ...(await api.claimQuest()) }
  }

  // ─────────────────────────── магазин ──────────────────────────

  async function fetchShop() {
    const res = await api.getShop()
    shop.value = res.items || []
    shopLoaded.value = true
  }

  async function buyItem(item) {
    const res = await api.buyItem(item)
    pet.value = { ...pet.value, ...res }
    await fetchShop().catch(() => {})
    return res
  }

  async function buySpecies(species) {
    const res = await api.buySpecies(species)
    pet.value = { ...pet.value, ...res }
    await fetchShop().catch(() => {})
    return res
  }

  // GET /shop/mystery — сам вызов И есть получение сюрприза (не отдельный
  // "предпросмотр"), поэтому вызывать один раз и сразу освежать питомца/магазин.
  async function claimMystery() {
    const item = await api.getMysteryItem()
    await Promise.all([fetchPet().catch(() => {}), fetchShop().catch(() => {})])
    return item
  }

  // ─────────────── прогулка / лечение / поглаживание ─────────────

  async function walkPet() {
    const res = await api.walkPet()
    pet.value = { ...pet.value, ...res }
    return res
  }

  async function healPet() {
    const res = await api.healPet()
    pet.value = { ...pet.value, ...res }
    return res
  }

  async function strokePet(ownerUserId) {
    const res = await api.strokePet(ownerUserId)
    const entry = zoo.value.find((p) => p.user_id === ownerUserId)
    if (entry) Object.assign(entry, res)
    // Списание у гладящего придёт сокетом pet:update (авторитетно), но кудосы
    // в шапке отражаем сразу (domain.StrokeCost = 1).
    if (pet.value) pet.value = { ...pet.value, kudos: Math.max(0, (pet.value.kudos || 0) - 1) }
    return res
  }

  // ─────────────────────────── зоопарк ──────────────────────────

  async function fetchZoo() {
    zoo.value = await api.getZoo()
  }

  // ─────────────────────────── рейтинг ──────────────────────────

  async function fetchRating() {
    rating.value = await api.getRating()
  }

  // ───────────────────────── «в эфире» ───────────────────────────

  async function fetchLive() {
    const res = await api.getLive()
    live.value = res.items || []
    liveLoaded.value = true
  }

  // ─────────────────────── история активности ────────────────────

  async function fetchActivityLog() {
    const res = await api.getActivityLog()
    activityLog.value = res.items || []
    activityLoaded.value = true
  }

  // ───────────────────────── сокет-события ───────────────────────

  function applyPetUpdate(data) {
    // Приходит только в свою user-комнату — синхронизация вкладок владельца.
    if (pet.value && data.user_id !== pet.value.user_id) return
    pet.value = { ...pet.value, ...data }
    const entry = zoo.value.find((p) => p.user_id === data.user_id)
    if (entry) Object.assign(entry, data)
  }

  // Смена компании / логаут — питомец и кэши прежней компании не должны
  // показываться новой сессии.
  function reset() {
    pet.value = null
    shop.value = []
    shopLoaded.value = false
    zoo.value = []
    rating.value = null
    live.value = []
    liveLoaded.value = false
    activityLog.value = []
    activityLoaded.value = false
  }

  return {
    pet, shop, shopLoaded, zoo, rating, live, liveLoaded, activityLog, activityLoaded,
    myId, myCompanyId, isMine,
    fetchPet, feedPet, renamePet, equipItem, switchSpecies, claimQuest, startAdventure,
    fetchShop, buyItem, buySpecies, claimMystery,
    walkPet, healPet, strokePet,
    fetchZoo, fetchRating, fetchLive, fetchActivityLog,
    applyPetUpdate,
    reset,
  }
})
