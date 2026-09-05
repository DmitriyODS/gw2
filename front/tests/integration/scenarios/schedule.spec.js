// Сценарий «расписание» через useSchedulesStore против живого schedulesvc:
// цикл недель, занятия и их повтор, укорочение цикла, шаринг на чтение,
// живая плитка и перенос расписания файлом (включая формат прототипа).
import { it, expect } from 'vitest'
import { describeIntegration, uniq } from '../setup/harness.js'
import { registerVerified } from '../setup/factory.js'
import { useSchedulesStore } from '@/stores/schedules.js'
import * as scheduleApi from '@/api/schedules.js'
import { dateKey, itemsOfDay, mondayOf, weekIndex } from '@/utils/scheduleCycle.js'

describeIntegration('schedules store: цикл и занятия', () => {
  it('создание расписания, занятия и отбор по неделям цикла', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useSchedulesStore()

    await store.fetchSchedules()
    expect(store.schedules.length).toBe(0)

    const schedule = await store.createSchedule({ name: uniq('Учёба '), cycle_weeks: 2 })
    expect(schedule.id).toBeGreaterThan(0)
    // Якорь цикла — всегда понедельник: цикл считается неделями.
    expect(dateKey(schedule.cycle_anchor)).toBe(dateKey(mondayOf(new Date())))

    const every = await store.createItem({
      weekday: 0, start_min: 620, end_min: 715, title: 'Вейвлеты', short: 'Вейвлеты', weeks: [],
    })
    const numerator = await store.createItem({
      weekday: 0, start_min: 500, end_min: 560, title: 'Физика', weeks: [1],
    })
    expect(store.items.length).toBe(2)

    const first = mondayOf(new Date())
    const second = new Date(first.getTime() + 7 * 24 * 3600 * 1000)
    expect(weekIndex(schedule.cycle_anchor, first, 2)).toBe(1)

    const week1 = itemsOfDay(store.selected, store.items, first).map((i) => i.id)
    const week2 = itemsOfDay(store.selected, store.items, second).map((i) => i.id)
    expect(week1).toEqual([numerator.id, every.id])
    expect(week2).toEqual([every.id])
  })

  it('своё правило повтора отменяет номера недель', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useSchedulesStore()
    await store.createSchedule({ name: uniq('Работа '), cycle_weeks: 2 })

    const item = await store.createItem({
      weekday: 2, start_min: 600, end_min: 700, title: 'Планёрка',
      weeks: [1], repeat_every: 2, repeat_from: dateKey(mondayOf(new Date())),
    })
    // Сервер оставляет ОДНО описание повтора: два противоречили бы друг другу.
    expect(item.weeks).toEqual([])
    expect(item.repeat_every).toBe(2)
    expect(item.repeat_from).toBe(dateKey(mondayOf(new Date())))
  })

  it('укорочение цикла не роняет занятия со шкалы', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useSchedulesStore()
    const schedule = await store.createSchedule({ name: uniq('Цикл '), cycle_weeks: 4 })

    await store.createItem({ weekday: 1, start_min: 600, end_min: 700, title: 'Раз в месяц', weeks: [3, 4] })
    await store.createItem({ weekday: 1, start_min: 720, end_min: 800, title: 'Через неделю', weeks: [1, 3] })

    const data = await store.updateSchedule(schedule.id, { cycle_weeks: 2 })
    // Сервер честно сообщает, скольких занятий это коснулось.
    expect(data.adjusted_items).toBe(2)
    expect(store.selected.cycle_weeks).toBe(2)

    const monthly = store.items.find((i) => i.title === 'Раз в месяц')
    const biweekly = store.items.find((i) => i.title === 'Через неделю')
    // Недель за пределами цикла не осталось → занятие стало еженедельным.
    expect(monthly.weeks).toEqual([])
    expect(biweekly.weeks).toEqual([1])
  })

  it('категории и поля живут у конкретного расписания', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useSchedulesStore()
    await store.createSchedule({ name: uniq('Поля ') })

    const category = await store.createCategory({ name: 'Лекция', color: 'blue' })
    expect(category.id).toBeGreaterThan(0)

    await store.saveFields([{ id: 0, label: 'Аудитория', type: 'text', show_on_block: true, show_in_card: true }])
    const field = store.selected.fields[0]
    expect(field.label).toBe('Аудитория')

    const item = await store.createItem({
      weekday: 0, start_min: 600, end_min: 700, title: 'Матан',
      category_id: category.id, data: { [String(field.id)]: '2-3.10' },
    })
    expect(item.category_id).toBe(category.id)
    expect(item.data[String(field.id)]).toBe('2-3.10')

    // Удаление категории занятие не уносит — оно лишь теряет цвет.
    await store.removeCategory(category.id)
    await store.reload()
    expect(store.items.find((i) => i.id === item.id).category_id).toBeNull()
  })
})

describeIntegration('schedules: шаринг только на чтение', () => {
  it('адресат видит расписание, но не может его править', async () => {
    const owner = await registerVerified()
    const guest = await registerVerified()

    owner.session.use()
    const store = useSchedulesStore()
    const schedule = await store.createSchedule({ name: uniq('Общее ') })
    await store.createItem({ weekday: 0, start_min: 600, end_min: 700, title: 'Пара' })
    await scheduleApi.shareWith(schedule.id, { user_id: guest.auth.userId })

    guest.session.use()
    const guestStore = useSchedulesStore()
    guestStore.reset()
    await guestStore.setTab('shared')
    expect(guestStore.schedules.some((s) => s.id === schedule.id)).toBe(true)

    await guestStore.select(schedule.id)
    expect(guestStore.canEdit).toBe(false)
    expect(guestStore.items.length).toBe(1)

    // Правка адресатом отклоняется явно: он расписание видит, и «не найдено»
    // было бы неправдой.
    await expect(scheduleApi.createItem(schedule.id, {
      weekday: 1, start_min: 600, end_min: 700, title: 'Своё',
    })).rejects.toMatchObject({ error: 'FORBIDDEN' })
  })

  it('постороннему расписание не существует', async () => {
    const owner = await registerVerified()
    const stranger = await registerVerified()

    owner.session.use()
    const store = useSchedulesStore()
    store.reset()
    const schedule = await store.createSchedule({ name: uniq('Личное ') })

    stranger.session.use()
    await expect(scheduleApi.getSchedule(schedule.id)).rejects.toMatchObject({ status: 404 })
  })

  it('публичная ссылка открывает расписание без входа', async () => {
    const owner = await registerVerified()
    owner.session.use()
    const store = useSchedulesStore()
    store.reset()
    const schedule = await store.createSchedule({ name: uniq('Ссылка ') })
    await store.createItem({ weekday: 3, start_min: 540, end_min: 620, title: 'Открытая пара' })

    const share = await scheduleApi.createShare(schedule.id)
    expect(share.code).toBeTruthy()

    const view = await scheduleApi.getSharedSchedule(share.code)
    expect(view.can_edit).toBe(false)
    expect(view.items.map((i) => i.title)).toContain('Открытая пара')
  })
})

describeIntegration('schedules: плитка и перенос', () => {
  it('повестка считает занятость, окна и ближайшее занятие', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useSchedulesStore()
    store.reset()
    await store.createSchedule({ name: uniq('Плитка ') })

    const today = new Date()
    const weekday = (today.getDay() + 6) % 7
    await store.createItem({ weekday, start_min: 600, end_min: 700, title: 'Первая' })
    await store.createItem({ weekday, start_min: 760, end_min: 820, title: 'Вторая' })

    const agenda = await scheduleApi.getAgenda(dateKey(today), 650)
    expect(agenda.total).toBe(2)
    expect(agenda.now?.title).toBe('Первая')
    expect(agenda.next?.title).toBe('Вторая')
    expect(agenda.busy_min).toBe(160)
    expect(agenda.gap_min).toBe(60)
  })

  it('расписание прототипа заезжает файлом и разворачивается в цикл из двух недель', async () => {
    const u = await registerVerified()
    u.session.use()
    const store = useSchedulesStore()
    store.reset()

    const legacy = JSON.stringify({
      items: [
        {
          day: 0, category: 'study', week: 'num', start: '10:20', end: '11:55',
          title: 'Вейвлет-преобразования', short: 'Вейвлеты', kind: 'лекция', place: '2-3.10',
        },
        { day: 2, category: 'work', week: 'both', start: '14:00', end: '18:00', title: 'Работа' },
      ],
    })
    const file = new Blob([legacy], { type: 'application/json' })
    const view = await scheduleApi.importNew(file, uniq('Прототип '))

    expect(view.schedule.cycle_weeks).toBe(2)
    expect(view.schedule.week_labels).toEqual(['Числитель', 'Знаменатель'])
    expect(view.schedule.categories.map((c) => c.name)).toEqual(['Учёба', 'Работа'])
    expect(view.schedule.fields.map((f) => f.label)).toEqual(['Вид', 'Место'])
    expect(view.items.length).toBe(2)

    const first = view.items.find((i) => i.title === 'Вейвлет-преобразования')
    expect(first.start_min).toBe(620)
    expect(first.weeks).toEqual([1])
    expect(Object.values(first.data)).toContain('2-3.10')

    // Свой формат — обратная операция: выгрузили и вернули как было.
    const resp = await scheduleApi.exportSchedule(view.schedule.id, 'json')
    const raw = await resp.text()
    const copy = await scheduleApi.importNew(new Blob([raw], { type: 'application/json' }), uniq('Копия '))
    expect(copy.items.length).toBe(2)
    expect(copy.schedule.cycle_weeks).toBe(2)
  })
})
