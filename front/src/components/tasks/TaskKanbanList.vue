<script setup>
/* Тело колонки канбана — виртуальный список: в режиме доски приходит до
   тысячи задач, а на экране колонки их десяток. Карточку рисует родитель
   слотом, чтобы перетаскивание оставалось его заботой. */
import { computed, ref } from 'vue'
import { useVirtualList } from '@/composables/useVirtualList.js'

const props = defineProps({
  items: { type: Array, required: true },
})

const bodyEl = ref(null)
const listEl = ref(null)
const list = useVirtualList({
  container: bodyEl,
  list: listEl,
  keys: computed(() => props.items.map((t) => t.id)),
  estimate: () => 132,
})
const visible = computed(() => props.items.slice(list.start.value, list.end.value))
const padTop = computed(() => list.offsetOf(list.start.value))
const padBottom = computed(() => list.total.value - list.offsetOf(list.end.value))
</script>

<template>
  <div ref="bodyEl" class="kanban-col-body" @scroll="list.onScroll">
    <div
      v-if="items.length"
      ref="listEl"
      :style="{ paddingTop: `${padTop}px`, paddingBottom: `${padBottom}px` }"
    >
      <div v-for="t in visible" :key="t.id" :ref="list.measureRef(t.id)" class="kanban-row">
        <slot :task="t" />
      </div>
    </div>
    <div v-else class="kanban-col-empty">Пусто</div>
  </div>
</template>

<style scoped>
.kanban-col-body {
  padding: 10px 10px 0;
  flex: 1;
  min-height: 80px;
  overflow-y: auto;
  overflow-anchor: none;
}

/* Промежуток между карточками — отступом строки: он входит в замер высоты. */
.kanban-row {
  display: flow-root;
  padding-bottom: 10px;
}

.kanban-col-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  font-size: 12px;
  color: var(--color-text-dim);
  opacity: 0.7;
}
</style>
