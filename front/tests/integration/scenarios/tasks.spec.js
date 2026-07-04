// Сценарий «задачи и юниты» через useTasksStore/useUnitsStore и api-модули
// против живого tasksvc: создание, старт/стоп юнита (1 активный), избранное,
// личный цвет (второй пользователь его не видит), запрет архива с активным юнитом.
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { newCompanyAdmin, newMember } from '../setup/factory.js'
import { useTasksStore } from '@/stores/tasks.js'
import { useUnitsStore } from '@/stores/units.js'
import { createDepartment } from '@/api/departments.js'
import { createUnitType } from '@/api/unitTypes.js'
import { createTask, toggleFavorite, setTaskColor, archiveTask, getTask } from '@/api/tasks.js'
import { createUnit, getActiveUnit } from '@/api/units.js'

async function setupCompany() {
  const admin = await newCompanyAdmin('taskadmin')
  admin.session.use()
  const dept = await createDepartment({ name: uniq('Отдел ') })
  const type = await createUnitType({ name: uniq('Разработка ') })
  return { admin, deptId: dept.id, typeId: type.id }
}

describeIntegration('tasks/units store', () => {
  it('создание задачи попадает в стор через fetchTasks', async () => {
    const { admin, deptId } = await setupCompany()
    admin.session.use()
    const store = useTasksStore()
    const task = await createTask({ name: 'Синхронизация шлюза', department_id: deptId })
    expect(task.id).toBeGreaterThan(0)

    await store.fetchTasks()
    expect(store.tasks.some((t) => t.id === task.id)).toBe(true)
    expect(store.total).toBeGreaterThanOrEqual(1)
  })

  it('один активный юнит: старт, active_unit в сторе, повторный старт → 409', async () => {
    const { admin, deptId, typeId } = await setupCompany()
    admin.session.use()
    const units = useUnitsStore()
    const task = await createTask({ name: 'Задача с юнитами', department_id: deptId })

    const unit = await createUnit(task.id, { name: 'первый', unit_type_id: typeId })
    units.startUnit(unit)
    expect(units.activeUnit?.id).toBe(unit.id)

    await units.fetchActiveUnit()
    expect(units.activeUnit?.id).toBe(unit.id)

    // Второй активный юнит запрещён.
    await expect(createUnit(task.id, { name: 'второй', unit_type_id: typeId }))
      .rejects.toMatchObject({ status: 409, error: 'ACTIVE_UNIT_EXISTS' })

    // Задачу с активным юнитом нельзя архивировать.
    await expect(archiveTask(task.id)).rejects.toMatchObject({ status: 422, error: 'HAS_ACTIVE_UNIT' })

    // Остановка юнита через стор.
    await units.stop()
    expect(units.activeUnit).toBeNull()
    const active = await getActiveUnit()
    expect(active).toBeNull()
  })

  it('избранное переключается', async () => {
    const { admin, deptId } = await setupCompany()
    admin.session.use()
    const store = useTasksStore()
    const task = await createTask({ name: 'Ревью кода', department_id: deptId })
    const on = await toggleFavorite(task.id)
    expect(on.is_favorite).toBe(true)
    store.setFavorite(task.id, true)
    const off = await toggleFavorite(task.id)
    expect(off.is_favorite).toBe(false)
  })

  it('личный цвет задачи виден только автору, второй участник его не видит', async () => {
    const { admin, deptId } = await setupCompany()
    admin.session.use()
    const task = await createTask({ name: 'Цветная задача', department_id: deptId })
    await setTaskColor(task.id, 'red')

    // Автор видит свой цвет.
    admin.session.use()
    const mine = await getTask(task.id)
    expect(mine.color).toBe('red')

    // Второй участник компании — своего цвета нет (null), чужой не подмешивается.
    const member = await newMember(admin, admin.companyId, 1, 'colormember')
    member.session.use()
    const theirs = await getTask(task.id)
    expect(theirs.color == null || theirs.color === '').toBe(true)
  })
})
