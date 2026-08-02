<template>
  <div
    ref="rootEl"
    class="ld"
    :class="{ narrow, 'detail-open': narrow && open, collapsed: listCollapsed }"
    :style="{ '--ld-list-width': listWidthPx }"
  >
    <div class="ld-col ld-list">
      <slot
        name="list"
        :narrow="narrow"
        :open-detail="openDetail"
        :collapsed="listCollapsed"
        :toggle="toggleList"
      />
    </div>

    <div class="ld-col ld-detail">
      <slot
        name="detail"
        :narrow="narrow"
        :close="close"
        :collapsed="listCollapsed"
        :toggle="toggleList"
      />
    </div>

    <!-- Оверлеи раздела (диалоги, лайтбоксы): они не принадлежат ни списку, ни
         содержимому, поэтому лежат обычным содержимым компонента. -->
    <slot />
  </div>
</template>

<script setup>
/* Раскладка «список ⇄ содержимое» (по образцу ListDetailsView в UWP): широко —
   две колонки, узко — один экран с переходом «список → элемент → назад».
   Заменила `.split-view` реестров/календарей/ежедневников и двухпанельный
   каркас настроек, которые решали одну задачу тремя разными способами.

   Ширину меряет СВОЮ (ResizeObserver), а не экрана: раздел живёт окном
   рабочего стола, и media-запросы про его размер ничего не знают.

   Обе колонки раздел заполняет сам — обычно `AppPage embedded`, — поэтому
   слоты получают `narrow` (нужна ли кнопка «назад»), `close`, а также
   `collapsed`/`toggle`: в широкой раскладке список убирается с глаз, чтобы
   отдать всё место содержимому (кнопку рисует AppPage по `menu`). */
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useBreakpoint } from '@/composables/useBreakpoint.js'

const props = defineProps({
  /** Открыта ли деталь — значимо только в узком режиме. */
  open: { type: Boolean, default: false },
  /** Свёрнут ли список (широкая раскладка). Работает и без v-model. */
  collapsed: { type: Boolean, default: null },
  listWidth: { type: [Number, String], default: 300 },
  narrowAt: { type: Number, default: 720 },
})

const emit = defineEmits(['update:open', 'update:collapsed', 'narrow-change'])

const { isMobile } = useBreakpoint()
const rootEl = ref(null)
const narrow = ref(false)

const listWidthPx = computed(() =>
  typeof props.listWidth === 'number' ? `${props.listWidth}px` : props.listWidth,
)

function openDetail() { emit('update:open', true) }
function close() { emit('update:open', false) }

/* Сворачивание списка живёт внутри компонента, пока раздел не взял его на себя
   через v-model:collapsed. В узкой раскладке смысла не имеет — там колонки и так
   показываются по одной. */
const collapsedInner = ref(false)

const listCollapsed = computed(() =>
  !narrow.value && (props.collapsed === null ? collapsedInner.value : props.collapsed),
)

function toggleList() {
  const next = !listCollapsed.value
  collapsedInner.value = next
  emit('update:collapsed', next)
}

let ro = null

onMounted(() => {
  if (typeof ResizeObserver === 'undefined') {
    narrow.value = isMobile.value
    emit('narrow-change', narrow.value)
    return
  }
  ro = new ResizeObserver(([entry]) => {
    const next = entry.contentRect.width < props.narrowAt
    if (next === narrow.value) return
    narrow.value = next
    emit('narrow-change', next)
  })
  ro.observe(rootEl.value)
})

onBeforeUnmount(() => ro?.disconnect())

// Разъехались до двух колонок — деталь снова видна всегда, и «назад» ей не нужен.
watch(narrow, (v) => { if (!v) emit('update:open', true) })

defineExpose({ narrow })
</script>

<style scoped>
.ld {
  display: grid;
  grid-template-columns: var(--ld-list-width) minmax(0, 1fr);
  gap: 16px;
  height: 100%;
  min-height: 0;
  padding: 16px;
  overflow: hidden;
  transition: grid-template-columns 0.22s cubic-bezier(0.2, 0, 0, 1);
}

.ld-col { min-width: 0; min-height: 0; }

/* Свёрнутый список: колонка схлопывается до нуля, содержимое отдаёт всё место
   правой панели. Список остаётся в DOM — его прокрутка и выделение переживают
   сворачивание. */
.ld.collapsed {
  grid-template-columns: 0 minmax(0, 1fr);
  gap: 0;
}

.ld.collapsed .ld-list {
  overflow: hidden;
  opacity: 0;
  pointer-events: none;
}

@media (prefers-reduced-motion: reduce) {
  .ld { transition: none; }
}

/* Узко — один экран: видна ровно одна колонка, вторая снята с потока. */
.ld.narrow {
  grid-template-columns: 1fr;
  gap: 0;
  padding: 0;
}

.ld.narrow .ld-detail { display: none; }
.ld.narrow.detail-open .ld-list { display: none; }
.ld.narrow.detail-open .ld-detail { display: block; }

@media (max-width: 768px) {
  .ld { padding: 0; gap: 0; }
}
</style>
