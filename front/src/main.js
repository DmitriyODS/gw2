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

const GrovePreset = definePreset(Aura, {
  semantic: {
    primary: {
      50: '#fce4ec', 100: '#f8bbd0', 200: '#f48fb1',
      300: '#f06292', 400: '#ec407a', 500: '#e91e63',
      600: '#d81b60', 700: '#c2185b', 800: '#ad1457',
      900: '#880e4f', 950: '#560027'
    },
    colorScheme: {
      light: {
        primary: {
          color: '#e040fb',
          inverseColor: '#ffffff',
          hoverColor: '#d500f9',
          activeColor: '#aa00ff'
        },
        highlight: {
          background: '#fce4ec',
          focusBackground: '#f8bbd0',
          color: '#880e4f',
          focusColor: '#560027'
        }
      },
      dark: {
        primary: {
          color: '#ce93d8',
          inverseColor: '#1a1a2e',
          hoverColor: '#ba68c8',
          activeColor: '#ab47bc'
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
    preset: GrovePreset,
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
