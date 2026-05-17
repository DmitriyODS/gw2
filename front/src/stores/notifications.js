import { defineStore } from 'pinia'

export const useNotificationsStore = defineStore('notifications', () => {
  let _toast = null

  function setToast(toastInstance) {
    _toast = toastInstance
  }

  function notify({ severity = 'info', summary = '', detail = '', life = 4000 }) {
    _toast?.add({ severity, summary, detail, life })
  }

  function success(detail, summary = 'Успешно') {
    notify({ severity: 'success', summary, detail })
  }

  function error(detail, summary = 'Ошибка') {
    notify({ severity: 'error', summary, detail, life: 6000 })
  }

  function warn(detail, summary = 'Внимание') {
    notify({ severity: 'warn', summary, detail })
  }

  return { setToast, notify, success, error, warn }
})
