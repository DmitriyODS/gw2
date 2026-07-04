// globalSetup интеграционного стенда фронта: поднимает реальный бэкенд
// (Postgres + Redis + нужные Go-сервисы) на отдельном от Go-харнеса блоке
// портов/БД, ждёт healthz, пишет статус в файл. Teardown гасит всё.
//
// Нет docker/postgres/go → стенд помечается недоступным (status.ready=false),
// сценарии сами делают describe.skip — прогон НЕ падает.
import { spawn, execFileSync, execFile } from 'node:child_process'
import net from 'node:net'
import http from 'node:http'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { promisify } from 'node:util'
import { PG, REDIS, PASETO, HTTP, GRPC, STATUS_FILE } from './config.js'

const execFileP = promisify(execFile)
const here = path.dirname(fileURLToPath(import.meta.url))

function tcpAlive(host, port, timeoutMs = 2000) {
  return new Promise((resolve) => {
    const sock = net.connect({ host, port }, () => { sock.destroy(); resolve(true) })
    sock.on('error', () => resolve(false))
    sock.setTimeout(timeoutMs, () => { sock.destroy(); resolve(false) })
  })
}

function findRepoRoot() {
  let d = here
  for (let i = 0; i < 8; i++) {
    if (fs.existsSync(path.join(d, 'back-go', 'go.work'))) return d
    d = path.dirname(d)
  }
  throw new Error('back-go/go.work не найден вверх от ' + here)
}

function dockerOk() {
  try { execFileSync('docker', ['info'], { stdio: 'ignore' }); return true }
  catch { return false }
}

async function ensurePostgres() {
  if (!(await tcpAlive('localhost', Number(PG.port)))) {
    try {
      execFileSync('docker', ['start', PG.container], { stdio: 'ignore' })
    } catch {
      execFileSync('docker', ['run', '-d', '--name', PG.container,
        '-e', 'POSTGRES_USER=grovework',
        '-e', 'POSTGRES_PASSWORD=grovework_local',
        '-e', 'POSTGRES_DB=grovework',
        '-p', `127.0.0.1:${PG.port}:5432`, PG.image], { stdio: 'ignore' })
    }
  }
  // Ждём готовности сервера (pg_isready внутри контейнера).
  const deadline = Date.now() + 60_000
  while (Date.now() < deadline) {
    try { execFileSync('docker', ['exec', PG.container, 'pg_isready', '-U', 'grovework'], { stdio: 'ignore' }); return }
    catch { await sleep(500) }
  }
  throw new Error(`postgres ${PG.container} не готов за 60с`)
}

async function ensureRedis() {
  if (await tcpAlive('localhost', Number(REDIS.port))) return // dev/собственный уже поднят
  try {
    execFileSync('docker', ['start', REDIS.container], { stdio: 'ignore' })
  } catch {
    execFileSync('docker', ['run', '-d', '--name', REDIS.container,
      '-p', `127.0.0.1:${REDIS.port}:6379`, 'redis:7-alpine'], { stdio: 'ignore' })
  }
  const deadline = Date.now() + 30_000
  while (Date.now() < deadline) {
    if (await tcpAlive('localhost', Number(REDIS.port))) return
    await sleep(300)
  }
  throw new Error('redis не поднялся за 30с')
}

// Пересоздать тестовую БД через psql внутри контейнера (без node-pg зависимости).
function recreateDB() {
  const psql = (sql) => execFileSync('docker', ['exec', PG.container,
    'psql', '-U', 'grovework', '-d', 'grovework', '-v', 'ON_ERROR_STOP=1', '-c', sql], { stdio: 'ignore' })
  psql(`DROP DATABASE IF EXISTS ${PG.dbName} WITH (FORCE)`)
  psql(`CREATE DATABASE ${PG.dbName}`)
}

// FLUSHDB тестовой базы Redis сырыми RESP-командами (без redis-клиента).
function flushRedis() {
  return new Promise((resolve, reject) => {
    const sock = net.connect({ host: 'localhost', port: Number(REDIS.port) }, () => {
      sock.write(`SELECT ${REDIS.db}\r\nFLUSHDB\r\n`)
    })
    sock.setTimeout(5000, () => { sock.destroy(); reject(new Error('redis flush timeout')) })
    let buf = ''
    sock.on('data', (d) => {
      buf += d.toString()
      if ((buf.match(/\r\n/g) || []).length >= 2) { sock.destroy(); resolve() }
    })
    sock.on('error', reject)
  })
}

async function runMigrations(repoRoot) {
  await execFileP('go', ['run', './cmd/migrate'], {
    cwd: path.join(repoRoot, 'back-go', 'migrate'),
    env: { ...process.env, DATABASE_URL: PG.dbURL },
    timeout: 180_000,
  })
}

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)) }

function waitHealthz(port, timeoutMs = 60_000) {
  const deadline = Date.now() + timeoutMs
  return new Promise((resolve, reject) => {
    const tick = () => {
      const req = http.get(`http://localhost:${port}/healthz`, (res) => {
        res.resume()
        if (res.statusCode === 200) return resolve()
        retry()
      })
      req.on('error', retry)
      req.setTimeout(2000, () => { req.destroy(); retry() })
    }
    const retry = () => {
      if (Date.now() > deadline) return reject(new Error(`сервис :${port} не поднялся за ${timeoutMs}мс`))
      setTimeout(tick, 300)
    }
    tick()
  })
}

// ── Управление процессами сервисов ─────────────────────────────
const procs = []

function startSvc(name, repoRoot, dir, pkg, env) {
  const logs = []
  const child = spawn('go', ['run', pkg], {
    cwd: path.join(repoRoot, dir),
    env: { ...process.env, ...env },
    detached: true, // своя process group — убьём вместе с дочерним бинарём
    stdio: ['ignore', 'pipe', 'pipe'],
  })
  child.stdout.on('data', (d) => logs.push(d.toString()))
  child.stderr.on('data', (d) => logs.push(d.toString()))
  procs.push({ name, child, logs })
}

function stopAll() {
  for (const p of procs) {
    try { process.kill(-p.child.pid, 'SIGTERM') } catch {}
  }
}

function dumpLogs() {
  for (const p of procs) {
    const s = p.logs.join('')
    if (s.trim()) console.error(`── логи ${p.name} ──\n${s.slice(-4000)}`)
  }
}

function writeStatus(status) {
  fs.writeFileSync(STATUS_FILE, JSON.stringify(status))
}

export default async function setup() {
  // По умолчанию — недоступно; при успехе перепишем.
  writeStatus({ ready: false, reason: '' })

  if (!dockerOk()) {
    console.warn('[integration] SKIP: docker недоступен')
    writeStatus({ ready: false, reason: 'docker недоступен' })
    return () => {}
  }

  const repoRoot = findRepoRoot()

  try {
    await ensurePostgres()
    await ensureRedis()
  } catch (e) {
    console.warn('[integration] SKIP: инфраструктура не поднялась:', e.message)
    writeStatus({ ready: false, reason: 'инфраструктура: ' + e.message })
    return () => {}
  }

  try {
    recreateDB()
    await flushRedis()
    await runMigrations(repoRoot)
  } catch (e) {
    console.error('[integration] БД/миграции:', e.message)
    writeStatus({ ready: false, reason: 'миграции: ' + e.message })
    return () => {}
  }

  const baseEnv = { DATABASE_URL: PG.dbURL, REDIS_URL: REDIS.url, PASETO_PUBLIC_KEY: PASETO.publicKey }

  // mailsvc — gRPC-транспорт для authsvc (письма best-effort; SMTP на 1025,
  // если mailpit нет — Send падает быстро connection refused, регистрация всё
  // равно проходит: код подтверждения пишется в БД до отправки).
  startSvc('mailsvc', repoRoot, 'back-go/mail', './cmd/mailsvc', {
    SMTP_HOST: 'localhost', SMTP_PORT: '1025', SMTP_TLS: 'none',
    SMTP_FROM: 'noreply@apitest.local',
    HTTP_ADDR: `:${HTTP.mail}`, GRPC_ADDR: `:${GRPC.mail}`,
  })
  startSvc('authsvc', repoRoot, 'back-go/auth', './cmd/authsvc', {
    ...baseEnv,
    PASETO_PRIVATE_KEY: PASETO.privateKey, PASETO_REFRESH_KEY: PASETO.refreshKey,
    UPLOAD_FOLDER: fs.mkdtempSync(path.join(process.env.TMPDIR || '/tmp', 'gw2-front-uploads-')),
    MAIL_GRPC_ADDR: `localhost:${GRPC.mail}`,
    APP_PUBLIC_BASE_URL: 'http://localhost:5173',
    HTTP_ADDR: `:${HTTP.auth}`,
  })
  startSvc('diarysvc', repoRoot, 'back-go/diary', './cmd/diarysvc', {
    ...baseEnv, HTTP_ADDR: `:${HTTP.diary}`,
  })
  startSvc('tasksvc', repoRoot, 'back-go/tasks', './cmd/tasksvc', {
    ...baseEnv,
    GROOVE_GRPC_ADDR: `localhost:${GRPC.groove}`,
    AI_GRPC_ADDR: 'localhost:19193', // aisvc не поднят — поиск fail-open в LIKE
    YOUGILE_ENC_KEY: 'CT5VF1jg6uFFbj4W_6RW3z3416bPlfbxdMYelrEOIXc=',
    HTTP_ADDR: `:${HTTP.tasks}`,
  })
  startSvc('registrysvc', repoRoot, 'back-go/registry', './cmd/registrysvc', {
    ...baseEnv,
    UPLOAD_FOLDER: fs.mkdtempSync(path.join(process.env.TMPDIR || '/tmp', 'gw2-front-reg-')),
    HTTP_ADDR: `:${HTTP.registry}`,
  })
  startSvc('calendarsvc', repoRoot, 'back-go/calendar', './cmd/calendarsvc', {
    ...baseEnv,
    UPLOAD_FOLDER: fs.mkdtempSync(path.join(process.env.TMPDIR || '/tmp', 'gw2-front-cal-')),
    HTTP_ADDR: `:${HTTP.calendar}`,
  })
  startSvc('msgsvc', repoRoot, 'back-go/messenger', './cmd/msgsvc', {
    ...baseEnv,
    UPLOAD_FOLDER: fs.mkdtempSync(path.join(process.env.TMPDIR || '/tmp', 'gw2-front-msg-')),
    GROOVE_GRPC_ADDR: `localhost:${GRPC.groove}`,
    GRPC_ADDR: `:${GRPC.messenger}`, HTTP_ADDR: `:${HTTP.messenger}`,
  })
  startSvc('groovesvc', repoRoot, 'back-go/groove', './cmd/groovesvc', {
    ...baseEnv,
    AI_GRPC_ADDR: 'localhost:19193', // не поднят — статичные реплики
    MESSENGER_GRPC_ADDR: `localhost:${GRPC.messenger}`,
    GRPC_ADDR: `:${GRPC.groove}`, HTTP_ADDR: `:${HTTP.groove}`,
  })

  try {
    await Promise.all([
      waitHealthz(HTTP.mail), waitHealthz(HTTP.auth), waitHealthz(HTTP.diary),
      waitHealthz(HTTP.tasks), waitHealthz(HTTP.registry), waitHealthz(HTTP.calendar),
      waitHealthz(HTTP.messenger), waitHealthz(HTTP.groove),
    ])
  } catch (e) {
    console.error('[integration] сервис не поднялся:', e.message)
    dumpLogs()
    stopAll()
    writeStatus({ ready: false, reason: e.message })
    return () => {}
  }

  writeStatus({ ready: true, dbURL: PG.dbURL, pgContainer: PG.container })
  console.log('[integration] бэкенд готов (auth/diary/tasks/registry/calendar/messenger/groove)')

  return async () => {
    stopAll()
    await sleep(500)
    for (const p of procs) { try { process.kill(-p.child.pid, 'SIGKILL') } catch {} }
    // Статус-файл НЕ удаляем: воркеры vitest могут импортировать harness после
    // teardown; файл перезаписывается на следующем прогоне (ready:false в начале).
  }
}
