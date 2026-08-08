/**
 * Поиск и команды Hola-ассистента.
 *
 * Наследник строки Spotlight: ищет параллельно по всем разделам (свои — по
 * загруженным сторам, содержимое — серверными ручками), разбирает быстрые
 * команды и предлагает выдачу поисковика для того же запроса. Живёт отдельно
 * от вью: этим же ядром пользуются вкладки «Поиск» и «Команды».
 */
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useDesktopStore } from '@/stores/desktop.js'
import { usePermission, ROLES } from '@/composables/usePermission.js'
import { useCompanySettings } from '@/composables/useCompanySettings.js'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useAuthStore } from '@/stores/auth.js'
import { useMessengerStore } from '@/stores/messenger.js'
import { useNotesStore } from '@/stores/notes.js'
import { useBoardsStore } from '@/stores/boards.js'
import { useRemindersStore } from '@/stores/reminders.js'
import { APPS, appById } from '@/desktop/apps.js'
import { shellActive } from '@/desktop/layout.js'
import { calculate, formatResult } from '@/utils/calc.js'
import { parseQuickCommand } from '@/utils/quickCommands.js'
import { humanWhen } from '@/utils/naturalDate.js'
import { resolveRecipients, searchStem } from '@/utils/recipients.js'
import { stripMarkdown } from '@/utils/markdown.js'
import { settingsSections } from '@/utils/settingsSections.js'
import { flatHelpArticles } from '@/utils/helpArticles.js'
import { enginesInOrder, getSearchEngine, openUrl, openWebSearch, parseUrl } from '@/utils/webSearch.js'
import { getTasks } from '@/api/tasks.js'
import { getNotes } from '@/api/notes.js'
import { getBoards } from '@/api/boards.js'
import { browse as browseDrive } from '@/api/drive.js'
import { fileIcon } from '@/utils/fileTypes.js'
import { searchEntries } from '@/api/diaries.js'
import { searchRecords } from '@/api/registries.js'
import { getPosts } from '@/api/portal.js'
import { getDirectory } from '@/api/users.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const LIMIT = 5
const DEBOUNCE = 220

const REPEAT_LABELS = {
  daily: 'каждый день', weekdays: 'по рабочим дням', weekly: 'каждую неделю',
  monthly: 'каждый месяц', yearly: 'каждый год',
}

function emptyHits() {
  return { tasks: [], notes: [], boards: [], drive: [], diaries: [], registries: [], portal: [], people: [], messages: [] }
}

export function useHolaSearch() {
  const router = useRouter()
  const desktop = useDesktopStore()
  const notif = useNotificationsStore()
  const auth = useAuthStore()
  const messenger = useMessengerStore()
  const notes = useNotesStore()
  const boards = useBoardsStore()
  const reminders = useRemindersStore()
  const { isSuperAdmin, hasActiveCompany, isAtLeast, canManageCompanies } = usePermission()
  const { settings, usesGroove } = useCompanySettings()
  const { isMobile } = useBreakpoint()

  const query = ref('')
  const loading = ref(false)
  const hits = ref(emptyHits())
  const engine = ref(getSearchEngine())

  const ctx = computed(() => ({
    hasCompany: hasActiveCompany(),
    isSuperAdmin: isSuperAdmin(),
    settings: settings.value,
  }))

  const needle = computed(() => query.value.trim().toLowerCase())

  /* Раздел рабочего стола открывается окном, а на мобильном каркасе (окон там
     нет) — обычным переходом. */
  function openPath(path) {
    if (shellActive.value) desktop.open(path)
    else router.push(path)
  }

  /* ── Калькулятор ── */
  const calc = computed(() => calculate(query.value))
  const calcText = computed(() => (calc.value === null ? '' : formatResult(calc.value)))

  async function copyCalc() {
    try {
      await navigator.clipboard.writeText(String(calc.value))
      return true
    } catch {
      notif.warn('Не удалось скопировать результат')
      return false
    }
  }

  /* ── Локальные источники: разделы, настройки, переписки ── */
  const appHits = computed(() => {
    const q = needle.value
    if (!q) return []
    return APPS
      .filter((a) => a.available(ctx.value) && a.title.toLowerCase().includes(q))
      .slice(0, LIMIT)
      .map((a) => ({ key: `app-${a.id}`, icon: a.icon, title: a.title, subtitle: 'Открыть раздел', path: a.path }))
  })

  const settingHits = computed(() => {
    const q = needle.value
    if (!q) return []
    return settingsSections({
      isMobile: isMobile.value,
      hasCompany: hasActiveCompany(),
      isAdmin: isAtLeast(ROLES.ADMIN),
      isSuperAdmin: isSuperAdmin(),
    })
      .filter((s) => [s.title, s.desc].some((t) => t.toLowerCase().includes(q)))
      .slice(0, LIMIT)
      .map((s) => ({
        key: `set-${s.key}`,
        icon: s.icon,
        title: s.title,
        subtitle: `Настройки · ${s.desc}`,
        // Пункт-ссылка ведёт в свой раздел напрямую — крюк через /settings
        // только мигнул бы списком настроек перед редиректом.
        path: s.to || `/settings?section=${s.key}`,
      }))
  })

  /* ── Справка и поддержка ──
     Тот же каталог статей, что у панели «Справка и поддержка» (utils/helpArticles.js):
     совпадение в тексте/советах поднимается подзаголовком как быстрый ответ,
     клик открывает статью напрямую (?article=), а не список. Если ничего не
     нашли — карточка «спросить в поддержке» с текстом запроса наготове. */
  const helpArticleHits = computed(() => {
    const q = needle.value
    if (!q) return []
    const ctx = {
      hasCompany: hasActiveCompany(),
      isSuperAdmin: isSuperAdmin(),
      canManageCompanies: canManageCompanies(),
      isManager: isAtLeast(ROLES.MANAGER),
      usesGroove: usesGroove.value,
    }
    return flatHelpArticles(ctx)
      .map((a) => ({ a, snippet: helpSnippet(a, q) }))
      .filter((x) => x.snippet)
      .slice(0, LIMIT)
      .map(({ a, snippet }) => ({
        key: `help-${a.id}`,
        icon: a.icon,
        title: a.title,
        subtitle: snippet,
        path: `/settings?section=help&article=${a.id}`,
      }))
  })

  /* Быстрый ответ: если запрос совпал внутри текста или совета — именно эта
     фраза и есть ответ, показываем её вместо общего подзаголовка статьи. */
  function helpSnippet(a, q) {
    const hit = [...(a.text || []), ...(a.tips || [])].find((s) => s.toLowerCase().includes(q))
    if (hit) return hit.length > 110 ? `${hit.slice(0, 110)}…` : hit
    if (a.title.toLowerCase().includes(q) || a.subtitle.toLowerCase().includes(q)) return a.subtitle
    return ''
  }

  const supportHits = computed(() => {
    const q = query.value.trim()
    if (q.length < 3 || helpArticleHits.value.length) return []
    return [{
      key: 'help-ask-support',
      icon: 'support_agent',
      title: `Спросить в поддержке: «${q}»`,
      subtitle: 'Ничего похожего в справке не нашли — сообщение уйдёт команде разработки',
      run: () => askSupport(q),
    }]
  })

  async function askSupport(text) {
    const id = await messenger.openDevChat()
    await messenger.send(id, { text })
    notif.success('Отправлено в поддержку')
    openPath(`/messenger/${id}`)
  }

  const chatHits = computed(() => {
    const q = needle.value
    if (!q) return []
    return messenger.conversations
      .filter((c) => chatName(c).toLowerCase().includes(q))
      .slice(0, LIMIT)
      .map((c) => ({
        key: `conv-${c.id}`,
        icon: c.is_dev_chat ? 'support_agent' : (c.is_group ? 'groups' : 'forum'),
        avatar: chatAvatar(c),
        title: chatName(c),
        subtitle: c.last_message?.text?.trim() || (c.is_group ? 'Групповой чат' : 'Личная переписка'),
        path: `/messenger/${c.id}`,
      }))
  })

  function chatName(c) {
    if (c.is_dev_chat) return 'Техподдержка'
    return (c.is_group ? c.title : c.other_user?.fio) || 'Чат'
  }

  function chatAvatar(c) {
    if (c.is_group) return c.avatar_path ? `/uploads/${c.avatar_path}` : null
    const u = c.other_user
    if (!u || c.is_dev_chat) return null
    return u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`
  }

  /* ── Интернет ──
     Выдачу не проксируем: строка ведёт на страницу поисковика в новой вкладке
     браузера. Выбранный по умолчанию идёт первым, остальные — следом. Введённый
     адрес сайта («vk.com», «https://ya.ru/maps») открываем напрямую — искать
     его в поисковике пользователь не просил. */
  const webUrl = computed(() => parseUrl(query.value))

  const webHits = computed(() => {
    const q = query.value.trim()
    if (q.length < 2) return []

    const engines = enginesInOrder(engine.value).map((e) => ({
      key: `web-${e.key}`,
      web: true,
      icon: 'travel_explore',
      title: `Найти в ${e.label} — «${q}»`,
      subtitle: 'Откроется новой вкладкой браузера',
      run: () => openWebSearch(e.key, q),
    }))

    const url = webUrl.value
    if (!url) return engines
    return [{
      key: 'web-open',
      web: true,
      icon: 'open_in_browser',
      title: `Открыть ${url.label}`,
      subtitle: 'Перейти на сайт в новой вкладке браузера',
      run: () => openUrl(url.href),
    }, ...engines]
  })

  /* ── Быстрые команды из фразы ── */
  const command = computed(() => parseQuickCommand(query.value))

  const canCreateTasks = computed(() =>
    appById('tasks').available(ctx.value) && !auth.user?.on_vacation)

  const commandHits = computed(() => {
    const cmd = command.value
    // Адресатов сообщения знает только сервер — их строки собирает search().
    if (!cmd || cmd.kind === 'message') return []
    const quoted = cmd.title ? ` «${cmd.title}»` : ''

    if (cmd.kind === 'task') {
      if (!canCreateTasks.value) return []
      return [{
        key: 'cmd-task',
        command: true,
        icon: 'add_task',
        title: `Создать задачу${quoted}`,
        subtitle: 'Откроется форма с заполненным названием',
        path: `/tasks?new=1&title=${encodeURIComponent(cmd.title)}`,
      }]
    }

    if (cmd.kind === 'board') {
      return [{
        key: 'cmd-board',
        command: true,
        icon: 'gesture',
        title: `Создать доску${quoted}`,
        subtitle: 'Доска появится в разделе и сразу откроется',
        run: () => createBoard(cmd.title),
      }]
    }

    if (cmd.kind === 'reminder') {
      // Срок в фразе не назвали — без него напоминание бессмысленно, поэтому
      // отдаём форму с готовым названием (как задаче).
      if (!cmd.at) {
        return [{
          key: 'cmd-reminder',
          command: true,
          icon: 'alarm_add',
          title: `Создать напоминание${quoted}`,
          subtitle: 'Откроется форма — останется выбрать время',
          path: `/reminders?new=1&title=${encodeURIComponent(cmd.title)}`,
        }]
      }
      return [{
        key: 'cmd-reminder',
        command: true,
        icon: 'alarm_add',
        title: `Напомнить${quoted}`,
        subtitle: [humanWhen(cmd.at), REPEAT_LABELS[cmd.repeat?.kind] || ''].filter(Boolean).join(' · '),
        run: () => createReminder(cmd),
      }]
    }

    return [{
      key: 'cmd-note',
      command: true,
      icon: 'note_add',
      title: `Создать заметку${quoted}`,
      subtitle: 'Заметка появится в разделе и сразу откроется',
      run: () => createNote(cmd.title),
    }]
  })

  async function createNote(title) {
    const note = await notes.createNote(title)
    openPath(`/notes/${note.id}`)
  }

  async function createBoard(title) {
    const board = await boards.createBoard(title || 'Новая доска')
    openPath(`/boards/${board.id}`)
  }

  async function createReminder(cmd) {
    await reminders.create({
      title: cmd.title || 'Напоминание',
      note: '',
      remind_at: cmd.at.toISOString(),
      timezone: Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC',
      repeat: cmd.repeat || { kind: 'none', interval: 1, days: [] },
    })
    notif.success(`Напомню ${humanWhen(cmd.at)}`)
  }

  /* ── Каталог быстрых команд (вкладка «Команды») ──
     Всё, что заводится одним нажатием, без ввода фразы. Фильтруется тем же
     полем: набранное слово сужает список. */
  const commandCatalog = computed(() => {
    const list = [
      {
        key: 'new-note',
        icon: 'note_add',
        title: 'Новая заметка',
        subtitle: 'Создать и сразу открыть',
        run: () => createNote(''),
      },
      {
        key: 'new-chat',
        icon: 'chat_add_on',
        title: 'Новый чат',
        subtitle: 'Выбрать собеседника в мессенджере',
        path: '/messenger?new=1',
      },
      {
        key: 'calculator',
        icon: 'calculate',
        title: 'Калькулятор',
        subtitle: 'Обычный и инженерный режимы',
        path: '/calculator',
      },
      {
        key: 'new-reminder',
        icon: 'alarm_add',
        title: 'Создать напоминание',
        subtitle: 'Форма с выбором срока и повтора',
        path: '/reminders?new=1',
      },
      {
        key: 'new-board',
        icon: 'gesture',
        title: 'Новая доска',
        subtitle: 'Чистый холст для рисования',
        run: () => createBoard(''),
      },
    ]
    if (canCreateTasks.value) {
      list.splice(1, 0, {
        key: 'new-task',
        icon: 'add_task',
        title: 'Новая задача',
        subtitle: 'Форма создания задачи',
        path: '/tasks?new=1',
      })
    }
    const q = needle.value
    if (!q) return list
    return list.filter((c) => `${c.title} ${c.subtitle}`.toLowerCase().includes(q))
  })

  /* ── Кому написать ──
     Своих собеседников (личные чаты и группы) знаем локально, остальных — из
     каталога компании; имя в дательном падеже разбирает utils/recipients.js. */
  function recipientPool(dirUsers) {
    const pool = []
    const seen = new Set()
    for (const c of messenger.conversations) {
      if (c.is_dev_chat) continue
      if (c.is_group) {
        pool.push({
          key: `g${c.id}`,
          names: [c.title || ''],
          title: c.title || 'Групповой чат',
          icon: 'groups',
          avatar: chatAvatar(c),
          conversationId: c.id,
        })
      } else if (c.other_user?.id) {
        const u = c.other_user
        seen.add(u.id)
        pool.push({
          key: `u${u.id}`,
          names: [u.fio || '', u.login || ''],
          title: u.fio || u.login,
          icon: 'forum',
          avatar: chatAvatar(c),
          conversationId: c.id,
          userId: u.id,
        })
      }
    }
    for (const u of dirUsers) {
      if (!u?.id || seen.has(u.id) || u.id === auth.userId) continue
      pool.push({
        key: `u${u.id}`,
        names: [u.fio || '', u.login || ''],
        title: u.fio || u.login,
        icon: 'account_circle',
        avatar: u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`,
        userId: u.id,
      })
    }
    return pool
  }

  function messageHits(cmd, dirUsers) {
    const { text, matches } = resolveRecipients(cmd.rest, recipientPool(dirUsers))
    return matches.slice(0, LIMIT).map((p) => ({
      key: `msg-${p.key}`,
      command: true,
      icon: p.icon,
      avatar: p.avatar,
      title: text ? `${p.title} — «${text}»` : p.title,
      subtitle: text ? 'Отправить и открыть чат' : 'Открыть переписку',
      run: () => sendTo(p, text),
    }))
  }

  async function sendTo(p, text) {
    const conversationId = p.conversationId ?? await messenger.openWith(p.userId)
    if (text) {
      await messenger.send(conversationId, { text })
      notif.success(`Отправлено: ${p.title}`)
    }
    openPath(`/messenger/${conversationId}`)
  }

  /* ── Выдача вкладки «Поиск» ──
     Секции идут от самого вероятного намерения к общему поиску; узнанный адрес
     сайта поднимает «Интернет» наверх — его набрали, чтобы туда перейти. */
  const sections = computed(() => {
    const list = [
    { key: 'messages', label: 'Написать', items: hits.value.messages },
    { key: 'commands', label: 'Быстрые действия', items: commandHits.value },
    { key: 'apps', label: 'Разделы', items: appHits.value },
    { key: 'tasks', label: 'Задачи', items: hits.value.tasks },
    { key: 'notes', label: 'Заметки', items: hits.value.notes },
    { key: 'boards', label: 'Доски', items: hits.value.boards },
    { key: 'drive', label: 'Диск', items: hits.value.drive },
    { key: 'diaries', label: 'Ежедневники', items: hits.value.diaries },
    { key: 'registries', label: 'Реестры', items: hits.value.registries },
    { key: 'portal', label: 'Портал', items: hits.value.portal },
    { key: 'chats', label: 'Переписки', items: chatHits.value },
    { key: 'people', label: 'Сотрудники', items: hits.value.people },
    { key: 'settings', label: 'Настройки', items: settingHits.value },
    { key: 'help', label: 'Справка', items: [...helpArticleHits.value, ...supportHits.value] },
    { key: 'web', label: 'Интернет', items: webHits.value },
    ].filter((s) => s.items.length)

    if (!webUrl.value) return list
    return [...list.filter((s) => s.key === 'web'), ...list.filter((s) => s.key !== 'web')]
  })

  let timer = null
  let ctrl = null
  let seq = 0

  /** Перезапуск поиска после изменения строки (с задержкой). */
  function schedule() {
    clearTimeout(timer)
    ctrl?.abort()
    const q = query.value.trim()
    if (q.length < 2) {
      hits.value = emptyHits()
      loading.value = false
      return
    }
    timer = setTimeout(() => search(q), DEBOUNCE)
  }

  async function search(q) {
    const my = ++seq
    ctrl = new AbortController()
    const opt = { signal: ctrl.signal }
    loading.value = true

    const withCompany = hasActiveCompany()
    /* «напиши васе …»: в каталоге ищем по основе имени («вас»), а не по всей
       фразе. Тот же запрос кормит и секцию «Сотрудники» — второй раз каталог
       не дёргаем, просто адресаты вытесняют её из выдачи. */
    const cmd = command.value
    const stem = cmd?.kind === 'message' ? searchStem(cmd.rest) : ''
    const dirQuery = stem || q

    const [tasks, noteHits, boardHits, driveHits, diaries, registries, portal, people] = await Promise.allSettled([
      withCompany ? getTasks({ search: q, per_page: LIMIT }, opt) : Promise.resolve(null),
      getNotes({ search: q }, opt),
      getBoards({ search: q }, opt),
      browseDrive({ search: q }, opt),
      searchEntries(q, LIMIT, opt),
      withCompany ? searchRecords(q, LIMIT, opt) : Promise.resolve(null),
      withCompany ? getPosts({ search: q, limit: LIMIT }, opt) : Promise.resolve(null),
      withCompany && dirQuery ? getDirectory(dirQuery, true) : Promise.resolve(null),
    ])
    if (my !== seq) return

    hits.value = {
      messages: cmd?.kind === 'message' ? messageHits(cmd, value(people) ?? []) : [],
      tasks: (value(tasks)?.tasks ?? value(tasks)?.items ?? []).slice(0, LIMIT).map((t) => ({
        key: `task-${t.id}`,
        icon: 'dashboard_customize',
        title: t.name || `Задача #${t.id}`,
        subtitle: t.department_name || t.stage_name || 'Задача',
        path: `/tasks/${t.id}`,
      })),
      notes: (value(noteHits)?.notes ?? []).slice(0, LIMIT).map((n) => ({
        key: `note-${n.id}`,
        icon: 'filter_none',
        title: n.title || 'Без названия',
        subtitle: (n.text_content || '').slice(0, 90) || 'Заметка',
        path: `/notes/${n.id}`,
      })),
      boards: (value(boardHits)?.boards ?? []).slice(0, LIMIT).map((b) => ({
        key: `board-${b.id}`,
        icon: 'gesture',
        title: b.title || 'Без названия',
        subtitle: (b.excerpt || '').slice(0, 90) || 'Доска',
        path: `/boards/${b.id}`,
      })),
      /* «Диск»: и папки, и файлы. Поиск там глобальный, поэтому текущая
         папка выдачу не сужает; папка открывается сразу собой. */
      drive: [
        ...(value(driveHits)?.folders ?? []).slice(0, LIMIT).map((f) => ({
          key: `drive-folder-${f.id}`,
          icon: 'folder',
          title: f.name || 'Папка',
          subtitle: 'Папка на диске',
          path: `/drive?folder=${f.id}`,
        })),
        ...(value(driveHits)?.files ?? []).slice(0, LIMIT).map((f) => ({
          key: `drive-${f.id}`,
          icon: fileIcon(f.mime, f.name),
          title: f.name || 'Файл',
          subtitle: 'Файл на диске',
          path: '/drive',
        })),
      ].slice(0, LIMIT),
      diaries: (value(diaries)?.items ?? []).map((e) => ({
        key: `diary-${e.entry_id}`,
        icon: 'event_list',
        title: e.title,
        subtitle: `${e.diary_name} · ${e.entry_date}`,
        path: `/diaries?diary=${e.diary_id}&q=${encodeURIComponent(e.title)}`,
      })),
      registries: (value(registries)?.items ?? []).map((r) => ({
        key: `record-${r.record_id}`,
        icon: 'list_alt_add',
        title: r.snippet || `Запись #${r.record_id}`,
        subtitle: r.registry_name,
        path: `/registries?registry=${r.registry_id}&q=${encodeURIComponent(q)}`,
      })),
      // Лента отдаёт закреплённые отдельным списком — в поиске они такие же посты.
      portal: [...(value(portal)?.pinned ?? []), ...(value(portal)?.posts ?? [])]
        .slice(0, LIMIT)
        .map((p) => {
          const text = stripMarkdown(p.body || '').trim()
          return {
            key: `post-${p.id}`,
            icon: 'web_stories',
            title: p.title || text.slice(0, 80) || `Публикация #${p.id}`,
            subtitle: (p.title ? text : '').slice(0, 90) || 'Публикация портала',
            path: `/portal/${p.id}`,
          }
        }),
      people: (cmd?.kind === 'message' ? [] : value(people) ?? []).slice(0, LIMIT).map((u) => ({
        key: `user-${u.id}`,
        icon: 'account_circle',
        avatar: u.avatar_path ? `/uploads/${u.avatar_path}` : `/api/users/${u.id}/identicon`,
        title: u.fio || u.login,
        subtitle: u.post || 'Сотрудник — открыть карточку',
        path: `/employees?user=${u.id}`,
      })),
    }
    loading.value = false
  }

  // Упавший источник не должен ронять весь поиск: остальные результаты покажем.
  function value(result) {
    return result.status === 'fulfilled' ? result.value : null
  }

  /**
   * Выполняет строку выдачи: переход открывает окно раздела, команда —
   * действие (её ждём, чтобы пользователь видел работу), веб-поиск уводит в
   * новую вкладку браузера.
   */
  async function run(item) {
    if (!item) return false
    if (!item.run) {
      openPath(item.path)
      return true
    }
    if (loading.value) return false
    loading.value = true
    try {
      await item.run()
      return true
    } catch (e) {
      notif.error(e?.message || 'Не удалось выполнить команду')
      return false
    } finally {
      loading.value = false
    }
  }

  function stop() {
    clearTimeout(timer)
    ctrl?.abort()
  }

  function setEngine(key) {
    engine.value = key
  }

  return {
    query, loading, sections, commandCatalog, calc, calcText, engine,
    schedule, run, stop, copyCalc, openPath, setEngine,
  }
}
