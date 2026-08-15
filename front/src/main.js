import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ConfirmationService from 'primevue/confirmationservice'
import { definePreset } from '@primeuix/themes'
import Aura from '@primeuix/themes/aura'

import App from './App.vue'
import router from './router/index.js'
import './assets/main.css'

const GroovePreset = definePreset(Aura, {
  semantic: {
    primary: {
      50:  'var(--_p-99)',
      100: 'var(--_p-95)',
      200: 'var(--_p-90)',
      300: 'var(--_p-80)',
      400: 'var(--_p-40)',
      500: 'var(--_p-40)',
      600: 'var(--_p-30)',
      700: 'var(--_p-20)',
      800: 'var(--_p-20)',
      900: 'var(--_p-10)',
      950: 'var(--_p-10)',
    },
    colorScheme: {
      light: {
        primary: {
          color:        'var(--color-primary)',
          inverseColor: 'var(--color-on-primary)',
          hoverColor:   'var(--color-primary-hover)',
          activeColor:  'var(--color-primary-hover)',
        },
        highlight: {
          background:      'var(--color-primary-container)',
          focusBackground: 'var(--color-primary-container)',
          color:           'var(--color-on-primary-container)',
          focusColor:      'var(--color-on-primary-container)',
        }
      },
      dark: {
        primary: {
          color:        'var(--color-primary)',
          inverseColor: 'var(--color-on-primary)',
          hoverColor:   'var(--color-primary-hover)',
          activeColor:  'var(--color-primary-hover)',
        },
        highlight: {
          background:      'var(--color-primary-container)',
          focusBackground: 'var(--color-primary-container)',
          color:           'var(--color-on-primary-container)',
          focusColor:      'var(--color-on-primary-container)',
        }
      }
    }
  }
})

const app = createApp(App)

/* Ошибки компонентов иначе теряются: Vue снимает поддерево, экран белеет, а в
   консоли — только сам факт. Пишем, ЧТО и ГДЕ упало, чтобы жалобу «раздел не
   открылся» можно было разобрать по тексту. Само окно раздела показывает эту
   ошибку пользователю (см. WindowContent.vue). */
app.config.errorHandler = (err, instance, info) => {
  const where = instance?.$options?.__name || instance?.$?.type?.__name || 'неизвестный компонент'
  console.error(`[gw] ошибка Vue в «${where}» (${info}):`, err)
}

app.use(createPinia())
app.use(router)
app.use(PrimeVue, {
  theme: {
    preset: GroovePreset,
    options: {
      prefix: 'p',
      darkModeSelector: '[data-dark="true"]',
      cssLayer: false
    }
  },
  locale: {
    firstDayOfWeek: 1,
    dayNames: ['воскресенье', 'понедельник', 'вторник', 'среда', 'четверг', 'пятница', 'суббота'],
    dayNamesShort: ['вс', 'пн', 'вт', 'ср', 'чт', 'пт', 'сб'],
    dayNamesMin: ['Вс', 'Пн', 'Вт', 'Ср', 'Чт', 'Пт', 'Сб'],
    monthNames: ['январь', 'февраль', 'март', 'апрель', 'май', 'июнь', 'июль', 'август', 'сентябрь', 'октябрь', 'ноябрь', 'декабрь'],
    monthNamesShort: ['янв', 'фев', 'мар', 'апр', 'май', 'июн', 'июл', 'авг', 'сен', 'окт', 'ноя', 'дек'],
    today: 'Сегодня',
    clear: 'Очистить',
    dateFormat: 'dd.mm.yy',
    weekHeader: 'Нед',
  }
})
app.use(ConfirmationService)

app.mount('#app')
// Сигнал бут-watchdog'у в index.html: приложение реально стартовало.
window.__gwBooted = true

/* Регистрации service worker здесь больше нет: он остался только ради показа
   OS-уведомлений и поднимается там, где нужен, — registerNotifyServiceWorker()
   после входа (см. utils/systemNotify.js). Раньше его будили сразу при загрузке,
   чтобы Chrome предлагал установку PWA, но кэша с офлайн-оболочкой у нас больше
   нет, а вместе с ним ушёл и смысл ранней регистрации.

   Что здесь осталось — уборка на dev-сервере: SW кэшировал модули Vite и отдавал
   их вперемешку со свежими, отчего разделы падали на ровном месте (см. sw.js).
   Установленный ранее SW продолжал бы вредить, поэтому снимаем его и его кэши. */
if (import.meta.env.DEV && 'serviceWorker' in navigator) {
  navigator.serviceWorker.getRegistrations()
    .then((regs) => regs.forEach((r) => r.unregister()))
    .catch(() => {})
  // window.caches нет в небезопасном контексте (заход по IP с телефона).
  if (window.caches) {
    caches.keys().then((keys) => keys.forEach((k) => caches.delete(k))).catch(() => {})
  }
}
