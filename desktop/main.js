/* Groove Work — десктоп-клиент.
 *
 * Тонкая обёртка: окно грузит ПРОД-URL (https://gw.kodass.ru), а не локальный
 * бандл — поэтому относительные пути /api, /ws, /uploads и HttpOnly
 * refresh-cookie (SameSite=Strict) работают без правок фронта и бэка,
 * а обновления UI прилетают обычным деплоем сервера. Нативного кода в
 * странице нет — preload не нужен.
 *
 * URL можно переопределить: env GW_DESKTOP_URL или {"url": "..."} в
 * <userData>/config.json (для стенда/дева).
 */
const { app, BrowserWindow, Tray, Menu, shell, dialog, session, desktopCapturer, nativeImage, net, ipcMain } = require('electron')
const path = require('node:path')
const fs = require('node:fs')

const DEFAULT_URL = 'https://gw.kodass.ru'

function readConfigUrl() {
  try {
    const raw = fs.readFileSync(path.join(app.getPath('userData'), 'config.json'), 'utf8')
    const url = JSON.parse(raw).url
    if (typeof url === 'string' && /^https?:\/\//.test(url)) return url
  } catch {}
  return null
}

// Один экземпляр: повторный запуск фокусирует существующее окно.
if (!app.requestSingleInstanceLock()) app.quit()

// Веб-версия «разогревает» AudioContext первым жестом из-за autoplay-политики
// браузера; в своём приложении просто разрешаем звук сразу — бипы уведомлений
// и предупреждений играют без предварительного клика.
app.commandLine.appendSwitch('autoplay-policy', 'no-user-gesture-required')

// Windows: без AppUserModelID тосты уведомлений не показываются
// (должен совпадать с appId в electron-builder).
app.setAppUserModelId('ru.kodass.groovework')

let mainWindow = null
let tray = null
let quitting = false

/* ── Положение/размер окна переживают перезапуск ── */
const stateFile = () => path.join(app.getPath('userData'), 'window-state.json')

function loadWindowState() {
  try {
    const s = JSON.parse(fs.readFileSync(stateFile(), 'utf8'))
    if (Number.isFinite(s.width) && Number.isFinite(s.height)) return s
  } catch {}
  return { width: 1280, height: 820 }
}

function saveWindowState() {
  if (!mainWindow || mainWindow.isDestroyed()) return
  try {
    const b = mainWindow.getNormalBounds()
    fs.writeFileSync(stateFile(), JSON.stringify({ ...b, maximized: mainWindow.isMaximized() }))
  } catch {}
}

function createWindow(appUrl) {
  const state = loadWindowState()
  mainWindow = new BrowserWindow({
    ...state,
    minWidth: 480,
    minHeight: 600,
    show: false,
    autoHideMenuBar: true,
    backgroundColor: '#1a1c1e',
    webPreferences: {
      // Удалённая страница: никакого Node в рендерере; узкий мост
      // window.GrooveDesktop (обновление обёртки) — через preload.
      contextIsolation: true,
      nodeIntegration: false,
      preload: path.join(__dirname, 'preload.js'),
      spellcheck: true,
    },
  })

  if (state.maximized) mainWindow.maximize()
  mainWindow.once('ready-to-show', () => mainWindow.show())

  let saveTimer = null
  const scheduleSave = () => { clearTimeout(saveTimer); saveTimer = setTimeout(saveWindowState, 400) }
  mainWindow.on('resize', scheduleSave)
  mainWindow.on('move', scheduleSave)

  // target=_blank: свой origin (вложения /uploads, публичные ссылки) — окно
  // приложения, всё чужое — системный браузер.
  mainWindow.webContents.setWindowOpenHandler(({ url }) => {
    if (url.startsWith(appUrl)) return { action: 'allow' }
    shell.openExternal(url)
    return { action: 'deny' }
  })

  // Уход со своего origin в самом окне (внешние ссылки без _blank) — тоже
  // в системный браузер.
  mainWindow.webContents.on('will-navigate', (e, url) => {
    if (!url.startsWith(appUrl)) {
      e.preventDefault()
      shell.openExternal(url)
    }
  })

  // beforeunload-гард веб-версии (идёт юнит) в Electron не показывает
  // нативный диалог сам — рисуем свою модалку здесь.
  mainWindow.webContents.on('will-prevent-unload', (e) => {
    const choice = dialog.showMessageBoxSync(mainWindow, {
      type: 'warning',
      buttons: ['Остаться', 'Закрыть'],
      defaultId: 0,
      cancelId: 0,
      title: 'Идёт работа',
      message: 'Сейчас идёт работа над юнитом.',
      detail: 'Если закрыть приложение, учёт времени продолжится на сервере, но вы перестанете его видеть. Закрыть?',
    })
    if (choice === 1) e.preventDefault() // preventDefault = игнорировать beforeunload и закрыться
  })

  // Крестик прячет окно в трей (WS живёт — уведомления продолжают приходить);
  // полный выход — из меню трея или Cmd/Ctrl+Q.
  mainWindow.on('close', (e) => {
    if (quitting) { saveWindowState(); return }
    e.preventDefault()
    mainWindow.hide()
  })

  /* ── Загрузка приложения с сервера: сплэш + автоповтор ──
     Раньше loadURL без обработки ошибок давал белый экран навсегда, если
     сеть в момент старта ещё не поднялась (автозапуск после входа в ОС,
     сон/пробуждение) — лечилось только ручными перезагрузками. Теперь окно
     сразу показывает локальный сплэш «Подключаемся…», а неудачная загрузка
     возвращает на сплэш с сообщением и повторяет попытку с бэкоффом. */
  const splash = (hash) => mainWindow.loadFile(path.join(__dirname, 'loading.html'), hash ? { hash } : {})
  let retryTimer = null
  let retryDelay = 2000
  const loadApp = () => {
    if (!mainWindow || mainWindow.isDestroyed()) return
    mainWindow.loadURL(appUrl)
  }

  mainWindow.webContents.on('did-fail-load', (e, code, desc, url, isMainFrame) => {
    // -3 (ERR_ABORTED) — навигацию сменила другая (в т.ч. наш сплэш), не ошибка.
    if (!isMainFrame || code === -3) return
    splash('error')
    clearTimeout(retryTimer)
    retryTimer = setTimeout(loadApp, retryDelay)
    retryDelay = Math.min(retryDelay * 2, 15_000)
  })

  // Успешная загрузка приложения — сбрасываем бэкофф на будущее.
  mainWindow.webContents.on('did-finish-load', () => {
    if (mainWindow.webContents.getURL().startsWith(appUrl)) retryDelay = 2000
  })

  // Упавший рендерер (OOM, краш GPU) — тоже не белый экран, а перезагрузка.
  mainWindow.webContents.on('render-process-gone', () => {
    splash()
    setTimeout(loadApp, 1000)
  })

  splash()
  loadApp()
}

function showWindow() {
  if (!mainWindow || mainWindow.isDestroyed()) return
  if (mainWindow.isMinimized()) mainWindow.restore()
  mainWindow.show()
  mainWindow.focus()
}

function createTray() {
  // Иконку трея берём с сервера нельзя — кладём из ресурсов сборки; в деве
  // рядом лежит build/icon.png.
  const icon = nativeImage
    .createFromPath(path.join(__dirname, 'build', 'icon.png'))
    .resize({ width: 18, height: 18 })
  icon.setTemplateImage(false)
  tray = new Tray(icon)
  tray.setToolTip('Groove Work')
  tray.setContextMenu(Menu.buildFromTemplate([
    { label: 'Открыть Groove Work', click: showWindow },
    { type: 'separator' },
    { label: 'Выйти', click: () => app.quit() },
  ]))
  tray.on('click', showWindow)
}

/* Стандартное меню (горячие клавиши копирования/вставки/зума обязаны жить,
   особенно на macOS); само меню скрыто autoHideMenuBar на Win/Linux. */
function buildMenu() {
  const template = [
    ...(process.platform === 'darwin' ? [{ role: 'appMenu' }] : []),
    { role: 'editMenu' },
    {
      label: 'Вид',
      submenu: [
        { role: 'reload' }, { role: 'forceReload' },
        { type: 'separator' },
        { role: 'resetZoom' }, { role: 'zoomIn' }, { role: 'zoomOut' },
        { type: 'separator' },
        { role: 'togglefullscreen' }, { role: 'toggleDevTools' },
      ],
    },
    { role: 'windowMenu' },
  ]
  Menu.setApplicationMenu(Menu.buildFromTemplate(template))
}

/* ── Обновление самой обёртки (аналог OTA мобильного приложения) ──
   UI приезжает с сервера сам при каждом деплое; здесь следим только за
   версией оболочки: apps/desktop/version.json на сервере против
   app.getVersion() (метка и артефакты выкладываются `make deploy-desktop`).
   Новее → предлагаем скачать установщик своей платформы. */
const UPDATE_CHECK_MS = 6 * 60 * 60 * 1000
let updateOffered = null // версия, которую уже предлагали — не спамим диалогом

function isNewer(a, b) {
  const pa = String(a).split('.').map(Number)
  const pb = String(b).split('.').map(Number)
  for (let i = 0; i < 3; i++) {
    if ((pa[i] || 0) !== (pb[i] || 0)) return (pa[i] || 0) > (pb[i] || 0)
  }
  return false
}

async function checkShellUpdate(appUrl) {
  let meta = null
  try {
    const res = await net.fetch(`${appUrl}/apps/desktop/version.json`, { cache: 'no-store' })
    if (!res.ok) return
    meta = await res.json()
  } catch { return } // сети нет — проверим в следующий раз
  const latest = meta?.current_version
  const key = process.platform === 'darwin' ? 'mac' : process.platform === 'win32' ? 'win' : 'linux'
  const file = meta?.files?.[key]
  if (!latest || !file || !isNewer(latest, app.getVersion()) || updateOffered === latest) return
  updateOffered = latest
  const { response } = await dialog.showMessageBox(mainWindow, {
    type: 'info',
    buttons: ['Скачать', 'Позже'],
    defaultId: 0,
    cancelId: 1,
    title: 'Обновление приложения',
    message: `Доступна новая версия приложения — ${latest}`,
    detail: `У вас ${app.getVersion()}. Скачать установщик? Новая версия встанет поверх текущей.`,
  })
  if (response === 0) shell.openExternal(`${appUrl}/apps/desktop/${file}`)
}

app.on('second-instance', showWindow)
app.on('before-quit', () => { quitting = true })
app.on('activate', showWindow) // macOS: клик по доку

app.whenReady().then(() => {
  const appUrl = process.env.GW_DESKTOP_URL || readConfigUrl() || DEFAULT_URL
  const appOrigin = new URL(appUrl).origin

  /* ── IPC-мост обновления обёртки (preload.js → window.GrooveDesktop) ──
     Принудительная проверка/установка из карточки «О приложении», без
     ожидания фоновой автопроверки. Ошибки — полем error (не reject: throw из
     handle доезжает до рендерера с техническим префиксом Electron). */
  const platformKey = process.platform === 'darwin' ? 'mac' : process.platform === 'win32' ? 'win' : 'linux'
  const fetchDesktopMeta = async () => {
    const res = await net.fetch(`${appUrl}/apps/desktop/version.json`, { cache: 'no-store' })
    if (!res.ok) return null
    return res.json().catch(() => null)
  }

  ipcMain.handle('gw:get-version', () => ({ version: app.getVersion() }))

  ipcMain.handle('gw:check-update', async () => {
    const meta = await fetchDesktopMeta().catch(() => null)
    if (!meta?.current_version) return { error: 'Не удалось проверить обновления — проверьте интернет' }
    return {
      current: app.getVersion(),
      latest: meta.current_version,
      updateAvailable: isNewer(meta.current_version, app.getVersion()) && !!meta.files?.[platformKey],
    }
  })

  // Скачивает установщик своей платформы во временный каталог (прогресс —
  // событиями в рендерер) и запускает его: NSIS ставится поверх сам, dmg/
  // AppImage открываются пользователю.
  ipcMain.handle('gw:download-update', async () => {
    const meta = await fetchDesktopMeta().catch(() => null)
    const file = meta?.files?.[platformKey]
    if (!file) return { error: 'Не удалось скачать обновление' }
    try {
      const res = await net.fetch(`${appUrl}/apps/desktop/${file}`, { cache: 'no-store' })
      if (!res.ok || !res.body) throw new Error(`HTTP ${res.status}`)
      const total = Number(res.headers.get('content-length')) || 0
      const dest = path.join(app.getPath('temp'), file)
      const ws = fs.createWriteStream(dest)
      let read = 0
      for await (const chunk of res.body) {
        ws.write(Buffer.from(chunk))
        read += chunk.length
        mainWindow?.webContents.send('gw:update-progress', total ? read / total : -1)
      }
      await new Promise((resolve) => ws.end(resolve))
      const openErr = await shell.openPath(dest)
      if (openErr) throw new Error(openErr)
      return { status: 'installing' }
    } catch {
      return { error: 'Не удалось скачать обновление — попробуйте ещё раз' }
    }
  })

  // Разрешения — только своему origin и только нужные приложению:
  // уведомления, микрофон/камера (звонки), захват экрана, fullscreen.
  const ALLOWED = new Set(['notifications', 'media', 'display-capture', 'fullscreen', 'clipboard-sanitized-write'])
  session.defaultSession.setPermissionRequestHandler((wc, permission, cb, details) => {
    cb(ALLOWED.has(permission) && details.requestingUrl?.startsWith(appOrigin))
  })

  // Демонстрация экрана в звонках: на macOS — системный пикер, иначе отдаём
  // основной экран (свой пикер источников — возможное развитие).
  session.defaultSession.setDisplayMediaRequestHandler(async (request, cb) => {
    try {
      const sources = await desktopCapturer.getSources({ types: ['screen'] })
      cb({ video: sources[0] })
    } catch {
      cb({})
    }
  }, { useSystemPicker: true })

  buildMenu()
  createWindow(appUrl)
  createTray()

  // Первая проверка — после того как окно загрузится и осядет.
  setTimeout(() => checkShellUpdate(appUrl), 15_000)
  setInterval(() => checkShellUpdate(appUrl), UPDATE_CHECK_MS)
})

// Окно скрывается, а не закрывается, поэтому сюда попадаем только при
// реальном выходе — не оставляем процесс-зомби ни на одной платформе.
app.on('window-all-closed', () => app.quit())
