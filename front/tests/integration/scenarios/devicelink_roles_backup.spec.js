/* Спаривание устройств (вход по QR и ТВ-киоск), справочник ролей и резервная
   копия платформы — три оставшиеся ручки authsvc.

   Спаривание — самый лакомый для чужих рук путь: код виден на экране, поэтому
   сессию отдаёт только знающий секрет инициатор и ровно один раз. Бэкап —
   инструмент супер-админа, и его импорт ДЕСТРУКТИВЕН, поэтому здесь же
   проверяется, что чужие руки к нему не дотянутся.

   Импорт архива в этом наборе выполняется по-настоящему: файлы сценариев идут
   последовательно (fileParallelism: false), а TRUNCATE + восстановление живут
   в одной транзакции. */
import { it, expect } from 'vitest'
import { describeIntegration, uniq, dbQuery, Session } from '../setup/harness.js'
import { registerVerified, newCompanyAdmin } from '../setup/factory.js'
import { useAuthStore } from '@/stores/auth.js'
import * as devicelink from '@/api/devicelink.js'
import * as rolesApi from '@/api/roles.js'
import * as backup from '@/api/backup.js'
import * as users from '@/api/users.js'
import { BACKUP_SECTIONS } from '@/utils/backupSections.js'

async function expectStatus(promise, status) {
  await expect(promise).rejects.toMatchObject({ status })
}

// Супер-админ: флаг платформенный, ролью его не выдать — ставим в БД и входим
// заново, чтобы клеймы обновились.
async function superAdmin(label = 'root') {
  const u = await registerVerified(label)
  dbQuery(`UPDATE users SET is_super_admin = TRUE WHERE id = ${u.auth.userId}`)
  const s = new Session(label + '-super')
  s.use()
  const a = useAuthStore()
  await a.login(u.login, u.password)
  return { session: s, auth: a, login: u.login, password: u.password, userId: u.auth.userId }
}

describeIntegration('devicelink API: вход по QR', () => {
  it('код заводится без авторизации и до подтверждения отдаёт «ждём»', async () => {
    const guest = new Session('device')
    guest.use()
    const started = await devicelink.linkStart('login')
    expect(started.code).toBeTruthy()
    expect(started.secret).toBeTruthy()
    expect(started.kind).toBe('login')
    expect(started.expires_in_sec).toBeGreaterThan(0)

    const pending = await devicelink.linkClaim(started.code, started.secret)
    expect(pending.status).toBe('pending')
  })

  it('подтверждение с телефона выдаёт сессию новому устройству — один раз', async () => {
    const guest = new Session('device')
    guest.use()
    const started = await devicelink.linkStart('login')

    const owner = await registerVerified('owner')
    owner.session.use()
    const info = await devicelink.linkInfo(started.code)
    expect(info.kind).toBe('login')
    expect(info.status).toBe('pending')
    await devicelink.linkApprove(started.code)

    guest.use()
    const claimed = await devicelink.linkClaim(started.code, started.secret)
    expect(claimed.status).toBe('ok')
    expect(claimed.session?.user_id ?? claimed.session?.userId).toBe(owner.auth.userId)

    // Код одноразовый: повторный запрос ничего не отдаст, даже с верным секретом.
    const again = await devicelink.linkClaim(started.code, started.secret)
    expect(again.status).toBe('expired')
  })

  it('без секрета код бесполезен: подсмотревший экран сессию не заберёт', async () => {
    const guest = new Session('device')
    guest.use()
    const started = await devicelink.linkStart('login')

    const owner = await registerVerified('owner')
    owner.session.use()
    await devicelink.linkApprove(started.code)

    const spy = new Session('spy')
    spy.use()
    await expect(devicelink.linkClaim(started.code, 'подсмотренный-секрет')).rejects.toBeTruthy()

    // Настоящий инициатор по-прежнему получает свою сессию.
    guest.use()
    expect((await devicelink.linkClaim(started.code, started.secret)).status).toBe('ok')
  })

  it('чужое повторное подтверждение отвергается', async () => {
    const guest = new Session('device')
    guest.use()
    const started = await devicelink.linkStart('login')

    const first = await registerVerified('first')
    first.session.use()
    await devicelink.linkApprove(started.code)
    // Тот же человек может подтвердить повторно — это идемпотентность, а не дыра.
    await devicelink.linkApprove(started.code)

    const second = await registerVerified('second')
    second.session.use()
    await expect(devicelink.linkApprove(started.code)).rejects.toBeTruthy()
  })

  it('несуществующий и просроченный код не работают', async () => {
    const guest = new Session('device')
    guest.use()
    await expect(devicelink.linkInfo('НЕТКОДА')).rejects.toBeTruthy()
    const claimed = await devicelink.linkClaim('НЕТКОДА', 'секрет')
    expect(claimed.status).toBe('expired')

    const u = await registerVerified()
    u.session.use()
    await expect(devicelink.linkApprove('НЕТКОДА')).rejects.toBeTruthy()
  })

  it('подтверждать спаривание вправе только вошедший', async () => {
    const guest = new Session('device')
    guest.use()
    const started = await devicelink.linkStart('login')
    await expectStatus(devicelink.linkApprove(started.code), 401)
  })
})

describeIntegration('devicelink API: ТВ-киоск', () => {
  it('ТВ-код требует активной компании и входит сразу в неё', async () => {
    const tv = new Session('tv')
    tv.use()
    const started = await devicelink.linkStart('tv')
    expect(started.kind).toBe('tv')

    /* Без активной компании подтверждать нечего: табло показывает данные
       КОНКРЕТНОЙ компании. Берём супер-админа — сессия без компании бывает
       только у него: остальным вход заводит личную компанию, если своих нет
       (см. startSession), и «человек без компании» через регистрацию больше не
       воспроизводится. */
    const root = await superAdmin('tvroot')
    root.session.use()
    await expect(devicelink.linkApprove(started.code)).rejects.toBeTruthy()

    const admin = await newCompanyAdmin('admin')
    admin.session.use()
    await devicelink.linkApprove(started.code)

    tv.use()
    const claimed = await devicelink.linkClaim(started.code, started.secret)
    expect(claimed.status).toBe('ok')
    expect(claimed.session?.company_id ?? claimed.session?.companyId).toBe(admin.companyId)
  })

  it('незнакомый вид спаривания трактуется как вход', async () => {
    const s = new Session('device')
    s.use()
    const started = await devicelink.linkStart('чепуха')
    expect(started.kind).toBe('login')
  })
})

describeIntegration('roles API', () => {
  it('справочник ролей — три фиксированные роли по возрастанию прав', async () => {
    const u = await registerVerified()
    u.session.use()
    const list = await rolesApi.getRoles()
    const items = list.roles ?? list
    expect(items.length).toBe(3)
    expect(items.map((r) => r.level).sort()).toEqual([1, 2, 3])
  })

  it('без авторизации справочник закрыт', async () => {
    const guest = new Session('guest')
    guest.use()
    await expectStatus(rolesApi.getRoles(), 401)
  })
})

describeIntegration('backup API: выгрузка и восстановление', () => {
  it('обычному пользователю бэкап недоступен', async () => {
    const u = await newCompanyAdmin()
    u.session.use()
    // Администратор компании — не супер-админ: в архиве данные всей платформы.
    await expectStatus(backup.exportBackup(), 403)
    await expectStatus(backup.importBackup(new File(['x'], 'b.zip')), 403)
  })

  it('супер-админ выгружает архив — целиком и по разделам', async () => {
    const root = await superAdmin()
    root.session.use()

    const all = Buffer.from(await (await backup.exportBackup()).arrayBuffer())
    expect(all.length).toBeGreaterThan(0)

    const part = Buffer.from(await (await backup.exportBackup(['auth'])).arrayBuffer())
    expect(part.length).toBeGreaterThan(0)
    // Выбранный раздел легче полного архива — иначе выбор ничего не значит.
    expect(part.length).toBeLessThanOrEqual(all.length)
  })

  it('выдуманный раздел не принимается', async () => {
    const root = await superAdmin()
    root.session.use()
    await expect(backup.exportBackup(['нет-такого-раздела'])).rejects.toBeTruthy()
  })

  it('каталог разделов фронта совпадает с серверным', async () => {
    const root = await superAdmin()
    root.session.use()
    // Раздел, которого сервер не знает, в интерфейсе означал бы кнопку в никуда.
    for (const section of BACKUP_SECTIONS) {
      const res = await backup.exportBackup([section.key])
      expect(res.ok).toBe(true)
    }
  })

  it('импорт восстанавливает состояние на момент выгрузки', async () => {
    const root = await superAdmin()
    root.session.use()
    const archive = Buffer.from(await (await backup.exportBackup()).arrayBuffer())

    // Всё, что появилось ПОСЛЕ выгрузки, восстановление обязано убрать —
    // иначе это не «вернуть как было», а слияние с непредсказуемым итогом.
    const later = uniq('after_')
    const stranger = await registerVerified('later')
    expect((await dbQuery(`SELECT count(*) FROM users WHERE id = ${stranger.auth.userId}`))[0][0]).toBe('1')

    root.session.use()
    const file = new File([archive], 'backup.zip', { type: 'application/zip' })
    await backup.importBackup(file)

    const gone = dbQuery(`SELECT count(*) FROM users WHERE id = ${stranger.auth.userId}`)
    expect(gone[0][0]).toBe('0')
    // А сам супер-админ, бывший в архиве, на месте — и по-прежнему ходит по API.
    expect((await users.getMe()).id).toBe(root.userId)
    expect(later).toBeTruthy()
  })

  it('битый архив не ломает базу', async () => {
    const root = await superAdmin()
    root.session.use()
    const before = dbQuery('SELECT count(*) FROM users')[0][0]

    await expect(backup.importBackup(new File(['совсем не zip'], 'broken.zip'))).rejects.toBeTruthy()
    // Транзакция целиком: неудачный импорт не оставляет пустых таблиц.
    expect(dbQuery('SELECT count(*) FROM users')[0][0]).toBe(before)
  })
})
