<template>
  <component :is="comp" v-if="comp" :key="reloadKey" v-bind="viewProps" />
  <div v-else class="win-missing">Раздел недоступен</div>
</template>

<script setup>
import { computed, defineAsyncComponent, h, markRaw, ref } from 'vue'
import { useDesktopStore } from '@/stores/desktop.js'
import { provideWindowRoute } from '@/desktop/windowRoute.js'
import BrandLoader from '@/components/common/BrandLoader.vue'

const props = defineProps({
  win: { type: Object, required: true },
})

const desktop = useDesktopStore()
const { winRoute } = provideWindowRoute(props.win, desktop)

/* Компонент раздела берём из записи маршрута — маршруты остаются единственным
   местом, где путь связан с экраном. Ленивые загрузчики кэшируем по функции:
   второе окно того же раздела не перезагружает чанк.

   Чанк может и не приехать: моргнула сеть или после выката поменялись имена
   файлов, а вкладка живёт со старым манифестом. Раньше окно в этом случае
   оставалось ПУСТЫМ — ни ошибки, ни повтора, и помогала только перезагрузка
   страницы. Теперь загрузка повторяется сама, а если не вышло — окно честно
   сообщает об этом и предлагает повторить. */
const WindowLoading = markRaw({
  render: () => h('div', { class: 'win-loading' }, [h(BrandLoader, { size: 64 })]),
})

// Сколько раз молча повторяем загрузку чанка перед тем, как показать ошибку.
const LOAD_RETRIES = 2

const asyncCache = new Map()

// Ключ перерисовки: сброс кэша сам по себе не заставит Vue перемонтировать
// компонент — нужен новый key.
const reloadKey = ref(0)

function windowError(loader) {
  return markRaw({
    render: () => h('div', { class: 'win-error' }, [
      h('p', { class: 'win-error-text' }, 'Не удалось загрузить раздел'),
      h('p', { class: 'win-error-hint' },
        'Проверьте соединение. Если приложение обновлялось, поможет перезагрузка страницы.'),
      h('div', { class: 'win-error-actions' }, [
        h('button', {
          class: 'gw-chip',
          type: 'button',
          onClick: () => {
            asyncCache.delete(loader)
            reloadKey.value += 1
          },
        }, 'Повторить'),
        h('button', {
          class: 'gw-chip',
          type: 'button',
          onClick: () => window.location.reload(),
        }, 'Перезагрузить страницу'),
      ]),
    ]),
  })
}

function componentFor(record) {
  const raw = record?.components?.default
  if (!raw) return null
  if (typeof raw !== 'function') return markRaw(raw)
  if (!asyncCache.has(raw)) {
    asyncCache.set(raw, markRaw(defineAsyncComponent({
      loader: raw,
      loadingComponent: WindowLoading,
      errorComponent: windowError(raw),
      delay: 120,
      // Долгая загрузка на медленной сети — не ошибка; ошибка это отказ.
      timeout: 30_000,
      onError(error, retry, fail, attempts) {
        if (attempts <= LOAD_RETRIES) retry()
        else fail()
      },
    })))
  }
  return asyncCache.get(raw)
}

const lastRecord = computed(() => winRoute.matched?.[winRoute.matched.length - 1] || null)
const comp = computed(() => componentFor(lastRecord.value))

// Те же правила, что у <router-view>: props: true отдаёт параметры маршрута.
const viewProps = computed(() => {
  const p = lastRecord.value?.props?.default
  if (p === true) return { ...winRoute.params }
  if (typeof p === 'function') return p(winRoute)
  if (p && typeof p === 'object') return p
  return {}
})
</script>

<style>
.win-loading {
  display: grid;
  place-items: center;
  height: 100%;
  min-height: 200px;
}

.win-missing {
  display: grid;
  place-items: center;
  height: 100%;
  color: var(--color-text-dim);
}

.win-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  height: 100%;
  padding: 24px;
  text-align: center;
}

.win-error-text { margin: 0; font-size: 1.05rem; font-weight: 600; }
.win-error-hint { margin: 0; font-size: 0.85rem; color: var(--color-text-dim); }
.win-error-actions { display: flex; gap: 8px; margin-top: 6px; }
</style>
