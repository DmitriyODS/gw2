import { useAuthStore } from '@/stores/auth.js'

// Роли в компании. Системной роли больше нет — платформенный «супер-админ»
// задаётся отдельным флагом is_super_admin, а не уровнем роли.
export const ROLES = {
  EMPLOYEE: 1,
  MANAGER: 2,
  ADMIN: 3,
}

export const ROLE_NAMES = {
  1: 'Сотрудник',
  2: 'Менеджер',
  3: 'Администратор',
}

export function usePermission() {
  const auth = useAuthStore()

  // Уровень роли в АКТИВНОЙ компании — из клеймов сессии (claims.role_level).
  // 0 означает «нет активной компании» (это нормальное состояние: пользователь
  // может быть авторизован, но не состоять ни в одной компании).
  function myLevel() {
    return auth.roleLevel
  }

  function isAtLeast(level) {
    return myLevel() >= level
  }

  function isAdmin() {
    return isAtLeast(ROLES.ADMIN)
  }

  function isManager() {
    return isAtLeast(ROLES.MANAGER)
  }

  // Платформенный супер-админ — отдельный флаг, НЕ роль компании.
  function isSuperAdmin() {
    return auth.isSuperAdmin
  }

  // Доступен ли раздел «Компании» (управление): супер-админ (все компании) или
  // администратор хотя бы одной компании (свои/созданные).
  function canManageCompanies() {
    return auth.isSuperAdmin || (auth.companies || []).some((c) => c.role_level >= ROLES.ADMIN)
  }

  // Есть ли активная компания в сессии — гейт компанийных разделов навигации.
  // У супер-админа активной компании не бывает: компанийный контент он не видит.
  function hasActiveCompany() {
    return !auth.isSuperAdmin && auth.roleLevel > 0
  }

  return { isAtLeast, myLevel, isSuperAdmin, isAdmin, isManager, canManageCompanies, hasActiveCompany, ROLES }
}
