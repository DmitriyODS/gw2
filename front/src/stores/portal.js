import { defineStore } from 'pinia'
import { reactive, ref } from 'vue'
import * as api from '@/api/portal.js'
import { getDirectory } from '@/api/users.js'
import { useAuthStore } from '@/stores/auth.js'

// Корпоративный портал (posts/topics/comments/reactions). Посты/комментарии
// несут только author_id/pinned_by (числа, без снапшота ФИО/аватара) —
// резолвим их через уже загруженный каталог сотрудников компании (тот же
// паттерн, что и в TvView.vue: getDirectory() → Map(id, user)).
export const usePortalStore = defineStore('portal', () => {
  const topics = ref([])
  // Лента — два серверных списка: pinnedPosts (все актуально закреплённые,
  // приходят только с первой страницей) и posts (хронология БЕЗ закреплённых,
  // догружается keyset-курсором через fetchMore).
  const posts = ref([])
  const pinnedPosts = ref([])
  const nextCursor = ref(null)
  const loadingMore = ref(false)
  // Пост, открытый по прямой ссылке /portal/:id — живёт в сторе, чтобы
  // реакции/комментарии/сокет-события работали для него так же, как для ленты.
  const highlight = ref(null)
  const loadingTopics = ref(false)
  const loadingPosts = ref(false)

  // Бейдж непрочитанных постов в навигации: счётчик серверный (общий между
  // устройствами), сокет post:new наращивает вживую, заход на портал сбрасывает.
  const unread = ref(0)
  const viewingFeed = ref(false)

  const filters = reactive({ topicId: null, search: '' })

  const authorMap = ref(new Map())
  const commentsByPost = reactive({})
  const loadingComments = reactive({})
  // Посты, чей просмотр уже отмечен в этой сессии — гард от повторных запросов
  // при каждом срабатывании IntersectionObserver карточки.
  const markedViews = new Set()

  function myCompanyId() {
    return useAuthStore().companyId ?? null
  }
  // Сокет-события приходят в комнату all с company_id — берём только свою компанию.
  function isMine(companyId) {
    const mine = myCompanyId()
    return companyId == null || mine == null || companyId === mine
  }

  // ── Непрочитанные (бейдж) ──
  async function fetchUnread() {
    try {
      const data = await api.getUnreadCount()
      unread.value = data.count ?? 0
    } catch { /* бейдж не критичен — при ошибке оставляем как есть */ }
  }

  // Best-effort: локально гасим бейдж сразу, серверную отметку шлём фоном.
  function markSeen() {
    unread.value = 0
    api.markSeen().catch(() => {})
  }

  // ── Каталог сотрудников (резолв автора) ──
  async function loadAuthors() {
    if (authorMap.value.size) return
    try {
      const list = await getDirectory()
      const m = new Map()
      for (const u of list) m.set(u.id, u)
      authorMap.value = m
    } catch { /* каталог — не критично для отображения ленты */ }
  }

  function resolveAuthor(id) {
    const u = authorMap.value.get(id)
    return {
      id,
      fio: u?.fio || 'Сотрудник',
      avatarUrl: u?.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${id}/identicon`,
    }
  }

  // ── Топики ──
  async function fetchTopics() {
    loadingTopics.value = true
    try {
      const data = await api.getTopics()
      topics.value = data.topics ?? []
    } finally {
      loadingTopics.value = false
    }
  }

  async function createTopic(data) {
    const t = await api.createTopic(data)
    if (!topics.value.some((x) => x.id === t.id)) topics.value.push(t)
    return t
  }

  async function updateTopic(id, data) {
    const t = await api.updateTopic(id, data)
    const i = topics.value.findIndex((x) => x.id === id)
    if (i !== -1) topics.value[i] = t
    return t
  }

  async function deleteTopic(id) {
    await api.deleteTopic(id)
    topics.value = topics.value.filter((x) => x.id !== id)
    if (filters.topicId === id) setTopic(null)
  }

  // ── Посты ──
  // fetchPosts — первая страница: сбрасывает оба списка и курсор.
  async function fetchPosts() {
    loadingPosts.value = true
    try {
      const data = await api.getPosts({ topicId: filters.topicId, search: filters.search })
      posts.value = data.posts ?? []
      pinnedPosts.value = data.pinned ?? []
      nextCursor.value = data.next_cursor ?? null
    } finally {
      loadingPosts.value = false
    }
  }

  // fetchMore — догрузка следующей страницы хронологии по курсору (append
  // с дедупом по id: пост мог приехать сокетом post:new между страницами).
  async function fetchMore() {
    if (!nextCursor.value || loadingMore.value) return
    loadingMore.value = true
    try {
      const data = await api.getPosts({
        topicId: filters.topicId, search: filters.search, cursor: nextCursor.value,
      })
      const known = new Set([...posts.value, ...pinnedPosts.value].map((p) => p.id))
      posts.value.push(...(data.posts ?? []).filter((p) => !known.has(p.id)))
      nextCursor.value = data.next_cursor ?? null
    } finally {
      loadingMore.value = false
    }
  }

  function setTopic(id) {
    filters.topicId = id
    fetchPosts()
  }

  function setSearch(value) {
    filters.search = value
    fetchPosts()
  }

  // Пост по прямой ссылке /portal/:id.
  async function loadHighlight(id) {
    const numId = Number(id)
    if (!numId) { highlight.value = null; return }
    try {
      highlight.value = await api.getPost(numId)
    } catch {
      highlight.value = null
    }
  }

  // Пост ищем в хронологии, среди закреплённых и в подсвеченном по ссылке —
  // мутации реакций и счётчиков комментариев должны работать для всех.
  function findPost(id) {
    return posts.value.find((x) => x.id === id)
      ?? pinnedPosts.value.find((x) => x.id === id)
      ?? (highlight.value?.id === id ? highlight.value : null)
  }

  function applyLocalPost(post) {
    const i = posts.value.findIndex((p) => p.id === post.id)
    if (i !== -1) posts.value[i] = post
    const j = pinnedPosts.value.findIndex((p) => p.id === post.id)
    if (j !== -1) pinnedPosts.value[j] = post
    if (highlight.value?.id === post.id) highlight.value = post
    return post
  }

  // ── Перемещение между секциями «Закреплено» ↔ хронология ──
  // Идемпотентно (снапшот поста — источник истины): зовётся и локально из
  // pinPost/unpinPost, и из сокет-события того же действия.

  function movePinned(post) {
    posts.value = posts.value.filter((p) => p.id !== post.id)
    const i = pinnedPosts.value.findIndex((p) => p.id === post.id)
    if (i !== -1) pinnedPosts.value[i] = post
    else pinnedPosts.value.unshift(post)
    if (highlight.value?.id === post.id) highlight.value = post
  }

  // Возврат в хронологию — вставка по created_at (лента отсортирована по
  // убыванию). Пост старше всей загруженной страницы уйдёт в её конец —
  // допустимое упрощение: следующая fetchPosts/fetchMore выправит порядок.
  function moveUnpinned(post) {
    pinnedPosts.value = pinnedPosts.value.filter((p) => p.id !== post.id)
    if (!posts.value.some((p) => p.id === post.id)) {
      const i = posts.value.findIndex((p) => (p.created_at || '') <= (post.created_at || ''))
      if (i === -1) posts.value.push(post)
      else posts.value.splice(i, 0, post)
    }
    if (highlight.value?.id === post.id) highlight.value = post
  }

  async function createPost(payload) {
    const post = await api.createPost(payload)
    // Сокет-событие post:new может обогнать HTTP-ответ — не задваиваем пост.
    const i = posts.value.findIndex((p) => p.id === post.id)
    if (i !== -1) {
      posts.value[i] = post
    } else if (filters.topicId == null || filters.topicId === post.topic_id) {
      posts.value.unshift(post)
    }
    return post
  }

  async function updatePost(id, payload) {
    const post = await api.updatePost(id, payload)
    return applyLocalPost(post)
  }

  async function deletePost(id) {
    await api.deletePost(id)
    posts.value = posts.value.filter((p) => p.id !== id)
    pinnedPosts.value = pinnedPosts.value.filter((p) => p.id !== id)
    if (highlight.value?.id === id) highlight.value = null
  }

  // days: 1/7/30 или null (бессрочно) — срок выбирается в меню карточки.
  async function pinPost(id, days = null) {
    const post = await api.pinPost(id, days)
    movePinned(post)
    return post
  }

  async function unpinPost(id) {
    const post = await api.unpinPost(id)
    moveUnpinned(post)
    return post
  }

  async function uploadAttachment(postId, file) {
    const att = await api.uploadAttachment(postId, file)
    const p = findPost(postId)
    if (p) p.attachments = [...(p.attachments || []), att]
    return att
  }

  async function deleteAttachment(postId, attachmentId) {
    await api.deleteAttachment(attachmentId)
    const p = findPost(postId)
    if (p) p.attachments = (p.attachments || []).filter((a) => a.id !== attachmentId)
  }

  async function refreshPost(id) {
    try {
      applyLocalPost(await api.getPost(id))
    } catch { /* пост мог быть удалён параллельно */ }
  }

  // ── Реакции (оптимистично, свой user_id — идемпотентность сокет-события) ──
  async function addReaction(postId, emoji) {
    const p = findPost(postId)
    if (p) {
      p.reaction_counts = { ...(p.reaction_counts || {}), [emoji]: (p.reaction_counts?.[emoji] || 0) + 1 }
      p.my_reactions = [...new Set([...(p.my_reactions || []), emoji])]
    }
    try {
      await api.addReaction(postId, emoji)
    } catch (e) {
      if (p) {
        p.reaction_counts = { ...(p.reaction_counts || {}), [emoji]: Math.max(0, (p.reaction_counts?.[emoji] || 1) - 1) }
        p.my_reactions = (p.my_reactions || []).filter((x) => x !== emoji)
      }
      throw e
    }
  }

  async function removeReaction(postId, emoji) {
    const p = findPost(postId)
    const had = p?.my_reactions?.includes(emoji)
    if (p) {
      p.reaction_counts = { ...(p.reaction_counts || {}), [emoji]: Math.max(0, (p.reaction_counts?.[emoji] || 0) - 1) }
      p.my_reactions = (p.my_reactions || []).filter((x) => x !== emoji)
    }
    try {
      await api.removeReaction(postId, emoji)
    } catch (e) {
      if (p && had) {
        p.reaction_counts = { ...(p.reaction_counts || {}), [emoji]: (p.reaction_counts?.[emoji] || 0) + 1 }
        p.my_reactions = [...new Set([...(p.my_reactions || []), emoji])]
      }
      throw e
    }
  }

  // ── Просмотры ──
  // Отмечаем просмотр один раз за сессию на пост; счётчик наращиваем
  // оптимистично, только если сам зритель поста ещё не видел (post.viewed).
  async function markView(postId) {
    if (markedViews.has(postId)) return
    markedViews.add(postId)
    const p = findPost(postId)
    const firstView = p && !p.viewed
    if (firstView) {
      p.viewed = true
      p.view_count = (p.view_count || 0) + 1
    }
    try {
      await api.markView(postId)
    } catch {
      // Разрешаем повтор при следующем показе; откатываем оптимистичный счёт.
      markedViews.delete(postId)
      if (firstView && p) {
        p.viewed = false
        p.view_count = Math.max(0, (p.view_count || 1) - 1)
      }
    }
  }

  // ── Комментарии (плоские) ──
  async function fetchComments(postId) {
    loadingComments[postId] = true
    try {
      const data = await api.getComments(postId)
      commentsByPost[postId] = data.comments ?? []
    } finally {
      loadingComments[postId] = false
    }
  }

  // Список — оптимистично (мгновенная обратная связь); comment_count поста
  // бампается ТОЛЬКО в сокет-обработчике (см. applyCommentSocket) — так
  // счётчик меняется ровно один раз на реальное событие, будь оно своё или
  // чужое, без двойного учёта.
  async function createComment(postId, text, replyToId = null) {
    const c = await api.createComment(postId, text, replyToId)
    const list = commentsByPost[postId]
    if (list && !list.some((x) => x.id === c.id)) list.push(c)
    return c
  }

  // Удаление уносит и ветку ответов (каскад FK на бэке) — из списка убираем
  // ровно то же поддерево, иначе осиротевшие ответы висели бы до перезагрузки.
  async function deleteComment(postId, commentId) {
    await api.deleteComment(commentId)
    dropCommentSubtree(postId, commentId)
  }

  // dropCommentSubtree — убрать комментарий со всеми потомками; возвращает,
  // сколько удалено (счётчик поста двигается ровно на это число).
  function dropCommentSubtree(postId, commentId) {
    const list = commentsByPost[postId]
    if (!list) return 0
    const doomed = new Set([commentId])
    // Список в хронологии: родитель всегда раньше ответа, одного прохода хватает.
    for (const c of list) {
      if (c.reply_to_id != null && doomed.has(c.reply_to_id)) doomed.add(c.id)
    }
    commentsByPost[postId] = list.filter((c) => !doomed.has(c.id))
    return doomed.size
  }

  // Лайк комментария — toggle: ответ авторитетен (счётчик считает сервер),
  // остальным прилетит comment:liked.
  async function likeComment(postId, commentId) {
    const res = await api.likeComment(commentId)
    applyCommentLike(postId, commentId, { liked: res.liked, like_count: res.like_count })
    return res
  }

  function applyCommentLike(postId, commentId, patch) {
    const list = commentsByPost[postId]
    if (!list) return
    const c = list.find((x) => x.id === commentId)
    if (c) Object.assign(c, patch)
  }

  function bumpCommentCount(postId, delta) {
    const p = findPost(postId)
    if (p) p.comment_count = Math.max(0, (p.comment_count || 0) + delta)
  }

  // ── Пересылка ──
  async function forwardPost(postId, opts) {
    return api.forwardPost(postId, opts)
  }

  // ── Сокет-события ──
  function applyTopicSocket(kind, payload) {
    if (!isMine(payload?.company_id)) return
    if (kind === 'deleted') {
      topics.value = topics.value.filter((t) => t.id !== payload.id)
      if (filters.topicId === payload.id) setTopic(null)
      return
    }
    const i = topics.value.findIndex((t) => t.id === payload.id)
    if (i === -1) topics.value.push(payload)
    else topics.value[i] = { ...topics.value[i], ...payload }
  }

  function applyPostSocket(kind, payload) {
    if (!isMine(payload?.company_id)) return
    // Бейдж непрочитанных: чужой новый пост либо уже виден (портал открыт —
    // сразу подтверждаем просмотр серверу), либо наращивает счётчик.
    if (kind === 'new' && payload.author_id !== useAuthStore().userId) {
      if (viewingFeed.value) markSeen()
      else unread.value++
    }
    if (kind === 'deleted') {
      posts.value = posts.value.filter((p) => p.id !== payload.id)
      pinnedPosts.value = pinnedPosts.value.filter((p) => p.id !== payload.id)
      if (highlight.value?.id === payload.id) highlight.value = null
      // Удалённый пост мог быть в числе непрочитанных — тихая серверная коррекция.
      if (unread.value > 0) fetchUnread()
      return
    }
    // Ping от загрузки вложения — частичный payload без снапшота поста.
    if (payload.attachment_added) {
      refreshPost(payload.id)
      return
    }
    // Закрепление/открепление — перемещение между секциями (payload —
    // полный снапшот поста; своё действие уже применено идемпотентным
    // movePinned/moveUnpinned в pinPost/unpinPost).
    if (kind === 'pinned' || kind === 'unpinned') {
      // Незнакомый пост вне активного фильтра не вмешиваем в выдачу.
      const known = !!findPost(payload.id)
      if (!known && (filters.search || (filters.topicId != null && filters.topicId !== payload.topic_id))) return
      if (kind === 'pinned') movePinned(payload)
      else moveUnpinned(payload)
      return
    }
    if (highlight.value?.id === payload.id) {
      highlight.value = { ...highlight.value, ...payload }
    }
    const j = pinnedPosts.value.findIndex((p) => p.id === payload.id)
    if (j !== -1) {
      pinnedPosts.value[j] = { ...pinnedPosts.value[j], ...payload }
      return
    }
    const i = posts.value.findIndex((p) => p.id === payload.id)
    if (i === -1) {
      // При активном поиске чужой новый пост не вмешиваем в отфильтрованную выдачу.
      if (kind === 'new' && filters.search) return
      if (filters.topicId == null || filters.topicId === payload.topic_id) posts.value.unshift(payload)
    } else {
      posts.value[i] = { ...posts.value[i], ...payload }
    }
  }

  function applyCommentSocket(kind, payload) {
    if (!isMine(payload?.company_id)) return
    const postId = payload.post_id
    if (kind === 'new') {
      const list = commentsByPost[postId]
      if (list && !list.some((c) => c.id === payload.id)) list.push(payload)
      bumpCommentCount(postId, 1)
    } else if (kind === 'liked') {
      // Свой лайк уже применён ответом likeComment — обновляем только счётчик
      // (флаг «мой лайк» у каждого свой и в событие не входит).
      applyCommentLike(postId, payload.id, { like_count: payload.like_count })
    } else {
      // Удалён родитель — уходит вся ветка (каскад FK); счётчик поста двигаем
      // на реальное число удалённых, а не на единицу.
      const removed = dropCommentSubtree(postId, payload.id)
      bumpCommentCount(postId, -(removed || 1))
    }
  }

  function applyReactionSocket(kind, payload) {
    if (!isMine(payload?.company_id)) return
    // Своё действие уже применено оптимистично в addReaction/removeReaction.
    if (payload.user_id === useAuthStore().userId) return
    const p = findPost(payload.post_id)
    if (!p) return
    const delta = kind === 'added' ? 1 : -1
    p.reaction_counts = { ...(p.reaction_counts || {}), [payload.emoji]: Math.max(0, (p.reaction_counts?.[payload.emoji] || 0) + delta) }
  }

  // Смена компании / логаут — данные прежней компании не должны утекать в UI.
  function reset() {
    topics.value = []
    posts.value = []
    pinnedPosts.value = []
    nextCursor.value = null
    loadingMore.value = false
    highlight.value = null
    // viewingFeed не трогаем: это состояние открытого экрана, а не данных.
    unread.value = 0
    filters.topicId = null
    filters.search = ''
    authorMap.value = new Map()
    markedViews.clear()
    for (const k of Object.keys(commentsByPost)) delete commentsByPost[k]
    for (const k of Object.keys(loadingComments)) delete loadingComments[k]
  }

  return {
    topics, posts, pinnedPosts, highlight, loadingTopics, loadingPosts, filters,
    nextCursor, loadingMore,
    unread, viewingFeed, fetchUnread, markSeen,
    authorMap, commentsByPost, loadingComments,
    loadAuthors, resolveAuthor,
    fetchTopics, createTopic, updateTopic, deleteTopic,
    fetchPosts, fetchMore, setTopic, setSearch, loadHighlight,
    createPost, updatePost, deletePost, pinPost, unpinPost, markView,
    uploadAttachment, deleteAttachment, addReaction, removeReaction,
    fetchComments, createComment, deleteComment, likeComment,
    forwardPost,
    applyTopicSocket, applyPostSocket, applyCommentSocket, applyReactionSocket,
    reset,
  }
})
