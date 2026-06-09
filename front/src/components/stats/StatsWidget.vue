<template>
  <div
    ref="rootRef"
    class="stats-widget"
    :class="[`size-${size}`, { pinned, dragging: isDragging, 'drag-over': dragOver }]"
    :style="{ order: orderOf(widgetId) }"
    :data-widget-id="widgetId"
    @dragover="onDragOver"
    @dragenter.prevent="onDragEnter"
    @dragleave="onDragLeave"
    @drop="onDrop"
  >
    <div class="widget-header">
      <span
        v-if="!isMobile"
        class="drag-handle"
        draggable="true"
        title="Перетащите, чтобы переместить"
        aria-label="Переместить виджет"
        @dragstart="onDragStart"
        @dragend="onDragEnd"
      >
        <span class="material-symbols-outlined">drag_indicator</span>
      </span>

      <h3>{{ title }}</h3>

      <div class="widget-tools">
        <button
          v-if="!isMobile"
          class="w-tool"
          :title="`Размер: ${sizeLabel}. Нажмите, чтобы изменить`"
          aria-label="Изменить размер"
          @click="cycleSize(widgetId)"
        >
          <span class="material-symbols-outlined">{{ sizeIcon }}</span>
        </button>
        <button
          class="w-tool"
          :class="{ active: pinned }"
          :title="pinned ? 'Открепить' : 'Закрепить наверху'"
          :aria-label="pinned ? 'Открепить' : 'Закрепить'"
          @click="togglePin(widgetId)"
        >
          <span class="material-symbols-outlined" :class="{ filled: pinned }">push_pin</span>
        </button>
        <button
          v-if="exportFn"
          class="w-tool"
          @click="handleExport"
          title="Скачать XLSX"
          aria-label="Скачать XLSX"
        >
          <span class="material-symbols-outlined">download</span>
        </button>
      </div>
    </div>
    <div class="widget-body">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { ref, computed } from 'vue'
import { useNotificationsStore } from '@/stores/notifications.js'
import { useBreakpoint } from '@/composables/useBreakpoint.js'
import { useStatsLayout } from '@/composables/useStatsLayout.js'

const props = defineProps({
  widgetId: {
    type: String,
    required: true,
  },
  title: {
    type: String,
    required: true,
  },
  exportFn: {
    type: Function,
    default: null,
  },
})

const notif = useNotificationsStore()
const { isMobile } = useBreakpoint()
const {
  sizeOf,
  pinnedOf,
  orderOf,
  cycleSize,
  togglePin,
  startDrag,
  endDrag,
  dropOn,
  draggingId,
} = useStatsLayout()

const rootRef = ref(null)
const dragOver = ref(false)

const size = computed(() => sizeOf(props.widgetId))
const pinned = computed(() => pinnedOf(props.widgetId))
const isDragging = computed(() => draggingId.value === props.widgetId)

const SIZE_META = {
  small: { label: 'маленький', icon: 'photo_size_select_small' },
  medium: { label: 'средний', icon: 'photo_size_select_large' },
  large: { label: 'большой', icon: 'fit_screen' },
}
const sizeLabel = computed(() => SIZE_META[size.value]?.label || '')
const sizeIcon = computed(() => SIZE_META[size.value]?.icon || 'aspect_ratio')

function onDragStart(e) {
  startDrag(props.widgetId)
  e.dataTransfer.effectAllowed = 'move'
  try {
    e.dataTransfer.setData('text/plain', props.widgetId)
    if (rootRef.value) e.dataTransfer.setDragImage(rootRef.value, 24, 24)
  } catch {
    /* старые браузеры */
  }
}

function onDragEnd() {
  endDrag()
  dragOver.value = false
}

function onDragOver(e) {
  if (!draggingId.value) return
  e.preventDefault()
  e.dataTransfer.dropEffect = 'move'
}

function onDragEnter() {
  if (draggingId.value && draggingId.value !== props.widgetId) dragOver.value = true
}

function onDragLeave(e) {
  // Игнорируем переходы между детьми внутри виджета.
  if (rootRef.value && e.relatedTarget && rootRef.value.contains(e.relatedTarget)) return
  dragOver.value = false
}

function onDrop(e) {
  if (!draggingId.value) return
  e.preventDefault()
  dragOver.value = false
  dropOn(props.widgetId)
}

async function handleExport() {
  if (!props.exportFn) return
  try {
    const response = await props.exportFn()
    let blob
    if (response instanceof Blob) {
      blob = response
    } else if (response && typeof response.blob === 'function') {
      blob = await response.blob()
    } else {
      blob = new Blob([response])
    }
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `export_${Date.now()}.xlsx`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  } catch (e) {
    notif.error(e.message || 'Ошибка экспорта')
  }
}
</script>

<style scoped>
.stats-widget {
  background: var(--color-surface);
  border: 1px solid var(--color-outline-dim);
  border-radius: var(--radius-xl, 20px);
  padding: 20px;
  display: flex;
  flex-direction: column;
  gap: 14px;
  box-shadow: var(--shadow-sm);
  max-height: var(--widget-max-height, 400px);
  transition: box-shadow 0.18s ease, border-color 0.18s ease, opacity 0.18s ease;
}

/* ── Размеры (ширина по колонкам сетки + высота) ── */
.stats-widget.size-small {
  grid-column: span 1;
  --widget-max-height: 360px;
}
.stats-widget.size-medium {
  grid-column: span 2;
  --widget-max-height: 420px;
}
.stats-widget.size-large {
  grid-column: 1 / -1;
  --widget-max-height: 560px;
}

.stats-widget.pinned {
  border-color: color-mix(in oklch, var(--color-primary) 45%, var(--color-outline-dim));
}

.stats-widget.dragging {
  opacity: 0.45;
}

.stats-widget.drag-over {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px color-mix(in oklch, var(--color-primary) 40%, transparent);
}

.widget-header {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.widget-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--color-text);
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.drag-handle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  margin-left: -4px;
  border-radius: var(--radius-full);
  color: var(--color-text-dim);
  cursor: grab;
  flex-shrink: 0;
  transition: background 0.15s, color 0.15s;
}
.drag-handle:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}
.drag-handle:active {
  cursor: grabbing;
}
.drag-handle .material-symbols-outlined {
  font-size: 20px;
}

.widget-tools {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.w-tool {
  background: none;
  border: none;
  border-radius: var(--radius-full);
  width: 36px;
  height: 36px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-dim);
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}

.w-tool:hover {
  background: var(--color-surface-high);
  color: var(--color-text);
}

.w-tool.active {
  color: var(--color-primary);
  background: color-mix(in oklch, var(--color-primary) 12%, transparent);
}

.w-tool .material-symbols-outlined {
  font-size: 20px;
}

.w-tool .material-symbols-outlined.filled {
  font-variation-settings: 'FILL' 1;
}

.widget-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

@media (max-width: 768px) {
  .stats-widget {
    padding: 16px;
    gap: 12px;
    border-radius: var(--radius-lg, 16px);
  }
  /* На мобильном всё в один столбец — размер ширины не применяем. */
  .stats-widget.size-small,
  .stats-widget.size-medium,
  .stats-widget.size-large {
    grid-column: 1 / -1;
    --widget-max-height: 460px;
  }
  .widget-header h3 {
    font-size: 15px;
  }
  .w-tool {
    width: 40px;
    height: 40px;
  }
  .w-tool .material-symbols-outlined {
    font-size: 18px;
  }
}
</style>
