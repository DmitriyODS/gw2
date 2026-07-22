import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import * as api from '@/api/registries.js'
import { useAuthStore } from '@/stores/auth.js'

export const useRegistriesStore = defineStore('registries', () => {
  const registries = ref([])          // [{id, name, fields:[...]}]
  const loadingList = ref(false)
  const selectedId = ref(null)

  const records = ref([])
  const total = ref(0)
  const loadingRecords = ref(false)

  const filters = reactive({
    search: '',
    sort: 'created_at', // 'created_at' | '<field_id>'
    order: 'desc',
    page: 1,
    per_page: 30,
  })

  let fetchSeq = 0
  let fetchCtrl = null

  const selected = computed(() => registries.value.find((r) => r.id === selectedId.value) || null)

  function myCompanyId() {
    return useAuthStore().companyId ?? null
  }
  // События приходят в комнату all с company_id — берём только свою компанию.
  function isMine(companyId) {
    const mine = myCompanyId()
    return companyId == null || mine == null || companyId === mine
  }

  // ── Реестры ──
  // Гарантируем, что у каждого реестра fields — массив (бэкенд может не прислать
  // ключ для реестра без полей), чтобы все потребители работали без проверок.
  function normalizeReg(r) {
    return { ...r, fields: Array.isArray(r?.fields) ? r.fields : [] }
  }

  async function fetchRegistries() {
    loadingList.value = true
    try {
      const data = await api.getRegistries()
      registries.value = (data.registries ?? data ?? []).map(normalizeReg)
      if (selectedId.value && !registries.value.some((r) => r.id === selectedId.value)) {
        selectedId.value = null
      }
    } finally {
      loadingList.value = false
    }
  }

  function select(id) {
    if (selectedId.value === id) return
    selectedId.value = id
    filters.search = ''
    filters.sort = 'created_at'
    filters.order = 'desc'
    filters.page = 1
    records.value = []
    total.value = 0
    if (id != null) fetchRecords()
  }

  // ── Записи ──
  async function fetchRecords({ silent = false } = {}) {
    if (selectedId.value == null) return
    const seq = ++fetchSeq
    fetchCtrl?.abort()
    fetchCtrl = new AbortController()
    if (!silent) loadingRecords.value = true
    try {
      const data = await api.getRecords(selectedId.value, { ...filters }, { signal: fetchCtrl.signal })
      if (seq !== fetchSeq) return
      records.value = data.items ?? []
      total.value = data.total ?? records.value.length
    } catch (e) {
      if (e?.name !== 'AbortError') throw e
    } finally {
      if (seq === fetchSeq) loadingRecords.value = false
    }
  }

  function setSort(fieldKey) {
    if (filters.sort === fieldKey) {
      filters.order = filters.order === 'asc' ? 'desc' : 'asc'
    } else {
      filters.sort = fieldKey
      filters.order = 'asc'
    }
    filters.page = 1
    fetchRecords()
  }

  function setSearch(value) {
    filters.search = value
    filters.page = 1
    fetchRecords()
  }

  function setPage(page) {
    filters.page = page
    fetchRecords()
  }

  async function createRecord(data) {
    await api.createRecord(selectedId.value, data)
    await fetchRecords({ silent: true })
  }

  async function updateRecord(recordId, data) {
    const rec = await api.updateRecord(selectedId.value, recordId, data)
    const i = records.value.findIndex((r) => r.id === recordId)
    if (i !== -1) records.value[i] = rec
    return rec
  }

  async function deleteRecord(recordId) {
    await api.deleteRecord(selectedId.value, recordId)
    await fetchRecords({ silent: true })
  }

  async function bulkDelete(ids) {
    await api.bulkDeleteRecords(selectedId.value, ids)
    await fetchRecords({ silent: true })
  }

  // ── Сокет-события ──
  function applyRegistrySocket(kind, payload) {
    if (!isMine(payload?.company_id)) return
    if (kind === 'deleted') {
      registries.value = registries.value.filter((r) => r.id !== payload.id)
      if (selectedId.value === payload.id) select(null)
      return
    }
    const i = registries.value.findIndex((r) => r.id === payload.id)
    const reg = normalizeReg({ id: payload.id, name: payload.name, position: payload.position, fields: payload.fields })
    if (i === -1) registries.value.push(reg)
    else registries.value[i] = { ...registries.value[i], ...reg }
    // Структура полей выбранного реестра изменилась — перечитываем записи.
    if (kind === 'updated' && selectedId.value === payload.id) fetchRecords({ silent: true })
  }

  function applyRecordSocket(kind, payload) {
    if (!isMine(payload?.company_id)) return
    if (payload?.registry_id !== selectedId.value) return
    // Чужие мутации проще отразить перечиткой текущей страницы (учтёт
    // сортировку/поиск/пагинацию без локального пересчёта).
    fetchRecords({ silent: true })
  }

  // Смена активной компании: реестры company-scoped, поэтому список, выбор и
  // записи прежней компании сбрасываем и грузим заново под новую.
  async function reloadForCompany() {
    selectedId.value = null
    records.value = []
    total.value = 0
    registries.value = []
    filters.search = ''
    filters.sort = 'created_at'
    filters.order = 'desc'
    filters.page = 1
    await fetchRegistries()
  }

  return {
    registries, loadingList, selectedId, selected,
    records, total, loadingRecords, filters,
    fetchRegistries, select, reloadForCompany,
    fetchRecords, setSort, setSearch, setPage,
    createRecord, updateRecord, deleteRecord, bulkDelete,
    applyRegistrySocket, applyRecordSocket,
  }
})
