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
  const season = ref(null)
  const house = ref(null)

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

  async function resetSpecies() {
    pet.value = { ...pet.value, ...(await api.resetSpecies()) }
  }

  async function claimQuest() {
    pet.value = { ...pet.value, ...(await api.claimQuest()) }
  }

  // Перерождение: сервер отдаёт свежий снапшот (gen+1, стадия 0, яйцо).
  async function prestigePet() {
    const res = await api.prestigePet()
    pet.value = { ...pet.value, ...res }
    return res
  }

  // ─────────────────────── сезонный трек ────────────────────────

  async function fetchSeason() {
    season.value = await api.getSeason()
  }

  async function claimSeasonReward(threshold) {
    season.value = await api.claimSeasonReward(threshold)
    // Награда меняет гардероб/домик/баланс — питомец придёт сокетом, но
    // обновим сразу (идемпотентно).
    await fetchPet().catch(() => {})
  }

  // ─────────────────────────── домик ─────────────────────────────

  async function fetchHouse() {
    house.value = await api.getHouse()
  }

  async function buyHouseDecor(key) {
    house.value = await api.buyHouseDecor(key)
    if (pet.value) pet.value = { ...pet.value, kudos: house.value.kudos }
  }

  async function arrangeHouse(placed) {
    house.value = await api.arrangeHouse(placed)
    if (pet.value) pet.value = { ...pet.value, house_placed: house.value.placed }
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
    // в шапке отражаем сразу (domain.StrokeCost = 2).
    if (pet.value) pet.value = { ...pet.value, kudos: Math.max(0, (pet.value.kudos || 0) - 2) }
    return res
  }

  // ─────────────────────────── зоопарк ──────────────────────────

  async function fetchZoo() {
    zoo.value = await api.getZoo()
  }

  // Удаление питомца сотрудника администратором: оптимистично убираем из
  // зоопарка, сокет pet:deleted продублирует остальным (обработчик идемпотентен).
  async function deleteColleaguePet(userId) {
    await api.deleteColleaguePet(userId)
    zoo.value = zoo.value.filter((p) => p.user_id !== userId)
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

  // ─────────────────────────── кудо-банк ─────────────────────────

  const bank = ref(null)
  const ledger = ref([])
  const ledgerNextBeforeId = ref(null)

  function applyBank(res) {
    bank.value = res
    if (pet.value) pet.value = { ...pet.value, kudos: res.kudos }
    if (res.interest_paid) {
      try {
        useNotificationsStore().success(`Вклад принёс +${res.interest_paid} кудосов процентов`)
      } catch { /* вне Pinia-контекста — не критично */ }
    }
  }

  async function fetchBank() {
    applyBank(await api.getBank())
  }

  // Выписка: первая страница (reset) либо продолжение keyset-курсора.
  async function fetchLedger({ more = false } = {}) {
    const beforeId = more ? ledgerNextBeforeId.value || 0 : 0
    const res = await api.getBankLedger(beforeId)
    ledger.value = more ? [...ledger.value, ...(res.items || [])] : res.items || []
    ledgerNextBeforeId.value = res.next_before_id ?? null
  }

  async function transferKudos(toUserId, amount, comment) {
    applyBank(await api.transferKudos(toUserId, amount, comment))
    await fetchLedger().catch(() => {})
  }

  async function bankDeposit(amount) {
    applyBank(await api.bankDeposit(amount))
  }

  async function bankWithdraw(amount) {
    applyBank(await api.bankWithdraw(amount))
  }

  async function bankTakeLoan(amount) {
    applyBank(await api.bankTakeLoan(amount))
  }

  async function bankRepayLoan(amount) {
    applyBank(await api.bankRepayLoan(amount))
  }

  // Входящий перевод (сокет kudos:received — адресно в мою комнату).
  function applyKudosReceived(data) {
    if (!isMine(data.company_id)) return
    const from = data.from?.fio ? ` от ${data.from.fio}` : ''
    const note = data.comment ? ` — «${data.comment}»` : ''
    try {
      useNotificationsStore().success(`+${data.amount} кудосов${from}${note}`)
    } catch { /* noop */ }
    // Баланс придёт авторитетным pet:update; сводку банка освежаем, если открыта.
    if (bank.value) fetchBank().catch(() => {})
  }

  // ───────────────────────── сокет-события ───────────────────────

  function applyPetUpdate(data) {
    // Приходит только в свою user-комнату — синхронизация вкладок владельца.
    if (pet.value && data.user_id !== pet.value.user_id) return
    pet.value = { ...pet.value, ...data }
    const entry = zoo.value.find((p) => p.user_id === data.user_id)
    if (entry) Object.assign(entry, data)
  }

  // Администратор удалил питомца: убираем из зоопарка; владельцу — свежий
  // питомец пересоздастся штатным GET (пересобираем сразу же).
  function applyPetDeleted(data) {
    if (!isMine(data.company_id)) return
    zoo.value = zoo.value.filter((p) => p.user_id !== data.user_id)
    if (myId.value === data.user_id) {
      pet.value = null
      fetchPet().catch(() => {})
    }
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
    season.value = null
    house.value = null
    bank.value = null
    ledger.value = []
    ledgerNextBeforeId.value = null
  }

  return {
    pet, shop, shopLoaded, zoo, rating, live, liveLoaded, activityLog, activityLoaded,
    season, house, bank, ledger, ledgerNextBeforeId,
    myId, myCompanyId, isMine,
    fetchPet, feedPet, renamePet, equipItem, switchSpecies, resetSpecies, claimQuest, startAdventure,
    prestigePet, fetchSeason, claimSeasonReward, fetchHouse, buyHouseDecor, arrangeHouse,
    fetchShop, buyItem, buySpecies, claimMystery,
    walkPet, healPet, strokePet,
    fetchZoo, deleteColleaguePet, fetchRating, fetchLive, fetchActivityLog,
    fetchBank, fetchLedger, transferKudos, bankDeposit, bankWithdraw, bankTakeLoan, bankRepayLoan,
    applyPetUpdate, applyPetDeleted, applyKudosReceived,
    reset,
  }
})
