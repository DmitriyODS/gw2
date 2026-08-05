// Оснастка интеграционных сценариев: работает КАК setupFile (ставит глобальный
// fetch-шим и jsdom-заглушки при импорте) И как модуль-хелпер (Session, доступ
// к БД, готовность стенда). Сценарии ходят через фронтовые api-модули/сторы —
// шим маршрутизирует относительные '/api/...' на localhost-порты сервисов и
// ведёт cookie-jar (refresh_token) отдельно на каждого пользователя-Session.
import { describe } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { execFileSync } from 'node:child_process'
import fs from 'node:fs'
import { routeBase, STATUS_FILE } from './config.js'

// ── Статус стенда (пишет globalSetup) ──────────────────────────
let status = { ready: false }
try { status = JSON.parse(fs.readFileSync(STATUS_FILE, 'utf8')) } catch { /* нет файла — стенд недоступен */ }
export const INTEGRATION_READY = !!status.ready
// describeIntegration — обычный describe при поднятом стенде, иначе skip
// (docker/postgres/go недоступны — прогон не падает).
export const describeIntegration = INTEGRATION_READY ? describe : describe.skip

// ── jsdom-заглушки, которых нет по умолчанию ───────────────────
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = () => ({
    matches: false, media: '', addEventListener() {}, removeEventListener() {},
    addListener() {}, removeListener() {}, dispatchEvent() { return false },
  })
}

// ── Cookie-jar и fetch-шим ─────────────────────────────────────
// Активный jar (Cookie) переключается вместе с активной Session — так один
// сценарий может действовать от нескольких пользователей по очереди.
let currentJar = new Map()

function jarHeader(jar) {
  const parts = []
  for (const [k, v] of jar) if (v) parts.push(`${k}=${v}`)
  return parts.join('; ')
}

function storeSetCookie(jar, raw) {
  // "name=value; Path=/; HttpOnly; Max-Age=0" — берём name=value; гасим при
  // обнулении (logout чистит refresh_token через Max-Age=0/пустое значение).
  const [pair, ...attrs] = raw.split(';')
  const eq = pair.indexOf('=')
  if (eq < 0) return
  const name = pair.slice(0, eq).trim()
  const value = pair.slice(eq + 1).trim()
  const cleared = value === '' || attrs.some((a) => /max-age=0\b/i.test(a.trim()))
  if (cleared) jar.delete(name)
  else jar.set(name, value)
}

const realFetch = globalThis.fetch
if (!realFetch) throw new Error('нет глобального fetch (Node < 18?)')

// jsdom-овские FormData/File node-fetch не понимает (это чужие для undici
// объекты) — тело уходит пустым, и сервер отвечает «файл не передан». Поэтому
// multipart собираем сами: так работают все файловые ручки (аватар, вложения,
// импорт заметок, восстановление из архива).
async function multipart(form) {
  const boundary = '----gwIntegration' + Math.random().toString(36).slice(2)
  const chunks = []
  const push = (s) => chunks.push(Buffer.from(s, 'utf8'))
  for (const [name, value] of form.entries()) {
    push(`--${boundary}\r\n`)
    if (value && typeof value.arrayBuffer === 'function') {
      const filename = value.name || 'file'
      const type = value.type || 'application/octet-stream'
      // Имя файла с кириллицей передаём по RFC 5987: иначе сервер видит
      // испорченное имя и не узнаёт расширение (импорт .txt/.json).
      const ascii = /^[\x20-\x7e]*$/.test(filename)
      const disp = ascii
        ? `filename="${filename}"`
        : `filename="file"; filename*=UTF-8''${encodeURIComponent(filename)}`
      push(`Content-Disposition: form-data; name="${name}"; ${disp}\r\n`)
      push(`Content-Type: ${type}\r\n\r\n`)
      chunks.push(Buffer.from(await value.arrayBuffer()))
      push('\r\n')
    } else {
      push(`Content-Disposition: form-data; name="${name}"\r\n\r\n${value}\r\n`)
    }
  }
  push(`--${boundary}--\r\n`)
  return { body: Buffer.concat(chunks), contentType: `multipart/form-data; boundary=${boundary}` }
}

function isFormData(body) {
  return !!body && typeof body === 'object' && typeof body.entries === 'function'
    && body[Symbol.toStringTag] === 'FormData'
}

globalThis.fetch = async function shimFetch(input, init = {}) {
  const url = typeof input === 'string' ? input : (input?.url ?? String(input))
  if (!url.startsWith('/api')) return realFetch(input, init)

  const base = routeBase(url)
  if (base === null) {
    // Сервис намеренно не поднят — эмулируем недоступность (как обрыв сети):
    // client.js поймает и бросит NETWORK_ERROR. Такие модули помечаются skip.
    throw new TypeError('fetch failed: service not started')
  }
  if (base === undefined) {
    return new Response(JSON.stringify({ error: 'not_found' }), {
      status: 404, headers: { 'Content-Type': 'application/json' },
    })
  }

  const headers = new Headers(init.headers || {})
  const cookie = jarHeader(currentJar)
  if (cookie) headers.set('Cookie', cookie)

  let body = init.body
  if (isFormData(body)) {
    const part = await multipart(body)
    body = part.body
    headers.set('Content-Type', part.contentType)
  }

  const resp = await realFetch(base + url, { ...init, body, headers, redirect: 'manual' })

  const setCookies = typeof resp.headers.getSetCookie === 'function' ? resp.headers.getSetCookie() : []
  for (const c of setCookies) storeSetCookie(currentJar, c)
  return resp
}

// ── Session: изолированный пользователь (свой pinia + свой cookie-jar) ──
export class Session {
  constructor(label = '') {
    this.label = label
    this.pinia = createPinia()
    this.jar = new Map()
  }
  // use — сделать активными этот pinia и cookie-jar (перед действиями от лица
  // данного пользователя). Возвращает себя для цепочек.
  use() {
    setActivePinia(this.pinia)
    currentJar = this.jar
    return this
  }
}

// ── Доступ к тестовой БД (через psql в контейнере) ─────────────
const pgContainer = status.pgContainer || 'gw2-apitest-db'
const dbName = 'gw2_apitest_front'

// dbQuery — строки результата как массив массивов строковых полей.
export function dbQuery(sql) {
  const out = execFileSync('docker', ['exec', pgContainer,
    'psql', '-U', 'grovework', '-d', dbName, '-t', '-A', '-F', '|', '-c', sql],
    { encoding: 'utf8' })
  return out.trim().split('\n').filter((r) => r.length).map((r) => r.split('|'))
}

// verificationCode — код подтверждения email из БД (надёжнее парсинга письма).
export function verificationCode(email) {
  const rows = dbQuery(`SELECT v.code FROM email_verifications v
    JOIN users u ON u.id = v.user_id WHERE lower(u.email) = lower('${email}')`)
  return rows[0]?.[0] || ''
}

// ── Уникальные имена в рамках прогона ──────────────────────────
const runId = 't' + Date.now().toString(36).slice(-6)
let seq = 0
export function uniq(prefix) { return `${prefix}${runId}${++seq}` }
