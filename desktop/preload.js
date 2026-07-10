/* Мост для веб-слоя (contextIsolation): фронт видит window.GrooveDesktop и
 * может принудительно проверить/поставить обновление самой обёртки — карточка
 * в «О приложении» (front/src/components/settings/AboutApp.vue). Никакого
 * Node в странице: только три узких IPC-вызова. */
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
})
