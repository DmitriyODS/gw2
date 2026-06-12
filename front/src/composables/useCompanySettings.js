import { computed } from 'vue'
import { useAuthStore } from '@/stores/auth.js'
import { useCompaniesStore } from '@/stores/companies.js'

// Возвращает реактивные настройки активной компании. У обычных ролей берётся
// из клеймов сессии (auth.companySettings). У Администратора системы — из selected
// company в companies-store; если ничего не выбрано, отдаём все фичи включёнными
// (что эквивалентно «глобальный системный режим»).
export function useCompanySettings() {
  const auth = useAuthStore()
  const companies = useCompaniesStore()

  const settings = computed(() => {
    if (auth.companyId != null) return auth.companySettings || {}
    return companies.activeCompany?.settings || {}
  })

  const usesYougile = computed(() => settings.value.uses_yougile !== false)
  const usesStages = computed(() => settings.value.uses_stages !== false)
  const usesCalls = computed(() => settings.value.uses_calls !== false)

  return { settings, usesYougile, usesStages, usesCalls }
}
