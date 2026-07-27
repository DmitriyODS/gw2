<template>
  <component :is="comp" v-if="comp" v-bind="viewProps" />
  <div v-else class="win-missing">Раздел недоступен</div>
</template>

<script setup>
import { computed, defineAsyncComponent, h, markRaw } from 'vue'
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
   второе окно того же раздела не перезагружает чанк. */
const WindowLoading = markRaw({
  render: () => h('div', { class: 'win-loading' }, [h(BrandLoader, { size: 64 })]),
})

const asyncCache = new Map()

function componentFor(record) {
  const raw = record?.components?.default
  if (!raw) return null
  if (typeof raw !== 'function') return markRaw(raw)
  if (!asyncCache.has(raw)) {
    asyncCache.set(raw, markRaw(defineAsyncComponent({
      loader: raw,
      loadingComponent: WindowLoading,
      delay: 120,
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
</style>
