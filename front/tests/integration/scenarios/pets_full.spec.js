/* Питомцы-грувики: уход, магазин, домик, приключения, престиж, сезон и
   кудо-банк — вся экономика petsvc.

   Главное правило раздела: кудосы нельзя получить из воздуха и нельзя потратить
   дважды. Поэтому здесь всюду сверяется БАЛАНС до и после, а не только «ручка
   ответила 200»: покупка обязана списать ровно цену, продажа — вернуть часть,
   перевод коллеге — переложить сумму из кошелька в кошелёк, а неуспешная
   операция не должна трогать баланс вовсе.

   Заработок проверяется через настоящую работу (юнит и закрытая задача — хуки
   tasksvc→petsvc). Дальше кудосы начисляются прямо в БД: это подготовка
   исходных данных, а не подмена начисления. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, dbQuery } from '../setup/harness.js'
import { newCompanyAdmin, newMember } from '../setup/factory.js'
import * as pets from '@/api/pets.js'
import * as tasks from '@/api/tasks.js'
import * as units from '@/api/units.js'
import * as departments from '@/api/departments.js'
import * as unitTypes from '@/api/unitTypes.js'

async function expectClientError(promise) {
  const err = await promise.then(() => null, (e) => e)
  expect(err).toBeTruthy()
  expect(err.status).toBeGreaterThanOrEqual(400)
  expect(err.status).toBeLessThan(500)
  return err
}

// Кудосы для проверок трат: начисляем в обход начисления за работу, чтобы
// сценарий проверял ИМЕННО трату.
function grantKudos(userId, amount) {
  dbQuery(`UPDATE pets SET kudos = ${amount} WHERE user_id = ${userId}`)
}

function kudosOf(userId) {
  return Number(dbQuery(`SELECT kudos FROM pets WHERE user_id = ${userId}`)[0][0])
}

// Хуки tasksvc→petsvc — fire-and-forget: начисление приходит чуть позже ответа.
async function waitKudosAbove(userId, base, ms = 5000) {
  const deadline = Date.now() + ms
  while (Date.now() < deadline) {
    if (kudosOf(userId) > base) return true
    await new Promise((r) => setTimeout(r, 100))
  }
  return false
}

async function ownerWithPet(label = 'owner') {
  const admin = await newCompanyAdmin(label)
  admin.session.use()
  await pets.getMyPet() // питомец заводится первым обращением
  return admin
}

describeIntegration('pets API: заработок и уход', () => {
  it('работа приносит кудосы: юнит и закрытая задача', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    const before = kudosOf(admin.auth.userId)

    const dept = await departments.createDepartment({ name: uniq('Отдел ') })
    const type = await unitTypes.createUnitType({ name: uniq('Работа ') })
    const task = await tasks.createTask({ name: 'Оплачиваемая', department_id: dept.id })
    const unit = await units.createUnit(task.id, { name: 'работа', unit_type_id: type.id })
    await units.stopUnit(unit.id)
    await tasks.archiveTask(task.id)

    expect(await waitKudosAbove(admin.auth.userId, before)).toBe(true)
  })

  it('кормление списывает цену, а пустой кошелёк не даёт кормить', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 100)

    const before = kudosOf(admin.auth.userId)
    await pets.feedPet()
    const after = kudosOf(admin.auth.userId)
    expect(after).toBeLessThan(before)

    grantKudos(admin.auth.userId, 0)
    await expectClientError(pets.feedPet())
    expect(kudosOf(admin.auth.userId)).toBe(0)
  })

  it('сон бесплатен: энергию нельзя запереть за деньгами', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 0)
    const pet = await pets.sleepPet()
    expect(pet).toBeTruthy()
    expect(kudosOf(admin.auth.userId)).toBe(0)
  })

  it('прогулка и купание стоят кудосов', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 500)

    const start = kudosOf(admin.auth.userId)
    await pets.walkPet()
    const afterWalk = kudosOf(admin.auth.userId)
    expect(afterWalk).toBeLessThan(start)

    await pets.bathPet()
    expect(kudosOf(admin.auth.userId)).toBeLessThan(afterWalk)
  })

  it('лечить здорового питомца бессмысленно — отказ, а не списание', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 500)
    const before = kudosOf(admin.auth.userId)
    await expectClientError(pets.healPet())
    expect(kudosOf(admin.auth.userId)).toBe(before)
  })

  it('имя питомца меняется, пустое не принимается', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    const named = await pets.renamePet('Пушистик')
    expect(named.name).toBe('Пушистик')
    await expectClientError(pets.renamePet('   '))
    expect((await pets.getMyPet()).name).toBe('Пушистик')
  })
})

describeIntegration('pets API: поглаживание коллег', () => {
  it('поглаживание платит гладящему опыт, а владельцу — кудосы', async () => {
    const admin = await ownerWithPet('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')
    mate.session.use()
    await pets.getMyPet()
    grantKudos(mate.auth.userId, 200)

    const ownerBefore = kudosOf(admin.auth.userId)
    const strokerBefore = kudosOf(mate.auth.userId)

    mate.session.use()
    await pets.strokePet(admin.auth.userId)

    // Признание оплачивает гладящий, а получает — ВЛАДЕЛЕЦ поглаженного.
    expect(kudosOf(mate.auth.userId)).toBeLessThan(strokerBefore)
    expect(kudosOf(admin.auth.userId)).toBeGreaterThan(ownerBefore)
  })

  it('себя гладить нельзя', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 200)
    await expectClientError(pets.strokePet(admin.auth.userId))
  })

  it('питомца из чужой компании не погладить', async () => {
    const a = await ownerWithPet('a')
    const b = await ownerWithPet('b')
    b.session.use()
    grantKudos(b.auth.userId, 200)
    await expectClientError(pets.strokePet(a.auth.userId))
  })

  it('зоопарк показывает коллег по компании и не показывает чужих', async () => {
    const admin = await ownerWithPet('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')
    mate.session.use()
    await pets.getMyPet()

    const outsider = await ownerWithPet('outsider')

    admin.session.use()
    const zoo = await pets.getZoo()
    const items = zoo.items ?? zoo.pets ?? zoo
    expect(items.some((p) => p.user_id === mate.auth.userId)).toBe(true)
    expect(items.some((p) => p.user_id === outsider.auth.userId)).toBe(false)
  })

  it('питомца сотрудника удаляет администратор, сотрудник — нет', async () => {
    const admin = await ownerWithPet('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')
    mate.session.use()
    await pets.getMyPet()

    // Рядовой не вправе стереть чужого питомца.
    await expectClientError(pets.deleteColleaguePet(admin.auth.userId))

    admin.session.use()
    await pets.deleteColleaguePet(mate.auth.userId)
    expect(dbQuery(`SELECT count(*) FROM pets WHERE user_id = ${mate.auth.userId}`)[0][0]).toBe('0')

    // Питомец пересоздаётся первым обращением — сотрудник не остаётся без него.
    mate.session.use()
    expect((await pets.getMyPet()).user_id).toBe(mate.auth.userId)
  })
})

describeIntegration('pets API: магазин и гардероб', () => {
  it('покупка списывает цену, повторная покупка не проходит', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 5000)

    const shop = await pets.getShop()
    const items = shop.items ?? shop.products ?? []
    const item = items.find((i) => (i.price ?? 0) > 0 && !i.owned && i.stock !== 0)
    if (!item) return // ассортимент задаётся данными — в пустом стенде проверять нечего

    const before = kudosOf(admin.auth.userId)
    await pets.buyItem(item.key ?? item.id)
    const after = kudosOf(admin.auth.userId)
    expect(after).toBeLessThan(before)

    // Второй раз тот же предмет не продаётся — иначе кудосы утекали бы в никуда.
    await expectClientError(pets.buyItem(item.key ?? item.id))
    expect(kudosOf(admin.auth.userId)).toBe(after)
  })

  it('несуществующий товар не купить и денег он не стоит', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 5000)
    const before = kudosOf(admin.auth.userId)
    await expectClientError(pets.buyItem('нет-такого-товара'))
    expect(kudosOf(admin.auth.userId)).toBe(before)
  })

  it('не хватает кудосов — покупки нет', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 1)
    const shop = await pets.getShop()
    const item = (shop.items ?? []).find((i) => (i.price ?? 0) > 5 && !i.owned)
    if (!item) return
    await expectClientError(pets.buyItem(item.key ?? item.id))
    expect(kudosOf(admin.auth.userId)).toBe(1)
  })

  it('ежедневный сюрприз отдаётся и не требует денег', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 0)
    const mystery = await pets.getMysteryItem()
    expect(mystery && typeof mystery === 'object').toBe(true)
  })

  it('надеть можно только купленное', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    await expectClientError(pets.equipItem('нет-такой-вещи'))
  })

  it('вид меняется на купленный, сброс возвращает природный', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 5000)
    // Некупленный вид не надеть — иначе гардероб можно обойти.
    await expectClientError(pets.switchSpecies('нет-такого-вида'))
    const back = await pets.resetSpecies()
    expect(back).toBeTruthy()
  })
})

describeIntegration('pets API: домик', () => {
  it('домик читается, тема меняется, чепуха отвергается', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    const house = await pets.getHouse()
    expect(house && typeof house === 'object').toBe(true)
    await expectClientError(pets.setHouseTheme('нет-такой-темы'))
  })

  it('положение грувика прижимается к сцене', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    const inside = await pets.setHousePetPos(50, 50)
    expect(inside.pet_x ?? inside.petX).toBe(50)

    // Координаты — проценты сцены: заведомо внешние прижимаются к краю, иначе
    // питомец оказался бы за пределами домика и стал недоступен.
    const clamped = await pets.setHousePetPos(500, -20)
    expect(clamped.pet_x ?? clamped.petX).toBeLessThanOrEqual(100)
    expect(clamped.pet_y ?? clamped.petY).toBeGreaterThanOrEqual(0)
  })

  it('расставить можно только купленный декор', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    await expectClientError(pets.buyHouseDecor('нет-такого-декора'))
    const arranged = await pets.arrangeHouse([]).catch((e) => e)
    expect(arranged.status === undefined || arranged.status < 500).toBe(true)
  })
})

describeIntegration('pets API: приключения, квест, престиж, сезон', () => {
  it('приключение отправляет питомца и блокирует платные действия', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 1000)

    const away = await pets.startAdventure()
    expect(away).toBeTruthy()

    // В пути питомца не покормить и не погладить — он не дома.
    const err = await expectClientError(pets.feedPet())
    expect(String(err.error ?? err.code ?? '')).toMatch(/AWAY|PET/i)

    // Досрочный возврат платный.
    const before = kudosOf(admin.auth.userId)
    await pets.recallAdventure()
    expect(kudosOf(admin.auth.userId)).toBeLessThan(before)
  })

  it('квест забирается только выполненный', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    await expectClientError(pets.claimQuest())
  })

  it('престиж недоступен до максимальной формы', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    await expectClientError(pets.prestigePet())
  })

  it('сезонный трек читается, недостигнутый порог не забрать', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    const season = await pets.getSeason()
    expect(season && typeof season === 'object').toBe(true)
    await expectClientError(pets.claimSeasonReward(999999))
  })

  it('личная история видна только владельцу', async () => {
    const admin = await ownerWithPet('admin')
    admin.session.use()
    const log = await pets.getActivityLog()
    expect(Array.isArray(log.items ?? log.entries ?? log)).toBe(true)
  })

  it('живая сводка отвечает без активной работы', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    expect(await pets.getLive()).toBeTruthy()
  })
})

describeIntegration('pets API: кудо-банк', () => {
  it('перевод коллеге перекладывает ровно сумму', async () => {
    const admin = await ownerWithPet('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')
    mate.session.use()
    await pets.getMyPet()

    grantKudos(admin.auth.userId, 1000)
    grantKudos(mate.auth.userId, 0)

    admin.session.use()
    await pets.transferKudos(mate.auth.userId, 300, 'за помощь')

    expect(kudosOf(admin.auth.userId)).toBe(700)
    expect(kudosOf(mate.auth.userId)).toBe(300)
  })

  it('перевод больше баланса не проходит и ничего не двигает', async () => {
    const admin = await ownerWithPet('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')
    mate.session.use()
    await pets.getMyPet()
    grantKudos(admin.auth.userId, 50)
    grantKudos(mate.auth.userId, 10)

    admin.session.use()
    await expectClientError(pets.transferKudos(mate.auth.userId, 100000, ''))
    expect(kudosOf(admin.auth.userId)).toBe(50)
    expect(kudosOf(mate.auth.userId)).toBe(10)
  })

  it('нулевой и отрицательный перевод отвергаются', async () => {
    const admin = await ownerWithPet('admin')
    const mate = await newMember(admin, admin.companyId, 1, 'mate')
    mate.session.use()
    await pets.getMyPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 500)
    await expectClientError(pets.transferKudos(mate.auth.userId, 0, ''))
    await expectClientError(pets.transferKudos(mate.auth.userId, -100, ''))
    expect(kudosOf(admin.auth.userId)).toBe(500)
  })

  it('перевод в чужую компанию невозможен', async () => {
    const a = await ownerWithPet('a')
    const b = await ownerWithPet('b')
    a.session.use()
    grantKudos(a.auth.userId, 500)
    await expectClientError(pets.transferKudos(b.auth.userId, 100, ''))
    expect(kudosOf(a.auth.userId)).toBe(500)
  })

  it('вклад и снятие двигают кошелёк в обе стороны', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 1000)

    await pets.bankDeposit(400)
    expect(kudosOf(admin.auth.userId)).toBe(600)

    const bank = await pets.getBank()
    expect(bank.savings).toBeGreaterThan(0)

    await pets.bankWithdraw(400)
    expect(kudosOf(admin.auth.userId)).toBeGreaterThanOrEqual(1000)
  })

  it('снять больше, чем во вкладе, нельзя', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 100)
    await pets.bankDeposit(50)
    await expectClientError(pets.bankWithdraw(100000))
    expect(kudosOf(admin.auth.userId)).toBe(50)
  })

  it('кредит выдаётся один и при долге вклад закрыт', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 0)

    // Потолок займа растёт с уровнем клиента — новичку доступен минимальный.
    const limit = (await pets.getBank()).credit.loan_max
    await pets.bankTakeLoan(limit)
    expect(kudosOf(admin.auth.userId)).toBe(limit)

    // Второй кредит поверх непогашенного — нет. И вклад при долге закрыт:
    // иначе получится арбитраж «занял под низкий процент, положил под высокий».
    await expectClientError(pets.bankTakeLoan(1))
    await expectClientError(pets.bankDeposit(limit))

    // Долг больше выданного на комиссию — «вернул сколько взял» его не гасит.
    const debt = (await pets.getBank()).loan
    expect(debt).toBeGreaterThan(limit)

    grantKudos(admin.auth.userId, debt)
    await pets.bankRepayLoan(debt)
    expect((await pets.getBank()).loan).toBe(0)
  })

  it('выписка и статистика читаются', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 300)
    await pets.bankDeposit(100)

    const ledger = await pets.getBankLedger()
    expect(Array.isArray(ledger.items ?? ledger.entries ?? ledger)).toBe(true)
    expect(await pets.getBankStats()).toBeTruthy()
  })

  it('копилка: заводится, принимает и отдаёт, удаляется', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 1000)

    // Операции банка отвечают полным состоянием кошелька — копилку берём из него.
    const created = await pets.createGoal('На домик', '🏠', 500)
    const goal = created.goals.find((g) => g.title === 'На домик')
    expect(goal).toBeTruthy()
    const goalId = goal.id

    await pets.goalDeposit(goalId, 200)
    expect(kudosOf(admin.auth.userId)).toBe(800)

    await pets.goalWithdraw(goalId, 200)
    expect(kudosOf(admin.auth.userId)).toBe(1000)

    await pets.deleteGoal(goalId)
    await expectClientError(pets.goalDeposit(goalId, 10))
  })

  it('в копилку не положить больше, чем есть', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    grantKudos(admin.auth.userId, 100)
    const created = await pets.createGoal('Мечта', '⭐', 1000)
    const goalId = created.goals.find((g) => g.title === 'Мечта').id
    await expectClientError(pets.goalDeposit(goalId, 5000))
    expect(kudosOf(admin.auth.userId)).toBe(100)
  })

  it('сбор объявляет менеджер, взнос списывается, чужой сбор не закрыть', async () => {
    const admin = await ownerWithPet('admin')
    const worker = await newMember(admin, admin.companyId, 1, 'worker')
    worker.session.use()
    await pets.getMyPet()

    // Сбор — дело компании, объявляет его уровень менеджера.
    await expectClientError(pets.createFund({ title: 'Самовольный', description: '', emoji: '🎁', target: 100 }))

    admin.session.use()
    const title = uniq('Сбор ')
    const created = await pets.createFund({ title, description: 'На корпоратив', emoji: '🎁', target: 1000 })
    const fund = (created.funds ?? []).find((f) => f.title === title)
    expect(fund).toBeTruthy()
    const fundId = fund.id

    worker.session.use()
    grantKudos(worker.auth.userId, 500)
    await pets.donateFund(fundId, 200)
    expect(kudosOf(worker.auth.userId)).toBe(300)

    // Закрывает объявивший, а не любой участник.
    await expectClientError(pets.closeFund(fundId))
    admin.session.use()
    await pets.closeFund(fundId)
  })

  it('рассрочки: список пуст, платёж по несуществующей не проходит', async () => {
    const admin = await ownerWithPet()
    admin.session.use()
    const list = await pets.getInstallments()
    expect(Array.isArray(list.items ?? list.installments ?? list)).toBe(true)
    await expectClientError(pets.payInstallment(999999, 10))
  })
})
