// Ведётся вручную: REST ролей живёт в authsvc (back-go/auth), в Flask-spec
// его больше нет (MANUAL_TAGS в scripts/gen-api.mjs). Роли фиксированы —
// только чтение.
import { apiRequest } from './client.js'

export const getRoles = () => apiRequest('/roles')
