import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { listCompanies as apiList } from '@/api/companies.js'
import { useAuthStore } from '@/stores/auth.js'

const STORAGE_KEY = 'gw_active_company_id'

export const useCompaniesStore = defineStore('companies', () => {
  const auth = useAuthStore()
  const items = ref([])
  const loading = ref(false)
  const loaded = ref(false)
  // Активная компания. Для обычных ролей всегда равна auth.companyId;
  // для Администратора системы — выбранная в селекторе или null (нет фильтра).
  const activeCompanyId = ref(_initActive())

  function _initActive() {
    try {
      const raw = localStorage.getItem(STORAGE_KEY)
      return raw ? Number(raw) : null
    } catch { return null }
  }

  const activeCompany = computed(() => {
    if (activeCompanyId.value == null) return null
    return items.value.find(c => c.id === activeCompanyId.value) || null
  })

  const effectiveCompanyId = computed(() => {
    // Сотрудник/Менеджер/Руководитель — всегда своя компания, селектор скрыт.
    if (auth.companyId != null) return auth.companyId
    return activeCompanyId.value
  })

  function setActive(companyId) {
    activeCompanyId.value = companyId
    try {
      if (companyId == null) localStorage.removeItem(STORAGE_KEY)
      else localStorage.setItem(STORAGE_KEY, String(companyId))
    } catch {}
  }

  async function load(force = false) {
    if (loading.value) return
    if (loaded.value && !force) return
    loading.value = true
    try {
      const res = await apiList()
      items.value = res.items || []
      loaded.value = true
      // Если выбранной компании больше нет (удалили) — сбрасываем.
      if (activeCompanyId.value != null &&
          !items.value.some(c => c.id === activeCompanyId.value)) {
        setActive(null)
      }
    } finally {
      loading.value = false
    }
  }

  function clear() {
    items.value = []
    loaded.value = false
    setActive(null)
  }

  // При смене auth.token сбрасываем — после logout не должны храниться чужие данные.
  watch(() => auth.token, (t) => { if (!t) clear() })

  return {
    items, loading, loaded, activeCompanyId, activeCompany,
    effectiveCompanyId, setActive, load, clear,
  }
})
