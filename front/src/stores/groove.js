import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import * as api from '@/api/groove.js'
import { CELEBRATED_KINDS } from '@/utils/groove.js'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'

export const useGrooveStore = defineStore('groove', () => {
  const events = ref([])
  const hasMore = ref(false)
  const loadingFeed = ref(false)
  const live = ref([])
  const liveLoaded = ref(false)
  // Личный дневной запас зарядов ⚡ (обновляется каждый день).
  const zapsLeft = ref(null)
  const zapsMax = ref(10)
  const pet = ref(null)
  const zoo = ref([])
  const raid = ref(null)
  const shopPrices = ref({})
  const seasonalItems = ref([])
  const seasonTitle = ref('')
  const rareItems = ref([])
  const speciesPrices = ref({})
  const commentsByEvent = ref({})
  const wrapped = ref(null)
  const wrappedLoading = ref(false)
  // Полноэкранный праздник вехи: {kind, payload, at}. Рендерит GrooveCelebration.
  const celebration = ref(null)
  let lastCelebrationKey = ''
  let lastCelebrationAt = 0

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

  // ─────────────────────────── лента ───────────────────────────

  async function fetchFeed() {
    loadingFeed.value = true
    try {
      const page = await api.getFeed()
      events.value = page.items
      hasMore.value = page.has_more
    } finally {
      loadingFeed.value = false
    }
  }

  async function loadMore() {
    if (loadingFeed.value || !hasMore.value || !events.value.length) return
    loadingFeed.value = true
    try {
      const beforeId = events.value[events.value.length - 1].id
      const page = await api.getFeed(beforeId)
      const known = new Set(events.value.map(e => e.id))
      events.value.push(...page.items.filter(e => !known.has(e.id)))
      hasMore.value = page.has_more
    } finally {
      loadingFeed.value = false
    }
  }

  function applyNewEvent(data) {
    if (!isMine(data.company_id)) return
    if (events.value.some(e => e.id === data.id)) return
    events.value.unshift({ my_reactions: [], ...data })
    // Свои вехи (и общая победа над рейдом) — повод для праздника.
    if (CELEBRATED_KINDS.has(data.kind)
      && (data.kind === 'raid_won' || data.user?.id === myId.value)) {
      celebrate(data.kind, data.payload || {})
    }
  }

  // ────────────────────────── праздники ─────────────────────────

  function celebrate(kind, payload) {
    // Дедуп: прямой триггер (ответ на кормление) и сокет-эхо несут одну веху.
    const key = `${kind}:${payload.stage ?? payload.days ?? payload.boss ?? ''}`
    const now = Date.now()
    if (key === lastCelebrationKey && now - lastCelebrationAt < 15000) return
    lastCelebrationKey = key
    lastCelebrationAt = now
    celebration.value = { kind, payload, at: now }
  }

  function clearCelebration() {
    celebration.value = null
  }

  // ─────────────────────────── реакции ──────────────────────────

  async function toggleReaction(event, emoji) {
    const mine = event.my_reactions || (event.my_reactions = [])
    const had = mine.includes(emoji)
    const counts = event.reactions || (event.reactions = {})
    // Оптимистично — сокет-эхо и ответ сервера приведут к точному состоянию.
    if (had) {
      mine.splice(mine.indexOf(emoji), 1)
      counts[emoji] = Math.max(0, (counts[emoji] || 1) - 1)
    } else {
      mine.push(emoji)
      counts[emoji] = (counts[emoji] || 0) + 1
    }
    try {
      const res = await api.toggleReaction(event.id, emoji)
      counts[emoji] = res.count
    } catch {
      if (had) { mine.push(emoji); counts[emoji] = (counts[emoji] || 0) + 1 }
      else { mine.splice(mine.indexOf(emoji), 1); counts[emoji] = Math.max(0, (counts[emoji] || 1) - 1) }
    }
  }

  function applyReaction(data) {
    const event = events.value.find(e => e.id === data.event_id)
    if (!event) return
    const counts = event.reactions || (event.reactions = {})
    counts[data.emoji] = data.count
    if (data.count === 0) delete counts[data.emoji]
    if (data.user_id === myId.value) {
      const mine = event.my_reactions || (event.my_reactions = [])
      const idx = mine.indexOf(data.emoji)
      if (data.added && idx === -1) mine.push(data.emoji)
      if (!data.added && idx !== -1) mine.splice(idx, 1)
    }
  }

  // ───────────────────────── комментарии ────────────────────────

  async function fetchComments(eventId) {
    commentsByEvent.value[eventId] = await api.getComments(eventId)
  }

  async function addComment(eventId, text, replyToId = null) {
    // Запись вернётся и сокет-эхом — applyComment дедуплицирует.
    const comment = await api.addComment(eventId, text, replyToId)
    applyComment({ event_id: eventId, comment })
  }

  function applyComment(data) {
    const list = commentsByEvent.value[data.event_id]
    if (list && !list.some(c => c.id === data.comment.id)) {
      list.push(data.comment)
    }
    const event = events.value.find(e => e.id === data.event_id)
    if (event && (!list || list === commentsByEvent.value[data.event_id])) {
      const known = commentsByEvent.value[data.event_id]
      event.comments_count = known ? known.length : (event.comments_count || 0) + 1
    }
  }

  async function removeComment(eventId, commentId) {
    await api.deleteComment(commentId)
    applyCommentDeleted({ event_id: eventId, comment_id: commentId })
  }

  function applyCommentDeleted(data) {
    const list = commentsByEvent.value[data.event_id]
    if (list) {
      const idx = list.findIndex(c => c.id === data.comment_id)
      if (idx !== -1) list.splice(idx, 1)
    }
    const event = events.value.find(e => e.id === data.event_id)
    if (event) {
      event.comments_count = list ? list.length : Math.max(0, (event.comments_count || 1) - 1)
    }
  }

  // ───────────────────── live, заряды, кудосы ────────────────────

  async function fetchLive() {
    const res = await api.getLive()
    live.value = res.items || []
    zapsLeft.value = res.zaps_left ?? null
    zapsMax.value = res.zaps_max ?? 10
    liveLoaded.value = true
  }

  function applyZapCount(data) {
    const entry = live.value.find(u => u.unit_id === data.unit_id)
    if (entry) entry.zaps = data.zaps
  }

  async function zap(toUserId) {
    const res = await api.sendZap(toUserId)
    const entry = live.value.find(u => u.user?.id === toUserId)
    if (entry) entry.zaps = res.zaps
    if (res.zaps_left != null) zapsLeft.value = res.zaps_left
  }

  const sendKudos = (toUserId, text) => api.sendKudos(toUserId, text)

  // ─────────────────────────── питомец ──────────────────────────

  async function fetchPet() {
    pet.value = await api.getMyPet()
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

  async function buyItem(item) {
    pet.value = { ...pet.value, ...(await api.buyItem(item)) }
  }

  async function fetchShop() {
    const state = await api.getShop()
    shopPrices.value = state.prices || {}
    seasonalItems.value = state.seasonal_items || []
    seasonTitle.value = state.season_title || ''
    rareItems.value = state.rare_items || []
    speciesPrices.value = state.species_prices || {}
  }

  async function buySpecies(species) {
    pet.value = { ...pet.value, ...(await api.buySpecies(species)) }
  }

  async function switchSpecies(species) {
    pet.value = { ...pet.value, ...(await api.switchSpecies(species)) }
  }

  async function claimQuest() {
    pet.value = { ...pet.value, ...(await api.claimQuest()) }
  }

  async function fetchWrapped() {
    wrappedLoading.value = true
    try {
      wrapped.value = await api.getWrapped()
    } finally {
      wrappedLoading.value = false
    }
  }

  const shareWrapped = () => api.shareWrapped()

  function applyPetUpdate(data) {
    // Приходит только в свою user-комнату — синхронизация вкладок владельца.
    if (pet.value && data.user_id !== pet.value.user_id) return
    pet.value = { ...pet.value, ...data }
    const entry = zoo.value.find(p => p.user_id === data.user_id)
    if (entry) Object.assign(entry, data)
  }

  // ─────────────────── локация и погода Грувика ─────────────────

  const location = ref(null)
  const weather = ref(null)
  const locationLoaded = ref(false)

  async function fetchLocation() {
    try {
      const res = await api.getLocation()
      location.value = res.location
      weather.value = res.weather
    } finally {
      locationLoaded.value = true
    }
  }

  async function saveLocation(payload) {
    const res = await api.setLocation(payload)
    location.value = res.location
    weather.value = res.weather
  }

  async function removeLocation() {
    await api.deleteLocation()
    location.value = null
    weather.value = null
  }

  // ─────────────────────────── зоопарк ──────────────────────────

  async function fetchZoo() {
    zoo.value = await api.getZoo()
  }

  async function strokePet(userId) {
    const res = await api.strokePet(userId)
    const entry = zoo.value.find(p => p.user_id === userId)
    if (entry) {
      entry.strokes_today = res.strokes_today
      entry.stroked_by_me = true
    }
  }

  // ──────────────────────────── рейд ────────────────────────────

  async function fetchRaid() {
    raid.value = await api.getRaid()
  }

  function applyRaidUpdate(data) {
    if (!isMine(data.company_id)) return
    if (!raid.value) return
    raid.value = {
      ...raid.value,
      progress: data.progress,
      target: data.target,
      boss: data.boss,
      defeated: data.defeated,
    }
    if (data.defeated_now) fetchPet().catch(() => {})
  }

  return {
    events, hasMore, loadingFeed, live, liveLoaded, zapsLeft, zapsMax, pet, zoo, raid,
    shopPrices, seasonalItems, seasonTitle, rareItems, speciesPrices, commentsByEvent,
    wrapped, wrappedLoading, celebration, myId, myCompanyId, isMine,
    fetchFeed, loadMore, applyNewEvent, celebrate, clearCelebration,
    toggleReaction, applyReaction,
    fetchComments, addComment, applyComment, removeComment, applyCommentDeleted,
    fetchLive, applyZapCount, zap, sendKudos,
    fetchPet, feedPet, renamePet, equipItem, buyItem, fetchShop,
    buySpecies, switchSpecies, claimQuest, applyPetUpdate,
    location, weather, locationLoaded, fetchLocation, saveLocation, removeLocation,
    fetchZoo, strokePet,
    fetchRaid, applyRaidUpdate,
    fetchWrapped, shareWrapped,
  }
})
