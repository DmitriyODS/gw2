// Ведётся вручную: REST подписок и магазина живёт в billingsvc (back-go/billing).
import { apiRequest } from './client.js'

function qs(params = {}) {
  const sp = new URLSearchParams()
  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') sp.set(k, v) })
  return sp.toString()
}

// ── Витрина и подписка ───────────────────────────────────────────────────────

// Всё для вкладки «Подписки»: тариф, расход места и токенов, линейка и аддоны.
export const getShowcase = () => apiRequest('/billing/showcase')

// Лимиты пользователя либо его компании (?company_id=).
export const getEntitlements = (companyId) =>
  apiRequest(`/billing/entitlements?${qs({ company_id: companyId })}`)

// Раздел «Настройки → Хранилище»: лимит, расход по разделам и крупнейшие файлы.
export const getStorage = () => apiRequest('/billing/storage')

// Удалить выбранные файлы — каждый снимает его сервис-владелец.
export const deleteStorageFiles = (keys) =>
  apiRequest('/billing/storage/files', { method: 'DELETE', body: { keys } })

// Сверка с владельцами: убрать потерявшие хозяина файлы, доучесть незнакомые
// и пересчитать занятое место.
export const sweepStorage = () => apiRequest('/billing/storage/sweep', { method: 'POST' })

export const getAiUsage = () => apiRequest('/billing/ai')

// Расчёт стоимости до оформления: { kind, item_code|product_id, period, qty, promo }.
export const quote = (body) => apiRequest('/billing/quote', { method: 'POST', body })

// Оформление заказа тем же телом, что и расчёт.
export const purchase = (body) => apiRequest('/billing/purchase', { method: 'POST', body })

export const activatePromo = (code) =>
  apiRequest('/billing/promo/activate', { method: 'POST', body: { code } })

export const setAutoRenew = (enabled) =>
  apiRequest('/billing/subscription/auto-renew', { method: 'POST', body: { enabled } })

export const cancelAddon = (id) => apiRequest(`/billing/addons/${id}`, { method: 'DELETE' })

// ── Заказы ───────────────────────────────────────────────────────────────────

export const getOrders = (limit = 30, offset = 0) =>
  apiRequest(`/billing/orders?${qs({ limit, offset })}`)

export const cancelOrder = (id) => apiRequest(`/billing/orders/${id}/cancel`, { method: 'POST' })

// ── Товары ───────────────────────────────────────────────────────────────────

export const getProducts = (params = {}) => apiRequest(`/billing/products?${qs(params)}`)

export const getProduct = (id) => apiRequest(`/billing/products/${id}`)

// «Мои товары»: купленное, выставленное на продажу и кошелёк автора.
export const getMyStore = () => apiRequest('/billing/my')

export const createMyProduct = (body) =>
  apiRequest('/billing/my/products', { method: 'POST', body })

export const updateMyProduct = (id, body) =>
  apiRequest(`/billing/my/products/${id}`, { method: 'PATCH', body })

export const submitMyProduct = (id) =>
  apiRequest(`/billing/my/products/${id}/submit`, { method: 'POST' })

export const withdrawMyProduct = (id) =>
  apiRequest(`/billing/my/products/${id}/withdraw`, { method: 'POST' })

export const deleteMyProduct = (id) =>
  apiRequest(`/billing/my/products/${id}`, { method: 'DELETE' })

export const requestPayout = (amount, requisites) =>
  apiRequest('/billing/my/payouts', { method: 'POST', body: { amount, requisites } })

// ── Аудит платформы (супер-админ) ────────────────────────────────────────────

export const adminOverview = () => apiRequest('/billing/admin/overview')

export const adminUpdatePlan = (code, body) =>
  apiRequest(`/billing/admin/plans/${code}`, { method: 'PATCH', body })

export const adminUpdateAddon = (code, body) =>
  apiRequest(`/billing/admin/addons/${code}`, { method: 'PATCH', body })

export const adminGetSettings = () => apiRequest('/billing/admin/settings')

export const adminUpdateSettings = (body) =>
  apiRequest('/billing/admin/settings', { method: 'PATCH', body })

export const adminSubscriptions = (params = {}) =>
  apiRequest(`/billing/admin/subscriptions?${qs(params)}`)

export const adminGrantSubscription = (body) =>
  apiRequest('/billing/admin/subscriptions/grant', { method: 'POST', body })

export const adminRevokeSubscription = (userId) =>
  apiRequest(`/billing/admin/subscriptions/${userId}`, { method: 'DELETE' })

export const adminGrantTokens = (userId, tokens) =>
  apiRequest('/billing/admin/tokens/grant', { method: 'POST', body: { user_id: userId, tokens } })

export const adminResetTokens = (userId) =>
  apiRequest('/billing/admin/tokens/reset', { method: 'POST', body: { user_id: userId } })

export const adminPromos = () => apiRequest('/billing/admin/promos')

export const adminCreatePromo = (body) =>
  apiRequest('/billing/admin/promos', { method: 'POST', body })

export const adminUpdatePromo = (id, body) =>
  apiRequest(`/billing/admin/promos/${id}`, { method: 'PATCH', body })

export const adminDeletePromo = (id) =>
  apiRequest(`/billing/admin/promos/${id}`, { method: 'DELETE' })

export const adminProducts = () => apiRequest('/billing/admin/products')

export const adminCreateProduct = (body) =>
  apiRequest('/billing/admin/products', { method: 'POST', body })

export const adminUpdateProduct = (id, body) =>
  apiRequest(`/billing/admin/products/${id}`, { method: 'PATCH', body })

export const adminDeleteProduct = (id) =>
  apiRequest(`/billing/admin/products/${id}`, { method: 'DELETE' })

export const adminReviewProduct = (id, approve, reason = '') =>
  apiRequest(`/billing/admin/products/${id}/review`, { method: 'POST', body: { approve, reason } })

export const adminOrders = (params = {}) => apiRequest(`/billing/admin/orders?${qs(params)}`)

export const adminConfirmOrder = (id) =>
  apiRequest(`/billing/admin/orders/${id}/confirm`, { method: 'POST' })

export const adminPayouts = () => apiRequest('/billing/admin/payouts')

export const adminProcessPayout = (id, status, note = '') =>
  apiRequest(`/billing/admin/payouts/${id}`, { method: 'POST', body: { status, note } })

export const adminAudit = (params = {}) => apiRequest(`/billing/admin/audit?${qs(params)}`)

export const adminSearchUsers = (q) => apiRequest(`/billing/admin/users?${qs({ q })}`)
