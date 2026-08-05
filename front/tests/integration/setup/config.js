// Общая конфигурация интеграционного стенда фронта.
//
// Порты — +10100 к dev-портам, ОТДЕЛЬНЫЙ блок от Go-харнеса (back-go/apitest,
// +10000): два стенда не должны конфликтовать по портам, БД и Redis-базе.
// БД gw2_apitest_front на выделенном Postgres-контейнере gw2-apitest-db:15432
// (тот же контейнер, что у Go-харнеса, но ДРУГАЯ база). Redis :6379 db 8
// (Go-харнес — db 9, dev — db 0). Импортируется и globalSetup (main-процесс),
// и setup-шимом (worker) — единый источник истины.

export const PG = {
  image: 'pgvector/pgvector:pg16',
  container: 'gw2-apitest-db',
  port: '15432',
  adminURL: 'postgresql://grovework:grovework_local@localhost:15432/grovework?sslmode=disable',
  dbName: 'gw2_apitest_front',
  dbURL: 'postgresql://grovework:grovework_local@localhost:15432/gw2_apitest_front?sslmode=disable',
}

export const REDIS = {
  container: 'gw2-apitest-redis',
  port: '6379',
  url: 'redis://localhost:6379/8',
  db: '8',
}

// Почта: приглашение в компанию отправляется fail-closed (без письма получатель
// не узнает токен), поэтому стенду нужен настоящий SMTP-приёмник.
export const MAIL = {
  container: 'gw2-apitest-mailpit',
  smtpPort: '1025',
  webPort: '8025',
}

// Dev-ключи PASETO (синхронно с dev.sh и Go-харнесом).
export const PASETO = {
  privateKey: '68eb779b2f672beb8fcd58d72a81ce1565a1417aed3788d1362bf4faaa3f62ac15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1',
  publicKey: '15ef439747fcad6ca627310942ba14b48f164fcbb5f65c10f61ca2aeb4b53fe1',
  refreshKey: 'd525374c4ec7b5e1c5b140fb9c1f4cffd9c3dbf052bb18f2f32bf9f92d9fa05c',
}

// Fernet-ключ шифрования личных AI-ключей (стендовый; в проде — секрет).
export const AI_ENC_KEY = 'Z3cyLWludGVncmF0aW9uLXN0YW5kLWZlcm5ldC1rZXk='

// HTTP-порты сервисов (для healthz и роутинга шима).
export const HTTP = {
  calls: 18190,
  auth: 18191,
  messenger: 18192,
  ai: 18193,
  pets: 18194,
  tasks: 18195,
  // gateway: 18196 — НЕ поднимается (не нужен сценариям), presence → skip
  push: 18197,
  mail: 18198,
  registry: 18199,
  calendar: 18200,
  diary: 18201,
  portal: 18202,
  notes: 18203,
  board: 18205,
  reminder: 18206,
  billing: 18207,
}

// gRPC-порты (+10100).
export const GRPC = {
  calls: 19190,
  messenger: 19192,
  ai: 19193,
  pets: 19194,
  tasks: 19195,
  mail: 19198,
  notes: 19203,
  billing: 19207,
}

// Файл со статусом стенда: globalSetup пишет его, шим/сценарии читают
// (env-переменные между main-процессом и worker'ами vitest не всегда доходят).
// Путь через process.cwd() (корень front) — import.meta.url в vite-воркере
// теряет абсолютный префикс, а cwd одинаков и в main-процессе, и в воркерах.
import path from 'node:path'
export const STATUS_FILE = path.resolve(process.cwd(), 'tests/integration/setup/.backend-status.json')

// Карта префиксов → базовый URL сервиса. Порядок важен (длинные раньше).
// Значение null — сервис не поднят, запрос по этому префиксу должен «упасть»
// как недоступный (сценарий помечает t.skip).
export function routeBase(path) {
  // path приходит без /api-префикса шиму передаётся полный '/api/...'.
  const p = path
  // ai-settings компании — regex, выигрывает у /api/companies.
  if (/^\/api\/companies\/\d+\/ai-settings/.test(p)) return `http://localhost:${HTTP.ai}`
  const table = [
    ['/api/calls', HTTP.calls],
    ['/api/auth', HTTP.auth],
    ['/api/users', HTTP.auth],
    ['/api/roles', HTTP.auth],
    ['/api/backup', HTTP.auth],
    ['/api/companies', HTTP.auth],
    ['/api/ai', HTTP.ai],
    ['/api/messenger/presence', null], // gateway не поднят (exact)
    ['/api/messenger', HTTP.messenger],
    ['/api/pets', HTTP.pets],
    ['/api/tasks', HTTP.tasks],
    ['/api/units', HTTP.tasks],
    ['/api/unit-types', HTTP.tasks],
    ['/api/departments', HTTP.tasks],
    ['/api/stages', HTTP.tasks],
    ['/api/stats', HTTP.tasks],
    ['/api/yougile', HTTP.tasks],
    ['/api/push', HTTP.push],
    ['/api/registries', HTTP.registry],
    ['/api/calendars', HTTP.calendar],
    ['/api/diaries', HTTP.diary],
    ['/api/portal', HTTP.portal],
    ['/api/notes', HTTP.notes],
    ['/api/boards', HTTP.board],
    ['/api/reminders', HTTP.reminder],
    ['/api/billing', HTTP.billing],
    ['/api/changelog', null], // статика nginx, не сервис
  ]
  // exact-совпадение presence проверяем отдельно (без query).
  const clean = p.split('?')[0]
  if (clean === '/api/messenger/presence') return null
  for (const [prefix, port] of table) {
    if (prefix === '/api/messenger/presence') continue
    if (p.startsWith(prefix)) return port == null ? null : `http://localhost:${port}`
  }
  return undefined // неизвестный /api → 404-подобная ошибка
}
