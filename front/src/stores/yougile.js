import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as api from '@/api/yougile.js'

/* Состояние YouGile-интеграции:
   - status — статус подключения текущего пользователя.
   - companySettings — настройки интеграции у компании (для UI визарда).
   - companies/projects/boards/columns — кеши для admin-визарда (короткоживущие).
*/
export const useYougileStore = defineStore('yougile', () => {
  const status = ref({
    connected: false,
    yg_login: null,
    key_fingerprint: null,
    last_validated_at: null,
    yg_company_id: null,
    company_enabled: false,
  })
  const statusLoaded = ref(false)

  const companySettings = ref(null)

  // Кеши списков — заполняются по запросу из визарда. Не персистим, пусть
  // живут только пока открыт диалог.
  const ygCompanies = ref([])
  const ygProjects = ref([])
  const ygBoards = ref([])
  const ygColumns = ref([])

  // Удобные вычисленные флаги для использования в карточке задачи и модалках:
  // фича доступна = (компания включила YouGile && пользователь подключён).
  const isAvailable = computed(
    () => status.value.connected && status.value.company_enabled,
  )

  async function refreshStatus() {
    const data = await api.getYougileStatus()
    status.value = data
    statusLoaded.value = true
    return data
  }

  async function connect({ login, password, yg_company_id = null }) {
    const res = await api.connectYougile({ login, password, yg_company_id })
    // Полный статус догружаем отдельным запросом — он отдаёт ещё
    // company_enabled и валидированную дату.
    await refreshStatus()
    return res
  }

  async function disconnect() {
    await api.disconnectYougile()
    await refreshStatus()
  }

  async function rotate({ password }) {
    const res = await api.rotateYougile({ password })
    await refreshStatus()
    return res
  }

  async function loadCompanySettings() {
    companySettings.value = await api.getCompanyYougileSettings()
    return companySettings.value
  }

  async function updateCompanySettings(payload) {
    companySettings.value = await api.updateCompanyYougileSettings(payload)
    // При смене настроек company_enabled у текущего юзера тоже меняется —
    // подтягиваем статус, чтобы карточки задач сразу подхватили фичу.
    await refreshStatus().catch(() => {})
    return companySettings.value
  }

  async function resetIntegration() {
    companySettings.value = await api.resetCompanyYougileIntegration()
    // Аккаунт инициатора отвязан, флаги компании сброшены — обновляем статус
    // и чистим короткоживущие кеши визарда.
    await refreshStatus().catch(() => {})
    ygCompanies.value = []
    ygProjects.value = []
    ygBoards.value = []
    ygColumns.value = []
    return companySettings.value
  }

  async function lookupCompanies({ login, password }) {
    ygCompanies.value = await api.lookupYougileCompanies({ login, password })
    return ygCompanies.value
  }

  async function loadProjects() {
    ygProjects.value = await api.listYougileProjects()
    return ygProjects.value
  }

  async function loadBoards(projectId) {
    ygBoards.value = await api.listYougileBoards(projectId)
    return ygBoards.value
  }

  async function loadColumns(boardId) {
    ygColumns.value = await api.listYougileColumns(boardId)
    return ygColumns.value
  }

  function reset() {
    status.value = {
      connected: false, yg_login: null, key_fingerprint: null,
      last_validated_at: null, yg_company_id: null, company_enabled: false,
    }
    statusLoaded.value = false
    companySettings.value = null
    ygCompanies.value = []
    ygProjects.value = []
    ygBoards.value = []
    ygColumns.value = []
  }

  return {
    status, statusLoaded, companySettings,
    ygCompanies, ygProjects, ygBoards, ygColumns,
    isAvailable,
    refreshStatus, connect, disconnect, rotate,
    loadCompanySettings, updateCompanySettings, resetIntegration,
    lookupCompanies, loadProjects, loadBoards, loadColumns,
    reset,
  }
})
