// Определение обёртки, в которой открыт фронт. Мобильная (Capacitor) —
// инжектированный мост window.Capacitor (надёжный) + метка GrooveWorkApp в UA
// (appendUserAgent, страховка); десктопная (Electron) — мост window.GrooveDesktop
// из preload.
export const isMobileShell = () =>
  !!window.Capacitor?.isNativePlatform?.() || /GrooveWorkApp/i.test(navigator.userAgent)

export const isDesktopShell = () => !!window.GrooveDesktop

export const inAppShell = () => isMobileShell() || isDesktopShell()

// Кастомная схема deep link'ов обёрток (зарегистрирована в Electron и в
// AndroidManifest): groovework://yandex-callback?code=… возвращает OAuth-флоу
// из системного браузера обратно в приложение.
export const APP_SCHEME = 'groovework'
