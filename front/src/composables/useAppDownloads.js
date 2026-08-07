import { ref, onMounted } from 'vue'

/**
 * Скачивание клиентских приложений — общая обвязка для промо-страницы и
 * раздела «О приложении».
 *
 * Артефакты лежат в статике nginx: APK — канонический `/apps/mobile/groovework.apk`
 * (его же качает автообновление старых обёрток), установщики компьютера —
 * `/apps/desktop/<имя>` С ВЕРСИЕЙ в имени. Карты имён и номер сборки приезжают
 * из version.json рядом с файлами: безымянные имена однажды дали раздачу
 * старого установщика из кэша.
 */
const APK_HREF = '/apps/mobile/groovework.apk'
const DESKTOP_OS_LABELS = { mac: 'macOS', win: 'Windows', linux: 'Linux' }

export function useAppDownloads() {
  const apkDownloadName = ref('groovework.apk')
  const desktopFiles = ref({
    mac: 'GrooveWork-mac.dmg',
    win: 'GrooveWork-win.exe',
    linux: 'GrooveWork-linux.AppImage',
  })

  const desktopFileHref = (os) => `/apps/desktop/${desktopFiles.value[os]}`

  const ua = navigator.userAgent
  const platform = navigator.platform || ua
  const desktopOs = /Mac/i.test(platform) ? 'mac' : /Win/i.test(platform) ? 'win' : 'linux'

  // В самом Electron-клиенте и на телефонах предлагать установщик бессмысленно.
  const showDesktop = !/Electron/i.test(ua) && !/Android|iPhone|iPad/i.test(ua)
  // Внутри мобильной обёртки (Capacitor) скачивание APK не нужно — приложение
  // уже установлено. Признаки: мост window.Capacitor (надёжный) и метка
  // GrooveWorkApp в UA (appendUserAgent, страховка).
  const showApk = !window.Capacitor?.isNativePlatform?.() && !/GrooveWorkApp/i.test(ua)

  async function load() {
    try {
      const meta = await (await fetch('/apps/desktop/version.json', { cache: 'no-store' })).json()
      if (meta?.files?.mac) desktopFiles.value = meta.files
    } catch { /* карта не приехала — останутся легаси-имена */ }
    try {
      const meta = await (await fetch('/apps/mobile/version.json', { cache: 'no-store' })).json()
      if (meta?.current_build) apkDownloadName.value = `groovework-${meta.current_build}.apk`
    } catch { /* noop */ }
  }

  onMounted(load)

  return {
    APK_HREF, apkDownloadName,
    desktopFiles, desktopFileHref, desktopOs, DESKTOP_OS_LABELS,
    showApk, showDesktop, load,
  }
}
