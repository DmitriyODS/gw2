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

// ─────────────────────────── магазин ───────────────────────────

export const getShop = () => apiRequest('/pets/shop')
export const getMysteryItem = () => apiRequest('/pets/shop/mystery')
export const buyItem = (item) => apiRequest('/pets/shop/buy', { method: 'POST', body: { item } })
export const buySpecies = (species) =>
  apiRequest('/pets/shop/buy-species', { method: 'POST', body: { species } })

// ─────────────────── прогулка / лечение / поглаживание ─────────

export const walkPet = () => apiRequest('/pets/walk', { method: 'POST' })
export const healPet = () => apiRequest('/pets/heal', { method: 'POST' })
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
