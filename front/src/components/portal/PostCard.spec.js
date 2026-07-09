import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createTestingPinia } from '@pinia/testing'
import { setActivePinia } from 'pinia'

// Store-экшены реальные (stubActions: false) — мокаем API-слой закрепления.
vi.mock('@/api/portal.js', () => ({
  pinPost: vi.fn((id, days) => Promise.resolve({ id, pinned_at: '2026-07-05T00:00:00Z', pinned_until: null, days })),
  unpinPost: vi.fn((id) => Promise.resolve({ id, pinned_at: null })),
}))

import * as api from '@/api/portal.js'
import PostCard from './PostCard.vue'
import { useAuthStore } from '@/stores/auth.js'

function mkPost(over = {}) {
  return {
    id: 1, company_id: 10, topic_id: null, author_id: 5,
    title: 'Заголовок', body: 'Текст поста',
    pinned_at: null, pinned_by: null,
    created_at: '2026-07-01T10:00:00Z', updated_at: '2026-07-01T10:00:00Z',
    attachments: [], comment_count: 0, reaction_counts: {}, my_reactions: [],
    ...over,
  }
}

// claims приватны в auth-сторе (см. usePermission.spec.js) — роль/id
// выставляем публичным applySession, а не initialState.
function factory(post, { roleLevel = 1, userId = 1, topics = [] } = {}) {
  const pinia = createTestingPinia({
    createSpy: vi.fn,
    stubActions: false,
    initialState: {
      portal: { topics, authorMap: new Map([[5, { id: 5, fio: 'Иван Иванов', avatar_path: null }]]) },
    },
  })
  setActivePinia(pinia)
  useAuthStore().applySession({ access_token: 't', user_id: userId, role_level: roleLevel })
  return mount(PostCard, {
    props: { post },
    global: { plugins: [pinia], stubs: { CommentsList: true } },
  })
}

describe('PostCard', () => {
  it('показывает резолвленного автора (fio из каталога сотрудников по author_id) и тело поста', () => {
    const w = factory(mkPost())
    expect(w.text()).toContain('Иван Иванов')
    expect(w.text()).toContain('Текст поста')
    expect(w.text()).toContain('Заголовок')
  })

  it('бейдж и класс «Закреплено» — только у закреплённого поста', () => {
    const unpinned = factory(mkPost())
    expect(unpinned.find('.post-pin-badge').exists()).toBe(false)
    expect(unpinned.find('.post-card').classes()).not.toContain('pinned')

    const pinned = factory(mkPost({ pinned_at: '2026-07-01T12:00:00Z' }))
    expect(pinned.find('.post-pin-badge').exists()).toBe(true)
    expect(pinned.find('.post-card').classes()).toContain('pinned')
  })

  it('меню управления (редактировать/удалить/закрепить) видно только автору или администратору', () => {
    const stranger = factory(mkPost({ author_id: 999 }), { roleLevel: 1, userId: 1 })
    expect(stranger.find('.post-menu').exists()).toBe(false)

    const author = factory(mkPost({ author_id: 1 }), { roleLevel: 1, userId: 1 })
    expect(author.find('.post-menu').exists()).toBe(true)

    const admin = factory(mkPost({ author_id: 999 }), { roleLevel: 3, userId: 1 })
    expect(admin.find('.post-menu').exists()).toBe(true)
  })

  it('реакции: показывает счётчик и подсвечивает свою реакцию активной', () => {
    const w = factory(mkPost({ reaction_counts: { '👍': 3 }, my_reactions: ['👍'] }))
    const btn = w.findAll('.post-reaction').find((b) => b.text().includes('👍'))
    expect(btn.classes()).toContain('active')
    expect(btn.text()).toContain('3')

    const other = w.findAll('.post-reaction').find((b) => b.text().includes('❤️'))
    expect(other.classes()).not.toContain('active')
  })

  it('топик поста отображается чипом с его названием', () => {
    const w = factory(mkPost({ topic_id: 7 }), { topics: [{ id: 7, name: 'Новости', color: 'blue' }] })
    expect(w.find('.post-topic-chip').text()).toBe('Новости')
  })

  it('счётчик комментариев отражает post.comment_count', () => {
    const w = factory(mkPost({ comment_count: 4 }))
    expect(w.text()).toContain('4')
  })

  it('тело поста рендерится как markdown', () => {
    const w = factory(mkPost({ body: '# Итоги\n**важно** и `код`\n- [x] готово' }))
    expect(w.find('.post-body-md h1').text()).toBe('Итоги')
    expect(w.find('.post-body-md strong').text()).toBe('важно')
    expect(w.find('.post-body-md .md-code').text()).toBe('код')
    expect(w.find('.post-body-md .md-task input[checked]').exists()).toBe(true)
  })

  it('сворачивание — по фактической высоте: высокий блок получает кнопку и класс collapsed', async () => {
    // В jsdom layout нет — scrollHeight мокается, замер триггерится watch'ем на body.
    const w = factory(mkPost({ body: 'x' }))
    Object.defineProperty(w.find('.post-body-md').element, 'scrollHeight', { value: 900 })
    await w.setProps({ post: mkPost({ body: 'длинный текст' }) })
    await w.vm.$nextTick()
    await w.vm.$nextTick()
    expect(w.find('.post-more-btn').exists()).toBe(true)
    expect(w.find('.post-body-md').classes()).toContain('collapsed')

    await w.find('.post-more-btn').trigger('click')
    expect(w.find('.post-body-md').classes()).not.toContain('collapsed')
  })

  it('пункт «Закрепить» раскрывает выбор срока; выбор «7 дней» зовёт API с days=7', async () => {
    const w = factory(mkPost({ author_id: 1 }), { userId: 1 })
    await w.find('.post-icon-btn').trigger('click')
    const pinBtn = w.findAll('.post-menu-item').find((b) => b.text().includes('Закрепить'))
    await pinBtn.trigger('click')

    const options = w.findAll('.post-menu-sub')
    expect(options.map((o) => o.text())).toEqual(['1 день', '7 дней', '30 дней', 'Бессрочно'])

    await options.find((o) => o.text() === '7 дней').trigger('click')
    expect(api.pinPost).toHaveBeenCalledWith(1, 7)
  })

  it('«Бессрочно» зовёт API с days=null; у закреплённого — только «Открепить»', async () => {
    const w = factory(mkPost({ author_id: 1 }), { userId: 1 })
    await w.find('.post-icon-btn').trigger('click')
    await w.findAll('.post-menu-item').find((b) => b.text().includes('Закрепить')).trigger('click')
    await w.findAll('.post-menu-sub').find((o) => o.text() === 'Бессрочно').trigger('click')
    expect(api.pinPost).toHaveBeenCalledWith(1, null)

    const pinned = factory(mkPost({ author_id: 1, pinned_at: '2026-07-01T00:00:00Z' }), { userId: 1 })
    await pinned.find('.post-icon-btn').trigger('click')
    const items = pinned.findAll('.post-menu-item').map((b) => b.text())
    expect(items.some((x) => x.includes('Открепить'))).toBe(true)
    expect(items.some((x) => x.includes('Закрепить') && !x.includes('Открепить'))).toBe(false)
  })

  it('у закреплённого на срок поста — подпись «закреплено до DD.MM»', () => {
    const w = factory(mkPost({ pinned_at: '2026-07-01T00:00:00Z', pinned_until: '2026-07-15T00:00:00Z' }))
    expect(w.find('.post-pin-until').text()).toContain('закреплено до 15.07')

    const forever = factory(mkPost({ pinned_at: '2026-07-01T00:00:00Z', pinned_until: null }))
    expect(forever.find('.post-pin-until').exists()).toBe(false)
  })

})
