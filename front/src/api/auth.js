// Контракт Go-микросервиса авторизации (back-go/auth) — ведётся вручную:
// authsvc не публикует Swagger, npm run gen:api этот файл не трогает.
import { apiRequest } from './client.js'

export const changeDefault = (data) => apiRequest('/auth/change-default', { method: 'POST', body: data })

export const login = (data) => apiRequest('/auth/login', { method: 'POST', body: data })

// Регистрация нового пользователя ({fio, email, login, password}). Сессию НЕ
// выдаёт — отвечает {status:'verification_required', email}; дальше нужно
// подтвердить email кодом/ссылкой.
export const register = (data) => apiRequest('/auth/register', { method: 'POST', body: data })

// Подсказка свободного логина по ФИО (для live-заполнения поля на регистрации).
export const suggestLogin = (fio) =>
  apiRequest(`/auth/suggest-login?fio=${encodeURIComponent(fio)}`, { method: 'GET' })

// Подтверждение email: по ссылке ({token}) или вводом кода ({email, code}).
// Возвращает полноценную сессию (как login).
export const verifyEmail = (data) => apiRequest('/auth/verify-email', { method: 'POST', body: data })

// Повторная отправка письма с кодом подтверждения.
export const resendVerification = (email) =>
  apiRequest('/auth/resend-verification', { method: 'POST', body: { email } })

// Запрос письма со ссылкой сброса пароля (ответ всегда ok — не раскрываем аккаунт).
export const forgotPassword = (email) =>
  apiRequest('/auth/forgot-password', { method: 'POST', body: { email } })

// Установка нового пароля по токену из письма. Возвращает {login} для префилла входа.
export const resetPassword = (token, newPassword) =>
  apiRequest('/auth/reset-password', { method: 'POST', body: { token, new_password: newPassword } })

// Завершение логина выбором компании (когда у пользователя их несколько):
// select_token получен в ответе login с needs_company_selection.
export const selectCompany = (data) => apiRequest('/auth/select-company', { method: 'POST', body: data })

// Смена активной компании в текущей сессии (перевыпуск токенов).
export const switchCompany = (companyId) =>
  apiRequest('/auth/switch-company', { method: 'POST', body: { company_id: companyId } })

// _isLogout — пометка для client.js: даже во время выхода этот запрос должен
// дойти до сервера (через refresh, если access протух), чтобы погасить
// refresh-cookie. Остальные 401 при выходе подавляются без шума.
export const logout = () => apiRequest('/auth/logout', { method: 'POST', _isLogout: true })

export const refreshToken = () => apiRequest('/auth/refresh', { method: 'POST' })
