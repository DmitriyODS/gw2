import { useAuthStore } from '@/stores/auth.js'

export const ROLES = {
  EMPLOYEE: 1,
  MANAGER: 2,
  DIRECTOR: 3,
  ADMIN: 4,
}

export const ROLE_NAMES = {
  1: 'Сотрудник',
  2: 'Менеджер',
  3: 'Руководитель',
  4: 'Администратор',
}

export function usePermission() {
  const auth = useAuthStore()

  // Уровень роли в АКТИВНОЙ компании — из клеймов сессии (claims.role_level),
  // а не из /users/me (там «первичная» роль): для многокомпанийного юзера роль
  // зависит от выбранной компании. Фолбэк на профиль — на время до загрузки me.
  function myLevel() {
    return auth.roleLevel || auth.user?.role?.level || 0
  }

  function isAtLeast(level) {
    return myLevel() >= level
  }

  function isRootAdmin() {
    return auth.isRootAdmin
  }

  return { isAtLeast, myLevel, isRootAdmin, ROLES }
}
