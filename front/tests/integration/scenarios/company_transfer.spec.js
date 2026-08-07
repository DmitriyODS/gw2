/* Перенос компании: выгрузка архивом и подъём из него новой компании.

   Архив собирает authsvc из кусков владельцев контента (задачи, реестры,
   календари, портал), поэтому здесь проверяется вся цепочка целиком: что
   уехало — то и приехало, ссылки внутри разделов переназначились, авторы
   сопоставились по логину, а чужому человеку выгрузка недоступна. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin, newMember } from '../setup/factory.js'
import * as companiesApi from '@/api/companies.js'
import * as tasksApi from '@/api/tasks.js'
import * as unitsApi from '@/api/units.js'
import * as departmentsApi from '@/api/departments.js'
import * as stagesApi from '@/api/stages.js'
import * as unitTypesApi from '@/api/unitTypes.js'
import * as registriesApi from '@/api/registries.js'
import * as portalApi from '@/api/portal.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

// Архив приходит blob-ответом: сценарию нужен File, чтобы отправить обратно.
async function archiveFile(companyId) {
  const res = await companiesApi.exportCompany(companyId)
  const blob = res instanceof Blob ? res : await res.blob()
  expect(blob.size).toBeGreaterThan(0)
  return new File([blob], 'company.gwcompany', { type: 'application/zip' })
}

describeIntegration('перенос компании: выгрузка и подъём', () => {
  it('содержимое разделов переезжает вместе со ссылками и авторами', async () => {
    const admin = await newCompanyAdmin('перенос')
    // Второй участник: его авторство должно сопоставиться по логину.
    const mate = await newMember(admin, admin.companyId, 1, 'коллега')

    admin.session.use()
    const deptName = uniq('Отдел ')
    const dept = await departmentsApi.createDepartment({ name: deptName })
    const stage = await stagesApi.createStage({ name: uniq('Этап '), color: 'blue' })
    const tagName = uniq('метка')
    const tag = await tasksApi.createTag(tagName, 'teal')
    const unitType = await unitTypesApi.createUnitType({ name: uniq('Тип ') })

    const taskName = uniq('Задача ')
    const task = await tasksApi.createTask({
      name: taskName,
      department_id: dept.id,
      stage_id: stage.id,
      responsible_user_id: mate.auth.userId,
    })
    await tasksApi.setTaskTags(task.id, [tag.id])
    await tasksApi.createTaskComment(task.id, 'первый комментарий')

    // Юнит: попадает в архив вместе со своим типом и задачей.
    const unit = await unitsApi.createUnit(task.id, { unit_type_id: unitType.id, name: 'работа' })
    await unitsApi.stopUnit(unit.id)

    // Реестр со структурой и записью.
    const registry = await registriesApi.createRegistry(uniq('Реестр '))
    const withFields = await registriesApi.replaceFields(registry.id, [
      { label: 'Клиент', type: 'text', position: 0 },
    ])
    const fieldId = String(withFields.fields[0].id)
    await registriesApi.createRecord(registry.id, { [fieldId]: 'Иванов' })

    // Портал: раздел, публикация и обсуждение.
    const topic = await portalApi.createTopic({ name: uniq('Тема '), color: 'blue' })
    const postTitle = uniq('Новость ')
    const post = await portalApi.createPost({ topicId: topic.id, title: postTitle, body: 'Тело #важно' })
    await portalApi.createComment(post.id, 'комментарий к посту')

    const file = await archiveFile(admin.companyId)
    const result = await companiesApi.importCompany(file, uniq('Копия '))
    expect(result.company_id).toBeGreaterThan(0)
    expect(result.sections.tasks).toBe(1)
    expect(result.sections.portal).toBe(1)

    // Переезжаем в новую компанию и сверяем содержимое.
    await admin.auth.switchCompany(result.company_id)

    const depts = await departmentsApi.getDepartments()
    expect(depts.map((d) => d.name)).toContain(deptName)
    const tags = await tasksApi.getTags()
    expect(tags.map((t) => t.name)).toContain(tagName)
    const types = await unitTypesApi.getUnitTypes()
    expect(types.map((t) => t.name)).toContain(unitType.name)

    const tasks = await tasksApi.getTasks({})
    const moved = (tasks.items ?? tasks).find((t) => t.name === taskName)
    expect(moved).toBeTruthy()
    // Ответственный сопоставлен по логину — это тот же человек, а не тот,
    // кто разворачивал архив.
    expect(moved.responsible_user_id).toBe(mate.auth.userId)
    // Ссылки внутри раздела переназначены на НОВЫЕ строки справочников.
    expect(moved.department_id).not.toBe(dept.id)
    expect(depts.some((d) => d.id === moved.department_id)).toBe(true)
    expect((moved.tags ?? []).map((t) => t.name)).toContain(tagName)

    const comments = await tasksApi.listTaskComments(moved.id)
    expect((comments.items ?? comments).map((c) => c.text)).toContain('первый комментарий')

    // Юнит переехал вместе с задачей и считается закрытым.
    const movedUnits = await unitsApi.getUnits(moved.id)
    expect((movedUnits.items ?? movedUnits).length).toBe(1)

    const registries = await registriesApi.getRegistries()
    const movedRegistry = (registries.registries ?? registries).find((r) => r.name === registry.name)
    expect(movedRegistry).toBeTruthy()
    const movedRegistryFull = await registriesApi.getRegistry(movedRegistry.id)
    const movedField = movedRegistryFull.fields[0]
    expect(movedField.label).toBe('Клиент')
    const records = await registriesApi.getRecords(movedRegistry.id)
    // Значения записи лежат по строковому id поля — при переносе ключи
    // переписываются на новые поля, иначе запись показалась бы пустой.
    expect((records.items ?? records)[0].data[String(movedField.id)]).toBe('Иванов')

    const feed = await portalApi.getPosts({})
    const movedPost = (feed.posts ?? feed.items ?? feed).find((p) => p.title === postTitle)
    expect(movedPost).toBeTruthy()
    expect(movedPost.topic_id).not.toBe(topic.id)
    const postComments = await portalApi.getComments(movedPost.id)
    expect(postComments.comments.map((c) => c.text)).toContain('комментарий к посту')

    // Исходная компания осталась нетронутой.
    await admin.auth.switchCompany(admin.companyId)
    const original = await tasksApi.getTasks({})
    expect((original.items ?? original).some((t) => t.name === taskName)).toBe(true)
  })

  it('выгрузка доступна только создателю компании', async () => {
    const admin = await newCompanyAdmin('права')
    const member = await newMember(admin, admin.companyId, 1, 'сотрудник')

    member.session.use()
    await expectStatus(companiesApi.exportCompany(admin.companyId), 403)

    const stranger = await registerVerified('посторонний')
    stranger.session.use()
    await expectStatus(companiesApi.exportCompany(admin.companyId), 403)
  })

  it('чужой файл архивом не считается', async () => {
    const admin = await newCompanyAdmin('битый архив')
    admin.session.use()
    const junk = new File(['это не архив'], 'junk.gwcompany', { type: 'application/zip' })
    await expectStatus(companiesApi.importCompany(junk), 400)
  })

  it('импортирующий становится создателем новой компании', async () => {
    const admin = await newCompanyAdmin('исходная')
    admin.session.use()
    await departmentsApi.createDepartment({ name: uniq('Отдел ') })
    const file = await archiveFile(admin.companyId)

    // Архив поднимает ДРУГОЙ человек: компания достаётся ему.
    const other = await registerVerified('получатель')
    other.session.use()
    const result = await companiesApi.importCompany(file, uniq('Принятая '))

    const company = await companiesApi.getCompany(result.company_id)
    expect(company.created_by).toBe(other.auth.userId)
    // Люди исходной компании не переехали: в новой пока он один.
    const members = await companiesApi.listCompanyMembers(result.company_id)
    expect((members.items ?? members).map((m) => m.id)).toEqual([other.auth.userId])
  })
})
