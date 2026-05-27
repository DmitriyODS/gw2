import { createApp } from 'vue'
import { createPinia } from 'pinia'
import PrimeVue from 'primevue/config'
import ToastService from 'primevue/toastservice'
import ConfirmationService from 'primevue/confirmationservice'
import { definePreset } from '@primevue/themes'
import Aura from '@primevue/themes/aura'

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
  }
})
app.use(ToastService)
app.use(ConfirmationService)

app.mount('#app')
