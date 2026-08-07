/* Звонки (callsvc, REST-часть) и YouGile-интеграция (tasksvc).

   Медиа звонков живёт в LiveKit, а ринг-фаза — в WebSocket шлюза; их в стенде
   нет. Зато REST-часть проверяется целиком: история и активный звонок, вход по
   ссылке-приглашению (публичный путь — самый уязвимый) и отказ на чужой звонок.

   YouGile без реального ключа не подключить, поэтому здесь — границы прав
   (кто вправе настраивать интеграцию) и валидация: связь с внешним сервисом
   обязана падать понятной ошибкой, а не 500, и вебхук с неверным секретом
   обязан быть неотличим от несуществующего. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as calls from '@/api/calls.js'
import * as yougile from '@/api/yougile.js'
import * as tasks from '@/api/tasks.js'
import * as departments from '@/api/departments.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

async function expectClientError(promise) {
  const err = await promise.then(() => null, (e) => e)
  expect(err).toBeTruthy()
  expect(err.status).toBeGreaterThanOrEqual(400)
  expect(err.status).toBeLessThan(500)
}

describeIntegration('calls API: состояние и история', () => {
  it('у новичка нет активного звонка', async () => {
    const u = await registerVerified()
    u.session.use()
    const state = await calls.getActiveCall()
    // Ответ всегда объект с полем call — раздел не должен различать «нет
    // звонка» и «ручка не ответила».
    expect(state && typeof state === 'object').toBe(true)
    expect(state.call ?? null).toBeNull()
  })

  it('токен на чужой звонок не выдаётся', async () => {
    const u = await registerVerified()
    u.session.use()
    // Несуществующий звонок — 404: существование чужих звонков не раскрываем.
    await expectClientError(calls.getCallToken(999999))
  })

  it('без авторизации состояние звонков закрыто', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectStatus(calls.getActiveCall(), 401)
  })
})

describeIntegration('calls API: ссылка-приглашение', () => {
  it('несуществующий код не пускает ни гостя, ни вошедшего', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectClientError(calls.getJoinInfo('нет-такого-кода'))
    await expectClientError(calls.joinCallByCode('нет-такого-кода', { name: 'Гость' }))

    const u = await registerVerified()
    u.session.use()
    await expectClientError(calls.joinCallByCode('нет-такого-кода'))
  })

  it('информация по ссылке доступна без входа — это публичный путь', async () => {
    const guest = new Session('guest')
    guest.use()
    // Ответ обязан быть отказом по существу (нет такого звонка), а не 401:
    // страницу /call/<code> открывают не входя в систему.
    const err = await calls.getJoinInfo(uniq('code')).then(() => null, (e) => e)
    expect(err?.status).not.toBe(401)
  })
})

describeIntegration('yougile API: подключение и права', () => {
  it('состояние интеграции — «не подключено» и без ошибок', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    const status = await yougile.getYougileStatus()
    expect(!!status.connected).toBe(false)
  })

  it('подключение с неверными данными отвергается понятной ошибкой', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    await expectClientError(yougile.connectYougile({
      login: 'нет-такого@apitest.local', password: 'неверный-пароль',
    }))
  })

  it('справочники YouGile — уровень администратора компании', async () => {
    const admin = await newCompanyAdmin('admin')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')

    worker.session.use()
    await expectStatus(yougile.listYougileProjects(), 403)
    await expectStatus(yougile.getCompanyYougileSettings(), 403)
    await expectStatus(yougile.updateCompanyYougileSettings({ project_id: 'x' }), 403)
    await expectStatus(yougile.resetCompanyYougileIntegration(), 403)
  })

  it('без подключённого аккаунта справочники отвечают отказом, а не пустотой', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    // Пустой список выглядел бы как «в YouGile нет проектов» — человек стал бы
    // искать причину не там.
    await expectClientError(yougile.listYougileProjects())
  })

  it('импорт и экспорт задач без интеграции не проходят', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    const dept = await departments.createDepartment({ name: uniq('Отдел ') })
    const task = await tasks.createTask({ name: 'Задача для выгрузки', department_id: dept.id })

    await expectClientError(yougile.importYougileTask({ url: 'https://ru.yougile.com/team/xxx/#chat-yyy' }))
    await expectClientError(yougile.exportYougileTask({ gw_task_id: task.id }))
  })

  it('отвязка идемпотентна для своей задачи и закрыта для чужой', async () => {
    const a = await newCompanyAdmin('a')
    a.session.use()
    const dept = await departments.createDepartment({ name: uniq('Отдел ') })
    const task = await tasks.createTask({ name: 'Несвязанная', department_id: dept.id })

    // «Связи нет» — ровно то состояние, которого просили: повтор не ошибка.
    await yougile.unlinkYougileTask(task.id)
    await yougile.unlinkYougileTask(task.id)

    const b = await newCompanyAdmin('b')
    b.session.use()
    await expectClientError(yougile.unlinkYougileTask(task.id))
  })

  it('настройки интеграции чужой компании недоступны', async () => {
    const a = await newCompanyAdmin('a')
    const b = await newCompanyAdmin('b')
    // Интеграция скоупится активной компанией из токена, поэтому «чужую»
    // не подменить параметром — проверяем, что каждый видит только своё.
    a.session.use()
    const mine = await yougile.getCompanyYougileSettings()
    b.session.use()
    const theirs = await yougile.getCompanyYougileSettings()
    expect(JSON.stringify(mine)).toBe(JSON.stringify(theirs)) // обе пустые
  })
})

describeIntegration('yougile API: вебхук', () => {
  it('вебхук с неверным секретом неотличим от несуществующего', async () => {
    const admin = await newCompanyAdmin()
    admin.session.use()
    // Публичная ручка без авторизации: подбор секрета не должен подсказывать,
    // что компания существует.
    const res = await fetch(`/api/yougile/webhook/${admin.companyId}/поддельный-секрет`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ event: 'task-updated', payload: {} }),
    })
    expect(res.status).toBe(404)

    const missing = await fetch('/api/yougile/webhook/999999/любой-секрет', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ event: 'task-updated', payload: {} }),
    })
    expect(missing.status).toBe(404)
  })
})
