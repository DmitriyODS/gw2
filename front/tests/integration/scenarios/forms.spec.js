// Сценарий «формы и опросы» через api-модули против живого formsvc:
// конструктор с ветвлением, приём ответов, режим теста, назначение с контролем
// исполнения и публичная ссылка. Проверяем и контракт api-модуля, и правила,
// которые обязан держать сервер: маршрут по разделам, «один ответ от человека»,
// закрытость правильных ответов от отвечающего.
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { newCompanyAdmin, newMember } from '../setup/factory.js'
import * as forms from '@/api/forms.js'

// qid — id вопроса по его тексту (id выдаёт сервер при сохранении структуры).
function qid(form, title) {
  for (const section of form.sections) {
    const q = section.questions.find((x) => x.title === title)
    if (q) return String(q.id)
  }
  throw new Error('вопрос не найден: ' + title)
}

function sectionId(form, title) {
  const s = form.sections.find((x) => x.title === title)
  if (!s) throw new Error('раздел не найден: ' + title)
  return s.id
}

// branching — форма из трёх разделов: ответ уводит либо во второй, либо в третий.
async function branchingForm(title, { quiz = false } = {}) {
  const created = await forms.createForm(title, quiz)
  const saved = await forms.saveStructure(created.id, [
    {
      id: created.sections[0].id,
      title: 'Начало',
      next_action: 'next',
      next_index: -1,
      questions: [{
        id: 0,
        type: 'radio',
        title: 'Вы наш клиент?',
        required: true,
        config: { options: ['Да', 'Нет'], targets: { Да: '#1', Нет: '#2' } },
        points: quiz ? 2 : 0,
        answer_key: quiz ? { value: 'Да' } : {},
      }],
    },
    {
      id: 0,
      title: 'Для клиентов',
      next_action: 'submit',
      next_index: -1,
      questions: [{
        id: 0, type: 'short_text', title: 'Что понравилось?', required: true,
        config: {}, points: 0, answer_key: {},
      }],
    },
    {
      id: 0,
      title: 'Для остальных',
      next_action: 'submit',
      next_index: -1,
      questions: [{
        id: 0, type: 'scale', title: 'Насколько интересно?', required: true,
        config: { min: 1, max: 5 }, points: 0, answer_key: {},
      }],
    },
  ])
  return saved
}

describeIntegration('forms api', () => {
  it('конструктор с ветвлением: переходы сохраняются идентификаторами разделов', async () => {
    const admin = await newCompanyAdmin('formadmin')
    admin.session.use()

    const form = await branchingForm(uniq('Опрос '))
    expect(form.sections.length).toBe(3)

    // Клиент шлёт позиции разделов («#1»), сервер отдаёт готовые id: у нового
    // раздела id появляется только при сохранении.
    const targets = form.sections[0].questions[0].config.targets
    expect(targets['Да']).toBe(String(sectionId(form, 'Для клиентов')))
    expect(targets['Нет']).toBe(String(sectionId(form, 'Для остальных')))

    // Структура читается обратно той же формой.
    const again = await forms.getForm(form.id)
    expect(again.sections.length).toBe(3)
    expect(again.sections[1].next_action).toBe('submit')
  })

  it('обязательными считаются только вопросы пройденного маршрута', async () => {
    const admin = await newCompanyAdmin('formflow')
    admin.session.use()

    const form = await branchingForm(uniq('Маршрут '))
    await forms.updateForm(form.id, { status: 'open' })

    // Ветка «Да» не требует вопроса ветки «Нет» — человек его не видел.
    const res = await forms.submitResponse(form.id, {
      answers: { [qid(form, 'Вы наш клиент?')]: 'Да', [qid(form, 'Что понравилось?')]: 'Скорость' },
    })
    expect(res.response.id).toBeGreaterThan(0)

    // А свой обязательный вопрос ветка требует.
    await expect(forms.submitResponse(form.id, {
      answers: { [qid(form, 'Вы наш клиент?')]: 'Нет' },
    })).rejects.toMatchObject({ error: 'VALIDATION' })
  })

  it('черновик ответов не принимает, закрытая форма — тоже', async () => {
    const admin = await newCompanyAdmin('formstatus')
    admin.session.use()

    const form = await branchingForm(uniq('Статус '))
    const answers = {
      [qid(form, 'Вы наш клиент?')]: 'Да',
      [qid(form, 'Что понравилось?')]: 'Всё',
    }
    await expect(forms.submitResponse(form.id, { answers }))
      .rejects.toMatchObject({ error: 'FORM_CLOSED' })

    await forms.updateForm(form.id, { status: 'open' })
    await forms.submitResponse(form.id, { answers })

    await forms.updateForm(form.id, { status: 'closed' })
    await expect(forms.submitResponse(form.id, { answers }))
      .rejects.toMatchObject({ error: 'FORM_CLOSED' })
  })

  it('«один ответ от человека» отбивается сервером', async () => {
    const admin = await newCompanyAdmin('formonce')
    admin.session.use()

    const form = await branchingForm(uniq('Однажды '))
    await forms.updateForm(form.id, { status: 'open', one_response: true })
    const answers = {
      [qid(form, 'Вы наш клиент?')]: 'Да',
      [qid(form, 'Что понравилось?')]: 'Скорость',
    }

    await forms.submitResponse(form.id, { answers })
    await expect(forms.submitResponse(form.id, { answers }))
      .rejects.toMatchObject({ error: 'ALREADY_ANSWERED' })
  })

  it('тест: балл считает сервер, а правильные ответы не видны отвечающему', async () => {
    const admin = await newCompanyAdmin('formquiz')
    admin.session.use()

    const form = await branchingForm(uniq('Тест '), { quiz: true })
    await forms.updateForm(form.id, { status: 'open' })

    const member = await newMember(admin, admin.companyId, 1, 'quizmember')
    // newMember оставляет активной сессию новичка — назначает всё равно автор.
    admin.session.use()
    await forms.shareWith(form.id, [{ user_id: member.auth.userId, access: 'respond' }])

    member.session.use()
    // Отвечающему структура приезжает без ключей правильных ответов.
    const fill = await forms.getFill(form.id)
    const first = fill.form.sections[0].questions[0]
    expect(first.answer_key == null || Object.keys(first.answer_key).length === 0).toBe(true)

    const res = await forms.submitResponse(form.id, {
      answers: {
        [qid(form, 'Вы наш клиент?')]: 'Да',
        [qid(form, 'Что понравилось?')]: 'Поддержка',
      },
    })
    expect(res.max_score).toBe(2)
    expect(res.score).toBe(2)
    expect(res.graded).toBe(true)

    // Автор видит ответ, сводку и контроль исполнения.
    admin.session.use()
    const list = await forms.getResponses(form.id)
    expect(list.total).toBe(1)

    const summary = await forms.getSummary(form.id)
    expect(summary.total).toBe(1)
    expect(summary.quiz.max_score).toBe(2)

    const progress = await forms.getProgress(form.id)
    expect(progress.assigned).toBe(1)
    expect(progress.responded).toBe(1)
  })

  it('назначенный не видит чужих ответов, посторонний не видит формы', async () => {
    const admin = await newCompanyAdmin('formaccess')
    admin.session.use()
    const form = await branchingForm(uniq('Доступ '))

    const member = await newMember(admin, admin.companyId, 1, 'accessmember')
    admin.session.use()
    await forms.shareWith(form.id, [{ user_id: member.auth.userId, access: 'respond' }])

    member.session.use()
    // Форму назначенный видит — значит нехватку уровня называем честно.
    await forms.getForm(form.id)
    await expect(forms.getResponses(form.id)).rejects.toMatchObject({ error: 'FORBIDDEN' })

    // Постороннему существование формы не раскрываем вовсе.
    const stranger = await newCompanyAdmin('formstranger')
    stranger.session.use()
    await expect(forms.getForm(form.id)).rejects.toMatchObject({ error: 'NOT_FOUND' })
  })

  it('публичная ссылка принимает ответ гостя и отмечает его источник', async () => {
    const admin = await newCompanyAdmin('formshare')
    admin.session.use()

    const form = await branchingForm(uniq('Ссылка '))
    const share = await forms.createShare(form.id, { name: 'для сайта' })

    // Черновик по ссылке не открывается: пока форма не запущена, её содержимое
    // — дело автора.
    await expect(forms.getSharedForm(share.code)).rejects.toMatchObject({ error: 'FORM_CLOSED' })

    await forms.updateForm(form.id, { status: 'open' })
    const view = await forms.getSharedForm(share.code)
    expect(view.can_respond).toBe(true)
    expect(view.form.sections.length).toBe(3)

    const res = await forms.submitSharedResponse(share.code, {
      answers: {
        [qid(form, 'Вы наш клиент?')]: 'Нет',
        [qid(form, 'Насколько интересно?')]: 4,
      },
      name: 'Гость',
    })
    expect(res.response.share_id).toBe(share.id)

    // Журнал ссылки считает переходы и пришедшие через неё ответы.
    const shares = await forms.getShares(form.id)
    const mine = shares.shares.find((s) => s.id === share.id)
    expect(mine.responses).toBe(1)
    expect(mine.visits).toBeGreaterThan(0)
  })

  it('«Запись»: места кончаются, и остаток виден до отправки', async () => {
    const admin = await newCompanyAdmin('formbooking')
    admin.session.use()

    const created = await forms.createForm(uniq('Смены '))
    const saved = await forms.saveStructure(created.id, [{
      id: created.sections[0].id,
      title: 'Запись',
      next_action: 'submit',
      next_index: -1,
      questions: [{
        id: 0, type: 'booking', title: 'Выберите смену', required: true,
        config: { options: ['Утро', 'Вечер'], capacity: { Утро: 1, Вечер: 5 } },
        points: 0, answer_key: {},
      }],
    }])
    await forms.updateForm(saved.id, { status: 'open' })
    const shift = qid(saved, 'Выберите смену')

    await forms.submitResponse(saved.id, { answers: { [shift]: 'Утро' } })

    // Единственное место утренней смены занято — сервер отказывает следующему.
    const member = await newMember(admin, admin.companyId, 1, 'bookingmember')
    admin.session.use()
    await forms.shareWith(saved.id, [{ user_id: member.auth.userId, access: 'respond' }])

    member.session.use()
    const fill = await forms.getFill(saved.id)
    expect(fill.booking[shift]['Утро']).toBe(1)

    await expect(forms.submitResponse(saved.id, { answers: { [shift]: 'Утро' } }))
      .rejects.toMatchObject({ error: 'NO_SLOTS' })

    // На вечернюю смену места ещё есть.
    const ok = await forms.submitResponse(saved.id, { answers: { [shift]: 'Вечер' } })
    expect(ok.response.id).toBeGreaterThan(0)
  })

  it('условное отображение: скрытый вопрос не обязателен и не сохраняется', async () => {
    const admin = await newCompanyAdmin('formvisible')
    admin.session.use()

    const created = await forms.createForm(uniq('Условия '))
    // Первым проходом заводим вопрос-источник: условие ссылается на его id,
    // а он появляется только после сохранения.
    const first = await forms.saveStructure(created.id, [{
      id: created.sections[0].id,
      title: 'Анкета',
      next_action: 'submit',
      next_index: -1,
      questions: [
        {
          id: 0, type: 'radio', title: 'Есть автомобиль?', required: true,
          config: { options: ['Да', 'Нет'] }, points: 0, answer_key: {},
        },
        {
          id: 0, type: 'short_text', title: 'Госномер', required: true,
          config: {}, points: 0, answer_key: {},
        },
      ],
    }])
    const sourceId = Number(qid(first, 'Есть автомобиль?'))
    const plateId = qid(first, 'Госномер')

    const withCondition = await forms.saveStructure(first.id, [{
      id: first.sections[0].id,
      title: 'Анкета',
      next_action: 'submit',
      next_index: -1,
      questions: first.sections[0].questions.map((q) => ({
        id: q.id,
        type: q.type,
        title: q.title,
        description: q.description,
        required: q.required,
        config: q.title === 'Госномер'
          ? { visible_question_id: sourceId, visible_values: ['Да'] }
          : q.config,
        points: 0,
        answer_key: {},
      })),
    }])
    await forms.updateForm(withCondition.id, { status: 'open' })

    // «Нет» — скрытый обязательный вопрос не требуется, а его значение не
    // сохраняется даже если пришло.
    const res = await forms.submitResponse(withCondition.id, {
      answers: { [String(sourceId)]: 'Нет', [plateId]: 'А123БВ' },
    })
    expect(res.response.answers[plateId]).toBeUndefined()

    // «Да» — вопрос показан и обязателен.
    await expect(forms.submitResponse(withCondition.id, {
      answers: { [String(sourceId)]: 'Да' },
    })).rejects.toMatchObject({ error: 'VALIDATION' })
  })

  it('список форм показывает свои и назначенные областями', async () => {
    const admin = await newCompanyAdmin('formscope')
    admin.session.use()
    const form = await branchingForm(uniq('Области '))

    const member = await newMember(admin, admin.companyId, 1, 'scopemember')
    admin.session.use()
    await forms.shareWith(form.id, [{ user_id: member.auth.userId, access: 'respond' }])

    const mine = await forms.getForms('mine')
    expect(mine.forms.some((f) => f.id === form.id)).toBe(true)

    member.session.use()
    const assigned = await forms.getForms('assigned')
    const row = assigned.forms.find((f) => f.id === form.id)
    expect(row).toBeTruthy()
    expect(row.my_access).toBe('respond')
    expect(row.my_responded).toBe(false)
  })
})
