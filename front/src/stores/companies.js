import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import {
  listCompanies as apiList,
  createCompany as apiCreate,
  updateCompany as apiUpdate,
  toggleCompanyActive as apiToggle,
  deleteCompany as apiDelete,
} from '@/api/companies.js'
import { useAuthStore } from '@/stores/auth.js'
import { storageGet, storageRemove, storageSet } from '@/utils/storage.js'

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
    const raw = storageGet(STORAGE_KEY, '')
    return raw ? Number(raw) : null
  }

  const activeCompany = computed(() => {
    if (activeCompanyId.value == null) return null
    return items.value.find(c => c.id === activeCompanyId.value) || null
  })

  const effectiveCompanyId = computed(() => {
    if (auth.companyId != null) return auth.companyId
    return activeCompanyId.value
  })

  function setActive(companyId) {
    activeCompanyId.value = companyId
    if (companyId == null) storageRemove(STORAGE_KEY)
    else storageSet(STORAGE_KEY, String(companyId))
  }

  async function load(force = false) {
    if (loading.value) return
    if (loaded.value && !force) return
    loading.value = true
    try {
      const res = await apiList()
      items.value = res.items || []
      loaded.value = true
      if (activeCompanyId.value != null &&
          !items.value.some(c => c.id === activeCompanyId.value)) {
        setActive(null)
      }
    } finally {
      loading.value = false
    }
  }

  function _replace(updated) {
    const idx = items.value.findIndex(c => c.id === updated.id)
    if (idx >= 0) items.value.splice(idx, 1, updated)
    else items.value.unshift(updated)
  }

  // Локально подмешать изменённые настройки компании в загруженный список —
  // чтобы UI (меню/гард раздела через useCompanySettings) у Администратора
  // системы отреагировал сразу, без перезагрузки списка. No-op, если компании
  // нет в items (обычная роль список компаний не грузит).
  function patchSettings(companyId, patch) {
    const idx = items.value.findIndex(c => c.id === companyId)
    if (idx < 0) return
    items.value[idx] = {
      ...items.value[idx],
      settings: { ...(items.value[idx].settings || {}), ...patch },
    }
  }

  async function create(payload) {
    const c = await apiCreate(payload)
    _replace(c)
    return c
  }

  async function update(id, payload) {
    const c = await apiUpdate(id, payload)
    _replace(c)
    return c
  }

  async function toggleActive(id, isActive) {
    // Оптимистичное обновление: обновляем флаг сразу, при ошибке откатываем.
    const idx = items.value.findIndex(c => c.id === id)
    if (idx < 0) return
    const prev = items.value[idx].is_active
    items.value[idx] = { ...items.value[idx], is_active: isActive }
    try {
      const c = await apiToggle(id, isActive)
      _replace(c)
      return c
    } catch (e) {
      items.value[idx] = { ...items.value[idx], is_active: prev }
      throw e
    }
  }

  async function remove(id) {
    await apiDelete(id)
    const idx = items.value.findIndex(c => c.id === id)
    if (idx >= 0) items.value.splice(idx, 1)
    if (activeCompanyId.value === id) setActive(null)
  }

  function clear() {
    items.value = []
    loaded.value = false
    setActive(null)
  }

  watch(() => auth.token, (t) => { if (!t) clear() })

  return {
    items, loading, loaded, activeCompanyId, activeCompany,
    effectiveCompanyId, setActive, load, clear, patchSettings,
    create, update, toggleActive, remove,
  }
})
