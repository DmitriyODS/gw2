<template>
  <!-- v-show, а не v-if: неактивный раздел остаётся смонтированным и держит
       своё состояние (прокрутку, фильтры, черновики) — панель задач служит
       переключателем приложений, а не ссылкой на перезагрузку экрана. -->
  <section v-show="active" ref="rootEl" class="mscreen" :class="{ active }">
    <div class="mscreen-body main-content">
      <WindowContent :win="win" />
    </div>

    <Transition name="mspl">
      <AppSplash v-if="booting" :title="title" :icon="app?.icon || 'web_asset'" />
    </Transition>
  </section>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import router from '@/router/index.js'
import { appById, windowTitle } from '@/desktop/apps.js'
import { provideFloatHost } from '@/desktop/windowHost.js'
import WindowContent from '@/components/desktop/WindowContent.vue'
import AppSplash from './AppSplash.vue'

// Сколько держим экран запуска. Дольше — раздражает, короче — мелькает.
const SPLASH_MS = 560

const props = defineProps({
  win: { type: Object, required: true },
  active: { type: Boolean, default: false },
})

/* Плавающие кнопки раздела (AppFab) телепортируются СЮДА: улетев в body, они
   продолжали бы висеть и над стартовым экраном, и над соседним разделом. */
const rootEl = ref(null)
provideFloatHost(rootEl)

const app = computed(() => appById(props.win.appId))
const title = computed(() => windowTitle(app.value, router.resolve(props.win.path)))

/* Экран запуска показываем только при ПЕРВОМ открытии раздела: возврат к уже
   открытому — мгновенное переключение, там сплеш был бы враньём. */
const booting = ref(true)
let timer = null

onMounted(() => { timer = setTimeout(() => { booting.value = false }, SPLASH_MS) })
onBeforeUnmount(() => clearTimeout(timer))
</script>

<style scoped>
/* Раздел занимает весь экран между панелями: ни полей, ни рамки, ни зазора —
   примыкает к обеим вплотную. */
.mscreen {
  position: absolute;
  inset: calc(var(--statusbar-height) + env(safe-area-inset-top, 0px)) 0
    calc(var(--taskbar-height) + env(safe-area-inset-bottom, 0px)) 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.mscreen-body {
  flex: 1;
  min-height: 0;
  position: relative;
}

/* Появление раздела — короткий «подъём» из глубины, как запуск приложения. */
.mscreen.active { animation: mscreen-in 0.22s cubic-bezier(0.2, 0, 0, 1); }

@keyframes mscreen-in {
  from { opacity: 0; scale: 0.97; }
  to { opacity: 1; scale: 1; }
}

.mspl-leave-active { transition: opacity 0.22s ease, scale 0.28s cubic-bezier(0.2, 0, 0, 1); }
.mspl-leave-to { opacity: 0; scale: 1.04; }

@media (prefers-reduced-motion: reduce) {
  .mscreen.active { animation: none; }
  .mspl-leave-active { transition: none; }
}
</style>
