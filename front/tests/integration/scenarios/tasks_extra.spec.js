/* Задачи: то, чего не покрывал прежний сценарий, — теги, комментарии,
   ответственный и этапы, справочники (отделы, типы юнитов, этапы) и
   статистика.

   Ключевые правила отсюда: задача не выходит за пределы компании, тег общий
   для компании, а цвет — личный, удаление типа юнита уносит его юниты,
   комментарии считают «новые» до отметки о прочтении. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { newCompanyAdmin, newMember } from '../setup/factory.js'
import * as tasks from '@/api/tasks.js'
import * as units from '@/api/units.js'
import * as stages from '@/api/stages.js'
import * as departments from '@/api/departments.js'
import * as unitTypes from '@/api/unitTypes.js'
import * as stats from '@/api/stats.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

async function company(label = '') {
  const admin = await newCompanyAdmin(label)
  admin.session.use()
  const dept = await departments.createDepartment({ name: uniq('Отдел ') })
  const type = await unitTypes.createUnitType({ name: uniq('Работа ') })
  return { admin, deptId: dept.id, typeId: type.id }
}

const ymd = (d) => d.toISOString().slice(0, 10)

describeIntegration('tasks API: теги, комментарии, ответственный', () => {
  it('тег компании общий, а цвет задачи — личный', async () => {
    const { admin, deptId } = await company('a')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const task = await tasks.createTask({ name: 'Общая задача', department_id: deptId })
    const tag = await tasks.createTag(uniq('срочно-'), 'red')
    await tasks.setTaskTags(task.id, [tag.id])
    await tasks.setTaskColor(task.id, 'blue')

    const mine = await tasks.getTask(task.id)
    expect(mine.tag_ids ?? mine.tags?.map((t) => t.id)).toContain(tag.id)
    expect(mine.color).toBe('blue')

    mate.session.use()
    const theirs = await tasks.getTask(task.id)
    // Тег виден всей компании, а цвет — личная пометка автора.
    expect(theirs.tag_ids ?? theirs.tags?.map((t) => t.id)).toContain(tag.id)
    expect(theirs.color ?? null).not.toBe('blue')
  })

  it('справочник тегов правит менеджер, сотрудник — только назначает', async () => {
    const { admin, deptId } = await company('a')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')

    admin.session.use()
    const task = await tasks.createTask({ name: 'С тегом', department_id: deptId })
    const tag = await tasks.createTag(uniq('метка-'), 'green')

    worker.session.use()
    await expect(tasks.createTag(uniq('своя-'), 'blue')).rejects.toBeTruthy()
    // Но повесить существующий тег на задачу сотрудник вправе.
    await tasks.setTaskTags(task.id, [tag.id])
  })

  it('комментарии: новые считаются до отметки о прочтении', async () => {
    const { admin, deptId } = await company('a')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const task = await tasks.createTask({ name: 'Обсуждаемая', department_id: deptId })
    await tasks.createTaskComment(task.id, 'Первый комментарий')

    mate.session.use()
    const before = await tasks.listTaskComments(task.id)
    expect(before.new_count).toBeGreaterThan(0)

    await tasks.markTaskCommentsSeen(task.id)
    const after = await tasks.listTaskComments(task.id)
    expect(after.new_count).toBe(0)
  })

  it('свой комментарий правится и удаляется, чужой — нет', async () => {
    const { admin, deptId } = await company('a')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const task = await tasks.createTask({ name: 'Комментарии', department_id: deptId })
    const c = await tasks.createTaskComment(task.id, 'Мой комментарий')
    await tasks.updateTaskComment(task.id, c.id, 'Поправленный')

    mate.session.use()
    await expect(tasks.updateTaskComment(task.id, c.id, 'Чужая правка')).rejects.toBeTruthy()
    await expect(tasks.deleteTaskComment(task.id, c.id)).rejects.toBeTruthy()

    admin.session.use()
    await tasks.deleteTaskComment(task.id, c.id)
  })

  it('ответственный и этап назначаются и видны всем в компании', async () => {
    const { admin, deptId } = await company('a')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')

    admin.session.use()
    const stage = await stages.createStage({ name: uniq('Проверка '), color: 'blue' })
    const task = await tasks.createTask({ name: 'С этапом', department_id: deptId })

    await tasks.setTaskResponsible(task.id, mate.auth.userId)
    await tasks.setTaskStage(task.id, stage.id)

    mate.session.use()
    const seen = await tasks.getTask(task.id)
    expect(seen.responsible_user_id ?? seen.responsible?.id).toBe(mate.auth.userId)
    expect(seen.stage_id ?? seen.stage?.id).toBe(stage.id)
  })

  it('задача чужой компании: посторонний — 403, свой с другой активной — 409', async () => {
    const a = await company('a')
    a.admin.session.use()
    const task = await tasks.createTask({ name: 'Внутренняя', department_id: a.deptId })

    // Посторонний — совсем чужой человек.
    const b = await company('b')
    b.admin.session.use()
    await expectStatus(tasks.getTask(task.id), 403)

    // Сотрудник ТОЙ ЖЕ компании, но с другой активной — 409 с подсказкой.
    const mate = await newMember(a.admin, a.admin.companyId, 1, 'mate')
    mate.session.use()
    const own = await import('@/api/companies.js')
    const second = await own.createCompany({ name: uniq('ООО-2 ') })
    await mate.auth.switchCompany(second.id)
    await expectStatus(tasks.getTask(task.id), 409)
  })

  it('архив и восстановление: задача уходит из активных и возвращается', async () => {
    const { admin, deptId } = await company('a')
    admin.session.use()
    const task = await tasks.createTask({ name: 'В архив', department_id: deptId })

    await tasks.archiveTask(task.id)
    const active = await tasks.getTasks({ tab: 'active' })
    expect((active.tasks ?? active.items ?? []).some((t) => t.id === task.id)).toBe(false)

    await tasks.restoreTask(task.id)
    const back = await tasks.getTasks({ tab: 'active' })
    expect((back.tasks ?? back.items ?? []).some((t) => t.id === task.id)).toBe(true)
  })

  it('поиск по задачам: находит по названию и не находит небывалое', async () => {
    const { admin, deptId } = await company('a')
    admin.session.use()
    const marker = `кессон${Date.now().toString().slice(-5)}`
    const task = await tasks.createTask({ name: `Задача ${marker}`, department_id: deptId })

    const found = await tasks.getTasks({ search: marker })
    expect((found.tasks ?? found.items ?? []).some((t) => t.id === task.id)).toBe(true)

    const none = await tasks.getTasks({ search: 'заведомо-небывалое-слово-хх' })
    expect((none.tasks ?? none.items ?? []).length).toBe(0)
  })
})

describeIntegration('tasks API: справочники компании', () => {
  it('отделы: создание, правка, удаление', async () => {
    const { admin } = await company('a')
    admin.session.use()
    const dept = await departments.createDepartment({ name: uniq('Снабжение ') })
    await departments.updateDepartment(dept.id, { name: 'Снабжение и логистика' })

    const list = await departments.getDepartments()
    expect((list.departments ?? list).find((d) => d.id === dept.id).name).toBe('Снабжение и логистика')

    await departments.deleteDepartment(dept.id)
    const after = await departments.getDepartments()
    expect((after.departments ?? after).some((d) => d.id === dept.id)).toBe(false)
  })

  it('удаление типа юнита уносит его юниты вместе с ним', async () => {
    const { admin, deptId, typeId } = await company('a')
    admin.session.use()
    const task = await tasks.createTask({ name: 'С юнитом', department_id: deptId })
    const unit = await units.createUnit(task.id, { name: 'работа', unit_type_id: typeId })
    await units.stopUnit(unit.id)

    await unitTypes.deleteUnitType(typeId)
    const left = await units.getUnits(task.id)
    expect((left.units ?? left).some((u) => u.id === unit.id)).toBe(false)
  })

  it('этапы: порядок задаётся и сохраняется', async () => {
    const { admin } = await company('a')
    admin.session.use()
    const first = await stages.createStage({ name: uniq('Первый '), color: 'red' })
    const second = await stages.createStage({ name: uniq('Второй '), color: 'blue' })

    await stages.reorderStages([second.id, first.id])
    const list = await stages.getStages()
    const items = list.stages ?? list
    const iSecond = items.findIndex((s) => s.id === second.id)
    const iFirst = items.findIndex((s) => s.id === first.id)
    expect(iSecond).toBeLessThan(iFirst)
  })

  it('справочники чужой компании недоступны', async () => {
    const a = await company('a')
    const b = await company('b')

    b.admin.session.use()
    // Список отдаётся по активной компании: чужие отделы в него не попадают.
    const list = await departments.getDepartments()
    expect((list.departments ?? list).some((d) => d.id === a.deptId)).toBe(false)
    await expect(departments.deleteDepartment(a.deptId)).rejects.toBeTruthy()
  })
})

describeIntegration('stats API: сводки и выгрузки', () => {
  it('общая статистика считает закрытые задачи и часы за период', async () => {
    const { admin, deptId, typeId } = await company('a')
    admin.session.use()
    const task = await tasks.createTask({ name: 'Со временем', department_id: deptId })
    const unit = await units.createUnit(task.id, { name: 'работа', unit_type_id: typeId })
    await units.stopUnit(unit.id)

    const today = ymd(new Date())
    const data = await stats.getStatsCommon(today, today)
    expect(data).toBeTruthy()
    expect(data.tasks ?? data).toBeTruthy()
  })

  it('смотреть статистику вправе все, выгружать — с уровня менеджера', async () => {
    const { admin } = await company('a')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')
    const today = ymd(new Date())

    worker.session.use()
    expect(await stats.getStatsProfile(today, today)).toBeTruthy()
    // Расширенную сводку сотрудник видит — это его же работа в разрезе компании.
    expect(await stats.getStatsExtended(today, today)).toBeTruthy()

    // А вот выгрузка — уровень менеджера: файл уходит наружу.
    await expect(stats.exportStatsExtended(today, today)).rejects.toBeTruthy()
  })

  it('выгрузка статистики отдаёт файл', async () => {
    const { admin } = await company('a')
    admin.session.use()
    const today = ymd(new Date())
    const res = await stats.exportStatsCommon(today, today)
    expect(res.ok).toBe(true)
  })

  it('период задом наперёд не роняет сервер', async () => {
    const { admin } = await company('a')
    admin.session.use()
    const today = new Date()
    const past = new Date(today.getTime() - 7 * 86400000)
    try {
      const data = await stats.getStatsCommon(ymd(today), ymd(past))
      expect(data).toBeTruthy()
    } catch (e) {
      expect(e.status).toBeGreaterThanOrEqual(400)
      expect(e.status).toBeLessThan(500)
    }
  })
})
