import { defineStore } from 'pinia'
import { computed, reactive, ref } from 'vue'
import * as api from '@/api/drive.js'
import { logActivity } from '@/utils/activityLog.js'

/* «Диск» — личные файлы с папками, корзиной и шарингом.

   Состояние держит ОДИН обзор: текущая папка, её содержимое и путь до корня.
   Сокет-события применяются идемпотентно (upsert по id), как в остальных
   разделах: собственное действие обновляет список сразу, а эхо не должно
   плодить дубли. */
export const useDriveStore = defineStore('drive', () => {
  const folder = ref(null) // null — корень диска
  const path = ref([])
  const folders = ref([])
  const files = ref([])
  const loading = ref(false)
  // Идущие загрузки: { id, name, size, progress, cancel }. Каждая строка —
  // reactive-объект: полосу двигает мутация поля, а не замена всего массива.
  const uploads = ref([])

  // Что показываем: обычный обзор папки либо сквозная выборка.
  const view = ref('files') // files | recent | starred | trash | shared
  const search = ref('')

  const isTrash = computed(() => view.value === 'trash')
  const isEmpty = computed(() => !folders.value.length && !files.value.length)

  function apply(data) {
    folder.value = data.folder || null
    path.value = data.path || []
    folders.value = data.folders || []
    files.value = data.files || []
  }

  /* folderId: id папки, null — КОРЕНЬ диска, не передан — «оставить текущую».
     Различать корень и «не передали» обязательно: иначе переход вверх по
     хлебным крошкам возвращал бы в ту же папку. */
  async function load({ folderId, keepView = true } = {}) {
    loading.value = true
    try {
      if (view.value === 'shared') {
        apply(await api.sharedWithMe())
        return
      }
      const target = folderId === undefined ? (folder.value?.id ?? null) : folderId
      const params = { folder_id: target }
      if (search.value.trim()) params.search = search.value.trim()
      else if (keepView && view.value !== 'files') params.view = view.value
      apply(await api.browse(params))
    } finally {
      loading.value = false
    }
  }

  async function openFolder(id) {
    view.value = 'files'
    search.value = ''
    await load({ folderId: id ?? null })
  }

  async function setView(next) {
    view.value = next
    search.value = ''
    // Сквозные выборки идут от корня: папка в них не участвует.
    await load({ folderId: next === 'files' ? folder.value?.id ?? null : null })
  }

  async function runSearch(query) {
    search.value = query
    await load({ folderId: null })
  }

  // ── Папки ──
  async function createFolder(name) {
    const created = await api.createFolder(name, folder.value?.id ?? null)
    upsertFolder(created)
    return created
  }

  async function renameFolder(id, name, color = '') {
    upsertFolder(await api.updateFolder(id, { name, color }))
  }

  async function moveFolder(id, parentId) {
    await api.moveFolder(id, parentId)
    folders.value = folders.value.filter((f) => f.id !== id)
  }

  async function trashFolder(id) {
    await api.trashFolder(id)
    folders.value = folders.value.filter((f) => f.id !== id)
  }

  async function restoreFolder(id) {
    await api.restoreFolder(id)
    folders.value = folders.value.filter((f) => f.id !== id)
  }

  async function purgeFolder(id) {
    await api.purgeFolder(id)
    folders.value = folders.value.filter((f) => f.id !== id)
  }

  // ── Файлы ──
  /* Загрузка: у каждой свой прогресс, и они идут параллельно — так ведут себя
     файловые менеджеры, а последовательная очередь на больших файлах выглядит
     зависшей. */
  async function upload(fileList) {
    const targetId = folder.value?.id ?? null
    const results = await Promise.allSettled([...fileList].map(async (file) => {
      const controller = new AbortController()
      const entry = reactive({
        id: `${file.name}:${Date.now()}:${Math.random()}`,
        name: file.name,
        size: file.size,
        progress: 0,
        cancel: () => controller.abort(),
      })
      uploads.value.push(entry)
      try {
        const created = await api.uploadFile(file, targetId, {
          onProgress: (p) => { entry.progress = p },
          signal: controller.signal,
        })
        // Файл мог прийти сокет-эхом раньше ответа — upsert по id.
        upsertFile(created)
        logActivity({ section: 'drive', id: created.id, title: created.name, path: '/drive' })
      } finally {
        uploads.value = uploads.value.filter((u) => u.id !== entry.id)
      }
    }))
    // Одна неудача не должна прятать остальные: сообщаем о первой, но
    // остальные файлы к этому моменту уже загрузились.
    const failed = results.find((r) => r.status === 'rejected')
    if (failed) throw failed.reason
  }

  /** Отменить идущую загрузку (кнопка в строке прогресса). */
  function cancelUpload(id) {
    uploads.value.find((u) => u.id === id)?.cancel?.()
  }

  async function renameFile(id, name) {
    upsertFile(await api.renameFile(id, name))
  }

  async function moveFile(id, folderId) {
    await api.moveFile(id, folderId)
    files.value = files.value.filter((f) => f.id !== id)
  }

  async function toggleStar(file) {
    upsertFile(await api.starFile(file.id, !file.starred))
  }

  async function trashFile(id) {
    await api.trashFile(id)
    files.value = files.value.filter((f) => f.id !== id)
  }

  async function restoreFile(id) {
    await api.restoreFile(id)
    files.value = files.value.filter((f) => f.id !== id)
  }

  async function purgeFile(id) {
    await api.purgeFile(id)
    files.value = files.value.filter((f) => f.id !== id)
  }

  async function emptyTrash() {
    const res = await api.emptyTrash()
    files.value = []
    folders.value = []
    return res
  }

  // ── Применение сокет-событий (идемпотентно) ──
  function upsertFile(file) {
    if (!file) return
    const i = files.value.findIndex((f) => f.id === file.id)
    if (i >= 0) files.value = files.value.map((f) => (f.id === file.id ? file : f))
    else if (sameFolder(file.folder_id)) files.value = [...files.value, file]
  }

  function upsertFolder(item) {
    if (!item) return
    const i = folders.value.findIndex((f) => f.id === item.id)
    if (i >= 0) folders.value = folders.value.map((f) => (f.id === item.id ? item : f))
    else if (sameFolder(item.parent_id)) folders.value = [...folders.value, item]
  }

  function removeFile(id) { files.value = files.value.filter((f) => f.id !== id) }
  function removeFolder(id) { folders.value = folders.value.filter((f) => f.id !== id) }

  // Событие про другую папку в текущий список не попадает.
  function sameFolder(id) {
    if (view.value !== 'files' || search.value) return false
    return (id ?? null) === (folder.value?.id ?? null)
  }

  function reset() {
    folder.value = null
    path.value = []
    folders.value = []
    files.value = []
    uploads.value = []
    view.value = 'files'
    search.value = ''
  }

  return {
    folder, path, folders, files, loading, uploads, view, search, isTrash, isEmpty,
    load, openFolder, setView, runSearch,
    createFolder, renameFolder, moveFolder, trashFolder, restoreFolder, purgeFolder,
    upload, cancelUpload, renameFile, moveFile, toggleStar, trashFile, restoreFile, purgeFile, emptyTrash,
    upsertFile, upsertFolder, removeFile, removeFolder, reset,
  }
})
