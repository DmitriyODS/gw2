import { useAuthStore } from '@/stores/auth.js'

export const ROLES = {
  EMPLOYEE: 1,
  MANAGER: 2,
  ADMIN: 3,
  SUPERADMIN: 4,
}

export const ROLE_NAMES = {
  1: 'Сотрудник',
  2: 'Менеджер',
  3: 'Администратор',
  4: 'Суперадминистратор',
}

export function usePermission() {
  const auth = useAuthStore()

  function myLevel() {
    return auth.user?.role?.level ?? 0
  }

  function isAtLeast(level) {
    return myLevel() >= level
  }

  return { isAtLeast, myLevel, ROLES }
}
