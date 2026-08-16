import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import * as api from '@/api/forms.js'
import { logActivity } from '@/utils/activityLog.js'

/* Форма принадлежит ЧЕЛОВЕКУ, а не компании: список приходит областями
   (вкладки «Все / Мои / Мне назначены / Совместные»), а что можно делать —
   говорит my_access, посчитанный сервером. Клиент его только показывает:
   решать права на клиенте нельзя. */
export const useFormsStore = defineStore('forms', () => {
  const forms = ref([])            // карточки списка (без структуры)
  const loadingList = ref(false)
  const selectedId = ref(null)
  const selected = ref(null)       // открытая форма со структурой
  const loadingForm = ref(false)
  const scope = ref('all')         // all | mine | assigned | shared

  const responses = ref([])
  const responsesTotal = ref(0)
  const loadingResponses = ref(false)
  const summary = ref(null)
  const progress = ref(null)

  const filters = reactive({
    search: '',
    sort: 'created_at',
    order: 'desc',
    page: 1,
    per_page: 30,
  })

  let responsesSeq = 0

  // Что доступно в открытой форме — по уровню, посчитанному сервером.
  const myAccess = computed(() => selected.value?.my_access || '')
  const canSeeResponses = computed(() => ['view', 'edit', 'owner'].includes(myAccess.value))
  const canEdit = computed(() => ['edit', 'owner'].includes(myAccess.value))
  const isOwner = computed(() => myAccess.value === 'owner')

  // ── Список ──
  async function fetchForms() {
    loadingList.value = true
    try {
      const data = await api.getForms(scope.value)
      forms.value = data.forms ?? []
      if (selectedId.value && !forms.value.some((f) => f.id === selectedId.value)) {
        // Форму могли забрать вместе с доступом — открытую карточку не рвём,
        // она перечитается сама при следующем обращении.
        if (!selected.value) selectedId.value = null
      }
    } finally {
      loadingList.value = false
    }
  }

  function setScope(value) {
    if (scope.value === value) return
    scope.value = value
    fetchForms()
  }

  /* mergeForm — вставить или обновить карточку списка по id. И оптимизм, и
     сокет идут через неё: событие может прийти раньше HTTP-ответа, и слепой
     push дал бы дубль. */
  function mergeForm(form) {
    const i = forms.value.findIndex((f) => f.id === form.id)
    if (i === -1) {
      forms.value.unshift(form)
      return
    }
    forms.value[i] = { ...forms.value[i], ...form }
  }

  // ── Открытая форма ──
  async function select(id) {
    if (selectedId.value === id && selected.value) return
    selectedId.value = id
    selected.value = null
    summary.value = null
    progress.value = null
    responses.value = []
    responsesTotal.value = 0
    resetFilters()
    if (id != null) await loadForm()
  }

  async function loadForm({ silent = false } = {}) {
    if (selectedId.value == null) return null
    if (!silent) loadingForm.value = true
    try {
      const form = await api.getForm(selectedId.value)
      if (form?.id !== selectedId.value) return null
      selected.value = form
      mergeForm(cardOf(form))
      return form
    } finally {
      loadingForm.value = false
    }
  }

  // cardOf — карточка списка из полной формы: структура списку не нужна.
  function cardOf(form) {
    const { sections, ...card } = form
    return card
  }

  function resetFilters() {
    filters.search = ''
    filters.sort = 'created_at'
    filters.order = 'desc'
    filters.page = 1
  }

  async function createForm(title, quiz = false) {
    const form = await api.createForm(title, quiz)
    mergeForm(cardOf(form))
    logActivity({ section: 'forms', id: form.id, title: form.title, path: `/forms?form=${form.id}` })
    return form
  }

  async function updateForm(patch) {
    const form = await api.updateForm(selectedId.value, patch)
    selected.value = { ...selected.value, ...form }
    mergeForm(cardOf(form))
    return form
  }

  async function saveStructure(sections) {
    const form = await api.saveStructure(selectedId.value, sections)
    selected.value = form
    mergeForm(cardOf(form))
    return form
  }

  async function removeForm(id) {
    await api.deleteForm(id)
    forms.value = forms.value.filter((f) => f.id !== id)
    if (selectedId.value === id) {
      selectedId.value = null
      selected.value = null
    }
  }

  async function duplicateForm(id) {
    const form = await api.duplicateForm(id)
    mergeForm(cardOf(form))
    return form
  }

  // ── Ответы, сводка, исполнение ──
  async function fetchResponses({ silent = false } = {}) {
    if (selectedId.value == null) return
    const seq = ++responsesSeq
    if (!silent) loadingResponses.value = true
    try {
      const data = await api.getResponses(selectedId.value, { ...filters })
      if (seq !== responsesSeq) return
      responses.value = data.items ?? []
      responsesTotal.value = data.total ?? responses.value.length
    } finally {
      if (seq === responsesSeq) loadingResponses.value = false
    }
  }

  function setSearch(value) {
    filters.search = value
    filters.page = 1
    fetchResponses()
  }

  function setPage(page) {
    filters.page = page
    fetchResponses()
  }

  function applySort({ sort, order }) {
    filters.sort = sort
    filters.order = order
    filters.page = 1
    fetchResponses()
  }

  async function fetchSummary() {
    if (selectedId.value == null) return
    summary.value = await api.getSummary(selectedId.value)
  }

  async function fetchProgress() {
    if (selectedId.value == null) return
    progress.value = await api.getProgress(selectedId.value)
  }

  async function deleteResponse(responseId) {
    await api.deleteResponse(selectedId.value, responseId)
    await refreshResponses()
  }

  async function clearResponses() {
    await api.deleteResponses(selectedId.value, { all: true })
    await refreshResponses()
  }

  async function publishGrades(responseId = 0) {
    await api.publishGrades(selectedId.value, responseId)
    await fetchResponses({ silent: true })
  }

  // refreshResponses — после правки набора ответов сводка и счётчики врут,
  // поэтому обновляются вместе со страницей.
  async function refreshResponses() {
    await fetchResponses({ silent: true })
    if (summary.value) await fetchSummary()
    if (progress.value) await fetchProgress()
    await loadForm({ silent: true })
  }

  // ── Сокет-события ──
  // События приходят поимённо тем, кому форма доступна, поэтому фильтровать по
  // компании не нужно — доставку решает сервер.
  function applyFormSocket(kind, payload) {
    if (!payload?.id && !payload?.form_id) return
    const id = payload.id ?? payload.form_id
    if (kind === 'deleted') {
      forms.value = forms.value.filter((f) => f.id !== id)
      if (selectedId.value === id) {
        selectedId.value = null
        selected.value = null
      }
      return
    }
    // Состав доступа изменился либо форма создана: список области мог
    // поменяться целиком.
    if (kind === 'shared' || kind === 'created') {
      fetchForms()
      if (selectedId.value === id) loadForm({ silent: true })
      return
    }
    /* Уровень доступа в событие не входит (у каждого получателя он свой) —
       сохраняем свой, иначе кнопки правки исчезли бы после чужого сохранения. */
    const known = forms.value.find((f) => f.id === id)
    if (known) mergeForm({ ...payload, id, my_access: known.my_access })
    if (selectedId.value === id) loadForm({ silent: true })
  }

  function applyResponseSocket(payload) {
    if (payload?.form_id !== selectedId.value) return
    refreshResponses()
  }

  return {
    forms, loadingList, selectedId, selected, loadingForm, scope,
    responses, responsesTotal, loadingResponses, summary, progress, filters,
    myAccess, canSeeResponses, canEdit, isOwner,
    fetchForms, setScope, select, loadForm,
    createForm, updateForm, saveStructure, removeForm, duplicateForm,
    fetchResponses, setSearch, setPage, applySort, fetchSummary, fetchProgress,
    deleteResponse, clearResponses, publishGrades,
    applyFormSocket, applyResponseSocket,
  }
})
