import { apiRequest } from './client.js'

export const getFeed = (beforeId = null, limit = 30) => {
  const params = new URLSearchParams()
  if (beforeId) params.set('before_id', beforeId)
  params.set('limit', limit)
  return apiRequest(`/groove/feed?${params}`)
}

export const toggleReaction = (eventId, emoji) =>
  apiRequest(`/groove/feed/${eventId}/reactions`, { method: 'POST', body: { emoji } })

export const getComments = (eventId) => apiRequest(`/groove/feed/${eventId}/comments`)

export const addComment = (eventId, text, replyToId = null) =>
  apiRequest(`/groove/feed/${eventId}/comments`, { method: 'POST', body: { text, reply_to_id: replyToId } })

export const deleteComment = (commentId) =>
  apiRequest(`/groove/comments/${commentId}`, { method: 'DELETE' })

export const sendKudos = (toUserId, category, text) =>
  apiRequest('/groove/kudos', { method: 'POST', body: { to_user_id: toUserId, category, text } })

export const getLive = () => apiRequest('/groove/live')

export const getMyPet = () => apiRequest('/groove/pet')

export const feedPet = () => apiRequest('/groove/pet/feed', { method: 'POST' })

export const renamePet = (name) =>
  apiRequest('/groove/pet/name', { method: 'POST', body: { name } })

export const equipItem = (item) =>
  apiRequest('/groove/pet/equip', { method: 'POST', body: { item } })

export const getShop = () => apiRequest('/groove/shop')

export const buyItem = (item) =>
  apiRequest('/groove/shop/buy', { method: 'POST', body: { item } })

export const buySpecies = (species) =>
  apiRequest('/groove/shop/buy-species', { method: 'POST', body: { species } })

export const switchSpecies = (species) =>
  apiRequest('/groove/pet/species', { method: 'POST', body: { species } })

export const claimQuest = () =>
  apiRequest('/groove/pet/quest/claim', { method: 'POST' })

export const getZoo = () => apiRequest('/groove/zoo')

export const getRaid = () => apiRequest('/groove/raid')

export const getRating = () => apiRequest('/groove/rating')

export const getMorningBriefing = (part) =>
  apiRequest(`/groove/morning?part=${encodeURIComponent(part)}`, { timeout: 20000 })

export const getWrapped = () => apiRequest('/groove/wrapped', { timeout: 20000 })

export const shareWrapped = () => apiRequest('/groove/wrapped/share', { method: 'POST', timeout: 20000 })

export const getGrooveTv = () => apiRequest('/groove/tv')

export const getLocation = () => apiRequest('/groove/location')

export const setLocation = ({ latitude, longitude, city = null }) =>
  apiRequest('/groove/location', { method: 'PUT', body: { latitude, longitude, city } })

export const deleteLocation = () => apiRequest('/groove/location', { method: 'DELETE' })

export const searchCities = (q) => apiRequest(`/groove/geo?q=${encodeURIComponent(q)}`)
