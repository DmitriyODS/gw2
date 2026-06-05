<template>
  <div class="stats-widget">
    <div class="widget-header">
      <h3>{{ title }}</h3>
      <button
        v-if="exportFn"
        class="export-btn"
        @click="handleExport"
        title="Скачать XLSX"
        aria-label="Скачать XLSX"
      >
        <span class="material-symbols-outlined">download</span>
      </button>
    </div>
    <div class="widget-body">
      <slot />
    </div>
  </div>
</template>

<script setup>
import { useNotificationsStore } from '@/stores/notifications.js'

const props = defineProps({
  title: {
    type: String,
    required: true
  },
  exportFn: {
    type: Function,
    default: null
  }
})

const notif = useNotificationsStore()

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
  max-height: var(--widget-max-height, 380px);
}

.widget-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-shrink: 0;
}

.widget-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--color-text);
}

.export-btn {
  background: var(--color-surface-high);
  border: none;
  border-radius: var(--radius-full);
  width: 40px;
  height: 40px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-dim);
  transition: background 0.15s, color 0.15s;
  flex-shrink: 0;
}

.export-btn:hover {
  background: var(--color-primary);
  color: var(--color-on-primary);
}

.export-btn .material-symbols-outlined {
  font-size: 20px;
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
    max-height: var(--widget-max-height, 420px);
  }
  .widget-header h3 {
    font-size: 15px;
  }
  .export-btn {
    width: 36px;
    height: 36px;
  }
  .export-btn .material-symbols-outlined {
    font-size: 18px;
  }
}
</style>
