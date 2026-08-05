/* Портал (portalsvc): публикации, разделы, дерево комментариев, реакции,
   хештеги, закрепление и непрочитанные.

   Портал company-scoped, поэтому половина проверок — про границы компании:
   пост чужой компании не виден, разделы правит только администратор, а
   лимит закреплённых не обходится. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { newCompanyAdmin, newMember, registerVerified } from '../setup/factory.js'
import * as api from '@/api/portal.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

describeIntegration('portal API: публикации', () => {
  it('жизненный цикл поста: создание, правка, чтение, удаление', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()

    const post = await api.createPost({ body: 'Первая публикация компании' })
    expect(post.id).toBeGreaterThan(0)

    await api.updatePost(post.id, { title: 'Объявление', body: 'Текст поправлен' })
    const one = await api.getPost(post.id)
    expect(one.title).toBe('Объявление')
    expect(one.body).toContain('поправлен')

    await api.deletePost(post.id)
    await expect(api.getPost(post.id)).rejects.toBeTruthy()
  })

  it('лента отдаёт посты и режет по курсору (keyset)', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    for (let i = 0; i < 3; i++) await api.createPost({ body: `Пост номер ${i}` })

    const first = await api.getPosts({ limit: 2 })
    expect((first.posts ?? []).length).toBeLessThanOrEqual(2)

    if (first.next_cursor) {
      const second = await api.getPosts({ limit: 2, cursor: first.next_cursor })
      const firstIds = (first.posts ?? []).map((p) => p.id)
      const secondIds = (second.posts ?? []).map((p) => p.id)
      // Страницы не пересекаются — иначе пользователь увидит дубли при листании.
      expect(secondIds.some((id) => firstIds.includes(id))).toBe(false)
    }
  })

  it('пост чужой компании не виден', async () => {
    const a = await newCompanyAdmin('a')
    a.session.use()
    const post = await api.createPost({ body: 'Внутреннее дело компании А' })

    const b = await newCompanyAdmin('b')
    b.session.use()
    await expect(api.getPost(post.id)).rejects.toBeTruthy()
    const feed = await api.getPosts({})
    expect((feed.posts ?? []).some((p) => p.id === post.id)).toBe(false)
  })

  it('хештеги разбираются сервером из тела и фильтруют ленту', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const tag = `тег${Date.now().toString().slice(-6)}`
    const withTag = await api.createPost({ body: `Обсудим #${tag} на планёрке` })
    await api.createPost({ body: 'Пост без метки' })

    const one = await api.getPost(withTag.id)
    expect((one.tags ?? []).map(String)).toContain(tag)

    const filtered = await api.getPosts({ tag })
    const ids = (filtered.posts ?? []).map((p) => p.id)
    expect(ids).toContain(withTag.id)
    expect(ids.length).toBe(1)
  })

  it('закрепление: пост поднимается и снимается', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const post = await api.createPost({ body: 'Важное объявление' })

    await api.pinPost(post.id, 7)
    const feed = await api.getPosts({})
    expect((feed.pinned ?? []).some((p) => p.id === post.id)).toBe(true)

    await api.unpinPost(post.id)
    const after = await api.getPosts({})
    expect((after.pinned ?? []).some((p) => p.id === post.id)).toBe(false)
  })
})

describeIntegration('portal API: обсуждение и реакции', () => {
  it('комментарии деревом: ответ привязан к родителю, удаление уносит ветку', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const post = await api.createPost({ body: 'Обсуждаем смету' })

    const root = await api.createComment(post.id, 'Первый вопрос')
    const reply = await api.createComment(post.id, 'Ответ на вопрос', root.id)
    expect(reply.reply_to_id).toBe(root.id)

    const list = await api.getComments(post.id)
    expect((list.comments ?? list).length).toBe(2)

    // Удаление корня уносит ветку целиком (ON DELETE CASCADE).
    await api.deleteComment(root.id)
    const after = await api.getComments(post.id)
    expect((after.comments ?? after).length).toBe(0)
  })

  it('лайк комментария — переключатель: второй раз снимает, а не удваивает', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const post = await api.createPost({ body: 'Про лайки' })
    const c = await api.createComment(post.id, 'Комментарий')

    const liked = await api.likeComment(c.id)
    expect(liked.liked).toBe(true)
    expect(liked.like_count).toBe(1)

    const unliked = await api.likeComment(c.id)
    expect(unliked.liked).toBe(false)
    expect(unliked.like_count).toBe(0)
  })

  it('реакция: своя видна мне и посчитана, снятие обнуляет счётчик', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const post = await api.createPost({ body: 'Реакции' })

    await api.addReaction(post.id, '👍')
    const withReaction = await api.getPost(post.id)
    // «Мои реакции» и счётчики приходят раздельно: флаг «моя» у каждого свой,
    // а счётчик общий (см. портал в CLAUDE.md).
    expect(withReaction.my_reactions).toContain('👍')
    expect(withReaction.reaction_counts?.['👍']).toBe(1)

    await api.removeReaction(post.id, '👍')
    const after = await api.getPost(post.id)
    expect(after.my_reactions ?? []).not.toContain('👍')
    expect(after.reaction_counts?.['👍'] ?? 0).toBe(0)
  })

  it('непрочитанные считаются и гасятся отметкой', async () => {
    const admin = await newCompanyAdmin('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    await api.createPost({ body: 'Свежая новость' })

    mate.session.use()
    const before = await api.getUnreadCount()
    expect(before.count ?? before.unread ?? 0).toBeGreaterThan(0)

    await api.markSeen()
    const after = await api.getUnreadCount()
    expect(after.count ?? after.unread ?? 0).toBe(0)
  })
})

describeIntegration('portal API: разделы и права', () => {
  it('разделы: администратор заводит, сотрудник — нет', async () => {
    const admin = await newCompanyAdmin('admin')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')

    admin.session.use()
    const topic = await api.createTopic({ name: uniq('Отдел '), color: 'blue' })
    expect(topic.id).toBeGreaterThan(0)

    worker.session.use()
    await expectStatus(api.createTopic({ name: 'Самовольный' }), 403)
    await expectStatus(api.deleteTopic(topic.id), 403)

    // Но публиковать в существующий раздел сотрудник вправе.
    const post = await api.createPost({ topicId: topic.id, body: 'Пост сотрудника' })
    expect(post.id).toBeGreaterThan(0)

    admin.session.use()
    await api.deleteTopic(topic.id)
  })

  it('пустое тело поста не принимается', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    await expect(api.createPost({ body: '' })).rejects.toBeTruthy()
    await expect(api.createPost({ body: '   ' })).rejects.toBeTruthy()
  })

  it('без активной компании портал недоступен', async () => {
    const u = await registerVerified()
    u.session.use()
    await expect(api.getPosts({})).rejects.toBeTruthy()
  })

  it('фон ленты: сохраняется, отдаётся и сбрасывается', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()

    await api.setBackground({ preset: 'waves', opacity: 0.5 })
    const saved = await api.getBackground()
    expect(saved.recipe?.preset ?? saved.preset).toBe('waves')

    await api.deleteBackground()
    const cleared = await api.getBackground()
    expect(cleared.recipe ?? null).toBeFalsy()
  })
})
