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

  function myLevel() {
    return auth.user?.role?.level ?? 0
  }

  function isAtLeast(level) {
    return myLevel() >= level
  }

  function isRootAdmin() {
    return !!auth.user?.is_root_admin
  }

  return { isAtLeast, myLevel, isRootAdmin, ROLES }
}
