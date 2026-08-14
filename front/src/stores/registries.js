import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import * as api from '@/api/registries.js'
import { useAuthStore } from '@/stores/auth.js'
import { logActivity } from '@/utils/activityLog.js'
import { sectionField, textValue } from '@/utils/registryFields.js'

/* Реестр принадлежит ЧЕЛОВЕКУ, а не компании: список приходит областями
   (вкладки «Все / Мои / Поделились / Компания»), а что можно делать — говорит
   my_access, посчитанный сервером. Клиент его только показывает: решать права
   на клиенте нельзя. */
export const useRegistriesStore = defineStore('registries', () => {
  const registries = ref([])          // [{id, name, my_access, fields:[...]}]
  const loadingList = ref(false)
  const selectedId = ref(null)
  const scope = ref('all')            // all | mine | shared | company

  const records = ref([])
  const total = ref(0)
  const loadingRecords = ref(false)

  const filters = reactive({
    search: '',
    sort: 'created_at', // 'created_at' | '<field_id>'
    order: 'desc',
    section: '',        // выбранная вкладка-подраздел ('' — все записи)
    filters: [],        // фильтры колонок: [{field_id, op, values}]
    page: 1,
    per_page: 30,
  })

  let fetchSeq = 0
  let fetchCtrl = null

  const selected = computed(() => registries.value.find((r) => r.id === selectedId.value) || null)

  // Что доступно в выбранном реестре — по уровню, посчитанному сервером.
  const myAccess = computed(() => selected.value?.my_access || '')
  const canEdit = computed(() => ['edit', 'admin', 'owner'].includes(myAccess.value))
  const canManage = computed(() => ['admin', 'owner'].includes(myAccess.value))
  const isOwner = computed(() => myAccess.value === 'owner')

  /* ── Реестры ──
     У реестра в сторе fields — ВСЕГДА массив: так потребители обходятся без
     проверок. Но «ключа fields нет» и «полей нет» — разные вещи: снимок без
     полей приходит от ручек, которые их не читают, и принимать его за пустой
     набор нельзя — из таблицы разом исчезали все колонки. Поэтому пустой
     массив подставляется только НОВОЙ записи списка (см. mergeRegistry). */
  function normalizeReg(r) {
    return Array.isArray(r?.fields) ? { ...r } : { ...r, fields: undefined }
  }

  async function fetchRegistries() {
    loadingList.value = true
    try {
      const data = await api.getRegistries(scope.value)
      // Список полями приходит всегда — здесь отсутствие ключа и правда значит
      // «полей нет».
      registries.value = (data.registries ?? data ?? [])
        .map((r) => ({ ...normalizeReg(r), fields: r?.fields ?? [] }))
      if (selectedId.value && !registries.value.some((r) => r.id === selectedId.value)) {
        selectedId.value = null
      }
    } finally {
      loadingList.value = false
    }
  }

  function setScope(value) {
    if (scope.value === value) return
    scope.value = value
    fetchRegistries()
  }

  function resetFilters() {
    filters.search = ''
    filters.sort = 'created_at'
    filters.order = 'desc'
    filters.section = ''
    filters.filters = []
    filters.page = 1
  }

  function select(id) {
    if (selectedId.value === id) return
    selectedId.value = id
    resetFilters()
    records.value = []
    total.value = 0
    if (id != null) fetchRecords()
  }

  async function createRegistry(name, accounting = false) {
    const reg = normalizeReg(await api.createRegistry(name, accounting))
    registries.value.push(reg)
    logActivity({ section: 'registries', id: reg.id, title: reg.name, path: `/registries?registry=${reg.id}` })
    return reg
  }

  async function renameRegistry(id, name) {
    const reg = normalizeReg(await api.updateRegistry(id, { name }))
    mergeRegistry(reg)
    return reg
  }

  async function updateRegistry(id, patch) {
    const reg = normalizeReg(await api.updateRegistry(id, patch))
    mergeRegistry(reg)
    if (selectedId.value === id) await fetchRecords({ silent: true })
    return reg
  }

  async function replaceFields(id, fields) {
    const reg = normalizeReg(await api.replaceFields(id, fields))
    mergeRegistry(reg)
    if (selectedId.value === id) await fetchRecords({ silent: true })
    return reg
  }

  async function removeRegistry(id) {
    await api.deleteRegistry(id)
    registries.value = registries.value.filter((r) => r.id !== id)
    if (selectedId.value === id) select(null)
  }

  /* mergeRegistry — вставить или обновить реестр в списке по id. И оптимизм, и
     сокет идут через неё: событие может прийти раньше HTTP-ответа, и слепой
     push дал бы дубль.

     Снимок без полей их НЕ затирает: сохраняем те, что уже знаем. */
  function mergeRegistry(reg) {
    const i = registries.value.findIndex((r) => r.id === reg.id)
    if (i === -1) {
      registries.value.push({ ...reg, fields: reg.fields ?? [] })
      return
    }
    const prev = registries.value[i]
    registries.value[i] = { ...prev, ...reg, fields: reg.fields ?? prev.fields ?? [] }
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
      if (e?.name !== 'AbortError' && e?.error !== 'ABORTED') throw e
    } finally {
      if (seq === fetchSeq) loadingRecords.value = false
    }
  }

  // Порядок сортировки решает вид записей — здесь только применяем готовую пару
  // и возвращаемся на первую страницу.
  function applySort({ sort, order }) {
    filters.sort = sort
    filters.order = order
    filters.page = 1
    fetchRecords()
  }

  function setSearch(value) {
    filters.search = value
    filters.page = 1
    fetchRecords()
  }

  // Вкладка-подраздел: повторный клик по активной возвращает к «Все».
  function setSection(value) {
    filters.section = filters.section === value ? '' : (value || '')
    filters.page = 1
    fetchRecords()
  }

  // Фильтры колонок: пустой набор значений снимает условие с колонки.
  function setColumnFilter(fieldId, filter) {
    const rest = filters.filters.filter((f) => f.field_id !== fieldId)
    filters.filters = filter ? [...rest, { field_id: fieldId, ...filter }] : rest
    filters.page = 1
    fetchRecords()
  }

  function clearColumnFilters() {
    filters.filters = []
    filters.page = 1
    fetchRecords()
  }

  function setPage(page) {
    filters.page = page
    fetchRecords()
  }

  async function createRecord(data) {
    const rec = await api.createRecord(selectedId.value, data)
    await fetchRecords({ silent: true })
    logActivity({
      section: 'registries', id: rec?.id, title: recordTitle(rec),
      path: `/registries?registry=${selectedId.value}`,
    })
    return rec
  }

  // Заголовок записи для ленты действий: первое непустое текстовое значение
  // (структура карточки у каждого реестра своя).
  function recordTitle(rec) {
    const fields = selected.value?.fields || []
    for (const f of fields) {
      const v = textValue(f, rec?.data?.[String(f.id)])
      if (v) return v
    }
    return selected.value?.name || 'Запись'
  }

  async function updateRecord(recordId, data) {
    const rec = await api.updateRecord(selectedId.value, recordId, data)
    mergeRecord(rec)
    return rec
  }

  function mergeRecord(rec) {
    const i = records.value.findIndex((r) => r.id === rec.id)
    if (i !== -1) records.value[i] = { ...records.value[i], ...rec }
  }

  async function deleteRecord(recordId) {
    await api.deleteRecord(selectedId.value, recordId)
    await fetchRecords({ silent: true })
  }

  async function bulkDelete(selection) {
    await api.bulkDeleteRecords(selectedId.value, selection, exportFilter())
    await fetchRecords({ silent: true })
  }

  // exportFilter — фильтр экрана в виде, который понимают выгрузка, печать и
  // массовое удаление: файл не должен расходиться с тем, что видно.
  function exportFilter() {
    return { search: filters.search, section: filters.section, filters: filters.filters }
  }

  // ── Учётный реестр ──
  async function issueRecord(recordId, body) {
    const issue = await api.issueRecord(selectedId.value, recordId, body)
    applyIssue(recordId, issue)
    return issue
  }

  async function extendIssue(recordId, body) {
    const issue = await api.extendIssue(selectedId.value, recordId, body)
    applyIssue(recordId, issue)
    return issue
  }

  async function returnIssue(recordId, comment) {
    await api.returnIssue(selectedId.value, recordId, comment)
    applyIssue(recordId, null)
  }

  // Открытая выдача живёт прямо в записи — плашка состояния перерисовывается
  // без перечитки всей страницы.
  function applyIssue(recordId, issue) {
    const i = records.value.findIndex((r) => r.id === recordId)
    if (i !== -1) records.value[i] = { ...records.value[i], issue: issue || undefined }
  }

  // ── Сокет-события ──
  // Событие приходит поимённо тем, кому реестр доступен, поэтому фильтровать по
  // компании больше не нужно — доставку решает сервер.
  function applyRegistrySocket(kind, payload) {
    if (kind === 'deleted') {
      registries.value = registries.value.filter((r) => r.id !== payload.id)
      if (selectedId.value === payload.id) select(null)
      return
    }
    // Изменился состав доступа — список области мог поменяться целиком.
    if (kind === 'shared') {
      fetchRegistries()
      return
    }
    const reg = normalizeReg({
      id: payload.id, owner_id: payload.owner_id, company_id: payload.company_id,
      name: payload.name, position: payload.position,
      section_field_id: payload.section_field_id, accounting: payload.accounting,
      fields: payload.fields,
    })
    // Уровень доступа в событие не входит (у каждого получателя он свой) —
    // сохраняем свой, иначе кнопки правки исчезли бы после чужого сохранения.
    const known = registries.value.some((r) => r.id === reg.id)
    if (!known && kind === 'created' && scope.value !== 'all' && scope.value !== 'mine') {
      return // в чужую область новый реестр не подкладываем
    }
    mergeRegistry(reg)
    // Структура полей выбранного реестра изменилась — перечитываем записи.
    if (kind === 'updated' && selectedId.value === payload.id) fetchRecords({ silent: true })
  }

  function applyRecordSocket(kind, payload) {
    if (payload?.registry_id !== selectedId.value) return
    if (kind === 'issue') {
      applyIssue(payload.record_id, payload.issue)
      return
    }
    // Чужие мутации проще отразить перечиткой текущей страницы (учтёт
    // сортировку/поиск/пагинацию без локального пересчёта).
    fetchRecords({ silent: true })
  }

  /* Смена активной компании: сам список реестров от неё больше не зависит
     (они личные), но компанийные шары — да: в новой компании доступны другие
     реестры. Поэтому список перечитываем, а выбор сохраняем, если он уцелел. */
  async function reloadForCompany() {
    await fetchRegistries()
    if (selectedId.value != null && !registries.value.some((r) => r.id === selectedId.value)) {
      select(null)
    }
  }

  // Активная компания нужна только подсказке «поделиться с компанией».
  function myCompanyId() {
    return useAuthStore().companyId ?? null
  }

  // Вкладки-подразделы выбранного реестра ('' — вкладка «Все»).
  const sections = computed(() => {
    const field = sectionField(selected.value)
    return field ? (field.config?.options || []).filter(Boolean) : []
  })

  return {
    registries, loadingList, selectedId, selected, scope,
    records, total, loadingRecords, filters, sections,
    myAccess, canEdit, canManage, isOwner, myCompanyId,
    fetchRegistries, setScope, select, reloadForCompany,
    createRegistry, renameRegistry, updateRegistry, replaceFields, removeRegistry,
    fetchRecords, applySort, setSearch, setSection, setPage,
    setColumnFilter, clearColumnFilters, exportFilter,
    createRecord, updateRecord, deleteRecord, bulkDelete,
    issueRecord, extendIssue, returnIssue,
    applyRegistrySocket, applyRecordSocket,
  }
})
