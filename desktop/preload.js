/* Мост для веб-слоя (contextIsolation): фронт видит window.GrooveDesktop —
 * обновление самой обёртки (карточка в «О приложении»), настройки обёртки
 * (карточка «Приложение для компьютера» в настройках) и фокус окна по клику
 * на уведомление. Никакого Node в странице: только узкие IPC-вызовы. */
const { contextBridge, ipcRenderer } = require('electron')

contextBridge.exposeInMainWorld('GrooveDesktop', {
  getVersion: () => ipcRenderer.invoke('gw:get-version'),
  checkUpdate: () => ipcRenderer.invoke('gw:check-update'),
  downloadUpdate: (onProgress) => {
    const listener = (_e, p) => {
      try { onProgress?.(p) } catch {}
    }
    ipcRenderer.on('gw:update-progress', listener)
    return ipcRenderer
      .invoke('gw:download-update')
      .finally(() => ipcRenderer.removeListener('gw:update-progress', listener))
  },
  // Настройки обёртки: автозапуск / трей / поведение крестика.
  getSettings: () => ipcRenderer.invoke('gw:get-settings'),
  setSetting: (key, value) => ipcRenderer.invoke('gw:set-setting', key, value),
  // Поднять окно из трея (клик по уведомлению веб-слоя).
  focusWindow: () => ipcRenderer.send('gw:focus'),
})
