// Дымовые вызовы главного GET каждого фронтового api-модуля (front/src/api/*.js)
// против живого бэкенда — ловим дрейф пути/метода/парсинга ответа. Модули,
// требующие неподнятых сервисов (calls/ai/push/gateway-presence/changelog),
// помечены skip с причиной.
import { it, expect, beforeAll } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { newCompanyAdmin } from '../setup/factory.js'

import { suggestLogin } from '@/api/auth.js'
import { getMe, getDirectory } from '@/api/users.js'
import { getRoles } from '@/api/roles.js'
import { listMyCompanies } from '@/api/companies.js'
import { getDepartments } from '@/api/departments.js'
import { getUnitTypes } from '@/api/unitTypes.js'
import { getStages } from '@/api/stages.js'
import { getTasks } from '@/api/tasks.js'
import { getActiveUnit } from '@/api/units.js'
import { getStatsProfile } from '@/api/stats.js'
import { getDiaries } from '@/api/diaries.js'
import { getRegistries } from '@/api/registries.js'
import { getCalendars } from '@/api/calendars.js'
import { getMyPet } from '@/api/pets.js'
import { listConversations } from '@/api/messenger.js'
import { getYougileStatus } from '@/api/yougile.js'
import { exportBackup } from '@/api/backup.js'

describeIntegration('smoke: главный GET каждого api-модуля', () => {
  let admin
  beforeAll(async () => { admin = await newCompanyAdmin('smoke') })

  it('auth.suggestLogin → {login}', async () => {
    admin.session.use()
    const r = await suggestLogin('Иванов Иван Иванович')
    expect(typeof r.login).toBe('string')
    expect(r.login.length).toBeGreaterThan(0)
  })

  it('users.getMe / getDirectory', async () => {
    admin.session.use()
    const me = await getMe()
    expect(me.id).toBe(admin.auth.userId)
    const dir = await getDirectory()
    expect(Array.isArray(dir.items ?? dir)).toBe(true)
  })

  it('roles.getRoles → 3 фиксированные роли', async () => {
    admin.session.use()
    const roles = await getRoles()
    const arr = roles.items ?? roles
    expect(Array.isArray(arr)).toBe(true)
    expect(arr.length).toBeGreaterThanOrEqual(3)
  })

  it('companies.listMyCompanies содержит мою', async () => {
    admin.session.use()
    const list = await listMyCompanies()
    const arr = list.companies ?? list.items ?? list
    expect(arr.some((c) => c.id === admin.companyId)).toBe(true)
  })

  it('departments.getDepartments → массив', async () => {
    admin.session.use()
    const d = await getDepartments()
    expect(Array.isArray(d.items ?? d)).toBe(true)
  })

  it('unitTypes.getUnitTypes → массив', async () => {
    admin.session.use()
    const d = await getUnitTypes()
    expect(Array.isArray(d.items ?? d)).toBe(true)
  })

  it('stages.getStages → массив', async () => {
    admin.session.use()
    const d = await getStages()
    expect(Array.isArray(d.items ?? d)).toBe(true)
  })

  it('tasks.getTasks → страница', async () => {
    admin.session.use()
    const d = await getTasks({ page: 1, per_page: 10 })
    expect(Array.isArray(d.tasks ?? d.items ?? d)).toBe(true)
  })

  it('units.getActiveUnit → null (нет активного)', async () => {
    admin.session.use()
    const u = await getActiveUnit()
    expect(u == null).toBe(true)
  })

  it('stats.getStatsProfile → объект', async () => {
    admin.session.use()
    const s = await getStatsProfile('2026-01-01', '2026-12-31')
    expect(s && typeof s === 'object').toBe(true)
  })

  it('diaries.getDiaries → {diaries}', async () => {
    admin.session.use()
    const d = await getDiaries('mine')
    expect(Array.isArray(d.diaries)).toBe(true)
  })

  it('registries.getRegistries → список', async () => {
    admin.session.use()
    const r = await getRegistries()
    expect(Array.isArray(r.registries ?? r.items ?? r)).toBe(true)
  })

  it('calendars.getCalendars → список', async () => {
    admin.session.use()
    const c = await getCalendars()
    expect(Array.isArray(c.calendars ?? c.items ?? c)).toBe(true)
  })

  it('pets.getMyPet → питомец', async () => {
    admin.session.use()
    const p = await getMyPet()
    expect(p.user_id).toBe(admin.auth.userId)
  })

  it('messenger.listConversations → список', async () => {
    admin.session.use()
    const c = await listConversations()
    expect(Array.isArray(c.items ?? c.conversations ?? c)).toBe(true)
  })

  it('yougile.getYougileStatus → {connected:false} без интеграции', async () => {
    admin.session.use()
    const s = await getYougileStatus()
    expect(s && typeof s === 'object').toBe(true)
    expect(!!s.connected).toBe(false)
  })

  it('backup.exportBackup для не-супер-админа → 403 (blob Response)', async () => {
    admin.session.use()
    const resp = await exportBackup()
    // blob:true — client.js возвращает сам Response; обычному пользователю 403.
    expect(resp.status).toBe(403)
  })

  // ── Модули, требующие неподнятых сервисов ──
  it.skip('ai.getAiSettings — aisvc не поднят (AI обязан быть fail-open)', () => {})
  it.skip('ai.getTvFact — aisvc не поднят', () => {})
  it.skip('calls.getActiveCall — callsvc не поднят', () => {})
  it.skip('messenger.getPresence — presence живёт в gatewaysvc (не поднят)', () => {})
  it.skip('changelog.get — /api/changelog отдаёт статика nginx, не сервис', () => {})
  // client.js — инфраструктурный модуль (fetch/refresh), покрыт всеми сценариями.
})
