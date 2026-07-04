import { describe, it, expect, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { usePermission, ROLES } from './usePermission.js'
import { useAuthStore } from '@/stores/auth.js'

// applySession — публичный способ выставить клеймы сессии (claims приватны).
function session(over = {}) {
  return { access_token: 't', role_level: 0, is_super_admin: false, companies: [], ...over }
}

describe('usePermission', () => {
  let auth
  beforeEach(() => {
    setActivePinia(createPinia())
    auth = useAuthStore()
  })

  it('ROLES: сотрудник<менеджер<админ', () => {
    expect(ROLES.EMPLOYEE).toBe(1)
    expect(ROLES.MANAGER).toBe(2)
    expect(ROLES.ADMIN).toBe(3)
  })

  it('нет активной компании (role_level 0) — не менеджер и не админ', () => {
    auth.applySession(session({ role_level: 0 }))
    const p = usePermission()
    expect(p.myLevel()).toBe(0)
    expect(p.isManager()).toBe(false)
    expect(p.isAdmin()).toBe(false)
  })

  it('сотрудник (1): не менеджер, не админ', () => {
    auth.applySession(session({ role_level: 1 }))
    const p = usePermission()
    expect(p.isAtLeast(ROLES.EMPLOYEE)).toBe(true)
    expect(p.isManager()).toBe(false)
    expect(p.isAdmin()).toBe(false)
  })

  it('менеджер (2): менеджер, но не админ', () => {
    auth.applySession(session({ role_level: 2 }))
    const p = usePermission()
    expect(p.isManager()).toBe(true)
    expect(p.isAdmin()).toBe(false)
  })

  it('администратор (3): менеджер и админ', () => {
    auth.applySession(session({ role_level: 3 }))
    const p = usePermission()
    expect(p.isManager()).toBe(true)
    expect(p.isAdmin()).toBe(true)
  })

  it('супер-админ НЕ проходит ролевые гейты компании (role_level 0)', () => {
    auth.applySession(session({ role_level: 0, is_super_admin: true }))
    const p = usePermission()
    expect(p.isSuperAdmin()).toBe(true)
    expect(p.isAdmin()).toBe(false)
    expect(p.isManager()).toBe(false)
    // но раздел «Компании» доступен ему всегда.
    expect(p.canManageCompanies()).toBe(true)
  })

  it('canManageCompanies: обычный — только если админ хотя бы одной компании', () => {
    auth.applySession(session({ role_level: 1, companies: [{ id: 1, role_level: 1 }, { id: 2, role_level: 2 }] }))
    expect(usePermission().canManageCompanies()).toBe(false)

    auth.applySession(session({ role_level: 1, companies: [{ id: 1, role_level: 1 }, { id: 2, role_level: 3 }] }))
    expect(usePermission().canManageCompanies()).toBe(true)
  })
})
