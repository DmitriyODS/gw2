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
