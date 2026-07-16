import { apiRequest } from './client.js'

// ─────────────────────────── питомец ───────────────────────────

export const getMyPet = () => apiRequest('/pets/pet')
export const feedPet = () => apiRequest('/pets/pet/feed', { method: 'POST' })
export const renamePet = (name) => apiRequest('/pets/pet/name', { method: 'POST', body: { name } })
export const equipItem = (item) => apiRequest('/pets/pet/equip', { method: 'POST', body: { item } })
export const switchSpecies = (species) =>
  apiRequest('/pets/pet/species', { method: 'POST', body: { species } })
// Сброс купленного облика — возврат к природному виду.
export const resetSpecies = () =>
  apiRequest('/pets/pet/species', { method: 'DELETE' })
export const claimQuest = () => apiRequest('/pets/pet/quest/claim', { method: 'POST' })
export const startAdventure = () => apiRequest('/pets/pet/adventure', { method: 'POST' })
// Досрочный возврат из приключения — платный (AdventureRecallCost), без награды.
export const recallAdventure = () => apiRequest('/pets/pet/adventure/recall', { method: 'POST' })
// Перерождение «Легенды»: поколение +1, стадия/XP в ноль, богатство остаётся.
export const prestigePet = () => apiRequest('/pets/pet/prestige', { method: 'POST' })

// ─────────────────────── сезонный трек ─────────────────────────

export const getSeason = () => apiRequest('/pets/season')
export const claimSeasonReward = (threshold) =>
  apiRequest('/pets/season/claim', { method: 'POST', body: { threshold } })

// ─────────────────────────── домик ─────────────────────────────

export const getHouse = () => apiRequest('/pets/house')
export const buyHouseDecor = (item) =>
  apiRequest('/pets/house/buy', { method: 'POST', body: { item } })
export const arrangeHouse = (placed) =>
  apiRequest('/pets/house/arrange', { method: 'POST', body: { placed } })
export const setHouseTheme = (theme) =>
  apiRequest('/pets/house/theme', { method: 'POST', body: { theme } })
export const setHousePetPos = (x, y) =>
  apiRequest('/pets/house/pet-pos', { method: 'POST', body: { x, y } })

// ─────────────────────────── магазин ───────────────────────────

export const getShop = () => apiRequest('/pets/shop')
export const getMysteryItem = () => apiRequest('/pets/shop/mystery')
export const buyItem = (item) => apiRequest('/pets/shop/buy', { method: 'POST', body: { item } })
export const buySpecies = (species) =>
  apiRequest('/pets/shop/buy-species', { method: 'POST', body: { species } })

// ───────── прогулка / лечение / сон / купание / поглаживание ────

export const walkPet = () => apiRequest('/pets/walk', { method: 'POST' })
export const healPet = () => apiRequest('/pets/heal', { method: 'POST' })
// Сон — единственное бесплатное действие ухода (восполняет энергию).
export const sleepPet = () => apiRequest('/pets/sleep', { method: 'POST' })
export const bathPet = () => apiRequest('/pets/bath', { method: 'POST' })
export const strokePet = (userId) => apiRequest(`/pets/stroke/${userId}`, { method: 'POST' })

// ────────────────────── зоопарк / рейтинг / эфир ────────────────

export const getZoo = () => apiRequest('/pets/zoo')
// Удаление питомца сотрудника — только администратор компании.
export const deleteColleaguePet = (userId) =>
  apiRequest(`/pets/zoo/${userId}`, { method: 'DELETE' })
export const getRating = () => apiRequest('/pets/rating')
export const getLive = () => apiRequest('/pets/live')
export const getActivityLog = () => apiRequest('/pets/activity')

// ─────────────────────────── кудо-банк ──────────────────────────

export const getBank = () => apiRequest('/pets/bank')
export const getBankLedger = (beforeId = 0) =>
  apiRequest(`/pets/bank/ledger${beforeId ? `?before_id=${beforeId}` : ''}`)
export const transferKudos = (toUserId, amount, comment = '') =>
  apiRequest('/pets/bank/transfer', {
    method: 'POST',
    body: { to_user_id: toUserId, amount, comment },
  })
export const bankDeposit = (amount) =>
  apiRequest('/pets/bank/deposit', { method: 'POST', body: { amount } })
export const bankWithdraw = (amount) =>
  apiRequest('/pets/bank/withdraw', { method: 'POST', body: { amount } })
export const bankTakeLoan = (amount) =>
  apiRequest('/pets/bank/loan', { method: 'POST', body: { amount } })
export const bankRepayLoan = (amount) =>
  apiRequest('/pets/bank/loan/repay', { method: 'POST', body: { amount } })
export const getBankStats = () => apiRequest('/pets/bank/stats')

// Копилки-цели.
export const createGoal = (title, emoji, target) =>
  apiRequest('/pets/bank/goals', { method: 'POST', body: { title, emoji, target } })
export const goalDeposit = (goalId, amount) =>
  apiRequest(`/pets/bank/goals/${goalId}/deposit`, { method: 'POST', body: { amount } })
export const goalWithdraw = (goalId, amount) =>
  apiRequest(`/pets/bank/goals/${goalId}/withdraw`, { method: 'POST', body: { amount } })
export const deleteGoal = (goalId) =>
  apiRequest(`/pets/bank/goals/${goalId}`, { method: 'DELETE' })

// Благотворительные сборы компании.
export const createFund = ({ title, description, emoji, target }) =>
  apiRequest('/pets/bank/funds', { method: 'POST', body: { title, description, emoji, target } })
export const donateFund = (fundId, amount) =>
  apiRequest(`/pets/bank/funds/${fundId}/donate`, { method: 'POST', body: { amount } })
export const closeFund = (fundId) =>
  apiRequest(`/pets/bank/funds/${fundId}/close`, { method: 'POST' })
