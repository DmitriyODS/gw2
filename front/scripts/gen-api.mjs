#!/usr/bin/env node
// Генератор API-клиента из Swagger-спецификации бэкенда.
// Запуск: npm run gen:api
// Требует запущенного бэкенда на VITE_API_URL или http://localhost:5001

import { writeFileSync } from 'fs'
import { resolve, dirname } from 'path'
import { fileURLToPath } from 'url'

const SPEC_URL = (process.env.VITE_API_URL ?? 'http://localhost:5001') + '/apispec.json'
const __dir = dirname(fileURLToPath(import.meta.url))
const OUT_DIR = resolve(__dir, '../src/api')

// ─── Явные имена функций для эндпоинтов, которые нельзя вывести из пути ───────
const NAME_OVERRIDES = {
  'post /api/auth/login':               'login',
  'post /api/auth/logout':              'logout',
  'post /api/auth/refresh':             'refreshToken',
  'post /api/auth/change-default':      'changeDefault',

  'get /api/backup/export':             'exportBackup',
  'post /api/backup/import':            'importBackup',

  'get /api/departments':               'getDepartments',
  'post /api/departments':              'createDepartment',
  'patch /api/departments/{dept_id}':   'updateDepartment',
  'delete /api/departments/{dept_id}':  'deleteDepartment',

  'get /api/roles':                     'getRoles',
  'post /api/roles':                    'createRole',
  'patch /api/roles/{role_id}':         'updateRole',
  'delete /api/roles/{role_id}':        'deleteRole',

  'get /api/stats/common':              'getStatsCommon',
  'get /api/stats/common/export':       'exportStatsCommon',
  'get /api/stats/extended':            'getStatsExtended',
  'get /api/stats/extended/export':     'exportStatsExtended',
  'get /api/stats/profile':             'getStatsProfile',

  'get /api/tasks':                     'getTasks',
  'post /api/tasks':                    'createTask',
  'get /api/tasks/{task_id}':           'getTask',
  'patch /api/tasks/{task_id}':         'updateTask',
  'delete /api/tasks/{task_id}':        'deleteTask',
  'post /api/tasks/{task_id}/archive':  'archiveTask',
  'post /api/tasks/{task_id}/restore':  'restoreTask',
  'post /api/tasks/{task_id}/favorite': 'toggleFavorite',
  'get /api/tasks/{task_id}/units':     'getUnits',
  'post /api/tasks/{task_id}/units':    'createUnit',

  'get /api/unit-types':                'getUnitTypes',
  'post /api/unit-types':               'createUnitType',
  'patch /api/unit-types/{type_id}':    'updateUnitType',
  'delete /api/unit-types/{type_id}':   'deleteUnitType',

  'get /api/units/active':              'getActiveUnit',
  'patch /api/units/{unit_id}':         'updateUnit',
  'delete /api/units/{unit_id}':        'deleteUnit',
  'post /api/units/{unit_id}/stop':     'stopUnit',

  'get /api/users':                     'getUsers',
  'post /api/users':                    'createUser',
  'get /api/users/me':                  'getMe',
  'patch /api/users/me':                'updateMe',
  'post /api/users/me/avatar':          'uploadAvatar',
  'delete /api/users/me/avatar':        'deleteAvatar',
  'get /api/users/{user_id}':           'getUser',
  'patch /api/users/{user_id}':         'updateUser',
  'delete /api/users/{user_id}':        'deleteUser',
  'patch /api/users/{user_id}/role':    'assignRole',
}

// Эндпоинты с бинарным ответом (blob: true)
const BLOB_OPS = new Set([
  'get /api/backup/export',
  'get /api/stats/common/export',
  'get /api/stats/extended/export',
])

// Эндпоинты, которые используются как URL напрямую (img src) — не генерировать функцию
const SKIP_OPS = new Set([
  'get /api/users/{user_id}/identicon',
])

// Теги, чьи клиенты ведутся ВРУЧНУЮ (REST уехал в Go-микросервисы, в
// Flask-spec остаётся лишь огрызок или ничего) — их файлы не перезаписываем:
// messenger.js — msgsvc (во Flask остался только exact /api/messenger/presence);
// groove.js — groovesvc; companies.js, roles.js, backup.js — authsvc;
// changelog.js — статика nginx; ai.js — aisvc; tasks.js, units.js,
// unitTypes.js, departments.js, stages.js, stats.js — tasksvc.
const MANUAL_TAGS = new Set([
  'messenger', 'groove', 'companies', 'roles', 'backup', 'changelog', 'ai',
  'tasks', 'units', 'unit-types', 'departments', 'stages', 'stats',
])

// ─── Утилиты ──────────────────────────────────────────────────────────────────

function toCamel(s) {
  return s.replace(/[-_]([a-z])/g, (_, c) => c.toUpperCase())
}

function tagToFilename(tag) {
  return toCamel(tag) + '.js'
}

// Параметры пути: [{name: 'task_id', in: 'path'}] → ['taskId']
function pathParamNames(parameters = []) {
  return parameters.filter(p => p.in === 'path').map(p => toCamel(p.name))
}

// Query-параметры только из пары (from, to) → используем именованный стиль
function isDateRange(parameters = []) {
  const qNames = parameters.filter(p => p.in === 'query').map(p => p.name)
  return qNames.length > 0 && qNames.every(n => n === 'from' || n === 'to')
}

function hasQueryParams(parameters = []) {
  return parameters.some(p => p.in === 'query')
}

function hasJsonBody(op) {
  return !!op.requestBody?.content?.['application/json']
}

function hasMultipartBody(op) {
  return !!op.requestBody?.content?.['multipart/form-data']
}

// Подставить path-параметры в URL
function buildUrl(path, pathParams) {
  const apiPath = path.slice(4) // убрать /api
  if (!pathParams.length) return `'${apiPath}'`
  const tpl = apiPath.replace(/\{([^}]+)\}/g, (_, p) => `\${${toCamel(p)}}`)
  return '`' + tpl + '`'
}

// ─── Генерация одной функции ───────────────────────────────────────────────

function generateFunc(path, method, op) {
  const key = `${method} ${path}`
  if (SKIP_OPS.has(key)) return null

  const funcName = NAME_OVERRIDES[key]
  if (!funcName) {
    console.warn(`  ⚠ Нет override для ${key}, пропускаем`)
    return null
  }

  const params  = op.parameters || []
  const pathPs  = pathParamNames(params)
  const url     = buildUrl(path, pathPs)
  const blob    = BLOB_OPS.has(key)
  const m       = method.toUpperCase()

  const dateRange = isDateRange(params)
  const queryPs   = hasQueryParams(params)
  const jsonBody  = hasJsonBody(op)
  const multipart = hasMultipartBody(op)

  // Сигнатура: pathParams + queryStyle + bodyStyle
  const sig = [...pathPs]
  if (dateRange)     sig.push('from', 'to')
  else if (queryPs)  sig.push('params = {}')
  if (multipart)     sig.push('file')
  else if (jsonBody) sig.push('data')

  const sigStr = sig.join(', ')

  // Тело функции
  let body

  if (dateRange) {
    const qNames = params.filter(p => p.in === 'query').map(p => `  if (${p.name} != null) qs.set('${p.name}', ${p.name})`)
    const blobOpt = blob ? ", { blob: true }" : ''
    body = `(${sigStr}) => {\n` +
           `  const qs = new URLSearchParams()\n` +
           qNames.join('\n') + '\n' +
           `  const q = qs.toString() ? \`?\${qs}\` : ''\n` +
           `  return apiRequest(${url} + q${blobOpt})\n` +
           `}`
  } else if (queryPs) {
    const blobOpt = blob ? ", { blob: true }" : ''
    body = `(${sigStr}) => {\n` +
           `  const qs = new URLSearchParams()\n` +
           `  Object.entries(params).forEach(([k, v]) => { if (v != null && v !== '') qs.set(k, v) })\n` +
           `  return apiRequest(${url} + '?' + qs${blobOpt})\n` +
           `}`
  } else if (multipart) {
    body = `(${sigStr}) => {\n` +
           `  const form = new FormData()\n` +
           `  form.append('file', file)\n` +
           `  return apiRequest(${url}, { method: '${m}', body: form })\n` +
           `}`
  } else if (jsonBody) {
    body = `(${sigStr}) => apiRequest(${url}, { method: '${m}', body: data })`
  } else if (m === 'GET') {
    body = blob
      ? `(${sigStr}) => apiRequest(${url}, { blob: true })`
      : `(${sigStr}) => apiRequest(${url})`
  } else {
    body = `(${sigStr}) => apiRequest(${url}, { method: '${m}' })`
  }

  return `export const ${funcName} = ${body}`
}

// ─── Генерация файла для одного тега ──────────────────────────────────────

function generateFile(ops) {
  const lines = [
    `// Сгенерировано из /apispec.json — не редактировать вручную`,
    `// Перегенерировать: npm run gen:api`,
    `import { apiRequest } from './client.js'`,
    ``,
  ]
  for (const { path, method, op } of ops) {
    const fn = generateFunc(path, method, op)
    if (fn) { lines.push(fn); lines.push('') }
  }
  return lines.join('\n')
}

// ─── main ─────────────────────────────────────────────────────────────────

async function main() {
  console.log(`Fetching spec: ${SPEC_URL}`)
  const res = await fetch(SPEC_URL)
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  const spec = await res.json()

  const byTag = {}
  for (const [path, methods] of Object.entries(spec.paths || {})) {
    for (const [method, op] of Object.entries(methods)) {
      const tag = (op.tags?.[0] ?? 'misc').toLowerCase()
      ;(byTag[tag] ??= []).push({ path, method, op })
    }
  }

  for (const [tag, ops] of Object.entries(byTag)) {
    if (MANUAL_TAGS.has(tag)) {
      console.log(`  ↷ ${tagToFilename(tag)} ведётся вручную — пропускаем`)
      continue
    }
    const filename = tagToFilename(tag)
    const content  = generateFile(ops)
    const outPath  = resolve(OUT_DIR, filename)
    writeFileSync(outPath, content, 'utf8')
    console.log(`  ✓ ${filename} (${ops.length} операций)`)
  }
  console.log('Готово!')
}

main().catch(e => { console.error('Ошибка:', e.message); process.exit(1) })
