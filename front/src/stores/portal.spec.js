import { describe, it, expect, beforeEach, vi } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'

// Мокаем API-слой — стор тестируем без сети.
vi.mock('@/api/portal.js', () => ({
  getTopics: vi.fn(() => Promise.resolve({ topics: [] })),
  createTopic: vi.fn(() => Promise.resolve({})),
  updateTopic: vi.fn(() => Promise.resolve({})),
  deleteTopic: vi.fn(() => Promise.resolve({})),
  getPosts: vi.fn(() => Promise.resolve({ pinned: [], posts: [], next_cursor: null })),
  getPost: vi.fn(() => Promise.resolve({})),
  createPost: vi.fn(() => Promise.resolve({})),
  updatePost: vi.fn(() => Promise.resolve({})),
  deletePost: vi.fn(() => Promise.resolve({})),
  pinPost: vi.fn(() => Promise.resolve({})),
  unpinPost: vi.fn(() => Promise.resolve({})),
  uploadAttachment: vi.fn(() => Promise.resolve({})),
  getComments: vi.fn(() => Promise.resolve({ comments: [] })),
  createComment: vi.fn(() => Promise.resolve({})),
  deleteComment: vi.fn(() => Promise.resolve({})),
  addReaction: vi.fn(() => Promise.resolve({})),
  removeReaction: vi.fn(() => Promise.resolve({})),
  forwardPost: vi.fn(() => Promise.resolve({})),
  getUnreadCount: vi.fn(() => Promise.resolve({ count: 0 })),
  markSeen: vi.fn(() => Promise.resolve({ status: 'ok' })),
}))

vi.mock('@/api/users.js', () => ({
  getDirectory: vi.fn(() => Promise.resolve([])),
}))

import * as api from '@/api/portal.js'
import { usePortalStore } from './portal.js'
import { useAuthStore } from './auth.js'

describe('portal store', () => {
  let store
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    store = usePortalStore()
  })

  function setMe(id, companyId = 10) {
    useAuthStore().applySession({ access_token: 't', user_id: id, company_id: companyId })
  }

  describe('серверная пагинация ленты', () => {
    it('fetchPosts (первая страница) заполняет pinned/posts/next_cursor с сервера', async () => {
      api.getPosts.mockResolvedValueOnce({
        pinned: [{ id: 1, pinned_at: '2026-07-01T00:00:00Z' }],
        posts: [{ id: 2 }, { id: 3 }],
        next_cursor: 'abc',
      })
      await store.fetchPosts()
      expect(store.pinnedPosts.map((p) => p.id)).toEqual([1])
      expect(store.posts.map((p) => p.id)).toEqual([2, 3])
      expect(store.nextCursor).toBe('abc')
    })

    it('fetchMore догружает по курсору: append с дедупом по id, курсор обновляется', async () => {
      store.posts = [{ id: 2 }, { id: 3 }]
      store.pinnedPosts = [{ id: 1 }]
      store.nextCursor = 'abc'
      api.getPosts.mockResolvedValueOnce({
        pinned: [],
        posts: [{ id: 3 }, { id: 1 }, { id: 4 }], // 3 и 1 уже загружены
        next_cursor: null,
      })
      await store.fetchMore()
      expect(api.getPosts).toHaveBeenCalledWith(expect.objectContaining({ cursor: 'abc' }))
      expect(store.posts.map((p) => p.id)).toEqual([2, 3, 4])
      expect(store.nextCursor).toBeNull()
    })

    it('fetchMore без курсора — no-op (постов больше нет)', async () => {
      store.nextCursor = null
      await store.fetchMore()
      expect(api.getPosts).not.toHaveBeenCalled()
    })

    it('fetchPosts сбрасывает оба списка и курсор (первая страница заново)', async () => {
      store.posts = [{ id: 9 }]
      store.pinnedPosts = [{ id: 8 }]
      store.nextCursor = 'old'
      api.getPosts.mockResolvedValueOnce({ pinned: [], posts: [], next_cursor: null })
      await store.fetchPosts()
      expect(store.posts).toHaveLength(0)
      expect(store.pinnedPosts).toHaveLength(0)
      expect(store.nextCursor).toBeNull()
    })
  })

  describe('pinPost / unpinPost — локальное перемещение между секциями', () => {
    it('pinPost(id, days) передаёт срок в API и переносит пост в pinnedPosts', async () => {
      store.posts = [{ id: 1, created_at: '2026-07-01T00:00:00Z', pinned_at: null }]
      const snap = { id: 1, created_at: '2026-07-01T00:00:00Z', pinned_at: '2026-07-05T00:00:00Z', pinned_until: '2026-07-12T00:00:00Z' }
      api.pinPost.mockResolvedValueOnce(snap)
      await store.pinPost(1, 7)
      expect(api.pinPost).toHaveBeenCalledWith(1, 7)
      expect(store.posts).toHaveLength(0)
      expect(store.pinnedPosts.map((p) => p.id)).toEqual([1])
      expect(store.pinnedPosts[0].pinned_until).toBe('2026-07-12T00:00:00Z')
    })

    it('unpinPost возвращает пост в хронологию по created_at', async () => {
      store.pinnedPosts = [{ id: 1, created_at: '2026-07-02T00:00:00Z', pinned_at: 'x' }]
      store.posts = [
        { id: 3, created_at: '2026-07-03T00:00:00Z' },
        { id: 2, created_at: '2026-07-01T00:00:00Z' },
      ]
      api.unpinPost.mockResolvedValueOnce({ id: 1, created_at: '2026-07-02T00:00:00Z', pinned_at: null })
      await store.unpinPost(1)
      expect(store.pinnedPosts).toHaveLength(0)
      expect(store.posts.map((p) => p.id)).toEqual([3, 1, 2])
    })
  })

  describe('applyPostSocket', () => {
    it('чужая компания игнорируется', () => {
      setMe(1, 10)
      store.applyPostSocket('new', { id: 5, company_id: 99, topic_id: null })
      expect(store.posts).toHaveLength(0)
    })

    it('new своей компании добавляет пост в начало ленты', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10, topic_id: null }]
      store.applyPostSocket('new', { id: 2, company_id: 10, topic_id: null })
      expect(store.posts.map((p) => p.id)).toEqual([2, 1])
    })

    it('идемпотентность: повторный new тем же id обновляет, а не дублирует', () => {
      setMe(1, 10)
      store.applyPostSocket('new', { id: 1, company_id: 10, topic_id: null, body: 'v1' })
      store.applyPostSocket('updated', { id: 1, company_id: 10, topic_id: null, body: 'v2' })
      expect(store.posts).toHaveLength(1)
      expect(store.posts[0].body).toBe('v2')
    })

    it('deleted убирает пост из ленты', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10 }]
      store.applyPostSocket('deleted', { id: 1, company_id: 10 })
      expect(store.posts).toHaveLength(0)
    })

    it('пост вне активного фильтра по топику не добавляется в ленту', () => {
      setMe(1, 10)
      store.filters.topicId = 3
      store.applyPostSocket('new', { id: 9, company_id: 10, topic_id: 7 })
      expect(store.posts).toHaveLength(0)
    })

    it('pinned переносит пост из хронологии в закреплённые (payload — снапшот)', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10, pinned_at: null }]
      store.applyPostSocket('pinned', { id: 1, company_id: 10, pinned_at: '2026-07-05T00:00:00Z' })
      expect(store.posts).toHaveLength(0)
      expect(store.pinnedPosts.map((p) => p.id)).toEqual([1])
    })

    it('unpinned возвращает пост из закреплённых в хронологию', () => {
      setMe(1, 10)
      store.pinnedPosts = [{ id: 1, company_id: 10, created_at: '2026-07-01T00:00:00Z', pinned_at: 'x' }]
      store.posts = [{ id: 2, company_id: 10, created_at: '2026-07-02T00:00:00Z' }]
      store.applyPostSocket('unpinned', { id: 1, company_id: 10, created_at: '2026-07-01T00:00:00Z', pinned_at: null })
      expect(store.pinnedPosts).toHaveLength(0)
      expect(store.posts.map((p) => p.id)).toEqual([2, 1])
    })

    it('pinned идемпотентен: повтор события не дублирует пост в закреплённых', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10 }]
      const snap = { id: 1, company_id: 10, pinned_at: 'x' }
      store.applyPostSocket('pinned', snap)
      store.applyPostSocket('pinned', snap)
      expect(store.pinnedPosts).toHaveLength(1)
    })

    it('updated обновляет пост и в секции закреплённых', () => {
      setMe(1, 10)
      store.pinnedPosts = [{ id: 1, company_id: 10, body: 'v1', pinned_at: 'x' }]
      store.applyPostSocket('updated', { id: 1, company_id: 10, body: 'v2' })
      expect(store.pinnedPosts[0].body).toBe('v2')
      expect(store.posts).toHaveLength(0) // не «раздвоился» в хронологию
    })

    it('deleted убирает пост и из закреплённых', () => {
      setMe(1, 10)
      store.pinnedPosts = [{ id: 1, company_id: 10, pinned_at: 'x' }]
      store.applyPostSocket('deleted', { id: 1, company_id: 10 })
      expect(store.pinnedPosts).toHaveLength(0)
    })
  })

  describe('бейдж непрочитанных постов', () => {
    it('чужой post:new вне портала наращивает unread', () => {
      setMe(1, 10)
      store.applyPostSocket('new', { id: 5, company_id: 10, topic_id: null, author_id: 2 })
      expect(store.unread).toBe(1)
      expect(api.markSeen).not.toHaveBeenCalled()
    })

    it('свой post:new счётчик не трогает', () => {
      setMe(1, 10)
      store.applyPostSocket('new', { id: 5, company_id: 10, topic_id: null, author_id: 1 })
      expect(store.unread).toBe(0)
    })

    it('пост чужой компании счётчик не трогает', () => {
      setMe(1, 10)
      store.applyPostSocket('new', { id: 5, company_id: 99, topic_id: null, author_id: 2 })
      expect(store.unread).toBe(0)
    })

    it('при открытом портале (viewingFeed) чужой post:new сразу подтверждается просмотром', () => {
      setMe(1, 10)
      store.viewingFeed = true
      store.applyPostSocket('new', { id: 5, company_id: 10, topic_id: null, author_id: 2 })
      expect(store.unread).toBe(0)
      expect(api.markSeen).toHaveBeenCalled()
    })

    it('markSeen гасит счётчик локально и шлёт серверную отметку', () => {
      store.unread = 7
      store.markSeen()
      expect(store.unread).toBe(0)
      expect(api.markSeen).toHaveBeenCalled()
    })

    it('fetchUnread берёт серверный счётчик; ошибка сети счётчик не трогает', async () => {
      api.getUnreadCount.mockResolvedValueOnce({ count: 3 })
      await store.fetchUnread()
      expect(store.unread).toBe(3)
      api.getUnreadCount.mockRejectedValueOnce(new Error('fail'))
      await store.fetchUnread()
      expect(store.unread).toBe(3)
    })

    it('post:deleted при unread>0 корректирует счётчик с сервера', async () => {
      setMe(1, 10)
      store.unread = 2
      api.getUnreadCount.mockResolvedValueOnce({ count: 1 })
      store.applyPostSocket('deleted', { id: 5, company_id: 10 })
      await Promise.resolve()
      expect(api.getUnreadCount).toHaveBeenCalled()
    })

    it('reset обнуляет unread и состояние пагинации', () => {
      store.unread = 4
      store.pinnedPosts = [{ id: 1 }]
      store.nextCursor = 'abc'
      store.reset()
      expect(store.unread).toBe(0)
      expect(store.pinnedPosts).toHaveLength(0)
      expect(store.nextCursor).toBeNull()
    })
  })

  describe('applyReactionSocket (идемпотентность своих действий)', () => {
    it('чужая реакция увеличивает счётчик', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10, reaction_counts: { '👍': 1 } }]
      store.applyReactionSocket('added', { post_id: 1, user_id: 2, emoji: '👍', company_id: 10 })
      expect(store.posts[0].reaction_counts['👍']).toBe(2)
    })

    it('своя реакция игнорируется в сокет-обработчике (уже применена оптимистично)', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10, reaction_counts: { '👍': 1 }, my_reactions: ['👍'] }]
      store.applyReactionSocket('added', { post_id: 1, user_id: 1, emoji: '👍', company_id: 10 })
      // Счётчик не бампается повторно сверх уже применённого оптимистично.
      expect(store.posts[0].reaction_counts['👍']).toBe(1)
    })
  })

  describe('addReaction / removeReaction (оптимистично + откат при ошибке)', () => {
    it('успешный addReaction применяет счётчик и остаётся применённым', async () => {
      store.posts = [{ id: 1, reaction_counts: {}, my_reactions: [] }]
      await store.addReaction(1, '🎉')
      expect(store.posts[0].reaction_counts['🎉']).toBe(1)
      expect(store.posts[0].my_reactions).toContain('🎉')
      expect(api.addReaction).toHaveBeenCalledWith(1, '🎉')
    })

    it('ошибка API откатывает оптимистичное изменение', async () => {
      api.addReaction.mockRejectedValueOnce(new Error('fail'))
      store.posts = [{ id: 1, reaction_counts: {}, my_reactions: [] }]
      await expect(store.addReaction(1, '🎉')).rejects.toThrow('fail')
      expect(store.posts[0].reaction_counts['🎉']).toBe(0)
      expect(store.posts[0].my_reactions).not.toContain('🎉')
    })
  })

  describe('applyCommentSocket (счётчик бампается ровно один раз)', () => {
    it('new — добавляет комментарий в загруженный список и увеличивает comment_count один раз', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10, comment_count: 0 }]
      store.commentsByPost[1] = []
      store.applyCommentSocket('new', { id: 100, post_id: 1, author_id: 2, text: 'hi', company_id: 10 })
      expect(store.commentsByPost[1]).toHaveLength(1)
      expect(store.posts[0].comment_count).toBe(1)
    })

    it('свой комментарий уже в списке (добавлен createComment) — сокет не дублирует запись, но досчитывает счётчик один раз', async () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10, comment_count: 0 }]
      store.commentsByPost[1] = []
      api.createComment.mockResolvedValueOnce({ id: 200, post_id: 1, author_id: 1, text: 'mine' })
      await store.createComment(1, 'mine')
      expect(store.commentsByPost[1]).toHaveLength(1)
      expect(store.posts[0].comment_count).toBe(0) // ещё не бампался — только list

      store.applyCommentSocket('new', { id: 200, post_id: 1, author_id: 1, text: 'mine', company_id: 10 })
      expect(store.commentsByPost[1]).toHaveLength(1) // не задублировалось
      expect(store.posts[0].comment_count).toBe(1) // бампнулось ровно один раз
    })

    it('deleted убирает комментарий и уменьшает comment_count (не ниже нуля)', () => {
      setMe(1, 10)
      store.posts = [{ id: 1, company_id: 10, comment_count: 0 }]
      store.commentsByPost[1] = [{ id: 1, post_id: 1 }]
      store.applyCommentSocket('deleted', { id: 1, post_id: 1, company_id: 10 })
      expect(store.commentsByPost[1]).toHaveLength(0)
      expect(store.posts[0].comment_count).toBe(0)
    })
  })

  describe('pinPost — ошибка лимита закреплённых (TOO_MANY_PINNED) не дублирует бизнес-логику бэка', () => {
    it('пробрасывает ошибку бэка и не меняет локальное состояние', async () => {
      api.pinPost.mockRejectedValueOnce({ error: 'TOO_MANY_PINNED', message: 'Слишком много закреплённых постов' })
      store.posts = [{ id: 1, pinned_at: null }]
      await expect(store.pinPost(1)).rejects.toMatchObject({ error: 'TOO_MANY_PINNED' })
      expect(store.posts[0].pinned_at).toBeNull()
    })
  })

  describe('applyTopicSocket', () => {
    it('created добавляет топик своей компании', () => {
      setMe(1, 10)
      store.applyTopicSocket('created', { id: 1, company_id: 10, name: 'Новости' })
      expect(store.topics.map((t) => t.id)).toEqual([1])
    })

    it('deleted сбрасывает активный фильтр, если он указывал на удалённый топик', () => {
      setMe(1, 10)
      store.topics = [{ id: 1, company_id: 10, name: 'A' }]
      store.filters.topicId = 1
      store.applyTopicSocket('deleted', { id: 1, company_id: 10 })
      expect(store.topics).toHaveLength(0)
      expect(store.filters.topicId).toBeNull()
    })
  })
})
